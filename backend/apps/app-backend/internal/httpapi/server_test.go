package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"oneflow/app-backend/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testJWTSecret = "test-jwt-secret"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestFetchOpenRouterSupportedChatModelsAppliesShortDeadline(t *testing.T) {
	var remaining time.Duration
	srv := &Server{
		cfg: config.Config{OpenRouterModelsURL: "https://openrouter.test/models"},
		httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			deadline, ok := req.Context().Deadline()
			if !ok {
				t.Fatal("OpenRouter model catalog request must have a deadline")
			}
			remaining = time.Until(deadline)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"openai/gpt-4o-mini","name":"GPT-4o Mini","architecture":{"input_modalities":["text"],"output_modalities":["text"]},"pricing":{"prompt":"0.00000015","completion":"0.0000006"}}]}`)),
				Request:    req,
			}, nil
		})},
	}

	if _, err := srv.fetchOpenRouterSupportedChatModels(context.Background(), nil, nil, true); err != nil {
		t.Fatalf("fetch model catalog: %v", err)
	}
	if remaining <= 0 || remaining > 5*time.Second {
		t.Fatalf("model catalog deadline = %s, want within 5s", remaining)
	}
}

func TestIsAllowedOpenRouterChatModelSupportsDeepSeek(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "openai gpt", id: "openai/gpt-4o-mini", want: true},
		{name: "anthropic claude", id: "anthropic/claude-haiku-4.5", want: true},
		{name: "deepseek chat", id: "deepseek/deepseek-v4-flash", want: true},
		{name: "deepseek reasoning", id: "deepseek/deepseek-r1", want: true},
		{name: "latest alias rejected", id: "deepseek/deepseek-chat:latest", want: false},
		{name: "unsupported provider rejected", id: "meta-llama/llama-3.3-70b-instruct", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAllowedOpenRouterChatModel(tt.id, ""); got != tt.want {
				t.Fatalf("isAllowedOpenRouterChatModel(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("APP_BACKEND_TEST_DSN"))
	if dsn == "" {
		if os.Getenv("APP_BACKEND_REQUIRE_DB_TESTS") == "true" {
			t.Fatal("APP_BACKEND_TEST_DSN is required for integration tests")
		}
		t.Skip("APP_BACKEND_TEST_DSN is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping test db: %v", err)
	}
	return pool
}

func testServer(t *testing.T, pool *pgxpool.Pool) *httptest.Server {
	t.Helper()
	return testServerWithConfig(t, pool, config.Config{})
}

func testServerWithConfig(t *testing.T, pool *pgxpool.Pool, cfg config.Config) *httptest.Server {
	t.Helper()
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = testJWTSecret
	}
	if cfg.InternalGatewayToken == "" {
		cfg.InternalGatewayToken = "test-internal-token"
	}
	if len(cfg.WebsocketAllowedOrigins) == 0 {
		cfg.WebsocketAllowedOrigins = []string{"http://localhost:3000"}
	}
	srv := New(config.Config{
		JWTSecret:               testJWTSecret,
		InternalGatewayToken:    cfg.InternalGatewayToken,
		WebsocketAllowedOrigins: cfg.WebsocketAllowedOrigins,
		WAGatewayBaseURL:        cfg.WAGatewayBaseURL,
		AIServiceBaseURL:        cfg.AIServiceBaseURL,
	}, pool, nil)
	httpServer := httptest.NewServer(srv.Routes())
	t.Cleanup(httpServer.Close)
	return httpServer
}

func testToken(t *testing.T, role string) string {
	t.Helper()
	return testTokenForAgent(t, role, fmt.Sprintf("00000000-0000-0000-0000-%012d", time.Now().UnixNano()%1_000_000_000_000))
}

func testTokenForAgent(t *testing.T, role, agentID string) string {
	t.Helper()
	claims := agentClaims{
		AgentID: agentID,
		Role:    role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Subject:   role,
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func TestPublicBillingPackagesExposeOnlyOfficialPlatformPlans(t *testing.T) {
	pool := testPool(t)
	httpServer := testServer(t, pool)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var platformOrganizationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM organizations WHERE slug = 'default-company' LIMIT 1`).Scan(&platformOrganizationID); err != nil {
		t.Fatalf("query platform organization: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	activeName := "Public Monthly " + suffix
	addonName := "Private Addon " + suffix
	tenantName := "Tenant Monthly " + suffix
	var tenantOrganizationID string
	var platformPricingSettingID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO organizations (name, slug)
		VALUES ($1, $2)
		RETURNING id
	`, "Public package isolation test", "public-package-test-"+suffix).Scan(&tenantOrganizationID); err != nil {
		t.Fatalf("insert tenant organization: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM credit_pricing_settings WHERE id = NULLIF($1, '')::uuid`, platformPricingSettingID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM credit_packages WHERE name IN ($1, $2, $3)`, activeName, addonName, tenantName)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM organizations WHERE id = $1`, tenantOrganizationID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO credit_packages (
		  name, plan_key, description, credit_amount, price, billing_period, max_whatsapp_sessions,
		  max_ai_agents, max_human_users, is_active, is_popular, organization_id
		)
		VALUES
		  ($1, 'starter', 'public plan', 321, 456000, 'monthly', 2, 3, 4, TRUE, TRUE, $4),
		  ($2, 'addon_test', 'addon plan', 100, 20000, 'monthly', 0, 0, 0, TRUE, FALSE, $4),
		  ($3, 'growth', 'tenant plan', 999, 999000, 'monthly', 9, 9, 9, TRUE, FALSE, $5)
	`, activeName, addonName, tenantName, platformOrganizationID, tenantOrganizationID); err != nil {
		t.Fatalf("insert package fixtures: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO credit_pricing_settings (
		  credit_unit_idr, usd_to_idr_rate, chat_model_name, chat_input_price_per_1m,
		  chat_output_price_per_1m, embedding_model_name, embedding_price_per_1m,
		  annual_discount_percent, is_active, organization_id, updated_at
		)
		VALUES (50, 16000, 'openai/gpt-4o-mini', 0.15, 0.60, 'openai/text-embedding-3-small', 0.02, 17.5, TRUE, $1, NOW() + INTERVAL '1 minute')
		RETURNING id::text
	`, platformOrganizationID).Scan(&platformPricingSettingID); err != nil {
		t.Fatalf("insert public annual discount fixture: %v", err)
	}

	resp, err := http.Get(httpServer.URL + "/api/public/billing/packages")
	if err != nil {
		t.Fatalf("get public packages: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public packages status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var payload struct {
		AnnualDiscountPercent float64          `json:"annualDiscountPercent"`
		Items                 []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode public packages: %v", err)
	}

	var publicItem map[string]any
	for _, item := range payload.Items {
		switch item["name"] {
		case activeName:
			publicItem = item
		case addonName, tenantName:
			t.Fatalf("public packages exposed excluded item %q", item["name"])
		}
	}
	if publicItem == nil {
		t.Fatalf("public packages did not include official platform plan %q", activeName)
	}
	if payload.AnnualDiscountPercent != 17.5 {
		t.Fatalf("public annual discount = %v, want 17.5", payload.AnnualDiscountPercent)
	}
	if publicItem["price"] != float64(456000) || publicItem["creditAmount"] != float64(321) || publicItem["isPopular"] != true {
		t.Fatalf("public package payload = %v, want configured display fields", publicItem)
	}
	if _, exists := publicItem["id"]; exists {
		t.Fatalf("public package payload should not expose an internal id: %v", publicItem)
	}
}

func TestStartWhatsAppTypingUsesWhatsAppSessionContext(t *testing.T) {
	paths := make(chan string, 2)
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(gateway.Close)

	s := New(config.Config{
		WAGatewayBaseURL:     gateway.URL,
		InternalGatewayToken: "test-internal-token",
	}, nil, nil)
	ctx := context.WithValue(context.Background(), contextWhatsAppSession, "session-123")

	stop := s.startWhatsAppTyping(ctx, "+628123456789")
	defer stop()

	select {
	case path := <-paths:
		if path != "/api/sessions/session-123/chat-presence" {
			t.Fatalf("typing presence path = %q, want session-scoped chat-presence", path)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for typing presence request")
	}

	stop()

	select {
	case path := <-paths:
		if path != "/api/sessions/session-123/chat-presence" {
			t.Fatalf("stop typing presence path = %q, want session-scoped chat-presence", path)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stop typing presence request")
	}
}

func TestEnsureWhatsAppConnectedSendsInternalTokenForSessionStatus(t *testing.T) {
	const internalToken = "test-internal-token"
	var gotPath string
	var gotToken string
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Internal-Token")
		if gotToken != internalToken {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid internal token"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "connected"})
	}))
	t.Cleanup(gateway.Close)

	s := New(config.Config{
		WAGatewayBaseURL:     gateway.URL,
		InternalGatewayToken: internalToken,
	}, nil, nil)
	ctx := context.WithValue(context.Background(), contextWhatsAppSession, "session-123")

	ok, err := s.ensureWhatsAppConnected(ctx)
	if err != nil {
		t.Fatalf("ensure whatsapp connected: %v", err)
	}
	if !ok {
		t.Fatal("ensure whatsapp connected = false, want true")
	}
	if gotPath != "/api/sessions/session-123/status" {
		t.Fatalf("status path = %q, want session status path", gotPath)
	}
	if gotToken != internalToken {
		t.Fatalf("internal token = %q, want configured token", gotToken)
	}
}

func TestEnsureWhatsAppConnectedReturnsErrorForGatewayStatusFailure(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid internal token"})
	}))
	t.Cleanup(gateway.Close)

	s := New(config.Config{WAGatewayBaseURL: gateway.URL}, nil, nil)
	ok, err := s.ensureWhatsAppConnected(context.Background())
	if err == nil {
		t.Fatal("ensure whatsapp connected error = nil, want gateway status error")
	}
	if ok {
		t.Fatal("ensure whatsapp connected = true, want false")
	}
	if !strings.Contains(err.Error(), "wa-gateway status returned 401") {
		t.Fatalf("error = %q, want status code detail", err.Error())
	}
}

func cleanupContacts(t *testing.T, pool *pgxpool.Pool, phones ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, phone := range phones {
		if strings.TrimSpace(phone) == "" {
			continue
		}
		if _, err := pool.Exec(ctx, `DELETE FROM contacts WHERE phone = $1`, phone); err != nil {
			t.Fatalf("cleanup contact %s: %v", phone, err)
		}
	}
}

func TestRecentConversationAIHistoryTxReturnsPriorConversationMessages(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var organizationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM organizations WHERE slug = 'default-company' LIMIT 1`).Scan(&organizationID); err != nil {
		t.Fatalf("query default organization: %v", err)
	}

	phone := fmt.Sprintf("+628177%09d", time.Now().UnixNano()%1_000_000_000)
	cleanupContacts(t, pool, phone)
	t.Cleanup(func() { cleanupContacts(t, pool, phone) })

	var contactID, conversationID, currentMessageID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO contacts (phone, name, organization_id)
		VALUES ($1, 'History TDD', $2)
		RETURNING id
	`, phone, organizationID).Scan(&contactID); err != nil {
		t.Fatalf("insert contact: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO conversations (contact_id, channel, mode, status, last_message_at, organization_id)
		VALUES ($1, 'whatsapp', 'ai', 'open', NOW(), $2)
		RETURNING id
	`, contactID, organizationID).Scan(&conversationID); err != nil {
		t.Fatalf("insert conversation: %v", err)
	}

	base := time.Now().UTC().Add(-10 * time.Minute)
	if _, err := pool.Exec(ctx, `
		INSERT INTO messages (conversation_id, sender_type, direction, content_type, text, sent_at, delivered_at, organization_id)
		VALUES
		  ($1, 'customer', 'inbound', 'text', 'halo sebelumnya', $2, $2, $6),
		  ($1, 'ai', 'outbound', 'text', 'jawaban sebelumnya', $3, $3, $6),
		  ($1, 'ai', 'outbound', 'text', '   ', $4, $4, $6),
		  ($1, 'customer', 'inbound', 'text', 'produk tadi apa?', $5, $5, $6)
	`, conversationID, base, base.Add(time.Minute), base.Add(2*time.Minute), base.Add(3*time.Minute), organizationID); err != nil {
		t.Fatalf("insert prior messages: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_type, direction, content_type, text, sent_at, delivered_at, organization_id)
		VALUES ($1, 'customer', 'inbound', 'text', 'berapa harganya?', $2, $2, $3)
		RETURNING id
	`, conversationID, base.Add(4*time.Minute), organizationID).Scan(&currentMessageID); err != nil {
		t.Fatalf("insert current message: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	srv := New(config.Config{JWTSecret: testJWTSecret}, pool, nil)
	historyCtx := context.WithValue(ctx, contextOrganizationID, organizationID)
	history, err := srv.recentConversationAIHistoryTx(historyCtx, tx, conversationID, currentMessageID, 10)
	if err != nil {
		t.Fatalf("recent history: %v", err)
	}

	want := []aiChatHistoryMessage{
		{Role: "user", Text: "halo sebelumnya"},
		{Role: "assistant", Text: "jawaban sebelumnya"},
		{Role: "user", Text: "produk tadi apa?"},
	}
	if len(history) != len(want) {
		t.Fatalf("history len = %d, want %d: %#v", len(history), len(want), history)
	}
	for i := range want {
		if history[i] != want[i] {
			t.Fatalf("history[%d] = %#v, want %#v", i, history[i], want[i])
		}
	}
}

func TestHandleContactsAllowsOnlySuperAdminAndHRAgent(t *testing.T) {
	pool := testPool(t)
	httpServer := testServer(t, pool)

	suffix := time.Now().UnixNano() % 1_000_000_000
	phone := fmt.Sprintf("+628199%09d", suffix)
	cleanupContacts(t, pool, phone)
	t.Cleanup(func() { cleanupContacts(t, pool, phone) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var contactID, conversationID, inboundID, outboundID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO contacts (phone, name, email, notes)
		VALUES ($1, 'TDD Customer', 'tdd@example.com', 'created by backend test')
		RETURNING id
	`, phone).Scan(&contactID); err != nil {
		t.Fatalf("insert contact: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO conversations (contact_id, channel, mode, status, last_message_at)
		VALUES ($1, 'whatsapp', 'ai', 'open', NOW())
		RETURNING id
	`, contactID).Scan(&conversationID); err != nil {
		t.Fatalf("insert conversation: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_type, direction, content_type, text, sent_at, delivered_at)
		VALUES ($1, 'customer', 'inbound', 'text', 'Halo dari test', NOW(), NOW())
		RETURNING id
	`, conversationID).Scan(&inboundID); err != nil {
		t.Fatalf("insert inbound message: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_type, direction, content_type, text, sent_at, delivered_at)
		VALUES ($1, 'ai', 'outbound', 'text', 'Jawaban test', NOW(), NOW())
		RETURNING id
	`, conversationID).Scan(&outboundID); err != nil {
		t.Fatalf("insert outbound message: %v", err)
	}
	if inboundID == outboundID {
		t.Fatalf("expected distinct test messages")
	}
	if _, err := pool.Exec(ctx, `
		UPDATE conversations
		SET last_message_id = $2, last_message_at = NOW()
		WHERE id = $1
	`, conversationID, outboundID); err != nil {
		t.Fatalf("update last message: %v", err)
	}

	for _, role := range []string{"owner", "admin"} {
		req, err := http.NewRequest(http.MethodGet, httpServer.URL+"/api/contacts", nil)
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+testToken(t, role))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request contacts as %s: %v", role, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("role %s status = %d, want %d", role, resp.StatusCode, http.StatusForbidden)
		}
	}

	for _, role := range []string{"super_admin", "operator"} {
		req, err := http.NewRequest(http.MethodGet, httpServer.URL+"/api/contacts", nil)
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+testToken(t, role))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request contacts as %s: %v", role, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("role %s status = %d, want %d", role, resp.StatusCode, http.StatusOK)
		}

		var payload struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode contacts response: %v", err)
		}
		item := findContactItem(payload.Items, phone)
		if item == nil {
			t.Fatalf("role %s did not receive inserted contact %s", role, phone)
		}
		if item["name"] != "TDD Customer" {
			t.Fatalf("name = %v, want TDD Customer", item["name"])
		}
		if item["mode"] != "ai" || item["status"] != "open" {
			t.Fatalf("mode/status = %v/%v, want ai/open", item["mode"], item["status"])
		}
		if item["lastMessageText"] != "Jawaban test" {
			t.Fatalf("lastMessageText = %v, want Jawaban test", item["lastMessageText"])
		}
		if item["totalMessages"] != float64(2) || item["inboundMessages"] != float64(1) || item["outboundMessages"] != float64(1) {
			t.Fatalf("message counts = total:%v inbound:%v outbound:%v, want 2/1/1", item["totalMessages"], item["inboundMessages"], item["outboundMessages"])
		}
	}
}

func TestResolveWhatsAppContactCreatesAndMergesCanonicalContact(t *testing.T) {
	pool := testPool(t)
	srv := New(config.Config{}, pool, nil)

	suffix := time.Now().UnixNano() % 1_000_000_000
	canonicalPhone := fmt.Sprintf("+628188%09d", suffix)
	rawPhone := fmt.Sprintf("+628177%09d", suffix)
	cleanupContacts(t, pool, canonicalPhone, rawPhone)
	t.Cleanup(func() { cleanupContacts(t, pool, canonicalPhone, rawPhone) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	contactID, currentName, err := srv.resolveWhatsAppContactTx(ctx, tx, canonicalPhone, canonicalPhone, "Auto Saved", true, true)
	if err != nil {
		t.Fatalf("resolve new contact: %v", err)
	}
	if strings.TrimSpace(contactID) == "" {
		t.Fatalf("expected contact id")
	}
	if currentName != "Auto Saved" {
		t.Fatalf("currentName = %q, want Auto Saved", currentName)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit new contact: %v", err)
	}

	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin merge tx: %v", err)
	}
	defer tx.Rollback(ctx)

	var rawContactID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO contacts (phone, name)
		VALUES ($1, 'Raw Saved')
		RETURNING id
	`, rawPhone).Scan(&rawContactID); err != nil {
		t.Fatalf("insert raw contact: %v", err)
	}

	mergedID, mergedName, err := srv.resolveWhatsAppContactTx(ctx, tx, canonicalPhone, rawPhone, "Updated Customer", true, true)
	if err != nil {
		t.Fatalf("resolve merged contact: %v", err)
	}
	if mergedID != contactID {
		t.Fatalf("mergedID = %s, want existing canonical contact %s", mergedID, contactID)
	}
	if mergedName != "Auto Saved" {
		t.Fatalf("mergedName = %q, want existing name Auto Saved", mergedName)
	}

	var rawCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM contacts WHERE phone = $1`, rawPhone).Scan(&rawCount); err != nil {
		t.Fatalf("count raw contact: %v", err)
	}
	if rawCount != 0 {
		t.Fatalf("raw contact count = %d, want 0 after merge", rawCount)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit merge contact: %v", err)
	}
}

func TestInternalAuthRejectsInboundWithoutGatewayToken(t *testing.T) {
	pool := testPool(t)
	httpServer := testServer(t, pool)

	body := bytes.NewBufferString(`{"phone":"+628100000000","text":"halo"}`)
	resp, err := http.Post(httpServer.URL+"/api/internal/wa/inbound", "application/json", body)
	if err != nil {
		t.Fatalf("post inbound without token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestManualReplyRequiresTakeoverAndSupportsMediaAfterTakeover(t *testing.T) {
	pool := testPool(t)
	wa := newWAMock(t, true)
	httpServer := testServerWithConfig(t, pool, config.Config{WAGatewayBaseURL: wa.server.URL})
	agentID, _ := createTestAgent(t, pool, "operator")
	token := testTokenForAgent(t, "operator", agentID)
	phone := fmt.Sprintf("+628166%09d", time.Now().UnixNano()%1_000_000_000)
	conversationID := createTestConversation(t, pool, phone, "ai", nil)

	resp := doJSON(t, http.MethodPost, httpServer.URL+"/api/conversations/"+conversationID+"/manual-message", token, map[string]any{"text": "halo"})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("manual reply before takeover status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
	resp.Body.Close()
	if wa.textSends != 0 || wa.mediaSends != 0 {
		t.Fatalf("manual reply before takeover sent outbound text=%d media=%d", wa.textSends, wa.mediaSends)
	}

	resp = doJSON(t, http.MethodPost, httpServer.URL+"/api/conversations/"+conversationID+"/takeover", token, map[string]any{"note": "test takeover"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("takeover status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()

	resp = doJSON(t, http.MethodPost, httpServer.URL+"/api/conversations/"+conversationID+"/manual-message", token, map[string]any{
		"text":        "ini lampiran",
		"mediaBase64": base64PNG1x1(),
		"mediaMime":   "image/png",
		"mediaName":   "screenshot.png",
		"mediaKind":   "image",
	})
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("manual media status = %d, want %d body=%s", resp.StatusCode, http.StatusOK, string(body))
	}
	resp.Body.Close()
	if wa.mediaSends != 1 {
		t.Fatalf("media sends = %d, want 1", wa.mediaSends)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var contentType, text, rawPayload string
	if err := pool.QueryRow(ctx, `
		SELECT content_type::text, COALESCE(text, ''), COALESCE(raw_payload, '{}'::jsonb)::text
		FROM messages
		WHERE conversation_id = $1 AND direction = 'outbound'
		ORDER BY created_at DESC
		LIMIT 1
	`, conversationID).Scan(&contentType, &text, &rawPayload); err != nil {
		t.Fatalf("query outbound message: %v", err)
	}
	if contentType != "image" || text != "ini lampiran" || !strings.Contains(rawPayload, "screenshot.png") {
		t.Fatalf("stored outbound media = contentType:%s text:%s raw:%s", contentType, text, rawPayload)
	}
}

func TestReturnToAIDoesNotRequireWhatsAppConnected(t *testing.T) {
	pool := testPool(t)
	wa := newWAMock(t, false)
	httpServer := testServerWithConfig(t, pool, config.Config{WAGatewayBaseURL: wa.server.URL})
	agentID, _ := createTestAgent(t, pool, "operator")
	token := testTokenForAgent(t, "operator", agentID)
	phone := fmt.Sprintf("+628164%09d", time.Now().UnixNano()%1_000_000_000)
	conversationID := createTestConversation(t, pool, phone, "human", &agentID)

	resp := doJSON(t, http.MethodPost, httpServer.URL+"/api/conversations/"+conversationID+"/return-to-ai", token, map[string]any{})
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("return to ai status = %d, want %d body=%s", resp.StatusCode, http.StatusOK, string(body))
	}
	resp.Body.Close()
	if wa.lastPath != "" {
		t.Fatalf("return to ai called wa-gateway path = %q, want no status check", wa.lastPath)
	}

	var mode string
	var assignedTo *string
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := pool.QueryRow(ctx, `
		SELECT mode::text, assigned_to
		FROM conversations
		WHERE id = $1
	`, conversationID).Scan(&mode, &assignedTo); err != nil {
		t.Fatalf("query conversation after return to ai: %v", err)
	}
	if mode != "ai" || assignedTo != nil {
		t.Fatalf("conversation after return to ai = mode:%s assigned:%v, want ai nil", mode, assignedTo)
	}
}

func TestManualMediaRejectsUnsupportedAndOversizedPayload(t *testing.T) {
	pool := testPool(t)
	wa := newWAMock(t, true)
	httpServer := testServerWithConfig(t, pool, config.Config{WAGatewayBaseURL: wa.server.URL})
	agentID, _ := createTestAgent(t, pool, "operator")
	token := testTokenForAgent(t, "operator", agentID)
	phone := fmt.Sprintf("+628165%09d", time.Now().UnixNano()%1_000_000_000)
	conversationID := createTestConversation(t, pool, phone, "human", &agentID)

	resp := doJSON(t, http.MethodPost, httpServer.URL+"/api/conversations/"+conversationID+"/manual-message", token, map[string]any{
		"mediaBase64": base64PNG1x1(),
		"mediaMime":   "application/x-msdownload",
		"mediaName":   "payload.exe",
		"mediaKind":   "document",
	})
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unsupported media status = %d, want %d body=%s", resp.StatusCode, http.StatusBadRequest, string(body))
	}
	resp.Body.Close()

	oversized := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, maxManualMediaBytes+1))
	resp = doJSON(t, http.MethodPost, httpServer.URL+"/api/conversations/"+conversationID+"/manual-message", token, map[string]any{
		"mediaBase64": oversized,
		"mediaMime":   "image/png",
		"mediaName":   "too-large.png",
		"mediaKind":   "image",
	})
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("oversized media status = %d, want %d body=%s", resp.StatusCode, http.StatusBadRequest, string(body))
	}
	resp.Body.Close()
	if wa.mediaSends != 0 {
		t.Fatalf("media sends = %d, want 0 after rejected media", wa.mediaSends)
	}
}

func TestOwnerCannotAccessConversationContent(t *testing.T) {
	pool := testPool(t)
	httpServer := testServer(t, pool)
	ownerID, _ := createTestAgent(t, pool, "owner")
	token := testTokenForAgent(t, "owner", ownerID)
	phone := fmt.Sprintf("+628164%09d", time.Now().UnixNano()%1_000_000_000)
	conversationID := createTestConversation(t, pool, phone, "ai", nil)

	req, err := http.NewRequest(http.MethodGet, httpServer.URL+"/api/conversations/"+conversationID, nil)
	if err != nil {
		t.Fatalf("build owner detail request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("owner detail request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("owner detail status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestConversationDetailOrdersMessagesBySentAt(t *testing.T) {
	pool := testPool(t)
	httpServer := testServer(t, pool)
	adminID, _ := createTestAgent(t, pool, "admin")
	token := testTokenForAgent(t, "admin", adminID)
	phone := fmt.Sprintf("+628166%09d", time.Now().UnixNano()%1_000_000_000)
	conversationID := createTestConversation(t, pool, phone, "ai", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	base := time.Now().UTC().Add(-10 * time.Minute)
	if _, err := pool.Exec(ctx, `
		INSERT INTO messages (conversation_id, sender_type, direction, content_type, text, sent_at, created_at)
		VALUES
		  ($1, 'customer', 'inbound', 'text', 'message sent second', $2, $3),
		  ($1, 'customer', 'inbound', 'text', 'message sent first', $4, $5)
	`, conversationID, base.Add(2*time.Minute), base, base.Add(1*time.Minute), base.Add(3*time.Minute)); err != nil {
		t.Fatalf("insert out-of-order messages: %v", err)
	}

	detail := getConversationDetail(t, httpServer.URL, conversationID, token)
	messages, ok := detail["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %#v, want two messages", detail["messages"])
	}
	first, _ := messages[0].(map[string]any)
	if first["text"] != "message sent first" {
		t.Fatalf("first message = %q, want sent_at order", first["text"])
	}
}

func TestAssignMakesComposerAvailableOnlyForAssignedAgent(t *testing.T) {
	pool := testPool(t)
	wa := newWAMock(t, true)
	httpServer := testServerWithConfig(t, pool, config.Config{WAGatewayBaseURL: wa.server.URL})
	adminID, _ := createTestAgent(t, pool, "admin")
	assigneeID, assigneeUsername := createTestAgent(t, pool, "operator")
	otherID, _ := createTestAgent(t, pool, "operator")
	adminToken := testTokenForAgent(t, "admin", adminID)
	assigneeToken := testTokenForAgent(t, "operator", assigneeID)
	otherToken := testTokenForAgent(t, "operator", otherID)
	phone := fmt.Sprintf("+628155%09d", time.Now().UnixNano()%1_000_000_000)
	conversationID := createTestConversation(t, pool, phone, "ai", nil)

	resp := doJSON(t, http.MethodPost, httpServer.URL+"/api/conversations/"+conversationID+"/assign", adminToken, map[string]any{"agentUsername": assigneeUsername})
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("assign status = %d, want %d body=%s", resp.StatusCode, http.StatusOK, string(body))
	}
	resp.Body.Close()

	assignedDetail := getConversationDetail(t, httpServer.URL, conversationID, assigneeToken)
	if enabled, _ := assignedDetail["composer"].(map[string]any)["enabled"].(bool); !enabled {
		t.Fatalf("assigned agent composer enabled = %v, want true", assignedDetail["composer"])
	}
	otherDetail := getConversationDetail(t, httpServer.URL, conversationID, otherToken)
	if enabled, _ := otherDetail["composer"].(map[string]any)["enabled"].(bool); enabled {
		t.Fatalf("other agent composer enabled = true, want false")
	}
}

func TestEscalationGroupGenerateBindAndDelete(t *testing.T) {
	pool := testPool(t)
	wa := newWAMock(t, true)
	httpServer := testServerWithConfig(t, pool, config.Config{WAGatewayBaseURL: wa.server.URL})
	adminID, _ := createTestAgent(t, pool, "admin")
	adminToken := testTokenForAgent(t, "admin", adminID)
	previousBound := currentBoundEscalationGroup(t, pool)
	defer restoreBoundEscalationGroup(t, pool, previousBound)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := pool.Exec(ctx, `UPDATE wa_escalation_group_bindings SET status = 'removed' WHERE status IN ('pending', 'bound')`); err != nil {
		t.Fatalf("cleanup escalation groups: %v", err)
	}

	resp := doJSON(t, http.MethodPost, httpServer.URL+"/api/escalation-group/bind-code", adminToken, map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("generate code status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var generated struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&generated); err != nil {
		t.Fatalf("decode generate response: %v", err)
	}
	resp.Body.Close()
	if generated.Code == "" {
		t.Fatalf("expected binding code")
	}

	resp = doInternalJSON(t, http.MethodPost, httpServer.URL+"/api/internal/wa/group-binding", map[string]any{
		"groupId":   "120363000000001@g.us",
		"groupName": "TDD Escalation",
		"senderJid": "628100000000@s.whatsapp.net",
		"text":      "kode: " + generated.Code,
	})
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("bind group status = %d, want %d body=%s", resp.StatusCode, http.StatusOK, string(body))
	}
	resp.Body.Close()

	status := getEscalationGroup(t, httpServer.URL, adminToken)
	escalationGroup := status["escalationGroup"].(map[string]any)
	group := escalationGroup["group"].(map[string]any)
	if group["groupId"] != "120363000000001@g.us" || escalationGroup["bound"] != true {
		t.Fatalf("group status = %#v, want bound group", escalationGroup)
	}
	if wa.textSends != 1 || wa.lastTextPhone != "120363000000001@g.us" || !strings.Contains(wa.lastTextBody, "Grup eskalasi berhasil terhubung") {
		t.Fatalf("confirmation send = count:%d phone:%q text:%q, want one group confirmation", wa.textSends, wa.lastTextPhone, wa.lastTextBody)
	}

	req, err := http.NewRequest(http.MethodDelete, httpServer.URL+"/api/escalation-group", nil)
	if err != nil {
		t.Fatalf("build delete request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete group request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete group status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	status = getEscalationGroup(t, httpServer.URL, adminToken)
	escalationGroup = status["escalationGroup"].(map[string]any)
	if escalationGroup["bound"] == true || escalationGroup["group"] != nil {
		t.Fatalf("group status after delete = %#v, want unbound", escalationGroup)
	}

	req, err = http.NewRequest(http.MethodPost, httpServer.URL+"/api/internal/wa/group-binding", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("build unauthenticated binding request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unauthenticated binding request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated binding status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestLogCreditUsagePrefersOpenRouterActualCost(t *testing.T) {
	pool := testPool(t)
	srv := New(config.Config{}, pool, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE credit_pricing_settings SET is_active = FALSE`); err != nil {
		t.Fatalf("deactivate pricing: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE credit_wallet SET is_active = FALSE`); err != nil {
		t.Fatalf("deactivate wallet: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO credit_pricing_settings (
		  credit_unit_idr, usd_to_idr_rate, chat_model_name, chat_input_price_per_1m,
		  chat_output_price_per_1m, embedding_model_name, embedding_price_per_1m, is_active
		)
		VALUES (1, 10000, 'openai/gpt-test', 1, 1, 'text-embedding-test', 1, TRUE)
	`); err != nil {
		t.Fatalf("insert pricing: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO credit_wallet (monthly_credit_limit, monthly_credits_used, monthly_credits_remaining, additional_credits_remaining, is_active)
		VALUES (1000, 0, 1000, 0, TRUE)
	`); err != nil {
		t.Fatalf("insert wallet: %v", err)
	}

	actualCostUSD := 0.001
	result, err := srv.logCreditUsageTx(ctx, tx, "", "", "", "tdd_actual_cost", "openai/gpt-test", 100000, 100000, 0, &actualCostUSD)
	if err != nil {
		t.Fatalf("log credit usage: %v", err)
	}
	var costUSD, costIDR float64
	var creditsUsed, remaining int
	var notes string
	if err := tx.QueryRow(ctx, `
		SELECT cost_usd::float8, cost_idr::float8, credits_used, notes
		FROM credit_usage_logs
		WHERE usage_type = 'tdd_actual_cost'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&costUSD, &costIDR, &creditsUsed, &notes); err != nil {
		t.Fatalf("query usage log: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT monthly_credits_remaining FROM credit_wallet WHERE is_active = TRUE`).Scan(&remaining); err != nil {
		t.Fatalf("query remaining wallet: %v", err)
	}
	if costUSD != 0.001 || costIDR != 10 || creditsUsed != 10 || remaining != 990 || !strings.Contains(notes, "openrouter_actual") {
		t.Fatalf("usage log = costUSD:%v costIDR:%v credits:%v remaining:%v notes:%q, want actual cost 0.001/10/10/990/openrouter_actual", costUSD, costIDR, creditsUsed, remaining, notes)
	}
	if result.CreditsUsed != 10 || result.MonthlyRemaining != 990 || result.CreditSource != "monthly" {
		t.Fatalf("usage result = credits:%d remaining:%d source:%q, want 10/990/monthly", result.CreditsUsed, result.MonthlyRemaining, result.CreditSource)
	}
}

func TestKnowledgeDocumentsExposeChunkReadinessAndReindex(t *testing.T) {
	pool := testPool(t)
	httpServer := testServer(t, pool)
	adminID, _ := createTestAgent(t, pool, "admin")
	token := testTokenForAgent(t, "admin", adminID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var documentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO knowledge_documents (title, knowledge_type, raw_text, status, created_by, updated_by)
		VALUES ('TDD Knowledge Readiness', 'text', 'Satu dua tiga empat lima enam.', 'published', $1, $1)
		RETURNING id
	`, adminID).Scan(&documentID); err != nil {
		t.Fatalf("insert document: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(ctx, `DELETE FROM knowledge_documents WHERE id = $1`, documentID)
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO knowledge_chunks (knowledge_document_id, chunk_index, chunk_text, embedding)
		VALUES ($1, 0, 'Chunk satu siap.', NULL), ($1, 1, 'Chunk dua siap.', NULL)
	`, documentID); err != nil {
		t.Fatalf("insert chunks: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, httpServer.URL+"/api/knowledge/documents", nil)
	if err != nil {
		t.Fatalf("build documents request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("documents request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("documents status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var listPayload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listPayload); err != nil {
		t.Fatalf("decode documents: %v", err)
	}
	item := findItemByID(listPayload.Items, documentID)
	if item == nil {
		t.Fatalf("document %s not returned", documentID)
	}
	if item["chunkCount"] != float64(2) || item["pendingEmbeddingCount"] != float64(2) || item["embeddingStatus"] != "pending" {
		t.Fatalf("readiness = chunk:%v pending:%v status:%v, want 2/2/pending", item["chunkCount"], item["pendingEmbeddingCount"], item["embeddingStatus"])
	}

	hrAgentID, _ := createTestAgent(t, pool, "operator")
	hrToken := testTokenForAgent(t, "operator", hrAgentID)
	resp = doJSON(t, http.MethodPost, httpServer.URL+"/api/knowledge/documents/"+documentID+"/reindex", hrToken, map[string]any{})
	if resp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("operator reindex status = %d, want %d body=%s", resp.StatusCode, http.StatusForbidden, string(body))
	}
	resp.Body.Close()

	resp = doJSON(t, http.MethodPost, httpServer.URL+"/api/knowledge/documents/"+documentID+"/reindex", token, map[string]any{})
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("reindex status = %d, want %d body=%s", resp.StatusCode, http.StatusOK, string(body))
	}
	resp.Body.Close()
	var staleChunkCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM knowledge_chunks
		WHERE knowledge_document_id = $1
		  AND chunk_text IN ('Chunk satu siap.', 'Chunk dua siap.')
	`, documentID).Scan(&staleChunkCount); err != nil {
		t.Fatalf("count chunks after reindex: %v", err)
	}
	if staleChunkCount != 0 {
		t.Fatalf("stale chunk count after reindex = %d, want 0", staleChunkCount)
	}
}

func doJSON(t *testing.T, method, url, token string, body any) *http.Response {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, url, err)
	}
	return resp
}

func doInternalJSON(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal internal body: %v", err)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build internal request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", "test-internal-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("internal request %s %s: %v", method, url, err)
	}
	return resp
}

func getConversationDetail(t *testing.T, baseURL, conversationID, token string) map[string]any {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/conversations/"+conversationID, nil)
	if err != nil {
		t.Fatalf("build detail request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("conversation detail request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("conversation detail status = %d, want %d body=%s", resp.StatusCode, http.StatusOK, string(body))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode conversation detail: %v", err)
	}
	return payload
}

func getEscalationGroup(t *testing.T, baseURL, token string) map[string]any {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/escalation-group", nil)
	if err != nil {
		t.Fatalf("build escalation group request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("escalation group request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("escalation group status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode escalation group: %v", err)
	}
	return payload
}

type escalationGroupSnapshot struct {
	ID string
}

func currentBoundEscalationGroup(t *testing.T, pool *pgxpool.Pool) escalationGroupSnapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var snapshot escalationGroupSnapshot
	err := pool.QueryRow(ctx, `
		SELECT id
		FROM wa_escalation_group_bindings
		WHERE status = 'bound' AND COALESCE(group_id, '') <> ''
		ORDER BY bound_at DESC NULLS LAST, updated_at DESC
		LIMIT 1
	`).Scan(&snapshot.ID)
	if err != nil {
		return escalationGroupSnapshot{}
	}
	return snapshot
}

func restoreBoundEscalationGroup(t *testing.T, pool *pgxpool.Pool, snapshot escalationGroupSnapshot) {
	t.Helper()
	if strings.TrimSpace(snapshot.ID) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := pool.Exec(ctx, `
		UPDATE wa_escalation_group_bindings
		SET status = CASE WHEN status = 'bound' THEN 'removed' ELSE status END,
		    updated_at = NOW()
		WHERE id <> $1 AND status = 'bound'
	`, snapshot.ID); err != nil {
		t.Fatalf("clear test escalation binding: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE wa_escalation_group_bindings
		SET status = 'bound',
		    bound_at = COALESCE(bound_at, NOW()),
		    updated_at = NOW()
		WHERE id = $1 AND COALESCE(group_id, '') <> ''
	`, snapshot.ID); err != nil {
		t.Fatalf("restore escalation group binding: %v", err)
	}
}

func base64PNG1x1() string {
	return "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="
}

func findContactItem(items []map[string]any, phone string) map[string]any {
	for _, item := range items {
		if item["phone"] == phone {
			return item
		}
	}
	return nil
}

func findItemByID(items []map[string]any, id string) map[string]any {
	for _, item := range items {
		if item["id"] == id {
			return item
		}
	}
	return nil
}

func createTestAgent(t *testing.T, pool *pgxpool.Pool, role string) (string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	username := fmt.Sprintf("tdd_%s_%d", strings.ReplaceAll(role, "_", ""), time.Now().UnixNano())
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agents (name, username, password_hash, role, is_active)
		VALUES ($1, $2, 'test-password-hash', $3::agent_role, TRUE)
		RETURNING id
	`, "TDD "+role, username, role).Scan(&id); err != nil {
		t.Fatalf("create test agent %s: %v", role, err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(ctx, `DELETE FROM agents WHERE id = $1`, id)
	})
	return id, username
}

func createTestConversation(t *testing.T, pool *pgxpool.Pool, phone string, mode string, assignedTo *string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cleanupContacts(t, pool, phone)
	t.Cleanup(func() { cleanupContacts(t, pool, phone) })
	var contactID, conversationID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO contacts (phone, name)
		VALUES ($1, 'Conversation TDD')
		RETURNING id
	`, phone).Scan(&contactID); err != nil {
		t.Fatalf("insert contact: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO conversations (contact_id, channel, mode, status, assigned_to, last_message_at)
		VALUES ($1, 'whatsapp', $2::conversation_mode, 'open', NULLIF($3, '')::uuid, NOW())
		RETURNING id
	`, contactID, mode, nullableTestString(assignedTo)).Scan(&conversationID); err != nil {
		t.Fatalf("insert conversation: %v", err)
	}
	return conversationID
}

func nullableTestString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type waMock struct {
	server        *httptest.Server
	textSends     int
	mediaSends    int
	lastPath      string
	lastTextPhone string
	lastTextBody  string
}

func newWAMock(t *testing.T, connected bool) *waMock {
	t.Helper()
	mock := &waMock{}
	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.lastPath = r.URL.Path
		switch r.URL.Path {
		case "/api/status":
			status := "disconnected"
			if connected {
				status = "connected"
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": status})
		case "/api/send-text":
			mock.textSends++
			var req struct {
				Phone string `json:"phone"`
				Text  string `json:"text"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			mock.lastTextPhone = req.Phone
			mock.lastTextBody = req.Text
			writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "externalMessageId": fmt.Sprintf("text-%d", mock.textSends)})
		case "/api/send-media":
			mock.mediaSends++
			_, _ = io.Copy(io.Discard, r.Body)
			writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "externalMessageId": fmt.Sprintf("media-%d", mock.mediaSends), "mediaKind": "image"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(mock.server.Close)
	return mock
}
