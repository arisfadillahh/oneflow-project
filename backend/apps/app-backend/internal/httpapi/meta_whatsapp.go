package httpapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *Server) handleMetaCloudConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" && role != "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":         s.cfg.MetaCloudEnabled,
		"testEnabled":     s.cfg.MetaTestEnabled,
		"appId":           s.cfg.MetaAppID,
		"configurationId": s.cfg.MetaConfigurationID,
		"graphVersion":    s.cfg.MetaGraphAPIVersion,
		"provider":        "meta_cloud",
		"onboardingMode":  "coexistence",
	})
}

func (s *Server) handleMetaCloudComplete(w http.ResponseWriter, r *http.Request, session map[string]string) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only an organization admin can connect official WhatsApp"})
		return
	}
	if !s.cfg.MetaCloudEnabled {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "official WhatsApp connection is not enabled"})
		return
	}
	if session["provider"] != "meta_cloud" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "session is not an official WhatsApp session"})
		return
	}
	var req struct {
		Code          string `json:"code"`
		WABAID        string `json:"wabaId"`
		PhoneNumberID string `json:"phoneNumberId"`
		Event         string `json:"event"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	req.WABAID = strings.TrimSpace(req.WABAID)
	req.PhoneNumberID = strings.TrimSpace(req.PhoneNumberID)
	req.Event = strings.TrimSpace(req.Event)
	if req.Code == "" || !validMetaGraphID(req.WABAID) || !validMetaGraphID(req.PhoneNumberID) || req.Event != "FINISH_WHATSAPP_BUSINESS_APP_ONBOARDING" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "a completed WhatsApp Business App onboarding session is required"})
		return
	}
	token, err := s.exchangeMetaCode(r.Context(), req.Code)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Meta token exchange failed"})
		return
	}
	if err := s.verifyMetaPhoneOwnership(r.Context(), req.WABAID, req.PhoneNumberID, token); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Meta could not verify the WhatsApp number for this business account; reconnect without resetting WhatsApp Business"})
		return
	}
	encryptedToken, err := s.encryptMetaCredential(token)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "official WhatsApp credentials are not configured safely"})
		return
	}
	if err := s.subscribeMetaWABA(r.Context(), req.WABAID, token); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Meta webhook subscription failed"})
		return
	}
	details, _ := json.Marshal(map[string]any{
		"provider":       "meta_cloud",
		"onboardingMode": "coexistence",
		"official":       true,
		"needsQr":        false,
		"historySync":    "pending",
	})
	tag, err := s.db.Exec(r.Context(), `
		UPDATE whatsapp_sessions
		SET status = 'connected',
		    provider = 'meta_cloud',
		    onboarding_mode = 'coexistence',
		    meta_waba_id = $3,
		    meta_phone_number_id = $4,
		    meta_business_token_ciphertext = $5,
		    meta_webhook_subscribed_at = NOW(),
		    meta_onboarded_at = NOW(),
		    meta_history_sync_status = 'pending',
		    meta_history_sync_deadline = NOW() + INTERVAL '24 hours',
		    details = $6::jsonb,
		    last_connected_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL AND provider = 'meta_cloud'
	`, session["id"], s.organizationID(r.Context()), req.WABAID, req.PhoneNumberID, encryptedToken, string(details))
	if err != nil || tag.RowsAffected() != 1 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save official WhatsApp connection"})
		return
	}
	_ = s.insertAuditLog(r.Context(), "whatsapp_session.meta_cloud_connect", "whatsapp_session", session["id"], map[string]any{
		"provider":        "meta_cloud",
		"onboarding_mode": "coexistence",
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "connected",
		"provider":          "meta_cloud",
		"onboardingMode":    "coexistence",
		"historySyncStatus": "pending",
	})
}

// handleMetaTestConnect is deliberately disabled by default. It exists only to
// produce App Review evidence with Meta's temporary test number before TP/BSP
// approval; production customers must use Embedded Signup.
func (s *Server) handleMetaTestConnect(w http.ResponseWriter, r *http.Request, session map[string]string) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only an organization admin can connect the Meta test channel"})
		return
	}
	if !s.cfg.MetaCloudEnabled || !s.cfg.MetaTestEnabled || s.cfg.MetaTestAccessToken == "" || !validMetaGraphID(s.cfg.MetaTestWABAID) || !validMetaGraphID(s.cfg.MetaTestPhoneNumberID) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Meta test channel is not enabled"})
		return
	}
	if session["provider"] != "meta_cloud" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "session is not an official WhatsApp session"})
		return
	}
	encryptedToken, err := s.encryptMetaCredential(s.cfg.MetaTestAccessToken)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "official WhatsApp credentials are not configured safely"})
		return
	}
	details, _ := json.Marshal(map[string]any{"provider": "meta_cloud", "onboardingMode": "meta_test", "official": false, "testOnly": true, "needsQr": false})
	tag, err := s.db.Exec(r.Context(), `
		UPDATE whatsapp_sessions
		SET status = 'connected', provider = 'meta_cloud', onboarding_mode = 'cloud_api',
		    meta_waba_id = $3, meta_phone_number_id = $4, meta_business_token_ciphertext = $5,
		    meta_onboarded_at = NOW(), details = $6::jsonb, last_connected_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL AND provider = 'meta_cloud'
	`, session["id"], s.organizationID(r.Context()), s.cfg.MetaTestWABAID, s.cfg.MetaTestPhoneNumberID, encryptedToken, string(details))
	if err != nil || tag.RowsAffected() != 1 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save Meta test connection"})
		return
	}
	_ = s.insertAuditLog(r.Context(), "whatsapp_session.meta_test_connect", "whatsapp_session", session["id"], map[string]any{"test_only": true})
	writeJSON(w, http.StatusOK, map[string]any{"status": "connected", "provider": "meta_cloud", "testOnly": true})
}

func (s *Server) exchangeMetaCode(ctx context.Context, code string) (string, error) {
	values := url.Values{}
	values.Set("client_id", s.cfg.MetaAppID)
	values.Set("client_secret", s.cfg.MetaAppSecret)
	values.Set("code", code)
	endpoint := fmt.Sprintf("%s/%s/oauth/access_token?%s", s.cfg.MetaGraphAPIBaseURL, s.cfg.MetaGraphAPIVersion, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("Meta token exchange returned %d", resp.StatusCode)
	}
	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || strings.TrimSpace(payload.AccessToken) == "" {
		return "", fmt.Errorf("Meta token exchange returned no access token")
	}
	return strings.TrimSpace(payload.AccessToken), nil
}

func (s *Server) verifyMetaPhoneOwnership(ctx context.Context, wabaID, phoneID, token string) error {
	if !validMetaGraphID(wabaID) || !validMetaGraphID(phoneID) {
		return fmt.Errorf("invalid Meta asset ID")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	after := ""
	seen := map[string]bool{}
	for page := 0; page < 20; page++ {
		query := url.Values{"fields": {"id"}, "limit": {"100"}}
		if after != "" {
			query.Set("after", after)
		}
		endpoint := fmt.Sprintf("%s/%s/%s/phone_numbers?%s", s.cfg.MetaGraphAPIBaseURL, s.cfg.MetaGraphAPIVersion, url.PathEscape(wabaID), query.Encode())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return err
		}
		var payload struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			Paging struct {
				Next    string `json:"next"`
				Cursors struct {
					After string `json:"after"`
				} `json:"cursors"`
			} `json:"paging"`
		}
		err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || err != nil {
			return fmt.Errorf("Meta phone ownership lookup failed")
		}
		for _, phone := range payload.Data {
			if phone.ID == phoneID {
				return nil
			}
		}
		if payload.Paging.Next == "" {
			break
		}
		// Rebuild the trusted Graph endpoint; never send a bearer token to paging.next.
		after = payload.Paging.Cursors.After
		if after == "" || seen[after] {
			return fmt.Errorf("Meta phone ownership pagination is invalid")
		}
		seen[after] = true
	}
	return fmt.Errorf("WhatsApp number is not verified for this WABA")
}

func validMetaGraphID(value string) bool {
	if len(value) == 0 || len(value) > 32 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func (s *Server) subscribeMetaWABA(ctx context.Context, wabaID, token string) error {
	endpoint := fmt.Sprintf("%s/%s/%s/subscribed_apps", s.cfg.MetaGraphAPIBaseURL, s.cfg.MetaGraphAPIVersion, url.PathEscape(wabaID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("Meta WABA subscribe returned %d", resp.StatusCode)
	}
	var payload struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil || !payload.Success {
		return fmt.Errorf("Meta WABA subscription was not confirmed")
	}
	return nil
}

func (s *Server) encryptMetaCredential(value string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(s.cfg.MetaCredentialKey)
	if err != nil || len(key) != 32 {
		return "", fmt.Errorf("invalid encryption key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func metaHistoryDeadlineLabel(deadline *time.Time) any {
	if deadline == nil {
		return nil
	}
	return deadline.UTC()
}
