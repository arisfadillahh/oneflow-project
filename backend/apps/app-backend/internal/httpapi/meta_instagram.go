package httpapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

const instagramOAuthTTL = 10 * time.Minute

type instagramOAuthClaims struct {
	OrganizationID string `json:"organizationId"`
	AgentID        string `json:"agentId"`
	AIAgentID      string `json:"aiAgentId"`
	jwt.RegisteredClaims
}

type instagramSessionRecord struct {
	ID                string     `json:"id"`
	AIAgentID         string     `json:"aiAgentId"`
	InstagramUserID   string     `json:"instagramUserId"`
	Username          string     `json:"username"`
	ProfilePictureURL string     `json:"profilePictureUrl,omitempty"`
	TokenExpiresAt    *time.Time `json:"tokenExpiresAt,omitempty"`
	WebhookConnected  bool       `json:"webhookConnected"`
	ConnectedAt       time.Time  `json:"connectedAt"`
}

func canManageInstagram(role string) bool {
	return role == "admin" || role == "super_admin"
}

func (s *Server) handleInstagramSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	rows, err := s.db.Query(r.Context(), `
		SELECT id, ai_agent_id, instagram_user_id, username,
		       COALESCE(profile_picture_url, ''), token_expires_at,
		       webhook_subscribed_at IS NOT NULL, connected_at
		FROM instagram_sessions
		WHERE organization_id = $1 AND deleted_at IS NULL
		ORDER BY connected_at DESC
	`, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := make([]instagramSessionRecord, 0)
	for rows.Next() {
		var item instagramSessionRecord
		if err := rows.Scan(&item.ID, &item.AIAgentID, &item.InstagramUserID, &item.Username, &item.ProfilePictureURL, &item.TokenExpiresAt, &item.WebhookConnected, &item.ConnectedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": s.cfg.InstagramEnabled, "items": items})
}

func (s *Server) handleInstagramSessionRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/instagram/sessions/"), "/")
	parts := strings.Split(path, "/")
	id := strings.TrimSpace(parts[0])
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 2 && parts[1] == "validate" && r.Method == http.MethodPost {
		s.handleInstagramSessionValidate(w, r, id)
		return
	}
	if len(parts) != 1 || r.Method != http.MethodDelete {
		http.NotFound(w, r)
		return
	}
	if !canManageInstagram(s.role(r.Context())) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only an organization admin can disconnect Instagram"})
		return
	}
	result, err := s.db.Exec(r.Context(), `
		UPDATE instagram_sessions SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
	`, id, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if result.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Instagram session not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

func (s *Server) handleInstagramSessionValidate(w http.ResponseWriter, r *http.Request, sessionID string) {
	if !canManageInstagram(s.role(r.Context())) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only an organization admin can validate Instagram"})
		return
	}
	var encryptedToken string
	if err := s.db.QueryRow(r.Context(), `
		SELECT access_token_ciphertext FROM instagram_sessions
		WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
	`, sessionID, s.organizationID(r.Context())).Scan(&encryptedToken); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Instagram session not found"})
		return
	}
	accessToken, err := s.decryptMetaCredential(encryptedToken)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Instagram credential cannot be read"})
		return
	}
	profileURL := s.cfg.InstagramGraphAPIBaseURL + "/me?" + url.Values{
		"fields":       {"id,username,account_type"},
		"access_token": {accessToken},
	}.Encode()
	conversationsURL := s.cfg.InstagramGraphAPIBaseURL + "/me/conversations?" + url.Values{
		"platform":     {"instagram"},
		"fields":       {"id,updated_time"},
		"limit":        {"1"},
		"access_token": {accessToken},
	}.Encode()
	if err := s.checkInstagramGraphEndpoint(r.Context(), profileURL); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Instagram profile validation failed"})
		return
	}
	if err := s.checkInstagramGraphEndpoint(r.Context(), conversationsURL); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Instagram messaging validation failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "connected", "permissionsChecked": []string{"instagram_business_basic", "instagram_business_manage_messages"}})
}

func (s *Server) checkInstagramGraphEndpoint(ctx context.Context, endpoint string) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Instagram Graph API returned %d", resp.StatusCode)
	}
	return nil
}

func (s *Server) handleInstagramOAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if !s.cfg.InstagramEnabled {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Instagram is not configured"})
		return
	}
	if !canManageInstagram(s.role(r.Context())) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only an organization admin can connect Instagram"})
		return
	}
	var req struct {
		AIAgentID string `json:"aiAgentId"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.AIAgentID = strings.TrimSpace(req.AIAgentID)
	var exists bool
	if err := s.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM ai_agents WHERE id = $1 AND organization_id = $2 AND is_active = TRUE)`, req.AIAgentID, s.organizationID(r.Context())).Scan(&exists); err != nil || !exists {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "select an active AI agent"})
		return
	}
	rawNonce := make([]byte, 24)
	if _, err := rand.Read(rawNonce); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cannot start Instagram login"})
		return
	}
	nonce := base64.RawURLEncoding.EncodeToString(rawNonce)
	expiresAt := time.Now().UTC().Add(instagramOAuthTTL)
	claims := instagramOAuthClaims{
		OrganizationID: s.organizationID(r.Context()),
		AgentID:        s.agentID(r.Context()),
		AIAgentID:      req.AIAgentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        nonce,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
	}
	state, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cannot sign Instagram login"})
		return
	}
	stateHash := sha256.Sum256([]byte(nonce))
	_, _ = s.db.Exec(r.Context(), `DELETE FROM instagram_oauth_states WHERE expires_at < NOW()`)
	if _, err := s.db.Exec(r.Context(), `
		INSERT INTO instagram_oauth_states (state_hash, organization_id, agent_id, ai_agent_id, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, hex.EncodeToString(stateHash[:]), claims.OrganizationID, claims.AgentID, claims.AIAgentID, expiresAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cannot start Instagram login"})
		return
	}
	authorizeURL := "https://www.instagram.com/oauth/authorize?" + url.Values{
		"client_id":     {s.cfg.InstagramAppID},
		"redirect_uri":  {s.cfg.InstagramRedirectURI},
		"response_type": {"code"},
		"scope":         {"instagram_business_basic,instagram_business_manage_messages"},
		"state":         {state},
	}.Encode()
	writeJSON(w, http.StatusOK, map[string]string{"authorizationUrl": authorizeURL})
}

func (s *Server) handleInstagramOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || !s.cfg.InstagramEnabled {
		http.NotFound(w, r)
		return
	}
	if providerError := strings.TrimSpace(r.URL.Query().Get("error")); providerError != "" {
		http.Redirect(w, r, s.cfg.AppPublicURL+"/channels?instagram=cancelled", http.StatusFound)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	claims := &instagramOAuthClaims{}
	token, err := jwt.ParseWithClaims(state, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("invalid signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid || code == "" || claims.ID == "" {
		http.Redirect(w, r, s.cfg.AppPublicURL+"/channels?instagram=invalid-state", http.StatusFound)
		return
	}
	stateHash := sha256.Sum256([]byte(claims.ID))
	result, err := s.db.Exec(r.Context(), `
		DELETE FROM instagram_oauth_states
		WHERE state_hash = $1 AND organization_id = $2 AND agent_id = $3 AND ai_agent_id = $4 AND expires_at > NOW()
	`, hex.EncodeToString(stateHash[:]), claims.OrganizationID, claims.AgentID, claims.AIAgentID)
	if err != nil || result.RowsAffected() != 1 {
		http.Redirect(w, r, s.cfg.AppPublicURL+"/channels?instagram=expired-state", http.StatusFound)
		return
	}
	accessToken, expiresAt, err := s.exchangeInstagramCode(r.Context(), code)
	if err != nil {
		http.Redirect(w, r, s.cfg.AppPublicURL+"/channels?instagram=token-error", http.StatusFound)
		return
	}
	profile, err := s.fetchInstagramProfile(r.Context(), accessToken)
	if err != nil {
		http.Redirect(w, r, s.cfg.AppPublicURL+"/channels?instagram=profile-error", http.StatusFound)
		return
	}
	ciphertext, err := s.encryptMetaCredential(accessToken)
	if err != nil {
		http.Redirect(w, r, s.cfg.AppPublicURL+"/channels?instagram=credential-error", http.StatusFound)
		return
	}
	_, err = s.db.Exec(r.Context(), `
		INSERT INTO instagram_sessions (
		  organization_id, ai_agent_id, instagram_user_id, username, profile_picture_url,
		  access_token_ciphertext, token_expires_at, connected_by, connected_at, updated_at
		) VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, NOW(), NOW())
		ON CONFLICT (instagram_user_id) WHERE deleted_at IS NULL DO UPDATE SET
		  organization_id = EXCLUDED.organization_id,
		  ai_agent_id = EXCLUDED.ai_agent_id,
		  username = EXCLUDED.username,
		  profile_picture_url = EXCLUDED.profile_picture_url,
		  access_token_ciphertext = EXCLUDED.access_token_ciphertext,
		  token_expires_at = EXCLUDED.token_expires_at,
		  connected_by = EXCLUDED.connected_by,
		  updated_at = NOW()
	`, claims.OrganizationID, claims.AIAgentID, profile.ID, profile.Username, profile.ProfilePictureURL, ciphertext, expiresAt, claims.AgentID)
	if err != nil {
		http.Redirect(w, r, s.cfg.AppPublicURL+"/channels?instagram=save-error", http.StatusFound)
		return
	}
	http.Redirect(w, r, s.cfg.AppPublicURL+"/channels?instagram=connected", http.StatusFound)
}

func (s *Server) handleInstagramWebhook(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.InstagramEnabled {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		if r.URL.Query().Get("hub.mode") != "subscribe" || !hmac.Equal([]byte(r.URL.Query().Get("hub.verify_token")), []byte(s.cfg.InstagramWebhookVerifyToken)) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "webhook verification failed"})
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(r.URL.Query().Get("hub.challenge")))
		return
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
	if err != nil || !validInstagramSignature(body, r.Header.Get("X-Hub-Signature-256"), s.cfg.InstagramAppSecret) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid webhook signature"})
		return
	}
	var payload instagramWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil || payload.Object != "instagram" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid Instagram webhook"})
		return
	}
	for _, entry := range payload.Entry {
		for _, event := range entry.Messaging {
			if event.Message.IsEcho || strings.TrimSpace(event.Message.ID) == "" || strings.TrimSpace(event.Sender.ID) == "" {
				continue
			}
			if err := s.persistInstagramInbound(r.Context(), entry.ID, event, body); err != nil {
				log.Printf("instagram webhook persist failed: recipient_id=%s message_id=%s error=%v", entry.ID, event.Message.ID, err)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

type instagramWebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID        string                  `json:"id"`
		Messaging []instagramWebhookEvent `json:"messaging"`
	} `json:"entry"`
}

type instagramWebhookEvent struct {
	Sender struct {
		ID string `json:"id"`
	} `json:"sender"`
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Timestamp int64 `json:"timestamp"`
	Message   struct {
		ID     string `json:"mid"`
		Text   string `json:"text"`
		IsEcho bool   `json:"is_echo"`
	} `json:"message"`
}

type instagramProfile struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	Name              string `json:"name"`
	ProfilePictureURL string `json:"profile_picture_url"`
}

func (s *Server) exchangeInstagramCode(ctx context.Context, code string) (string, *time.Time, error) {
	body := url.Values{
		"client_id":     {s.cfg.InstagramAppID},
		"client_secret": {s.cfg.InstagramAppSecret},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {s.cfg.InstagramRedirectURI},
		"code":          {code},
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.instagram.com/oauth/access_token", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	var short struct {
		AccessToken string `json:"access_token"`
	}
	if resp.StatusCode >= 300 || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&short) != nil || short.AccessToken == "" {
		return "", nil, fmt.Errorf("Instagram token exchange failed")
	}
	longURL := s.cfg.InstagramGraphAPIBaseURL + "/access_token?" + url.Values{
		"grant_type":    {"ig_exchange_token"},
		"client_secret": {s.cfg.InstagramAppSecret},
		"access_token":  {short.AccessToken},
	}.Encode()
	longReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, longURL, nil)
	longResp, err := s.httpClient.Do(longReq)
	if err != nil {
		return "", nil, err
	}
	defer longResp.Body.Close()
	var long struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if longResp.StatusCode >= 300 || json.NewDecoder(io.LimitReader(longResp.Body, 1<<20)).Decode(&long) != nil || long.AccessToken == "" {
		return "", nil, fmt.Errorf("Instagram long-lived token exchange failed")
	}
	expiresAt := time.Now().UTC().Add(time.Duration(long.ExpiresIn) * time.Second)
	return long.AccessToken, &expiresAt, nil
}

func (s *Server) fetchInstagramProfile(ctx context.Context, accessToken string) (instagramProfile, error) {
	profileURL := s.cfg.InstagramGraphAPIBaseURL + "/me?" + url.Values{
		"fields":       {"id,username,name,profile_picture_url"},
		"access_token": {accessToken},
	}.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, profileURL, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return instagramProfile{}, err
	}
	defer resp.Body.Close()
	var profile instagramProfile
	if resp.StatusCode >= 300 || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&profile) != nil || profile.ID == "" {
		return instagramProfile{}, fmt.Errorf("Instagram profile lookup failed")
	}
	if profile.Username == "" {
		profile.Username = profile.Name
	}
	return profile, nil
}

func (s *Server) decryptMetaCredential(value string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(s.cfg.MetaCredentialKey)
	if err != nil || len(key) != 32 {
		return "", fmt.Errorf("invalid encryption key")
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid encrypted credential")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func validInstagramSignature(body []byte, header, secret string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) || secret == "" {
		return false
	}
	want, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hmac.Equal(mac.Sum(nil), want)
}

func (s *Server) persistInstagramInbound(ctx context.Context, recipientID string, event instagramWebhookEvent, raw []byte) error {
	var sessionID, organizationID, aiAgentID string
	err := s.db.QueryRow(ctx, `
		SELECT id, organization_id, ai_agent_id
		FROM instagram_sessions
		WHERE instagram_user_id = $1 AND deleted_at IS NULL
	`, recipientID).Scan(&sessionID, &organizationID, &aiAgentID)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var duplicate bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM messages WHERE external_message_id = $1 AND organization_id = $2)`, event.Message.ID, organizationID).Scan(&duplicate); err != nil || duplicate {
		return err
	}
	identity := "ig:" + sessionID + ":" + event.Sender.ID
	profile, _ := s.fetchInstagramUserProfile(ctx, event.Sender.ID, mustInstagramSessionToken(ctx, tx, sessionID, s))
	contactName := strings.TrimSpace(profile.Username)
	if contactName == "" {
		contactName = "Instagram user"
	}
	var contactID string
	err = tx.QueryRow(ctx, `
		INSERT INTO contacts (phone, name, organization_id, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (organization_id, phone) DO UPDATE
		SET name = COALESCE(NULLIF(EXCLUDED.name, ''), contacts.name), updated_at = NOW()
		RETURNING id
	`, identity, contactName, organizationID).Scan(&contactID)
	if err != nil {
		return err
	}
	receivedAt := time.Now().UTC()
	if event.Timestamp > 0 {
		receivedAt = time.UnixMilli(event.Timestamp).UTC()
	}
	var conversationID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM conversations
		WHERE contact_id = $1 AND organization_id = $2 AND instagram_session_id = $3
	`, contactID, organizationID, sessionID).Scan(&conversationID)
	if err != nil {
		if err != pgx.ErrNoRows {
			return err
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO conversations (contact_id, channel, mode, status, last_message_at, organization_id, instagram_session_id, ai_agent_id)
			VALUES ($1, 'instagram', 'ai', 'open', $2, $3, $4, $5)
			RETURNING id
		`, contactID, receivedAt, organizationID, sessionID, aiAgentID).Scan(&conversationID)
		if err != nil {
			return err
		}
	}
	var messageID string
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, external_message_id, sender_type, direction, content_type, text, raw_payload, sent_at, delivered_at, organization_id)
		VALUES ($1, $2, 'candidate', 'inbound', 'text', $3, $4::jsonb, $5, $5, $6)
		RETURNING id
	`, conversationID, event.Message.ID, event.Message.Text, string(raw), receivedAt, organizationID).Scan(&messageID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_id = $2, last_message_at = $3, status = 'open', updated_at = NOW() WHERE id = $1`, conversationID, messageID, receivedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE instagram_sessions SET webhook_subscribed_at = COALESCE(webhook_subscribed_at, NOW()), updated_at = NOW() WHERE id = $1`, sessionID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO instagram_webhook_events (event_key, instagram_session_id, payload_hash) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, event.Message.ID, sessionID, hashBytes(raw)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) fetchInstagramUserProfile(ctx context.Context, userID, accessToken string) (instagramProfile, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(accessToken) == "" {
		return instagramProfile{}, fmt.Errorf("Instagram user profile is unavailable")
	}
	profileURL := s.cfg.InstagramGraphAPIBaseURL + "/" + url.PathEscape(userID) + "?" + url.Values{
		"fields":       {"id,username,name,profile_picture_url"},
		"access_token": {accessToken},
	}.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, profileURL, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return instagramProfile{}, err
	}
	defer resp.Body.Close()
	var profile instagramProfile
	if resp.StatusCode >= 300 || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&profile) != nil || profile.ID == "" {
		return instagramProfile{}, fmt.Errorf("Instagram user profile lookup failed")
	}
	if profile.Username == "" {
		profile.Username = profile.Name
	}
	return profile, nil
}

func instagramRecipientID(identity, sessionID string) (string, bool) {
	prefix := "ig:" + sessionID + ":"
	if !strings.HasPrefix(identity, prefix) {
		return "", false
	}
	recipientID := strings.TrimSpace(strings.TrimPrefix(identity, prefix))
	return recipientID, recipientID != ""
}

func mustInstagramSessionToken(ctx context.Context, tx pgx.Tx, sessionID string, s *Server) string {
	var encrypted string
	if err := tx.QueryRow(ctx, `SELECT access_token_ciphertext FROM instagram_sessions WHERE id = $1`, sessionID).Scan(&encrypted); err != nil {
		return ""
	}
	token, _ := s.decryptMetaCredential(encrypted)
	return token
}

func hashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func (s *Server) sendInstagramText(ctx context.Context, sessionID, recipientID, text string) (waSendResult, error) {
	var encryptedToken, instagramUserID string
	if err := s.db.QueryRow(ctx, `
		SELECT access_token_ciphertext, instagram_user_id
		FROM instagram_sessions
		WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
	`, sessionID, s.organizationID(ctx)).Scan(&encryptedToken, &instagramUserID); err != nil {
		return waSendResult{}, fmt.Errorf("Instagram session is not connected")
	}
	accessToken, err := s.decryptMetaCredential(encryptedToken)
	if err != nil {
		return waSendResult{}, err
	}
	body, _ := json.Marshal(map[string]any{"recipient": map[string]string{"id": recipientID}, "message": map[string]string{"text": text}})
	endpoint := fmt.Sprintf("%s/%s/messages?access_token=%s", s.cfg.InstagramGraphAPIBaseURL, url.PathEscape(instagramUserID), url.QueryEscape(accessToken))
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return waSendResult{}, err
	}
	defer resp.Body.Close()
	var payload struct {
		MessageID string `json:"message_id"`
	}
	if resp.StatusCode >= 300 || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload) != nil {
		return waSendResult{}, fmt.Errorf("Instagram send returned %d", resp.StatusCode)
	}
	return waSendResult{Status: "sent", ExternalMessageID: payload.MessageID}, nil
}
