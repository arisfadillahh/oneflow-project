package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func officialWhatsAppProvider(input string) (string, error) {
	provider := strings.TrimSpace(input)
	if provider == "" || provider == "meta_cloud" {
		return "meta_cloud", nil
	}
	return "", fmt.Errorf("unofficial WhatsApp has been retired; use Meta Coexistence")
}

func (s *Server) handleWhatsAppSessions(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" && role != "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := s.db.Query(r.Context(), `
			SELECT ws.id::text, ws.label, COALESCE(ws.phone_number, ''), ws.mode, ws.status, ws.session_key, ws.is_default,
			       ws.details, ws.created_at, ws.updated_at, COALESCE(ws.ai_agent_id::text, ''), COALESCE(aa.name, ''),
			       COALESCE(ws.provider, 'whatsmeow'), COALESCE(ws.onboarding_mode, ''),
			       COALESCE(ws.meta_waba_id, ''), COALESCE(ws.meta_phone_number_id, ''),
			       ws.meta_history_sync_status, ws.meta_history_sync_deadline
			FROM whatsapp_sessions ws
			LEFT JOIN ai_agents aa ON aa.id = ws.ai_agent_id AND aa.organization_id = ws.organization_id
			WHERE ws.organization_id = $1
			  AND ws.deleted_at IS NULL
			ORDER BY is_default DESC, updated_at DESC
		`, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, label, phone, mode, status, sessionKey, aiAgentID, aiAgentName, provider, onboardingMode, metaWABAID, metaPhoneNumberID, historySyncStatus string
			var isDefault bool
			var details []byte
			var createdAt, updatedAt time.Time
			var historySyncDeadline *time.Time
			if err := rows.Scan(&id, &label, &phone, &mode, &status, &sessionKey, &isDefault, &details, &createdAt, &updatedAt, &aiAgentID, &aiAgentName, &provider, &onboardingMode, &metaWABAID, &metaPhoneNumberID, &historySyncStatus, &historySyncDeadline); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			itemDetails := map[string]any{}
			_ = json.Unmarshal(details, &itemDetails)
			items = append(items, map[string]any{
				"id": id, "label": label, "phoneNumber": phone, "mode": mode, "status": status,
				"sessionKey": sessionKey, "isDefault": isDefault, "details": itemDetails,
				"aiAgentId": aiAgentID, "aiAgentName": aiAgentName,
				"createdAt": createdAt, "updatedAt": updatedAt,
				"provider": provider, "onboardingMode": onboardingMode,
				"metaWabaId": metaWABAID, "metaPhoneNumberId": metaPhoneNumberID,
				"historySyncStatus": historySyncStatus, "historySyncDeadline": metaHistoryDeadlineLabel(historySyncDeadline),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		if role != "owner" && role != "super_admin" && role != "admin" && role != "operator" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		var req struct {
			Label     string `json:"label"`
			Mode      string `json:"mode"`
			Provider  string `json:"provider"`
			AIAgentID string `json:"aiAgentId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.Label = normalizeWhatsAppSessionLabel(req.Label)
		req.Mode = strings.TrimSpace(req.Mode)
		if req.Mode == "" {
			req.Mode = "real"
		}
		if req.Mode != "real" && req.Mode != "mock" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid mode"})
			return
		}
		provider, providerErr := officialWhatsAppProvider(req.Provider)
		if providerErr != nil {
			writeJSON(w, http.StatusGone, map[string]string{"error": providerErr.Error()})
			return
		}
		req.Provider = provider
		if req.Provider == "meta_cloud" {
			if role != "owner" && role != "super_admin" && role != "admin" {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "only an organization admin can connect official WhatsApp"})
				return
			}
			if !s.cfg.MetaCloudEnabled {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "official WhatsApp connection is not enabled"})
				return
			}
			req.Mode = "real"
		}
		req.AIAgentID = strings.TrimSpace(req.AIAgentID)
		if req.AIAgentID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "aiAgentId is required"})
			return
		}
		if ok, err := s.aiAgentBelongsToOrganization(r.Context(), req.AIAgentID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		} else if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid aiAgentId is required"})
			return
		}
		if err := s.ensurePlanCapacity(r.Context(), "whatsapp"); err != nil {
			writeJSON(w, http.StatusPaymentRequired, map[string]string{"error": err.Error()})
			return
		}
		var id string
		err := s.db.QueryRow(r.Context(), `
			INSERT INTO whatsapp_sessions (organization_id, label, mode, status, session_key, is_default, ai_agent_id, created_by, provider, onboarding_mode)
			VALUES ($1, $2, $3, 'disconnected', $4, FALSE, $5, NULLIF($6, '')::uuid, $7, CASE WHEN $7 = 'meta_cloud' THEN 'coexistence' ELSE NULL END)
			RETURNING id
		`, s.organizationID(r.Context()), req.Label, req.Mode, sessionKeyForOrganization(s.organizationID(r.Context())), req.AIAgentID, s.agentID(r.Context()), req.Provider).Scan(&id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		_ = s.insertAuditLog(r.Context(), "whatsapp_session.create", "whatsapp_session", id, map[string]any{"label": req.Label, "ai_agent_id": req.AIAgentID, "provider": req.Provider})
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": id})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleWhatsAppSessionRoutes(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" && role != "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/whatsapp/sessions/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	session, err := s.requireOrganizationSession(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	if action == "" {
		switch r.Method {
		case http.MethodPut:
			if role != "owner" && role != "super_admin" && role != "admin" && role != "operator" {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
				return
			}
			var req struct {
				Label string `json:"label"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
				return
			}
			label := normalizeWhatsAppSessionLabel(req.Label)
			_, err := s.db.Exec(r.Context(), `UPDATE whatsapp_sessions SET label = $3, updated_at = NOW() WHERE id = $1 AND organization_id = $2`, id, s.organizationID(r.Context()), label)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			_ = s.insertAuditLog(r.Context(), "whatsapp_session.rename", "whatsapp_session", id, map[string]any{"label": label})
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case http.MethodDelete:
			if role != "owner" && role != "super_admin" && role != "admin" && role != "operator" {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
				return
			}
			_ = s.disconnectWhatsAppSession(r.Context(), id)
			tag, err := s.db.Exec(r.Context(), `
				UPDATE whatsapp_sessions
				SET status = 'disconnected',
				    is_default = FALSE,
				    meta_business_token_ciphertext = CASE WHEN provider = 'meta_cloud' THEN NULL ELSE meta_business_token_ciphertext END,
				    last_disconnected_at = NOW(),
				    deleted_at = NOW(),
				    updated_at = NOW()
				WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
			`, id, s.organizationID(r.Context()))
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if tag.RowsAffected() == 0 {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "connected default session cannot be deleted"})
				return
			}
			_ = s.insertAuditLog(r.Context(), "whatsapp_session.delete", "whatsapp_session", id, map[string]any{})
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
		return
	}

	switch action {
	case "default":
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if role != "owner" && role != "super_admin" && role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		tx, err := s.db.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer tx.Rollback(r.Context())
		if _, err := tx.Exec(r.Context(), `UPDATE whatsapp_sessions SET is_default = FALSE, updated_at = NOW() WHERE organization_id = $1 AND deleted_at IS NULL`, s.organizationID(r.Context())); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if _, err := tx.Exec(r.Context(), `UPDATE whatsapp_sessions SET is_default = TRUE, updated_at = NOW() WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL`, id, s.organizationID(r.Context())); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		_ = s.insertAuditLog(r.Context(), "whatsapp_session.set_default", "whatsapp_session", id, map[string]any{})
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case "status":
		s.proxyWhatsAppSession(w, r, session, http.MethodGet, "status", nil)
	case "qr":
		writeJSON(w, http.StatusGone, map[string]string{"error": "unofficial WhatsApp has been retired; use Meta Coexistence"})
		return
	case "connect":
		writeJSON(w, http.StatusGone, map[string]string{"error": "use Meta Embedded Signup to connect official WhatsApp"})
		return
	case "disconnect":
		s.proxyWhatsAppSession(w, r, session, http.MethodPost, "disconnect", nil)
		if session["provider"] == "meta_cloud" {
			_, _ = s.db.Exec(r.Context(), `
				UPDATE whatsapp_sessions
				SET meta_business_token_ciphertext = NULL,
				    updated_at = NOW()
				WHERE id = $1 AND organization_id = $2
			`, session["id"], s.organizationID(r.Context()))
		}
	case "meta-templates":
		if role != "owner" && role != "super_admin" && role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only an organization admin can manage official WhatsApp templates"})
			return
		}
		if session["provider"] != "meta_cloud" || !s.cfg.MetaCloudEnabled {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "official WhatsApp connection is not enabled for this session"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.proxyWhatsAppSession(w, r, session, http.MethodGet, "meta-templates", nil)
		case http.MethodPost:
			var payload map[string]any
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&payload); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid template request"})
				return
			}
			s.proxyWhatsAppSession(w, r, session, http.MethodPost, "meta-templates", payload)
		default:
			http.NotFound(w, r)
		}
	case "meta-template-send":
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if role != "owner" && role != "super_admin" && role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only an organization admin can send official WhatsApp templates"})
			return
		}
		if session["provider"] != "meta_cloud" || !s.cfg.MetaCloudEnabled {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "official WhatsApp connection is not enabled for this session"})
			return
		}
		if err := s.ensureWhatsAppPlanAccess(r.Context()); err != nil {
			writeJSON(w, http.StatusPaymentRequired, map[string]string{"error": err.Error()})
			return
		}
		if !s.enforceRateLimit(w, r, s.waLimit, "wa:meta-template-send:"+id) {
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid template message request"})
			return
		}
		s.proxyWhatsAppSession(w, r, session, http.MethodPost, "meta-template-send", payload)
	case "meta-complete":
		s.handleMetaCloudComplete(w, r, session)
	case "meta-test-connect":
		s.handleMetaTestConnect(w, r, session)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) proxyWhatsAppSession(w http.ResponseWriter, r *http.Request, session map[string]string, method, action string, body any) {
	if r.Method != method {
		http.NotFound(w, r)
		return
	}
	payload := io.Reader(nil)
	if body != nil {
		encoded, _ := json.Marshal(body)
		payload = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(r.Context(), method, fmt.Sprintf("%s/api/sessions/%s/%s", s.cfg.WAGatewayBaseURL, session["id"], action), payload)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("X-Internal-Token", s.cfg.InternalGatewayToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) disconnectWhatsAppSession(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/sessions/%s/disconnect", s.cfg.WAGatewayBaseURL, sessionID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Internal-Token", s.cfg.InternalGatewayToken)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("wa-gateway returned %d", resp.StatusCode)
	}
	return nil
}

func (s *Server) resolveInboundWhatsAppSession(ctx context.Context, sessionID, sessionKey string) (string, string, error) {
	sessionID = strings.TrimSpace(sessionID)
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionID == "" && sessionKey == "" {
		return "", "", fmt.Errorf("whatsapp session id or session key is required")
	}
	var id, organizationID string
	err := s.db.QueryRow(ctx, `
		SELECT id::text, organization_id::text
		FROM whatsapp_sessions
		WHERE deleted_at IS NULL
		  AND provider = 'meta_cloud'
		  AND ((NULLIF($1, '')::uuid IS NOT NULL AND id = NULLIF($1, '')::uuid)
		   OR (NULLIF($2, '') IS NOT NULL AND session_key = $2))
		LIMIT 1
	`, sessionID, sessionKey).Scan(&id, &organizationID)
	return id, organizationID, err
}
