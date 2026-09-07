package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"oneflow/wa-gateway/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	qrcode "github.com/skip2/go-qrcode"
)

type transport interface {
	Status(ctx context.Context) (map[string]any, error)
	GetQR(ctx context.Context) (map[string]any, error)
	PairPhone(ctx context.Context, phone string) (map[string]any, error)
	Connect(ctx context.Context) (map[string]any, error)
	Disconnect(ctx context.Context) (map[string]any, error)
	SendText(ctx context.Context, phone, text string) (map[string]any, error)
	SendMedia(ctx context.Context, media outboundMediaRequest) (map[string]any, error)
	SetChatPresence(ctx context.Context, phone string, active bool) (map[string]any, error)
}

type unavailableTransport struct {
	err error
}

func (t unavailableTransport) Status(context.Context) (map[string]any, error) {
	return nil, t.err
}

func (t unavailableTransport) GetQR(context.Context) (map[string]any, error) {
	return nil, t.err
}

func (t unavailableTransport) PairPhone(context.Context, string) (map[string]any, error) {
	return nil, t.err
}

func (t unavailableTransport) Connect(context.Context) (map[string]any, error) {
	return nil, t.err
}

func (t unavailableTransport) Disconnect(context.Context) (map[string]any, error) {
	return nil, t.err
}

func (t unavailableTransport) SendText(context.Context, string, string) (map[string]any, error) {
	return nil, t.err
}

func (t unavailableTransport) SendMedia(context.Context, outboundMediaRequest) (map[string]any, error) {
	return nil, t.err
}

func (t unavailableTransport) SetChatPresence(context.Context, string, bool) (map[string]any, error) {
	return nil, t.err
}

type Server struct {
	cfg           config.Config
	db            *pgxpool.Pool
	httpClient    *http.Client
	transport     transport
	metaTransport transport
	limit         *rateLimiter
}

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]rateBucket
}

type rateBucket struct {
	count    int
	resetAt  time.Time
	lastSeen time.Time
}

type contextKey string

const contextWhatsAppSessionID contextKey = "whatsapp_session_id"

const (
	maxTextJSONBodyBytes     = 256 << 10
	maxMediaJSONBodyBytes    = 12 << 20
	maxOutboundMediaBytes    = 8 << 20
	downstreamRequestTimeout = 150 * time.Second
)

func New(cfg config.Config, db *pgxpool.Pool) *Server {
	server := &Server{
		cfg:        cfg,
		db:         db,
		httpClient: &http.Client{Timeout: downstreamRequestTimeout},
		limit:      newRateLimiter(60, time.Minute),
	}
	// Retired transports must not initialize or restore old phone sessions.
	server.transport = unavailableTransport{err: errors.New("unofficial WhatsApp has been retired; use Meta Coexistence")}
	server.metaTransport = newMetaCloudTransport(cfg, db, server.httpClient)
	return server
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/qr", retiredLegacyEndpoint)
	mux.HandleFunc("/api/qr-image", retiredLegacyEndpoint)
	mux.HandleFunc("/api/pair-phone", retiredLegacyEndpoint)
	mux.HandleFunc("/api/connect", retiredLegacyEndpoint)
	mux.HandleFunc("/api/disconnect", s.handleDisconnect)
	mux.HandleFunc("/api/meta/webhook", s.handleMetaWebhook)
	mux.Handle("/api/sessions/", s.internalAuth(http.HandlerFunc(s.handleSessionRoutes)))
	mux.Handle("/api/send-text", s.internalAuth(http.HandlerFunc(s.handleSendText)))
	mux.Handle("/api/send-media", s.internalAuth(http.HandlerFunc(s.handleSendMedia)))
	mux.Handle("/api/chat-presence", s.internalAuth(http.HandlerFunc(s.handleChatPresence)))
	return securityHeaders(withCORS(mux, s.cfg.AllowedOrigins))
}

func (s *Server) handleSessionRoutes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/"), "/")
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" {
		http.NotFound(w, r)
		return
	}
	sessionID := strings.TrimSpace(parts[0])
	if parts[1] == "qr" || parts[1] == "qr-image" || parts[1] == "pair-phone" || parts[1] == "connect" {
		retiredLegacyEndpoint(w, r)
		return
	}
	var exists bool
	if err := s.db.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM whatsapp_sessions WHERE id = $1)`, sessionID).Scan(&exists); err != nil || !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	r = r.WithContext(context.WithValue(r.Context(), contextWhatsAppSessionID, sessionID))
	switch parts[1] {
	case "status":
		s.handleStatus(w, r)
	case "qr":
		s.handleQR(w, r)
	case "qr-image":
		s.handleQRImage(w, r)
	case "pair-phone":
		s.handlePairPhone(w, r)
	case "connect":
		s.handleConnect(w, r)
	case "disconnect":
		s.handleDisconnect(w, r)
	case "send-text":
		s.handleSendText(w, r)
	case "send-media":
		s.handleSendMedia(w, r)
	case "chat-presence":
		s.handleChatPresence(w, r)
	case "meta-templates":
		s.handleMetaTemplates(w, r)
	case "meta-template-send":
		s.handleMetaTemplateSend(w, r)
	default:
		http.NotFound(w, r)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler, allowedOrigins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if corsOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		} else {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "origin is not allowed"})
			return
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Internal-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsOriginAllowed(origin string, allowedOrigins []string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	for _, allowed := range allowedOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, buckets: map[string]rateBucket{}}
}

func (rl *rateLimiter) Allow(key string) bool {
	if rl == nil || rl.limit <= 0 || rl.window <= 0 {
		return true
	}
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for bucketKey, bucket := range rl.buckets {
		if now.Sub(bucket.lastSeen) > 2*rl.window {
			delete(rl.buckets, bucketKey)
		}
	}
	bucket := rl.buckets[key]
	if bucket.resetAt.IsZero() || now.After(bucket.resetAt) {
		rl.buckets[key] = rateBucket{count: 1, resetAt: now.Add(rl.window), lastSeen: now}
		return true
	}
	if bucket.count >= rl.limit {
		bucket.lastSeen = now
		rl.buckets[key] = bucket
		return false
	}
	bucket.count++
	bucket.lastSeen = now
	rl.buckets[key] = bucket
	return true
}

func (s *Server) enforceRateLimit(w http.ResponseWriter, r *http.Request, scope string) bool {
	key := scope + ":" + clientIP(r)
	if sessionID := sessionIDFromContext(r.Context()); sessionID != "" {
		key += ":" + sessionID
	}
	if !s.limit.Allow(key) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		return false
	}
	return true
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func (s *Server) internalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(r.Header.Get("X-Internal-Token"))
		if token == "" || token != s.cfg.InternalGatewayToken {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid internal token"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "down", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "wa-gateway"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.transportForContext(r.Context()).Status(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleQR(w http.ResponseWriter, r *http.Request) {
	if !s.enforceRateLimit(w, r, "qr") {
		return
	}
	status, err := s.transportForContext(r.Context()).GetQR(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleQRImage(w http.ResponseWriter, r *http.Request) {
	if !s.enforceRateLimit(w, r, "qr-image") {
		return
	}
	status, err := s.transportForContext(r.Context()).GetQR(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	qr, _ := status["qr"].(string)
	if strings.TrimSpace(qr) == "" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "qr is not ready"})
		return
	}
	png, err := qrcode.Encode(qr, qrcode.Medium, 320)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if !s.enforceRateLimit(w, r, "connect") {
		return
	}
	status, err := s.transportForContext(r.Context()).Connect(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handlePairPhone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone is required"})
		return
	}
	status, err := s.transportForContext(r.Context()).PairPhone(r.Context(), req.Phone)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	status, err := s.transportForContext(r.Context()).Disconnect(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleSendText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxTextJSONBodyBytes)
	var req struct {
		Phone string `json:"phone"`
		Text  string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	req.Text = strings.TrimSpace(req.Text)
	if req.Phone == "" || req.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone and text are required"})
		return
	}

	result, err := s.transportForContext(r.Context()).SendText(r.Context(), req.Phone, req.Text)
	if err != nil {
		_ = s.recordOutboundResult(r.Context(), false, err.Error())
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	_ = s.recordOutboundResult(r.Context(), true, "")
	writeJSON(w, http.StatusOK, result)
}

type outboundMediaRequest struct {
	Phone       string `json:"phone"`
	Text        string `json:"text"`
	MediaBase64 string `json:"mediaBase64"`
	MediaMime   string `json:"mediaMime"`
	MediaName   string `json:"mediaName"`
	MediaKind   string `json:"mediaKind"`
}

func (s *Server) handleSendMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMediaJSONBodyBytes)
	var req outboundMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	req.Text = strings.TrimSpace(req.Text)
	req.MediaBase64 = strings.TrimSpace(req.MediaBase64)
	req.MediaMime = normalizeOutboundMediaMime(req.MediaMime)
	req.MediaName = strings.TrimSpace(req.MediaName)
	req.MediaKind = normalizeOutboundMediaKind(req.MediaKind, req.MediaMime)
	if req.Phone == "" || req.MediaBase64 == "" || req.MediaMime == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone, mediaBase64, and mediaMime are required"})
		return
	}
	if !allowedOutboundMediaMime(req.MediaMime) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported media type"})
		return
	}
	data, err := decodeBase64Payload(req.MediaBase64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mediaBase64 is invalid"})
		return
	}
	if len(data) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "media payload is empty"})
		return
	}
	if len(data) > maxOutboundMediaBytes {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "media payload is too large"})
		return
	}

	result, err := s.transportForContext(r.Context()).SendMedia(r.Context(), req)
	if err != nil {
		_ = s.recordOutboundResult(r.Context(), false, err.Error())
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	_ = s.recordOutboundResult(r.Context(), true, "")
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleChatPresence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxTextJSONBodyBytes)
	var req struct {
		Phone  string `json:"phone"`
		Active bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone is required"})
		return
	}

	result, err := s.transportForContext(r.Context()).SetChatPresence(r.Context(), req.Phone, req.Active)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type mockTransport struct {
	cfg config.Config
	db  *pgxpool.Pool
}

func (m *mockTransport) Status(ctx context.Context) (map[string]any, error) {
	return readStatusRow(ctx, m.db)
}

func (m *mockTransport) GetQR(ctx context.Context) (map[string]any, error) {
	sessionID := sessionIDFromContext(ctx)
	_ = m.upsertStatus(ctx, "qr_pending", map[string]any{
		"mode":      "mock",
		"mock":      true,
		"needsQr":   true,
		"qr":        "MOCK-QR-ONEFLOW-2026",
		"expiresIn": 60,
		"sessionId": sessionID,
	})
	return map[string]any{
		"status":    "qr_ready",
		"mode":      "mock",
		"mock":      true,
		"qr":        "MOCK-QR-ONEFLOW-2026",
		"expiresIn": 60,
		"hint":      "Scan this mock QR to simulate WhatsApp device login.",
		"sessionId": sessionID,
	}, nil
}

func (m *mockTransport) Connect(ctx context.Context) (map[string]any, error) {
	payload := map[string]any{
		"connectedAt":       time.Now().UTC(),
		"escalationGroupId": m.cfg.EscalationGroupID,
		"mock":              true,
		"mode":              "mock",
		"needsQr":           false,
	}
	if err := m.upsertStatus(ctx, "connected", payload); err != nil {
		return nil, err
	}
	return map[string]any{"status": "connected", "details": payload}, nil
}

func (m *mockTransport) PairPhone(ctx context.Context, phone string) (map[string]any, error) {
	return map[string]any{
		"status":   "pair_code_ready",
		"mode":     "mock",
		"mock":     true,
		"phone":    phone,
		"pairCode": "MOCK-PAIR-1234-5678",
	}, nil
}

func (m *mockTransport) Disconnect(ctx context.Context) (map[string]any, error) {
	payload := map[string]any{
		"disconnectedAt": time.Now().UTC(),
		"mock":           true,
		"mode":           "mock",
		"needsQr":        true,
	}
	if err := m.upsertStatus(ctx, "disconnected", payload); err != nil {
		return nil, err
	}
	return map[string]any{"status": "disconnected", "details": payload}, nil
}

func (m *mockTransport) SendText(ctx context.Context, phone, text string) (map[string]any, error) {
	if status, err := readStatusRow(ctx, m.db); err != nil {
		return nil, err
	} else if status["status"] != "connected" {
		return nil, errors.New("mock transport is disconnected")
	}
	return map[string]any{
		"status":            "sent",
		"mode":              "mock",
		"externalMessageId": fmt.Sprintf("mock-out-%d", time.Now().UnixNano()),
		"phone":             phone,
	}, nil
}

func (m *mockTransport) SendMedia(ctx context.Context, media outboundMediaRequest) (map[string]any, error) {
	if status, err := readStatusRow(ctx, m.db); err != nil {
		return nil, err
	} else if status["status"] != "connected" {
		return nil, errors.New("mock transport is disconnected")
	}
	return map[string]any{
		"status":            "sent",
		"mode":              "mock",
		"externalMessageId": fmt.Sprintf("mock-media-out-%d", time.Now().UnixNano()),
		"phone":             media.Phone,
		"mediaKind":         media.MediaKind,
	}, nil
}

func (m *mockTransport) SetChatPresence(ctx context.Context, phone string, active bool) (map[string]any, error) {
	state := "paused"
	if active {
		state = "composing"
	}
	return map[string]any{
		"status": "ok",
		"mode":   "mock",
		"phone":  phone,
		"state":  state,
	}, nil
}

func normalizeOutboundMediaKind(kind, mimeType string) string {
	kind = strings.TrimSpace(strings.ToLower(kind))
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if kind == "image" || kind == "document" {
		return kind
	}
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	return "document"
}

func normalizeOutboundMediaMime(mimeType string) string {
	mimeType = strings.TrimSpace(strings.ToLower(mimeType))
	if mimeType == "" {
		return "application/octet-stream"
	}
	if mimeType == "jpg" || mimeType == "jpeg" {
		return "image/jpeg"
	}
	if mimeType == "png" {
		return "image/png"
	}
	if mimeType == "pdf" {
		return "application/pdf"
	}
	return mimeType
}

func allowedOutboundMediaMime(mimeType string) bool {
	mimeType = normalizeOutboundMediaMime(mimeType)
	if strings.HasPrefix(mimeType, "image/") {
		return mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/webp" || mimeType == "image/gif"
	}
	switch mimeType {
	case "application/pdf",
		"text/plain",
		"text/csv",
		"application/json",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return true
	default:
		return false
	}
}

func decodeBase64Payload(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if comma := strings.Index(value, ","); comma >= 0 && strings.HasPrefix(strings.ToLower(value[:comma]), "data:") {
		value = value[comma+1:]
	}
	value = strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\r', '\t':
			return -1
		default:
			return r
		}
	}, value)
	return base64.StdEncoding.DecodeString(value)
}

func (m *mockTransport) upsertStatus(ctx context.Context, status string, details map[string]any) error {
	return upsertStatus(ctx, m.db, status, details)
}

func readStatusRow(ctx context.Context, db *pgxpool.Pool) (map[string]any, error) {
	if sessionID := sessionIDFromContext(ctx); sessionID != "" {
		var status string
		var details string
		err := db.QueryRow(ctx, `
			SELECT status, COALESCE(details, '{}'::jsonb)::text
			FROM whatsapp_sessions
			WHERE id = $1 AND deleted_at IS NULL
		`, sessionID).Scan(&status, &details)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"status":  status,
			"details": json.RawMessage(details),
		}, nil
	}
	var status string
	var details string
	err := db.QueryRow(ctx, `
		SELECT status, COALESCE(details, '{}'::jsonb)::text
		FROM system_status
		WHERE service_name = 'wa-gateway'
	`).Scan(&status, &details)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status":  status,
		"details": json.RawMessage(details),
	}, nil
}

func upsertStatus(ctx context.Context, db *pgxpool.Pool, status string, details map[string]any) error {
	encoded, _ := json.Marshal(details)
	if sessionID := sessionIDFromContext(ctx); sessionID != "" {
		_, err := db.Exec(ctx, `
			UPDATE whatsapp_sessions
			SET status = $2,
			    details = $3::jsonb,
			    is_default = CASE
			      WHEN $2 = 'connected' AND NOT EXISTS (
			        SELECT 1 FROM whatsapp_sessions ws2
			        WHERE ws2.organization_id = whatsapp_sessions.organization_id
			          AND ws2.deleted_at IS NULL
			          AND ws2.is_default = TRUE
			          AND ws2.id <> whatsapp_sessions.id
			      ) THEN TRUE
			      ELSE is_default
			    END,
			    last_connected_at = CASE WHEN $2 = 'connected' THEN NOW() ELSE last_connected_at END,
			    last_disconnected_at = CASE WHEN $2 = 'disconnected' THEN NOW() ELSE last_disconnected_at END,
			    updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`, sessionID, normalizeSessionStatus(status), string(encoded))
		if err != nil {
			return err
		}
	}
	_, err := db.Exec(ctx, `
		INSERT INTO system_status (service_name, status, details, updated_at)
		VALUES ('wa-gateway', $1, $2::jsonb, NOW())
		ON CONFLICT (service_name) DO UPDATE SET
		  status = EXCLUDED.status,
		  details = EXCLUDED.details,
		  updated_at = NOW()
	`, status, string(encoded))
	return err
}

func sessionIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextWhatsAppSessionID).(string)
	return strings.TrimSpace(id)
}

func (s *Server) transportForContext(ctx context.Context) transport {
	sessionID := sessionIDFromContext(ctx)
	if sessionID == "" {
		return s.transport
	}
	var provider string
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(provider, 'whatsmeow')
		FROM whatsapp_sessions
		WHERE id = $1 AND deleted_at IS NULL
	`, sessionID).Scan(&provider)
	if err != nil {
		return unavailableTransport{err: fmt.Errorf("session transport unavailable")}
	}
	if provider == "meta_cloud" {
		return s.metaTransport
	}
	return s.transport
}

func normalizeSessionStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "connected", "connecting", "qr_pending", "error", "disconnected":
		return strings.TrimSpace(status)
	case "qr_ready":
		return "qr_pending"
	default:
		return "disconnected"
	}
}

func (s *Server) recordOutboundResult(ctx context.Context, success bool, failureReason string) error {
	current, err := readStatusRow(ctx, s.db)
	if err != nil {
		return err
	}
	details := map[string]any{}
	if raw, ok := current["details"].(json.RawMessage); ok && len(raw) > 0 {
		_ = json.Unmarshal(raw, &details)
	}

	if success {
		details["failedOutboundCount"] = 0
		details["lastSuccessfulOutboundAt"] = time.Now().UTC()
		delete(details, "lastFailedOutboundAt")
		delete(details, "lastFailedOutboundError")
	} else {
		failedCount := 0
		if value, ok := details["failedOutboundCount"].(float64); ok {
			failedCount = int(value)
		}
		details["failedOutboundCount"] = failedCount + 1
		details["lastFailedOutboundAt"] = time.Now().UTC()
		details["lastFailedOutboundError"] = failureReason
	}

	status, _ := current["status"].(string)
	if status == "" {
		status = "disconnected"
	}
	return upsertStatus(ctx, s.db, status, details)
}

func postJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, payload any) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("downstream returned %d", resp.StatusCode)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func ListenAndServe(cfg config.Config, db *pgxpool.Pool) error {
	server := New(cfg, db)
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return httpServer.ListenAndServe()
}
