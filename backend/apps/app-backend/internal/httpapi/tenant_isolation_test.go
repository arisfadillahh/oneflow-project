package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func testTokenForAgentOrg(t *testing.T, role, agentID, organizationID, organizationRole string) string {
	t.Helper()
	claims := agentClaims{
		AgentID:          agentID,
		Role:             role,
		OrganizationID:   organizationID,
		OrganizationRole: organizationRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Subject:   agentID,
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func TestTenantIsolationOrgACannotReadOrMutateOrgBConversation(t *testing.T) {
	pool := testPool(t)
	httpServer := testServer(t, pool)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suffix := time.Now().UnixNano()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	var orgA, orgB, agentA, contactA, contactB, conversationA, conversationB string
	if err := pool.QueryRow(ctx, `INSERT INTO organizations (name, slug) VALUES ($1, $2) RETURNING id`, "Tenant A", fmt.Sprintf("tenant-a-%d", suffix)).Scan(&orgA); err != nil {
		t.Fatalf("insert org a: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO organizations (name, slug) VALUES ($1, $2) RETURNING id`, "Tenant B", fmt.Sprintf("tenant-b-%d", suffix)).Scan(&orgB); err != nil {
		t.Fatalf("insert org b: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(ctx, `DELETE FROM organizations WHERE id IN ($1, $2)`, orgA, orgB)
	})
	if err := pool.QueryRow(ctx, `
		INSERT INTO agents (name, username, password_hash, role, current_organization_id)
		VALUES ('Tenant A Admin', $1, $2, 'admin', $3)
		RETURNING id
	`, fmt.Sprintf("tenant-a-admin-%d", suffix), string(passwordHash), orgA).Scan(&agentA); err != nil {
		t.Fatalf("insert agent a: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO organization_members (organization_id, agent_id, role, status) VALUES ($1, $2, 'org_admin', 'active')`, orgA, agentA); err != nil {
		t.Fatalf("insert member a: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO contacts (phone, name, organization_id) VALUES ($1, 'Org A Customer', $2) RETURNING id`, fmt.Sprintf("+628111%d", suffix%1_000_000), orgA).Scan(&contactA); err != nil {
		t.Fatalf("insert contact a: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO contacts (phone, name, organization_id) VALUES ($1, 'Org B Customer', $2) RETURNING id`, fmt.Sprintf("+628222%d", suffix%1_000_000), orgB).Scan(&contactB); err != nil {
		t.Fatalf("insert contact b: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO conversations (contact_id, channel, mode, status, last_message_at, organization_id) VALUES ($1, 'whatsapp', 'human', 'open', NOW(), $2) RETURNING id`, contactA, orgA).Scan(&conversationA); err != nil {
		t.Fatalf("insert conversation a: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO conversations (contact_id, channel, mode, status, last_message_at, organization_id) VALUES ($1, 'whatsapp', 'human', 'open', NOW(), $2) RETURNING id`, contactB, orgB).Scan(&conversationB); err != nil {
		t.Fatalf("insert conversation b: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO messages (conversation_id, sender_type, direction, content_type, text, sent_at, delivered_at, organization_id)
		VALUES ($1, 'customer', 'inbound', 'text', 'A secret', NOW(), NOW(), $2),
		       ($3, 'customer', 'inbound', 'text', 'B secret', NOW(), NOW(), $4)
	`, conversationA, orgA, conversationB, orgB); err != nil {
		t.Fatalf("insert messages: %v", err)
	}

	token := testTokenForAgentOrg(t, "operator", agentA, orgA, "org_admin")
	req, err := http.NewRequest(http.MethodGet, httpServer.URL+"/api/contacts", nil)
	if err != nil {
		t.Fatalf("build contacts request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("contacts request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("contacts status = %d, want 200", resp.StatusCode)
	}
	var contactsPayload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&contactsPayload); err != nil {
		t.Fatalf("decode contacts: %v", err)
	}
	for _, item := range contactsPayload.Items {
		if item["contactName"] == "Org B Customer" || item["lastMessageText"] == "B secret" {
			t.Fatalf("org A contacts leaked org B item: %v", item)
		}
	}

	token = testTokenForAgentOrg(t, "admin", agentA, orgA, "org_admin")

	req, err = http.NewRequest(http.MethodGet, httpServer.URL+"/api/conversations/"+conversationB, nil)
	if err != nil {
		t.Fatalf("build conversation detail request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("conversation detail request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("org A reading org B conversation status = %d, want 404", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodPost, httpServer.URL+"/api/conversations/"+conversationB+"/return-to-ai", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("build mutation request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("mutation request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		t.Fatalf("org A mutating org B conversation status = %d body=%v, want 403", resp.StatusCode, body)
	}
}
