package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// handleInternalWAEcho stores messages sent from the WhatsApp Business app during Coexistence.
// Echoes are human-originated output and must not trigger AI generation or customer-memory writes.
func (s *Server) handleInternalWAEcho(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if !s.cfg.MetaCloudEnabled {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMediaJSONBodyBytes)
	var req struct {
		WhatsAppSessionID string    `json:"whatsappSessionId"`
		Phone             string    `json:"phone"`
		Text              string    `json:"text"`
		ExternalMessageID string    `json:"externalMessageId"`
		SentAt            time.Time `json:"sentAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.WhatsAppSessionID = strings.TrimSpace(req.WhatsAppSessionID)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Text = strings.TrimSpace(req.Text)
	req.ExternalMessageID = strings.TrimSpace(req.ExternalMessageID)
	if req.Phone == "" || req.Text == "" || req.ExternalMessageID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone, text, and externalMessageId are required"})
		return
	}
	if req.SentAt.IsZero() {
		req.SentAt = time.Now().UTC()
	}

	sessionID, organizationID, err := s.resolveInboundWhatsAppSession(r.Context(), req.WhatsAppSessionID, "")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid whatsapp session is required"})
		return
	}
	r = r.WithContext(context.WithValue(r.Context(), contextOrganizationID, organizationID))
	r = r.WithContext(context.WithValue(r.Context(), contextWhatsAppSession, sessionID))
	var provider, status string
	err = s.db.QueryRow(r.Context(), `
		SELECT COALESCE(provider, 'whatsmeow'), status::text
		FROM whatsapp_sessions
		WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
	`, sessionID, organizationID).Scan(&provider, &status)
	if err != nil || provider != "meta_cloud" || status != "connected" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "connected official whatsapp session is required"})
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	var duplicateConversationID string
	err = tx.QueryRow(r.Context(), `
		SELECT conversation_id::text
		FROM messages
		WHERE external_message_id = $1 AND organization_id = $2
	`, req.ExternalMessageID, organizationID).Scan(&duplicateConversationID)
	if err == nil {
		_ = tx.Commit(r.Context())
		writeJSON(w, http.StatusOK, map[string]any{"status": "duplicate", "conversationId": duplicateConversationID})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rawPhone := req.Phone
	resolvedPhone, identityPayload, err := s.resolveWhatsAppInboundPhoneTx(r.Context(), tx, rawPhone)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	contactID, _, err := s.resolveWhatsAppContactTx(r.Context(), tx, resolvedPhone, rawPhone, "", false, true)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var conversationID string
	err = tx.QueryRow(r.Context(), `
		SELECT id::text
		FROM conversations
		WHERE contact_id = $1 AND organization_id = $2 AND whatsapp_session_id = $3
		FOR UPDATE
	`, contactID, organizationID, sessionID).Scan(&conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		var aiAgentID string
		aiAgentID, _ = s.aiAgentIDForWhatsAppSession(r.Context(), sessionID)
		err = tx.QueryRow(r.Context(), `
			INSERT INTO conversations (contact_id, channel, mode, status, last_message_at, organization_id, whatsapp_session_id, ai_agent_id)
			VALUES ($1, 'whatsapp', 'human', 'open', $2, $3, $4, NULLIF($5, '')::uuid)
			RETURNING id::text
		`, contactID, req.SentAt, organizationID, sessionID, aiAgentID).Scan(&conversationID)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rawPayload := map[string]any{
		"source":            "meta_smb_message_echo",
		"phone":             resolvedPhone,
		"rawPhone":          rawPhone,
		"identityResolver":  identityPayload,
		"externalMessageId": req.ExternalMessageID,
		"whatsappSessionId": sessionID,
	}
	var messageID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO messages (conversation_id, external_message_id, sender_type, direction, content_type, text, raw_payload, sent_at, organization_id)
		VALUES ($1, $2, 'agent', 'outbound', 'text', $3, $4::jsonb, $5, $6)
		RETURNING id::text
	`, conversationID, req.ExternalMessageID, req.Text, marshalJSON(rawPayload), req.SentAt, organizationID).Scan(&messageID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	_, err = tx.Exec(r.Context(), `
		UPDATE conversations
		SET last_message_id = $2,
		    last_message_at = $3,
		    mode = 'human',
		    status = 'open',
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $4
	`, conversationID, messageID, req.SentAt, organizationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertSystemEventTx(r.Context(), tx, conversationID, "meta_business_app_echo", map[string]any{
		"source":              "meta_smb_message_echo",
		"external_message_id": req.ExternalMessageID,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "stored", "conversationId": conversationID, "messageId": messageID})
}
