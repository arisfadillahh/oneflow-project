package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var slugUnsafePattern = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req struct {
		Name             string `json:"name"`
		Username         string `json:"username"`
		Password         string `json:"password"`
		Phone            string `json:"phone"`
		OrganizationName string `json:"organizationName"`
		OrganizationSlug string `json:"organizationSlug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	req.Phone = strings.TrimSpace(req.Phone)
	req.OrganizationName = strings.TrimSpace(req.OrganizationName)
	req.OrganizationSlug = normalizeOrganizationSlug(req.OrganizationSlug)
	if req.OrganizationSlug == "" {
		req.OrganizationSlug = normalizeOrganizationSlug(req.OrganizationName)
	}
	failureKey := authFailureKey(r, "auth:register", "")
	if s.authFailureLocked(w, failureKey) {
		return
	}
	if req.Name == "" || req.Username == "" || req.Password == "" || req.OrganizationName == "" || req.OrganizationSlug == "" {
		s.writeAuthFailure(w, failureKey, http.StatusBadRequest, map[string]string{"error": "name, username, password, and organizationName are required"})
		return
	}
	if len(req.Password) < 8 {
		s.writeAuthFailure(w, failureKey, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	var organizationID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO organizations (name, slug)
		VALUES ($1, $2)
		RETURNING id
	`, req.OrganizationName, req.OrganizationSlug).Scan(&organizationID)
	if err != nil {
		s.writeAuthFailure(w, failureKey, http.StatusConflict, map[string]string{"error": "registration failed, please try again"})
		return
	}

	var agentID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO agents (name, username, password_hash, phone, role, current_organization_id)
		VALUES ($1, $2, $3, NULLIF($4, ''), 'super_admin', $5)
		RETURNING id
	`, req.Name, req.Username, string(hash), req.Phone, organizationID).Scan(&agentID)
	if err != nil {
		s.writeAuthFailure(w, failureKey, http.StatusConflict, map[string]string{"error": "registration failed, please try again"})
		return
	}
	if _, err := tx.Exec(r.Context(), `
		UPDATE organizations SET created_by = $2 WHERE id = $1
	`, organizationID, agentID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, err := tx.Exec(r.Context(), `
		INSERT INTO organization_members (organization_id, agent_id, role, status)
		VALUES ($1, $2, 'org_owner', 'active')
	`, organizationID, agentID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := createOrganizationDefaultsTx(r.Context(), tx, organizationID, agentID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	token, err := s.signAgentToken(agentID, "super_admin", organizationID, "org_owner")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to sign token"})
		return
	}
	s.authFails.Clear(failureKey)
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]string{
			"id":               agentID,
			"name":             req.Name,
			"username":         req.Username,
			"role":             "super_admin",
			"organizationId":   organizationID,
			"organizationRole": "org_owner",
			"organizationName": req.OrganizationName,
			"organizationSlug": req.OrganizationSlug,
		},
	})
}

func (s *Server) handleOrganizations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	rows, err := s.db.Query(r.Context(), `
		SELECT o.id::text, o.name, o.slug, om.role, o.status, o.created_at
		FROM organization_members om
		JOIN organizations o ON o.id = om.organization_id
		WHERE om.agent_id = $1 AND om.status = 'active' AND o.status = 'active'
		  AND ($3 = 'owner' OR o.slug <> 'default-company')
		ORDER BY CASE WHEN om.organization_id = NULLIF($2, '')::uuid THEN 0 ELSE 1 END, o.name ASC
	`, s.agentID(r.Context()), s.organizationID(r.Context()), s.role(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name, slug, role, status string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &slug, &role, &status, &createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		items = append(items, map[string]any{
			"id":        id,
			"name":      name,
			"slug":      slug,
			"role":      role,
			"status":    status,
			"active":    id == s.organizationID(r.Context()),
			"createdAt": createdAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "activeOrganizationId": s.organizationID(r.Context())})
}

func (s *Server) handleOrganizationSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if s.role(r.Context()) != "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "organization switching is owner-only"})
		return
	}
	var req struct {
		OrganizationID string `json:"organizationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.OrganizationID = strings.TrimSpace(req.OrganizationID)
	organizationID, organizationRole, err := s.resolveAuthorizedOrganization(r.Context(), s.agentID(r.Context()), req.OrganizationID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "organization access denied"})
		return
	}
	if _, err := s.db.Exec(r.Context(), `UPDATE agents SET current_organization_id = $2, updated_at = NOW() WHERE id = $1`, s.agentID(r.Context()), organizationID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	token, err := s.signAgentToken(s.agentID(r.Context()), s.role(r.Context()), organizationID, organizationRole)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to sign token"})
		return
	}
	_ = s.insertAuditLog(r.Context(), "organization.switch", "organization", organizationID, map[string]any{})
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "organizationId": organizationID, "organizationRole": organizationRole})
}

func (s *Server) handleTeamMembers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := s.db.Query(r.Context(), `
			SELECT a.id::text, a.name, a.username, COALESCE(a.phone, ''), a.role::text, om.role, om.status, om.joined_at
			FROM organization_members om
			JOIN agents a ON a.id = om.agent_id
			WHERE om.organization_id = $1
			ORDER BY CASE om.role WHEN 'org_owner' THEN 0 WHEN 'org_admin' THEN 1 WHEN 'supervisor' THEN 2 ELSE 3 END, a.name ASC
		`, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, name, username, phone, agentRole, orgRole, status string
			var joinedAt time.Time
			if err := rows.Scan(&id, &name, &username, &phone, &agentRole, &orgRole, &status, &joinedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			items = append(items, map[string]any{"id": id, "name": name, "username": username, "phone": phone, "role": agentRole, "organizationRole": orgRole, "status": status, "joinedAt": joinedAt})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleTeamMemberRoutes(w http.ResponseWriter, r *http.Request) {
	if !canManageOrganizationMembers(s.organizationRole(r.Context())) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only organization owner/admin can manage team"})
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/team/members/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var req struct {
			Role     string `json:"role"`
			IsActive *bool  `json:"isActive"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.Role = normalizeOrganizationRole(req.Role)
		if req.Role == "" || req.Role == "org_owner" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid non-owner role is required"})
			return
		}
		status := "active"
		if req.IsActive != nil && !*req.IsActive {
			status = "disabled"
		}
		tag, err := s.db.Exec(r.Context(), `
			UPDATE organization_members
			SET role = $3, status = $4, updated_at = NOW()
			WHERE organization_id = $1 AND agent_id = $2 AND role <> 'org_owner'
		`, s.organizationID(r.Context()), id, req.Role, status)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "member not found"})
			return
		}
		_ = s.insertAuditLog(r.Context(), "team.member_update", "agent", id, map[string]any{"role": req.Role, "status": status})
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case http.MethodDelete:
		tag, err := s.db.Exec(r.Context(), `
			UPDATE organization_members
			SET status = 'disabled', updated_at = NOW()
			WHERE organization_id = $1 AND agent_id = $2 AND role <> 'org_owner'
		`, s.organizationID(r.Context()), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "member not found"})
			return
		}
		_ = s.insertAuditLog(r.Context(), "team.member_remove", "agent", id, map[string]any{})
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleTeamInvites(w http.ResponseWriter, r *http.Request) {
	if !canManageOrganizationMembers(s.organizationRole(r.Context())) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only organization owner/admin can manage invites"})
		return
	}
	_ = s.cleanupExpiredInvites(r.Context())
	switch r.Method {
	case http.MethodGet:
		rows, err := s.db.Query(r.Context(), `
			SELECT id::text, COALESCE(email, ''), COALESCE(phone, ''), role, status, expires_at, created_at
			FROM organization_invites
			WHERE organization_id = $1
			ORDER BY created_at DESC
			LIMIT 100
		`, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, email, phone, role, status string
			var expiresAt, createdAt time.Time
			if err := rows.Scan(&id, &email, &phone, &role, &status, &expiresAt, &createdAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			items = append(items, map[string]any{"id": id, "email": email, "phone": phone, "role": role, "status": status, "expiresAt": expiresAt, "createdAt": createdAt})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var req struct {
			Email string `json:"email"`
			Phone string `json:"phone"`
			Role  string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		req.Phone = strings.TrimSpace(req.Phone)
		req.Role = normalizeOrganizationRole(req.Role)
		if req.Role == "" || req.Role == "org_owner" || (req.Email == "" && req.Phone == "") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email/phone and valid non-owner role are required"})
			return
		}
		if err := s.ensurePlanCapacity(r.Context(), "human_user"); err != nil {
			writeJSON(w, http.StatusPaymentRequired, map[string]string{"error": err.Error()})
			return
		}
		token, tokenHash, err := newInviteToken()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create invite token"})
			return
		}
		var inviteID string
		err = s.db.QueryRow(r.Context(), `
			INSERT INTO organization_invites (organization_id, email, phone, role, token_hash, status, invited_by, expires_at)
			VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4, $5, 'pending', NULLIF($6, '')::uuid, NOW() + INTERVAL '7 days')
			RETURNING id
		`, s.organizationID(r.Context()), req.Email, req.Phone, req.Role, tokenHash, s.agentID(r.Context())).Scan(&inviteID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		_ = s.insertAuditLog(r.Context(), "team.invite_create", "organization_invite", inviteID, map[string]any{"role": req.Role})
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": inviteID, "token": token})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleTeamInviteRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.NotFound(w, r)
		return
	}
	if !canManageOrganizationMembers(s.organizationRole(r.Context())) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only organization owner/admin can manage invites"})
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/team/invites/"), "/")
	tag, err := s.db.Exec(r.Context(), `
		UPDATE organization_invites
		SET status = 'revoked', updated_at = NOW()
		WHERE id = $1 AND organization_id = $2 AND status = 'pending'
	`, id, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "invite not found"})
		return
	}
	_ = s.insertAuditLog(r.Context(), "team.invite_revoke", "organization_invite", id, map[string]any{})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	_ = s.cleanupExpiredInvites(r.Context())
	var req struct {
		Token    string `json:"token"`
		Name     string `json:"name"`
		Username string `json:"username"`
		Password string `json:"password"`
		Phone    string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Token == "" || req.Name == "" || req.Username == "" || len(req.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token, name, username, and password are required"})
		return
	}
	tokenHash := inviteTokenHash(req.Token)
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	var inviteID, organizationID, organizationRole string
	err = tx.QueryRow(r.Context(), `
		SELECT id::text, organization_id::text, role
		FROM organization_invites
		WHERE token_hash = $1 AND status = 'pending' AND expires_at > NOW()
		FOR UPDATE
	`, tokenHash).Scan(&inviteID, &organizationID, &organizationRole)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "invite not found or expired"})
		return
	}
	r = r.WithContext(context.WithValue(r.Context(), contextOrganizationID, organizationID))
	if err := s.ensurePlanCapacity(r.Context(), "human_user"); err != nil {
		writeJSON(w, http.StatusPaymentRequired, map[string]string{"error": err.Error()})
		return
	}

	agentRole := organizationRoleToAgentRole(organizationRole)
	var agentID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO agents (name, username, password_hash, phone, role, current_organization_id)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6)
		RETURNING id
	`, req.Name, req.Username, string(hash), req.Phone, agentRole, organizationID).Scan(&agentID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invite acceptance failed, please try again"})
		return
	}
	if _, err := tx.Exec(r.Context(), `
		INSERT INTO organization_members (organization_id, agent_id, role, status, invited_by)
		SELECT organization_id, $2, role, 'active', invited_by
		FROM organization_invites
		WHERE id = $1
	`, inviteID, agentID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, err := tx.Exec(r.Context(), `
		UPDATE organization_invites
		SET status = 'accepted', accepted_by = $2, accepted_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, inviteID, agentID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	token, err := s.signAgentToken(agentID, agentRole, organizationID, organizationRole)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to sign token"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "token": token})
}

func createOrganizationDefaultsTx(ctx context.Context, tx pgx.Tx, organizationID, agentID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO ai_settings (
		  system_prompt, escalation_prompt, fallback_waiting_message,
		  allow_clarification, max_clarification_count, answer_only_from_knowledge,
		  dont_broaden_topic, forbid_promises, forbid_sensitive_answers,
		  allow_auto_update_contact_name, only_fill_name_if_empty, is_active,
		  updated_by, organization_id
		)
		VALUES (
		  'You are the company AI chatbot assistant. Answer only from approved business knowledge.',
		  'Escalate if confidence is low, sensitive, or outside approved knowledge.',
		  'Tim kami sedang meninjau pertanyaan Anda. Mohon tunggu sebentar ya.',
		  TRUE, 1, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE,
		  NULLIF($2, '')::uuid, $1
		)
	`, organizationID, agentID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO credit_wallet (
		  monthly_credit_limit, monthly_credits_used, monthly_credits_remaining,
		  additional_credits_remaining, last_reset_at, next_reset_at, is_active, organization_id
		)
		VALUES (30, 0, 30, 0, NOW(), NOW() + INTERVAL '30 days', TRUE, $1)
	`, organizationID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO credit_pricing_settings (
		  credit_unit_idr, usd_to_idr_rate, chat_model_name, chat_input_price_per_1m,
		  chat_output_price_per_1m, embedding_model_name, embedding_price_per_1m,
		  is_active, updated_by, organization_id
		)
		VALUES (50, 16000, 'openai/gpt-4o-mini', 0.15, 0.60, 'openai/text-embedding-3-small', 0.02, TRUE, NULLIF($2, '')::uuid, $1)
	`, organizationID, agentID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO credit_packages (
		  name, plan_key, description, credit_amount, price, max_whatsapp_sessions,
		  max_ai_agents, max_human_users, billing_period, is_active, is_popular, organization_id
		)
		VALUES
		  ('Starter', 'starter', 'Untuk bisnis kecil yang mulai pakai CS AI.', 1000, 99000, 1, 2, 3, 'monthly', TRUE, FALSE, $1),
		  ('Growth', 'growth', 'Untuk tim operasional aktif dengan beberapa agent.', 3000, 299000, 2, 5, 10, 'monthly', TRUE, TRUE, $1),
		  ('Business', 'business', 'Untuk multi-brand atau volume chat tinggi.', 7000, 699000, 5, 15, 25, 'monthly', TRUE, FALSE, $1),
		  ('Top Up 500', 'addon_500', 'Add-on 500 kredit ekstra. Tidak mengubah limit WhatsApp, AI agent, atau user.', 500, 25000, 0, 0, 0, 'one_time', TRUE, FALSE, $1),
		  ('Top Up 1.500', 'addon_1500', 'Add-on 1.500 kredit ekstra untuk lonjakan chat sementara.', 1500, 75000, 0, 0, 0, 'one_time', TRUE, FALSE, $1),
		  ('Top Up 5.000', 'addon_5000', 'Add-on 5.000 kredit ekstra untuk campaign atau periode ramai.', 5000, 250000, 0, 0, 0, 'one_time', TRUE, FALSE, $1)
	`, organizationID)
	return err
}

func (s *Server) signAgentToken(agentID, role, organizationID, organizationRole string) (string, error) {
	claims := agentClaims{
		AgentID:          agentID,
		Role:             role,
		OrganizationID:   organizationID,
		OrganizationRole: organizationRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			Subject:   agentID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func normalizeOrganizationSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugUnsafePattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if len(value) > 64 {
		value = strings.Trim(value[:64], "-")
	}
	return value
}

func normalizeOrganizationRole(role string) string {
	switch strings.TrimSpace(role) {
	case "org_admin", "supervisor", "operator", "org_owner":
		return strings.TrimSpace(role)
	case "hr_agent":
		return "operator"
	case "hr_manager":
		return "supervisor"
	default:
		return ""
	}
}

func canManageOrganizationMembers(role string) bool {
	return role == "org_owner" || role == "org_admin"
}

func organizationRoleToAgentRole(role string) string {
	switch role {
	case "org_admin":
		return "admin"
	default:
		return "operator"
	}
}

func newInviteToken() (string, string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(raw[:])
	return token, inviteTokenHash(token), nil
}

func inviteTokenHash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func (s *Server) cleanupExpiredInvites(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		UPDATE organization_invites
		SET status = 'expired', updated_at = NOW()
		WHERE status = 'pending' AND expires_at <= NOW()
	`)
	return err
}

func (s *Server) requireOrganizationSession(ctx context.Context, sessionID string) (map[string]string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, errors.New("session id is required")
	}
	var id, label, status, sessionKey, provider string
	err := s.db.QueryRow(ctx, `
		SELECT id::text, label, status, session_key, COALESCE(provider, 'whatsmeow')
		FROM whatsapp_sessions
		WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
	`, sessionID, s.organizationID(ctx)).Scan(&id, &label, &status, &sessionKey, &provider)
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "label": label, "status": status, "sessionKey": sessionKey, "provider": provider}, nil
}

func normalizeWhatsAppSessionLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "WhatsApp Session"
	}
	if len(label) > 80 {
		label = strings.TrimSpace(label[:80])
	}
	return label
}

func sessionKeyForOrganization(organizationID string) string {
	token, _, err := newInviteToken()
	if err != nil {
		return fmt.Sprintf("%s:%d", organizationID, time.Now().UnixNano())
	}
	return organizationID + ":" + token[:16]
}
