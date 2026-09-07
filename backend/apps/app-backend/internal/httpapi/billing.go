package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	errUnsupportedCheckoutPeriod     = errors.New("unsupported checkout billing period")
	errCheckoutBillingPeriodMismatch = errors.New("checkout billing period does not match package")
)

func resolveCheckoutBillingPeriod(requestedPeriod, packagePeriod string) (string, error) {
	actual := strings.ToLower(strings.TrimSpace(packagePeriod))
	if actual == "" {
		actual = "monthly"
	}
	if actual != "monthly" && actual != "one_time" {
		return "", errUnsupportedCheckoutPeriod
	}

	requested := strings.ToLower(strings.TrimSpace(requestedPeriod))
	if requested == "" {
		return actual, nil
	}
	if requested == "yearly" {
		requested = "annual"
	}
	if actual == "monthly" && (requested == "monthly" || requested == "annual") {
		return requested, nil
	}
	if actual == "one_time" && requested == "one_time" {
		return requested, nil
	}
	if requested == "monthly" || requested == "annual" || requested == "one_time" {
		return "", errCheckoutBillingPeriodMismatch
	}
	return "", errUnsupportedCheckoutPeriod
}

func annualCheckoutAmount(monthlyPrice, annualDiscountPercent float64) float64 {
	discount := math.Max(0, math.Min(99.99, annualDiscountPercent))
	return math.Round(monthlyPrice * 12 * (1 - discount/100))
}

func (s *Server) activeAnnualDiscountPercent(ctx context.Context, organizationID string) (float64, error) {
	var annualDiscountPercent float64
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE((
			SELECT annual_discount_percent
			FROM credit_pricing_settings
			WHERE organization_id = $1
			  AND is_active = TRUE
			ORDER BY updated_at DESC
			LIMIT 1
		), 10)
	`, organizationID).Scan(&annualDiscountPercent)
	return annualDiscountPercent, err
}

func (s *Server) handleBillingWallet(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var payload struct {
		MonthlyCreditLimit      int        `json:"monthlyCreditLimit"`
		MonthlyCreditsUsed      int        `json:"monthlyCreditsUsed"`
		MonthlyCreditsRemaining int        `json:"monthlyCreditsRemaining"`
		AdditionalCredits       int        `json:"additionalCreditsRemaining"`
		LastResetAt             *time.Time `json:"lastResetAt"`
		NextResetAt             *time.Time `json:"nextResetAt"`
	}
	query := `
		SELECT
		  monthly_credit_limit,
		  monthly_credits_used,
		  monthly_credits_remaining,
		  additional_credits_remaining,
		  last_reset_at,
		  next_reset_at
		FROM credit_wallet
		WHERE is_active = TRUE AND organization_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`
	args := []any{s.organizationID(r.Context())}
	if role == "owner" {
		query = `
			SELECT
			  COALESCE(SUM(monthly_credit_limit), 0),
			  COALESCE(SUM(monthly_credits_used), 0),
			  COALESCE(SUM(monthly_credits_remaining), 0),
			  COALESCE(SUM(additional_credits_remaining), 0),
			  MIN(last_reset_at),
			  MAX(next_reset_at)
			FROM credit_wallet
			WHERE is_active = TRUE
		`
		args = nil
	}
	err := s.db.QueryRow(r.Context(), query, args...).Scan(
		&payload.MonthlyCreditLimit,
		&payload.MonthlyCreditsUsed,
		&payload.MonthlyCreditsRemaining,
		&payload.AdditionalCredits,
		&payload.LastResetAt,
		&payload.NextResetAt,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	plan, _ := s.activeOrganizationPlanLimits(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"wallet": payload, "plan": plan})
}

func (s *Server) handleBillingUsageLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	limit := 100
	if parsed, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && parsed > 0 && parsed <= 500 {
		limit = parsed
	}

	query := `
		SELECT
		  l.id,
		  COALESCE(l.conversation_id::text, ''),
		  COALESCE(l.message_id::text, ''),
		  COALESCE(l.ai_run_id::text, ''),
		  l.usage_type,
		  l.credit_source,
		  COALESCE(NULLIF(l.model_name, ''), ar.model_name, ''),
		  COALESCE(l.input_tokens, 0),
		  COALESCE(l.output_tokens, 0),
		  COALESCE(l.embedding_tokens, 0),
		  COALESCE(l.input_tokens, 0) + COALESCE(l.output_tokens, 0) + COALESCE(l.embedding_tokens, 0),
		  COALESCE(l.cost_idr, 0),
		  l.credits_used,
		  l.credit_unit_idr_snapshot,
		  l.usd_to_idr_rate_snapshot,
		  COALESCE(l.notes, ''),
		  COALESCE(ar.decision, ''),
		  COALESCE(ar.question_summary, ''),
		  COALESCE(ar.latency_ms, 0),
		  COALESCE(NULLIF(ct.name, ''), ''),
		  COALESCE(ct.phone, ''),
		  COALESCE(o.name, ''),
		  l.created_at
		FROM credit_usage_logs l
		LEFT JOIN ai_runs ar ON ar.id = l.ai_run_id AND ar.organization_id = l.organization_id
		LEFT JOIN conversations c ON c.id = l.conversation_id AND c.organization_id = l.organization_id
		LEFT JOIN contacts ct ON ct.id = c.contact_id AND ct.organization_id = l.organization_id
		LEFT JOIN organizations o ON o.id = l.organization_id
		WHERE l.organization_id = $1
		ORDER BY l.created_at DESC
		LIMIT $2
	`
	args := []any{s.organizationID(r.Context()), limit}
	if role == "owner" {
		query = strings.Replace(query, "WHERE l.organization_id = $1", "", 1)
		query = strings.Replace(query, "LIMIT $2", "LIMIT $1", 1)
		args = []any{limit}
	}
	rows, err := s.db.Query(r.Context(), query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id, conversationID, messageID, aiRunID, usageType, creditSource, modelName, notes, decision, questionSummary, contactName, contactPhone, organizationName string
		var inputTokens, outputTokens, embeddingTokens, totalTokens, creditsUsed, latencyMS int
		var costIDR, creditUnitIDR, usdToIDRRate float64
		var createdAt time.Time
		if err := rows.Scan(&id, &conversationID, &messageID, &aiRunID, &usageType, &creditSource, &modelName, &inputTokens, &outputTokens, &embeddingTokens, &totalTokens, &costIDR, &creditsUsed, &creditUnitIDR, &usdToIDRRate, &notes, &decision, &questionSummary, &latencyMS, &contactName, &contactPhone, &organizationName, &createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		status := "success"
		if decision == "escalate" {
			status = "escalated"
		}
		if role == "owner" {
			conversationID = ""
			messageID = ""
			contactName = ""
			contactPhone = ""
			questionSummary = ""
		}
		items = append(items, map[string]any{
			"id":                    id,
			"conversationId":        conversationID,
			"messageId":             messageID,
			"aiRunId":               aiRunID,
			"usageType":             usageType,
			"creditSource":          creditSource,
			"modelName":             modelName,
			"modelUsed":             modelName,
			"inputTokens":           inputTokens,
			"outputTokens":          outputTokens,
			"embeddingTokens":       embeddingTokens,
			"totalTokens":           totalTokens,
			"costIdr":               costIDR,
			"creditsUsed":           creditsUsed,
			"creditUnitIdrSnapshot": creditUnitIDR,
			"usdToIdrRateSnapshot":  usdToIDRRate,
			"notes":                 notes,
			"decision":              decision,
			"status":                status,
			"questionSummary":       questionSummary,
			"latencyMs":             latencyMS,
			"contactName":           contactName,
			"contactPhone":          contactPhone,
			"organizationName":      organizationName,
			"createdAt":             createdAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleBillingUsageLogRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/billing/usage-logs/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	var conversationID, messageID, aiRunID, usageType, creditSource, modelName, notes string
	var inputTokens, outputTokens, embeddingTokens, creditsUsed int
	var costIDR, creditUnitIDR, usdToIDRRate float64
	var createdAt time.Time
	err := s.db.QueryRow(r.Context(), `
		SELECT
		  COALESCE(conversation_id::text, ''),
		  COALESCE(message_id::text, ''),
		  COALESCE(ai_run_id::text, ''),
		  usage_type,
		  credit_source,
		  COALESCE(model_name, ''),
		  COALESCE(input_tokens, 0),
		  COALESCE(output_tokens, 0),
		  COALESCE(embedding_tokens, 0),
		  COALESCE(cost_idr, 0),
		  credits_used,
		  credit_unit_idr_snapshot,
		  usd_to_idr_rate_snapshot,
		  COALESCE(notes, ''),
		  created_at
		FROM credit_usage_logs
		WHERE id = $1 AND organization_id = $2
	`, id, s.organizationID(r.Context())).Scan(&conversationID, &messageID, &aiRunID, &usageType, &creditSource, &modelName, &inputTokens, &outputTokens, &embeddingTokens, &costIDR, &creditsUsed, &creditUnitIDR, &usdToIDRRate, &notes, &createdAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "usage log not found"})
		return
	}

	detail := map[string]any{
		"id":                    id,
		"conversationId":        conversationID,
		"messageId":             messageID,
		"aiRunId":               aiRunID,
		"usageType":             usageType,
		"creditSource":          creditSource,
		"modelName":             modelName,
		"inputTokens":           inputTokens,
		"outputTokens":          outputTokens,
		"embeddingTokens":       embeddingTokens,
		"costIdr":               costIDR,
		"creditsUsed":           creditsUsed,
		"creditUnitIdrSnapshot": creditUnitIDR,
		"usdToIdrRateSnapshot":  usdToIDRRate,
		"notes":                 notes,
		"createdAt":             createdAt,
	}

	if aiRunID != "" {
		var decision, questionSummary, answerText, escalationReason string
		var confidenceScore float64
		var latencyMS int
		var runCreatedAt time.Time
		err := s.db.QueryRow(r.Context(), `
			SELECT decision, COALESCE(question_summary, ''), COALESCE(answer_text, ''),
			       COALESCE(confidence_score, 0), COALESCE(escalation_reason, ''),
			       COALESCE(latency_ms, 0), created_at
			FROM ai_runs
			WHERE id = $1 AND organization_id = $2
		`, aiRunID, s.organizationID(r.Context())).Scan(&decision, &questionSummary, &answerText, &confidenceScore, &escalationReason, &latencyMS, &runCreatedAt)
		if err == nil {
			run := map[string]any{
				"id":               aiRunID,
				"decision":         decision,
				"confidenceScore":  confidenceScore,
				"escalationReason": escalationReason,
				"latencyMS":        latencyMS,
				"createdAt":        runCreatedAt,
			}
			if role != "owner" {
				run["questionSummary"] = questionSummary
				run["answerText"] = answerText
			}
			detail["aiRun"] = run
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"log": detail})
}

func (s *Server) handleBillingAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" && role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var analytics struct {
		TotalCreditsUsed int              `json:"totalCreditsUsed"`
		TotalCostIDR     float64          `json:"totalCostIdr"`
		ByUsageType      []map[string]any `json:"byUsageType"`
		ByCreditSource   []map[string]any `json:"byCreditSource"`
		ByModel          []map[string]any `json:"byModel"`
		ByOrganization   []map[string]any `json:"byOrganization"`
		Daily            []map[string]any `json:"daily"`
	}
	scopeWhere := "WHERE organization_id = $1"
	scopeArgs := []any{s.organizationID(r.Context())}
	if role == "owner" {
		scopeWhere = ""
		scopeArgs = nil
	}
	if err := s.db.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(credits_used), 0), COALESCE(SUM(cost_idr), 0)
		FROM credit_usage_logs
		`+scopeWhere, scopeArgs...).Scan(&analytics.TotalCreditsUsed, &analytics.TotalCostIDR); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	appendBreakdown := func(query string) ([]map[string]any, error) {
		rows, err := s.db.Query(r.Context(), query, scopeArgs...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var label string
			var credits int
			var cost float64
			var count int
			if err := rows.Scan(&label, &credits, &cost, &count); err != nil {
				return nil, err
			}
			items = append(items, map[string]any{
				"label":       label,
				"creditsUsed": credits,
				"costIdr":     cost,
				"count":       count,
			})
		}
		return items, nil
	}

	var err error
	analytics.ByUsageType, err = appendBreakdown(`
		SELECT usage_type, COALESCE(SUM(credits_used), 0), COALESCE(SUM(cost_idr), 0), COUNT(*)
		FROM credit_usage_logs
		` + scopeWhere + `
		GROUP BY usage_type
		ORDER BY SUM(credits_used) DESC
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	analytics.ByCreditSource, err = appendBreakdown(`
		SELECT credit_source, COALESCE(SUM(credits_used), 0), COALESCE(SUM(cost_idr), 0), COUNT(*)
		FROM credit_usage_logs
		` + scopeWhere + `
		GROUP BY credit_source
		ORDER BY SUM(credits_used) DESC
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	analytics.ByModel, err = appendBreakdown(`
		SELECT COALESCE(NULLIF(model_name, ''), 'unknown'), COALESCE(SUM(credits_used), 0), COALESCE(SUM(cost_idr), 0), COUNT(*)
		FROM credit_usage_logs
		` + scopeWhere + `
		GROUP BY COALESCE(NULLIF(model_name, ''), 'unknown')
		ORDER BY SUM(credits_used) DESC
		LIMIT 10
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if role == "owner" {
		rows, err := s.db.Query(r.Context(), `
			SELECT COALESCE(o.name, 'Unknown'), COALESCE(SUM(l.credits_used), 0), COALESCE(SUM(l.cost_idr), 0), COUNT(*)
			FROM credit_usage_logs l
			LEFT JOIN organizations o ON o.id = l.organization_id
			GROUP BY COALESCE(o.name, 'Unknown')
			ORDER BY SUM(l.credits_used) DESC
			LIMIT 20
		`)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()
		for rows.Next() {
			var label string
			var credits int
			var cost float64
			var count int
			if err := rows.Scan(&label, &credits, &cost, &count); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			analytics.ByOrganization = append(analytics.ByOrganization, map[string]any{
				"label":       label,
				"creditsUsed": credits,
				"costIdr":     cost,
				"count":       count,
			})
		}
	}

	dailyQuery := `
		SELECT date_trunc('day', created_at)::date::text, COALESCE(SUM(credits_used), 0), COALESCE(SUM(cost_idr), 0), COUNT(*)
		FROM credit_usage_logs
		WHERE organization_id = $1 AND created_at >= NOW() - INTERVAL '14 days'
		GROUP BY 1
		ORDER BY 1 ASC
	`
	dailyArgs := scopeArgs
	if role == "owner" {
		dailyQuery = strings.Replace(dailyQuery, "WHERE organization_id = $1 AND", "WHERE", 1)
		dailyArgs = nil
	}
	rows, err := s.db.Query(r.Context(), dailyQuery, dailyArgs...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	analytics.Daily = []map[string]any{}
	for rows.Next() {
		var label string
		var credits int
		var cost float64
		var count int
		if err := rows.Scan(&label, &credits, &cost, &count); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		analytics.Daily = append(analytics.Daily, map[string]any{
			"label":       label,
			"creditsUsed": credits,
			"costIdr":     cost,
			"count":       count,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"analytics": analytics})
}

func (s *Server) handleBillingPricing(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "owner" && role != "super_admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only owner or super admin can access pricing"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		var payload struct {
			ID                         string            `json:"id"`
			CreditUnitIDR              float64           `json:"creditUnitIdr"`
			USDToIDRRate               float64           `json:"usdToIdrRate"`
			ChatModelName              string            `json:"chatModelName"`
			ChatInputPricePer1M        float64           `json:"chatInputPricePer1m"`
			ChatOutputPricePer1M       float64           `json:"chatOutputPricePer1m"`
			EmbeddingModelName         string            `json:"embeddingModelName"`
			EmbeddingPricePer1M        float64           `json:"embeddingPricePer1m"`
			AnnualDiscountPercent      float64           `json:"annualDiscountPercent"`
			MonthlyCreditLimit         int               `json:"monthlyCreditLimit"`
			MonthlyCreditsRemaining    int               `json:"monthlyCreditsRemaining"`
			AdditionalCreditsRemaining int               `json:"additionalCreditsRemaining"`
			ModelAliases               map[string]string `json:"modelAliases"`
			AvailableModelIDs          []string          `json:"availableModelIds"`
			UpdatedAt                  time.Time         `json:"updatedAt"`
		}
		err := s.db.QueryRow(r.Context(), `
			SELECT
			  cps.id,
			  cps.credit_unit_idr,
			  cps.usd_to_idr_rate,
			  cps.chat_model_name,
			  cps.chat_input_price_per_1m,
			  cps.chat_output_price_per_1m,
			  cps.embedding_model_name,
			  cps.embedding_price_per_1m,
			  COALESCE(cps.annual_discount_percent, 10),
			  cw.monthly_credit_limit,
			  cw.monthly_credits_remaining,
			  cw.additional_credits_remaining,
			  cps.updated_at
			FROM credit_pricing_settings cps
			CROSS JOIN LATERAL (
			  SELECT monthly_credit_limit, monthly_credits_remaining, additional_credits_remaining
			  FROM credit_wallet
			  WHERE is_active = TRUE AND organization_id = $1
			  ORDER BY created_at ASC
			  LIMIT 1
			) cw
			WHERE cps.is_active = TRUE AND cps.organization_id = $1
			ORDER BY cps.updated_at DESC
			LIMIT 1
		`, s.organizationID(r.Context())).Scan(
			&payload.ID,
			&payload.CreditUnitIDR,
			&payload.USDToIDRRate,
			&payload.ChatModelName,
			&payload.ChatInputPricePer1M,
			&payload.ChatOutputPricePer1M,
			&payload.EmbeddingModelName,
			&payload.EmbeddingPricePer1M,
			&payload.AnnualDiscountPercent,
			&payload.MonthlyCreditLimit,
			&payload.MonthlyCreditsRemaining,
			&payload.AdditionalCreditsRemaining,
			&payload.UpdatedAt,
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		payload.ModelAliases = s.openRouterChatModelAliases(r.Context())
		payload.AvailableModelIDs = openRouterAvailableChatModelIDList(s.openRouterAvailableChatModelIDs(r.Context()))
		writeJSON(w, http.StatusOK, map[string]any{"pricing": payload})
	case http.MethodPost:
		var req struct {
			CreditUnitIDR         float64           `json:"creditUnitIdr"`
			USDToIDRRate          float64           `json:"usdToIdrRate"`
			ChatModelName         string            `json:"chatModelName"`
			ChatInputPricePer1M   float64           `json:"chatInputPricePer1m"`
			ChatOutputPricePer1M  float64           `json:"chatOutputPricePer1m"`
			EmbeddingModelName    string            `json:"embeddingModelName"`
			EmbeddingPricePer1M   float64           `json:"embeddingPricePer1m"`
			AnnualDiscountPercent float64           `json:"annualDiscountPercent"`
			MonthlyCreditLimit    int               `json:"monthlyCreditLimit"`
			ModelAliases          map[string]string `json:"modelAliases"`
			AvailableModelIDs     []string          `json:"availableModelIds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.ChatModelName = strings.TrimSpace(req.ChatModelName)
		if req.ChatModelName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "chatModelName is required"})
			return
		}
		if !isAllowedOpenRouterChatModel(req.ChatModelName, "") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "chatModelName must be an allowed OpenRouter GPT, Claude, or DeepSeek model"})
			return
		}
		if role != "owner" && !s.openRouterAvailableChatModelIDs(r.Context())[strings.ToLower(req.ChatModelName)] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "chatModelName is not available for this workspace"})
			return
		}

		tx, err := s.db.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer tx.Rollback(r.Context())

		if role == "super_admin" {
			var current struct {
				CreditUnitIDR         float64
				USDToIDRRate          float64
				ChatInputPricePer1M   float64
				ChatOutputPricePer1M  float64
				EmbeddingModelName    string
				EmbeddingPricePer1M   float64
				AnnualDiscountPercent float64
			}
			if err := tx.QueryRow(r.Context(), `
				SELECT
				  credit_unit_idr,
				  usd_to_idr_rate,
				  chat_input_price_per_1m,
				  chat_output_price_per_1m,
				  embedding_model_name,
				  embedding_price_per_1m,
				  COALESCE(annual_discount_percent, 10)
				FROM credit_pricing_settings
				WHERE is_active = TRUE AND organization_id = $1
				ORDER BY updated_at DESC
				LIMIT 1
			`, s.organizationID(r.Context())).Scan(
				&current.CreditUnitIDR,
				&current.USDToIDRRate,
				&current.ChatInputPricePer1M,
				&current.ChatOutputPricePer1M,
				&current.EmbeddingModelName,
				&current.EmbeddingPricePer1M,
				&current.AnnualDiscountPercent,
			); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if inputPrice, outputPrice, ok := openRouterChatModelDefaultPrices(req.ChatModelName); ok {
				current.ChatInputPricePer1M = inputPrice
				current.ChatOutputPricePer1M = outputPrice
			}
			if _, err := tx.Exec(r.Context(), `UPDATE credit_pricing_settings SET is_active = FALSE, updated_at = NOW() WHERE is_active = TRUE AND organization_id = $1`, s.organizationID(r.Context())); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if _, err := tx.Exec(r.Context(), `
				INSERT INTO credit_pricing_settings (
				  credit_unit_idr,
				  usd_to_idr_rate,
				  chat_model_name,
				  chat_input_price_per_1m,
				  chat_output_price_per_1m,
				  embedding_model_name,
				  embedding_price_per_1m,
				  annual_discount_percent,
				  is_active,
				  updated_by,
				  organization_id
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE, $9, $10)
			`, current.CreditUnitIDR, current.USDToIDRRate, req.ChatModelName, current.ChatInputPricePer1M, current.ChatOutputPricePer1M, current.EmbeddingModelName, current.EmbeddingPricePer1M, current.AnnualDiscountPercent, s.agentID(r.Context()), s.organizationID(r.Context())); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := s.insertAuditLogTx(r.Context(), tx, "billing.model_update", "billing", "", map[string]any{
				"chat_model_name": req.ChatModelName,
			}); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := tx.Commit(r.Context()); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}

		req.EmbeddingModelName = strings.TrimSpace(req.EmbeddingModelName)
		if req.CreditUnitIDR <= 0 || req.USDToIDRRate <= 0 || req.ChatInputPricePer1M < 0 || req.ChatOutputPricePer1M < 0 || req.EmbeddingModelName == "" || req.EmbeddingPricePer1M < 0 || req.AnnualDiscountPercent < 0 || req.AnnualDiscountPercent >= 100 || req.MonthlyCreditLimit <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid pricing fields and monthlyCreditLimit are required"})
			return
		}

		if _, err := tx.Exec(r.Context(), `UPDATE credit_pricing_settings SET is_active = FALSE, updated_at = NOW() WHERE is_active = TRUE AND organization_id = $1`, s.organizationID(r.Context())); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO credit_pricing_settings (
			  credit_unit_idr,
			  usd_to_idr_rate,
			  chat_model_name,
			  chat_input_price_per_1m,
			  chat_output_price_per_1m,
			  embedding_model_name,
			  embedding_price_per_1m,
			  annual_discount_percent,
			  is_active,
			  updated_by,
			  organization_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE, $9, $10)
		`, req.CreditUnitIDR, req.USDToIDRRate, req.ChatModelName, req.ChatInputPricePer1M, req.ChatOutputPricePer1M, req.EmbeddingModelName, req.EmbeddingPricePer1M, req.AnnualDiscountPercent, s.agentID(r.Context()), s.organizationID(r.Context())); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if _, err := tx.Exec(r.Context(), `
			UPDATE credit_wallet
			SET monthly_credit_limit = $1,
			    monthly_credits_remaining = GREATEST(0, $1 - monthly_credits_used),
			    updated_at = NOW()
			WHERE is_active = TRUE AND organization_id = $2
		`, req.MonthlyCreditLimit, s.organizationID(r.Context())); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.upsertOpenRouterChatModelAliasesTx(r.Context(), tx, req.ModelAliases); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		availableModelIDs, err := s.replaceOpenRouterAvailableChatModelsTx(r.Context(), tx, req.AvailableModelIDs, req.ChatModelName)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLogTx(r.Context(), tx, "billing.pricing_update", "billing", "", map[string]any{
			"chat_model_name":         req.ChatModelName,
			"credit_unit_idr":         req.CreditUnitIDR,
			"usd_to_idr_rate":         req.USDToIDRRate,
			"monthly_credit_limit":    req.MonthlyCreditLimit,
			"annual_discount_percent": req.AnnualDiscountPercent,
			"model_aliases":           req.ModelAliases,
			"available_model_ids":     availableModelIDs,
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

func (s *Server) handleBillingAdjustments(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only owner can manage credit adjustments"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := s.db.Query(r.Context(), `
			SELECT ca.id, ca.adjustment_type, ca.amount, COALESCE(ca.notes, ''), COALESCE(a.username, ''), ca.created_at
			FROM credit_adjustments ca
			LEFT JOIN agents a ON a.id = ca.created_by
			WHERE ca.organization_id = $1
			ORDER BY ca.created_at DESC
			LIMIT 100
		`, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		items := []map[string]any{}
		for rows.Next() {
			var id, adjustmentType, notes, createdBy string
			var amount int
			var createdAt time.Time
			if err := rows.Scan(&id, &adjustmentType, &amount, &notes, &createdBy, &createdAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			items = append(items, map[string]any{
				"id":             id,
				"adjustmentType": adjustmentType,
				"amount":         amount,
				"notes":          notes,
				"createdBy":      createdBy,
				"createdAt":      createdAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var req struct {
			AdjustmentType string `json:"adjustmentType"`
			Amount         int    `json:"amount"`
			Notes          string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.AdjustmentType = strings.TrimSpace(req.AdjustmentType)
		req.Notes = strings.TrimSpace(req.Notes)
		if req.Amount <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "amount must be positive"})
			return
		}

		tx, err := s.db.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer tx.Rollback(r.Context())

		switch req.AdjustmentType {
		case "add_monthly":
			_, err = tx.Exec(r.Context(), `UPDATE credit_wallet SET monthly_credits_remaining = monthly_credits_remaining + $1, updated_at = NOW() WHERE is_active = TRUE AND organization_id = $2`, req.Amount, s.organizationID(r.Context()))
		case "subtract_monthly":
			_, err = tx.Exec(r.Context(), `UPDATE credit_wallet SET monthly_credits_remaining = GREATEST(0, monthly_credits_remaining - $1), updated_at = NOW() WHERE is_active = TRUE AND organization_id = $2`, req.Amount, s.organizationID(r.Context()))
		case "add_additional":
			_, err = tx.Exec(r.Context(), `UPDATE credit_wallet SET additional_credits_remaining = additional_credits_remaining + $1, updated_at = NOW() WHERE is_active = TRUE AND organization_id = $2`, req.Amount, s.organizationID(r.Context()))
		case "subtract_additional":
			_, err = tx.Exec(r.Context(), `UPDATE credit_wallet SET additional_credits_remaining = GREATEST(0, additional_credits_remaining - $1), updated_at = NOW() WHERE is_active = TRUE AND organization_id = $2`, req.Amount, s.organizationID(r.Context()))
		case "set_monthly_limit":
			_, err = tx.Exec(r.Context(), `
				UPDATE credit_wallet
				SET monthly_credit_limit = $1,
				    monthly_credits_remaining = GREATEST(0, $1 - monthly_credits_used),
				    updated_at = NOW()
				WHERE is_active = TRUE AND organization_id = $2
			`, req.Amount, s.organizationID(r.Context()))
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported adjustmentType"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		var id string
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO credit_adjustments (adjustment_type, amount, notes, created_by, organization_id)
			VALUES ($1, $2, NULLIF($3, ''), $4, $5)
			RETURNING id
		`, req.AdjustmentType, req.Amount, req.Notes, s.agentID(r.Context()), s.organizationID(r.Context())).Scan(&id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLogTx(r.Context(), tx, "billing.credit_adjustment", "credit_adjustment", id, map[string]any{
			"adjustment_type": req.AdjustmentType,
			"amount":          req.Amount,
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "id": id})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleBillingPackages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		annualDiscountPercent, err := s.activeAnnualDiscountPercent(r.Context(), s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		rows, err := s.db.Query(r.Context(), `
			SELECT id, name, COALESCE(plan_key, ''), COALESCE(description, ''), credit_amount, price,
			       COALESCE(max_whatsapp_sessions, 1), COALESCE(max_ai_agents, 1), COALESCE(max_human_users, 1),
			       COALESCE(billing_period, 'monthly'), COALESCE(is_popular, FALSE), is_active, updated_at
			FROM credit_packages
			WHERE organization_id = $1
			ORDER BY price ASC, updated_at DESC
		`, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		items := []map[string]any{}
		for rows.Next() {
			var id, name, planKey, description, billingPeriod string
			var creditAmount, maxWhatsAppSessions, maxAIAgents, maxHumanUsers int
			var price float64
			var isPopular, isActive bool
			var updatedAt time.Time
			if err := rows.Scan(&id, &name, &planKey, &description, &creditAmount, &price, &maxWhatsAppSessions, &maxAIAgents, &maxHumanUsers, &billingPeriod, &isPopular, &isActive, &updatedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			items = append(items, map[string]any{
				"id":                    id,
				"name":                  name,
				"planKey":               planKey,
				"description":           description,
				"creditAmount":          creditAmount,
				"price":                 price,
				"maxWhatsAppSessions":   maxWhatsAppSessions,
				"maxAiAgents":           maxAIAgents,
				"maxHumanUsers":         maxHumanUsers,
				"billingPeriod":         billingPeriod,
				"annualDiscountPercent": annualDiscountPercent,
				"isPopular":             isPopular,
				"isActive":              isActive,
				"updatedAt":             updatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"annualDiscountPercent": annualDiscountPercent, "items": items})
	case http.MethodPost:
		role := s.role(r.Context())
		if role != "owner" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only owner can manage packages"})
			return
		}

		var req struct {
			Name          string  `json:"name"`
			Description   string  `json:"description"`
			CreditAmount  int     `json:"creditAmount"`
			Price         float64 `json:"price"`
			MaxWhatsApp   int     `json:"maxWhatsAppSessions"`
			MaxAIAgents   int     `json:"maxAiAgents"`
			MaxUsers      int     `json:"maxHumanUsers"`
			BillingPeriod string  `json:"billingPeriod"`
			IsPopular     bool    `json:"isPopular"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Description = strings.TrimSpace(req.Description)
		req.BillingPeriod = normalizePackageBillingPeriod(req.BillingPeriod)
		if strings.TrimSpace(req.Name) == "" || req.CreditAmount <= 0 || req.Price <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, creditAmount, and price are required"})
			return
		}

		var id string
		if req.MaxWhatsApp <= 0 {
			req.MaxWhatsApp = 1
		}
		if req.MaxAIAgents <= 0 {
			req.MaxAIAgents = 1
		}
		if req.MaxUsers <= 0 {
			req.MaxUsers = 1
		}
		organizationID := s.organizationID(r.Context())
		popular := req.IsPopular && req.BillingPeriod == "monthly"
		tx, err := s.db.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer tx.Rollback(r.Context())
		if popular {
			if _, err := tx.Exec(r.Context(), `
				UPDATE credit_packages
				SET is_popular = FALSE,
				    updated_at = NOW()
				WHERE organization_id = $1
				  AND COALESCE(billing_period, 'monthly') = 'monthly'
				  AND is_popular = TRUE
			`, organizationID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
		err = tx.QueryRow(r.Context(), `
			INSERT INTO credit_packages (name, description, credit_amount, price, max_whatsapp_sessions, max_ai_agents, max_human_users, billing_period, is_active, is_popular, organization_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE, $9, $10)
			RETURNING id
		`, req.Name, req.Description, req.CreditAmount, req.Price, req.MaxWhatsApp, req.MaxAIAgents, req.MaxUsers, req.BillingPeriod, popular, organizationID).Scan(&id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLogTx(r.Context(), tx, "billing.package_create", "credit_package", id, map[string]any{
			"name":           req.Name,
			"credit_amount":  req.CreditAmount,
			"price":          req.Price,
			"billing_period": req.BillingPeriod,
			"is_popular":     popular,
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "id": id})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handlePublicBillingPackages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	var annualDiscountPercent float64
	if err := s.db.QueryRow(r.Context(), `
		SELECT COALESCE((
			SELECT cps.annual_discount_percent
			FROM credit_pricing_settings cps
			JOIN organizations o ON o.id = cps.organization_id
			WHERE o.slug = 'default-company'
			  AND o.status = 'active'
			  AND cps.is_active = TRUE
			ORDER BY cps.updated_at DESC
			LIMIT 1
		), 10)
	`).Scan(&annualDiscountPercent); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT cp.name, COALESCE(cp.description, ''), cp.credit_amount, cp.price,
		       COALESCE(cp.max_whatsapp_sessions, 1), COALESCE(cp.max_ai_agents, 1), COALESCE(cp.max_human_users, 1),
		       COALESCE(cp.is_popular, FALSE)
		FROM credit_packages cp
		JOIN organizations o ON o.id = cp.organization_id
		WHERE o.slug = 'default-company'
		  AND o.status = 'active'
		  AND cp.is_active = TRUE
		  AND COALESCE(cp.billing_period, 'monthly') = 'monthly'
		  AND cp.plan_key IN ('starter', 'growth', 'business')
		ORDER BY cp.price ASC, cp.updated_at DESC
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var name, description string
		var creditAmount, maxWhatsAppSessions, maxAIAgents, maxHumanUsers int
		var price float64
		var isPopular bool
		if err := rows.Scan(&name, &description, &creditAmount, &price, &maxWhatsAppSessions, &maxAIAgents, &maxHumanUsers, &isPopular); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		items = append(items, map[string]any{
			"name":                name,
			"description":         description,
			"creditAmount":        creditAmount,
			"price":               price,
			"maxWhatsAppSessions": maxWhatsAppSessions,
			"maxAiAgents":         maxAIAgents,
			"maxHumanUsers":       maxHumanUsers,
			"isPopular":           isPopular,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"annualDiscountPercent": annualDiscountPercent,
		"items":                 items,
	})
}

func normalizePackageBillingPeriod(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "one_time" || normalized == "topup" || normalized == "top_up" {
		return "one_time"
	}
	return "monthly"
}

func (s *Server) handleBillingPackageRoutes(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role != "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only owner can manage packages"})
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/billing/packages/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Name          string  `json:"name"`
			Description   string  `json:"description"`
			CreditAmount  int     `json:"creditAmount"`
			Price         float64 `json:"price"`
			MaxWhatsApp   int     `json:"maxWhatsAppSessions"`
			MaxAIAgents   int     `json:"maxAiAgents"`
			MaxUsers      int     `json:"maxHumanUsers"`
			BillingPeriod string  `json:"billingPeriod"`
			IsActive      *bool   `json:"isActive"`
			IsPopular     bool    `json:"isPopular"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Description = strings.TrimSpace(req.Description)
		req.BillingPeriod = normalizePackageBillingPeriod(req.BillingPeriod)
		if req.Name == "" || req.CreditAmount <= 0 || req.Price <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, creditAmount, and price are required"})
			return
		}
		if req.MaxWhatsApp <= 0 {
			req.MaxWhatsApp = 1
		}
		if req.MaxAIAgents <= 0 {
			req.MaxAIAgents = 1
		}
		if req.MaxUsers <= 0 {
			req.MaxUsers = 1
		}
		active := true
		if req.IsActive != nil {
			active = *req.IsActive
		}
		organizationID := s.organizationID(r.Context())
		popular := active && req.IsPopular && req.BillingPeriod == "monthly"
		tx, err := s.db.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer tx.Rollback(r.Context())
		if popular {
			if _, err := tx.Exec(r.Context(), `
				UPDATE credit_packages
				SET is_popular = FALSE,
				    updated_at = NOW()
				WHERE organization_id = $1
				  AND id <> NULLIF($2, '')::uuid
				  AND COALESCE(billing_period, 'monthly') = 'monthly'
				  AND is_popular = TRUE
			`, organizationID, id); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
		tag, err := tx.Exec(r.Context(), `
			UPDATE credit_packages
			SET name = $3,
			    description = $4,
			    credit_amount = $5,
			    price = $6,
			    max_whatsapp_sessions = $7,
			    max_ai_agents = $8,
			    max_human_users = $9,
			    billing_period = $10,
			    is_active = $11,
			    is_popular = $12,
			    updated_at = NOW()
			WHERE id = NULLIF($1, '')::uuid
			  AND organization_id = $2
		`, id, organizationID, req.Name, req.Description, req.CreditAmount, req.Price, req.MaxWhatsApp, req.MaxAIAgents, req.MaxUsers, req.BillingPeriod, active, popular)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "package not found"})
			return
		}
		if err := s.insertAuditLogTx(r.Context(), tx, "billing.package_update", "credit_package", id, map[string]any{
			"name":           req.Name,
			"credit_amount":  req.CreditAmount,
			"price":          req.Price,
			"billing_period": req.BillingPeriod,
			"is_active":      active,
			"is_popular":     popular,
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case http.MethodDelete:
		tag, err := s.db.Exec(r.Context(), `
			UPDATE credit_packages
			SET is_active = FALSE,
			    is_popular = FALSE,
			    updated_at = NOW()
			WHERE id = NULLIF($1, '')::uuid
			  AND organization_id = $2
		`, id, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "package not found"})
			return
		}
		if err := s.insertAuditLog(r.Context(), "billing.package_delete", "credit_package", id, map[string]any{"soft_delete": true}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleBillingPurchases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		role := s.role(r.Context())
		if role == "operator" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		query := `
			SELECT
		  p.id,
		  p.credit_package_id,
		  COALESCE(cp.name, ''),
			  COALESCE(cp.plan_key, ''),
			  COALESCE(cp.max_whatsapp_sessions, 1),
			  COALESCE(cp.max_ai_agents, 1),
			  COALESCE(cp.max_human_users, 1),
			  COALESCE(p.billing_period, cp.billing_period, 'monthly'),
			  p.credit_amount,
			  p.price,
			  p.payment_method,
			  p.payment_status,
			  COALESCE(req.username, ''),
			  COALESCE(conf.username, ''),
			  COALESCE(p.notes, ''),
			  COALESCE(o.name, ''),
			  p.created_at,
			  p.confirmed_at,
			  COALESCE(p.midtrans_order_id, ''),
			  COALESCE(p.snap_token, ''),
			  COALESCE(p.snap_redirect_url, ''),
			  COALESCE(p.gross_amount, 0),
			  COALESCE(p.transaction_status, ''),
		  COALESCE(p.fraud_status, ''),
		  COALESCE(p.payment_type, ''),
		  p.paid_at,
		  p.expired_at,
		  p.active_from,
		  p.active_until
		  FROM credit_purchases p
			LEFT JOIN credit_packages cp ON cp.id = p.credit_package_id AND cp.organization_id = p.organization_id
			LEFT JOIN agents req ON req.id = p.requested_by
			LEFT JOIN agents conf ON conf.id = p.confirmed_by
			LEFT JOIN organizations o ON o.id = p.organization_id
			WHERE p.organization_id = $1
			ORDER BY p.created_at DESC
		`
		args := []any{s.organizationID(r.Context())}
		if role == "owner" {
			query = strings.Replace(query, "WHERE p.organization_id = $1", "", 1)
			args = nil
		}
		rows, err := s.db.Query(r.Context(), query, args...)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		items := []map[string]any{}
		for rows.Next() {
			var id, packageID, packageName, planKey, billingPeriod, paymentMethod, paymentStatus, requestedBy, confirmedBy, notes, organizationName string
			var midtransOrderID, snapToken, snapRedirectURL, transactionStatus, fraudStatus, paymentType string
			var creditAmount, maxWhatsAppSessions, maxAIAgents, maxHumanUsers int
			var price, grossAmount float64
			var createdAt time.Time
			var confirmedAt, paidAt, expiredAt, activeFrom, activeUntil *time.Time
			if err := rows.Scan(&id, &packageID, &packageName, &planKey, &maxWhatsAppSessions, &maxAIAgents, &maxHumanUsers, &billingPeriod, &creditAmount, &price, &paymentMethod, &paymentStatus, &requestedBy, &confirmedBy, &notes, &organizationName, &createdAt, &confirmedAt, &midtransOrderID, &snapToken, &snapRedirectURL, &grossAmount, &transactionStatus, &fraudStatus, &paymentType, &paidAt, &expiredAt, &activeFrom, &activeUntil); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			paymentFee := grossAmount - price
			if paymentFee < 0 {
				paymentFee = 0
			}
			items = append(items, map[string]any{
				"id":                id,
				"packageId":         packageID,
				"packageName":       packageName,
				"planKey":           planKey,
				"billingPeriod":     billingPeriod,
				"limits":            map[string]int{"whatsapp": maxWhatsAppSessions, "aiAgents": maxAIAgents, "humanUsers": maxHumanUsers},
				"creditAmount":      creditAmount,
				"price":             price,
				"paymentMethod":     paymentMethod,
				"paymentStatus":     paymentStatus,
				"requestedBy":       requestedBy,
				"confirmedBy":       confirmedBy,
				"notes":             notes,
				"organization":      organizationName,
				"createdAt":         createdAt,
				"confirmedAt":       confirmedAt,
				"midtransOrderId":   midtransOrderID,
				"snapToken":         snapToken,
				"snapRedirectUrl":   snapRedirectURL,
				"grossAmount":       grossAmount,
				"paymentFee":        paymentFee,
				"transactionStatus": transactionStatus,
				"fraudStatus":       fraudStatus,
				"paymentType":       paymentType,
				"paidAt":            paidAt,
				"expiredAt":         expiredAt,
				"activeFrom":        activeFrom,
				"activeUntil":       activeUntil,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		role := s.role(r.Context())
		if role != "admin" && role != "super_admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin or super_admin can request purchases"})
			return
		}

		var req struct {
			PackageID     string `json:"packageId"`
			BillingPeriod string `json:"billingPeriod"`
			PaymentMethod string `json:"paymentMethod"`
			Notes         string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		req.PaymentMethod = normalizePurchasePaymentMethod(req.PaymentMethod)
		if req.PaymentMethod == "card" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Pembayaran kartu belum tersedia. Gunakan Virtual Account atau QRIS."})
			return
		}
		req.Notes = strings.TrimSpace(req.Notes)

		var packageName string
		var packagePlanKey string
		var creditAmount int
		var price float64
		var catalogBillingPeriod string
		err := s.db.QueryRow(r.Context(), `
			SELECT name, COALESCE(plan_key, ''), credit_amount, price, COALESCE(billing_period, 'monthly')
			FROM credit_packages
			WHERE id = $1 AND is_active = TRUE AND organization_id = $2
		`, req.PackageID, s.organizationID(r.Context())).Scan(&packageName, &packagePlanKey, &creditAmount, &price, &catalogBillingPeriod)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "package not found"})
			return
		}
		billingPeriod, err := resolveCheckoutBillingPeriod(req.BillingPeriod, catalogBillingPeriod)
		if err != nil {
			switch {
			case errors.Is(err, errCheckoutBillingPeriodMismatch):
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Periode pembayaran tidak sesuai dengan paket yang dipilih."})
			default:
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Periode paket belum didukung untuk checkout."})
			}
			return
		}
		organizationID := s.organizationID(r.Context())
		annualDiscountPercent := float64(0)
		if billingPeriod == "annual" {
			annualDiscountPercent, err = s.activeAnnualDiscountPercent(r.Context(), organizationID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			price = annualCheckoutAmount(price, annualDiscountPercent)
		}
		if billingPeriod == "one_time" {
			var hasActivePlan bool
			err = s.db.QueryRow(r.Context(), `
				SELECT EXISTS (
				  SELECT 1
				  FROM credit_purchases p
				  JOIN credit_packages cp ON cp.id = p.credit_package_id AND cp.organization_id = p.organization_id
				  WHERE p.organization_id = $1
				    AND p.payment_status = 'confirmed'
				    AND COALESCE(cp.billing_period, 'monthly') = 'monthly'
				    AND (p.active_until IS NULL OR p.active_until > NOW())
				)
			`, s.organizationID(r.Context())).Scan(&hasActivePlan)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if !hasActivePlan {
				writeJSON(w, http.StatusPaymentRequired, map[string]string{"error": "top-up add-on requires an active monthly package"})
				return
			}
		} else {
			activePlan, err := s.activeOrganizationPlanLimits(r.Context())
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			activeRank := planPackageRank(activePlan.PlanKey)
			selectedRank := planPackageRank(packagePlanKey)
			if activeRank > 0 && selectedRank > 0 && selectedRank < activeRank {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "Paket aktif saat ini sudah lebih tinggi. Pilih paket yang setara atau upgrade ke paket yang lebih tinggi."})
				return
			}
		}

		quote := calculatePurchasePaymentQuote(price, req.PaymentMethod)
		agentID := s.agentID(r.Context())
		var id string
		var existingPaymentMethod, existingPaymentStatus, existingSnapToken, existingSnapRedirectURL, existingMidtransOrderID string
		var existingGrossAmount float64
		reusedPurchase := false
		err = s.db.QueryRow(r.Context(), `
			SELECT id, payment_method, payment_status, COALESCE(snap_token, ''), COALESCE(snap_redirect_url, ''), COALESCE(midtrans_order_id, ''), COALESCE(gross_amount, 0)
			FROM credit_purchases
			WHERE organization_id = $1
			  AND credit_package_id = $2
			  AND billing_period = $3
			  AND payment_status IN ('requested', 'pending')
			ORDER BY created_at DESC
			LIMIT 1
		`, organizationID, req.PackageID, billingPeriod).Scan(&id, &existingPaymentMethod, &existingPaymentStatus, &existingSnapToken, &existingSnapRedirectURL, &existingMidtransOrderID, &existingGrossAmount)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err == nil {
			reusedPurchase = true
			if s.midtransConfigured() && existingMidtransOrderID != "" {
				if paymentStatus, syncErr := s.syncMidtransPurchaseStatus(r.Context(), id, organizationID); syncErr == nil {
					existingPaymentStatus = paymentStatus
					if paymentStatus != "requested" && paymentStatus != "pending" {
						reusedPurchase = false
						id = ""
					}
				}
			}
		}

		if reusedPurchase && existingSnapToken != "" && existingGrossAmount > 0 && int64(math.Round(existingGrossAmount)) != quote.GrossAmount {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Nominal pembayaran berubah. Batalkan pembayaran yang belum selesai, lalu buat checkout baru dengan harga terkini."})
			return
		}

		if reusedPurchase {
			if _, err := s.db.Exec(r.Context(), `
				UPDATE credit_purchases
				SET credit_amount = $2,
				    price = $3,
				    billing_period = $4,
				    payment_method_changed_at = CASE WHEN payment_method <> $5 THEN NOW() ELSE payment_method_changed_at END,
				    payment_method = $5,
				    notes = CASE WHEN NULLIF($6, '') IS NULL THEN notes ELSE $6 END
				WHERE id = $1 AND organization_id = $7
			`, id, creditAmount, price, billingPeriod, req.PaymentMethod, req.Notes, organizationID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		} else {
			err = s.db.QueryRow(r.Context(), `
				INSERT INTO credit_purchases (
				  credit_package_id, credit_amount, price, billing_period, payment_method, payment_status, requested_by, notes, organization_id
				)
				VALUES ($1, $2, $3, $4, $5, 'requested', $6, $7, $8)
				RETURNING id
			`, req.PackageID, creditAmount, price, billingPeriod, req.PaymentMethod, agentID, req.Notes, organizationID).Scan(&id)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := s.insertAuditLog(r.Context(), "billing.purchase_request", "credit_purchase", id, map[string]any{
				"package_id":              req.PackageID,
				"credit_amount":           creditAmount,
				"price":                   price,
				"billing_period":          billingPeriod,
				"annual_discount_percent": annualDiscountPercent,
				"payment_method":          req.PaymentMethod,
			}); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}

		response := map[string]any{
			"status":          "ok",
			"id":              id,
			"paymentStatus":   "requested",
			"reusedPurchase":  reusedPurchase,
			"billingPeriod":   billingPeriod,
			"packageAmount":   quote.PackageAmount,
			"paymentFee":      quote.PaymentFee,
			"paymentFeeLabel": quote.FeeLabel,
			"grossAmount":     quote.GrossAmount,
		}
		if reusedPurchase {
			response["paymentStatus"] = existingPaymentStatus
		}
		needsNewSnap := !reusedPurchase || existingPaymentMethod != req.PaymentMethod || existingSnapToken == "" || existingSnapRedirectURL == "" || int64(math.Round(existingGrossAmount)) != quote.GrossAmount
		if reusedPurchase && !needsNewSnap {
			existingPaymentFee := int64(existingGrossAmount) - quote.PackageAmount
			if existingPaymentFee < 0 {
				existingPaymentFee = 0
			}
			response["snapToken"] = existingSnapToken
			response["snapRedirectUrl"] = existingSnapRedirectURL
			response["midtransOrderId"] = existingMidtransOrderID
			response["midtransClientKey"] = s.cfg.MidtransClientKey
			response["midtransEnvironment"] = s.cfg.MidtransEnv
			response["paymentFee"] = existingPaymentFee
			response["grossAmount"] = int64(existingGrossAmount)
		}
		if s.midtransConfigured() && needsNewSnap {
			var customerName string
			_ = s.db.QueryRow(r.Context(), `
				SELECT COALESCE(NULLIF(name, ''), username, '')
				FROM agents
				WHERE id = $1
			`, agentID).Scan(&customerName)
			orderID := newMidtransOrderID(id)
			snapResult, err := s.createMidtransSnapTransaction(r.Context(), midtransSnapInput{
				PurchaseID: id,
				OrderID:    orderID,
				PackageID:  req.PackageID,
				PackageName: func() string {
					if billingPeriod == "annual" {
						return packageName + " Tahunan"
					}
					return packageName
				}(),
				CreditAmount:  creditAmount,
				Price:         float64(quote.GrossAmount),
				PaymentFee:    quote.PaymentFee,
				PaymentMethod: req.PaymentMethod,
				CustomerName:  customerName,
			})
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Midtrans belum bisa membuat transaksi sandbox. Coba lagi atau gunakan approval manual."})
				return
			}
			if _, err := s.db.Exec(r.Context(), `
				UPDATE credit_purchases
				SET payment_status = 'pending',
				    midtrans_order_id = $2,
				    snap_token = $3,
				    snap_redirect_url = $4,
				    gross_amount = $5,
				    transaction_status = 'pending'
				WHERE id = $1
			`, id, snapResult.OrderID, snapResult.Token, snapResult.RedirectURL, snapResult.GrossAmount); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := s.saveMidtransPaymentOrder(r.Context(), id, snapResult, req.PaymentMethod); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			response["paymentStatus"] = "pending"
			response["snapToken"] = snapResult.Token
			response["snapRedirectUrl"] = snapResult.RedirectURL
			response["midtransOrderId"] = snapResult.OrderID
			response["midtransClientKey"] = s.cfg.MidtransClientKey
			response["midtransEnvironment"] = s.cfg.MidtransEnv
			response["paymentFee"] = snapResult.GrossAmount - quote.PackageAmount
			response["grossAmount"] = snapResult.GrossAmount
		}
		writeJSON(w, http.StatusOK, response)
	default:
		http.NotFound(w, r)
	}
}

func invoicePurchaseNumber(id string, issuedAt time.Time) string {
	clean := strings.ToUpper(strings.ReplaceAll(id, "-", ""))
	if len(clean) > 8 {
		clean = clean[:8]
	}
	if clean == "" {
		clean = "00000000"
	}
	return fmt.Sprintf("INV-%s-%s", issuedAt.Format("20060102"), clean)
}

func (s *Server) handleBillingPurchaseInvoice(w http.ResponseWriter, r *http.Request, purchaseID string) {
	role := s.role(r.Context())
	if role == "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	query := `
		SELECT
		  p.id,
		  COALESCE(cp.name, ''),
		  COALESCE(cp.plan_key, ''),
		  COALESCE(p.billing_period, cp.billing_period, 'monthly'),
		  p.credit_amount,
		  p.price,
		  p.payment_method,
		  p.payment_status,
		  COALESCE(req.username, ''),
		  COALESCE(conf.username, ''),
		  COALESCE(o.name, ''),
		  COALESCE(o.slug, ''),
		  p.created_at,
		  p.confirmed_at,
		  COALESCE(p.midtrans_order_id, ''),
		  COALESCE(p.gross_amount, 0),
		  COALESCE(p.transaction_status, ''),
		  COALESCE(p.payment_type, ''),
		  p.paid_at,
		  p.active_from,
		  p.active_until
		FROM credit_purchases p
		LEFT JOIN credit_packages cp ON cp.id = p.credit_package_id AND cp.organization_id = p.organization_id
		LEFT JOIN agents req ON req.id = p.requested_by
		LEFT JOIN agents conf ON conf.id = p.confirmed_by
		LEFT JOIN organizations o ON o.id = p.organization_id
		WHERE p.id = $1 AND p.organization_id = $2
	`
	args := []any{purchaseID, s.organizationID(r.Context())}
	if role == "owner" {
		query = strings.Replace(query, " AND p.organization_id = $2", "", 1)
		args = []any{purchaseID}
	}

	var id, packageName, planKey, billingPeriod, paymentMethod, paymentStatus, requestedBy, confirmedBy, organizationName, organizationSlug string
	var midtransOrderID, transactionStatus, paymentType string
	var creditAmount int
	var price, grossAmount float64
	var createdAt time.Time
	var confirmedAt, paidAt, activeFrom, activeUntil *time.Time
	if err := s.db.QueryRow(r.Context(), query, args...).Scan(&id, &packageName, &planKey, &billingPeriod, &creditAmount, &price, &paymentMethod, &paymentStatus, &requestedBy, &confirmedBy, &organizationName, &organizationSlug, &createdAt, &confirmedAt, &midtransOrderID, &grossAmount, &transactionStatus, &paymentType, &paidAt, &activeFrom, &activeUntil); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "purchase not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if paymentStatus != "confirmed" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Invoice tersedia setelah pembayaran berhasil."})
		return
	}

	issuedAt := createdAt
	if paidAt != nil {
		issuedAt = *paidAt
	} else if confirmedAt != nil {
		issuedAt = *confirmedAt
	}
	if strings.TrimSpace(packageName) == "" {
		packageName = "Paket Oneflow.id"
	}
	total := grossAmount
	if total <= 0 {
		total = price
	}
	paymentFee := total - price
	if paymentFee < 0 {
		paymentFee = 0
	}
	lineItems := []map[string]any{
		{
			"description":  packageName,
			"quantity":     1,
			"creditAmount": creditAmount,
			"amount":       price,
		},
	}
	if paymentFee > 0 {
		lineItems = append(lineItems, map[string]any{
			"description": "Biaya pembayaran",
			"quantity":    1,
			"amount":      paymentFee,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"invoice": map[string]any{
			"id":         id,
			"number":     invoicePurchaseNumber(id, issuedAt),
			"issuedAt":   issuedAt,
			"status":     "paid",
			"currency":   "IDR",
			"subtotal":   price,
			"paymentFee": paymentFee,
			"total":      total,
			"seller": map[string]any{
				"name":        "Oneflow.id",
				"description": "AI chat automation platform",
			},
			"billTo": map[string]any{
				"name": organizationName,
				"slug": organizationSlug,
			},
			"purchase": map[string]any{
				"id":                id,
				"packageName":       packageName,
				"planKey":           planKey,
				"billingPeriod":     billingPeriod,
				"creditAmount":      creditAmount,
				"paymentMethod":     paymentMethod,
				"paymentStatus":     paymentStatus,
				"paymentType":       paymentType,
				"transactionStatus": transactionStatus,
				"midtransOrderId":   midtransOrderID,
				"requestedBy":       requestedBy,
				"confirmedBy":       confirmedBy,
				"createdAt":         createdAt,
				"confirmedAt":       confirmedAt,
				"paidAt":            paidAt,
				"activeFrom":        activeFrom,
				"activeUntil":       activeUntil,
			},
			"lineItems": lineItems,
		},
	})
}

func (s *Server) handleBillingPurchaseRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/billing/purchases/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if parts[1] == "invoice" {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		s.handleBillingPurchaseInvoice(w, r, parts[0])
		return
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	if parts[1] == "sync" {
		role := s.role(r.Context())
		if role == "operator" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		paymentStatus, err := s.syncMidtransPurchaseStatus(r.Context(), parts[0], s.organizationID(r.Context()))
		if err != nil {
			switch {
			case errors.Is(err, errMidtransNotConfigured):
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payment gateway is not configured"})
			case errors.Is(err, errCreditPurchaseNotFound):
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "purchase not found"})
			default:
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "paymentStatus": paymentStatus})
		return
	}

	if parts[1] == "cancel" {
		role := s.role(r.Context())
		if role == "operator" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		var paymentStatus, midtransOrderID string
		cancelLookupQuery := `
			SELECT payment_status, COALESCE(midtrans_order_id, '')
			FROM credit_purchases
			WHERE id = $1 AND organization_id = $2
		`
		cancelLookupArgs := []any{parts[0], s.organizationID(r.Context())}
		if role == "owner" {
			cancelLookupQuery = strings.Replace(cancelLookupQuery, " AND organization_id = $2", "", 1)
			cancelLookupArgs = []any{parts[0]}
		}
		if err := s.db.QueryRow(r.Context(), cancelLookupQuery, cancelLookupArgs...).Scan(&paymentStatus, &midtransOrderID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "purchase not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if paymentStatus != "requested" && paymentStatus != "pending" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Pembayaran ini tidak bisa dibatalkan."})
			return
		}

		if s.midtransConfigured() && strings.TrimSpace(midtransOrderID) != "" {
			notification, rawBody, err := s.cancelMidtransTransaction(r.Context(), midtransOrderID)
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Pembayaran belum bisa dibatalkan. Coba lagi beberapa saat lagi atau lanjutkan pembayaran yang tersedia."})
				return
			}
			tx, err := s.db.Begin(r.Context())
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			defer tx.Rollback(r.Context())
			if err := s.applyMidtransNotificationTx(r.Context(), tx, *notification, rawBody); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := s.insertAuditLogTx(r.Context(), tx, "billing.purchase_cancel", "credit_purchase", parts[0], map[string]any{"midtrans_order_id": midtransOrderID}); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if err := tx.Commit(r.Context()); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "paymentStatus": "cancelled"})
			return
		}

		cancelUpdateQuery := `
			UPDATE credit_purchases
			SET payment_status = 'cancelled',
			    transaction_status = 'cancel'
			WHERE id = $1 AND organization_id = $2 AND payment_status IN ('requested', 'pending')
		`
		cancelUpdateArgs := []any{parts[0], s.organizationID(r.Context())}
		if role == "owner" {
			cancelUpdateQuery = strings.Replace(cancelUpdateQuery, " AND organization_id = $2", "", 1)
			cancelUpdateArgs = []any{parts[0]}
		}
		if _, err := s.db.Exec(r.Context(), cancelUpdateQuery, cancelUpdateArgs...); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLog(r.Context(), "billing.purchase_cancel", "credit_purchase", parts[0], map[string]any{"source": "local"}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "paymentStatus": "cancelled"})
		return
	}

	if parts[1] != "confirm" {
		http.NotFound(w, r)
		return
	}

	role := s.role(r.Context())
	if role != "owner" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only owner can confirm purchases"})
		return
	}

	id := parts[0]
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	confirmedBy := s.agentID(r.Context())
	if err := s.confirmCreditPurchaseTx(r.Context(), tx, id, confirmedBy, nil, "billing.purchase_confirm", false); err != nil {
		if errors.Is(err, errCreditPurchaseNotFound) || errors.Is(err, errCreditPurchaseNotConfirmable) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "pending purchase not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
