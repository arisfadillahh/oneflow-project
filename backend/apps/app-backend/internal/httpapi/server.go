package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"oneflow/app-backend/internal/config"
	"oneflow/app-backend/internal/storage"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const (
	contextAgentID          contextKey = "agent_id"
	contextRole             contextKey = "role"
	contextOrganizationID   contextKey = "organization_id"
	contextOrganizationRole contextKey = "organization_role"
	contextWhatsAppSession  contextKey = "whatsapp_session_id"
	contextAIAgentID        contextKey = "ai_agent_id"
)

const (
	operatorRole         = "operator"
	legacyOperatorRole   = "hr_agent"
	supervisorRole       = "supervisor"
	legacySupervisorRole = "hr_manager"
)

const (
	maxMediaJSONBodyBytes          = 12 << 20
	maxManualMediaBytes            = 8 << 20
	maxInboundImageBytes           = 4 << 20
	maxKnowledgeUploadBytes        = 5 << 20
	openRouterModelsRequestTimeout = 3 * time.Second
)

const (
	groupNotificationTriggerHumanHandoff           = "human_handoff"
	groupNotificationTriggerAIAnswerCuration       = "ai_answer_curation"
	groupNotificationTriggerCommerceOrderCreated   = "commerce_order_created"
	groupNotificationTriggerCommerceOrderConfirmed = "commerce_order_confirmed"
	groupNotificationTriggerBookingCreated         = "booking_created"
	groupNotificationTriggerBookingUpdated         = "booking_updated"
	defaultHumanHandoffGroupTemplate               = "[{{business.name}}] Customer butuh bantuan\n\nNama: {{customer.name}}\nNomor: {{customer.phone}}\nAlasan: {{handoff.reason}}\nPesan terakhir: {{conversation.last_message}}\n\nBuka inbox: {{conversation.link}}"
	defaultAIAnswerCurationGroupTemplate           = "[{{business.name}}] Kurasi jawaban AI\n\nStatus: {{curation.status}}\nAgent: {{ai.agent}}\nCustomer test: {{customer.name}}\nPertanyaan: {{curation.question}}\nJawaban AI: {{curation.answer}}\nCatatan: {{curation.notes}}\nAksi berikutnya: {{curation.next_action}}\n\nBuka Playground: {{curation.link}}"
	defaultCommerceOrderCreatedGroupTemplate       = "[{{business.name}}] Pesanan baru\n\nCustomer: {{customer.name}}\nNomor: {{customer.phone}}\nTotal: {{order.total}}\nItem: {{order.items}}\nStatus: {{order.status}}\n\nCek order: {{order.link}}"
	defaultCommerceOrderConfirmedGroupTemplate     = "[{{business.name}}] Pesanan dikonfirmasi\n\nCustomer: {{customer.name}}\nNomor: {{customer.phone}}\nTotal: {{order.total}}\nItem: {{order.items}}\n\nCek order: {{order.link}}"
	defaultBookingCreatedGroupTemplate             = "[{{business.name}}] Booking baru\n\nCustomer: {{customer.name}}\nNomor: {{customer.phone}}\nLayanan: {{booking.service}}\nJadwal: {{booking.time}}\nStatus: {{booking.status}}\n\nCek booking: {{booking.link}}"
	defaultBookingUpdatedGroupTemplate             = "[{{business.name}}] Booking diperbarui\n\nCustomer: {{customer.name}}\nNomor: {{customer.phone}}\nLayanan: {{booking.service}}\nJadwal: {{booking.time}}\nStatus: {{booking.status}}\n\nCek booking: {{booking.link}}"
)

var groupNotificationVariablePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)
var customerPreferencePattern = regexp.MustCompile(`(?i)\b(?:saya|aku|gue|gua|kami)\s+((?:tidak|nggak|ngga|ga|gak|kurang)\s+suka|lebih\s+suka|suka|prefer|favorit(?:nya)?)\s+([^.!?\n]{2,160})`)
var customerInterestPattern = regexp.MustCompile(`(?i)\b(?:saya|aku|gue|gua|kami)\s+(tertarik|minat|lagi\s+cari|sedang\s+cari|cari)\s+([^.!?\n]{2,160})`)
var customerCommunicationPreferencePattern = regexp.MustCompile(`(?i)\b(?:saya|aku|gue|gua|kami)\s+(?:lebih\s+suka|prefer)\s+(?:dihubungi|di-?chat|dikontak|di(?:\s+)?wa|ditelepon)\s+(?:via|lewat|di)?\s*([^.!?\n]{2,100})`)
var customerBudgetPreferencePattern = regexp.MustCompile(`(?i)\b(?:budget|anggaran|bujet)\s*(?:saya|aku|gue|gua|kami)?\s*(?:sekitar|kisaran|maksimal|max)?\s*([^.!?\n]{2,120})`)
var customerMemoryAmountPattern = regexp.MustCompile(`(?i)\brp\s*\d+|\b\d+(?:[.,]\d+)?\s*(?:rb|ribu|k|jt|juta|%|persen)`)
var customerMemoryBusinessClaimPattern = regexp.MustCompile(`(?i)\b(harga|price|promo|diskon|discount|gratis|free|stok|stock|jam\s+buka|opening\s+hours|refund|retur|pengembalian|garansi|warranty|ongkir|shipping|pajak|tax|bayar|pembayaran|payment|transfer|lunas|order|pesanan|dikirim|terkirim|selesai|delivered|paid)\b`)
var customerMemoryAuthorityClaimPattern = regexp.MustCompile(`(?i)\b(owner|admin|bos|manager|manajer|supervisor)\b.{0,40}\b(bilang|setuju|approve|approved|konfirmasi|confirmed|resmi)\b`)
var customerMemoryPromptInstructionPattern = regexp.MustCompile(`(?i)\b(mulai\s+sekarang\s+kamu|ingat\s+terus|selalu\s+jawab|abaikan\s+instruksi|ignore\s+instruction|system\s+prompt|developer\s+message)\b`)
var customerMemoryQuestionCuePattern = regexp.MustCompile(`(?i)\b(apa|siapa|mana|berapa|kapan|gimana|bagaimana|rekomendasi|saran|suggest)\b`)
var customerMemoryTrailingQuestionClausePattern = regexp.MustCompile(`(?i)\s*[,;]\s*(?:ada|apakah|boleh|bisa|mau|rekomendasi|saran|suggest|yang)\b.*$`)

type Server struct {
	cfg        config.Config
	db         *pgxpool.Pool
	upgrader   websocket.Upgrader
	httpClient *http.Client
	storage    *storage.Client

	authLimit       *rateLimiter
	playgroundLimit *rateLimiter
	purchaseLimit   *rateLimiter
	internalLimit   *rateLimiter
	waLimit         *rateLimiter
	authFails       *authFailureTracker
}

type rateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	limit   int
	buckets map[string]rateBucket
}

type rateBucket struct {
	count    int
	resetAt  time.Time
	lastSeen time.Time
}

type authFailureTracker struct {
	mu          sync.Mutex
	maxFailures int
	lockout     time.Duration
	entries     map[string]authFailureEntry
}

type authFailureEntry struct {
	failures    int
	lockedUntil time.Time
	lastSeen    time.Time
}

var dummyPasswordHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

type agentClaims struct {
	AgentID          string `json:"agent_id"`
	Role             string `json:"role"`
	OrganizationID   string `json:"organization_id"`
	OrganizationRole string `json:"organization_role"`
	jwt.RegisteredClaims
}

type aiDecisionResponse struct {
	Decision          string         `json:"decision"`
	AnswerText        string         `json:"answer_text"`
	EscalationReason  string         `json:"escalation_reason"`
	ConfidenceScore   float64        `json:"confidence_score"`
	ModelName         string         `json:"model_name"`
	LatencyMS         int            `json:"latency_ms"`
	CreatedAt         time.Time      `json:"created_at"`
	RetrievalMetadata map[string]any `json:"retrieval_metadata"`
	UsageMetadata     map[string]any `json:"usage_metadata"`
}

type aiChatHistoryMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type aiImagePayload struct {
	Base64   string
	MimeType string
}

type messageMediaPayload struct {
	Base64   string
	MimeType string
	FileName string
	Kind     string
	Size     int
	URL      string
	Key      string
}

type creditPricingSnapshot struct {
	CreditUnitIDR float64
	USDToIDRRate  float64
	InputPrice    float64
	OutputPrice   float64
}

type creditEstimate struct {
	CostUSD     float64
	CostIDR     float64
	CreditsUsed int
}

type creditUsageResult struct {
	creditEstimate
	CreditSource        string
	MonthlyConsumed     int
	AdditionalConsumed  int
	MonthlyRemaining    int
	AdditionalRemaining int
}

func New(cfg config.Config, db *pgxpool.Pool, storageClient *storage.Client) *Server {
	return &Server{
		cfg:     cfg,
		db:      db,
		storage: storageClient,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				for _, allowed := range cfg.WebsocketAllowedOrigins {
					if origin == allowed {
						return true
					}
				}
				return false
			},
		},
		httpClient:      &http.Client{Timeout: 120 * time.Second},
		authLimit:       newRateLimiter(10, 10*time.Minute),
		playgroundLimit: newRateLimiter(12, time.Minute),
		purchaseLimit:   newRateLimiter(20, time.Minute),
		internalLimit:   newRateLimiter(600, time.Minute),
		waLimit:         newRateLimiter(30, time.Minute),
		authFails:       newAuthFailureTracker(5, 15*time.Minute),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/public/billing/packages", s.handlePublicBillingPackages)
	mux.HandleFunc("/api/auth/login", s.rateLimited(s.authLimit, "auth:login", s.handleLogin))
	mux.HandleFunc("/api/auth/register", s.rateLimited(s.authLimit, "auth:register", s.handleRegister))
	mux.HandleFunc("/api/team/invites/accept", s.rateLimited(s.authLimit, "auth:invite_accept", s.handleAcceptInvite))
	mux.HandleFunc("/ws", s.handleWS)
	mux.Handle("/api/internal/wa/inbound", s.internalAuth(s.rateLimitedHandler(s.internalLimit, "internal:wa_inbound", http.HandlerFunc(s.handleInternalWAInbound))))
	mux.Handle("/api/internal/wa/echo", s.internalAuth(s.rateLimitedHandler(s.internalLimit, "internal:wa_echo", http.HandlerFunc(s.handleInternalWAEcho))))
	mux.Handle("/api/internal/wa/group-binding", s.internalAuth(s.rateLimitedHandler(s.internalLimit, "internal:wa_group_binding", http.HandlerFunc(retiredLegacyEndpoint))))
	mux.HandleFunc("/api/payments/midtrans/notification", s.handleMidtransNotification)
	mux.Handle("/api/me", s.auth(http.HandlerFunc(s.handleMe)))
	mux.Handle("/api/organizations", s.auth(http.HandlerFunc(s.handleOrganizations)))
	mux.Handle("/api/organizations/switch", s.auth(http.HandlerFunc(s.handleOrganizationSwitch)))
	mux.Handle("/api/team/members", s.auth(http.HandlerFunc(s.handleTeamMembers)))
	mux.Handle("/api/team/members/", s.auth(http.HandlerFunc(s.handleTeamMemberRoutes)))
	mux.Handle("/api/team/invites", s.auth(http.HandlerFunc(s.handleTeamInvites)))
	mux.Handle("/api/team/invites/", s.auth(http.HandlerFunc(s.handleTeamInviteRoutes)))
	mux.Handle("/api/account/profile", s.auth(http.HandlerFunc(s.handleAccountProfile)))
	mux.Handle("/api/account/password", s.auth(http.HandlerFunc(s.handleAccountPassword)))
	mux.Handle("/api/notifications", s.auth(http.HandlerFunc(s.handleNotifications)))
	mux.Handle("/api/agents", s.auth(http.HandlerFunc(s.handleAgents)))
	mux.Handle("/api/agents/", s.auth(http.HandlerFunc(s.handleAgentRoutes)))
	mux.Handle("/api/ai-agents", s.auth(http.HandlerFunc(s.handleAIAgents)))
	mux.Handle("/api/ai-agents/", s.auth(http.HandlerFunc(s.handleAIAgentRoutes)))
	mux.Handle("/api/inbox", s.auth(http.HandlerFunc(s.handleInbox)))
	mux.Handle("/api/contacts", s.auth(http.HandlerFunc(s.handleContacts)))
	mux.Handle("/api/contacts/", s.auth(http.HandlerFunc(s.handleContactRoutes)))
	mux.Handle("/api/contact-tags", s.auth(http.HandlerFunc(s.handleContactTags)))
	mux.Handle("/api/follow-up-tasks", s.auth(http.HandlerFunc(s.handleFollowUpTasks)))
	mux.Handle("/api/follow-up-tasks/", s.auth(http.HandlerFunc(s.handleFollowUpTaskRoutes)))
	mux.Handle("/api/message-templates", s.auth(http.HandlerFunc(s.handleMessageTemplates)))
	mux.Handle("/api/message-templates/", s.auth(http.HandlerFunc(s.handleMessageTemplateRoutes)))
	mux.Handle("/api/broadcasts", s.auth(http.HandlerFunc(s.handleBroadcasts)))
	mux.Handle("/api/deals", s.auth(http.HandlerFunc(s.handleDeals)))
	mux.Handle("/api/deals/", s.auth(http.HandlerFunc(s.handleDealRoutes)))
	mux.Handle("/api/tickets", s.auth(http.HandlerFunc(s.handleTickets)))
	mux.Handle("/api/tickets/", s.auth(http.HandlerFunc(s.handleTicketRoutes)))
	mux.Handle("/api/business-tools", s.auth(http.HandlerFunc(s.handleBusinessTools)))
	mux.Handle("/api/business-tools/", s.auth(http.HandlerFunc(s.handleBusinessToolRoutes)))
	mux.Handle("/api/commerce/products", s.auth(http.HandlerFunc(s.handleCommerceProducts)))
	mux.Handle("/api/commerce/products/", s.auth(http.HandlerFunc(s.handleCommerceProductRoutes)))
	mux.Handle("/api/commerce/order-pipelines", s.auth(http.HandlerFunc(s.handleCommerceOrderPipelines)))
	mux.Handle("/api/commerce/order-pipelines/", s.auth(http.HandlerFunc(s.handleCommerceOrderPipelineRoutes)))
	mux.Handle("/api/commerce/order-stages", s.auth(http.HandlerFunc(s.handleCommerceOrderStages)))
	mux.Handle("/api/commerce/order-stages/", s.auth(http.HandlerFunc(s.handleCommerceOrderStageRoutes)))
	mux.Handle("/api/commerce/order-settings", s.auth(http.HandlerFunc(s.handleCommerceOrderSettings)))
	mux.Handle("/api/commerce/order-drafts", s.auth(http.HandlerFunc(s.handleCommerceOrderDrafts)))
	mux.Handle("/api/commerce/order-drafts/", s.auth(http.HandlerFunc(s.handleCommerceOrderDraftRoutes)))
	mux.Handle("/api/booking/services", s.auth(http.HandlerFunc(s.handleBookingServices)))
	mux.Handle("/api/booking/services/", s.auth(http.HandlerFunc(s.handleBookingServiceRoutes)))
	mux.Handle("/api/booking/appointments", s.auth(http.HandlerFunc(s.handleBookingAppointments)))
	mux.Handle("/api/booking/appointments/", s.auth(http.HandlerFunc(s.handleBookingAppointmentRoutes)))
	mux.Handle("/api/deal-pipelines", s.auth(http.HandlerFunc(s.handleDealPipelines)))
	mux.Handle("/api/deal-pipelines/", s.auth(http.HandlerFunc(s.handleDealPipelineRoutes)))
	mux.Handle("/api/deal-stages", s.auth(http.HandlerFunc(s.handleDealStages)))
	mux.Handle("/api/deal-stages/", s.auth(http.HandlerFunc(s.handleDealStageRoutes)))
	mux.Handle("/api/deal-activities/", s.auth(http.HandlerFunc(s.handleDealActivityRoutes)))
	mux.Handle("/api/dashboard/summary", s.auth(http.HandlerFunc(s.handleDashboardSummary)))
	mux.Handle("/api/system/health", s.auth(http.HandlerFunc(s.handleSystemHealth)))
	mux.Handle("/api/system/alerts", s.auth(http.HandlerFunc(s.handleSystemAlerts)))
	mux.Handle("/api/system/alerts/", s.auth(http.HandlerFunc(s.handleSystemAlertRoutes)))
	mux.Handle("/api/support/report", s.auth(http.HandlerFunc(s.handleSupportReport)))
	mux.Handle("/api/escalation-group", s.auth(http.HandlerFunc(s.handleEscalationGroup)))
	mux.Handle("/api/escalation-group/bind-code", s.auth(http.HandlerFunc(retiredLegacyEndpoint)))
	mux.Handle("/api/group-notification-rules", s.auth(http.HandlerFunc(s.handleGroupNotificationRules)))
	mux.Handle("/api/group-notification-rules/test", s.auth(http.HandlerFunc(s.handleGroupNotificationRuleTest)))
	mux.Handle("/api/group-notification-rules/curation", s.auth(http.HandlerFunc(s.handleGroupNotificationCuration)))
	mux.Handle("/api/analytics/overview", s.auth(http.HandlerFunc(s.handleAnalyticsOverview)))
	mux.Handle("/api/ai-settings", s.auth(http.HandlerFunc(s.handleAISettings)))
	mux.Handle("/api/ai-models", s.auth(http.HandlerFunc(s.handleAIModels)))
	mux.Handle("/api/ai-runs", s.auth(http.HandlerFunc(s.handleAIRuns)))
	mux.Handle("/api/ai-runs/", s.auth(http.HandlerFunc(s.handleAIRunRoutes)))
	mux.Handle("/api/playground/shortcuts", s.auth(http.HandlerFunc(s.handlePlaygroundShortcuts)))
	mux.Handle("/api/playground/shortcuts/", s.auth(http.HandlerFunc(s.handlePlaygroundShortcutRoutes)))
	mux.Handle("/api/playground/run", s.auth(http.HandlerFunc(s.handlePlaygroundRun)))
	if !strings.EqualFold(s.cfg.AppEnv, "production") {
		mux.Handle("/api/dev/simulate-conversation", s.auth(http.HandlerFunc(s.handleSimulateConversation)))
		mux.Handle("/api/dev/simulate-inbound", s.auth(http.HandlerFunc(s.handleSimulateInbound)))
	}
	mux.Handle("/api/knowledge/faqs", s.auth(http.HandlerFunc(s.handleKnowledgeFAQs)))
	mux.Handle("/api/knowledge/faqs/", s.auth(http.HandlerFunc(s.handleKnowledgeFAQRoutes)))
	mux.Handle("/api/knowledge/documents", s.auth(http.HandlerFunc(s.handleKnowledgeDocuments)))
	mux.Handle("/api/knowledge/documents/", s.auth(http.HandlerFunc(s.handleKnowledgeDocumentRoutes)))
	mux.Handle("/api/knowledge/documents/upload", s.auth(http.HandlerFunc(s.handleKnowledgeDocumentUpload)))
	mux.Handle("/api/knowledge-download", s.auth(http.HandlerFunc(s.handleKnowledgeDocumentDownload)))
	mux.Handle("/api/job-positions", s.auth(http.HandlerFunc(s.handleJobPositions)))
	mux.Handle("/api/job-positions/", s.auth(http.HandlerFunc(s.handleJobPositionRoutes)))
	mux.Handle("/api/billing/wallet", s.auth(http.HandlerFunc(s.handleBillingWallet)))
	mux.Handle("/api/billing/usage-logs", s.auth(http.HandlerFunc(s.handleBillingUsageLogs)))
	mux.Handle("/api/billing/usage-logs/", s.auth(http.HandlerFunc(s.handleBillingUsageLogRoutes)))
	mux.Handle("/api/billing/analytics", s.auth(http.HandlerFunc(s.handleBillingAnalytics)))
	mux.Handle("/api/billing/pricing", s.auth(http.HandlerFunc(s.handleBillingPricing)))
	mux.Handle("/api/billing/adjustments", s.auth(http.HandlerFunc(s.handleBillingAdjustments)))
	mux.Handle("/api/billing/packages", s.auth(http.HandlerFunc(s.handleBillingPackages)))
	mux.Handle("/api/billing/packages/", s.auth(http.HandlerFunc(s.handleBillingPackageRoutes)))
	mux.Handle("/api/billing/purchases", s.auth(s.rateLimitedHandler(s.purchaseLimit, "billing:purchases", http.HandlerFunc(s.handleBillingPurchases))))
	mux.Handle("/api/billing/purchases/", s.auth(s.rateLimitedHandler(s.purchaseLimit, "billing:purchases", http.HandlerFunc(s.handleBillingPurchaseRoutes))))
	mux.Handle("/api/whatsapp/sessions", s.auth(http.HandlerFunc(s.handleWhatsAppSessions)))
	mux.Handle("/api/whatsapp/sessions/", s.auth(http.HandlerFunc(s.handleWhatsAppSessionRoutes)))
	mux.Handle("/api/whatsapp/meta/config", s.auth(http.HandlerFunc(s.handleMetaCloudConfig)))
	mux.Handle("/api/conversations/", s.auth(http.HandlerFunc(s.handleConversationRoutes)))
	return securityHeaders(cors(mux, s.cfg.WebsocketAllowedOrigins))
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

func cors(next http.Handler, allowedOrigins []string) http.Handler {
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
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

func newAuthFailureTracker(maxFailures int, lockout time.Duration) *authFailureTracker {
	return &authFailureTracker{maxFailures: maxFailures, lockout: lockout, entries: map[string]authFailureEntry{}}
}

func (rl *rateLimiter) Allow(key string) bool {
	allowed, _ := rl.AllowAt(key)
	return allowed
}

func (rl *rateLimiter) AllowAt(key string) (bool, time.Time) {
	if rl == nil || rl.limit <= 0 || rl.window <= 0 {
		return true, time.Time{}
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
		resetAt := now.Add(rl.window)
		rl.buckets[key] = rateBucket{count: 1, resetAt: resetAt, lastSeen: now}
		return true, resetAt
	}
	if bucket.count >= rl.limit {
		bucket.lastSeen = now
		rl.buckets[key] = bucket
		return false, bucket.resetAt
	}
	bucket.count++
	bucket.lastSeen = now
	rl.buckets[key] = bucket
	return true, bucket.resetAt
}

func (tracker *authFailureTracker) Locked(key string) (time.Time, bool) {
	if tracker == nil || tracker.maxFailures <= 0 || tracker.lockout <= 0 {
		return time.Time{}, false
	}
	now := time.Now()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	entry, ok := tracker.entries[key]
	if !ok {
		return time.Time{}, false
	}
	if !entry.lockedUntil.IsZero() {
		if now.Before(entry.lockedUntil) {
			entry.lastSeen = now
			tracker.entries[key] = entry
			return entry.lockedUntil, true
		}
		delete(tracker.entries, key)
		return time.Time{}, false
	}
	if now.Sub(entry.lastSeen) > tracker.lockout {
		delete(tracker.entries, key)
	}
	return time.Time{}, false
}

func (tracker *authFailureTracker) RecordFailure(key string) (time.Time, bool) {
	if tracker == nil || tracker.maxFailures <= 0 || tracker.lockout <= 0 {
		return time.Time{}, false
	}
	now := time.Now()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	for entryKey, entry := range tracker.entries {
		if now.Sub(entry.lastSeen) > 2*tracker.lockout {
			delete(tracker.entries, entryKey)
		}
	}
	entry := tracker.entries[key]
	if !entry.lockedUntil.IsZero() && now.Before(entry.lockedUntil) {
		entry.lastSeen = now
		tracker.entries[key] = entry
		return entry.lockedUntil, true
	}
	if now.Sub(entry.lastSeen) > tracker.lockout {
		entry = authFailureEntry{}
	}
	entry.failures++
	entry.lastSeen = now
	if entry.failures >= tracker.maxFailures {
		entry.lockedUntil = now.Add(tracker.lockout)
	}
	tracker.entries[key] = entry
	return entry.lockedUntil, !entry.lockedUntil.IsZero() && now.Before(entry.lockedUntil)
}

func (tracker *authFailureTracker) Clear(key string) {
	if tracker == nil {
		return
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	delete(tracker.entries, key)
}

func (s *Server) rateLimited(rl *rateLimiter, scope string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions {
			if allowed, resetAt := rl.AllowAt(scope + ":" + clientIP(r)); !allowed {
				setRetryAfter(w, resetAt)
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) rateLimitedHandler(rl *rateLimiter, scope string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions {
			if !s.enforceRateLimit(w, r, rl, scope) {
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) enforceRateLimit(w http.ResponseWriter, r *http.Request, rl *rateLimiter, scope string) bool {
	key := scope + ":" + clientIP(r)
	if agentID := s.agentID(r.Context()); agentID != "" {
		key += ":" + agentID
	}
	if organizationID := s.organizationID(r.Context()); organizationID != "" {
		key += ":" + organizationID
	}
	if allowed, resetAt := rl.AllowAt(key); !allowed {
		setRetryAfter(w, resetAt)
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		return false
	}
	return true
}

func authFailureKey(r *http.Request, scope string, identifier string) string {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	if identifier == "" {
		identifier = "-"
	}
	return scope + ":" + clientIP(r) + ":" + identifier
}

func (s *Server) authFailureLocked(w http.ResponseWriter, key string) bool {
	lockedUntil, locked := s.authFails.Locked(key)
	if !locked {
		return false
	}
	setRetryAfter(w, lockedUntil)
	writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many failed attempts, try again later"})
	return true
}

func (s *Server) writeAuthFailure(w http.ResponseWriter, key string, status int, payload any) {
	lockedUntil, locked := s.authFails.RecordFailure(key)
	if locked {
		setRetryAfter(w, lockedUntil)
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many failed attempts, try again later"})
		return
	}
	writeJSON(w, status, payload)
}

func setRetryAfter(w http.ResponseWriter, lockedUntil time.Time) {
	seconds := int(time.Until(lockedUntil).Seconds())
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
}

func clientIP(r *http.Request) string {
	if cfIP := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cfIP != "" {
		return cfIP
	}
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

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		claims := &agentClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			return []byte(s.cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		claims.Role = canonicalAgentRole(claims.Role)

		organizationID, organizationRole, err := s.resolveAuthorizedOrganization(r.Context(), claims.AgentID, claims.OrganizationID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid organization access"})
			return
		}

		ctx := context.WithValue(r.Context(), contextAgentID, claims.AgentID)
		ctx = context.WithValue(ctx, contextRole, claims.Role)
		ctx = context.WithValue(ctx, contextOrganizationID, organizationID)
		ctx = context.WithValue(ctx, contextOrganizationRole, canonicalOrganizationRole(organizationRole))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "down"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	failureKey := authFailureKey(r, "auth:login", req.Username)
	if s.authFailureLocked(w, failureKey) {
		return
	}

	row := s.db.QueryRow(r.Context(), `
		SELECT
		  a.id,
		  a.name,
		  a.username,
		  a.role::text,
		  a.password_hash,
		  org.organization_id::text,
		  org.organization_name,
		  org.organization_slug,
		  org.organization_role
		FROM agents a
		JOIN LATERAL (
		  SELECT
		    om.organization_id,
		    o.name AS organization_name,
		    o.slug AS organization_slug,
		    om.role AS organization_role
		  FROM organization_members om
		  JOIN organizations o ON o.id = om.organization_id
		  WHERE om.agent_id = a.id
		    AND om.status = 'active'
		    AND o.status = 'active'
		  ORDER BY
		    CASE WHEN om.organization_id = a.current_organization_id THEN 0 ELSE 1 END,
		    om.joined_at DESC
		  LIMIT 1
		) org ON TRUE
		WHERE a.username = $1 AND a.is_active = TRUE
	`, req.Username)

	var id, name, username, role, passwordHash, organizationID, organizationName, organizationSlug, organizationRole string
	if err := row.Scan(&id, &name, &username, &role, &passwordHash, &organizationID, &organizationName, &organizationSlug, &organizationRole); err != nil {
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(req.Password))
		s.writeAuthFailure(w, failureKey, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		s.writeAuthFailure(w, failureKey, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	s.authFails.Clear(failureKey)

	_, _ = s.db.Exec(r.Context(), `UPDATE agents SET last_login_at = NOW() WHERE id = $1`, id)

	claims := agentClaims{
		AgentID:          id,
		Role:             role,
		OrganizationID:   organizationID,
		OrganizationRole: organizationRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			Subject:   id,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to sign token"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": signed,
		"user": map[string]string{
			"id":               id,
			"name":             name,
			"username":         username,
			"role":             role,
			"organizationId":   organizationID,
			"organizationRole": organizationRole,
			"organizationName": organizationName,
			"organizationSlug": organizationSlug,
		},
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	row := s.db.QueryRow(r.Context(), `
		SELECT
		  a.id,
		  a.name,
		  a.username,
		  a.role::text,
		  o.id::text,
		  o.name,
		  o.slug,
		  om.role
		FROM agents a
		JOIN organization_members om ON om.agent_id = a.id
		JOIN organizations o ON o.id = om.organization_id
		WHERE a.id = $1
		  AND om.organization_id = $2
		  AND om.status = 'active'
		  AND o.status = 'active'
	`, s.agentID(r.Context()), s.organizationID(r.Context()))

	var id, name, username, role, organizationID, organizationName, organizationSlug, organizationRole string
	if err := row.Scan(&id, &name, &username, &role, &organizationID, &organizationName, &organizationSlug, &organizationRole); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":               id,
		"name":             name,
		"username":         username,
		"role":             role,
		"organizationId":   organizationID,
		"organizationRole": organizationRole,
		"organization": map[string]string{
			"id":   organizationID,
			"name": organizationName,
			"slug": organizationSlug,
		},
	})
}

func (s *Server) handleAccountProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.NotFound(w, r)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if len([]rune(name)) > 120 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is too long"})
		return
	}

	row := s.db.QueryRow(r.Context(), `
		UPDATE agents
		SET name = $2,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, username, role
	`, s.agentID(r.Context()), name)

	var id, updatedName, username, role string
	if err := row.Scan(&id, &updatedName, &username, &role); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	if err := s.insertAuditLog(r.Context(), "account.profile_update", "agent", id, map[string]any{"name": updatedName}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]string{
			"id":       id,
			"name":     updatedName,
			"username": username,
			"role":     role,
		},
	})
}

func (s *Server) handleAccountPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if strings.TrimSpace(req.CurrentPassword) == "" || strings.TrimSpace(req.NewPassword) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "current and new password are required"})
		return
	}
	if len(req.NewPassword) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new password must be at least 8 characters"})
		return
	}

	var passwordHash string
	if err := s.db.QueryRow(r.Context(), `
		SELECT password_hash
		FROM agents
		WHERE id = $1 AND is_active = TRUE
	`, s.agentID(r.Context())).Scan(&passwordHash); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.CurrentPassword)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "current password is incorrect"})
		return
	}
	nextHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, err := s.db.Exec(r.Context(), `
		UPDATE agents
		SET password_hash = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, s.agentID(r.Context()), string(nextHash)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertAuditLog(r.Context(), "account.password_change", "agent", s.agentID(r.Context()), map[string]any{}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	items, err := s.accountNotifications(r.Context(), s.role(r.Context()), s.agentID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "super_admin" && role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only super_admin can manage agents"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := s.db.Query(r.Context(), `
			SELECT agents.id, agents.name, agents.username, COALESCE(agents.phone, ''), agents.role::text, agents.is_active, agents.last_login_at, agents.created_at, agents.updated_at
			FROM agents
			JOIN organization_members om ON om.agent_id = agents.id
			WHERE om.organization_id = $1 AND om.status <> 'disabled'
			ORDER BY
			  CASE agents.role::text
			    WHEN 'owner' THEN 0
			    WHEN 'super_admin' THEN 1
			    WHEN 'admin' THEN 2
			    ELSE 3
			  END,
			  username ASC
		`, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		items := []map[string]any{}
		for rows.Next() {
			var id, name, username, phone, agentRole string
			var isActive bool
			var lastLoginAt *time.Time
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&id, &name, &username, &phone, &agentRole, &isActive, &lastLoginAt, &createdAt, &updatedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			items = append(items, map[string]any{
				"id":          id,
				"name":        name,
				"username":    username,
				"phone":       phone,
				"role":        agentRole,
				"isActive":    isActive,
				"lastLoginAt": lastLoginAt,
				"createdAt":   createdAt,
				"updatedAt":   updatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		if role != "super_admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only super_admin can manage agents"})
			return
		}
		var req struct {
			Name     string `json:"name"`
			Username string `json:"username"`
			Password string `json:"password"`
			Phone    string `json:"phone"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Username = strings.TrimSpace(req.Username)
		req.Password = strings.TrimSpace(req.Password)
		req.Phone = strings.TrimSpace(req.Phone)
		req.Role = canonicalAgentRole(req.Role)
		if req.Name == "" || req.Username == "" || req.Password == "" || !isAssignableAgentRole(req.Role) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, username, password, and valid role are required"})
			return
		}
		if err := s.ensurePlanCapacity(r.Context(), "human_user"); err != nil {
			writeJSON(w, http.StatusPaymentRequired, map[string]string{"error": err.Error()})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
			return
		}
		var id string
		if err := s.db.QueryRow(r.Context(), `
			INSERT INTO agents (name, username, password_hash, phone, role, current_organization_id)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6)
			RETURNING id
		`, req.Name, req.Username, string(hash), req.Phone, req.Role, s.organizationID(r.Context())).Scan(&id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if _, err := s.db.Exec(r.Context(), `
			INSERT INTO organization_members (organization_id, agent_id, role, status, invited_by)
			VALUES ($1, $2, $3, 'active', NULLIF($4, '')::uuid)
			ON CONFLICT (organization_id, agent_id) DO UPDATE SET
			  role = EXCLUDED.role,
			  status = 'active',
			  updated_at = NOW()
		`, s.organizationID(r.Context()), id, agentRoleToOrganizationRole(req.Role), s.agentID(r.Context())); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLog(r.Context(), "agent.create", "agent", id, map[string]any{"username": req.Username, "role": req.Role}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "id": id})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleAgentRoutes(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "super_admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only super_admin can manage agents"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]

	if len(parts) == 1 && r.Method == http.MethodPut {
		var req struct {
			Name     string `json:"name"`
			Phone    string `json:"phone"`
			Role     string `json:"role"`
			IsActive bool   `json:"isActive"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Phone = strings.TrimSpace(req.Phone)
		req.Role = canonicalAgentRole(req.Role)
		if req.Name == "" || !isAssignableAgentRole(req.Role) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and valid role are required"})
			return
		}
		if id == s.agentID(r.Context()) && !req.IsActive {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot deactivate yourself"})
			return
		}
		tag, err := s.db.Exec(r.Context(), `
			UPDATE agents
			SET name = $2,
			    phone = NULLIF($3, ''),
			    role = $4,
			    is_active = $5,
			    updated_at = NOW()
			WHERE id = $1
			  AND EXISTS (
			    SELECT 1 FROM organization_members om
			    WHERE om.agent_id = agents.id AND om.organization_id = $6
			  )
		`, id, req.Name, req.Phone, req.Role, req.IsActive, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent not found"})
			return
		}
		if _, err := s.db.Exec(r.Context(), `
			UPDATE organization_members
			SET role = $3,
			    status = CASE WHEN $4 THEN 'active' ELSE 'disabled' END,
			    updated_at = NOW()
			WHERE organization_id = $1 AND agent_id = $2
		`, s.organizationID(r.Context()), id, agentRoleToOrganizationRole(req.Role), req.IsActive); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLog(r.Context(), "agent.update", "agent", id, map[string]any{"role": req.Role, "is_active": req.IsActive}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 2 && parts[1] == "reset-password" && r.Method == http.MethodPost {
		var req struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.Password = strings.TrimSpace(req.Password)
		if len(req.Password) < 6 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
			return
		}
		tag, err := s.db.Exec(r.Context(), `
			UPDATE agents
			SET password_hash = $2,
			    updated_at = NOW()
			WHERE id = $1
			  AND EXISTS (
			    SELECT 1 FROM organization_members om
			    WHERE om.agent_id = agents.id AND om.organization_id = $3
			  )
		`, id, string(hash), s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent not found"})
			return
		}
		if err := s.insertAuditLog(r.Context(), "agent.reset_password", "agent", id, map[string]any{}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	http.NotFound(w, r)
}

func (s *Server) handleAIAgents(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role == "owner" || role == "operator" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		items, err := s.aiAgentItems(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		if role != "super_admin" && role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		req, ok := decodeAIAgentRequest(w, r)
		if !ok {
			return
		}
		if !s.openRouterAvailableChatModelIDs(r.Context())[strings.ToLower(strings.TrimSpace(req.ModelName))] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "modelName is not available for this workspace"})
			return
		}
		if err := s.ensurePlanCapacity(r.Context(), "ai_agent"); err != nil {
			writeJSON(w, http.StatusPaymentRequired, map[string]string{"error": err.Error()})
			return
		}
		var id string
		err := s.db.QueryRow(r.Context(), `
			INSERT INTO ai_agents (
			  organization_id, name, model_name, system_prompt, escalation_prompt, fallback_waiting_message,
			  allow_clarification, max_clarification_count,
			  answer_only_from_knowledge, dont_broaden_topic, forbid_promises, forbid_sensitive_answers,
			  require_action_confirmation, escalate_low_confidence, guide_next_step, concise_response,
			  allow_auto_update_contact_name, only_fill_name_if_empty,
			  customer_memory_enabled, customer_memory_auto_save_enabled, customer_memory_admin_notes_enabled,
			  customer_memory_ai_extraction_enabled, customer_memory_verifier_enabled, customer_memory_llm_validator_enabled,
			  customer_memory_max_items, customer_memory_max_chars, customer_memory_retention_days,
			  is_active, created_by, updated_by
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,TRUE,NULLIF($28, '')::uuid,NULLIF($28, '')::uuid)
			RETURNING id
		`, s.organizationID(r.Context()), req.Name, req.ModelName, req.SystemPrompt, req.EscalationPrompt, req.FallbackWaitingMessage, req.AllowClarification, req.MaxClarificationCount, req.AnswerOnlyFromKnowledge, req.DontBroadenTopic, req.ForbidPromises, req.ForbidSensitiveAnswers, req.RequireActionConfirmation, req.EscalateLowConfidence, req.GuideNextStep, req.ConciseResponse, req.AllowAutoUpdateContactName, req.OnlyFillNameIfEmpty, req.CustomerMemoryEnabled, req.CustomerMemoryAutoSaveEnabled, req.CustomerMemoryAdminNotesEnabled, req.CustomerMemoryAIExtractionEnabled, req.CustomerMemoryVerifierEnabled, req.CustomerMemoryLLMValidatorEnabled, req.CustomerMemoryMaxItems, req.CustomerMemoryMaxChars, req.CustomerMemoryRetentionDays, s.agentID(r.Context())).Scan(&id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		_ = s.insertAuditLog(r.Context(), "ai_agent.create", "ai_agent", id, map[string]any{"name": req.Name, "model_name": req.ModelName})
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "id": id})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleAIAgentRoutes(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "super_admin" && role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/ai-agents/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		req, ok := decodeAIAgentRequest(w, r)
		if !ok {
			return
		}
		if !s.openRouterAvailableChatModelIDs(r.Context())[strings.ToLower(strings.TrimSpace(req.ModelName))] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "modelName is not available for this workspace"})
			return
		}
		tag, err := s.db.Exec(r.Context(), `
			UPDATE ai_agents
			SET name = $3,
			    model_name = $4,
			    system_prompt = $5,
			    escalation_prompt = $6,
			    fallback_waiting_message = $7,
			    allow_clarification = $8,
			    max_clarification_count = $9,
			    answer_only_from_knowledge = $10,
			    dont_broaden_topic = $11,
			    forbid_promises = $12,
			    forbid_sensitive_answers = $13,
			    require_action_confirmation = $14,
			    escalate_low_confidence = $15,
			    guide_next_step = $16,
			    concise_response = $17,
			    allow_auto_update_contact_name = $18,
			    only_fill_name_if_empty = $19,
			    customer_memory_enabled = $20,
			    customer_memory_auto_save_enabled = $21,
			    customer_memory_admin_notes_enabled = $22,
			    customer_memory_ai_extraction_enabled = $23,
			    customer_memory_verifier_enabled = $24,
			    customer_memory_llm_validator_enabled = $25,
			    customer_memory_max_items = $26,
			    customer_memory_max_chars = $27,
			    customer_memory_retention_days = $28,
			    is_active = $29,
			    updated_by = NULLIF($30, '')::uuid,
			    updated_at = NOW()
			WHERE id = $1 AND organization_id = $2
		`, id, s.organizationID(r.Context()), req.Name, req.ModelName, req.SystemPrompt, req.EscalationPrompt, req.FallbackWaitingMessage, req.AllowClarification, req.MaxClarificationCount, req.AnswerOnlyFromKnowledge, req.DontBroadenTopic, req.ForbidPromises, req.ForbidSensitiveAnswers, req.RequireActionConfirmation, req.EscalateLowConfidence, req.GuideNextStep, req.ConciseResponse, req.AllowAutoUpdateContactName, req.OnlyFillNameIfEmpty, req.CustomerMemoryEnabled, req.CustomerMemoryAutoSaveEnabled, req.CustomerMemoryAdminNotesEnabled, req.CustomerMemoryAIExtractionEnabled, req.CustomerMemoryVerifierEnabled, req.CustomerMemoryLLMValidatorEnabled, req.CustomerMemoryMaxItems, req.CustomerMemoryMaxChars, req.CustomerMemoryRetentionDays, req.IsActive, s.agentID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "ai agent not found"})
			return
		}
		_ = s.insertAuditLog(r.Context(), "ai_agent.update", "ai_agent", id, map[string]any{"name": req.Name, "model_name": req.ModelName, "is_active": req.IsActive})
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case http.MethodDelete:
		var agentName string
		err := s.db.QueryRow(r.Context(), `
			SELECT name
			FROM ai_agents
			WHERE id = NULLIF($1, '')::uuid
			  AND organization_id = NULLIF($2, '')::uuid
		`, id, s.organizationID(r.Context())).Scan(&agentName)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "ai agent not found"})
			return
		}

		rows, err := s.db.Query(r.Context(), `
			SELECT id::text, label, status
			FROM whatsapp_sessions
			WHERE organization_id = NULLIF($1, '')::uuid
			  AND ai_agent_id = NULLIF($2, '')::uuid
			  AND deleted_at IS NULL
			ORDER BY updated_at DESC
		`, s.organizationID(r.Context()), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		linkedSessions := []map[string]string{}
		for rows.Next() {
			var sessionID, label, status string
			if err := rows.Scan(&sessionID, &label, &status); err != nil {
				rows.Close()
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			linkedSessions = append(linkedSessions, map[string]string{"id": sessionID, "label": label, "status": status})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		rows.Close()
		if len(linkedSessions) > 0 {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":    "ai agent is still linked to whatsapp sessions",
				"code":     "AI_AGENT_LINKED_WHATSAPP",
				"sessions": linkedSessions,
			})
			return
		}

		tx, err := s.db.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer tx.Rollback(r.Context())
		if _, err := tx.Exec(r.Context(), `
			UPDATE whatsapp_sessions
			SET ai_agent_id = NULL,
			    updated_at = NOW()
			WHERE organization_id = NULLIF($1, '')::uuid
			  AND ai_agent_id = NULLIF($2, '')::uuid
			  AND deleted_at IS NOT NULL
		`, s.organizationID(r.Context()), id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		tag, err := tx.Exec(r.Context(), `
			DELETE FROM ai_agents
			WHERE id = NULLIF($1, '')::uuid
			  AND organization_id = NULLIF($2, '')::uuid
		`, id, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "ai agent not found"})
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		_ = s.insertAuditLog(r.Context(), "ai_agent.delete", "ai_agent", id, map[string]any{"name": agentName})
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.NotFound(w, r)
	}
}

type aiAgentRequest struct {
	Name                              string
	ModelName                         string
	SystemPrompt                      string
	EscalationPrompt                  string
	FallbackWaitingMessage            string
	AllowClarification                bool
	MaxClarificationCount             int
	AnswerOnlyFromKnowledge           bool
	DontBroadenTopic                  bool
	ForbidPromises                    bool
	ForbidSensitiveAnswers            bool
	RequireActionConfirmation         bool
	EscalateLowConfidence             bool
	GuideNextStep                     bool
	ConciseResponse                   bool
	AllowAutoUpdateContactName        bool
	OnlyFillNameIfEmpty               bool
	CustomerMemoryEnabled             bool
	CustomerMemoryAutoSaveEnabled     bool
	CustomerMemoryAdminNotesEnabled   bool
	CustomerMemoryAIExtractionEnabled bool
	CustomerMemoryVerifierEnabled     bool
	CustomerMemoryLLMValidatorEnabled bool
	CustomerMemoryMaxItems            int
	CustomerMemoryMaxChars            int
	CustomerMemoryRetentionDays       int
	IsActive                          bool
}

func decodeAIAgentRequest(w http.ResponseWriter, r *http.Request) (aiAgentRequest, bool) {
	var raw struct {
		Name                              string `json:"name"`
		ModelName                         string `json:"modelName"`
		SystemPrompt                      string `json:"systemPrompt"`
		EscalationPrompt                  string `json:"escalationPrompt"`
		FallbackWaitingMessage            string `json:"fallbackWaitingMessage"`
		AllowClarification                *bool  `json:"allowClarification"`
		MaxClarificationCount             *int   `json:"maxClarificationCount"`
		AnswerOnlyFromKnowledge           *bool  `json:"answerOnlyFromKnowledge"`
		DontBroadenTopic                  *bool  `json:"dontBroadenTopic"`
		ForbidPromises                    *bool  `json:"forbidPromises"`
		ForbidSensitiveAnswers            *bool  `json:"forbidSensitiveAnswers"`
		RequireActionConfirmation         *bool  `json:"requireActionConfirmation"`
		EscalateLowConfidence             *bool  `json:"escalateLowConfidence"`
		GuideNextStep                     *bool  `json:"guideNextStep"`
		ConciseResponse                   *bool  `json:"conciseResponse"`
		AllowAutoUpdateContactName        *bool  `json:"allowAutoUpdateContactName"`
		OnlyFillNameIfEmpty               *bool  `json:"onlyFillNameIfEmpty"`
		CustomerMemoryEnabled             *bool  `json:"customerMemoryEnabled"`
		CustomerMemoryAutoSaveEnabled     *bool  `json:"customerMemoryAutoSaveEnabled"`
		CustomerMemoryAdminNotesEnabled   *bool  `json:"customerMemoryAdminNotesEnabled"`
		CustomerMemoryAIExtractionEnabled *bool  `json:"customerMemoryAiExtractionEnabled"`
		CustomerMemoryVerifierEnabled     *bool  `json:"customerMemoryVerifierEnabled"`
		CustomerMemoryLLMValidatorEnabled *bool  `json:"customerMemoryLlmValidatorEnabled"`
		CustomerMemoryMaxItems            *int   `json:"customerMemoryMaxItems"`
		CustomerMemoryMaxChars            *int   `json:"customerMemoryMaxChars"`
		CustomerMemoryRetentionDays       *int   `json:"customerMemoryRetentionDays"`
		IsActive                          *bool  `json:"isActive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return aiAgentRequest{}, false
	}
	req := aiAgentRequest{
		Name:                          strings.TrimSpace(raw.Name),
		ModelName:                     strings.ToLower(strings.TrimSpace(raw.ModelName)),
		SystemPrompt:                  strings.TrimSpace(raw.SystemPrompt),
		EscalationPrompt:              strings.TrimSpace(raw.EscalationPrompt),
		FallbackWaitingMessage:        strings.TrimSpace(raw.FallbackWaitingMessage),
		AllowClarification:            true,
		MaxClarificationCount:         1,
		AnswerOnlyFromKnowledge:       true,
		DontBroadenTopic:              true,
		ForbidPromises:                true,
		ForbidSensitiveAnswers:        true,
		RequireActionConfirmation:     true,
		EscalateLowConfidence:         true,
		GuideNextStep:                 true,
		ConciseResponse:               true,
		OnlyFillNameIfEmpty:           true,
		CustomerMemoryVerifierEnabled: true,
		CustomerMemoryMaxItems:        5,
		CustomerMemoryMaxChars:        800,
		CustomerMemoryRetentionDays:   180,
		IsActive:                      true,
	}
	if req.ModelName == "" {
		req.ModelName = "openai/gpt-4o-mini"
	}
	if req.FallbackWaitingMessage == "" {
		req.FallbackWaitingMessage = "Pesan Anda sudah kami teruskan ke tim kami. Mohon tunggu sebentar ya."
	}
	if raw.AllowClarification != nil {
		req.AllowClarification = *raw.AllowClarification
	}
	if raw.MaxClarificationCount != nil {
		req.MaxClarificationCount = *raw.MaxClarificationCount
	}
	if req.MaxClarificationCount < 0 {
		req.MaxClarificationCount = 0
	}
	if req.MaxClarificationCount > 5 {
		req.MaxClarificationCount = 5
	}
	if raw.AnswerOnlyFromKnowledge != nil {
		req.AnswerOnlyFromKnowledge = *raw.AnswerOnlyFromKnowledge
	}
	if raw.DontBroadenTopic != nil {
		req.DontBroadenTopic = *raw.DontBroadenTopic
	}
	if raw.ForbidPromises != nil {
		req.ForbidPromises = *raw.ForbidPromises
	}
	if raw.ForbidSensitiveAnswers != nil {
		req.ForbidSensitiveAnswers = *raw.ForbidSensitiveAnswers
	}
	if raw.RequireActionConfirmation != nil {
		req.RequireActionConfirmation = *raw.RequireActionConfirmation
	}
	if raw.EscalateLowConfidence != nil {
		req.EscalateLowConfidence = *raw.EscalateLowConfidence
	}
	if raw.GuideNextStep != nil {
		req.GuideNextStep = *raw.GuideNextStep
	}
	if raw.ConciseResponse != nil {
		req.ConciseResponse = *raw.ConciseResponse
	}
	if raw.AllowAutoUpdateContactName != nil {
		req.AllowAutoUpdateContactName = *raw.AllowAutoUpdateContactName
	}
	if raw.OnlyFillNameIfEmpty != nil {
		req.OnlyFillNameIfEmpty = *raw.OnlyFillNameIfEmpty
	}
	if raw.CustomerMemoryEnabled != nil {
		req.CustomerMemoryEnabled = *raw.CustomerMemoryEnabled
	}
	if raw.CustomerMemoryAutoSaveEnabled != nil {
		req.CustomerMemoryAutoSaveEnabled = *raw.CustomerMemoryAutoSaveEnabled
	}
	if raw.CustomerMemoryAdminNotesEnabled != nil {
		req.CustomerMemoryAdminNotesEnabled = *raw.CustomerMemoryAdminNotesEnabled
	}
	if raw.CustomerMemoryAIExtractionEnabled != nil {
		req.CustomerMemoryAIExtractionEnabled = *raw.CustomerMemoryAIExtractionEnabled
	}
	if raw.CustomerMemoryVerifierEnabled != nil {
		req.CustomerMemoryVerifierEnabled = *raw.CustomerMemoryVerifierEnabled
	}
	if raw.CustomerMemoryLLMValidatorEnabled != nil {
		req.CustomerMemoryLLMValidatorEnabled = *raw.CustomerMemoryLLMValidatorEnabled
	}
	if raw.CustomerMemoryMaxItems != nil {
		req.CustomerMemoryMaxItems = *raw.CustomerMemoryMaxItems
	}
	if raw.CustomerMemoryMaxChars != nil {
		req.CustomerMemoryMaxChars = *raw.CustomerMemoryMaxChars
	}
	if raw.CustomerMemoryRetentionDays != nil {
		req.CustomerMemoryRetentionDays = *raw.CustomerMemoryRetentionDays
	}
	req.CustomerMemoryMaxItems = clampInt(req.CustomerMemoryMaxItems, 1, 10, 5)
	req.CustomerMemoryMaxChars = clampInt(req.CustomerMemoryMaxChars, 200, 1200, 800)
	req.CustomerMemoryRetentionDays = clampInt(req.CustomerMemoryRetentionDays, 30, 730, 180)
	if raw.IsActive != nil {
		req.IsActive = *raw.IsActive
	}
	if req.Name == "" || !isAllowedOpenRouterChatModel(req.ModelName, "") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and valid modelName are required"})
		return aiAgentRequest{}, false
	}
	return req, true
}

func (s *Server) aiAgentItems(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, model_name, system_prompt, escalation_prompt, fallback_waiting_message,
		       allow_clarification, max_clarification_count,
		       answer_only_from_knowledge, dont_broaden_topic, forbid_promises, forbid_sensitive_answers,
		       require_action_confirmation, escalate_low_confidence, guide_next_step, concise_response,
		       allow_auto_update_contact_name, only_fill_name_if_empty,
		       customer_memory_enabled, customer_memory_auto_save_enabled, customer_memory_admin_notes_enabled,
		       customer_memory_ai_extraction_enabled, customer_memory_verifier_enabled, customer_memory_llm_validator_enabled,
		       customer_memory_max_items, customer_memory_max_chars, customer_memory_retention_days,
		       is_active, created_at, updated_at
		FROM ai_agents
		WHERE organization_id = $1
		ORDER BY is_active DESC, updated_at DESC
	`, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name, modelName, systemPrompt, escalationPrompt, fallbackMessage string
		var allowClarification, answerOnly, dontBroaden, forbidPromises, forbidSensitive, requireConfirmation, escalateLowConfidence, guideNextStep, conciseResponse, allowName, onlyEmpty, customerMemoryEnabled, customerMemoryAutoSave, customerMemoryAdminNotes, customerMemoryAIExtraction, customerMemoryVerifier, customerMemoryLLMValidator, active bool
		var maxClarificationCount, customerMemoryMaxItems, customerMemoryMaxChars, customerMemoryRetentionDays int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &modelName, &systemPrompt, &escalationPrompt, &fallbackMessage, &allowClarification, &maxClarificationCount, &answerOnly, &dontBroaden, &forbidPromises, &forbidSensitive, &requireConfirmation, &escalateLowConfidence, &guideNextStep, &conciseResponse, &allowName, &onlyEmpty, &customerMemoryEnabled, &customerMemoryAutoSave, &customerMemoryAdminNotes, &customerMemoryAIExtraction, &customerMemoryVerifier, &customerMemoryLLMValidator, &customerMemoryMaxItems, &customerMemoryMaxChars, &customerMemoryRetentionDays, &active, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id": id, "name": name, "modelName": modelName, "systemPrompt": systemPrompt,
			"escalationPrompt": escalationPrompt, "fallbackWaitingMessage": fallbackMessage,
			"allowClarification": allowClarification, "maxClarificationCount": maxClarificationCount,
			"answerOnlyFromKnowledge": answerOnly, "dontBroadenTopic": dontBroaden,
			"forbidPromises": forbidPromises, "forbidSensitiveAnswers": forbidSensitive,
			"requireActionConfirmation": requireConfirmation, "escalateLowConfidence": escalateLowConfidence,
			"guideNextStep": guideNextStep, "conciseResponse": conciseResponse,
			"allowAutoUpdateContactName": allowName, "onlyFillNameIfEmpty": onlyEmpty,
			"customerMemoryEnabled":             customerMemoryEnabled,
			"customerMemoryAutoSaveEnabled":     customerMemoryAutoSave,
			"customerMemoryAdminNotesEnabled":   customerMemoryAdminNotes,
			"customerMemoryAiExtractionEnabled": customerMemoryAIExtraction,
			"customerMemoryVerifierEnabled":     customerMemoryVerifier,
			"customerMemoryLlmValidatorEnabled": customerMemoryLLMValidator,
			"customerMemoryMaxItems":            customerMemoryMaxItems,
			"customerMemoryMaxChars":            customerMemoryMaxChars,
			"customerMemoryRetentionDays":       customerMemoryRetentionDays,
			"isActive":                          active, "createdAt": createdAt, "updatedAt": updatedAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) aiAgentBelongsToOrganization(ctx context.Context, id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM ai_agents
		  WHERE id = NULLIF($1, '')::uuid
		    AND organization_id = NULLIF($2, '')::uuid
		    AND is_active = TRUE
		)
	`, id, s.organizationID(ctx)).Scan(&exists)
	return exists, err
}

func (s *Server) aiAgentIDForWhatsAppSession(ctx context.Context, sessionID string) (string, error) {
	var aiAgentID string
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(ai_agent_id::text, '')
		FROM whatsapp_sessions
		WHERE id = NULLIF($1, '')::uuid
		  AND organization_id = NULLIF($2, '')::uuid
		  AND deleted_at IS NULL
	`, strings.TrimSpace(sessionID), s.organizationID(ctx)).Scan(&aiAgentID)
	return strings.TrimSpace(aiAgentID), err
}

func (s *Server) contextWithRequestedWhatsAppSession(w http.ResponseWriter, r *http.Request) context.Context {
	sessionID := strings.TrimSpace(r.URL.Query().Get("sessionId"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(r.URL.Query().Get("whatsappSessionId"))
	}
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sessionId is required"})
		return nil
	}
	if _, err := s.requireOrganizationSession(r.Context(), sessionID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return nil
	}
	return context.WithValue(r.Context(), contextWhatsAppSession, sessionID)
}

func (s *Server) handleInbox(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role == "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "owner cannot access client chat inbox"})
		return
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT
		  c.id,
		  ct.id::text,
		  COALESCE(NULLIF(ct.name, ''), ct.phone),
		  ct.phone,
		  c.mode::text,
		  c.status::text,
		  COALESCE(c.whatsapp_session_id::text, ''),
		  COALESCE(ws.label, ''),
		  COALESCE(ws.provider, 'whatsmeow'),
		  COALESCE(c.ai_agent_id::text, ''),
		  COALESCE(aa.name, ''),
		  COALESCE(a.id::text, ''),
		  COALESCE(a.name, ''),
		  COALESCE(m.text, ''),
		  c.last_message_at,
		  inbound.last_customer_message_at,
		  COALESCE(c.escalation_reason, ''),
		  c.priority,
		  c.sla_due_at,
		  COALESCE(c.internal_note, ''),
		  COALESCE(deal_stats.open_deals, 0),
		  COALESCE(deal_stats.open_deal_items, '[]'::jsonb)::text
		FROM conversations c
		JOIN contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
		LEFT JOIN whatsapp_sessions ws ON ws.id = c.whatsapp_session_id AND ws.organization_id = c.organization_id
		LEFT JOIN ai_agents aa ON aa.id = c.ai_agent_id AND aa.organization_id = c.organization_id
		LEFT JOIN agents a ON a.id = c.assigned_to
		LEFT JOIN messages m ON m.id = c.last_message_id AND m.organization_id = c.organization_id
		LEFT JOIN LATERAL (
		  SELECT
		    COUNT(*) FILTER (WHERE d.status = 'open') AS open_deals,
		    COALESCE(
		      jsonb_agg(
		        jsonb_build_object(
		          'id', d.id::text,
		          'title', d.title,
		          'stageName', st.name,
		          'valueAmount', d.value_amount,
		          'ownerName', COALESCE(owner.name, ''),
		          'updatedAt', d.updated_at
		        )
		        ORDER BY d.updated_at DESC
		      ) FILTER (WHERE d.status = 'open'),
		      '[]'::jsonb
		    ) AS open_deal_items
		  FROM deals d
		  JOIN deal_stages st ON st.id = d.stage_id AND st.organization_id = d.organization_id
		  LEFT JOIN agents owner ON owner.id = d.owner_agent_id
		  WHERE d.organization_id = c.organization_id
		    AND (d.conversation_id = c.id OR d.contact_id = ct.id)
		) deal_stats ON TRUE
		LEFT JOIN LATERAL (
		  SELECT MAX(COALESCE(sent_at, created_at)) AS last_customer_message_at
		  FROM messages
		  WHERE conversation_id = c.id
		    AND organization_id = c.organization_id
		    AND direction = 'inbound'
		) inbound ON TRUE
		WHERE c.channel = 'whatsapp' AND c.organization_id = $1
		ORDER BY c.last_message_at DESC
	`, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id, contactID, contactName, phone, mode, status, whatsAppSessionID, whatsAppSessionLabel, whatsappProvider, aiAgentID, aiAgentName, assignedToID, assignedToName, lastMessageText, escalationReason, priority, internalNote, openDealsJSON string
		var lastMessageAt time.Time
		var slaDueAt, lastCustomerMessageAt *time.Time
		var openDealCount int
		if err := rows.Scan(&id, &contactID, &contactName, &phone, &mode, &status, &whatsAppSessionID, &whatsAppSessionLabel, &whatsappProvider, &aiAgentID, &aiAgentName, &assignedToID, &assignedToName, &lastMessageText, &lastMessageAt, &lastCustomerMessageAt, &escalationReason, &priority, &slaDueAt, &internalNote, &openDealCount, &openDealsJSON); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		item := map[string]any{
			"id":                    id,
			"contactId":             contactID,
			"contactName":           contactName,
			"phone":                 phone,
			"mode":                  mode,
			"status":                status,
			"whatsappSessionId":     whatsAppSessionID,
			"whatsappSession":       whatsAppSessionLabel,
			"whatsappProvider":      whatsappProvider,
			"aiAgentId":             aiAgentID,
			"aiAgentName":           aiAgentName,
			"assignedToId":          assignedToID,
			"assignedToName":        assignedToName,
			"lastMessageText":       lastMessageText,
			"lastMessageAt":         lastMessageAt,
			"lastCustomerMessageAt": lastCustomerMessageAt,
			"whatsappWindow":        newWhatsAppCustomerWindowInfo(lastCustomerMessageAt, time.Now()),
			"escalationReason":      escalationReason,
			"priority":              priority,
			"slaDueAt":              slaDueAt,
			"internalNote":          internalNote,
			"openDealCount":         openDealCount,
			"openDeals":             parseJSONValue(openDealsJSON),
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "super_admin" && role != "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only super_admin and operator can access contacts"})
		return
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT
		  ct.id,
		  ct.phone,
		  COALESCE(ct.name, ''),
		  COALESCE(ct.email, ''),
		  COALESCE(ct.notes, ''),
		  ct.lifecycle_status,
		  COALESCE(ct.owner_agent_id::text, ''),
		  COALESCE(owner.name, ''),
		  COALESCE(ct.custom_fields, '{}'::jsonb)::text,
		  COALESCE(tag_stats.tags, '[]'::jsonb)::text,
		  COALESCE(task_stats.open_tasks, 0),
		  task_stats.next_due_at,
		  ct.created_at,
		  ct.updated_at,
		  COALESCE(c.id::text, ''),
		  COALESCE(c.mode::text, ''),
		  COALESCE(c.status::text, ''),
		  c.last_message_at,
		  inbound.last_customer_message_at,
		  COALESCE(a.name, ''),
		  COALESCE(m.text, ''),
		  COALESCE(stats.total_messages, 0),
		  COALESCE(stats.inbound_messages, 0),
		  COALESCE(stats.outbound_messages, 0)
		FROM contacts ct
		LEFT JOIN agents owner ON owner.id = ct.owner_agent_id
		LEFT JOIN conversations c ON c.contact_id = ct.id AND c.channel = 'whatsapp' AND c.organization_id = ct.organization_id
		LEFT JOIN agents a ON a.id = c.assigned_to
		LEFT JOIN messages m ON m.id = c.last_message_id AND m.organization_id = ct.organization_id
		LEFT JOIN LATERAL (
		  SELECT COALESCE(jsonb_agg(jsonb_build_object('id', t.id::text, 'name', t.name, 'color', t.color) ORDER BY t.name), '[]'::jsonb) AS tags
		  FROM contact_tag_links l
		  JOIN contact_tags t ON t.id = l.tag_id AND t.organization_id = l.organization_id
		  WHERE l.contact_id = ct.id AND l.organization_id = ct.organization_id
		) tag_stats ON TRUE
		LEFT JOIN LATERAL (
		  SELECT COUNT(*) AS open_tasks, MIN(due_at) AS next_due_at
		  FROM follow_up_tasks
		  WHERE contact_id = ct.id AND organization_id = ct.organization_id AND status = 'open'
		) task_stats ON TRUE
		LEFT JOIN LATERAL (
		  SELECT
		    COUNT(*) AS total_messages,
		    COUNT(*) FILTER (WHERE direction = 'inbound') AS inbound_messages,
		    COUNT(*) FILTER (WHERE direction = 'outbound') AS outbound_messages
		  FROM messages
		  WHERE conversation_id = c.id AND organization_id = ct.organization_id
		) stats ON TRUE
		LEFT JOIN LATERAL (
		  SELECT MAX(COALESCE(sent_at, created_at)) AS last_customer_message_at
		  FROM messages
		  WHERE conversation_id = c.id
		    AND organization_id = ct.organization_id
		    AND direction = 'inbound'
		) inbound ON TRUE
		WHERE ct.phone <> '__playground__' AND ct.organization_id = $1
		ORDER BY COALESCE(c.last_message_at, ct.updated_at) DESC
	`, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id, phone, name, email, notes, lifecycleStatus, ownerAgentID, ownerName, customFieldsJSON, tagsJSON, conversationID, mode, status, assignedToName, lastMessageText string
		var createdAt, updatedAt time.Time
		var lastMessageAt, lastCustomerMessageAt, nextTaskDueAt *time.Time
		var totalMessages, inboundMessages, outboundMessages, openTasks int
		if err := rows.Scan(
			&id,
			&phone,
			&name,
			&email,
			&notes,
			&lifecycleStatus,
			&ownerAgentID,
			&ownerName,
			&customFieldsJSON,
			&tagsJSON,
			&openTasks,
			&nextTaskDueAt,
			&createdAt,
			&updatedAt,
			&conversationID,
			&mode,
			&status,
			&lastMessageAt,
			&lastCustomerMessageAt,
			&assignedToName,
			&lastMessageText,
			&totalMessages,
			&inboundMessages,
			&outboundMessages,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		items = append(items, map[string]any{
			"id":                    id,
			"phone":                 phone,
			"name":                  name,
			"email":                 email,
			"notes":                 notes,
			"lifecycleStatus":       lifecycleStatus,
			"ownerAgentId":          ownerAgentID,
			"ownerName":             ownerName,
			"customFields":          parseJSONObject(customFieldsJSON),
			"tags":                  parseJSONValue(tagsJSON),
			"openTasks":             openTasks,
			"nextTaskDueAt":         nextTaskDueAt,
			"createdAt":             createdAt,
			"updatedAt":             updatedAt,
			"conversationId":        conversationID,
			"mode":                  mode,
			"status":                status,
			"assignedToName":        assignedToName,
			"lastMessageText":       lastMessageText,
			"lastMessageAt":         lastMessageAt,
			"lastCustomerMessageAt": lastCustomerMessageAt,
			"whatsappWindow":        newWhatsAppCustomerWindowInfo(lastCustomerMessageAt, time.Now()),
			"totalMessages":         totalMessages,
			"inboundMessages":       inboundMessages,
			"outboundMessages":      outboundMessages,
		})
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())

	var summary struct {
		Open          int    `json:"open"`
		PendingHuman  int    `json:"pendingHuman"`
		Resolved      int    `json:"resolved"`
		WhatsAppState string `json:"whatsAppState"`
	}
	if err := s.db.QueryRow(r.Context(), `
		SELECT
		  COUNT(*) FILTER (WHERE status = 'open'),
		  COUNT(*) FILTER (WHERE status = 'pending_human'),
		  COUNT(*) FILTER (WHERE status = 'resolved')
		FROM conversations
		WHERE channel = 'whatsapp' AND organization_id = $1
	`, s.organizationID(r.Context())).Scan(&summary.Open, &summary.PendingHuman, &summary.Resolved); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	_ = s.db.QueryRow(r.Context(), `SELECT status FROM system_status WHERE service_name = 'wa-gateway'`).Scan(&summary.WhatsAppState)

	resp := map[string]any{"summary": summary}
	resp["crm"] = s.crmDashboardSummary(r.Context())
	if role == "owner" {
		resp["wallet"] = s.ownerWallet(r.Context())
	}
	if alerts, err := s.currentOpenSystemAlerts(r.Context()); err == nil {
		resp["systemAlerts"] = alerts
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) crmDashboardSummary(ctx context.Context) map[string]any {
	var openDeals, pendingConversations, overdueFollowUps, overdueDealTasks int
	var openValue, weightedValue int64
	var topOwnerID, topOwnerName string
	var topOwnerOpenDeals int
	var topOwnerOpenValue int64
	_ = s.db.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE d.status = 'open'),
		  COALESCE(SUM(d.value_amount) FILTER (WHERE d.status = 'open'), 0),
		  COALESCE(SUM((d.value_amount * st.probability) / 100) FILTER (WHERE d.status = 'open'), 0)
		FROM deals d
		JOIN deal_stages st ON st.id = d.stage_id AND st.organization_id = d.organization_id
		WHERE d.organization_id = $1 AND d.status <> 'archived'
	`, s.organizationID(ctx)).Scan(&openDeals, &openValue, &weightedValue)
	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM conversations
		WHERE organization_id = $1 AND channel = 'whatsapp' AND status = 'pending_human'
	`, s.organizationID(ctx)).Scan(&pendingConversations)
	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM follow_up_tasks
		WHERE organization_id = $1 AND status = 'open' AND due_at IS NOT NULL AND due_at < NOW()
	`, s.organizationID(ctx)).Scan(&overdueFollowUps)
	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM deal_activities a
		JOIN deals d ON d.id = a.deal_id AND d.organization_id = a.organization_id
		WHERE a.organization_id = $1
		  AND d.status = 'open'
		  AND a.activity_type = 'task'
	  AND a.completed_at IS NULL
	  AND a.due_at IS NOT NULL
	  AND a.due_at < NOW()
	`, s.organizationID(ctx)).Scan(&overdueDealTasks)
	if err := s.db.QueryRow(ctx, `
		SELECT
		  COALESCE(d.owner_agent_id::text, ''),
		  COALESCE(NULLIF(owner.name, ''), 'Unassigned'),
		  COUNT(*),
		  COALESCE(SUM(d.value_amount), 0)
		FROM deals d
		LEFT JOIN agents owner ON owner.id = d.owner_agent_id
		WHERE d.organization_id = $1 AND d.status = 'open'
		GROUP BY d.owner_agent_id, owner.name
		ORDER BY COUNT(*) DESC, COALESCE(SUM(d.value_amount), 0) DESC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&topOwnerID, &topOwnerName, &topOwnerOpenDeals, &topOwnerOpenValue); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		topOwnerID = ""
		topOwnerName = ""
		topOwnerOpenDeals = 0
		topOwnerOpenValue = 0
	}
	return map[string]any{
		"openDeals":            openDeals,
		"openDealValue":        openValue,
		"weightedDealValue":    weightedValue,
		"pendingConversations": pendingConversations,
		"overdueFollowUps":     overdueFollowUps,
		"overdueDealTasks":     overdueDealTasks,
		"topOwner": map[string]any{
			"id":        topOwnerID,
			"name":      topOwnerName,
			"openDeals": topOwnerOpenDeals,
			"openValue": topOwnerOpenValue,
		},
	}
}

func (s *Server) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT service_name, status, COALESCE(details, '{}'::jsonb)::text, updated_at
		FROM system_status
		ORDER BY service_name ASC
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	services := []map[string]any{}
	for rows.Next() {
		var serviceName, status, details string
		var updatedAt time.Time
		if err := rows.Scan(&serviceName, &status, &details, &updatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		services = append(services, map[string]any{
			"serviceName": serviceName,
			"status":      status,
			"details":     json.RawMessage(details),
			"updatedAt":   updatedAt,
		})
	}

	alerts := []string{}
	for _, svc := range services {
		serviceName, _ := svc["serviceName"].(string)
		serviceStatus, _ := svc["status"].(string)
		updatedAt, _ := svc["updatedAt"].(time.Time)
		if threshold, ok := serviceStaleThreshold(serviceName); ok && !updatedAt.IsZero() && time.Since(updatedAt) >= threshold {
			alerts = append(alerts, fmt.Sprintf("%s heartbeat is stale for %d minutes.", serviceName, int(time.Since(updatedAt).Minutes())))
		}
		if serviceName == "wa-gateway" && serviceStatus != "connected" {
			alerts = append(alerts, "WhatsApp is disconnected and chat operations should remain blocked.")
			if !updatedAt.IsZero() && time.Since(updatedAt) >= 10*time.Minute {
				alerts = append(alerts, fmt.Sprintf("WhatsApp gateway has stayed disconnected for %d minutes.", int(time.Since(updatedAt).Minutes())))
			}
		}
		if raw, ok := svc["details"].(json.RawMessage); ok && len(raw) > 0 {
			var details map[string]any
			if err := json.Unmarshal(raw, &details); err == nil {
				if serviceName == "wa-gateway" {
					if failedCount, ok := jsonNumberToInt(details["failedOutboundCount"]); ok && failedCount >= 3 {
						alerts = append(alerts, fmt.Sprintf("WhatsApp gateway has %d consecutive failed outbound sends.", failedCount))
					}
				}
				if serviceName == "worker" {
					if pendingDocuments, ok := jsonNumberToInt(details["pendingKnowledgeDocuments"]); ok && pendingDocuments > 0 {
						alerts = append(alerts, fmt.Sprintf("Worker backlog: %d published knowledge documents still need text extraction.", pendingDocuments))
					}
					if pendingEmbeddings, ok := jsonNumberToInt(details["pendingEmbeddings"]); ok && pendingEmbeddings > 0 {
						alerts = append(alerts, fmt.Sprintf("Worker backlog: %d knowledge records still need embeddings.", pendingEmbeddings))
					}
					if resetError, _ := details["lastMonthlyResetError"].(string); strings.TrimSpace(resetError) != "" {
						alerts = append(alerts, fmt.Sprintf("Monthly credit reset failure: %s", resetError))
					}
				}
			}
		}
	}

	walletSummary := map[string]int{}
	var monthlyRemaining, additionalRemaining, monthlyUsed int
	if err := s.db.QueryRow(r.Context(), `
		SELECT monthly_credits_remaining, additional_credits_remaining, monthly_credits_used
		FROM credit_wallet
		WHERE is_active = TRUE AND organization_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`, s.organizationID(r.Context())).Scan(&monthlyRemaining, &additionalRemaining, &monthlyUsed); err == nil {
		walletSummary["monthlyRemaining"] = monthlyRemaining
		walletSummary["additionalRemaining"] = additionalRemaining
		walletSummary["monthlyUsed"] = monthlyUsed
		if monthlyRemaining+additionalRemaining <= 100 {
			alerts = append(alerts, fmt.Sprintf("AI credits are critically low: %d credits remaining.", monthlyRemaining+additionalRemaining))
		}
	}

	systemAlerts, err := s.syncSystemAlerts(r.Context(), alerts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"services": services, "alerts": alerts, "systemAlerts": systemAlerts, "wallet": walletSummary})
}

func (s *Server) handleSystemAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	alerts, err := s.currentOpenSystemAlerts(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": alerts})
}

func (s *Server) handleSystemAlertRoutes(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/system/alerts/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "ack" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	tag, err := s.db.Exec(r.Context(), `
		UPDATE system_alerts
		SET acknowledged_at = NOW(),
		    acknowledged_by = NULLIF($2, '')::uuid
		WHERE id = $1 AND status = 'open'
		  AND organization_id = $3
	`, parts[0], s.agentID(r.Context()), s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert not found"})
		return
	}
	if err := s.insertAuditLog(r.Context(), "system_alert.acknowledge", "system_alert", parts[0], map[string]any{}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSupportReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if !s.requireWhatsAppConnected(w, r) {
		return
	}

	var req struct {
		Detail      string `json:"detail"`
		CurrentView string `json:"currentView"`
		CurrentURL  string `json:"currentUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	targetPhone := strings.TrimSpace(s.cfg.SupportReportPhone)
	if targetPhone == "" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "support report phone is not configured"})
		return
	}

	agentName, username := "", ""
	_ = s.db.QueryRow(r.Context(), `
		SELECT COALESCE(name, ''), COALESCE(username, '')
		FROM agents
		WHERE id = $1
	`, s.agentID(r.Context())).Scan(&agentName, &username)

	detail := strings.TrimSpace(req.Detail)
	if detail == "" {
		detail = "User menekan tombol laporin masalah dari dashboard."
	}
	if detailRunes := []rune(detail); len(detailRunes) > 700 {
		detail = string(detailRunes[:700]) + "..."
	}

	currentView := strings.TrimSpace(req.CurrentView)
	if currentView == "" {
		currentView = "-"
	}
	currentURL := strings.TrimSpace(req.CurrentURL)
	if currentURL == "" {
		currentURL = "-"
	}

	message := fmt.Sprintf(
		"*LAPORAN MASALAH DASHBOARD*\n\nPelapor: %s (@%s)\nRole: %s\nHalaman: %s\nURL: %s\nWaktu: %s WIB\n\nDetail:\n%s",
		emptyFallback(agentName, "-"),
		emptyFallback(username, "-"),
		emptyFallback(s.role(r.Context()), "-"),
		currentView,
		currentURL,
		time.Now().In(time.FixedZone("WIB", 7*60*60)).Format("02 Jan 2006 15:04"),
		detail,
	)

	sendResult, err := s.sendWhatsAppText(r.Context(), targetPhone, message)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	_ = s.insertAuditLog(r.Context(), "support.report", "whatsapp", targetPhone, map[string]any{
		"currentView":       currentView,
		"currentUrl":        currentURL,
		"externalMessageId": sendResult.ExternalMessageID,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "sent",
		"externalMessageId": sendResult.ExternalMessageID,
	})
}

func (s *Server) handleEscalationGroup(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" && role != "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		ctx := s.contextWithRequestedWhatsAppSession(w, r)
		if ctx == nil {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"escalationGroup": s.escalationGroupStatus(ctx)})
	case http.MethodDelete:
		if role != "owner" && role != "super_admin" && role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		ctx := s.contextWithRequestedWhatsAppSession(w, r)
		if ctx == nil {
			return
		}
		_, err := s.db.Exec(r.Context(), `
			UPDATE wa_escalation_group_bindings
			SET status = CASE WHEN status = 'bound' THEN 'removed' ELSE 'cancelled' END,
			    updated_at = NOW()
			WHERE status IN ('bound', 'pending') AND organization_id = $1
			  AND whatsapp_session_id = NULLIF($2, '')::uuid
		`, s.organizationID(r.Context()), whatsAppSessionIDFromContext(ctx))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLog(r.Context(), "wa_escalation_group.remove", "wa_escalation_group", "", map[string]any{}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "escalationGroup": s.escalationGroupStatus(ctx)})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleEscalationGroupBindCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	ctx := s.contextWithRequestedWhatsAppSession(w, r)
	if ctx == nil {
		return
	}
	code := generateEscalationBindingCode()
	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())
	if _, err := tx.Exec(r.Context(), `
		UPDATE wa_escalation_group_bindings
		SET status = 'replaced', updated_at = NOW()
		WHERE status = 'pending' AND organization_id = $1
		  AND whatsapp_session_id = NULLIF($2, '')::uuid
	`, s.organizationID(r.Context()), whatsAppSessionIDFromContext(ctx)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, err := tx.Exec(r.Context(), `
		INSERT INTO wa_escalation_group_bindings (binding_code, status, expires_at, created_by, organization_id, whatsapp_session_id)
		VALUES ($1, 'pending', $2, NULLIF($3, '')::uuid, $4, NULLIF($5, '')::uuid)
	`, code, expiresAt, s.agentID(r.Context()), s.organizationID(r.Context()), whatsAppSessionIDFromContext(ctx)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertAuditLogTx(r.Context(), tx, "wa_escalation_group.generate_code", "wa_escalation_group", "", map[string]any{
		"expires_at": expiresAt,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "pending",
		"code":            code,
		"text":            code,
		"expiresAt":       expiresAt,
		"escalationGroup": s.escalationGroupStatus(ctx),
	})
}

func (s *Server) handleGroupNotificationRules(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" && role != "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		items, err := s.groupNotificationRuleItems(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":     items,
			"variables": groupNotificationVariableItems(),
		})
	case http.MethodPut:
		if role != "owner" && role != "super_admin" && role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		var raw struct {
			TriggerKey   string `json:"triggerKey"`
			IsEnabled    *bool  `json:"isEnabled"`
			TemplateText string `json:"templateText"`
		}
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		triggerKey := strings.TrimSpace(raw.TriggerKey)
		if triggerKey == "" {
			triggerKey = groupNotificationTriggerHumanHandoff
		}
		if _, ok := groupNotificationTriggerDefinitionByKey(triggerKey); !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported notification trigger"})
			return
		}
		templateText := strings.TrimSpace(raw.TemplateText)
		if templateText == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateText is required"})
			return
		}
		if err := validateGroupNotificationTemplate(triggerKey, templateText); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		isEnabled := true
		if raw.IsEnabled != nil {
			isEnabled = *raw.IsEnabled
		}
		if _, err := s.db.Exec(r.Context(), `
			INSERT INTO group_notification_rules (
			  organization_id, trigger_key, is_enabled, template_text, created_by, updated_by
			)
			VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid, NULLIF($5, '')::uuid)
			ON CONFLICT (organization_id, trigger_key) DO UPDATE SET
			  is_enabled = EXCLUDED.is_enabled,
			  template_text = EXCLUDED.template_text,
			  updated_by = EXCLUDED.updated_by,
			  updated_at = NOW()
		`, s.organizationID(r.Context()), triggerKey, isEnabled, templateText, s.agentID(r.Context())); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		_ = s.insertAuditLog(r.Context(), "group_notification_rule.update", "group_notification_rule", triggerKey, map[string]any{"trigger_key": triggerKey, "is_enabled": isEnabled})
		rule, err := s.groupNotificationRuleItem(r.Context(), triggerKey)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "rule": rule})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleGroupNotificationRuleTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	ctx := s.contextWithRequestedWhatsAppSession(w, r)
	if ctx == nil {
		return
	}
	groupID := strings.TrimSpace(s.activeNotificationGroupID(ctx))
	if groupID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "notification group is not connected"})
		return
	}
	var req struct {
		TriggerKey string `json:"triggerKey"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	triggerKey := strings.TrimSpace(req.TriggerKey)
	if triggerKey == "" {
		triggerKey = groupNotificationTriggerHumanHandoff
	}
	text, enabled, err := s.formatGroupNotificationTest(ctx, triggerKey)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "unsupported notification trigger") {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	if !enabled {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "notification rule is disabled"})
		return
	}
	if err := s.notifyNotificationGroupByID(ctx, groupID, text); err != nil {
		_ = s.recordNotificationGroupDeliveryFailure(context.Background(), "group_notification_test", groupID, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "sent", "preview": text})
}

func (s *Server) handleGroupNotificationCuration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" && role != "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	ctx := r.Context()
	if sessionID := strings.TrimSpace(r.URL.Query().Get("sessionId")); sessionID != "" {
		if _, err := s.requireOrganizationSession(ctx, sessionID); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			return
		}
		ctx = context.WithValue(ctx, contextWhatsAppSession, sessionID)
	}
	groupID := strings.TrimSpace(s.activeNotificationGroupID(ctx))
	if groupID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "notification group is not connected"})
		return
	}

	var req aiAnswerCurationNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	text, enabled, err := s.formatAIAnswerCurationGroupNotification(ctx, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !enabled {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "notification rule is disabled"})
		return
	}
	if err := s.notifyNotificationGroupByID(ctx, groupID, text); err != nil {
		_ = s.recordNotificationGroupDeliveryFailure(context.Background(), "group_notification_curation", groupID, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	_ = s.insertAuditLog(ctx, "group_notification_rule.curation_send", "group_notification_rule", groupNotificationTriggerAIAnswerCuration, map[string]any{
		"status": strings.TrimSpace(req.StatusLabel),
		"tone":   strings.TrimSpace(req.Tone),
	})
	writeJSON(w, http.StatusOK, map[string]any{"status": "sent", "preview": text})
}

func (s *Server) handleInternalWAGroupBinding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req struct {
		WhatsAppSessionID  string    `json:"whatsappSessionId"`
		WhatsAppSessionKey string    `json:"sessionKey"`
		GroupID            string    `json:"groupId"`
		GroupName          string    `json:"groupName"`
		SenderJID          string    `json:"senderJid"`
		Text               string    `json:"text"`
		ExternalMessageID  string    `json:"externalMessageId"`
		ReceivedAt         time.Time `json:"receivedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.GroupID = strings.TrimSpace(req.GroupID)
	req.WhatsAppSessionID = strings.TrimSpace(req.WhatsAppSessionID)
	req.WhatsAppSessionKey = strings.TrimSpace(req.WhatsAppSessionKey)
	req.GroupName = strings.TrimSpace(req.GroupName)
	req.SenderJID = strings.TrimSpace(req.SenderJID)
	req.Text = strings.TrimSpace(req.Text)
	req.ExternalMessageID = strings.TrimSpace(req.ExternalMessageID)
	if req.GroupID == "" || !strings.HasSuffix(strings.ToLower(req.GroupID), "@g.us") || req.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "groupId and text are required"})
		return
	}
	whatsAppSessionID, organizationID, err := s.resolveInboundWhatsAppSession(r.Context(), req.WhatsAppSessionID, req.WhatsAppSessionKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid whatsapp session is required"})
		return
	}
	r = r.WithContext(context.WithValue(r.Context(), contextOrganizationID, organizationID))
	r = r.WithContext(context.WithValue(r.Context(), contextWhatsAppSession, whatsAppSessionID))

	_, _ = s.db.Exec(r.Context(), `
		UPDATE wa_escalation_group_bindings
		SET status = 'expired', updated_at = NOW()
		WHERE status = 'pending' AND expires_at <= NOW()
		  AND (organization_id IS NULL OR organization_id = NULLIF($1, '')::uuid)
		  AND (whatsapp_session_id IS NULL OR whatsapp_session_id = NULLIF($2, '')::uuid)
	`, s.organizationID(r.Context()), whatsAppSessionID)
	rows, err := s.db.Query(r.Context(), `
		SELECT id, binding_code
		FROM wa_escalation_group_bindings
		WHERE status = 'pending' AND expires_at > NOW()
		  AND (organization_id IS NULL OR organization_id = NULLIF($1, '')::uuid)
		  AND (whatsapp_session_id IS NULL OR whatsapp_session_id = NULLIF($2, '')::uuid)
		ORDER BY created_at DESC
		LIMIT 10
	`, s.organizationID(r.Context()), whatsAppSessionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	var matchedID, matchedCode string
	for rows.Next() {
		var id, code string
		if err := rows.Scan(&id, &code); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if escalationBindingCodeMatches(req.Text, code) {
			matchedID = id
			matchedCode = code
			break
		}
	}
	if matchedID == "" {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ignored", "reason": "no_matching_code"})
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())
	if _, err := tx.Exec(r.Context(), `
		UPDATE wa_escalation_group_bindings
		SET status = 'replaced', updated_at = NOW()
		WHERE status = 'bound' AND organization_id = (
		  SELECT organization_id FROM wa_escalation_group_bindings WHERE id = $1
		)
		AND whatsapp_session_id = (
		  SELECT whatsapp_session_id FROM wa_escalation_group_bindings WHERE id = $1
		)
	`, matchedID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	tag, err := tx.Exec(r.Context(), `
		UPDATE wa_escalation_group_bindings
		SET status = 'bound',
		    group_id = $2,
		    group_name = NULLIF($3, ''),
		    sender_jid = NULLIF($4, ''),
		    external_message_id = NULLIF($5, ''),
		    bound_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1 AND status = 'pending' AND expires_at > NOW()
	`, matchedID, req.GroupID, req.GroupName, req.SenderJID, req.ExternalMessageID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "binding code expired or already used"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	_ = s.resolveEscalationGroupUnboundAlerts(r.Context(), whatsAppSessionID)
	confirmationSent := true
	if err := s.notifyNotificationGroupByID(r.Context(), req.GroupID, formatEscalationGroupBoundMessage(req.GroupName)); err != nil {
		confirmationSent = false
		_ = s.recordNotificationGroupDeliveryFailure(context.Background(), "group_binding_confirmation", req.GroupID, err)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "bound",
		"code":             matchedCode,
		"groupId":          req.GroupID,
		"confirmationSent": confirmationSent,
	})
}

func (s *Server) syncSystemAlerts(ctx context.Context, messages []string) ([]map[string]any, error) {
	seen := map[string]bool{}
	for _, message := range messages {
		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}
		fingerprint := systemAlertFingerprint("system_health", message)
		seen[fingerprint] = true
		var id string
		var inserted bool
		err := s.db.QueryRow(ctx, `
			INSERT INTO system_alerts (fingerprint, severity, source, message, status, first_seen_at, last_seen_at, delivered_at, organization_id)
			VALUES ($1, 'critical', 'system_health', $2, 'open', NOW(), NOW(), NOW(), NULLIF($3, '')::uuid)
			ON CONFLICT (fingerprint) DO UPDATE SET
			  severity = EXCLUDED.severity,
			  message = EXCLUDED.message,
			  status = 'open',
			  last_seen_at = NOW(),
			  resolved_at = NULL
			RETURNING id, (xmax = 0) AS inserted
		`, fingerprint, message, s.organizationID(ctx)).Scan(&id, &inserted)
		if err != nil {
			return nil, err
		}
		if inserted {
			if err := s.insertAuditLog(ctx, "system_alert.deliver", "system_alert", id, map[string]any{"source": "system_health", "message": message}); err != nil {
				return nil, err
			}
		}
	}

	openRows, err := s.db.Query(ctx, `
		SELECT id, fingerprint
		FROM system_alerts
		WHERE source = 'system_health' AND status = 'open'
		  AND (organization_id IS NULL OR organization_id = NULLIF($1, '')::uuid)
	`, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer openRows.Close()
	for openRows.Next() {
		var id, fingerprint string
		if err := openRows.Scan(&id, &fingerprint); err != nil {
			return nil, err
		}
		if !seen[fingerprint] {
			if _, err := s.db.Exec(ctx, `
				UPDATE system_alerts
				SET status = 'resolved',
				    resolved_at = NOW()
				WHERE id = $1
				  AND (organization_id IS NULL OR organization_id = NULLIF($2, '')::uuid)
			`, id, s.organizationID(ctx)); err != nil {
				return nil, err
			}
		}
	}

	return s.currentOpenSystemAlerts(ctx)
}

func (s *Server) deliverProviderQuotaAlert(ctx context.Context, decision aiDecisionResponse) error {
	if !decisionHasProviderQuotaError(decision) {
		return nil
	}
	message := fmt.Sprintf("AI provider quota exhausted for model %s. Top up provider credits or reduce max_tokens before AI can answer.", decision.ModelName)
	fingerprint := systemAlertFingerprint("ai_provider", "quota_exhausted:"+decision.ModelName)
	var id string
	var inserted bool
	err := s.db.QueryRow(ctx, `
		INSERT INTO system_alerts (fingerprint, severity, source, message, status, first_seen_at, last_seen_at, delivered_at, organization_id)
		VALUES ($1, 'critical', 'ai_provider', $2, 'open', NOW(), NOW(), NOW(), NULLIF($3, '')::uuid)
		ON CONFLICT (fingerprint) DO UPDATE SET
		  severity = EXCLUDED.severity,
		  message = EXCLUDED.message,
		  status = 'open',
		  last_seen_at = NOW(),
		  resolved_at = NULL
		RETURNING id, (xmax = 0) AS inserted
	`, fingerprint, message, s.organizationID(ctx)).Scan(&id, &inserted)
	if err != nil {
		return err
	}
	if inserted {
		return s.insertAuditLog(ctx, "system_alert.deliver", "system_alert", id, map[string]any{
			"source": "ai_provider",
			"model":  decision.ModelName,
			"reason": decision.EscalationReason,
		})
	}
	return nil
}

func (s *Server) deliverAICreditAlert(ctx context.Context, neededCredits int) error {
	message := fmt.Sprintf("Internal AI credits are insufficient. Needed %d credits before AI can answer.", neededCredits)
	fingerprint := systemAlertFingerprint("billing", "ai_credits_insufficient")
	var id string
	var inserted bool
	err := s.db.QueryRow(ctx, `
		INSERT INTO system_alerts (fingerprint, severity, source, message, status, first_seen_at, last_seen_at, delivered_at, organization_id)
		VALUES ($1, 'critical', 'billing', $2, 'open', NOW(), NOW(), NOW(), NULLIF($3, '')::uuid)
		ON CONFLICT (fingerprint) DO UPDATE SET
		  severity = EXCLUDED.severity,
		  message = EXCLUDED.message,
		  status = 'open',
		  last_seen_at = NOW(),
		  resolved_at = NULL
		RETURNING id, (xmax = 0) AS inserted
	`, fingerprint, message, s.organizationID(ctx)).Scan(&id, &inserted)
	if err != nil {
		return err
	}
	if inserted {
		return s.insertAuditLog(ctx, "system_alert.deliver", "system_alert", id, map[string]any{
			"source":        "billing",
			"neededCredits": neededCredits,
		})
	}
	return nil
}

func decisionHasProviderQuotaError(decision aiDecisionResponse) bool {
	generation, _ := decision.RetrievalMetadata["generation"].(map[string]any)
	if generation == nil {
		return false
	}
	if fmt.Sprint(generation["errorType"]) == "provider_quota_exhausted" {
		return true
	}
	errorText := strings.ToLower(fmt.Sprint(generation["error"]))
	return strings.Contains(errorText, "requires more credits") ||
		strings.Contains(errorText, "quota") ||
		strings.Contains(errorText, "rate limit")
}

func (s *Server) currentOpenSystemAlerts(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, severity, source, message, status, first_seen_at, last_seen_at, delivered_at, acknowledged_at, resolved_at
		FROM system_alerts
		WHERE status = 'open'
		  AND (organization_id IS NULL OR organization_id = NULLIF($1, '')::uuid)
		ORDER BY last_seen_at DESC
		LIMIT 50
	`, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id, severity, source, message, status string
		var firstSeenAt, lastSeenAt, deliveredAt time.Time
		var acknowledgedAt, resolvedAt *time.Time
		if err := rows.Scan(&id, &severity, &source, &message, &status, &firstSeenAt, &lastSeenAt, &deliveredAt, &acknowledgedAt, &resolvedAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":             id,
			"severity":       severity,
			"source":         source,
			"message":        message,
			"status":         status,
			"firstSeenAt":    firstSeenAt,
			"lastSeenAt":     lastSeenAt,
			"deliveredAt":    deliveredAt,
			"acknowledgedAt": acknowledgedAt,
			"resolvedAt":     resolvedAt,
		})
	}
	return items, nil
}

func systemAlertFingerprint(source, message string) string {
	sum := sha256.Sum256([]byte(source + ":" + message))
	return hex.EncodeToString(sum[:])
}

func serviceStaleThreshold(serviceName string) (time.Duration, bool) {
	switch serviceName {
	case "worker", "ai-service":
		return 90 * time.Second, true
	default:
		return 0, false
	}
}

func (s *Server) handleAnalyticsOverview(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	rangeKey, startAt, bucket := analyticsWindow(r.URL.Query().Get("range"))
	if role == "owner" {
		s.handleOwnerAnalytics(w, r, rangeKey, startAt, bucket)
		return
	}

	var analytics struct {
		Range             string           `json:"range"`
		Open              int              `json:"open"`
		PendingHuman      int              `json:"pendingHuman"`
		Resolved          int              `json:"resolved"`
		AssignedToMe      int              `json:"assignedToMe"`
		EscalatedCount    int              `json:"escalatedCount"`
		ResolvedByAI      int              `json:"resolvedByAi"`
		ResolvedByHuman   int              `json:"resolvedByHuman"`
		InboundCount      int              `json:"inboundCount"`
		AIAnsweredCount   int              `json:"aiAnsweredCount"`
		AIReplyCount      int              `json:"aiReplyCount"`
		HumanReplyCount   int              `json:"humanReplyCount"`
		AvgAILatencyMS    int              `json:"avgAiLatencyMs"`
		EscalationReasons []map[string]any `json:"escalationReasons"`
		AgentPerformance  []map[string]any `json:"agentPerformance"`
		Series            []map[string]any `json:"series"`
	}
	analytics.Range = rangeKey
	err := s.db.QueryRow(r.Context(), `
		SELECT
		  COUNT(*) FILTER (WHERE status = 'open'),
		  COUNT(*) FILTER (WHERE status = 'pending_human'),
		  COUNT(*) FILTER (WHERE status = 'resolved'),
		  COUNT(*) FILTER (WHERE assigned_to = $1),
		  COUNT(*) FILTER (WHERE escalated_at IS NOT NULL)
		FROM conversations
		WHERE channel = 'whatsapp' AND organization_id = $2
	`, s.agentID(r.Context()), s.organizationID(r.Context())).Scan(
		&analytics.Open,
		&analytics.PendingHuman,
		&analytics.Resolved,
		&analytics.AssignedToMe,
		&analytics.EscalatedCount,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	err = s.db.QueryRow(r.Context(), `
		SELECT COALESCE(ROUND(AVG(latency_ms))::int, 0)
		FROM ai_runs
		WHERE latency_ms IS NOT NULL
		  AND latency_ms > 0
		  AND created_at >= $1
		  AND organization_id = $2
	`, startAt, s.organizationID(r.Context())).Scan(&analytics.AvgAILatencyMS)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	err = s.db.QueryRow(r.Context(), `
		SELECT
		  COUNT(*) FILTER (WHERE event_type = 'conversation_resolved' AND COALESCE(payload->>'resolved_by', '') = 'ai'),
		  COUNT(*) FILTER (WHERE event_type = 'conversation_resolved' AND COALESCE(payload->>'resolved_by', '') = 'human'),
		  COUNT(*) FILTER (WHERE event_type = 'escalated')
		FROM conversation_events
		WHERE created_at >= $1 AND organization_id = $2
	`, startAt, s.organizationID(r.Context())).Scan(&analytics.ResolvedByAI, &analytics.ResolvedByHuman, &analytics.EscalatedCount)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	err = s.db.QueryRow(r.Context(), `
		SELECT COUNT(*)
		FROM messages
		WHERE direction = 'inbound' AND sent_at >= $1 AND organization_id = $2
	`, startAt, s.organizationID(r.Context())).Scan(&analytics.InboundCount)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	err = s.db.QueryRow(r.Context(), `
		SELECT COUNT(*)
		FROM ai_runs
		WHERE decision = 'answer' AND created_at >= $1 AND organization_id = $2
	`, startAt, s.organizationID(r.Context())).Scan(&analytics.AIAnsweredCount)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	err = s.db.QueryRow(r.Context(), `
		SELECT
		  COUNT(*) FILTER (WHERE sender_type = 'ai'),
		  COUNT(*) FILTER (WHERE sender_type = 'agent')
		FROM messages
		WHERE direction = 'outbound' AND COALESCE(sent_at, created_at) >= $1 AND organization_id = $2
	`, startAt, s.organizationID(r.Context())).Scan(&analytics.AIReplyCount, &analytics.HumanReplyCount)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rows, err := s.db.Query(r.Context(), fmt.Sprintf(`
		WITH opened AS (
		  SELECT date_trunc('%s', created_at) AS bucket, COUNT(*) AS total
		  FROM conversations
		  WHERE channel = 'whatsapp' AND created_at >= $1 AND organization_id = $2
		  GROUP BY 1
		),
		resolved AS (
		  SELECT date_trunc('%s', created_at) AS bucket, COUNT(*) AS total
		  FROM conversation_events
		  WHERE event_type = 'conversation_resolved' AND created_at >= $1 AND organization_id = $2
		  GROUP BY 1
		),
		escalated AS (
		  SELECT date_trunc('%s', created_at) AS bucket, COUNT(*) AS total
		  FROM conversation_events
		  WHERE event_type = 'escalated' AND created_at >= $1 AND organization_id = $2
		  GROUP BY 1
		),
		inbound AS (
		  SELECT date_trunc('%s', sent_at) AS bucket, COUNT(*) AS total
		  FROM messages
		  WHERE direction = 'inbound' AND sent_at >= $1 AND organization_id = $2
		  GROUP BY 1
		),
		ai_replies AS (
		  SELECT date_trunc('%s', COALESCE(sent_at, created_at)) AS bucket, COUNT(*) AS total
		  FROM messages
		  WHERE direction = 'outbound' AND sender_type = 'ai' AND COALESCE(sent_at, created_at) >= $1 AND organization_id = $2
		  GROUP BY 1
		),
		human_replies AS (
		  SELECT date_trunc('%s', COALESCE(sent_at, created_at)) AS bucket, COUNT(*) AS total
		  FROM messages
		  WHERE direction = 'outbound' AND sender_type = 'agent' AND COALESCE(sent_at, created_at) >= $1 AND organization_id = $2
		  GROUP BY 1
		)
		SELECT
		  COALESCE(opened.bucket, resolved.bucket, escalated.bucket, inbound.bucket, ai_replies.bucket, human_replies.bucket) AS bucket,
		  COALESCE(opened.total, 0),
		  COALESCE(resolved.total, 0),
		  COALESCE(escalated.total, 0),
		  COALESCE(inbound.total, 0),
		  COALESCE(ai_replies.total, 0),
		  COALESCE(human_replies.total, 0)
		FROM opened
		FULL OUTER JOIN resolved ON resolved.bucket = opened.bucket
		FULL OUTER JOIN escalated ON escalated.bucket = COALESCE(opened.bucket, resolved.bucket)
		FULL OUTER JOIN inbound ON inbound.bucket = COALESCE(opened.bucket, resolved.bucket, escalated.bucket)
		FULL OUTER JOIN ai_replies ON ai_replies.bucket = COALESCE(opened.bucket, resolved.bucket, escalated.bucket, inbound.bucket)
		FULL OUTER JOIN human_replies ON human_replies.bucket = COALESCE(opened.bucket, resolved.bucket, escalated.bucket, inbound.bucket, ai_replies.bucket)
		ORDER BY 1 ASC
	`, bucket, bucket, bucket, bucket, bucket, bucket), startAt, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	analytics.Series = []map[string]any{}
	for rows.Next() {
		var bucketAt time.Time
		var opened, resolved, escalated, inbound, aiReplies, humanReplies int
		if err := rows.Scan(&bucketAt, &opened, &resolved, &escalated, &inbound, &aiReplies, &humanReplies); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		analytics.Series = append(analytics.Series, map[string]any{
			"date":         bucketAt.UTC().Format("2006-01-02"),
			"label":        analyticsBucketLabel(bucket, bucketAt),
			"opened":       opened,
			"resolved":     resolved,
			"escalated":    escalated,
			"inbound":      inbound,
			"aiReplies":    aiReplies,
			"humanReplies": humanReplies,
		})
	}

	reasonRows, err := s.db.Query(r.Context(), `
		SELECT TRIM(escalation_reason) AS reason, COUNT(*) AS total
		FROM ai_runs
		WHERE created_at >= $1
		  AND TRIM(COALESCE(escalation_reason, '')) <> ''
		  AND organization_id = $2
		GROUP BY 1
		ORDER BY total DESC, reason ASC
		LIMIT 5
	`, startAt, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer reasonRows.Close()

	analytics.EscalationReasons = []map[string]any{}
	for reasonRows.Next() {
		var reason string
		var total int
		if err := reasonRows.Scan(&reason, &total); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		analytics.EscalationReasons = append(analytics.EscalationReasons, map[string]any{
			"reason": reason,
			"count":  total,
		})
	}

	agentRows, err := s.db.Query(r.Context(), `
		WITH reply_counts AS (
		  SELECT sender_agent_id AS agent_id, COUNT(*) AS replies
		  FROM messages
		  WHERE direction = 'outbound'
		    AND sender_type = 'agent'
		    AND sender_agent_id IS NOT NULL
		    AND COALESCE(sent_at, created_at) >= $1
		    AND organization_id = $2
		  GROUP BY 1
		),
		assigned_counts AS (
		  SELECT assigned_to AS agent_id, COUNT(*) AS assigned_open
		  FROM conversations
		  WHERE channel = 'whatsapp'
		    AND assigned_to IS NOT NULL
		    AND status <> 'resolved'
		    AND organization_id = $2
		  GROUP BY 1
		),
		resolved_counts AS (
		  SELECT actor_id AS agent_id, COUNT(*) AS resolved
		  FROM conversation_events
		  WHERE event_type = 'conversation_resolved'
		    AND COALESCE(payload->>'resolved_by', '') = 'human'
		    AND actor_id IS NOT NULL
		    AND created_at >= $1
		    AND organization_id = $2
		  GROUP BY 1
		)
		SELECT
		  a.id::text,
		  a.name,
		  a.username,
		  a.role::text,
		  COALESCE(reply_counts.replies, 0),
		  COALESCE(assigned_counts.assigned_open, 0),
		  COALESCE(resolved_counts.resolved, 0)
		FROM agents a
		JOIN organization_members om ON om.agent_id = a.id
		LEFT JOIN reply_counts ON reply_counts.agent_id = a.id
		LEFT JOIN assigned_counts ON assigned_counts.agent_id = a.id
		LEFT JOIN resolved_counts ON resolved_counts.agent_id = a.id
		WHERE a.is_active = TRUE
		  AND om.organization_id = $2
		  AND om.status = 'active'
		  AND a.role IN ('super_admin', 'admin', 'operator')
		ORDER BY COALESCE(reply_counts.replies, 0) DESC, COALESCE(assigned_counts.assigned_open, 0) DESC, a.name ASC
		LIMIT 8
	`, startAt, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer agentRows.Close()

	analytics.AgentPerformance = []map[string]any{}
	for agentRows.Next() {
		var id, name, username, role string
		var replies, assignedOpen, resolved int
		if err := agentRows.Scan(&id, &name, &username, &role, &replies, &assignedOpen, &resolved); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		analytics.AgentPerformance = append(analytics.AgentPerformance, map[string]any{
			"id":           id,
			"name":         name,
			"username":     username,
			"role":         role,
			"replies":      replies,
			"assignedOpen": assignedOpen,
			"resolved":     resolved,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"analytics": analytics})
}

func (s *Server) handleOwnerAnalytics(w http.ResponseWriter, r *http.Request, rangeKey string, startAt time.Time, bucket string) {
	var analytics struct {
		Range               string           `json:"range"`
		TotalCreditsUsed    int              `json:"totalCreditsUsed"`
		MonthlyRemaining    int              `json:"monthlyRemaining"`
		AdditionalRemaining int              `json:"additionalRemaining"`
		TotalUsageRecords   int              `json:"totalUsageRecords"`
		TotalCostIDR        float64          `json:"totalCostIdr"`
		AIPlaygroundRuns    int              `json:"aiPlaygroundRuns"`
		LiveInboundRuns     int              `json:"liveInboundRuns"`
		Series              []map[string]any `json:"series"`
	}
	analytics.Range = rangeKey
	err := s.db.QueryRow(r.Context(), `
		SELECT
		  COALESCE(SUM(credits_used) FILTER (WHERE created_at >= $1 AND organization_id = $2), 0),
		  (SELECT monthly_credits_remaining FROM credit_wallet WHERE organization_id = $2 ORDER BY created_at ASC LIMIT 1),
		  (SELECT additional_credits_remaining FROM credit_wallet WHERE organization_id = $2 ORDER BY created_at ASC LIMIT 1),
		  COUNT(*) FILTER (WHERE created_at >= $1 AND organization_id = $2),
		  COALESCE(SUM(cost_idr) FILTER (WHERE created_at >= $1 AND organization_id = $2), 0),
		  COUNT(*) FILTER (WHERE usage_type = 'playground' AND created_at >= $1 AND organization_id = $2),
		  COUNT(*) FILTER (WHERE usage_type = 'live_inbound' AND created_at >= $1 AND organization_id = $2)
		FROM credit_usage_logs
	`, startAt, s.organizationID(r.Context())).Scan(
		&analytics.TotalCreditsUsed,
		&analytics.MonthlyRemaining,
		&analytics.AdditionalRemaining,
		&analytics.TotalUsageRecords,
		&analytics.TotalCostIDR,
		&analytics.AIPlaygroundRuns,
		&analytics.LiveInboundRuns,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rows, err := s.db.Query(r.Context(), fmt.Sprintf(`
		SELECT
		  date_trunc('%s', created_at) AS bucket,
		  COALESCE(SUM(credits_used), 0) AS credits_used,
		  COALESCE(SUM(cost_idr), 0) AS cost_idr,
		  COUNT(*) FILTER (WHERE usage_type = 'playground') AS playground_runs,
		  COUNT(*) FILTER (WHERE usage_type = 'live_inbound') AS live_inbound_runs
		FROM credit_usage_logs
		WHERE created_at >= $1
		  AND organization_id = $2
		GROUP BY 1
		ORDER BY 1 ASC
	`, bucket), startAt, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	analytics.Series = []map[string]any{}
	for rows.Next() {
		var bucketAt time.Time
		var creditsUsed, playgroundRuns, liveInboundRuns int
		var costIDR float64
		if err := rows.Scan(&bucketAt, &creditsUsed, &costIDR, &playgroundRuns, &liveInboundRuns); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		analytics.Series = append(analytics.Series, map[string]any{
			"date":            bucketAt.UTC().Format("2006-01-02"),
			"label":           analyticsBucketLabel(bucket, bucketAt),
			"creditsUsed":     creditsUsed,
			"costIdr":         costIDR,
			"playgroundRuns":  playgroundRuns,
			"liveInboundRuns": liveInboundRuns,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"analytics": analytics})
}

func (s *Server) handleAISettings(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role == "owner" || role == "operator" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		var payload map[string]any
		if err := s.db.QueryRow(r.Context(), `
			SELECT row_to_json(t)
			FROM (
			  SELECT
			    id,
			    system_prompt,
			    escalation_prompt,
			    fallback_waiting_message,
			    allow_clarification,
			    max_clarification_count,
			    answer_only_from_knowledge,
			    dont_broaden_topic,
			    forbid_promises,
			    forbid_sensitive_answers,
			    require_action_confirmation,
			    escalate_low_confidence,
			    guide_next_step,
			    concise_response,
			    allow_auto_update_contact_name,
			    only_fill_name_if_empty,
			    customer_memory_enabled,
			    customer_memory_auto_save_enabled,
			    customer_memory_admin_notes_enabled,
			    customer_memory_ai_extraction_enabled,
			    customer_memory_verifier_enabled,
			    customer_memory_max_items,
			    customer_memory_max_chars,
			    customer_memory_retention_days,
			    is_active,
			    updated_at
			  FROM ai_settings
			  WHERE is_active = TRUE AND organization_id = $1
			  ORDER BY updated_at DESC
			  LIMIT 1
			) t
		`, s.organizationID(r.Context())).Scan(&payload); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"settings": payload})
	case http.MethodPost:
		if role != "admin" && role != "super_admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		var req struct {
			SystemPrompt            string `json:"systemPrompt"`
			EscalationPrompt        string `json:"escalationPrompt"`
			FallbackWaitingMessage  string `json:"fallbackWaitingMessage"`
			AllowClarification      bool   `json:"allowClarification"`
			MaxClarificationCount   int    `json:"maxClarificationCount"`
			AnswerOnlyFromKnowledge bool   `json:"answerOnlyFromKnowledge"`
			DontBroadenTopic        bool   `json:"dontBroadenTopic"`
			ForbidPromises          bool   `json:"forbidPromises"`
			ForbidSensitiveAnswers  bool   `json:"forbidSensitiveAnswers"`
			RequireActionConfirm    bool   `json:"requireActionConfirmation"`
			EscalateLowConfidence   bool   `json:"escalateLowConfidence"`
			GuideNextStep           bool   `json:"guideNextStep"`
			ConciseResponse         bool   `json:"conciseResponse"`
			AllowAutoUpdateContact  bool   `json:"allowAutoUpdateContactName"`
			OnlyFillNameIfEmpty     bool   `json:"onlyFillNameIfEmpty"`
			CustomerMemoryEnabled   bool   `json:"customerMemoryEnabled"`
			CustomerMemoryAutoSave  bool   `json:"customerMemoryAutoSaveEnabled"`
			CustomerMemoryAdmin     bool   `json:"customerMemoryAdminNotesEnabled"`
			CustomerMemoryAIExtract bool   `json:"customerMemoryAiExtractionEnabled"`
			CustomerMemoryVerifier  bool   `json:"customerMemoryVerifierEnabled"`
			CustomerMemoryMaxItems  int    `json:"customerMemoryMaxItems"`
			CustomerMemoryMaxChars  int    `json:"customerMemoryMaxChars"`
			CustomerMemoryRetention int    `json:"customerMemoryRetentionDays"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.CustomerMemoryMaxItems = clampInt(req.CustomerMemoryMaxItems, 1, 10, 5)
		req.CustomerMemoryMaxChars = clampInt(req.CustomerMemoryMaxChars, 200, 1200, 800)
		req.CustomerMemoryRetention = clampInt(req.CustomerMemoryRetention, 30, 730, 180)
		tx, err := s.db.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer tx.Rollback(r.Context())
		if _, err := tx.Exec(r.Context(), `UPDATE ai_settings SET is_active = FALSE WHERE is_active = TRUE AND organization_id = $1`, s.organizationID(r.Context())); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO ai_settings (
			  system_prompt,
			  escalation_prompt,
			  fallback_waiting_message,
			  allow_clarification,
			  max_clarification_count,
			  answer_only_from_knowledge,
			  dont_broaden_topic,
			  forbid_promises,
			  forbid_sensitive_answers,
			  require_action_confirmation,
			  escalate_low_confidence,
			  guide_next_step,
			  concise_response,
			  allow_auto_update_contact_name,
			  only_fill_name_if_empty,
			  customer_memory_enabled,
			  customer_memory_auto_save_enabled,
			  customer_memory_admin_notes_enabled,
			  customer_memory_ai_extraction_enabled,
			  customer_memory_verifier_enabled,
			  customer_memory_max_items,
			  customer_memory_max_chars,
			  customer_memory_retention_days,
			  is_active,
			  updated_by,
			  organization_id
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,TRUE,$24,$25)
		`,
			req.SystemPrompt,
			req.EscalationPrompt,
			req.FallbackWaitingMessage,
			req.AllowClarification,
			req.MaxClarificationCount,
			req.AnswerOnlyFromKnowledge,
			req.DontBroadenTopic,
			req.ForbidPromises,
			req.ForbidSensitiveAnswers,
			req.RequireActionConfirm,
			req.EscalateLowConfidence,
			req.GuideNextStep,
			req.ConciseResponse,
			req.AllowAutoUpdateContact,
			req.OnlyFillNameIfEmpty,
			req.CustomerMemoryEnabled,
			req.CustomerMemoryAutoSave,
			req.CustomerMemoryAdmin,
			req.CustomerMemoryAIExtract,
			req.CustomerMemoryVerifier,
			req.CustomerMemoryMaxItems,
			req.CustomerMemoryMaxChars,
			req.CustomerMemoryRetention,
			s.agentID(r.Context()),
			s.organizationID(r.Context()),
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLogTx(r.Context(), tx, "ai_settings.update", "ai_settings", "", map[string]any{
			"answer_only_from_knowledge":  req.AnswerOnlyFromKnowledge,
			"dont_broaden_topic":          req.DontBroadenTopic,
			"allow_clarification":         req.AllowClarification,
			"max_clarification_count":     req.MaxClarificationCount,
			"require_action_confirmation": req.RequireActionConfirm,
			"escalate_low_confidence":     req.EscalateLowConfidence,
			"guide_next_step":             req.GuideNextStep,
			"concise_response":            req.ConciseResponse,
			"customer_memory_enabled":     req.CustomerMemoryEnabled,
			"customer_memory_auto_save":   req.CustomerMemoryAutoSave,
			"customer_memory_ai_extract":  req.CustomerMemoryAIExtract,
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleAIModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	aliases := s.openRouterChatModelAliases(r.Context())
	availableModels := s.openRouterAvailableChatModelIDs(r.Context())
	includeUnavailable := role == "owner"
	models, err := s.fetchOpenRouterSupportedChatModels(r.Context(), aliases, availableModels, includeUnavailable)
	source := "openrouter"
	resp := map[string]any{"items": models, "source": source}
	if err != nil {
		resp["items"] = fallbackOpenRouterChatModels(aliases, availableModels, includeUnavailable)
		resp["source"] = "fallback"
		resp["warning"] = err.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePlaygroundShortcuts(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "admin" && role != "super_admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin and super_admin can manage AI playground"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		items, err := s.listPlaygroundShortcuts(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
		var req struct {
			Label            string `json:"label"`
			Question         string `json:"question"`
			ExpectedBehavior string `json:"expectedBehavior"`
			IsActive         bool   `json:"isActive"`
			SortOrder        int    `json:"sortOrder"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.Label = strings.TrimSpace(req.Label)
		req.Question = strings.TrimSpace(req.Question)
		req.ExpectedBehavior = strings.TrimSpace(req.ExpectedBehavior)
		if req.Label == "" || req.Question == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "label and question are required"})
			return
		}
		if req.SortOrder == 0 {
			req.SortOrder = 100
		}
		_, err := s.db.Exec(r.Context(), `
			INSERT INTO playground_shortcuts (label, question, expected_behavior, is_active, sort_order, created_by, updated_by, organization_id)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::uuid, NULLIF($6, '')::uuid, $7)
		`, req.Label, req.Question, req.ExpectedBehavior, req.IsActive, req.SortOrder, s.agentID(r.Context()), s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLog(r.Context(), "playground_shortcut.create", "playground_shortcut", "", map[string]any{"label": req.Label}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		items, err := s.listPlaygroundShortcuts(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "items": items})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handlePlaygroundShortcutRoutes(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "admin" && role != "super_admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin and super_admin can manage AI playground"})
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/playground/shortcuts/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPut:
		r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
		var req struct {
			Label            string `json:"label"`
			Question         string `json:"question"`
			ExpectedBehavior string `json:"expectedBehavior"`
			IsActive         bool   `json:"isActive"`
			SortOrder        int    `json:"sortOrder"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.Label = strings.TrimSpace(req.Label)
		req.Question = strings.TrimSpace(req.Question)
		req.ExpectedBehavior = strings.TrimSpace(req.ExpectedBehavior)
		if req.Label == "" || req.Question == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "label and question are required"})
			return
		}
		if req.SortOrder == 0 {
			req.SortOrder = 100
		}
		tag, err := s.db.Exec(r.Context(), `
			UPDATE playground_shortcuts
			SET label = $2,
			    question = $3,
			    expected_behavior = $4,
			    is_active = $5,
			    sort_order = $6,
			    updated_by = NULLIF($7, '')::uuid,
			    updated_at = NOW()
			WHERE id = $1 AND organization_id = $8
		`, id, req.Label, req.Question, req.ExpectedBehavior, req.IsActive, req.SortOrder, s.agentID(r.Context()), s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "shortcut not found"})
			return
		}
		if err := s.insertAuditLog(r.Context(), "playground_shortcut.update", "playground_shortcut", id, map[string]any{"label": req.Label}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		items, err := s.listPlaygroundShortcuts(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "items": items})
	case http.MethodDelete:
		tag, err := s.db.Exec(r.Context(), `DELETE FROM playground_shortcuts WHERE id = $1 AND organization_id = $2`, id, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "shortcut not found"})
			return
		}
		if err := s.insertAuditLog(r.Context(), "playground_shortcut.delete", "playground_shortcut", id, map[string]any{}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		items, err := s.listPlaygroundShortcuts(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "items": items})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) listPlaygroundShortcuts(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, label, question, expected_behavior, is_active, sort_order, created_at, updated_at
		FROM playground_shortcuts
		WHERE organization_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, label, question, expectedBehavior string
		var isActive bool
		var sortOrder int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &label, &question, &expectedBehavior, &isActive, &sortOrder, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":               id,
			"label":            label,
			"question":         question,
			"expectedBehavior": expectedBehavior,
			"isActive":         isActive,
			"sortOrder":        sortOrder,
			"createdAt":        createdAt,
			"updatedAt":        updatedAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) handlePlaygroundRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "admin" && role != "super_admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin and super_admin can run AI playground"})
		return
	}
	if !s.enforceRateLimit(w, r, s.playgroundLimit, "playground:run") {
		return
	}

	var req struct {
		MessageText         string                 `json:"messageText"`
		CustomerName        string                 `json:"customerName"`
		LegacyCandidateName string                 `json:"candidateName"`
		AIAgentID           string                 `json:"aiAgentId"`
		History             []aiChatHistoryMessage `json:"history"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if strings.TrimSpace(req.CustomerName) == "" {
		req.CustomerName = req.LegacyCandidateName
	}
	req.AIAgentID = strings.TrimSpace(req.AIAgentID)
	if strings.TrimSpace(req.MessageText) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "messageText is required"})
		return
	}
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
	if ok, needed, err := s.ensureCreditsAvailable(r.Context(), 900, 180); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	} else if !ok {
		if err := s.deliverAICreditAlert(r.Context(), needed); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusPaymentRequired, map[string]any{
			"error":         "insufficient AI credits",
			"neededCredits": needed,
		})
		return
	}

	conversationID, err := s.ensurePlaygroundConversation(r.Context(), req.CustomerName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	aiContext := context.WithValue(r.Context(), contextAIAgentID, req.AIAgentID)
	decision, err := s.callAIWithHistory(aiContext, conversationID, req.MessageText, req.CustomerName, req.History)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	if err := s.deliverProviderQuotaAlert(r.Context(), decision); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	aiSettingsUpdatedAt, _ := s.activeAISettingsUpdatedAt(r.Context())
	aiRunID := ""
	err = s.db.QueryRow(r.Context(), `
		INSERT INTO ai_runs (
		  conversation_id,
		  message_id,
		  decision,
		  question_summary,
		  answer_text,
		  confidence_score,
		  escalation_reason,
		  retrieval_metadata,
		  model_name,
		  latency_ms,
		  ai_settings_updated_at_snapshot,
		  organization_id
		)
		VALUES ($1, NULL, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11)
		RETURNING id
	`,
		conversationID,
		decision.Decision,
		req.MessageText,
		nullableString(decision.AnswerText),
		decision.ConfidenceScore,
		nullableString(decision.EscalationReason),
		marshalJSON(map[string]any{
			"mode":      "playground",
			"aiAgentId": req.AIAgentID,
			"retrieval": decision.RetrievalMetadata,
		}),
		decision.ModelName,
		decision.LatencyMS,
		aiSettingsUpdatedAt,
		s.organizationID(r.Context()),
	).Scan(&aiRunID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	inputTokens, outputTokens, embeddingTokens, actualCostUSD := usageTokensFromDecision(decision, 900, 180, 0)
	usageResult, err := s.logCreditUsage(r.Context(), "", "", aiRunID, "playground", decision.ModelName, inputTokens, outputTokens, embeddingTokens, actualCostUSD)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertAIRunCostSteps(r.Context(), aiRunID, decision, usageResult); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if decision.UsageMetadata == nil {
		decision.UsageMetadata = map[string]any{}
	}
	decision.UsageMetadata["input_tokens"] = inputTokens
	decision.UsageMetadata["output_tokens"] = outputTokens
	decision.UsageMetadata["embedding_tokens"] = embeddingTokens
	decision.UsageMetadata["cost_usd"] = usageResult.CostUSD
	decision.UsageMetadata["cost_idr"] = usageResult.CostIDR
	decision.UsageMetadata["credits_used"] = usageResult.CreditsUsed
	decision.UsageMetadata["creditsUsed"] = usageResult.CreditsUsed
	decision.UsageMetadata["credit_source"] = usageResult.CreditSource
	decision.UsageMetadata["creditSource"] = usageResult.CreditSource
	decision.UsageMetadata["monthly_credits_consumed"] = usageResult.MonthlyConsumed
	decision.UsageMetadata["additional_credits_consumed"] = usageResult.AdditionalConsumed
	decision.UsageMetadata["monthly_credits_remaining"] = usageResult.MonthlyRemaining
	decision.UsageMetadata["additional_credits_remaining"] = usageResult.AdditionalRemaining

	writeJSON(w, http.StatusOK, map[string]any{
		"result":  decision,
		"aiRunId": aiRunID,
		"wallet": map[string]any{
			"monthlyRemaining":    usageResult.MonthlyRemaining,
			"additionalRemaining": usageResult.AdditionalRemaining,
		},
	})
}

func (s *Server) handleSimulateInbound(w http.ResponseWriter, r *http.Request) {
	if s.cfg.AppEnv == "production" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "admin" && role != "super_admin" && role != "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var req struct {
		ConversationID string `json:"conversationId"`
		Text           string `json:"text"`
		AutoReply      bool   `json:"autoReply"`
		ForceDecision  string `json:"forceDecision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.ConversationID = strings.TrimSpace(req.ConversationID)
	req.Text = strings.TrimSpace(req.Text)
	req.ForceDecision = strings.TrimSpace(req.ForceDecision)
	if req.ConversationID == "" || req.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversationId and text are required"})
		return
	}
	if req.ForceDecision != "" && req.ForceDecision != "answer" && req.ForceDecision != "escalate" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "forceDecision must be empty, answer, or escalate"})
		return
	}

	var contactName string
	if err := s.db.QueryRow(r.Context(), `
		SELECT ct.name
		FROM conversations c
		JOIN contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
		WHERE c.id = $1 AND c.organization_id = $2
	`, req.ConversationID, s.organizationID(r.Context())).Scan(&contactName); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	inboundExternalID := fmt.Sprintf("sim-in-%d", time.Now().UnixNano())
	var inboundMessageID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO messages (
		  conversation_id,
		  external_message_id,
		  sender_type,
		  direction,
		  content_type,
		  text,
		  sent_at,
		  delivered_at,
		  organization_id
		)
		VALUES ($1, $2, 'customer', 'inbound', 'text', $3, NOW(), NOW(), $4)
		RETURNING id
	`, req.ConversationID, inboundExternalID, req.Text, s.organizationID(r.Context())).Scan(&inboundMessageID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE conversations
		SET last_message_id = $2,
		    last_message_at = NOW(),
		    status = 'open',
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $3
	`, req.ConversationID, inboundMessageID, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertEventTx(r.Context(), tx, req.ConversationID, "conversation_reopened", s.agentID(r.Context()), map[string]any{
		"simulated": true,
		"text":      req.Text,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	response := map[string]any{
		"status":            "ok",
		"inboundMessageId":  inboundMessageID,
		"conversationId":    req.ConversationID,
		"autoReplyExecuted": false,
	}

	if req.AutoReply {
		decision, err := s.callAI(r.Context(), req.ConversationID, req.Text, contactName)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		if err := s.deliverProviderQuotaAlert(r.Context(), decision); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if req.ForceDecision == "answer" {
			decision.Decision = "answer"
			if strings.TrimSpace(decision.AnswerText) == "" {
				decision.AnswerText = "Pesan Anda sudah kami terima. Tahap berikutnya akan dikonfirmasi oleh tim operasional."
			}
			decision.EscalationReason = ""
		}
		if req.ForceDecision == "escalate" {
			decision.Decision = "escalate"
			decision.AnswerText = ""
			if strings.TrimSpace(decision.EscalationReason) == "" {
				decision.EscalationReason = "Simulated escalation requested from dev harness."
			}
		}

		aiSettingsUpdatedAt, _ := s.activeAISettingsUpdatedAt(r.Context())
		aiRunID := ""
		err = tx.QueryRow(r.Context(), `
			INSERT INTO ai_runs (
			  conversation_id,
			  message_id,
			  decision,
			  question_summary,
			  answer_text,
			  confidence_score,
			  escalation_reason,
			  retrieval_metadata,
			  model_name,
			  latency_ms,
			  ai_settings_updated_at_snapshot,
			  organization_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12)
			RETURNING id
		`,
			req.ConversationID,
			inboundMessageID,
			decision.Decision,
			req.Text,
			nullableString(decision.AnswerText),
			decision.ConfidenceScore,
			nullableString(decision.EscalationReason),
			marshalJSON(map[string]any{
				"mode":      "simulated_inbound",
				"forced":    req.ForceDecision,
				"retrieval": decision.RetrievalMetadata,
			}),
			decision.ModelName,
			decision.LatencyMS,
			aiSettingsUpdatedAt,
			s.organizationID(r.Context()),
		).Scan(&aiRunID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		response["aiRunId"] = aiRunID
		response["decision"] = decision.Decision
		response["autoReplyExecuted"] = true
		response["forced"] = req.ForceDecision

		if decision.Decision == "answer" && strings.TrimSpace(decision.AnswerText) != "" {
			outboundExternalID := fmt.Sprintf("sim-ai-%d", time.Now().UnixNano())
			var outboundMessageID string
			err = tx.QueryRow(r.Context(), `
				INSERT INTO messages (
				  conversation_id,
				  external_message_id,
				  sender_type,
				  direction,
				  content_type,
				  text,
				  sent_at,
				  delivered_at,
				  organization_id
				)
				VALUES ($1, $2, 'ai', 'outbound', 'text', $3, NOW(), NOW(), $4)
				RETURNING id
			`, req.ConversationID, outboundExternalID, decision.AnswerText, s.organizationID(r.Context())).Scan(&outboundMessageID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			_, err = tx.Exec(r.Context(), `
				UPDATE conversations
				SET last_message_id = $2,
				    last_message_at = NOW(),
				    mode = 'ai',
				    status = 'open',
				    escalation_reason = NULL,
				    updated_at = NOW()
				WHERE id = $1 AND organization_id = $3
			`, req.ConversationID, outboundMessageID, s.organizationID(r.Context()))
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := s.insertEventTx(r.Context(), tx, req.ConversationID, "ai_replied", s.agentID(r.Context()), map[string]any{
				"simulated": true,
				"ai_run_id": aiRunID,
			}); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			response["replyMessageId"] = outboundMessageID
		} else {
			_, err = tx.Exec(r.Context(), `
				UPDATE conversations
				SET mode = 'human',
				    status = 'pending_human',
				    escalation_reason = $2,
				    updated_at = NOW()
			WHERE id = $1 AND organization_id = $3
		`, req.ConversationID, nullableString(decision.EscalationReason), s.organizationID(r.Context()))
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := s.insertEventTx(r.Context(), tx, req.ConversationID, "escalated", s.agentID(r.Context()), map[string]any{
				"simulated":         true,
				"ai_run_id":         aiRunID,
				"escalation_reason": decision.EscalationReason,
				"confidence_score":  decision.ConfidenceScore,
			}); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleSimulateConversation(w http.ResponseWriter, r *http.Request) {
	if s.cfg.AppEnv == "production" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if !isOpsRole(role) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var req struct {
		CustomerName        string `json:"customerName"`
		LegacyCandidateName string `json:"candidateName"`
		Phone               string `json:"phone"`
		Text                string `json:"text"`
		AutoReply           bool   `json:"autoReply"`
		ForceDecision       string `json:"forceDecision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if strings.TrimSpace(req.CustomerName) == "" {
		req.CustomerName = req.LegacyCandidateName
	}
	req.CustomerName = strings.TrimSpace(req.CustomerName)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Text = strings.TrimSpace(req.Text)
	req.ForceDecision = strings.TrimSpace(strings.ToLower(req.ForceDecision))

	if req.CustomerName == "" {
		req.CustomerName = "Pelanggan Test"
	}
	if req.Phone == "" {
		req.Phone = fmt.Sprintf("+620%010d", time.Now().UnixNano()%10000000000)
	}
	if req.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text is required"})
		return
	}
	if req.ForceDecision != "" && req.ForceDecision != "answer" && req.ForceDecision != "escalate" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "forceDecision must be empty, answer, or escalate"})
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	var contactID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO contacts (name, phone, notes, organization_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, req.CustomerName, req.Phone, "created from dev harness", s.organizationID(r.Context())).Scan(&contactID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "contacts_phone_key") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "phone already exists, use a different number for a new simulated conversation"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var conversationID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO conversations (contact_id, channel, mode, status, last_message_at, organization_id)
		VALUES ($1, 'whatsapp', 'ai', 'open', NOW(), $2)
		RETURNING id
	`, contactID, s.organizationID(r.Context())).Scan(&conversationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	inboundExternalID := fmt.Sprintf("sim-new-%d", time.Now().UnixNano())
	var inboundMessageID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO messages (
		  conversation_id,
		  external_message_id,
		  sender_type,
		  direction,
		  content_type,
		  text,
		  sent_at,
		  delivered_at,
		  organization_id
		)
		VALUES ($1, $2, 'customer', 'inbound', 'text', $3, NOW(), NOW(), $4)
		RETURNING id
	`, conversationID, inboundExternalID, req.Text, s.organizationID(r.Context())).Scan(&inboundMessageID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE conversations
		SET last_message_id = $2,
		    last_message_at = NOW(),
		    mode = 'ai',
		    status = 'open',
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $3
	`, conversationID, inboundMessageID, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := s.insertEventTx(r.Context(), tx, conversationID, "conversation_opened", s.agentID(r.Context()), map[string]any{
		"simulated":     true,
		"customer_name": req.CustomerName,
		"phone":         req.Phone,
		"text":          req.Text,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	response := map[string]any{
		"status":            "ok",
		"contactId":         contactID,
		"conversationId":    conversationID,
		"inboundMessageId":  inboundMessageID,
		"customerName":      req.CustomerName,
		"phone":             req.Phone,
		"autoReplyExecuted": false,
	}

	if req.AutoReply {
		decision, err := s.callAI(r.Context(), conversationID, req.Text, req.CustomerName)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		if err := s.deliverProviderQuotaAlert(r.Context(), decision); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if req.ForceDecision == "answer" {
			decision.Decision = "answer"
			if strings.TrimSpace(decision.AnswerText) == "" {
				decision.AnswerText = "Pesan Anda sudah kami terima. Tahap berikutnya akan dikonfirmasi oleh tim operasional."
			}
			decision.EscalationReason = ""
		}
		if req.ForceDecision == "escalate" {
			decision.Decision = "escalate"
			decision.AnswerText = ""
			if strings.TrimSpace(decision.EscalationReason) == "" {
				decision.EscalationReason = "Simulated escalation requested from dev harness."
			}
		}

		aiSettingsUpdatedAt, _ := s.activeAISettingsUpdatedAt(r.Context())
		aiRunID := ""
		err = tx.QueryRow(r.Context(), `
			INSERT INTO ai_runs (
			  conversation_id,
			  message_id,
			  decision,
			  question_summary,
			  answer_text,
			  confidence_score,
			  escalation_reason,
			  retrieval_metadata,
			  model_name,
			  latency_ms,
			  ai_settings_updated_at_snapshot,
			  organization_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12)
			RETURNING id
		`,
			conversationID,
			inboundMessageID,
			decision.Decision,
			req.Text,
			nullableString(decision.AnswerText),
			decision.ConfidenceScore,
			nullableString(decision.EscalationReason),
			marshalJSON(map[string]any{
				"mode":      "simulated_conversation",
				"forced":    req.ForceDecision,
				"retrieval": decision.RetrievalMetadata,
			}),
			decision.ModelName,
			decision.LatencyMS,
			aiSettingsUpdatedAt,
			s.organizationID(r.Context()),
		).Scan(&aiRunID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		response["aiRunId"] = aiRunID
		response["decision"] = decision.Decision
		response["autoReplyExecuted"] = true
		response["forced"] = req.ForceDecision

		if decision.Decision == "answer" && strings.TrimSpace(decision.AnswerText) != "" {
			outboundExternalID := fmt.Sprintf("sim-new-ai-%d", time.Now().UnixNano())
			var outboundMessageID string
			err = tx.QueryRow(r.Context(), `
				INSERT INTO messages (
				  conversation_id,
				  external_message_id,
				  sender_type,
				  direction,
				  content_type,
				  text,
				  sent_at,
				  delivered_at,
				  organization_id
				)
				VALUES ($1, $2, 'ai', 'outbound', 'text', $3, NOW(), NOW(), $4)
				RETURNING id
			`, conversationID, outboundExternalID, decision.AnswerText, s.organizationID(r.Context())).Scan(&outboundMessageID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			_, err = tx.Exec(r.Context(), `
				UPDATE conversations
				SET last_message_id = $2,
				    last_message_at = NOW(),
				    mode = 'ai',
				    status = 'open',
				    escalation_reason = NULL,
				    updated_at = NOW()
				WHERE id = $1 AND organization_id = $3
			`, conversationID, outboundMessageID, s.organizationID(r.Context()))
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := s.insertEventTx(r.Context(), tx, conversationID, "ai_replied", s.agentID(r.Context()), map[string]any{
				"simulated": true,
				"ai_run_id": aiRunID,
			}); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			response["replyMessageId"] = outboundMessageID
		} else {
			_, err = tx.Exec(r.Context(), `
				UPDATE conversations
				SET mode = 'human',
				    status = 'pending_human',
				    escalation_reason = $2,
				    escalated_at = NOW(),
				    updated_at = NOW()
			WHERE id = $1 AND organization_id = $3
		`, conversationID, nullableString(decision.EscalationReason), s.organizationID(r.Context()))
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := s.insertEventTx(r.Context(), tx, conversationID, "escalated", s.agentID(r.Context()), map[string]any{
				"simulated":         true,
				"ai_run_id":         aiRunID,
				"escalation_reason": decision.EscalationReason,
				"confidence_score":  decision.ConfidenceScore,
			}); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleAIRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role == "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "owner cannot inspect ai runs"})
		return
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT
		  id,
		  decision,
		  COALESCE(question_summary, ''),
		  COALESCE(answer_text, ''),
		  COALESCE(confidence_score, 0),
		  COALESCE(escalation_reason, ''),
		  COALESCE(retrieval_metadata, '{}'::jsonb),
		  COALESCE(model_name, ''),
		  COALESCE(latency_ms, 0),
		  created_at
		FROM ai_runs
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id, decision, questionSummary, answerText, escalationReason, modelName string
		var confidenceScore float64
		var latencyMS int
		var createdAt time.Time
		var retrievalMetadata []byte
		if err := rows.Scan(
			&id,
			&decision,
			&questionSummary,
			&answerText,
			&confidenceScore,
			&escalationReason,
			&retrievalMetadata,
			&modelName,
			&latencyMS,
			&createdAt,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		metadata := map[string]any{}
		_ = json.Unmarshal(retrievalMetadata, &metadata)
		items = append(items, map[string]any{
			"id":                id,
			"decision":          decision,
			"questionSummary":   questionSummary,
			"answerText":        answerText,
			"confidenceScore":   confidenceScore,
			"escalationReason":  escalationReason,
			"retrievalMetadata": metadata,
			"modelName":         modelName,
			"latencyMS":         latencyMS,
			"createdAt":         createdAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAIRunRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role == "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "owner cannot inspect ai runs"})
		return
	}

	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/ai-runs/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	var decision, questionSummary, answerText, escalationReason, modelName, conversationID, messageID string
	var confidenceScore float64
	var latencyMS int
	var createdAt time.Time
	var retrievalMetadata []byte
	err := s.db.QueryRow(r.Context(), `
		SELECT
		  COALESCE(conversation_id::text, ''),
		  COALESCE(message_id::text, ''),
		  decision,
		  COALESCE(question_summary, ''),
		  COALESCE(answer_text, ''),
		  COALESCE(confidence_score, 0),
		  COALESCE(escalation_reason, ''),
		  COALESCE(retrieval_metadata, '{}'::jsonb),
		  COALESCE(model_name, ''),
		  COALESCE(latency_ms, 0),
		  created_at
		FROM ai_runs
		WHERE id = $1
		  AND organization_id = $2
	`, id, s.organizationID(r.Context())).Scan(&conversationID, &messageID, &decision, &questionSummary, &answerText, &confidenceScore, &escalationReason, &retrievalMetadata, &modelName, &latencyMS, &createdAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "ai run not found"})
		return
	}
	metadata := map[string]any{}
	_ = json.Unmarshal(retrievalMetadata, &metadata)
	writeJSON(w, http.StatusOK, map[string]any{
		"run": map[string]any{
			"id":                id,
			"conversationId":    conversationID,
			"messageId":         messageID,
			"decision":          decision,
			"questionSummary":   questionSummary,
			"answerText":        answerText,
			"confidenceScore":   confidenceScore,
			"escalationReason":  escalationReason,
			"retrievalMetadata": metadata,
			"modelName":         modelName,
			"latencyMS":         latencyMS,
			"createdAt":         createdAt,
		},
	})
}

func (s *Server) handleConversationRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/conversations/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]

	if len(parts) == 1 && r.Method == http.MethodGet {
		s.handleConversationDetail(w, r, id)
		return
	}

	if len(parts) == 2 && r.Method == http.MethodPost {
		switch parts[1] {
		case "takeover":
			s.handleTakeover(w, r, id)
			return
		case "force-takeover":
			s.handleForceTakeover(w, r, id)
			return
		case "assign":
			s.handleAssign(w, r, id)
			return
		case "return-to-ai":
			s.handleReturnToAI(w, r, id)
			return
		case "resolve":
			s.handleResolve(w, r, id)
			return
		case "manual-message":
			s.handleManualMessage(w, r, id)
			return
		case "workflow":
			s.handleConversationWorkflow(w, r, id)
			return
		}
	}

	http.NotFound(w, r)
}

func (s *Server) handleConversationDetail(w http.ResponseWriter, r *http.Request, id string) {
	role := s.role(r.Context())
	if role == "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "owner cannot access client chat content"})
		return
	}

	var detail struct {
		ID                    string     `json:"id"`
		ContactID             string     `json:"contactId"`
		ContactName           string     `json:"contactName"`
		Phone                 string     `json:"phone"`
		Mode                  string     `json:"mode"`
		Status                string     `json:"status"`
		WhatsAppSessionID     string     `json:"whatsappSessionId"`
		WhatsAppSessionName   string     `json:"whatsappSession"`
		WhatsAppProvider      string     `json:"whatsappProvider"`
		AIAgentID             string     `json:"aiAgentId"`
		AIAgentName           string     `json:"aiAgentName"`
		AssignedToID          *string    `json:"assignedToId"`
		AssignedToName        string     `json:"assignedToName"`
		EscalationReason      string     `json:"escalationReason"`
		Priority              string     `json:"priority"`
		SLADueAt              *time.Time `json:"slaDueAt"`
		InternalNote          string     `json:"internalNote"`
		LastMessageAt         time.Time  `json:"lastMessageAt"`
		LastCustomerMessageAt *time.Time `json:"lastCustomerMessageAt"`
		OpenDealCount         int        `json:"openDealCount"`
		OpenDeals             any        `json:"openDeals"`
	}
	var openDealsJSON string

	err := s.db.QueryRow(r.Context(), `
		SELECT
		  c.id,
		  ct.id::text,
		  COALESCE(NULLIF(ct.name, ''), ct.phone),
		  ct.phone,
		  c.mode::text,
		  c.status::text,
		  COALESCE(c.whatsapp_session_id::text, ''),
		  COALESCE(ws.label, ''),
		  COALESCE(ws.provider, 'whatsmeow'),
		  COALESCE(c.ai_agent_id::text, ''),
		  COALESCE(aa.name, ''),
	  c.assigned_to,
	  COALESCE(a.name, ''),
	  COALESCE(c.escalation_reason, ''),
	  c.priority,
	  c.sla_due_at,
	  COALESCE(c.internal_note, ''),
	  c.last_message_at,
	  inbound.last_customer_message_at,
	  COALESCE(deal_stats.open_deals, 0),
	  COALESCE(deal_stats.open_deal_items, '[]'::jsonb)::text
		FROM conversations c
		JOIN contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
		LEFT JOIN whatsapp_sessions ws ON ws.id = c.whatsapp_session_id AND ws.organization_id = c.organization_id
		LEFT JOIN ai_agents aa ON aa.id = c.ai_agent_id AND aa.organization_id = c.organization_id
		LEFT JOIN agents a ON a.id = c.assigned_to
		LEFT JOIN LATERAL (
		  SELECT
		    COUNT(*) FILTER (WHERE d.status = 'open') AS open_deals,
		    COALESCE(
		      jsonb_agg(
		        jsonb_build_object(
		          'id', d.id::text,
		          'title', d.title,
		          'stageName', st.name,
		          'valueAmount', d.value_amount,
		          'ownerName', COALESCE(owner.name, ''),
		          'updatedAt', d.updated_at
		        )
		        ORDER BY d.updated_at DESC
		      ) FILTER (WHERE d.status = 'open'),
		      '[]'::jsonb
		    ) AS open_deal_items
		  FROM deals d
		  JOIN deal_stages st ON st.id = d.stage_id AND st.organization_id = d.organization_id
		  LEFT JOIN agents owner ON owner.id = d.owner_agent_id
		  WHERE d.organization_id = c.organization_id
		    AND (d.conversation_id = c.id OR d.contact_id = ct.id)
		) deal_stats ON TRUE
		LEFT JOIN LATERAL (
		  SELECT MAX(COALESCE(sent_at, created_at)) AS last_customer_message_at
		  FROM messages
		  WHERE conversation_id = c.id
		    AND organization_id = c.organization_id
		    AND direction = 'inbound'
		) inbound ON TRUE
		WHERE c.id = $1 AND c.organization_id = $2
	`, id, s.organizationID(r.Context())).Scan(
		&detail.ID,
		&detail.ContactID,
		&detail.ContactName,
		&detail.Phone,
		&detail.Mode,
		&detail.Status,
		&detail.WhatsAppSessionID,
		&detail.WhatsAppSessionName,
		&detail.WhatsAppProvider,
		&detail.AIAgentID,
		&detail.AIAgentName,
		&detail.AssignedToID,
		&detail.AssignedToName,
		&detail.EscalationReason,
		&detail.Priority,
		&detail.SLADueAt,
		&detail.InternalNote,
		&detail.LastMessageAt,
		&detail.LastCustomerMessageAt,
		&detail.OpenDealCount,
		&openDealsJSON,
	)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}
	detail.OpenDeals = parseJSONValue(openDealsJSON)
	whatsappWindow := newWhatsAppCustomerWindowInfo(detail.LastCustomerMessageAt, time.Now())

	messages := []map[string]any{}
	rows, err := s.db.Query(r.Context(), `
			SELECT sender_type::text, direction::text, content_type::text, COALESCE(text, ''), COALESCE(raw_payload, '{}'::jsonb)::text, COALESCE(sent_at, created_at)
			FROM messages
			WHERE conversation_id = $1 AND organization_id = $2
			ORDER BY COALESCE(sent_at, created_at) ASC, created_at ASC, id ASC
		`, id, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var senderType, direction, contentType, text, rawPayload string
		var createdAt time.Time
		if err := rows.Scan(&senderType, &direction, &contentType, &text, &rawPayload, &createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		parsedRawPayload := map[string]any{}
		_ = json.Unmarshal([]byte(rawPayload), &parsedRawPayload)
		s.applyPresignedMediaURL(r.Context(), parsedRawPayload)
		messages = append(messages, map[string]any{
			"senderType":  senderType,
			"direction":   direction,
			"contentType": contentType,
			"text":        text,
			"rawPayload":  parsedRawPayload,
			"createdAt":   createdAt,
		})
	}

	canReply, composerHint := s.computeComposerPolicy(role, s.agentID(r.Context()), detail.AssignedToID, detail.Mode)
	writeJSON(w, http.StatusOK, map[string]any{
		"conversation": detail,
		"messages":     messages,
		"composer": map[string]any{
			"enabled":        canReply,
			"hint":           composerHint,
			"whatsappWindow": whatsappWindow,
		},
	})
}

func (s *Server) handleTakeover(w http.ResponseWriter, r *http.Request, id string) {
	agentID := s.agentID(r.Context())
	role := s.role(r.Context())
	if !isOpsRole(role) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only operational roles can take over chats"})
		return
	}
	actionCtx, err := s.contextWithConversationWhatsAppSession(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}
	r = r.WithContext(actionCtx)
	if !s.requireWhatsAppConnected(w, r) {
		return
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Note = strings.TrimSpace(req.Note)

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	var assignedTo *string
	if err := tx.QueryRow(r.Context(), `SELECT assigned_to FROM conversations WHERE id = $1 AND organization_id = $2 FOR UPDATE`, id, s.organizationID(r.Context())).Scan(&assignedTo); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}

	if role == "operator" && assignedTo != nil && *assignedTo != agentID {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "operator cannot take over conversations assigned to another agent"})
		return
	}

	eventType := "taken_over_by_human"
	if assignedTo != nil && *assignedTo != agentID && (role == "admin" || role == "super_admin") {
		eventType = "force_taken_over"
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE conversations
		SET mode = 'human',
		    status = 'open',
		    assigned_to = $2,
		    human_taken_over_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $3
	`, id, agentID, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	payload := map[string]any{"forced": eventType == "force_taken_over"}
	if req.Note != "" {
		payload["note"] = req.Note
	}
	if err := s.insertEventTx(r.Context(), tx, id, eventType, agentID, payload); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertAuditLogTx(r.Context(), tx, "conversation."+eventType, "conversation", id, payload); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleForceTakeover(w http.ResponseWriter, r *http.Request, id string) {
	agentID := s.agentID(r.Context())
	role := s.role(r.Context())
	if role != "admin" && role != "super_admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin and super_admin can force takeover chats"})
		return
	}
	actionCtx, err := s.contextWithConversationWhatsAppSession(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}
	r = r.WithContext(actionCtx)
	if !s.requireWhatsAppConnected(w, r) {
		return
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Note = strings.TrimSpace(req.Note)

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	var previousAssignedTo, previousMode, previousStatus string
	if err := tx.QueryRow(r.Context(), `
		SELECT COALESCE(assigned_to::text, ''), mode::text, status::text
		FROM conversations
		WHERE id = $1 AND organization_id = $2
		FOR UPDATE
	`, id, s.organizationID(r.Context())).Scan(&previousAssignedTo, &previousMode, &previousStatus); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE conversations
		SET mode = 'human',
		    status = 'open',
		    assigned_to = $2,
		    human_taken_over_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $3
	`, id, agentID, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	payload := map[string]any{
		"forced":               true,
		"previous_assigned_to": previousAssignedTo,
		"previous_mode":        previousMode,
		"previous_status":      previousStatus,
	}
	if req.Note != "" {
		payload["note"] = req.Note
	}
	if err := s.insertEventTx(r.Context(), tx, id, "force_taken_over", agentID, payload); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertAuditLogTx(r.Context(), tx, "conversation.force_takeover", "conversation", id, payload); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAssign(w http.ResponseWriter, r *http.Request, id string) {
	role := s.role(r.Context())
	if role != "admin" && role != "super_admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin and super_admin can assign chats"})
		return
	}
	if err := s.ensureConversationActionAllowed(r.Context(), id, role, s.agentID(r.Context()), "assign chat"); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}
	actionCtx, err := s.contextWithConversationWhatsAppSession(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}
	r = r.WithContext(actionCtx)
	if !s.requireWhatsAppConnected(w, r) {
		return
	}

	var req struct {
		AgentUsername string `json:"agentUsername"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if strings.TrimSpace(req.AgentUsername) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "agentUsername is required"})
		return
	}

	var assigneeID, assigneeRole string
	err = s.db.QueryRow(r.Context(), `
		SELECT id, role::text FROM agents WHERE username = $1 AND is_active = TRUE
	`, req.AgentUsername).Scan(&assigneeID, &assigneeRole)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "assignee not found"})
		return
	}
	if assigneeRole == "owner" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "owner cannot be assigned to chats"})
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(), `
		UPDATE conversations
		SET assigned_to = $2,
		    mode = 'human',
		    status = 'open',
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $3
	`, id, assigneeID, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertEventTx(r.Context(), tx, id, "assignment_changed", s.agentID(r.Context()), map[string]any{"assigned_to_username": req.AgentUsername}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertAuditLogTx(r.Context(), tx, "conversation.assign", "conversation", id, map[string]any{"assigned_to_username": req.AgentUsername}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReturnToAI(w http.ResponseWriter, r *http.Request, id string) {
	agentID := s.agentID(r.Context())
	role := s.role(r.Context())
	if err := s.ensureConversationActionAllowed(r.Context(), id, role, agentID, "return chat to AI"); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	if _, err := tx.Exec(r.Context(), `
		UPDATE conversations
		SET mode = 'ai',
		    status = 'open',
		    assigned_to = NULL,
		    returned_to_ai_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $2
	`, id, s.organizationID(r.Context())); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertEventTx(r.Context(), tx, id, "returned_to_ai", agentID, map[string]any{"internal_only_summary": true}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertAuditLogTx(r.Context(), tx, "conversation.return_to_ai", "conversation", id, map[string]any{"internal_only_summary": true}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request, id string) {
	agentID := s.agentID(r.Context())
	role := s.role(r.Context())
	if err := s.ensureConversationActionAllowed(r.Context(), id, role, agentID, "resolve chat"); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}
	actionCtx, err := s.contextWithConversationWhatsAppSession(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}
	r = r.WithContext(actionCtx)
	if !s.requireWhatsAppConnected(w, r) {
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	if _, err := tx.Exec(r.Context(), `
		UPDATE conversations
		SET status = 'resolved',
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $2
	`, id, s.organizationID(r.Context())); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertEventTx(r.Context(), tx, id, "conversation_resolved", agentID, map[string]any{"resolved_by": "human"}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertAuditLogTx(r.Context(), tx, "conversation.resolve", "conversation", id, map[string]any{"resolved_by": "human"}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleInternalWAInbound(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMediaJSONBodyBytes)

	var req struct {
		WhatsAppSessionID   string    `json:"whatsappSessionId"`
		WhatsAppSessionKey  string    `json:"sessionKey"`
		Phone               string    `json:"phone"`
		CustomerName        string    `json:"customerName"`
		LegacyCandidateName string    `json:"candidateName"`
		Text                string    `json:"text"`
		ContentType         string    `json:"contentType"`
		ImageBase64         string    `json:"imageBase64"`
		ImageMimeType       string    `json:"imageMimeType"`
		ImageSizeBytes      int       `json:"imageSizeBytes"`
		ExternalMessageID   string    `json:"externalMessageId"`
		ReceivedAt          time.Time `json:"receivedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	req.Phone = strings.TrimSpace(req.Phone)
	req.WhatsAppSessionID = strings.TrimSpace(req.WhatsAppSessionID)
	req.WhatsAppSessionKey = strings.TrimSpace(req.WhatsAppSessionKey)
	if strings.TrimSpace(req.CustomerName) == "" {
		req.CustomerName = req.LegacyCandidateName
	}
	req.CustomerName = strings.TrimSpace(req.CustomerName)
	req.Text = strings.TrimSpace(req.Text)
	req.ContentType = strings.TrimSpace(req.ContentType)
	req.ImageBase64 = strings.TrimSpace(req.ImageBase64)
	req.ImageMimeType = strings.TrimSpace(req.ImageMimeType)
	req.ExternalMessageID = strings.TrimSpace(req.ExternalMessageID)
	if req.ContentType == "" {
		req.ContentType = "text"
	}
	hasImage := req.ImageBase64 != "" && strings.HasPrefix(strings.ToLower(req.ImageMimeType), "image/")
	if hasImage {
		req.ImageMimeType = normalizeMediaMime(req.ImageMimeType)
		if !allowedMessageMediaMime(req.ImageMimeType) || !strings.HasPrefix(req.ImageMimeType, "image/") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported image type"})
			return
		}
		if err := validateBase64PayloadSize(req.ImageBase64, maxInboundImageBytes, "image"); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	if req.Text == "" && hasImage {
		req.Text = "Mohon bantu cek gambar/screenshot ini."
	}
	if req.Phone == "" || (req.Text == "" && !hasImage) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone and text or image are required"})
		return
	}
	whatsAppSessionID, organizationID, err := s.resolveInboundWhatsAppSession(r.Context(), req.WhatsAppSessionID, req.WhatsAppSessionKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid whatsapp session is required"})
		return
	}
	r = r.WithContext(context.WithValue(r.Context(), contextOrganizationID, organizationID))
	r = r.WithContext(context.WithValue(r.Context(), contextWhatsAppSession, whatsAppSessionID))
	aiAgentID, err := s.aiAgentIDForWhatsAppSession(r.Context(), whatsAppSessionID)
	if err != nil || strings.TrimSpace(aiAgentID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "whatsapp session has no ai agent"})
		return
	}
	messageContentType := "text"
	var imageForAI *aiImagePayload
	if hasImage {
		messageContentType = "image"
		imageForAI = &aiImagePayload{
			Base64:   req.ImageBase64,
			MimeType: req.ImageMimeType,
		}
	}
	if req.ReceivedAt.IsZero() {
		req.ReceivedAt = time.Now().UTC()
	}

	allowAutoUpdateContactName, onlyFillNameIfEmpty, err := s.activeContactNamePolicy(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	if req.ExternalMessageID != "" {
		var existingConversationID string
		err = tx.QueryRow(r.Context(), `
			SELECT conversation_id
			FROM messages
			WHERE external_message_id = $1 AND organization_id = $2
		`, req.ExternalMessageID, s.organizationID(r.Context())).Scan(&existingConversationID)
		if err == nil {
			if err := tx.Commit(r.Context()); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"status":         "duplicate",
				"conversationId": existingConversationID,
			})
			return
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	rawInboundPhone := req.Phone
	resolvedPhone, identityPayload, err := s.resolveWhatsAppInboundPhoneTx(r.Context(), tx, rawInboundPhone)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	req.Phone = resolvedPhone

	contactID, currentName, err := s.resolveWhatsAppContactTx(r.Context(), tx, req.Phone, rawInboundPhone, req.CustomerName, allowAutoUpdateContactName, onlyFillNameIfEmpty)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if currentName == "" {
		currentName = "Contact"
	}

	var conversationID, currentMode, currentStatus string
	err = tx.QueryRow(r.Context(), `
		SELECT id, mode::text, status::text
		FROM conversations
		WHERE contact_id = $1 AND organization_id = $2 AND whatsapp_session_id = $3
		FOR UPDATE
	`, contactID, s.organizationID(r.Context()), whatsAppSessionID).Scan(&conversationID, &currentMode, &currentStatus)
	newConversation := false
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(r.Context(), `
				INSERT INTO conversations (contact_id, channel, mode, status, last_message_at, organization_id, whatsapp_session_id, ai_agent_id)
				VALUES ($1, 'whatsapp', 'ai', 'open', $2, $3, $4, $5)
				RETURNING id, mode::text, status::text
			`, contactID, req.ReceivedAt, s.organizationID(r.Context()), whatsAppSessionID, aiAgentID).Scan(&conversationID, &currentMode, &currentStatus)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			newConversation = true
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	var inboundMedia *messageMediaPayload
	if hasImage {
		inboundMedia = &messageMediaPayload{
			Base64:   req.ImageBase64,
			MimeType: req.ImageMimeType,
			FileName: "whatsapp-image",
			Kind:     "image",
			Size:     req.ImageSizeBytes,
		}
		if err := s.storeMessageMedia(r.Context(), conversationID, inboundMedia); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}

	rawPayload := map[string]any{
		"source":            "wa-gateway",
		"phone":             req.Phone,
		"rawPhone":          rawInboundPhone,
		"identityResolver":  identityPayload,
		"externalMessageId": req.ExternalMessageID,
		"whatsappSessionId": whatsAppSessionID,
	}
	if hasImage {
		rawPayload["media"] = mediaRawPayload(inboundMedia)
		rawPayload["image"] = rawPayload["media"]
		rawPayload["forwardedToAI"] = true
	}

	var inboundMessageID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO messages (
		  conversation_id,
		  external_message_id,
		  sender_type,
		  direction,
		  content_type,
		  text,
		  raw_payload,
		  sent_at,
		  delivered_at,
		  organization_id
		)
		VALUES ($1, NULLIF($2, ''), 'customer', 'inbound', $3::message_content_type, $4, $5::jsonb, $6, $6, $7)
		RETURNING id
	`, conversationID, req.ExternalMessageID, messageContentType, req.Text, marshalJSON(rawPayload), req.ReceivedAt, s.organizationID(r.Context())).Scan(&inboundMessageID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	nextMode := currentMode
	nextAssignedTo := "assigned_to"
	if newConversation || currentStatus == "resolved" {
		nextMode = "ai"
		nextAssignedTo = "NULL"
	}
	_, err = tx.Exec(r.Context(), fmt.Sprintf(`
		UPDATE conversations
		SET last_message_id = $2,
		    last_message_at = $3,
		    status = 'open',
		    mode = $4::conversation_mode,
		    assigned_to = %s,
		    escalation_reason = NULL,
		    ai_agent_id = COALESCE(ai_agent_id, NULLIF($6, '')::uuid),
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $5
	`, nextAssignedTo), conversationID, inboundMessageID, req.ReceivedAt, nextMode, s.organizationID(r.Context()), aiAgentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	eventType := "conversation_reopened"
	if newConversation {
		eventType = "conversation_opened"
	}
	if err := s.insertSystemEventTx(r.Context(), tx, conversationID, eventType, map[string]any{
		"source":              "wa-gateway",
		"external_message_id": req.ExternalMessageID,
		"raw_phone":           rawInboundPhone,
		"canonical_phone":     req.Phone,
		"identity_resolver":   identityPayload,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, err := s.maybeCaptureContactMemoryFromInboundTx(r.Context(), tx, contactID, conversationID, inboundMessageID, req.Text); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	response := map[string]any{
		"status":         "received",
		"conversationId": conversationID,
		"messageId":      inboundMessageID,
		"mode":           nextMode,
	}
	groupNotificationText := ""

	if nextMode == "ai" {
		shouldRunAI := true
		estimatedInputTokens := 1100
		estimatedOutputTokens := 220
		if hasImage {
			estimatedInputTokens = 2100
			estimatedOutputTokens = 260
		}
		if ok, needed, err := s.ensureCreditsAvailable(r.Context(), estimatedInputTokens, estimatedOutputTokens); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		} else if !ok {
			if err := s.deliverAICreditAlert(r.Context(), needed); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			fallbackMessage, err := s.activeFallbackWaitingMessage(r.Context())
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			sendResult, err := s.sendWhatsAppText(r.Context(), req.Phone, fallbackMessage)
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
				return
			}
			var outboundMessageID string
			err = tx.QueryRow(r.Context(), `
				INSERT INTO messages (
				  conversation_id,
				  external_message_id,
				  sender_type,
				  direction,
				  content_type,
				  text,
				  sent_at,
				  delivered_at,
				  organization_id
				)
				VALUES ($1, NULLIF($2, ''), 'ai', 'outbound', 'text', $3, NOW(), NOW(), $4)
				RETURNING id
			`, conversationID, sendResult.ExternalMessageID, fallbackMessage, s.organizationID(r.Context())).Scan(&outboundMessageID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := s.insertEscalationSummaryMessageTx(r.Context(), tx, conversationID, currentName, req.Phone, req.Text, "Kredit AI tidak mencukupi. Perlu dibantu tim."); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			_, err = tx.Exec(r.Context(), `
				UPDATE conversations
				SET last_message_id = $2,
				    last_message_at = NOW(),
				    mode = 'human',
				    status = 'pending_human',
				    assigned_to = NULL,
				    escalation_reason = $3,
				    escalated_at = NOW(),
				    updated_at = NOW()
				WHERE id = $1 AND organization_id = $4
			`, conversationID, outboundMessageID, "Insufficient AI credits. Escalated to human.", s.organizationID(r.Context()))
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := s.insertSystemEventTx(r.Context(), tx, conversationID, "escalated", map[string]any{
				"source":          "wa-gateway",
				"quota_exceeded":  true,
				"needed_credits":  needed,
				"escalation_type": "insufficient_credits",
			}); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			response["decision"] = "escalate"
			response["reason"] = "insufficient_ai_credits"
			nextMode = "human"
			noticeText, noticeEnabled, noticeErr := s.formatHumanHandoffGroupNotification(r.Context(), conversationID, req.Phone, currentName, req.Text, "Kredit AI tidak mencukupi. Perlu dibantu tim.")
			if noticeErr != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": noticeErr.Error()})
				return
			}
			if noticeEnabled {
				groupNotificationText = noticeText
			}
			shouldRunAI = false
		}

		if shouldRunAI {
			history, err := s.recentConversationAIHistoryTx(r.Context(), tx, conversationID, inboundMessageID, 10)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			stopWhatsAppTyping := s.startWhatsAppTyping(r.Context(), req.Phone)
			_ = s.setAITyping(r.Context(), conversationID, true)
			typingActive := true
			stopTyping := func() {
				if !typingActive {
					return
				}
				stopWhatsAppTyping()
				_ = s.setAITyping(context.Background(), conversationID, false)
				typingActive = false
			}
			defer stopTyping()
			aiContext := context.WithValue(r.Context(), contextAIAgentID, aiAgentID)
			decision, err := s.callAIWithHistoryAndImage(aiContext, conversationID, req.Text, currentName, history, imageForAI)
			stopTyping()
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
				return
			}
			if err := s.deliverProviderQuotaAlert(r.Context(), decision); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}

			aiSettingsUpdatedAt, _ := s.activeAISettingsUpdatedAt(r.Context())
			aiRunID := ""
			err = tx.QueryRow(r.Context(), `
				INSERT INTO ai_runs (
				  conversation_id,
				  message_id,
				  decision,
				  question_summary,
				  answer_text,
				  confidence_score,
				  escalation_reason,
				  retrieval_metadata,
				  model_name,
				  latency_ms,
				  ai_settings_updated_at_snapshot,
				  organization_id
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12)
				RETURNING id
			`,
				conversationID,
				inboundMessageID,
				decision.Decision,
				req.Text,
				nullableString(decision.AnswerText),
				decision.ConfidenceScore,
				nullableString(decision.EscalationReason),
				marshalJSON(map[string]any{
					"mode":      "live_inbound",
					"retrieval": decision.RetrievalMetadata,
				}),
				decision.ModelName,
				decision.LatencyMS,
				aiSettingsUpdatedAt,
				s.organizationID(r.Context()),
			).Scan(&aiRunID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			response["aiRunId"] = aiRunID
			response["decision"] = decision.Decision
			inputTokens, outputTokens, embeddingTokens, actualCostUSD := usageTokensFromDecision(decision, estimatedInputTokens, estimatedOutputTokens, 0)
			usageResult, err := s.logCreditUsageTx(r.Context(), tx, conversationID, inboundMessageID, aiRunID, "live_inbound", decision.ModelName, inputTokens, outputTokens, embeddingTokens, actualCostUSD)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := s.insertAIRunCostStepsTx(r.Context(), tx, aiRunID, decision, usageResult); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}

			if decision.Decision == "answer" && strings.TrimSpace(decision.AnswerText) != "" {
				sendResult, err := s.sendWhatsAppText(r.Context(), req.Phone, decision.AnswerText)
				if err != nil {
					writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
					return
				}
				var outboundMessageID string
				err = tx.QueryRow(r.Context(), `
					INSERT INTO messages (
					  conversation_id,
					  external_message_id,
					  sender_type,
					  direction,
					  content_type,
					  text,
					  sent_at,
					  delivered_at,
					  organization_id
					)
					VALUES ($1, NULLIF($2, ''), 'ai', 'outbound', 'text', $3, NOW(), NOW(), $4)
					RETURNING id
				`, conversationID, sendResult.ExternalMessageID, decision.AnswerText, s.organizationID(r.Context())).Scan(&outboundMessageID)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				nextStatus := "open"
				autoResolved := shouldAutoResolveAIConversation(decision)
				if autoResolved {
					nextStatus = "resolved"
				}
				_, err = tx.Exec(r.Context(), `
					UPDATE conversations
					SET last_message_id = $2,
					    last_message_at = NOW(),
					    mode = 'ai',
					    status = $3::conversation_status,
					    escalation_reason = NULL,
					    updated_at = NOW()
					WHERE id = $1 AND organization_id = $4
				`, conversationID, outboundMessageID, nextStatus, s.organizationID(r.Context()))
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				if err := s.insertSystemEventTx(r.Context(), tx, conversationID, "ai_replied", map[string]any{
					"source":    "wa-gateway",
					"ai_run_id": aiRunID,
				}); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				if autoResolved {
					if err := s.insertSystemEventTx(r.Context(), tx, conversationID, "conversation_resolved", map[string]any{
						"resolved_by": "ai",
						"reason":      "conversation_end",
						"ai_run_id":   aiRunID,
					}); err != nil {
						writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
						return
					}
				}
			} else {
				fallbackMessage, err := s.activeFallbackWaitingMessage(r.Context())
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				sendResult, err := s.sendWhatsAppText(r.Context(), req.Phone, fallbackMessage)
				if err != nil {
					writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
					return
				}
				var outboundMessageID string
				err = tx.QueryRow(r.Context(), `
					INSERT INTO messages (
					  conversation_id,
					  external_message_id,
					  sender_type,
					  direction,
					  content_type,
					  text,
					  sent_at,
					  delivered_at,
					  organization_id
					)
					VALUES ($1, NULLIF($2, ''), 'ai', 'outbound', 'text', $3, NOW(), NOW(), $4)
					RETURNING id
				`, conversationID, sendResult.ExternalMessageID, fallbackMessage, s.organizationID(r.Context())).Scan(&outboundMessageID)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				if err := s.insertEscalationSummaryMessageTx(r.Context(), tx, conversationID, currentName, req.Phone, req.Text, decision.EscalationReason); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				_, err = tx.Exec(r.Context(), `
					UPDATE conversations
					SET last_message_id = $2,
					    last_message_at = NOW(),
					    mode = 'human',
					    status = 'pending_human',
					    assigned_to = NULL,
					    escalation_reason = $3,
					    escalated_at = NOW(),
					    updated_at = NOW()
					WHERE id = $1 AND organization_id = $4
				`, conversationID, outboundMessageID, nullableString(decision.EscalationReason), s.organizationID(r.Context()))
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				if err := s.insertSystemEventTx(r.Context(), tx, conversationID, "escalated", map[string]any{
					"source":            "wa-gateway",
					"ai_run_id":         aiRunID,
					"escalation_reason": decision.EscalationReason,
				}); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				noticeText, noticeEnabled, noticeErr := s.formatHumanHandoffGroupNotification(r.Context(), conversationID, req.Phone, currentName, req.Text, decision.EscalationReason)
				if noticeErr != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": noticeErr.Error()})
					return
				}
				if noticeEnabled {
					groupNotificationText = noticeText
				}
			}
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if groupNotificationText != "" {
		_ = s.notifyNotificationGroup(r.Context(), groupNotificationText)
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleManualMessage(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role == "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "owner cannot send chat messages"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMediaJSONBodyBytes)

	var req struct {
		Text        string `json:"text"`
		Phone       string `json:"phone"`
		MediaBase64 string `json:"mediaBase64"`
		MediaMime   string `json:"mediaMime"`
		MediaName   string `json:"mediaName"`
		MediaKind   string `json:"mediaKind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	req.Text = strings.TrimSpace(req.Text)
	req.MediaBase64 = strings.TrimSpace(req.MediaBase64)
	req.MediaMime = normalizeMediaMime(req.MediaMime)
	req.MediaName = strings.TrimSpace(req.MediaName)
	req.MediaKind = normalizeMessageMediaKind(req.MediaKind, req.MediaMime)
	hasMedia := req.MediaBase64 != "" && req.MediaMime != ""
	if req.Text == "" && !hasMedia {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text or media is required"})
		return
	}
	if hasMedia {
		if !allowedMessageMediaMime(req.MediaMime) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported media type"})
			return
		}
		if err := validateBase64PayloadSize(req.MediaBase64, maxManualMediaBytes, "media"); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}

	agentID := s.agentID(r.Context())
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	var assignedTo *string
	var mode, phone, whatsAppSessionID, whatsappProvider string
	var lastCustomerMessageAt *time.Time
	err = tx.QueryRow(r.Context(), `
		SELECT
		  c.assigned_to,
		  c.mode::text,
		  ct.phone,
		  COALESCE(c.whatsapp_session_id::text, ''),
		  COALESCE(ws.provider, 'whatsmeow'),
		  inbound.last_customer_message_at
		FROM conversations c
		JOIN contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
		LEFT JOIN whatsapp_sessions ws ON ws.id = c.whatsapp_session_id AND ws.organization_id = c.organization_id
		LEFT JOIN LATERAL (
		  SELECT MAX(COALESCE(sent_at, created_at)) AS last_customer_message_at
		  FROM messages
		  WHERE conversation_id = c.id
		    AND organization_id = c.organization_id
		    AND direction = 'inbound'
		) inbound ON TRUE
		WHERE c.id = $1 AND c.organization_id = $2
		FOR UPDATE
	`, id, s.organizationID(r.Context())).Scan(&assignedTo, &mode, &phone, &whatsAppSessionID, &whatsappProvider, &lastCustomerMessageAt)
	if err != nil && req.Phone != "" {
		// Inbound webhooks can create a fresh conversation for the same contact
		// while an Inbox tab still holds the previous conversation id.
		err = tx.QueryRow(r.Context(), `
			SELECT
			  c.assigned_to,
			  c.mode::text,
			  ct.phone,
			  COALESCE(c.whatsapp_session_id::text, ''),
			  COALESCE(ws.provider, 'whatsmeow'),
			  inbound.last_customer_message_at
			FROM conversations c
			JOIN contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
			LEFT JOIN whatsapp_sessions ws ON ws.id = c.whatsapp_session_id AND ws.organization_id = c.organization_id
			LEFT JOIN LATERAL (
			  SELECT MAX(COALESCE(sent_at, created_at)) AS last_customer_message_at
			  FROM messages
			  WHERE conversation_id = c.id AND organization_id = c.organization_id AND direction = 'inbound'
			) inbound ON TRUE
			WHERE c.organization_id = $1
			  AND regexp_replace(ct.phone, '[^0-9]', '', 'g') = regexp_replace($2, '[^0-9]', '', 'g')
			ORDER BY c.last_message_at DESC
			LIMIT 1
			FOR UPDATE OF c
		`, s.organizationID(r.Context()), req.Phone).Scan(&assignedTo, &mode, &phone, &whatsAppSessionID, &whatsappProvider, &lastCustomerMessageAt)
		if err == nil {
			var fallbackID string
			if scanErr := tx.QueryRow(r.Context(), `
				SELECT c.id::text
				FROM conversations c
				JOIN contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
				WHERE c.organization_id = $1
				  AND regexp_replace(ct.phone, '[^0-9]', '', 'g') = regexp_replace($2, '[^0-9]', '', 'g')
				ORDER BY c.last_message_at DESC LIMIT 1
			`, s.organizationID(r.Context()), req.Phone).Scan(&fallbackID); scanErr == nil {
				id = fallbackID
			}
		}
	}
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}

	canReply, hint := s.computeComposerPolicy(role, agentID, assignedTo, mode)
	if !canReply {
		writeJSON(w, http.StatusConflict, map[string]string{"error": hint})
		return
	}
	if strings.EqualFold(whatsappProvider, "meta_cloud") && !whatsappCustomerWindowAllowsFreeform(lastCustomerMessageAt, time.Now()) {
		window := newWhatsAppCustomerWindowInfo(lastCustomerMessageAt, time.Now())
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":          "WhatsApp customer service window is closed. Use an approved Meta message template before sending free-form follow-up.",
			"code":           "WA_TEMPLATE_REQUIRED",
			"whatsappWindow": window,
		})
		return
	}

	requestContext := context.WithValue(r.Context(), contextWhatsAppSession, whatsAppSessionID)
	if !s.requireWhatsAppConnected(w, r.WithContext(requestContext)) {
		return
	}
	var mediaPayload *messageMediaPayload
	var sendResult waSendResult
	if hasMedia {
		mediaPayload = &messageMediaPayload{
			Base64:   req.MediaBase64,
			MimeType: req.MediaMime,
			FileName: req.MediaName,
			Kind:     req.MediaKind,
		}
		if err := s.storeMessageMedia(requestContext, id, mediaPayload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		sendResult, err = s.sendWhatsAppMedia(requestContext, phone, req.Text, *mediaPayload)
	} else {
		sendResult, err = s.sendWhatsAppText(requestContext, phone, req.Text)
	}
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	externalID := sendResult.ExternalMessageID
	if externalID == "" {
		externalID = fmt.Sprintf("manual-%d", time.Now().UnixNano())
	}
	messageContentType := "text"
	rawPayload := map[string]any{
		"source": "dashboard",
	}
	if hasMedia {
		messageContentType = normalizeMessageMediaKind(mediaPayload.Kind, mediaPayload.MimeType)
		rawPayload["media"] = mediaRawPayload(mediaPayload)
	}
	var messageID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO messages (
		  conversation_id,
		  external_message_id,
		  sender_type,
		  direction,
		  content_type,
		  text,
		  sender_agent_id,
		  raw_payload,
		  sent_at,
		  organization_id
		)
		VALUES ($1, $2, 'agent', 'outbound', $3::message_content_type, $4, $5, $6::jsonb, NOW(), $7)
		RETURNING id
	`, id, externalID, messageContentType, req.Text, agentID, marshalJSON(rawPayload), s.organizationID(r.Context())).Scan(&messageID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE conversations
		SET last_message_id = $2,
		    last_message_at = NOW(),
		    mode = 'human',
		    status = 'open',
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $3
	`, id, messageID, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := s.insertEventTx(r.Context(), tx, id, "manual_message_sent", agentID, map[string]any{"message_id": messageID}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		payload, err := s.wsSnapshot(r.Context())
		if err != nil {
			_ = conn.WriteJSON(map[string]any{"error": err.Error()})
			return
		}
		if err := conn.WriteJSON(payload); err != nil {
			return
		}
		<-ticker.C
	}
}

func (s *Server) wsSnapshot(ctx context.Context) (map[string]any, error) {
	var status string
	var details []byte
	err := s.db.QueryRow(ctx, `
		SELECT status, COALESCE(details, '{}'::jsonb)::text
		FROM system_status
		WHERE service_name = 'wa-gateway'
	`).Scan(&status, &details)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	var openCount, pendingCount int
	if err := s.db.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE status = 'open'),
		  COUNT(*) FILTER (WHERE status = 'pending_human')
		FROM conversations
	`).Scan(&openCount, &pendingCount); err != nil {
		return nil, err
	}

	services := []map[string]any{}
	serviceRows, err := s.db.Query(ctx, `
		SELECT service_name, status, COALESCE(details, '{}'::jsonb)::text, updated_at
		FROM system_status
		ORDER BY service_name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer serviceRows.Close()
	for serviceRows.Next() {
		var serviceName, serviceStatus, serviceDetails string
		var updatedAt time.Time
		if err := serviceRows.Scan(&serviceName, &serviceStatus, &serviceDetails, &updatedAt); err != nil {
			return nil, err
		}
		services = append(services, map[string]any{
			"serviceName": serviceName,
			"status":      serviceStatus,
			"details":     json.RawMessage(serviceDetails),
			"updatedAt":   updatedAt,
		})
	}

	recentEvents := []map[string]any{}
	eventRows, err := s.db.Query(ctx, `
		SELECT event_type::text, created_at
		FROM conversation_events
		ORDER BY created_at DESC
		LIMIT 5
	`)
	if err != nil {
		return nil, err
	}
	defer eventRows.Close()
	for eventRows.Next() {
		var eventType string
		var createdAt time.Time
		if err := eventRows.Scan(&eventType, &createdAt); err != nil {
			return nil, err
		}
		recentEvents = append(recentEvents, map[string]any{
			"eventType": eventType,
			"createdAt": createdAt,
		})
	}

	wallet := s.ownerWallet(ctx)
	systemAlerts, err := s.currentOpenSystemAlerts(ctx)
	if err != nil {
		return nil, err
	}
	aiTyping := map[string]any{"active": false}
	var aiTypingDetails []byte
	err = s.db.QueryRow(ctx, `
		SELECT COALESCE(details, '{}'::jsonb)::text
		FROM system_status
		WHERE service_name = 'ai-typing'
		  AND status = 'active'
		  AND COALESCE((details->>'expiresAt')::timestamptz, updated_at + interval '2 minutes') > NOW()
		LIMIT 1
	`).Scan(&aiTypingDetails)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil && len(aiTypingDetails) > 0 {
		aiTyping = map[string]any{
			"active":  true,
			"details": json.RawMessage(aiTypingDetails),
		}
	}

	return map[string]any{
		"type": "system_snapshot",
		"wa": map[string]any{
			"status":  status,
			"details": json.RawMessage(details),
		},
		"openConversations":    openCount,
		"pendingConversations": pendingCount,
		"services":             services,
		"systemAlerts":         systemAlerts,
		"aiTyping":             aiTyping,
		"wallet":               wallet,
		"recentEvents":         recentEvents,
		"timestamp":            time.Now().UTC(),
	}, nil
}

func (s *Server) insertEvent(ctx context.Context, conversationID, eventType, agentID string, payload map[string]any) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := s.insertEventTx(ctx, tx, conversationID, eventType, agentID, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) insertEventTx(ctx context.Context, tx pgx.Tx, conversationID, eventType, agentID string, payload map[string]any) error {
	return s.insertEventTxWithActor(ctx, tx, conversationID, eventType, "agent", nullableString(agentID), payload)
}

func (s *Server) insertSystemEventTx(ctx context.Context, tx pgx.Tx, conversationID, eventType string, payload map[string]any) error {
	return s.insertEventTxWithActor(ctx, tx, conversationID, eventType, "system", nil, payload)
}

func (s *Server) insertEventTxWithActor(ctx context.Context, tx pgx.Tx, conversationID, eventType, actorType string, actorID any, payload map[string]any) error {
	encoded, _ := json.Marshal(payload)
	_, err := tx.Exec(ctx, `
		INSERT INTO conversation_events (conversation_id, event_type, actor_type, actor_id, payload, organization_id)
		VALUES ($1, $2, $3, $4, $5::jsonb, NULLIF($6, '')::uuid)
	`, conversationID, eventType, actorType, actorID, string(encoded), s.organizationID(ctx))
	return err
}

func (s *Server) setAITyping(ctx context.Context, conversationID string, active bool) error {
	status := "idle"
	details := map[string]any{
		"active": false,
	}
	if active {
		now := time.Now().UTC()
		status = "active"
		details = map[string]any{
			"active":         true,
			"conversationId": conversationID,
			"startedAt":      now,
			"expiresAt":      now.Add(2 * time.Minute),
		}
	}
	encoded, _ := json.Marshal(details)
	_, err := s.db.Exec(ctx, `
		INSERT INTO system_status (service_name, status, details, updated_at)
		VALUES ('ai-typing', $1, $2::jsonb, NOW())
		ON CONFLICT (service_name) DO UPDATE SET
		  status = EXCLUDED.status,
		  details = EXCLUDED.details,
		  updated_at = NOW()
	`, status, string(encoded))
	return err
}

func (s *Server) role(ctx context.Context) string {
	role, _ := ctx.Value(contextRole).(string)
	return canonicalAgentRole(role)
}

func (s *Server) organizationID(ctx context.Context) string {
	return organizationIDFromContext(ctx)
}

func (s *Server) organizationRole(ctx context.Context) string {
	role, _ := ctx.Value(contextOrganizationRole).(string)
	return canonicalOrganizationRole(role)
}

func canonicalAgentRole(role string) string {
	if strings.TrimSpace(role) == legacyOperatorRole {
		return operatorRole
	}
	return strings.TrimSpace(role)
}

func canonicalOrganizationRole(role string) string {
	switch strings.TrimSpace(role) {
	case legacyOperatorRole:
		return operatorRole
	case legacySupervisorRole:
		return supervisorRole
	default:
		return strings.TrimSpace(role)
	}
}

func organizationIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextOrganizationID).(string)
	return id
}

func whatsAppSessionIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextWhatsAppSession).(string)
	return id
}

func aiAgentIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextAIAgentID).(string)
	return strings.TrimSpace(id)
}

func (s *Server) waGatewayPath(ctx context.Context, action string) string {
	action = strings.Trim(action, "/")
	sessionID := strings.TrimSpace(whatsAppSessionIDFromContext(ctx))
	if sessionID == "" {
		return "/api/" + action
	}
	return "/api/sessions/" + sessionID + "/" + action
}

func (s *Server) resolveAuthorizedOrganization(ctx context.Context, agentID, requestedOrganizationID string) (string, string, error) {
	if strings.TrimSpace(agentID) == "" {
		return "", "", errors.New("missing agent id")
	}

	row := s.db.QueryRow(ctx, `
		SELECT om.organization_id::text, om.role
		FROM organization_members om
		JOIN organizations o ON o.id = om.organization_id
		LEFT JOIN agents a ON a.id = om.agent_id
		WHERE om.agent_id = $1
		  AND om.status = 'active'
		  AND o.status = 'active'
		  AND (NULLIF($2, '')::uuid IS NULL OR om.organization_id = NULLIF($2, '')::uuid)
		ORDER BY
		  CASE WHEN om.organization_id = a.current_organization_id THEN 0 ELSE 1 END,
		  om.joined_at DESC
		LIMIT 1
	`, agentID, strings.TrimSpace(requestedOrganizationID))

	var organizationID, organizationRole string
	if err := row.Scan(&organizationID, &organizationRole); err != nil {
		return "", "", err
	}
	return organizationID, organizationRole, nil
}

func (s *Server) defaultOrganizationID(ctx context.Context) (string, error) {
	var organizationID string
	err := s.db.QueryRow(ctx, `
		SELECT id::text
		FROM organizations
		WHERE slug = 'default-company'
		ORDER BY created_at ASC
		LIMIT 1
	`).Scan(&organizationID)
	return organizationID, err
}

func isOpsRole(role string) bool {
	return role == "admin" || role == "super_admin" || role == "operator"
}

func isAssignableAgentRole(role string) bool {
	return role == "super_admin" || role == "admin" || role == "operator"
}

func agentRoleToOrganizationRole(role string) string {
	switch role {
	case "owner":
		return "org_owner"
	case "super_admin", "admin":
		return "org_admin"
	default:
		return "operator"
	}
}

func (s *Server) agentID(ctx context.Context) string {
	id, _ := ctx.Value(contextAgentID).(string)
	return id
}

func (s *Server) accountNotifications(ctx context.Context, role, agentID string) ([]map[string]any, error) {
	items := []map[string]any{}
	add := func(id, kind, severity, title, message, actionView string, count int) {
		items = append(items, map[string]any{
			"id":         id,
			"kind":       kind,
			"severity":   severity,
			"title":      title,
			"message":    message,
			"actionView": actionView,
			"count":      count,
			"createdAt":  time.Now().UTC(),
		})
	}

	var waStatus string
	err := s.db.QueryRow(ctx, `
		SELECT status
		FROM system_status
		WHERE service_name = 'wa-gateway'
	`).Scan(&waStatus)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if waStatus != "" && waStatus != "connected" {
		add("wa-disconnected", "whatsapp", "critical", "WhatsApp disconnected", "Hubungkan ulang WhatsApp supaya inbox dan eskalasi tetap jalan.", "whatsapp", 1)
	}

	var monthlyRemaining, additionalRemaining int
	err = s.db.QueryRow(ctx, `
		SELECT monthly_credits_remaining, additional_credits_remaining
		FROM credit_wallet
		WHERE is_active = TRUE AND organization_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&monthlyRemaining, &additionalRemaining)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	totalRemaining := monthlyRemaining + additionalRemaining
	if err == nil && totalRemaining <= 100 {
		if role == "owner" {
			add("credits-low", "billing", "critical", "Credit almost empty", fmt.Sprintf("Sisa %d credit. Top up atau confirm request agar AI tidak berhenti menjawab.", totalRemaining), "wallet", totalRemaining)
		} else {
			add("credits-low", "billing", "warning", "Credit hampir habis", fmt.Sprintf("Sisa %d credit. Ajukan top up ke owner dari Wallet & Requests.", totalRemaining), "wallet", totalRemaining)
		}
	}

	if role == "owner" {
		var pendingPurchases int
		if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM credit_purchases WHERE payment_status = 'requested' AND organization_id = $1`, s.organizationID(ctx)).Scan(&pendingPurchases); err != nil {
			return nil, err
		}
		if pendingPurchases > 0 {
			add("pending-purchases", "billing", "warning", "Top up request waiting", fmt.Sprintf("%d request pembelian credit menunggu konfirmasi owner.", pendingPurchases), "wallet", pendingPurchases)
		}
	}

	if role != "owner" {
		var pendingHuman int
		if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM conversations WHERE status = 'pending_human' AND organization_id = $1`, s.organizationID(ctx)).Scan(&pendingHuman); err != nil {
			return nil, err
		}
		if pendingHuman > 0 {
			add("pending-human", "inbox", "warning", "Chat needs human", fmt.Sprintf("%d percakapan menunggu takeover atau assignment.", pendingHuman), "operations", pendingHuman)
		}

		var assignedToMe int
		if err := s.db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM conversations
			WHERE assigned_to = $1
			  AND status <> 'resolved'
			  AND organization_id = $2
		`, agentID, s.organizationID(ctx)).Scan(&assignedToMe); err != nil {
			return nil, err
		}
		if assignedToMe > 0 {
			add("assigned-to-me", "assignment", "info", "Assigned to you", fmt.Sprintf("%d percakapan aktif sedang assigned ke akun ini.", assignedToMe), "operations", assignedToMe)
		}
	}

	alertRows, err := s.db.Query(ctx, `
		SELECT id, severity, source, message
		FROM system_alerts
		WHERE status = 'open'
		  AND organization_id = $1
		ORDER BY last_seen_at DESC
		LIMIT 5
	`, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer alertRows.Close()
	for alertRows.Next() {
		var id, severity, source, message string
		if err := alertRows.Scan(&id, &severity, &source, &message); err != nil {
			return nil, err
		}
		actionView := "health"
		if role == "operator" {
			actionView = "whatsapp"
		}
		title := "System alert"
		if strings.Contains(strings.ToLower(message), "knowledge") || strings.Contains(strings.ToLower(message), "embedding") {
			title = "Knowledge needs indexing"
			if role != "owner" {
				actionView = "knowledge"
			}
		}
		add("system-alert-"+id, source, severity, title, message, actionView, 1)
	}

	sort.SliceStable(items, func(i, j int) bool {
		weight := map[string]int{"critical": 0, "warning": 1, "info": 2}
		return weight[fmt.Sprint(items[i]["severity"])] < weight[fmt.Sprint(items[j]["severity"])]
	})
	return items, nil
}

func emptyFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func (s *Server) ownerWallet(ctx context.Context) map[string]any {
	var monthlyLimit, monthlyRemaining, additionalRemaining int
	err := s.db.QueryRow(ctx, `
		SELECT monthly_credit_limit, monthly_credits_remaining, additional_credits_remaining
		FROM credit_wallet
		WHERE organization_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&monthlyLimit, &monthlyRemaining, &additionalRemaining)
	if err != nil {
		return nil
	}
	return map[string]any{
		"monthlyLimit":        monthlyLimit,
		"monthlyRemaining":    monthlyRemaining,
		"additionalRemaining": additionalRemaining,
	}
}

func (s *Server) computeComposerPolicy(role, agentID string, assignedTo *string, mode string) (bool, string) {
	if role == "owner" {
		return false, "owner cannot access client chat content"
	}
	if mode != "human" {
		return false, "take over the conversation before replying manually"
	}
	if assignedTo == nil || *assignedTo == "" {
		return false, "take over or assign the conversation before replying manually"
	}
	if *assignedTo == agentID {
		return true, ""
	}
	if role == "admin" || role == "super_admin" {
		return false, "force takeover or reassign the conversation to yourself before replying manually"
	}
	return false, "chat is assigned to another agent, composer must stay disabled"
}

func (s *Server) ensureConversationActionAllowed(ctx context.Context, conversationID, role, agentID, actionLabel string) error {
	if role == "owner" {
		return fmt.Errorf("owner cannot %s", actionLabel)
	}
	if role == "admin" || role == "super_admin" {
		var exists bool
		err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM conversations WHERE id = $1 AND organization_id = $2)`, conversationID, s.organizationID(ctx)).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("conversation not found")
		}
		return nil
	}
	if role != "operator" {
		return fmt.Errorf("only operational roles can %s", actionLabel)
	}

	var assignedTo *string
	var mode string
	err := s.db.QueryRow(ctx, `
		SELECT assigned_to, mode::text
		FROM conversations
		WHERE id = $1 AND organization_id = $2
	`, conversationID, s.organizationID(ctx)).Scan(&assignedTo, &mode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("conversation not found")
		}
		return err
	}
	if mode != "human" {
		return fmt.Errorf("operator must take over the conversation before %s", actionLabel)
	}
	if assignedTo == nil || *assignedTo != agentID {
		return fmt.Errorf("operator can only %s when assigned to self", actionLabel)
	}
	return nil
}

func (s *Server) ensureWhatsAppConnected(ctx context.Context) (bool, error) {
	sessionID := strings.TrimSpace(whatsAppSessionIDFromContext(ctx))
	path := "/api/status"
	if sessionID != "" {
		path = "/api/sessions/" + sessionID + "/status"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.WAGatewayBaseURL+path, nil)
	if err != nil {
		return false, err
	}
	if token := strings.TrimSpace(s.cfg.InternalGatewayToken); token != "" {
		req.Header.Set("X-Internal-Token", token)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return false, fmt.Errorf("wa-gateway status returned %d", resp.StatusCode)
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, err
	}
	return payload.Status == "connected", nil
}

func (s *Server) requireWhatsAppConnected(w http.ResponseWriter, r *http.Request) bool {
	ok, err := s.ensureWhatsAppConnected(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return false
	}
	if !ok {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "whatsapp is disconnected"})
		return false
	}
	return true
}

func (s *Server) contextWithConversationWhatsAppSession(ctx context.Context, conversationID string) (context.Context, error) {
	var sessionID string
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(whatsapp_session_id::text, '')
		FROM conversations
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, strings.TrimSpace(conversationID), s.organizationID(ctx)).Scan(&sessionID)
	if err != nil {
		return ctx, err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ctx, nil
	}
	return context.WithValue(ctx, contextWhatsAppSession, sessionID), nil
}

type waSendResult struct {
	Status            string `json:"status"`
	ExternalMessageID string `json:"externalMessageId"`
	MediaKind         string `json:"mediaKind"`
}

func (s *Server) sendWhatsAppText(ctx context.Context, phone, text string) (waSendResult, error) {
	body, _ := json.Marshal(map[string]any{
		"phone": phone,
		"text":  text,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.WAGatewayBaseURL+s.waGatewayPath(ctx, "send-text"), bytes.NewReader(body))
	if err != nil {
		return waSendResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", s.cfg.InternalGatewayToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return waSendResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return waSendResult{}, fmt.Errorf("wa-gateway returned %d", resp.StatusCode)
	}

	var payload waSendResult
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return waSendResult{}, err
	}
	return payload, nil
}

func (s *Server) sendWhatsAppMedia(ctx context.Context, phone, text string, media messageMediaPayload) (waSendResult, error) {
	body, _ := json.Marshal(map[string]any{
		"phone":       phone,
		"text":        text,
		"mediaBase64": media.Base64,
		"mediaMime":   media.MimeType,
		"mediaName":   media.FileName,
		"mediaKind":   normalizeMessageMediaKind(media.Kind, media.MimeType),
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.WAGatewayBaseURL+s.waGatewayPath(ctx, "send-media"), bytes.NewReader(body))
	if err != nil {
		return waSendResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", s.cfg.InternalGatewayToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return waSendResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return waSendResult{}, fmt.Errorf("wa-gateway returned %d", resp.StatusCode)
	}

	var payload waSendResult
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return waSendResult{}, err
	}
	return payload, nil
}

func (s *Server) startWhatsAppTyping(ctx context.Context, phone string) func() {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return func() {}
	}
	baseCtx := context.Background()
	if ctx != nil {
		if sessionID := strings.TrimSpace(whatsAppSessionIDFromContext(ctx)); sessionID != "" {
			baseCtx = context.WithValue(baseCtx, contextWhatsAppSession, sessionID)
		}
	}
	done := make(chan struct{})
	go func() {
		send := func(active bool) {
			ctx, cancel := context.WithTimeout(baseCtx, 3*time.Second)
			defer cancel()
			_ = s.sendWhatsAppPresence(ctx, phone, active)
		}
		send(true)
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				send(true)
			case <-done:
				send(false)
				return
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
		})
	}
}

func (s *Server) sendWhatsAppPresence(ctx context.Context, phone string, active bool) error {
	body, _ := json.Marshal(map[string]any{
		"phone":  phone,
		"active": active,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.WAGatewayBaseURL+s.waGatewayPath(ctx, "chat-presence"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", s.cfg.InternalGatewayToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("wa-gateway returned %d", resp.StatusCode)
	}
	return nil
}

func (s *Server) storeMessageMedia(ctx context.Context, conversationID string, media *messageMediaPayload) error {
	if media == nil || strings.TrimSpace(media.Base64) == "" {
		return nil
	}
	data, err := decodeBase64Payload(media.Base64)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("media payload is empty")
	}
	if len(data) > maxManualMediaBytes {
		return fmt.Errorf("media payload is too large")
	}
	media.Size = len(data)
	media.MimeType = normalizeMediaMime(media.MimeType)
	if !allowedMessageMediaMime(media.MimeType) {
		return fmt.Errorf("unsupported media type")
	}
	media.Kind = normalizeMessageMediaKind(media.Kind, media.MimeType)
	if media.FileName = cleanMediaFileName(media.FileName, media.MimeType, media.Kind); media.FileName == "" {
		media.FileName = defaultMessageMediaName(media.MimeType, media.Kind)
	}
	if s.storage == nil {
		return nil
	}
	hash := sha256.Sum256(append([]byte(fmt.Sprintf("%s:%d:", conversationID, time.Now().UnixNano())), data...))
	objectKey := fmt.Sprintf("organizations/%s/conversation-media/%s/%s-%s", s.organizationID(ctx), conversationID, hex.EncodeToString(hash[:8]), media.FileName)
	url, err := s.storage.UploadFile(ctx, objectKey, data, media.MimeType)
	if err != nil {
		return err
	}
	media.URL = url
	media.Key = objectKey
	return nil
}

func mediaRawPayload(media *messageMediaPayload) map[string]any {
	if media == nil {
		return nil
	}
	payload := map[string]any{
		"kind":      normalizeMessageMediaKind(media.Kind, media.MimeType),
		"mimeType":  media.MimeType,
		"fileName":  media.FileName,
		"sizeBytes": media.Size,
	}
	if media.URL != "" {
		payload["url"] = media.URL
	}
	if media.Key != "" {
		payload["objectKey"] = media.Key
	}
	return payload
}

func (s *Server) applyPresignedMediaURL(ctx context.Context, rawPayload map[string]any) {
	if s.storage == nil || rawPayload == nil {
		return
	}
	media, ok := rawPayload["media"].(map[string]any)
	if !ok {
		media, ok = rawPayload["image"].(map[string]any)
		if !ok {
			return
		}
	}
	objectKey, _ := media["objectKey"].(string)
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return
	}
	url, err := s.storage.PresignedURL(ctx, objectKey, time.Hour)
	if err != nil {
		return
	}
	media["url"] = url
	if image, ok := rawPayload["image"].(map[string]any); ok {
		image["url"] = url
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

func validateBase64PayloadSize(value string, maxBytes int, label string) error {
	data, err := decodeBase64Payload(value)
	if err != nil {
		return fmt.Errorf("%s payload is invalid", label)
	}
	if len(data) == 0 {
		return fmt.Errorf("%s payload is empty", label)
	}
	if len(data) > maxBytes {
		return fmt.Errorf("%s payload is too large", label)
	}
	return nil
}

func normalizeMediaMime(mimeType string) string {
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

func allowedMessageMediaMime(mimeType string) bool {
	mimeType = normalizeMediaMime(mimeType)
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

func normalizeMessageMediaKind(kind, mimeType string) string {
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

func cleanMediaFileName(filename, mimeType, kind string) string {
	filename = strings.TrimSpace(filepath.Base(filename))
	if filename == "." {
		filename = ""
	}
	if filename == "" {
		return defaultMessageMediaName(mimeType, kind)
	}
	cleaned := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '-'
		}
		return r
	}, filename)
	if filepath.Ext(cleaned) == "" {
		cleaned += mediaExtension(mimeType, kind)
	}
	return cleaned
}

func defaultMessageMediaName(mimeType, kind string) string {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if kind == "image" {
		if strings.Contains(mimeType, "png") {
			return "image.png"
		}
		if strings.Contains(mimeType, "webp") {
			return "image.webp"
		}
		return "image.jpg"
	}
	if strings.Contains(mimeType, "pdf") {
		return "document.pdf"
	}
	if strings.Contains(mimeType, "word") {
		return "document.docx"
	}
	return "attachment.bin"
}

func mediaExtension(mimeType, kind string) string {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if kind == "image" {
		if strings.Contains(mimeType, "png") {
			return ".png"
		}
		if strings.Contains(mimeType, "webp") {
			return ".webp"
		}
		return ".jpg"
	}
	if strings.Contains(mimeType, "pdf") {
		return ".pdf"
	}
	if strings.Contains(mimeType, "word") {
		return ".docx"
	}
	if strings.Contains(mimeType, "text/plain") {
		return ".txt"
	}
	return ".bin"
}

type groupNotificationTriggerDefinition struct {
	Key             string
	PluginKey       string
	Label           string
	Description     string
	DefaultTemplate string
	DefaultEnabled  bool
}

type aiAnswerCurationNotificationRequest struct {
	AIAgentName  string   `json:"aiAgentName"`
	CustomerName string   `json:"customerName"`
	Question     string   `json:"question"`
	AnswerText   string   `json:"answerText"`
	Decision     string   `json:"decision"`
	Tone         string   `json:"tone"`
	StatusLabel  string   `json:"statusLabel"`
	Issues       []string `json:"issues"`
	Passes       []string `json:"passes"`
	NextAction   string   `json:"nextAction"`
	Confidence   float64  `json:"confidence"`
	CreditsUsed  any      `json:"creditsUsed"`
}

func groupNotificationTriggerDefinitions() []groupNotificationTriggerDefinition {
	return []groupNotificationTriggerDefinition{
		{
			Key:             groupNotificationTriggerHumanHandoff,
			PluginKey:       "ai_cs",
			Label:           "AI butuh bantuan",
			Description:     "Dikirim saat AI mengalihkan chat ke tim manusia.",
			DefaultTemplate: defaultHumanHandoffGroupTemplate,
			DefaultEnabled:  true,
		},
		{
			Key:             groupNotificationTriggerAIAnswerCuration,
			PluginKey:       "ai_cs",
			Label:           "Kurasi jawaban AI",
			Description:     "Dikirim manual dari Playground saat hasil test AI perlu dibagikan ke grup tim.",
			DefaultTemplate: defaultAIAnswerCurationGroupTemplate,
			DefaultEnabled:  true,
		},
		{
			Key:             groupNotificationTriggerCommerceOrderCreated,
			PluginKey:       "commerce",
			Label:           "Pesanan baru",
			Description:     "Dikirim saat plugin Produk & Pesanan membuat draft/order baru.",
			DefaultTemplate: defaultCommerceOrderCreatedGroupTemplate,
			DefaultEnabled:  false,
		},
		{
			Key:             groupNotificationTriggerCommerceOrderConfirmed,
			PluginKey:       "commerce",
			Label:           "Pesanan dikonfirmasi",
			Description:     "Dikirim saat status pesanan berubah menjadi confirmed.",
			DefaultTemplate: defaultCommerceOrderConfirmedGroupTemplate,
			DefaultEnabled:  false,
		},
		{
			Key:             groupNotificationTriggerBookingCreated,
			PluginKey:       "booking",
			Label:           "Booking baru",
			Description:     "Dikirim saat plugin Booking membuat appointment baru.",
			DefaultTemplate: defaultBookingCreatedGroupTemplate,
			DefaultEnabled:  false,
		},
		{
			Key:             groupNotificationTriggerBookingUpdated,
			PluginKey:       "booking",
			Label:           "Booking diperbarui",
			Description:     "Dikirim saat jadwal atau status booking berubah.",
			DefaultTemplate: defaultBookingUpdatedGroupTemplate,
			DefaultEnabled:  false,
		},
	}
}

func groupNotificationTriggerDefinitionByKey(triggerKey string) (groupNotificationTriggerDefinition, bool) {
	triggerKey = strings.TrimSpace(triggerKey)
	for _, item := range groupNotificationTriggerDefinitions() {
		if item.Key == triggerKey {
			return item, true
		}
	}
	return groupNotificationTriggerDefinition{}, false
}

func groupNotificationVariableItems() []map[string]any {
	items := []struct {
		key     string
		label   string
		group   string
		example string
	}{
		{"business.name", "Nama bisnis", "Bisnis", "Kedai Sari"},
		{"customer.name", "Nama customer", "Customer", "Ayu"},
		{"customer.phone", "Nomor customer", "Customer", "+6281234567890"},
		{"conversation.id", "ID percakapan", "Percakapan", "8f2c..."},
		{"conversation.link", "Link inbox", "Percakapan", "https://app.oneflow.id/dashboard/inbox"},
		{"conversation.last_message", "Pesan terakhir", "Percakapan", "Saya mau komplain pembayaran."},
		{"conversation.summary", "Ringkasan", "Percakapan", "Pertanyaan customer dan alasan handoff."},
		{"handoff.reason", "Alasan bantuan", "AI handoff", "Customer butuh keputusan admin."},
		{"ai.agent", "Nama AI agent", "AI", "Sales Bot"},
		{"curation.status", "Status kurasi", "Kurasi", "Perlu dicek"},
		{"curation.decision", "Keputusan AI", "Kurasi", "Dijawab AI"},
		{"curation.question", "Pertanyaan test", "Kurasi", "Berapa harga paketnya?"},
		{"curation.answer", "Jawaban AI", "Kurasi", "Harga paket tersedia mulai dari Rp250.000."},
		{"curation.notes", "Catatan kurasi", "Kurasi", "Knowledge belum ter-match."},
		{"curation.next_action", "Aksi berikutnya", "Kurasi", "Lengkapi FAQ harga lalu test ulang."},
		{"curation.confidence", "Confidence", "Kurasi", "72%"},
		{"curation.credits", "Kredit terpakai", "Kurasi", "2 kredit"},
		{"curation.link", "Link Playground", "Kurasi", "https://app.oneflow.id/playground"},
		{"plugin.name", "Nama plugin", "Plugin", "Produk & Pesanan"},
		{"event.type", "Jenis event", "Event", "Pesanan baru"},
		{"order.id", "ID pesanan", "Pesanan", "ORD-123"},
		{"order.status", "Status pesanan", "Pesanan", "draft"},
		{"order.total", "Total pesanan", "Pesanan", "Rp250.000"},
		{"order.items", "Item pesanan", "Pesanan", "2x Kopi Susu, 1x Roti"},
		{"order.link", "Link pesanan", "Pesanan", "https://app.oneflow.id/dashboard/commerce-orders"},
		{"booking.id", "ID booking", "Booking", "BOOK-123"},
		{"booking.service", "Layanan booking", "Booking", "Konsultasi 60 menit"},
		{"booking.time", "Jadwal booking", "Booking", "13 Mei 2026 14:00"},
		{"booking.status", "Status booking", "Booking", "scheduled"},
		{"booking.link", "Link booking", "Booking", "https://app.oneflow.id/dashboard/booking"},
		{"event.created_at", "Waktu notifikasi", "Event", "13 Mei 2026 10:30"},
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"key":     item.key,
			"label":   item.label,
			"group":   item.group,
			"example": item.example,
		})
	}
	return result
}

func allowedGroupNotificationVariables() map[string]bool {
	allowed := map[string]bool{}
	for _, item := range groupNotificationVariableItems() {
		allowed[fmt.Sprint(item["key"])] = true
	}
	return allowed
}

func validateGroupNotificationTemplate(triggerKey, templateText string) error {
	if _, ok := groupNotificationTriggerDefinitionByKey(triggerKey); !ok {
		return fmt.Errorf("unsupported notification trigger")
	}
	templateText = strings.TrimSpace(templateText)
	if templateText == "" {
		return fmt.Errorf("templateText is required")
	}
	if len([]rune(templateText)) > 2000 {
		return fmt.Errorf("templateText is too long")
	}
	allowed := allowedGroupNotificationVariables()
	matches := groupNotificationVariablePattern.FindAllStringSubmatch(templateText, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		key := strings.TrimSpace(match[1])
		if !allowed[key] {
			return fmt.Errorf("unknown template variable: %s", key)
		}
	}
	return nil
}

func renderGroupNotificationTemplate(templateText string, values map[string]string) string {
	return strings.TrimSpace(groupNotificationVariablePattern.ReplaceAllStringFunc(templateText, func(token string) string {
		match := groupNotificationVariablePattern.FindStringSubmatch(token)
		if len(match) < 2 {
			return token
		}
		value := strings.TrimSpace(values[strings.TrimSpace(match[1])])
		if value == "" {
			return "-"
		}
		return value
	}))
}

func defaultGroupNotificationRuleItem(triggerKey string) map[string]any {
	def, ok := groupNotificationTriggerDefinitionByKey(triggerKey)
	if !ok {
		return nil
	}
	return map[string]any{
		"triggerKey":   def.Key,
		"pluginKey":    def.PluginKey,
		"label":        def.Label,
		"description":  def.Description,
		"isEnabled":    def.DefaultEnabled,
		"templateText": def.DefaultTemplate,
		"isDefault":    true,
		"updatedAt":    nil,
	}
}

func (s *Server) groupNotificationRuleItem(ctx context.Context, triggerKey string) (map[string]any, error) {
	if _, ok := groupNotificationTriggerDefinitionByKey(triggerKey); !ok {
		return nil, fmt.Errorf("unsupported notification trigger")
	}
	item := defaultGroupNotificationRuleItem(triggerKey)
	var isEnabled bool
	var templateText string
	var updatedAt time.Time
	err := s.db.QueryRow(ctx, `
		SELECT is_enabled, template_text, updated_at
		FROM group_notification_rules
		WHERE organization_id = $1 AND trigger_key = $2
	`, s.organizationID(ctx), triggerKey).Scan(&isEnabled, &templateText, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, nil
	}
	if err != nil {
		return nil, err
	}
	item["isEnabled"] = isEnabled
	item["templateText"] = templateText
	item["isDefault"] = false
	item["updatedAt"] = updatedAt
	return item, nil
}

func (s *Server) groupNotificationRuleItems(ctx context.Context) ([]map[string]any, error) {
	definitions := groupNotificationTriggerDefinitions()
	items := make([]map[string]any, 0, len(definitions))
	for _, definition := range definitions {
		item, err := s.groupNotificationRuleItem(ctx, definition.Key)
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *Server) activeGroupNotificationRule(ctx context.Context, triggerKey string) (bool, string, error) {
	item, err := s.groupNotificationRuleItem(ctx, triggerKey)
	if err != nil {
		return false, "", err
	}
	return item["isEnabled"] == true, fmt.Sprint(item["templateText"]), nil
}

func (s *Server) currentOrganizationName(ctx context.Context) string {
	var name string
	if err := s.db.QueryRow(ctx, `SELECT name FROM organizations WHERE id = $1`, s.organizationID(ctx)).Scan(&name); err == nil && strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return "Oneflow.id"
}

func (s *Server) formatHumanHandoffGroupNotification(ctx context.Context, conversationID, phone, contactName, question, reason string) (string, bool, error) {
	enabled, templateText, err := s.activeGroupNotificationRule(ctx, groupNotificationTriggerHumanHandoff)
	if err != nil {
		return "", false, err
	}
	if !enabled {
		return "", false, nil
	}
	name := strings.TrimSpace(contactName)
	if name == "" {
		name = "Contact"
	}
	phone = strings.TrimSpace(phone)
	if phone == "" {
		phone = "-"
	}
	question = strings.TrimSpace(question)
	if question == "" {
		question = "-"
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "Customer membutuhkan bantuan tim."
	}
	conversationLink := s.dashboardURL("inbox")
	if conversationID != "" && conversationID != "TEST" {
		conversationLink += "?conversation=" + conversationID
	}
	values := map[string]string{
		"business.name":             s.currentOrganizationName(ctx),
		"customer.name":             name,
		"customer.phone":            phone,
		"conversation.id":           conversationID,
		"conversation.link":         conversationLink,
		"conversation.last_message": question,
		"conversation.summary":      escalationSummaryText(question, reason),
		"handoff.reason":            reason,
		"event.created_at":          time.Now().Format("02 Jan 2006 15:04"),
	}
	text := renderGroupNotificationTemplate(templateText, values)
	if text == "" {
		text = s.formatEscalationTeamNotice(conversationID, phone, name, escalationSummaryText(question, reason))
	}
	return text, true, nil
}

func trimNotificationValue(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if maxRunes <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "..."
}

func curationNotesText(issues, passes []string) string {
	source := issues
	if len(source) == 0 {
		source = passes
	}
	parts := []string{}
	for _, item := range source {
		item = trimNotificationValue(item, 160)
		if item == "" {
			continue
		}
		parts = append(parts, "- "+item)
		if len(parts) >= 4 {
			break
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "\n")
}

func curationCreditText(value any) string {
	switch parsed := value.(type) {
	case nil:
		return "-"
	case float64:
		if parsed <= 0 {
			return "-"
		}
		if parsed == math.Trunc(parsed) {
			return fmt.Sprintf("%.0f kredit", parsed)
		}
		return fmt.Sprintf("%.2f kredit", parsed)
	case string:
		parsed = strings.TrimSpace(parsed)
		if parsed == "" || parsed == "-" {
			return "-"
		}
		if strings.Contains(strings.ToLower(parsed), "kredit") {
			return parsed
		}
		return parsed + " kredit"
	default:
		text := strings.TrimSpace(fmt.Sprint(parsed))
		if text == "" || text == "<nil>" {
			return "-"
		}
		return text
	}
}

func (s *Server) formatAIAnswerCurationGroupNotification(ctx context.Context, req aiAnswerCurationNotificationRequest) (string, bool, error) {
	enabled, templateText, err := s.activeGroupNotificationRule(ctx, groupNotificationTriggerAIAnswerCuration)
	if err != nil {
		return "", false, err
	}
	if !enabled {
		return "", false, nil
	}

	status := trimNotificationValue(req.StatusLabel, 80)
	if status == "" {
		status = "Perlu dicek"
	}
	decision := trimNotificationValue(req.Decision, 80)
	if decision == "" {
		decision = "-"
	}
	agentName := trimNotificationValue(req.AIAgentName, 120)
	if agentName == "" {
		agentName = "AI Agent"
	}
	customerName := trimNotificationValue(req.CustomerName, 120)
	if customerName == "" {
		customerName = "Pelanggan Test"
	}
	nextAction := trimNotificationValue(req.NextAction, 220)
	if nextAction == "" {
		nextAction = "Cek ulang instruksi, Knowledge, atau aturan handoff lalu test lagi."
	}
	confidence := "-"
	if req.Confidence > 0 {
		confidence = fmt.Sprintf("%.0f%%", math.Round(req.Confidence*100))
	}
	values := map[string]string{
		"business.name":        s.currentOrganizationName(ctx),
		"customer.name":        customerName,
		"customer.phone":       "-",
		"ai.agent":             agentName,
		"curation.status":      status,
		"curation.decision":    decision,
		"curation.question":    trimNotificationValue(req.Question, 280),
		"curation.answer":      trimNotificationValue(req.AnswerText, 560),
		"curation.notes":       trimNotificationValue(curationNotesText(req.Issues, req.Passes), 520),
		"curation.next_action": nextAction,
		"curation.confidence":  confidence,
		"curation.credits":     curationCreditText(req.CreditsUsed),
		"curation.link":        strings.TrimRight(s.dashboardURL("playground"), "/"),
		"event.type":           "Kurasi jawaban AI",
		"event.created_at":     time.Now().Format("02 Jan 2006 15:04"),
	}
	if values["curation.question"] == "" {
		values["curation.question"] = "-"
	}
	if values["curation.answer"] == "" {
		values["curation.answer"] = "-"
	}
	text := renderGroupNotificationTemplate(templateText, values)
	if text == "" {
		text = fmt.Sprintf("[%s] Kurasi jawaban AI\nStatus: %s\nAgent: %s\nCatatan:\n%s", values["business.name"], status, agentName, values["curation.notes"])
	}
	return text, true, nil
}

func (s *Server) formatGroupNotificationTest(ctx context.Context, triggerKey string) (string, bool, error) {
	switch strings.TrimSpace(triggerKey) {
	case groupNotificationTriggerHumanHandoff, "":
		return s.formatHumanHandoffGroupNotification(ctx, "TEST", "+6281234567890", "Pelanggan Test", "Saya butuh bantuan admin.", "Contoh test notifikasi dari dashboard.")
	case groupNotificationTriggerAIAnswerCuration:
		return s.formatAIAnswerCurationGroupNotification(ctx, aiAnswerCurationNotificationRequest{
			AIAgentName:  "Sales Bot",
			CustomerName: "Pelanggan Test",
			Question:     "Berapa harga paket yang paling cocok untuk toko saya?",
			AnswerText:   "Paket yang cocok tergantung kebutuhan toko. Untuk harga final, tim akan bantu cek kebutuhan dan mengonfirmasi detailnya.",
			Decision:     "Dijawab AI",
			StatusLabel:  "Perlu dicek",
			Issues:       []string{"Tidak terlihat sumber Knowledge yang eksplisit untuk harga paket."},
			NextAction:   "Tambahkan FAQ harga/paket lalu test ulang di Playground.",
			Confidence:   0.62,
			CreditsUsed:  float64(2),
		})
	case groupNotificationTriggerCommerceOrderCreated, groupNotificationTriggerCommerceOrderConfirmed:
		status := "draft"
		if strings.TrimSpace(triggerKey) == groupNotificationTriggerCommerceOrderConfirmed {
			status = "confirmed"
		}
		return s.formatBusinessPluginGroupNotification(ctx, triggerKey, map[string]any{
			"id":            "TEST-ORDER",
			"customerName":  "Pelanggan Test",
			"customerPhone": "+6281234567890",
			"status":        status,
			"totalAmount":   250000,
			"items": []map[string]any{
				{"productName": "Produk Contoh", "quantity": 2},
			},
		})
	case groupNotificationTriggerBookingCreated, groupNotificationTriggerBookingUpdated:
		return s.formatBusinessPluginGroupNotification(ctx, triggerKey, map[string]any{
			"id":             "TEST-BOOKING",
			"customerName":   "Pelanggan Test",
			"customerPhone":  "+6281234567890",
			"serviceName":    "Konsultasi Contoh",
			"scheduledStart": time.Now().Add(24 * time.Hour),
			"status":         "scheduled",
		})
	default:
		return "", false, fmt.Errorf("unsupported notification trigger")
	}
}

func (s *Server) formatBusinessPluginGroupNotification(ctx context.Context, triggerKey string, item map[string]any) (string, bool, error) {
	enabled, templateText, err := s.activeGroupNotificationRule(ctx, triggerKey)
	if err != nil {
		return "", false, err
	}
	if !enabled {
		return "", false, nil
	}
	def, ok := groupNotificationTriggerDefinitionByKey(triggerKey)
	if !ok {
		return "", false, fmt.Errorf("unsupported notification trigger")
	}
	values := map[string]string{
		"business.name":    s.currentOrganizationName(ctx),
		"plugin.name":      pluginNotificationName(def.PluginKey),
		"event.type":       def.Label,
		"event.created_at": time.Now().Format("02 Jan 2006 15:04"),
		"customer.name":    valueOrFallback(stringFromMap(item, "customerName"), "Customer"),
		"customer.phone":   valueOrFallback(stringFromMap(item, "customerPhone"), "-"),
	}
	conversationID := strings.TrimSpace(stringFromMap(item, "conversationId"))
	values["conversation.id"] = valueOrFallback(conversationID, "-")
	values["conversation.link"] = s.dashboardURL("inbox")

	switch def.PluginKey {
	case "commerce":
		values["order.id"] = valueOrFallback(stringFromMap(item, "id"), "-")
		values["order.status"] = valueOrFallback(stringFromMap(item, "status"), "-")
		values["order.total"] = formatIDR(floatFromMap(item, "totalAmount"))
		values["order.items"] = commerceOrderItemsSummary(item["items"])
		values["order.link"] = s.dashboardURL("pesanan")
		values["conversation.summary"] = fmt.Sprintf("Pesanan %s untuk %s senilai %s.", values["order.status"], values["customer.name"], values["order.total"])
	case "booking":
		values["booking.id"] = valueOrFallback(stringFromMap(item, "id"), "-")
		values["booking.service"] = valueOrFallback(stringFromMap(item, "serviceName"), "-")
		values["booking.time"] = formatNotificationDateTime(item["scheduledStart"])
		values["booking.status"] = valueOrFallback(stringFromMap(item, "status"), "-")
		values["booking.link"] = s.dashboardURL("booking")
		values["conversation.summary"] = fmt.Sprintf("Booking %s untuk %s pada %s.", values["booking.service"], values["customer.name"], values["booking.time"])
	}
	text := renderGroupNotificationTemplate(templateText, values)
	if text == "" {
		text = fmt.Sprintf("[%s] %s\nCustomer: %s\nNomor: %s", values["business.name"], def.Label, values["customer.name"], values["customer.phone"])
	}
	return text, true, nil
}

func (s *Server) notifyBusinessPluginGroup(ctx context.Context, triggerKey string, item map[string]any) {
	text, enabled, err := s.formatBusinessPluginGroupNotification(ctx, triggerKey, item)
	if err != nil {
		_ = s.recordNotificationGroupDeliveryFailure(context.Background(), "group_notification_format_"+triggerKey, s.activeNotificationGroupID(ctx), err)
		return
	}
	if !enabled || strings.TrimSpace(text) == "" {
		return
	}
	_ = s.notifyNotificationGroup(ctx, text)
}

func (s *Server) dashboardURL(segment string) string {
	base := strings.TrimRight(s.cfg.AppPublicURL, "/")
	if base == "" {
		base = "http://localhost:3000"
	}
	segment = strings.Trim(segment, "/")
	if segment == "" {
		return base + "/dashboard"
	}
	return base + "/dashboard/" + segment
}

func pluginNotificationName(pluginKey string) string {
	switch pluginKey {
	case "commerce":
		return "Produk & Pesanan"
	case "booking":
		return "Booking"
	default:
		return "AI CS"
	}
}

func valueOrFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func formatIDR(value float64) string {
	if value <= 0 {
		return "Rp0"
	}
	raw := strconv.FormatInt(int64(math.Round(value)), 10)
	parts := []string{}
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	parts = append([]string{raw}, parts...)
	return "Rp" + strings.Join(parts, ".")
}

func formatNotificationDateTime(value any) string {
	switch parsed := value.(type) {
	case time.Time:
		if parsed.IsZero() {
			return "-"
		}
		return parsed.Format("02 Jan 2006 15:04")
	case string:
		if parsed == "" {
			return "-"
		}
		if ts, err := time.Parse(time.RFC3339, parsed); err == nil {
			return ts.Format("02 Jan 2006 15:04")
		}
		return parsed
	default:
		return "-"
	}
}

func commerceOrderItemsSummary(value any) string {
	rawItems, ok := value.([]map[string]any)
	if !ok {
		if looseItems, looseOK := value.([]any); looseOK {
			items := make([]map[string]any, 0, len(looseItems))
			for _, raw := range looseItems {
				if item, itemOK := raw.(map[string]any); itemOK {
					items = append(items, item)
				}
			}
			rawItems = items
			ok = true
		}
	}
	if !ok || len(rawItems) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(rawItems))
	for _, item := range rawItems {
		name := valueOrFallback(stringFromMap(item, "productName"), "Item")
		qty := intFromMap(item, "quantity")
		if qty <= 0 {
			qty = 1
		}
		parts = append(parts, fmt.Sprintf("%dx %s", qty, name))
	}
	return strings.Join(parts, ", ")
}

func (s *Server) notifyNotificationGroup(ctx context.Context, text string) error {
	groupID := strings.TrimSpace(s.activeNotificationGroupID(ctx))
	if groupID == "" || strings.TrimSpace(text) == "" {
		if strings.TrimSpace(text) != "" {
			_ = s.recordNotificationGroupDeliveryFailure(context.Background(), "escalation_unbound_"+whatsAppSessionIDFromContext(ctx), "", errors.New("no escalation group bound for whatsapp session"))
		}
		return nil
	}
	err := s.notifyNotificationGroupByID(ctx, groupID, text)
	if err != nil {
		_ = s.recordNotificationGroupDeliveryFailure(context.Background(), "escalation_notice", groupID, err)
	}
	return err
}

func (s *Server) notifyNotificationGroupByID(ctx context.Context, groupID, text string) error {
	groupID = strings.TrimSpace(groupID)
	text = strings.TrimSpace(text)
	if groupID == "" || text == "" {
		return nil
	}
	_, err := s.sendWhatsAppText(ctx, groupID, text)
	return err
}

func (s *Server) recordNotificationGroupDeliveryFailure(ctx context.Context, source, groupID string, deliveryErr error) error {
	if deliveryErr == nil {
		return nil
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = "notification_group_delivery"
	}
	groupID = strings.TrimSpace(groupID)
	target := groupID
	if target == "" {
		target = "unbound group"
	}
	message := fmt.Sprintf("Failed to send WhatsApp escalation group notification (%s) to %s: %s", source, target, deliveryErr.Error())
	_, err := s.db.Exec(ctx, `
		INSERT INTO system_alerts (fingerprint, severity, source, message, status, first_seen_at, last_seen_at, delivered_at, organization_id)
		VALUES ($1, 'warning', 'wa_escalation_group', $2, 'open', NOW(), NOW(), NOW(), NULLIF($3, '')::uuid)
		ON CONFLICT (fingerprint) DO UPDATE SET
		  message = EXCLUDED.message,
		  status = 'open',
		  last_seen_at = NOW(),
		  delivered_at = NOW()
	`, "wa-escalation-group-delivery-"+source, message, s.organizationID(ctx))
	return err
}

func (s *Server) resolveEscalationGroupUnboundAlerts(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	_, err := s.db.Exec(ctx, `
		UPDATE system_alerts
		SET status = 'resolved',
		    resolved_at = NOW()
		WHERE source = 'wa_escalation_group'
		  AND status = 'open'
		  AND fingerprint = $1
		  AND (organization_id IS NULL OR organization_id = NULLIF($2, '')::uuid)
	`, "wa-escalation-group-delivery-escalation_unbound_"+sessionID, s.organizationID(ctx))
	return err
}

func (s *Server) activeNotificationGroupID(ctx context.Context) string {
	var groupID string
	sessionID := whatsAppSessionIDFromContext(ctx)
	err := s.db.QueryRow(ctx, `
		SELECT group_id
		FROM wa_escalation_group_bindings
		WHERE status = 'bound'
		  AND COALESCE(group_id, '') <> ''
		  AND (NULLIF($1, '')::uuid IS NULL OR organization_id = NULLIF($1, '')::uuid)
		  AND (NULLIF($2, '')::uuid IS NULL OR whatsapp_session_id = NULLIF($2, '')::uuid)
		ORDER BY bound_at DESC NULLS LAST, updated_at DESC
		LIMIT 1
	`, s.organizationID(ctx), sessionID).Scan(&groupID)
	if err == nil && strings.TrimSpace(groupID) != "" {
		return strings.TrimSpace(groupID)
	}
	return ""
}

func (s *Server) escalationGroupStatus(ctx context.Context) map[string]any {
	status := map[string]any{
		"bound":   false,
		"pending": nil,
		"group":   nil,
	}
	sessionID := whatsAppSessionIDFromContext(ctx)

	var groupID, groupName, code string
	var boundAt, updatedAt time.Time
	err := s.db.QueryRow(ctx, `
		SELECT group_id, COALESCE(group_name, ''), binding_code, bound_at, updated_at
		FROM wa_escalation_group_bindings
		WHERE status = 'bound'
		  AND COALESCE(group_id, '') <> ''
		  AND (NULLIF($1, '')::uuid IS NULL OR organization_id = NULLIF($1, '')::uuid)
		  AND (NULLIF($2, '')::uuid IS NULL OR whatsapp_session_id = NULLIF($2, '')::uuid)
		ORDER BY bound_at DESC NULLS LAST, updated_at DESC
		LIMIT 1
	`, s.organizationID(ctx), sessionID).Scan(&groupID, &groupName, &code, &boundAt, &updatedAt)
	if err == nil {
		status["bound"] = true
		status["group"] = map[string]any{
			"groupId":   groupID,
			"groupName": groupName,
			"code":      code,
			"boundAt":   boundAt,
			"updatedAt": updatedAt,
		}
	}

	var pendingCode string
	var pendingExpiresAt, pendingCreatedAt time.Time
	err = s.db.QueryRow(ctx, `
		SELECT binding_code, expires_at, created_at
		FROM wa_escalation_group_bindings
		WHERE status = 'pending'
		  AND expires_at > NOW()
		  AND (NULLIF($1, '')::uuid IS NULL OR organization_id = NULLIF($1, '')::uuid)
		  AND (NULLIF($2, '')::uuid IS NULL OR whatsapp_session_id = NULLIF($2, '')::uuid)
		ORDER BY created_at DESC
		LIMIT 1
	`, s.organizationID(ctx), sessionID).Scan(&pendingCode, &pendingExpiresAt, &pendingCreatedAt)
	if err == nil {
		status["pending"] = map[string]any{
			"code":      pendingCode,
			"text":      pendingCode,
			"expiresAt": pendingExpiresAt,
			"createdAt": pendingCreatedAt,
		}
	}
	return status
}

func generateEscalationBindingCode() string {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("UR-ESC-%06d", time.Now().UnixNano()%1_000_000)
	}
	return "UR-ESC-" + strings.ToUpper(hex.EncodeToString(raw[:]))
}

func escalationBindingCodeMatches(text, code string) bool {
	text = strings.ToUpper(strings.TrimSpace(text))
	code = strings.ToUpper(strings.TrimSpace(code))
	if text == "" || code == "" {
		return false
	}
	return strings.Contains(text, code)
}

func (s *Server) formatEscalationTeamNotice(conversationID, phone, contactName, reason string) string {
	name := strings.TrimSpace(contactName)
	if name == "" {
		name = "Contact"
	}
	summary := strings.TrimSpace(reason)
	if summary == "" {
		summary = "User membutuhkan bantuan human."
	}
	return fmt.Sprintf(
		"[Escalation]\nConversation: %s\nContact: %s\nPhone: %s\nRingkasan masalah: %s",
		conversationID,
		name,
		phone,
		summary,
	)
}

func escalationSummaryText(question, reason string) string {
	question = strings.TrimSpace(question)
	reason = strings.TrimSpace(reason)
	if question == "" {
		question = "-"
	}
	if reason == "" {
		reason = "Customer membutuhkan bantuan tim."
	}
	return fmt.Sprintf("Pertanyaan customer: %s\nAlasan eskalasi: %s", question, reason)
}

func (s *Server) formatEscalationInboxSummary(contactName, phone, question, reason string) string {
	name := strings.TrimSpace(contactName)
	if name == "" {
		name = "Contact"
	}
	phone = strings.TrimSpace(phone)
	if phone == "" {
		phone = "-"
	}
	return fmt.Sprintf(
		"Ringkasan eskalasi\nKontak: %s\nPhone: %s\n%s",
		name,
		phone,
		escalationSummaryText(question, reason),
	)
}

func (s *Server) insertEscalationSummaryMessageTx(ctx context.Context, tx pgx.Tx, conversationID, contactName, phone, question, reason string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO messages (
		  conversation_id,
		  sender_type,
		  direction,
		  content_type,
		  text,
		  sent_at,
		  delivered_at,
		  organization_id
		)
		VALUES ($1, 'system', 'outbound', 'system', $2, NOW(), NOW(), $3)
	`, conversationID, s.formatEscalationInboxSummary(contactName, phone, question, reason), s.organizationID(ctx))
	return err
}

func formatEscalationGroupBoundMessage(groupName string) string {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = "grup ini"
	}
	return fmt.Sprintf(
		"[Oneflow.id]\nGrup notifikasi berhasil terhubung ke dashboard.\n\nGrup: %s\nStatus: aktif\n\nNotifikasi operasional, termasuk saat AI membutuhkan bantuan tim, akan dikirim ke grup ini.",
		groupName,
	)
}

func (s *Server) callAI(ctx context.Context, conversationID, messageText, customerName string) (aiDecisionResponse, error) {
	return s.callAIWithHistory(ctx, conversationID, messageText, customerName, nil)
}

func (s *Server) callAIWithImage(ctx context.Context, conversationID, messageText, customerName string, image *aiImagePayload) (aiDecisionResponse, error) {
	return s.callAIWithHistoryAndImage(ctx, conversationID, messageText, customerName, nil, image)
}

func (s *Server) callAIWithHistory(ctx context.Context, conversationID, messageText, customerName string, history []aiChatHistoryMessage) (aiDecisionResponse, error) {
	return s.callAIWithHistoryAndImage(ctx, conversationID, messageText, customerName, history, nil)
}

func (s *Server) callAIWithHistoryAndImage(ctx context.Context, conversationID, messageText, customerName string, history []aiChatHistoryMessage, image *aiImagePayload) (aiDecisionResponse, error) {
	aiAgentID := aiAgentIDFromContext(ctx)
	if aiAgentID == "" {
		aiAgentID, _ = s.aiAgentIDForConversation(ctx, conversationID)
	}
	bodyPayload := map[string]any{
		"conversation_id": conversationID,
		"organization_id": s.organizationID(ctx),
		"message_text":    messageText,
		"customer_name":   customerName,
	}
	if aiAgentID != "" {
		bodyPayload["ai_agent_id"] = aiAgentID
	}
	if len(history) > 0 {
		bodyPayload["history"] = sanitizeAIHistory(history, 10)
	}
	if image != nil && strings.TrimSpace(image.Base64) != "" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(image.MimeType)), "image/") {
		bodyPayload["image_base64"] = strings.TrimSpace(image.Base64)
		bodyPayload["image_mime_type"] = strings.TrimSpace(image.MimeType)
	}
	body, _ := json.Marshal(bodyPayload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.AIServiceBaseURL+"/api/decide", bytes.NewReader(body))
	if err != nil {
		return aiDecisionResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return aiDecisionResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return aiDecisionResponse{}, fmt.Errorf("ai-service returned %d", resp.StatusCode)
	}

	var payload aiDecisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return aiDecisionResponse{}, err
	}
	return payload, nil
}

func (s *Server) recentConversationAIHistoryTx(ctx context.Context, tx pgx.Tx, conversationID, excludeMessageID string, limit int) ([]aiChatHistoryMessage, error) {
	conversationID = strings.TrimSpace(conversationID)
	organizationID := strings.TrimSpace(s.organizationID(ctx))
	if conversationID == "" || organizationID == "" || limit <= 0 {
		return nil, nil
	}

	rows, err := tx.Query(ctx, `
		SELECT direction, text
		FROM (
		  SELECT
		    direction::text AS direction,
		    COALESCE(text, '') AS text,
		    COALESCE(sent_at, created_at) AS message_at,
		    created_at,
		    id
		  FROM messages
		  WHERE conversation_id = $1
		    AND organization_id = $2
		    AND (NULLIF($3, '')::uuid IS NULL OR id <> NULLIF($3, '')::uuid)
		    AND direction::text IN ('inbound', 'outbound')
		    AND COALESCE(BTRIM(text), '') <> ''
		  ORDER BY COALESCE(sent_at, created_at) DESC, created_at DESC, id DESC
		  LIMIT $4
		) recent
		ORDER BY message_at ASC, created_at ASC, id ASC
	`, conversationID, organizationID, strings.TrimSpace(excludeMessageID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]aiChatHistoryMessage, 0, limit)
	for rows.Next() {
		var direction, text string
		if err := rows.Scan(&direction, &text); err != nil {
			return nil, err
		}
		role := "assistant"
		if direction == "inbound" {
			role = "user"
		}
		history = append(history, aiChatHistoryMessage{Role: role, Text: text})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sanitizeAIHistory(history, limit), nil
}

func (s *Server) aiAgentIDForConversation(ctx context.Context, conversationID string) (string, error) {
	var aiAgentID string
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(ai_agent_id::text, '')
		FROM conversations
		WHERE id = NULLIF($1, '')::uuid
		  AND organization_id = NULLIF($2, '')::uuid
	`, strings.TrimSpace(conversationID), s.organizationID(ctx)).Scan(&aiAgentID)
	return strings.TrimSpace(aiAgentID), err
}

func sanitizeAIHistory(history []aiChatHistoryMessage, limit int) []aiChatHistoryMessage {
	cleaned := make([]aiChatHistoryMessage, 0, len(history))
	for _, item := range history {
		role := strings.TrimSpace(item.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		if len(text) > 1200 {
			text = text[:1200]
		}
		cleaned = append(cleaned, aiChatHistoryMessage{Role: role, Text: text})
	}
	if limit > 0 && len(cleaned) > limit {
		return cleaned[len(cleaned)-limit:]
	}
	return cleaned
}

func clampInt(value, minValue, maxValue, defaultValue int) int {
	if value == 0 {
		value = defaultValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

type contactMemoryWriteSettings struct {
	ContactEnabled      bool
	Enabled             bool
	AutoSaveEnabled     bool
	AIExtractionEnabled bool
	VerifierEnabled     bool
	LLMValidatorEnabled bool
	AdminNotesEnabled   bool
	MaxItems            int
	MaxChars            int
	RetentionDays       int
}

type contactMemoryDraft struct {
	MemoryType      string
	Value           string
	NormalizedValue string
	Confidence      float64
}

type contactMemoryLLMValidationResult struct {
	Decision        string         `json:"decision"`
	RiskCategory    string         `json:"risk_category"`
	Reason          string         `json:"reason"`
	MemoryType      string         `json:"memory_type"`
	Value           string         `json:"value"`
	NormalizedValue string         `json:"normalized_value"`
	Confidence      float64        `json:"confidence"`
	ModelName       string         `json:"model_name"`
	UsageMetadata   map[string]any `json:"usage_metadata"`
}

func (result contactMemoryLLMValidationResult) DecisionToGateResult() string {
	switch strings.ToLower(strings.TrimSpace(result.Decision)) {
	case "approve":
		return "pass"
	case "reject":
		return "reject"
	default:
		return "uncertain"
	}
}

func (s *Server) maybeCaptureContactMemoryFromInboundTx(ctx context.Context, tx pgx.Tx, contactID, conversationID, messageID, text string) (int, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	settings, err := s.contactMemoryWriteSettingsTx(ctx, tx, contactID, conversationID)
	if err != nil {
		return 0, err
	}
	if !settings.ContactEnabled || !settings.Enabled || !settings.AutoSaveEnabled || !settings.AIExtractionEnabled {
		return 0, nil
	}

	drafts := extractContactMemoryDrafts(text, settings.MaxItems, settings.MaxChars)
	if len(drafts) == 0 {
		return 0, nil
	}

	captured := 0
	for _, draft := range drafts {
		riskCategory, matchedCategory := classifyContactMemoryRisk(text + " " + draft.Value)
		gate1Result := "pass"
		gate2Result := ""
		finalStatus := "approved"
		rejectionReason := ""
		var validation contactMemoryLLMValidationResult
		if riskCategory == "uncertain" {
			gate1Result = "uncertain"
			finalStatus = "review"
			rejectionReason = "memory draft needs LLM validator review"
			if settings.LLMValidatorEnabled {
				validation, err = s.validateContactMemoryWithLLM(ctx, text, draft)
				if err != nil {
					gate2Result = "uncertain"
					rejectionReason = "LLM validator unavailable"
				} else {
					gate2Result = validation.DecisionToGateResult()
					rejectionReason = validation.Reason
					switch validation.Decision {
					case "approve":
						finalStatus = "approved"
						riskCategory = "safe"
						if normalizedType := normalizeContactMemoryType(validation.MemoryType); normalizedType != "" {
							draft.MemoryType = normalizedType
						}
						if strings.TrimSpace(validation.Value) != "" {
							draft.Value = cleanContactMemoryDraftValue(validation.Value, settings.MaxChars)
						}
						if strings.TrimSpace(validation.NormalizedValue) != "" {
							draft.NormalizedValue = normalizeContactMemoryValue(validation.NormalizedValue)
						}
						if draft.NormalizedValue == "" {
							draft.NormalizedValue = normalizeContactMemoryValue(draft.Value)
						}
						if validation.Confidence > 0 {
							draft.Confidence = validation.Confidence
						}
						rejectionReason = ""
					case "reject":
						finalStatus = "rejected"
						if validation.RiskCategory != "" {
							riskCategory = validation.RiskCategory
						}
						if rejectionReason == "" {
							rejectionReason = "business or instruction claim must come from official knowledge/tool data"
						}
					default:
						finalStatus = "review"
						if validation.RiskCategory != "" {
							riskCategory = validation.RiskCategory
						}
						if rejectionReason == "" {
							rejectionReason = "LLM validator could not decide safely"
						}
					}
				}
			}
		} else if riskCategory != "safe" {
			gate1Result = "reject"
			finalStatus = "rejected"
			rejectionReason = "business or instruction claim must come from official knowledge/tool data"
		}
		if finalStatus == "approved" && len(validation.UsageMetadata) > 0 {
			inputTokens, outputTokens, embeddingTokens, actualCostUSD := usageTokensFromDecision(aiDecisionResponse{UsageMetadata: validation.UsageMetadata}, 420, 80, 0)
			modelName := strings.TrimSpace(validation.ModelName)
			if modelName == "" {
				modelName = "customer_memory_validator"
			}
			if _, err := s.logCreditUsageTx(ctx, tx, conversationID, messageID, "", "customer_memory_validator", modelName, inputTokens, outputTokens, embeddingTokens, actualCostUSD); err != nil {
				gate2Result = "uncertain"
				finalStatus = "review"
				rejectionReason = "memory validator credit logging failed"
			}
		}

		var draftID string
		extractedFact := map[string]any{
			"type":            draft.MemoryType,
			"value":           draft.Value,
			"normalizedValue": draft.NormalizedValue,
			"riskCategory":    riskCategory,
		}
		if gate2Result != "" {
			extractedFact["gate2Result"] = gate2Result
		}
		if validation.Reason != "" {
			extractedFact["validatorReason"] = validation.Reason
		}
		if len(validation.UsageMetadata) > 0 {
			extractedFact["validatorUsageMetadata"] = validation.UsageMetadata
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO contact_memory_drafts (
			  organization_id, contact_id, conversation_id, message_id, raw_text, extracted_fact,
			  memory_type, risk_category, source_type, gate1_result, gate1_matched_category,
			  gate2_result, final_status, rejection_reason, confidence
			)
			VALUES (
			  $1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid, $5, $6::jsonb,
			  $7, $8, 'ai_extracted', $9, NULLIF($10, ''), NULLIF($11, ''), $12, NULLIF($13, ''), $14
			)
			RETURNING id::text
		`,
			s.organizationID(ctx),
			contactID,
			conversationID,
			messageID,
			text,
			marshalJSON(extractedFact),
			draft.MemoryType,
			riskCategory,
			gate1Result,
			matchedCategory,
			gate2Result,
			finalStatus,
			rejectionReason,
			draft.Confidence,
		).Scan(&draftID); err != nil {
			return captured, err
		}

		if finalStatus != "approved" {
			continue
		}
		var expiresAt *time.Time
		if settings.RetentionDays > 0 {
			expires := time.Now().Add(time.Duration(settings.RetentionDays) * 24 * time.Hour)
			expiresAt = &expires
		}
		if _, err := s.upsertContactMemoryTx(ctx, tx, contactID, draft.MemoryType, draft.Value, draft.NormalizedValue, "ai_extracted", draftID, conversationID, messageID, draft.Confidence, expiresAt); err != nil {
			return captured, err
		}
		captured++
	}
	return captured, nil
}

func (s *Server) validateContactMemoryWithLLM(ctx context.Context, rawText string, draft contactMemoryDraft) (contactMemoryLLMValidationResult, error) {
	body, _ := json.Marshal(map[string]any{
		"raw_text":         strings.TrimSpace(rawText),
		"memory_type":      draft.MemoryType,
		"value":            draft.Value,
		"normalized_value": draft.NormalizedValue,
		"confidence":       draft.Confidence,
	})
	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, s.cfg.AIServiceBaseURL+"/api/customer-memory/validate", bytes.NewReader(body))
	if err != nil {
		return contactMemoryLLMValidationResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return contactMemoryLLMValidationResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return contactMemoryLLMValidationResult{}, fmt.Errorf("ai-service memory validator returned %d: %s", resp.StatusCode, strings.TrimSpace(string(errorBody)))
	}
	var result contactMemoryLLMValidationResult
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return contactMemoryLLMValidationResult{}, err
	}
	result.Decision = strings.ToLower(strings.TrimSpace(result.Decision))
	if result.Decision != "approve" && result.Decision != "reject" && result.Decision != "uncertain" {
		result.Decision = "uncertain"
	}
	result.RiskCategory = strings.ToLower(strings.TrimSpace(result.RiskCategory))
	if result.RiskCategory == "" {
		result.RiskCategory = "uncertain"
	}
	if result.Decision == "approve" && result.RiskCategory != "safe" {
		result.Decision = "reject"
		if result.Reason == "" {
			result.Reason = "LLM validator marked the memory draft as unsafe"
		}
	}
	result.MemoryType = normalizeContactMemoryType(result.MemoryType)
	if result.MemoryType == "" {
		result.MemoryType = draft.MemoryType
	}
	result.Value = cleanContactMemoryDraftValue(result.Value, 1200)
	if result.Value == "" {
		result.Value = draft.Value
	}
	result.NormalizedValue = normalizeContactMemoryValue(result.NormalizedValue)
	if result.NormalizedValue == "" {
		result.NormalizedValue = normalizeContactMemoryValue(result.Value)
	}
	if result.Confidence < 0 {
		result.Confidence = 0
	}
	if result.Confidence > 1 {
		result.Confidence = 1
	}
	if result.Confidence == 0 && result.Decision == "approve" {
		result.Confidence = draft.Confidence
	}
	result.ModelName = strings.TrimSpace(result.ModelName)
	if result.UsageMetadata == nil {
		result.UsageMetadata = map[string]any{}
	}
	return result, nil
}

func (s *Server) contactMemoryWriteSettingsTx(ctx context.Context, tx pgx.Tx, contactID, conversationID string) (contactMemoryWriteSettings, error) {
	settings := contactMemoryWriteSettings{
		MaxItems:      5,
		MaxChars:      800,
		RetentionDays: 180,
	}
	err := tx.QueryRow(ctx, `
		SELECT
		  TRUE AS contact_enabled,
		  COALESCE(aa.customer_memory_enabled, FALSE) AS enabled,
		  COALESCE(aa.customer_memory_enabled, FALSE) AS auto_save_enabled,
		  COALESCE(aa.customer_memory_enabled, FALSE) AS ai_extraction_enabled,
		  COALESCE(aa.customer_memory_verifier_enabled, TRUE) AS verifier_enabled,
		  COALESCE(aa.customer_memory_llm_validator_enabled, FALSE) AS llm_validator_enabled,
		  COALESCE(aa.customer_memory_enabled, FALSE) AS admin_notes_enabled,
		  COALESCE(aa.customer_memory_max_items, 5) AS max_items,
		  COALESCE(aa.customer_memory_max_chars, 800) AS max_chars,
		  COALESCE(aa.customer_memory_retention_days, 180) AS retention_days
		FROM conversations c
		JOIN contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
		LEFT JOIN ai_agents aa ON aa.id = c.ai_agent_id AND aa.organization_id = c.organization_id
		WHERE c.id = NULLIF($1, '')::uuid
		  AND c.contact_id = NULLIF($2, '')::uuid
		  AND c.organization_id = $3
	`, conversationID, contactID, s.organizationID(ctx)).Scan(
		&settings.ContactEnabled,
		&settings.Enabled,
		&settings.AutoSaveEnabled,
		&settings.AIExtractionEnabled,
		&settings.VerifierEnabled,
		&settings.LLMValidatorEnabled,
		&settings.AdminNotesEnabled,
		&settings.MaxItems,
		&settings.MaxChars,
		&settings.RetentionDays,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return settings, nil
		}
		return settings, err
	}
	settings.MaxItems = clampInt(settings.MaxItems, 1, 10, 5)
	settings.MaxChars = clampInt(settings.MaxChars, 200, 1200, 800)
	settings.RetentionDays = clampInt(settings.RetentionDays, 30, 730, 180)
	return settings, nil
}

func extractContactMemoryDrafts(text string, maxItems, maxChars int) []contactMemoryDraft {
	maxItems = clampInt(maxItems, 1, 10, 5)
	maxChars = clampInt(maxChars, 200, 1200, 800)
	items := make([]contactMemoryDraft, 0, maxItems)
	seen := map[string]bool{}
	add := func(memoryType, value string, confidence float64) {
		if len(items) >= maxItems {
			return
		}
		value = cleanContactMemoryDraftValue(value, maxChars)
		normalized := normalizeContactMemoryValue(value)
		if value == "" || normalized == "" || isQuestionLikeContactMemoryDraft(text, normalized) || seen[memoryType+"|"+normalized] {
			return
		}
		seen[memoryType+"|"+normalized] = true
		items = append(items, contactMemoryDraft{
			MemoryType:      memoryType,
			Value:           value,
			NormalizedValue: normalized,
			Confidence:      confidence,
		})
	}

	for _, match := range customerCommunicationPreferencePattern.FindAllStringSubmatch(text, -1) {
		if len(match) >= 2 {
			add("communication_preference", "prefer dihubungi "+match[1], 0.78)
		}
	}
	for _, match := range customerPreferencePattern.FindAllStringSubmatch(text, -1) {
		if len(match) >= 3 {
			add("preference", strings.TrimSpace(match[1]+" "+match[2]), 0.76)
		}
	}
	for _, match := range customerInterestPattern.FindAllStringSubmatch(text, -1) {
		if len(match) >= 3 {
			add("interest", strings.TrimSpace(match[1]+" "+match[2]), 0.68)
		}
	}
	for _, match := range customerBudgetPreferencePattern.FindAllStringSubmatch(text, -1) {
		if len(match) >= 2 {
			add("preference", "budget "+match[1], 0.64)
		}
	}
	return items
}

func cleanContactMemoryDraftValue(value string, maxChars int) string {
	value = strings.TrimSpace(value)
	value = customerMemoryTrailingQuestionClausePattern.ReplaceAllString(value, "")
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Trim(value, " \t\r\n.,;:!?\"'`")
	if value == "" {
		return ""
	}
	if maxChars <= 0 || maxChars > 1200 {
		maxChars = 800
	}
	if len(value) > maxChars {
		value = strings.TrimSpace(value[:maxChars])
	}
	return value
}

func isQuestionLikeContactMemoryDraft(rawText, normalizedValue string) bool {
	normalizedValue = strings.TrimSpace(strings.ToLower(normalizedValue))
	if normalizedValue == "" {
		return true
	}

	for _, prefix := range []string{
		"suka ",
		"tidak suka ",
		"nggak suka ",
		"ngga suka ",
		"ga suka ",
		"gak suka ",
		"kurang suka ",
		"lebih suka ",
		"prefer ",
		"favoritnya ",
		"favorit ",
		"tertarik ",
		"minat ",
		"lagi cari ",
		"sedang cari ",
		"cari ",
		"budget ",
	} {
		if strings.HasPrefix(normalizedValue, prefix) {
			remainder := strings.TrimSpace(strings.TrimPrefix(normalizedValue, prefix))
			if remainder == "" || customerMemoryQuestionCuePattern.MatchString(remainder) {
				return true
			}
			break
		}
	}

	rawText = strings.TrimSpace(strings.ToLower(rawText))
	if strings.Contains(rawText, "?") && customerMemoryQuestionCuePattern.MatchString(normalizedValue) {
		return true
	}
	return false
}

func classifyContactMemoryRisk(text string) (string, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "safe", ""
	}
	if customerMemoryPromptInstructionPattern.MatchString(text) {
		return "prompt_instruction", "prompt_instruction"
	}
	if customerMemoryAuthorityClaimPattern.MatchString(text) {
		return "system_action_claim", "authority_claim"
	}
	if customerMemoryBusinessClaimPattern.MatchString(text) {
		return "business_claim", "business_claim"
	}
	if customerMemoryAmountPattern.MatchString(text) {
		return "uncertain", "amount"
	}
	return "safe", ""
}

func (s *Server) activeAISettingsUpdatedAt(ctx context.Context) (time.Time, error) {
	var updatedAt time.Time
	err := s.db.QueryRow(ctx, `SELECT updated_at FROM ai_settings WHERE is_active = TRUE AND organization_id = $1 ORDER BY updated_at DESC LIMIT 1`, s.organizationID(ctx)).Scan(&updatedAt)
	return updatedAt, err
}

func (s *Server) activeFallbackWaitingMessage(ctx context.Context) (string, error) {
	var message string
	if sessionID := whatsAppSessionIDFromContext(ctx); sessionID != "" {
		err := s.db.QueryRow(ctx, `
			SELECT aa.fallback_waiting_message
			FROM whatsapp_sessions ws
			JOIN ai_agents aa ON aa.id = ws.ai_agent_id AND aa.organization_id = ws.organization_id
			WHERE ws.id = NULLIF($1, '')::uuid AND ws.organization_id = NULLIF($2, '')::uuid AND ws.deleted_at IS NULL
			LIMIT 1
		`, sessionID, s.organizationID(ctx)).Scan(&message)
		if err == nil {
			return message, nil
		}
	}
	err := s.db.QueryRow(ctx, `
		SELECT fallback_waiting_message
		FROM ai_settings
		WHERE is_active = TRUE AND organization_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&message)
	return message, err
}

func (s *Server) activeContactNamePolicy(ctx context.Context) (bool, bool, error) {
	var allowAutoUpdate bool
	var onlyFillIfEmpty bool
	if sessionID := whatsAppSessionIDFromContext(ctx); sessionID != "" {
		err := s.db.QueryRow(ctx, `
			SELECT aa.allow_auto_update_contact_name, aa.only_fill_name_if_empty
			FROM whatsapp_sessions ws
			JOIN ai_agents aa ON aa.id = ws.ai_agent_id AND aa.organization_id = ws.organization_id
			WHERE ws.id = NULLIF($1, '')::uuid AND ws.organization_id = NULLIF($2, '')::uuid AND ws.deleted_at IS NULL
			LIMIT 1
		`, sessionID, s.organizationID(ctx)).Scan(&allowAutoUpdate, &onlyFillIfEmpty)
		if err == nil {
			return allowAutoUpdate, onlyFillIfEmpty, nil
		}
	}
	err := s.db.QueryRow(ctx, `
		SELECT allow_auto_update_contact_name, only_fill_name_if_empty
		FROM ai_settings
		WHERE is_active = TRUE AND organization_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&allowAutoUpdate, &onlyFillIfEmpty)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, true, nil
		}
		return false, false, err
	}
	return allowAutoUpdate, onlyFillIfEmpty, nil
}

func (s *Server) fetchOpenRouterSupportedChatModels(ctx context.Context, aliases map[string]string, availableModels map[string]bool, includeUnavailable bool) ([]map[string]any, error) {
	requestCtx, cancel := context.WithTimeout(ctx, openRouterModelsRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, s.cfg.OpenRouterModelsURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("openrouter models returned %d", resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			ContextLimit int    `json:"context_length"`
			Architecture struct {
				InputModalities  []string `json:"input_modalities"`
				OutputModalities []string `json:"output_modalities"`
			} `json:"architecture"`
			Pricing struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	models := []map[string]any{}
	for _, item := range payload.Data {
		id := strings.TrimSpace(item.ID)
		name := strings.TrimSpace(item.Name)
		if !isAllowedOpenRouterChatModel(id, name) || !hasModelModality(item.Architecture.InputModalities, "text") || !hasModelModality(item.Architecture.OutputModalities, "text") {
			continue
		}
		isAvailable := availableModels[strings.ToLower(id)]
		if !includeUnavailable && !isAvailable {
			continue
		}
		promptPrice, _ := strconv.ParseFloat(item.Pricing.Prompt, 64)
		completionPrice, _ := strconv.ParseFloat(item.Pricing.Completion, 64)
		alias := strings.TrimSpace(aliases[strings.ToLower(id)])
		upstreamName := name
		if upstreamName == "" {
			upstreamName = defaultOpenRouterChatModelDisplayName(id)
		}
		models = append(models, map[string]any{
			"id":               id,
			"name":             openRouterChatModelDisplayName(id, name, aliases),
			"alias":            alias,
			"upstreamName":     upstreamName,
			"available":        isAvailable,
			"provider":         openRouterChatModelProvider(id),
			"contextLength":    item.ContextLimit,
			"inputPricePer1m":  promptPrice * 1_000_000,
			"outputPricePer1m": completionPrice * 1_000_000,
			"pricingUnit":      "usd_per_1m_tokens",
			"pricingSource":    "openrouter_models",
			"creditEstimate":   openRouterChatModelCreditEstimate(id),
		})
	}
	sort.Slice(models, func(i, j int) bool {
		leftRank := openRouterChatModelSortIndex(fmt.Sprint(models[i]["id"]))
		rightRank := openRouterChatModelSortIndex(fmt.Sprint(models[j]["id"]))
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return strings.ToLower(fmt.Sprint(models[i]["id"])) < strings.ToLower(fmt.Sprint(models[j]["id"]))
	})
	if len(models) == 0 {
		return nil, fmt.Errorf("no supported OpenRouter chat models returned")
	}
	return models, nil
}

func hasModelModality(values []string, expected string) bool {
	expected = strings.ToLower(strings.TrimSpace(expected))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == expected {
			return true
		}
	}
	return false
}

func isAllowedOpenRouterChatModel(id, name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(id))
	label := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" || len(normalized) > 160 {
		return false
	}
	if strings.HasPrefix(normalized, "~") || strings.Contains(normalized, "latest") {
		return false
	}
	if strings.HasPrefix(normalized, "openai/") {
		return strings.Contains(normalized, "gpt") || strings.Contains(label, "gpt")
	}
	if strings.HasPrefix(normalized, "anthropic/") {
		return strings.Contains(normalized, "claude") || strings.Contains(label, "claude")
	}
	if strings.HasPrefix(normalized, "deepseek/") {
		return strings.Contains(normalized, "deepseek") || strings.Contains(label, "deepseek")
	}
	return false
}

func defaultOpenRouterChatModelAliases() map[string]string {
	return map[string]string{
		"openai/gpt-4o-mini":          "Oneflow.id Basic Model",
		"openai/gpt-4.1-mini":         "Oneflow.id Advance Model",
		"deepseek/deepseek-v4-flash":  "DeepSeek V4 Flash",
		"deepseek/deepseek-v4-pro":    "DeepSeek V4 Pro",
		"deepseek/deepseek-v3.2":      "DeepSeek V3.2",
		"deepseek/deepseek-chat-v3.1": "DeepSeek V3.1",
		"deepseek/deepseek-r1":        "DeepSeek R1",
	}
}

func defaultOpenRouterAvailableChatModelIDs() map[string]bool {
	return map[string]bool{
		"openai/gpt-4o-mini":  true,
		"openai/gpt-4.1-mini": true,
	}
}

func (s *Server) openRouterChatModelAliases(ctx context.Context) map[string]string {
	aliases := defaultOpenRouterChatModelAliases()
	rows, err := s.db.Query(ctx, `SELECT model_id, display_name FROM ai_model_aliases`)
	if err != nil {
		return aliases
	}
	defer rows.Close()
	for rows.Next() {
		var modelID, displayName string
		if err := rows.Scan(&modelID, &displayName); err != nil {
			continue
		}
		modelID = strings.ToLower(strings.TrimSpace(modelID))
		displayName = strings.TrimSpace(displayName)
		if !isAllowedOpenRouterChatModel(modelID, "") || displayName == "" {
			continue
		}
		aliases[modelID] = displayName
	}
	return aliases
}

func (s *Server) openRouterAvailableChatModelIDs(ctx context.Context) map[string]bool {
	rows, err := s.db.Query(ctx, `SELECT model_id, is_available FROM ai_model_aliases`)
	if err != nil {
		return defaultOpenRouterAvailableChatModelIDs()
	}
	defer rows.Close()

	available := map[string]bool{}
	hasRows := false
	for rows.Next() {
		var modelID string
		var isAvailable bool
		if err := rows.Scan(&modelID, &isAvailable); err != nil {
			continue
		}
		hasRows = true
		modelID = strings.ToLower(strings.TrimSpace(modelID))
		if isAvailable && isAllowedOpenRouterChatModel(modelID, "") {
			available[modelID] = true
		}
	}
	if !hasRows || len(available) == 0 {
		return defaultOpenRouterAvailableChatModelIDs()
	}
	return available
}

func openRouterAvailableChatModelIDList(available map[string]bool) []string {
	ids := make([]string, 0, len(available))
	for modelID, isAvailable := range available {
		if isAvailable && isAllowedOpenRouterChatModel(modelID, "") {
			ids = append(ids, modelID)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		leftRank := openRouterChatModelSortIndex(ids[i])
		rightRank := openRouterChatModelSortIndex(ids[j])
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return ids[i] < ids[j]
	})
	return ids
}

func (s *Server) upsertOpenRouterChatModelAliasesTx(ctx context.Context, tx pgx.Tx, aliases map[string]string) error {
	for modelID, displayName := range aliases {
		modelID = strings.ToLower(strings.TrimSpace(modelID))
		displayName = strings.TrimSpace(displayName)
		if !isAllowedOpenRouterChatModel(modelID, "") || displayName == "" {
			continue
		}
		if len([]rune(displayName)) > 80 {
			displayName = string([]rune(displayName)[:80])
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO ai_model_aliases (model_id, display_name, updated_by, updated_at)
			VALUES ($1, $2, NULLIF($3, '')::uuid, NOW())
			ON CONFLICT (model_id) DO UPDATE
			SET display_name = EXCLUDED.display_name,
			    updated_by = EXCLUDED.updated_by,
			    updated_at = NOW()
		`, modelID, displayName, s.agentID(ctx)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) replaceOpenRouterAvailableChatModelsTx(ctx context.Context, tx pgx.Tx, availableModelIDs []string, activeModelID string) ([]string, error) {
	available := map[string]bool{}
	for _, modelID := range availableModelIDs {
		modelID = strings.ToLower(strings.TrimSpace(modelID))
		if isAllowedOpenRouterChatModel(modelID, "") {
			available[modelID] = true
		}
	}
	activeModelID = strings.ToLower(strings.TrimSpace(activeModelID))
	if isAllowedOpenRouterChatModel(activeModelID, "") {
		available[activeModelID] = true
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("at least one available chat model is required")
	}
	if _, err := tx.Exec(ctx, `
		UPDATE ai_model_aliases
		SET is_available = FALSE,
		    updated_by = NULLIF($1, '')::uuid,
		    updated_at = NOW()
	`, s.agentID(ctx)); err != nil {
		return nil, err
	}
	defaultAliases := defaultOpenRouterChatModelAliases()
	for modelID := range available {
		if _, err := tx.Exec(ctx, `
			INSERT INTO ai_model_aliases (model_id, display_name, is_available, updated_by, updated_at)
			VALUES ($1, $2, TRUE, NULLIF($3, '')::uuid, NOW())
			ON CONFLICT (model_id) DO UPDATE
			SET is_available = TRUE,
			    updated_by = EXCLUDED.updated_by,
			    updated_at = NOW()
		`, modelID, openRouterChatModelDisplayName(modelID, "", defaultAliases), s.agentID(ctx)); err != nil {
			return nil, err
		}
	}
	return openRouterAvailableChatModelIDList(available), nil
}

func openRouterChatModelDisplayName(id, upstreamName string, aliases map[string]string) string {
	normalized := strings.ToLower(strings.TrimSpace(id))
	if alias := strings.TrimSpace(aliases[normalized]); alias != "" {
		return alias
	}
	if upstreamName = strings.TrimSpace(upstreamName); upstreamName != "" {
		return upstreamName
	}
	return defaultOpenRouterChatModelDisplayName(id)
}

func defaultOpenRouterChatModelDisplayName(id string) string {
	normalized := strings.ToLower(strings.TrimSpace(id))
	switch normalized {
	case "openai/gpt-4o-mini":
		return "Oneflow.id Basic Model"
	case "openai/gpt-4.1-mini":
		return "Oneflow.id Advance Model"
	default:
		clean := strings.TrimPrefix(normalized, "~")
		if parts := strings.SplitN(clean, "/", 2); len(parts) == 2 {
			return parts[1]
		}
		return strings.TrimSpace(id)
	}
}

func openRouterChatModelDefaultPrices(id string) (float64, float64, bool) {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "openai/gpt-4o-mini":
		return 0.15, 0.60, true
	case "openai/gpt-4.1-mini":
		return 0.40, 1.60, true
	case "anthropic/claude-haiku-4.5":
		return 1.00, 5.00, true
	case "deepseek/deepseek-v4-flash":
		return 0.10, 0.20, true
	case "deepseek/deepseek-v4-pro":
		return 0.435, 0.87, true
	case "deepseek/deepseek-v3.2":
		return 0.252, 0.378, true
	case "deepseek/deepseek-chat-v3.1":
		return 0.21, 0.79, true
	case "deepseek/deepseek-r1":
		return 0.70, 2.50, true
	default:
		return 0, 0, false
	}
}

func openRouterChatModelCreditEstimate(id string) map[string]int {
	normalized := strings.ToLower(strings.TrimSpace(id))
	if strings.Contains(normalized, "opus") || strings.Contains(normalized, "pro") {
		return map[string]int{
			"averageAnswer": 6,
			"complexAnswer": 12,
			"imageAnswer":   16,
			"escalation":    6,
		}
	}
	if strings.Contains(normalized, "sonnet") || strings.Contains(normalized, "gpt-5") || strings.Contains(normalized, "gpt-4.1") || strings.Contains(normalized, "deepseek") {
		return map[string]int{
			"averageAnswer": 3,
			"complexAnswer": 6,
			"imageAnswer":   8,
			"escalation":    3,
		}
	}
	return map[string]int{
		"averageAnswer": 1,
		"complexAnswer": 2,
		"imageAnswer":   3,
		"escalation":    1,
	}
}

func openRouterChatModelProvider(id string) string {
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(id)), "~")
	if parts := strings.SplitN(normalized, "/", 2); len(parts) == 2 {
		if parts[0] == "anthropic" {
			return "Claude"
		}
		if parts[0] == "openai" {
			return "GPT"
		}
		if parts[0] == "deepseek" {
			return "DeepSeek"
		}
		return parts[0]
	}
	return "OpenRouter"
}

func openRouterChatModelSortIndex(id string) int {
	normalized := strings.ToLower(strings.TrimSpace(id))
	switch normalized {
	case "openai/gpt-4o-mini":
		return 0
	case "openai/gpt-4.1-mini":
		return 1
	}
	if strings.Contains(normalized, "/gpt") {
		return 30
	}
	if strings.Contains(normalized, "deepseek/") || strings.Contains(normalized, "/deepseek") {
		return 35
	}
	if strings.Contains(normalized, "/claude") {
		return 40
	}
	return 99
}

func fallbackOpenRouterChatModels(aliases map[string]string, availableModels map[string]bool, includeUnavailable bool) []map[string]any {
	fallbackIDs := []string{
		"openai/gpt-4o-mini",
		"openai/gpt-4.1-mini",
		"deepseek/deepseek-v4-flash",
		"deepseek/deepseek-v4-pro",
		"deepseek/deepseek-v3.2",
		"deepseek/deepseek-chat-v3.1",
		"deepseek/deepseek-r1",
	}
	models := make([]map[string]any, 0, len(fallbackIDs))
	for _, id := range fallbackIDs {
		isAvailable := availableModels[strings.ToLower(id)]
		if !includeUnavailable && !isAvailable {
			continue
		}
		input, output, _ := openRouterChatModelDefaultPrices(id)
		alias := strings.TrimSpace(aliases[strings.ToLower(id)])
		models = append(models, map[string]any{
			"id":               id,
			"name":             openRouterChatModelDisplayName(id, "", aliases),
			"alias":            alias,
			"upstreamName":     defaultOpenRouterChatModelDisplayName(id),
			"available":        isAvailable,
			"provider":         openRouterChatModelProvider(id),
			"inputPricePer1m":  input,
			"outputPricePer1m": output,
			"pricingSource":    "fallback",
			"creditEstimate":   openRouterChatModelCreditEstimate(id),
		})
	}
	return models
}

func isAllowedOpenAIChatModel(id string) bool {
	return isAllowedOpenRouterChatModel(id, "")
}

func defaultOpenAIChatModelAliases() map[string]string {
	return defaultOpenRouterChatModelAliases()
}

func openAIChatModelDisplayName(id string, aliases map[string]string) string {
	return openRouterChatModelDisplayName(id, "", aliases)
}

func openAIChatModelDefaultPrices(id string) (float64, float64, bool) {
	return openRouterChatModelDefaultPrices(id)
}

func openAIChatModelCreditEstimate(id string) map[string]int {
	return openRouterChatModelCreditEstimate(id)
}

func openAIChatModelSortIndex(id string) int {
	return openRouterChatModelSortIndex(id)
}

func fallbackOpenAIModels(aliases map[string]string) []map[string]any {
	return fallbackOpenRouterChatModels(aliases, defaultOpenRouterAvailableChatModelIDs(), true)
}

func shouldApplyContactName(currentName, customerName string, allowAutoUpdate, onlyFillIfEmpty bool) bool {
	customerName = strings.TrimSpace(customerName)
	currentName = strings.TrimSpace(currentName)
	if customerName == "" || !allowAutoUpdate {
		return false
	}
	if onlyFillIfEmpty {
		return currentName == ""
	}
	return currentName != customerName
}

func (s *Server) ensurePlaygroundConversation(ctx context.Context, customerName string) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var contactID string
	contactName := strings.TrimSpace(customerName)
	if contactName == "" {
		contactName = "Pelanggan Test"
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO contacts (phone, name, notes, organization_id)
		VALUES ('__playground__', $2, 'internal dashboard playground logging anchor', $1)
		ON CONFLICT (organization_id, phone) DO UPDATE
		SET name = EXCLUDED.name,
		    notes = EXCLUDED.notes,
		    updated_at = NOW()
		RETURNING id
	`, s.organizationID(ctx), contactName).Scan(&contactID)
	if err != nil {
		return "", err
	}

	var conversationID string
	err = tx.QueryRow(ctx, `
		INSERT INTO conversations (contact_id, channel, mode, status, last_message_at, organization_id)
		SELECT $1, 'playground', 'ai', 'resolved', NOW(), $2
		WHERE NOT EXISTS (
		  SELECT 1
		  FROM conversations
		  WHERE contact_id = $1 AND organization_id = $2
		)
		RETURNING id
	`, contactID, s.organizationID(ctx)).Scan(&conversationID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
		err = tx.QueryRow(ctx, `
			UPDATE conversations
			SET channel = 'playground',
			    updated_at = NOW()
			WHERE contact_id = $1 AND organization_id = $2
			RETURNING id
		`, contactID, s.organizationID(ctx)).Scan(&conversationID)
		if err != nil {
			return "", err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return conversationID, nil
}

type contactLookupResult struct {
	ID    string
	Phone string
	Name  string
	Found bool
}

func (s *Server) resolveWhatsAppInboundPhoneTx(ctx context.Context, tx pgx.Tx, rawPhone string) (string, map[string]any, error) {
	normalizedPhone, digits := normalizeWhatsAppPhoneValue(rawPhone)
	payload := map[string]any{
		"rawPhone":        strings.TrimSpace(rawPhone),
		"normalizedPhone": normalizedPhone,
		"matchedLidMap":   false,
		"identityKind":    "unknown",
		"canonicalPhone":  normalizedPhone,
		"canonicalSource": "incoming",
	}
	if digits == "" {
		return normalizedPhone, payload, nil
	}

	var lid, pn string
	err := tx.QueryRow(ctx, `
		SELECT lid, pn
		FROM whatsmeow_lid_map
		WHERE lid = $1 OR pn = $1
		LIMIT 1
	`, digits).Scan(&lid, &pn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			payload["identityKind"] = "phone_or_unmapped_id"
			return normalizedPhone, payload, nil
		}
		return "", nil, err
	}

	canonicalPhone := "+" + pn
	identityKind := "phone_number"
	if digits == lid {
		identityKind = "lid"
	}
	payload["matchedLidMap"] = true
	payload["identityKind"] = identityKind
	payload["lid"] = lid
	payload["phoneNumber"] = pn
	payload["canonicalPhone"] = canonicalPhone
	payload["canonicalSource"] = "whatsmeow_lid_map"
	return canonicalPhone, payload, nil
}

func (s *Server) resolveWhatsAppContactTx(ctx context.Context, tx pgx.Tx, canonicalPhone, rawPhone, customerName string, allowAutoUpdate, onlyFillNameIfEmpty bool) (string, string, error) {
	canonicalPhone, _ = normalizeWhatsAppPhoneValue(canonicalPhone)
	rawNormalizedPhone, _ := normalizeWhatsAppPhoneValue(rawPhone)

	canonicalContact, err := lookupContactByPhoneTx(ctx, tx, canonicalPhone, s.organizationID(ctx))
	if err != nil {
		return "", "", err
	}
	rawContact := contactLookupResult{}
	if rawNormalizedPhone != "" && rawNormalizedPhone != canonicalPhone {
		rawContact, err = lookupContactByPhoneTx(ctx, tx, rawNormalizedPhone, s.organizationID(ctx))
		if err != nil {
			return "", "", err
		}
	}

	if canonicalContact.Found {
		if rawContact.Found && rawContact.ID != canonicalContact.ID {
			if err := mergeWhatsAppContactsTx(ctx, tx, canonicalContact, rawContact); err != nil {
				return "", "", err
			}
			if canonicalContact.Name == "" && rawContact.Name != "" {
				canonicalContact.Name = rawContact.Name
			}
		}
		return applyResolvedContactNameTx(ctx, tx, canonicalContact.ID, canonicalContact.Name, customerName, allowAutoUpdate, onlyFillNameIfEmpty)
	}

	if rawContact.Found {
		if _, err := tx.Exec(ctx, `
			UPDATE contacts
			SET phone = $2, updated_at = NOW()
			WHERE id = $1 AND organization_id = $3
		`, rawContact.ID, canonicalPhone, s.organizationID(ctx)); err != nil {
			return "", "", err
		}
		return applyResolvedContactNameTx(ctx, tx, rawContact.ID, rawContact.Name, customerName, allowAutoUpdate, onlyFillNameIfEmpty)
	}

	contactNameToSave := ""
	if shouldApplyContactName("", customerName, allowAutoUpdate, onlyFillNameIfEmpty) {
		contactNameToSave = customerName
	}
	var contactID, currentName string
	err = tx.QueryRow(ctx, `
		INSERT INTO contacts (phone, name, organization_id)
		VALUES ($1, NULLIF($2, ''), $3)
		RETURNING id, COALESCE(name, '')
	`, canonicalPhone, contactNameToSave, s.organizationID(ctx)).Scan(&contactID, &currentName)
	return contactID, currentName, err
}

func lookupContactByPhoneTx(ctx context.Context, tx pgx.Tx, phone string, organizationID string) (contactLookupResult, error) {
	if strings.TrimSpace(phone) == "" {
		return contactLookupResult{}, nil
	}
	var result contactLookupResult
	err := tx.QueryRow(ctx, `
		SELECT id, phone, COALESCE(name, '')
		FROM contacts
		WHERE phone = $1 AND organization_id = $2
		FOR UPDATE
	`, phone, organizationID).Scan(&result.ID, &result.Phone, &result.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return contactLookupResult{}, nil
		}
		return contactLookupResult{}, err
	}
	result.Found = true
	return result, nil
}

func applyResolvedContactNameTx(ctx context.Context, tx pgx.Tx, contactID, currentName, customerName string, allowAutoUpdate, onlyFillNameIfEmpty bool) (string, string, error) {
	if shouldApplyContactName(currentName, customerName, allowAutoUpdate, onlyFillNameIfEmpty) {
		if _, err := tx.Exec(ctx, `UPDATE contacts SET name = $2, updated_at = NOW() WHERE id = $1 AND organization_id = $3`, contactID, customerName, organizationIDFromContext(ctx)); err != nil {
			return "", "", err
		}
		currentName = customerName
	}
	return contactID, currentName, nil
}

func mergeWhatsAppContactsTx(ctx context.Context, tx pgx.Tx, target, source contactLookupResult) error {
	if target.ID == "" || source.ID == "" || target.ID == source.ID {
		return nil
	}
	if target.Name == "" && source.Name != "" {
		if _, err := tx.Exec(ctx, `UPDATE contacts SET name = $2, updated_at = NOW() WHERE id = $1 AND organization_id = $3`, target.ID, source.Name, organizationIDFromContext(ctx)); err != nil {
			return err
		}
	}

	targetConversationID, targetHasConversation, err := contactConversationIDTx(ctx, tx, target.ID)
	if err != nil {
		return err
	}
	sourceConversationID, sourceHasConversation, err := contactConversationIDTx(ctx, tx, source.ID)
	if err != nil {
		return err
	}

	if sourceHasConversation {
		if targetHasConversation {
			if _, err := tx.Exec(ctx, `UPDATE messages SET conversation_id = $1 WHERE conversation_id = $2 AND organization_id = $3`, targetConversationID, sourceConversationID, organizationIDFromContext(ctx)); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE conversation_events SET conversation_id = $1 WHERE conversation_id = $2 AND organization_id = $3`, targetConversationID, sourceConversationID, organizationIDFromContext(ctx)); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE ai_runs SET conversation_id = $1 WHERE conversation_id = $2 AND organization_id = $3`, targetConversationID, sourceConversationID, organizationIDFromContext(ctx)); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE credit_usage_logs SET conversation_id = $1 WHERE conversation_id = $2 AND organization_id = $3`, targetConversationID, sourceConversationID, organizationIDFromContext(ctx)); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM conversations WHERE id = $1 AND organization_id = $2`, sourceConversationID, organizationIDFromContext(ctx)); err != nil {
				return err
			}
			if err := refreshConversationLastMessageTx(ctx, tx, targetConversationID); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(ctx, `
				UPDATE conversations
				SET contact_id = $2, updated_at = NOW()
				WHERE id = $1 AND organization_id = $3
			`, sourceConversationID, target.ID, organizationIDFromContext(ctx)); err != nil {
				return err
			}
		}
	}

	_, err = tx.Exec(ctx, `DELETE FROM contacts WHERE id = $1 AND organization_id = $2`, source.ID, organizationIDFromContext(ctx))
	return err
}

func contactConversationIDTx(ctx context.Context, tx pgx.Tx, contactID string) (string, bool, error) {
	var conversationID string
	err := tx.QueryRow(ctx, `SELECT id FROM conversations WHERE contact_id = $1 AND organization_id = $2 FOR UPDATE`, contactID, organizationIDFromContext(ctx)).Scan(&conversationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return conversationID, true, nil
}

func refreshConversationLastMessageTx(ctx context.Context, tx pgx.Tx, conversationID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE conversations c
		SET last_message_id = latest.id,
		    last_message_at = COALESCE(latest.sent_at, latest.created_at),
		    updated_at = NOW()
		FROM (
			SELECT id, sent_at, created_at
			FROM messages
			WHERE conversation_id = $1 AND organization_id = $2
			ORDER BY COALESCE(sent_at, created_at) DESC
			LIMIT 1
		) latest
		WHERE c.id = $1 AND c.organization_id = $2
	`, conversationID, organizationIDFromContext(ctx))
	return err
}

func normalizeWhatsAppPhoneValue(value string) (string, string) {
	digits := digitsOnly(value)
	if digits == "" {
		return strings.TrimSpace(value), ""
	}
	return "+" + digits, digits
}

func digitsOnly(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func (s *Server) activeCreditPricing(ctx context.Context) (creditPricingSnapshot, error) {
	var snapshot creditPricingSnapshot
	err := s.db.QueryRow(ctx, `
		SELECT credit_unit_idr, usd_to_idr_rate, chat_input_price_per_1m, chat_output_price_per_1m
		FROM credit_pricing_settings
		WHERE is_active = TRUE AND organization_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&snapshot.CreditUnitIDR, &snapshot.USDToIDRRate, &snapshot.InputPrice, &snapshot.OutputPrice)
	return snapshot, err
}

func (s *Server) activeCreditPricingTx(ctx context.Context, tx pgx.Tx) (creditPricingSnapshot, error) {
	var snapshot creditPricingSnapshot
	err := tx.QueryRow(ctx, `
		SELECT credit_unit_idr, usd_to_idr_rate, chat_input_price_per_1m, chat_output_price_per_1m
		FROM credit_pricing_settings
		WHERE is_active = TRUE AND organization_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&snapshot.CreditUnitIDR, &snapshot.USDToIDRRate, &snapshot.InputPrice, &snapshot.OutputPrice)
	return snapshot, err
}

func estimateCreditUsage(snapshot creditPricingSnapshot, inputTokens, outputTokens int) creditEstimate {
	costUSD := (float64(inputTokens)/1_000_000.0)*snapshot.InputPrice + (float64(outputTokens)/1_000_000.0)*snapshot.OutputPrice
	return creditUsageFromCostUSD(snapshot, costUSD)
}

func creditUsageFromCostUSD(snapshot creditPricingSnapshot, costUSD float64) creditEstimate {
	if costUSD < 0 {
		costUSD = 0
	}
	costIDR := costUSD * snapshot.USDToIDRRate
	creditsUsed := int(math.Ceil(costIDR / snapshot.CreditUnitIDR))
	if creditsUsed < 1 {
		creditsUsed = 1
	}
	return creditEstimate{
		CostUSD:     costUSD,
		CostIDR:     costIDR,
		CreditsUsed: creditsUsed,
	}
}

func (s *Server) ensureCreditsAvailable(ctx context.Context, inputTokens, outputTokens int) (bool, int, error) {
	snapshot, err := s.activeCreditPricing(ctx)
	if err != nil {
		return false, 0, err
	}
	estimate := estimateCreditUsage(snapshot, inputTokens, outputTokens)

	var monthlyRemaining, additionalRemaining int
	err = s.db.QueryRow(ctx, `
		SELECT monthly_credits_remaining, additional_credits_remaining
		FROM credit_wallet
		WHERE is_active = TRUE AND organization_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&monthlyRemaining, &additionalRemaining)
	if err != nil {
		return false, 0, err
	}
	return monthlyRemaining+additionalRemaining >= estimate.CreditsUsed, estimate.CreditsUsed, nil
}

func (s *Server) logCreditUsage(ctx context.Context, conversationID, messageID, aiRunID, usageType, modelName string, inputTokens, outputTokens, embeddingTokens int, actualCostUSD *float64) (creditUsageResult, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return creditUsageResult{}, err
	}
	defer tx.Rollback(ctx)

	result, err := s.logCreditUsageTx(ctx, tx, conversationID, messageID, aiRunID, usageType, modelName, inputTokens, outputTokens, embeddingTokens, actualCostUSD)
	if err != nil {
		return creditUsageResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return creditUsageResult{}, err
	}
	return result, nil
}

func (s *Server) logCreditUsageTx(ctx context.Context, tx pgx.Tx, conversationID, messageID, aiRunID, usageType, modelName string, inputTokens, outputTokens, embeddingTokens int, actualCostUSD *float64) (creditUsageResult, error) {
	snapshot, err := s.activeCreditPricingTx(ctx, tx)
	if err != nil {
		return creditUsageResult{}, err
	}
	estimate := estimateCreditUsage(snapshot, inputTokens, outputTokens)
	costNote := "estimated"
	if actualCostUSD != nil {
		estimate = creditUsageFromCostUSD(snapshot, *actualCostUSD)
		costNote = "openrouter_actual"
	}

	var walletID string
	var monthlyUsed, monthlyRemaining, additionalRemaining int
	err = tx.QueryRow(ctx, `
		SELECT id, monthly_credits_used, monthly_credits_remaining, additional_credits_remaining
		FROM credit_wallet
		WHERE is_active = TRUE AND organization_id = $1
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE
	`, s.organizationID(ctx)).Scan(&walletID, &monthlyUsed, &monthlyRemaining, &additionalRemaining)
	if err != nil {
		return creditUsageResult{}, err
	}
	if monthlyRemaining+additionalRemaining < estimate.CreditsUsed {
		return creditUsageResult{}, fmt.Errorf("insufficient AI credits")
	}

	monthlyConsumed := estimate.CreditsUsed
	if monthlyConsumed > monthlyRemaining {
		monthlyConsumed = monthlyRemaining
	}
	additionalConsumed := estimate.CreditsUsed - monthlyConsumed

	creditSource := "monthly"
	switch {
	case monthlyConsumed == 0 && additionalConsumed > 0:
		creditSource = "additional"
	case monthlyConsumed > 0 && additionalConsumed > 0:
		creditSource = "mixed"
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO credit_usage_logs (
		  conversation_id,
		  message_id,
		  ai_run_id,
		  usage_type,
		  credit_source,
		  model_name,
		  input_tokens,
		  output_tokens,
		  embedding_tokens,
		  cost_usd,
		  cost_idr,
		  credits_used,
		  credit_unit_idr_snapshot,
		  usd_to_idr_rate_snapshot,
		  notes,
		  organization_id
		)
		VALUES (
		  NULLIF($1, '')::uuid,
		  NULLIF($2, '')::uuid,
		  NULLIF($3, '')::uuid,
		  $4,
		  $5,
		  $6,
		  $7,
		  $8,
		  $9,
		  $10,
		  $11,
		  $12,
		  $13,
		  $14,
		  $15,
		  $16
		)
	`,
		conversationID,
		messageID,
		aiRunID,
		usageType,
		creditSource,
		modelName,
		inputTokens,
		outputTokens,
		embeddingTokens,
		estimate.CostUSD,
		estimate.CostIDR,
		estimate.CreditsUsed,
		snapshot.CreditUnitIDR,
		snapshot.USDToIDRRate,
		fmt.Sprintf("usage=%s monthly=%d additional=%d cost=%s", usageType, monthlyConsumed, additionalConsumed, costNote),
		s.organizationID(ctx),
	)
	if err != nil {
		return creditUsageResult{}, err
	}

	nextMonthlyRemaining := monthlyRemaining - monthlyConsumed
	nextAdditionalRemaining := additionalRemaining - additionalConsumed
	_, err = tx.Exec(ctx, `
		UPDATE credit_wallet
		SET monthly_credits_used = $2,
		    monthly_credits_remaining = $3,
		    additional_credits_remaining = $4,
		    updated_at = NOW()
		WHERE id = $1 AND organization_id = $5
	`, walletID, monthlyUsed+monthlyConsumed, nextMonthlyRemaining, nextAdditionalRemaining, s.organizationID(ctx))
	if err != nil {
		return creditUsageResult{}, err
	}
	return creditUsageResult{
		creditEstimate:      estimate,
		CreditSource:        creditSource,
		MonthlyConsumed:     monthlyConsumed,
		AdditionalConsumed:  additionalConsumed,
		MonthlyRemaining:    nextMonthlyRemaining,
		AdditionalRemaining: nextAdditionalRemaining,
	}, nil
}

func usageTokensFromDecision(decision aiDecisionResponse, fallbackInput, fallbackOutput, fallbackEmbedding int) (int, int, int, *float64) {
	inputTokens := fallbackInput
	outputTokens := fallbackOutput
	embeddingTokens := fallbackEmbedding
	var actualCostUSD *float64
	if value, ok := jsonNumberToInt(decision.UsageMetadata["input_tokens"]); ok && value >= 0 {
		inputTokens = value
	}
	if value, ok := jsonNumberToInt(decision.UsageMetadata["output_tokens"]); ok && value >= 0 {
		outputTokens = value
	}
	if value, ok := jsonNumberToInt(decision.UsageMetadata["embedding_tokens"]); ok && value >= 0 {
		embeddingTokens = value
	}
	if value, ok := jsonNumberToFloat(decision.UsageMetadata["cost_usd"]); ok && value >= 0 {
		actualCostUSD = &value
	}
	return inputTokens, outputTokens, embeddingTokens, actualCostUSD
}

func shouldAutoResolveAIConversation(decision aiDecisionResponse) bool {
	if decision.Decision != "answer" {
		return false
	}
	if metadataQueryType(decision.RetrievalMetadata) == "conversation_end" {
		return true
	}
	answer := strings.ToLower(strings.TrimSpace(decision.AnswerText))
	if answer == "" {
		return false
	}
	closingHints := []string{
		"sama-sama",
		"terima kasih",
		"semoga sukses",
		"mohon tidak membalas",
	}
	matchCount := 0
	for _, hint := range closingHints {
		if strings.Contains(answer, hint) {
			matchCount++
		}
	}
	return matchCount >= 2
}

func metadataQueryType(metadata map[string]any) string {
	for _, path := range [][]string{
		{"orchestrator", "queryType"},
		{"generation", "queryType"},
		{"intentAnalyzer", "queryType"},
	} {
		if value := nestedString(metadata, path...); value != "" {
			return value
		}
	}
	return ""
}

func nestedString(root map[string]any, path ...string) string {
	var current any = root
	for _, key := range path {
		asMap, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = asMap[key]
	}
	return strings.TrimSpace(fmt.Sprint(current))
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func jsonNumberToInt(value any) (int, bool) {
	switch parsed := value.(type) {
	case float64:
		return int(parsed), true
	case float32:
		return int(parsed), true
	case int:
		return parsed, true
	case int32:
		return int(parsed), true
	case int64:
		return int(parsed), true
	case json.Number:
		v, err := parsed.Int64()
		if err != nil {
			return 0, false
		}
		return int(v), true
	default:
		return 0, false
	}
}

func jsonNumberToFloat(value any) (float64, bool) {
	switch parsed := value.(type) {
	case float64:
		return parsed, true
	case float32:
		return float64(parsed), true
	case int:
		return float64(parsed), true
	case int32:
		return float64(parsed), true
	case int64:
		return float64(parsed), true
	case json.Number:
		v, err := parsed.Float64()
		if err != nil {
			return 0, false
		}
		return v, true
	default:
		return 0, false
	}
}

func analyticsWindow(rangeValue string) (string, time.Time, string) {
	now := time.Now().UTC()
	switch strings.ToLower(strings.TrimSpace(rangeValue)) {
	case "1d", "daily":
		return "daily", now.Add(-24 * time.Hour), "hour"
	case "5d", "five_day", "five-day":
		return "five_day", now.Add(-5 * 24 * time.Hour), "day"
	case "weekly":
		return "weekly", now.Add(-7 * 24 * time.Hour), "day"
	case "1m", "monthly":
		return "monthly", now.Add(-30 * 24 * time.Hour), "day"
	case "1y", "yearly":
		return "yearly", now.AddDate(-1, 0, 0), "month"
	case "5y", "five_year", "five-year":
		return "five_year", now.AddDate(-5, 0, 0), "month"
	case "max":
		return "max", time.Unix(0, 0).UTC(), "year"
	default:
		return "monthly", now.Add(-30 * 24 * time.Hour), "day"
	}
}

func analyticsBucketLabel(bucket string, value time.Time) string {
	if bucket == "hour" {
		return value.UTC().Format("02 Jan 15:00")
	}
	if bucket == "month" {
		return value.UTC().Format("Jan 2006")
	}
	if bucket == "year" {
		return value.UTC().Format("2006")
	}
	return value.UTC().Format("02 Jan")
}

func safeObjectKey(prefix, originalName string) string {
	ext := strings.ToLower(filepath.Ext(originalName))
	base := strings.TrimSuffix(filepath.Base(originalName), filepath.Ext(originalName))
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '-'
		}
	}, base)
	base = strings.Trim(base, "-")
	if base == "" {
		base = "document"
	}
	return fmt.Sprintf("%s/%d-%s%s", prefix, time.Now().UnixNano(), base, ext)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func marshalJSON(payload any) string {
	body, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(body)
}

func ListenAndServe(cfg config.Config, db *pgxpool.Pool) error {
	storageClient, err := storage.New(context.Background(), cfg)
	if err != nil {
		return err
	}
	server := New(cfg, db, storageClient)
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.AppPort),
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return httpServer.ListenAndServe()
}
