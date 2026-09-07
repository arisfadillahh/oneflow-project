package httpapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"oneflow/wa-gateway/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type metaCloudTransport struct {
	cfg        config.Config
	db         *pgxpool.Pool
	httpClient *http.Client
}

func newMetaCloudTransport(cfg config.Config, db *pgxpool.Pool, client *http.Client) *metaCloudTransport {
	return &metaCloudTransport{cfg: cfg, db: db, httpClient: client}
}

func (m *metaCloudTransport) Status(ctx context.Context) (map[string]any, error) {
	return readStatusRow(ctx, m.db)
}

func (m *metaCloudTransport) GetQR(ctx context.Context) (map[string]any, error) {
	return nil, errors.New("official WhatsApp uses Meta Embedded Signup in the dashboard")
}

func (m *metaCloudTransport) PairPhone(ctx context.Context, phone string) (map[string]any, error) {
	return nil, errors.New("official WhatsApp pairing is completed through Meta Embedded Signup")
}

func (m *metaCloudTransport) Connect(ctx context.Context) (map[string]any, error) {
	status, err := readStatusRow(ctx, m.db)
	if err != nil {
		return nil, err
	}
	if status["status"] == "connected" {
		return status, nil
	}
	return map[string]any{
		"status": "disconnected",
		"details": map[string]any{
			"provider":           "meta_cloud",
			"onboardingMode":     "coexistence",
			"requiresMetaSignup": true,
		},
	}, nil
}

func (m *metaCloudTransport) Disconnect(ctx context.Context) (map[string]any, error) {
	details := map[string]any{
		"provider":       "meta_cloud",
		"onboardingMode": "coexistence",
		"official":       true,
		"disconnectedAt": time.Now().UTC(),
	}
	if err := upsertStatus(ctx, m.db, "disconnected", details); err != nil {
		return nil, err
	}
	return map[string]any{"status": "disconnected", "details": details}, nil
}

func (m *metaCloudTransport) SendText(ctx context.Context, phone, text string) (map[string]any, error) {
	if !m.cfg.MetaCloudEnabled {
		return nil, errors.New("official WhatsApp connection is disabled")
	}
	credentials, err := m.credentials(ctx)
	if err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                normalizeMetaRecipientPhone(phone),
		"type":              "text",
		"text":              map[string]any{"body": text, "preview_url": false},
	})
	endpoint := fmt.Sprintf("%s/%s/%s/messages", m.cfg.MetaGraphAPIBaseURL, m.cfg.MetaGraphAPIVersion, url.PathEscape(credentials.PhoneNumberID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+credentials.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("Meta send message returned %d", resp.StatusCode)
	}
	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Messages) == 0 {
		return nil, errors.New("Meta send message returned no message id")
	}
	return map[string]any{
		"status":            "sent",
		"mode":              "meta_cloud",
		"externalMessageId": result.Messages[0].ID,
		"phone":             phone,
	}, nil
}

type metaCloudCredentials struct {
	PhoneNumberID string
	WABAID        string
	Token         string
}

type metaTemplateDraft struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Language string `json:"language"`
	Body     string `json:"body"`
}

type metaTemplateSendRequest struct {
	Phone    string `json:"phone"`
	Name     string `json:"name"`
	Language string `json:"language"`
}

var metaTemplateNamePattern = regexp.MustCompile(`^[a-z0-9_]{1,512}$`)
var metaTemplateLanguagePattern = regexp.MustCompile(`^[a-z]{2}(?:_[A-Z]{2})?$`)

func validateMetaTemplateDraft(draft *metaTemplateDraft) error {
	draft.Name = strings.TrimSpace(strings.ToLower(draft.Name))
	draft.Category = strings.ToUpper(strings.TrimSpace(draft.Category))
	draft.Language = strings.TrimSpace(draft.Language)
	draft.Body = strings.TrimSpace(draft.Body)
	if !metaTemplateNamePattern.MatchString(draft.Name) {
		return errors.New("template name must contain lowercase letters, numbers, or underscore only")
	}
	if draft.Category != "UTILITY" && draft.Category != "MARKETING" {
		return errors.New("template category must be UTILITY or MARKETING")
	}
	if !metaTemplateLanguagePattern.MatchString(draft.Language) {
		return errors.New("template language is invalid")
	}
	if draft.Body == "" || len([]rune(draft.Body)) > 1024 {
		return errors.New("template body is required and must be at most 1024 characters")
	}
	return nil
}

func validateMetaTemplateSendRequest(request *metaTemplateSendRequest) error {
	request.Phone = normalizeMetaRecipientPhone(request.Phone)
	request.Name = strings.TrimSpace(strings.ToLower(request.Name))
	request.Language = strings.TrimSpace(request.Language)
	if request.Phone == "" {
		return errors.New("phone is required")
	}
	if !metaTemplateNamePattern.MatchString(request.Name) {
		return errors.New("template name is invalid")
	}
	if !metaTemplateLanguagePattern.MatchString(request.Language) {
		return errors.New("template language is invalid")
	}
	return nil
}

func normalizeMetaRecipientPhone(phone string) string {
	var digits strings.Builder
	for _, character := range phone {
		if character >= '0' && character <= '9' {
			digits.WriteRune(character)
		}
	}
	return digits.String()
}

func (m *metaCloudTransport) ListTemplates(ctx context.Context) (map[string]any, error) {
	if !m.cfg.MetaCloudEnabled {
		return nil, errors.New("official WhatsApp connection is disabled")
	}
	credentials, err := m.credentials(ctx)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/%s/%s/message_templates?fields=id,name,status,category,language,components&limit=100", m.cfg.MetaGraphAPIBaseURL, m.cfg.MetaGraphAPIVersion, url.PathEscape(credentials.WABAID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+credentials.Token)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("Meta list templates returned %d", resp.StatusCode)
	}
	var result struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.New("Meta list templates returned invalid data")
	}
	return map[string]any{"items": result.Data}, nil
}

func (m *metaCloudTransport) CreateTemplate(ctx context.Context, draft metaTemplateDraft) (map[string]any, error) {
	if !m.cfg.MetaCloudEnabled {
		return nil, errors.New("official WhatsApp connection is disabled")
	}
	if err := validateMetaTemplateDraft(&draft); err != nil {
		return nil, err
	}
	credentials, err := m.credentials(ctx)
	if err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{
		"name":     draft.Name,
		"category": draft.Category,
		"language": draft.Language,
		"components": []map[string]any{{
			"type": "BODY",
			"text": draft.Body,
		}},
	})
	endpoint := fmt.Sprintf("%s/%s/%s/message_templates", m.cfg.MetaGraphAPIBaseURL, m.cfg.MetaGraphAPIVersion, url.PathEscape(credentials.WABAID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+credentials.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("Meta create template returned %d", resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.New("Meta create template returned invalid data")
	}
	result["name"] = draft.Name
	result["category"] = draft.Category
	result["language"] = draft.Language
	return result, nil
}

func (m *metaCloudTransport) SendTemplate(ctx context.Context, request metaTemplateSendRequest) (map[string]any, error) {
	if !m.cfg.MetaCloudEnabled {
		return nil, errors.New("official WhatsApp connection is disabled")
	}
	if err := validateMetaTemplateSendRequest(&request); err != nil {
		return nil, err
	}
	credentials, err := m.credentials(ctx)
	if err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                request.Phone,
		"type":              "template",
		"template": map[string]any{
			"name": request.Name,
			"language": map[string]string{
				"code": request.Language,
			},
		},
	})
	endpoint := fmt.Sprintf("%s/%s/%s/messages", m.cfg.MetaGraphAPIBaseURL, m.cfg.MetaGraphAPIVersion, url.PathEscape(credentials.PhoneNumberID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+credentials.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("Meta send template returned %d", resp.StatusCode)
	}
	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Messages) == 0 {
		return nil, errors.New("Meta send template returned no message id")
	}
	return map[string]any{
		"status":            "sent",
		"mode":              "meta_cloud",
		"externalMessageId": result.Messages[0].ID,
		"phone":             request.Phone,
		"templateName":      request.Name,
	}, nil
}

func (s *Server) handleMetaTemplates(w http.ResponseWriter, r *http.Request) {
	transport, ok := s.metaTransport.(*metaCloudTransport)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "official WhatsApp transport is unavailable"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		result, err := transport.ListTemplates(r.Context())
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodPost:
		var request metaTemplateDraft
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid template request"})
			return
		}
		if err := validateMetaTemplateDraft(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		result, err := transport.CreateTemplate(r.Context(), request)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleMetaTemplateSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	transport, ok := s.metaTransport.(*metaCloudTransport)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "official WhatsApp transport is unavailable"})
		return
	}
	var request metaTemplateSendRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid template message request"})
		return
	}
	if err := validateMetaTemplateSendRequest(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	result, err := transport.SendTemplate(r.Context(), request)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (m *metaCloudTransport) SendMedia(ctx context.Context, media outboundMediaRequest) (map[string]any, error) {
	return nil, errors.New("official WhatsApp media sending is not enabled yet")
}

func (m *metaCloudTransport) SetChatPresence(ctx context.Context, phone string, active bool) (map[string]any, error) {
	return map[string]any{"status": "ok", "mode": "meta_cloud", "phone": phone}, nil
}

func (m *metaCloudTransport) credentials(ctx context.Context) (metaCloudCredentials, error) {
	sessionID := sessionIDFromContext(ctx)
	if sessionID == "" {
		return metaCloudCredentials{}, errors.New("official WhatsApp session is required")
	}
	var phoneNumberID, wabaID, ciphertext, status string
	err := m.db.QueryRow(ctx, `
		SELECT COALESCE(meta_phone_number_id, ''), COALESCE(meta_waba_id, ''), COALESCE(meta_business_token_ciphertext, ''), status
		FROM whatsapp_sessions
		WHERE id = $1 AND deleted_at IS NULL AND provider = 'meta_cloud'
	`, sessionID).Scan(&phoneNumberID, &wabaID, &ciphertext, &status)
	if err != nil {
		return metaCloudCredentials{}, err
	}
	if status != "connected" || phoneNumberID == "" || wabaID == "" || ciphertext == "" {
		return metaCloudCredentials{}, errors.New("official WhatsApp session is not connected")
	}
	token, err := decryptMetaCredential(ciphertext, m.cfg.MetaCredentialKey)
	if err != nil {
		return metaCloudCredentials{}, err
	}
	return metaCloudCredentials{PhoneNumberID: phoneNumberID, WABAID: wabaID, Token: token}, nil
}

func decryptMetaCredential(value, configuredKey string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(configuredKey)
	if err != nil || len(key) != 32 {
		return "", errors.New("official WhatsApp credentials are not configured safely")
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", errors.New("official WhatsApp credential is invalid")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("official WhatsApp credential is invalid")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("official WhatsApp credential cannot be decrypted")
	}
	return string(plain), nil
}

type metaWebhookMessage struct {
	From      string `json:"from"`
	To        string `json:"to"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      struct {
		Body string `json:"body"`
	} `json:"text"`
}

type metaWebhookEnvelope struct {
	Entry []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				Metadata struct {
					PhoneNumberID string `json:"phone_number_id"`
				} `json:"metadata"`
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaID string `json:"wa_id"`
				} `json:"contacts"`
				Messages      []metaWebhookMessage `json:"messages"`
				MessageEchoes []metaWebhookMessage `json:"message_echoes"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

type metaSyncWebhook struct {
	ID    string `json:"id"`
	Event string `json:"event"`
	Data  struct {
		Metadata struct {
			PhoneNumberID string `json:"phone_number_id"`
		} `json:"metadata"`
		History []struct {
			Errors []struct {
				Code int `json:"code"`
			} `json:"errors"`
		} `json:"history"`
	} `json:"data"`
}

func metaHistorySyncStatus(event metaSyncWebhook) string {
	status := "received"
	for _, history := range event.Data.History {
		for _, itemErr := range history.Errors {
			if itemErr.Code == 2593109 {
				return "declined"
			}
			status = "failed"
		}
	}
	return status
}

func (s *Server) handleMetaWebhook(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.MetaWebhookEnabled {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		if r.URL.Query().Get("hub.mode") == "subscribe" && hmac.Equal([]byte(r.URL.Query().Get("hub.verify_token")), []byte(s.cfg.MetaWebhookVerifyToken)) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(r.URL.Query().Get("hub.challenge")))
			return
		}
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid webhook verification"})
		return
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid webhook payload"})
		return
	}
	if !validMetaSignature(raw, r.Header.Get("X-Hub-Signature-256"), s.cfg.MetaAppSecret) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid webhook signature"})
		return
	}
	payloadHash := sha256.Sum256(raw)
	hashText := hex.EncodeToString(payloadHash[:])

	var syncEvent metaSyncWebhook
	if err := json.Unmarshal(raw, &syncEvent); err == nil && (syncEvent.Event == "history" || syncEvent.Event == "smb_app_state_sync") {
		sessionID, err := s.metaSessionID(r.Context(), syncEvent.Data.Metadata.PhoneNumberID)
		if err == nil && sessionID != "" {
			eventKey := syncEvent.Event + ":" + strings.TrimSpace(syncEvent.ID) + ":" + hashText
			_, _ = s.recordMetaWebhookEvent(r.Context(), eventKey, sessionID, syncEvent.Event, hashText)
			if syncEvent.Event == "history" {
				_, _ = s.db.Exec(r.Context(), `
					UPDATE whatsapp_sessions
					SET meta_history_sync_status = $2, updated_at = NOW()
					WHERE id = $1 AND provider = 'meta_cloud'
				`, sessionID, metaHistorySyncStatus(syncEvent))
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
		return
	}

	var envelope metaWebhookEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid webhook payload"})
		return
	}
	for _, entry := range envelope.Entry {
		for _, change := range entry.Changes {
			sessionID, err := s.metaSessionID(r.Context(), change.Value.Metadata.PhoneNumberID)
			if err != nil || sessionID == "" {
				continue
			}
			for _, message := range change.Value.Messages {
				eventKey := "message:" + message.ID
				if change.Field != "messages" {
					_, _ = s.recordMetaWebhookEvent(r.Context(), eventKey, sessionID, change.Field, hashText)
					continue
				}
				text := strings.TrimSpace(message.Text.Body)
				if text == "" {
					_, _ = s.recordMetaWebhookEvent(r.Context(), eventKey, sessionID, change.Field, hashText)
					continue
				}
				receivedAt := time.Now().UTC()
				if timestamp, parseErr := strconv.ParseInt(message.Timestamp, 10, 64); parseErr == nil {
					receivedAt = time.Unix(timestamp, 0).UTC()
				}
				name := ""
				for _, contact := range change.Value.Contacts {
					if normalizePhone(contact.WaID) == normalizePhone(message.From) {
						name = strings.TrimSpace(contact.Profile.Name)
						break
					}
				}
				payload := map[string]any{
					"whatsappSessionId": sessionID,
					"phone":             "+" + normalizePhone(message.From),
					"customerName":      name,
					"text":              text,
					"externalMessageId": message.ID,
					"receivedAt":        receivedAt,
					"contentType":       "text",
				}
				headers := map[string]string{"X-Internal-Token": s.cfg.InternalGatewayToken}
				if err := postJSON(r.Context(), s.httpClient, s.cfg.AppBackendBaseURL+"/api/internal/wa/inbound", headers, payload); err != nil {
					writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to store incoming WhatsApp message"})
					return
				}
				if _, err := s.recordMetaWebhookEvent(r.Context(), eventKey, sessionID, change.Field, hashText); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to record incoming WhatsApp event"})
					return
				}
			}
			for _, message := range change.Value.MessageEchoes {
				eventKey := "echo:" + message.ID
				if change.Field != "smb_message_echoes" {
					_, _ = s.recordMetaWebhookEvent(r.Context(), eventKey, sessionID, change.Field, hashText)
					continue
				}
				text := strings.TrimSpace(message.Text.Body)
				customerPhone := normalizePhone(message.To)
				if text == "" || customerPhone == "" {
					_, _ = s.recordMetaWebhookEvent(r.Context(), eventKey, sessionID, change.Field, hashText)
					continue
				}
				sentAt := time.Now().UTC()
				if timestamp, parseErr := strconv.ParseInt(message.Timestamp, 10, 64); parseErr == nil {
					sentAt = time.Unix(timestamp, 0).UTC()
				}
				payload := map[string]any{
					"whatsappSessionId": sessionID,
					"phone":             "+" + customerPhone,
					"text":              text,
					"externalMessageId": message.ID,
					"sentAt":            sentAt,
				}
				headers := map[string]string{"X-Internal-Token": s.cfg.InternalGatewayToken}
				if err := postJSON(r.Context(), s.httpClient, s.cfg.AppBackendBaseURL+"/api/internal/wa/echo", headers, payload); err != nil {
					writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to store WhatsApp Business app echo"})
					return
				}
				if _, err := s.recordMetaWebhookEvent(r.Context(), eventKey, sessionID, change.Field, hashText); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to record WhatsApp Business app echo"})
					return
				}
			}
			if len(change.Value.Messages) == 0 && len(change.Value.MessageEchoes) == 0 {
				eventKey := change.Field + ":" + entry.ID + ":" + hashText
				_, _ = s.recordMetaWebhookEvent(r.Context(), eventKey, sessionID, change.Field, hashText)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (s *Server) metaSessionID(ctx context.Context, phoneNumberID string) (string, error) {
	phoneNumberID = strings.TrimSpace(phoneNumberID)
	if phoneNumberID == "" {
		return "", pgx.ErrNoRows
	}
	var sessionID string
	err := s.db.QueryRow(ctx, `
		SELECT id::text
		FROM whatsapp_sessions
		WHERE provider = 'meta_cloud'
		  AND meta_phone_number_id = $1
		  AND status = 'connected'
		  AND deleted_at IS NULL
		LIMIT 1
	`, phoneNumberID).Scan(&sessionID)
	return sessionID, err
}

func (s *Server) recordMetaWebhookEvent(ctx context.Context, key, sessionID, eventType, payloadHash string) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		INSERT INTO whatsapp_meta_webhook_events (event_key, whatsapp_session_id, event_type, payload_hash)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4)
		ON CONFLICT (event_key) DO NOTHING
	`, key, sessionID, eventType, payloadHash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func validMetaSignature(payload []byte, signature, secret string) bool {
	if strings.TrimSpace(secret) == "" || !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	expected := hmac.New(sha256.New, []byte(secret))
	_, _ = expected.Write(payload)
	provided, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return false
	}
	return hmac.Equal(expected.Sum(nil), provided)
}
