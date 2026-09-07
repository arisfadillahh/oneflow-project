package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type ticketRequest struct {
	Title          *string        `json:"title"`
	Description    *string        `json:"description"`
	IssueType      *string        `json:"issueType"`
	Status         *string        `json:"status"`
	StageID        *string        `json:"stageId"`
	Priority       *string        `json:"priority"`
	Severity       *string        `json:"severity"`
	ContactID      *string        `json:"contactId"`
	ConversationID *string        `json:"conversationId"`
	AssignedToID   *string        `json:"assignedToId"`
	Source         *string        `json:"source"`
	SLADueAt       *time.Time     `json:"slaDueAt"`
	Metadata       map[string]any `json:"metadata"`
}

type ticketCommentRequest struct {
	Body       string `json:"body"`
	IsInternal *bool  `json:"isInternal"`
}

type ticketStageRequest struct {
	Name     *string `json:"name"`
	Status   *string `json:"status"`
	Position *int    `json:"position"`
	Color    *string `json:"color"`
}

type ticketAITriageRequest struct {
	ConversationID string `json:"conversationId"`
	ContactID      string `json:"contactId"`
	TicketID       string `json:"ticketId"`
}

type aiTicketTriageResponse struct {
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	IssueType     string         `json:"issue_type"`
	Status        string         `json:"status"`
	Priority      string         `json:"priority"`
	Severity      string         `json:"severity"`
	Summary       string         `json:"summary"`
	NextAction    string         `json:"next_action"`
	Confidence    float64        `json:"confidence"`
	ModelName     string         `json:"model_name"`
	LatencyMS     int            `json:"latency_ms"`
	UsageMetadata map[string]any `json:"usage_metadata"`
}

var ticketIssueTypes = []string{"complaint", "product_question", "payment", "delivery", "booking", "technical", "refund", "other"}

var defaultTicketStages = []struct {
	Name     string
	Status   string
	Position int
	Color    string
}{
	{Name: "Baru", Status: "new", Position: 10, Color: "#2563EB"},
	{Name: "Triage", Status: "triage", Position: 20, Color: "#7C3AED"},
	{Name: "Diproses", Status: "in_progress", Position: 30, Color: "#0F766E"},
	{Name: "Menunggu Customer", Status: "waiting_customer", Position: 40, Color: "#F59E0B"},
	{Name: "Menunggu Internal", Status: "waiting_internal", Position: 50, Color: "#F97316"},
	{Name: "Resolved", Status: "resolved", Position: 60, Color: "#16A34A"},
	{Name: "Closed", Status: "closed", Position: 70, Color: "#64748B"},
	{Name: "Batal", Status: "cancelled", Position: 80, Color: "#991B1B"},
}

func (s *Server) handleTickets(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	if !s.requireBusinessModule(w, r, "tickets") {
		return
	}
	switch r.Method {
	case http.MethodGet:
		payload, err := s.listTickets(r.Context(), strings.TrimSpace(r.URL.Query().Get("status")), strings.TrimSpace(r.URL.Query().Get("query")))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, payload)
	case http.MethodPost:
		var req ticketRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createTicket(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleTicketRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	if !s.requireBusinessModule(w, r, "tickets") {
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/tickets/"), "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	if path == "ai/triage" {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		s.handleTicketAITriage(w, r)
		return
	}
	if path == "stages" {
		s.handleTicketStages(w, r)
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[0] == "stages" {
		s.handleTicketStageRoutes(w, r, parts[1])
		return
	}
	ticketID := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			item, err := s.ticketDetail(r.Context(), ticketID)
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, pgx.ErrNoRows) {
					status = http.StatusNotFound
				}
				writeJSON(w, status, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodPatch:
			var req ticketRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
				return
			}
			item, err := s.updateTicket(r.Context(), ticketID, req)
			if err != nil {
				status := http.StatusBadRequest
				if errors.Is(err, pgx.ErrNoRows) {
					status = http.StatusNotFound
				}
				writeJSON(w, status, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"item": item})
		default:
			http.NotFound(w, r)
		}
		return
	}
	if len(parts) == 2 && parts[1] == "comments" {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var req ticketCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, comments, err := s.createTicketComment(r.Context(), ticketID, req)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, pgx.ErrNoRows) {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item, "comments": comments})
		return
	}
	http.NotFound(w, r)
}

func (s *Server) listTickets(ctx context.Context, statusFilter, query string) (map[string]any, error) {
	statusFilter = strings.TrimSpace(statusFilter)
	if statusFilter == "all" {
		statusFilter = ""
	}
	query = strings.TrimSpace(query)
	rows, err := s.db.Query(ctx, ticketListQuery()+`
		WHERE t.organization_id = $1
		  AND ($2 = '' OR t.status = $2)
		  AND (
		    $3 = ''
		    OR lower(t.ticket_number || ' ' || t.title || ' ' || t.description || ' ' || COALESCE(ct.name, '') || ' ' || COALESCE(ct.phone, '')) LIKE '%' || lower($3) || '%'
		  )
		ORDER BY
		  CASE t.priority WHEN 'urgent' THEN 1 WHEN 'high' THEN 2 WHEN 'normal' THEN 3 ELSE 4 END,
		  COALESCE(t.sla_due_at, t.updated_at) ASC,
		  t.updated_at DESC
		LIMIT 120
	`, s.organizationID(ctx), statusFilter, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanTicketRows(rows)
	if err != nil {
		return nil, err
	}
	summary, err := s.ticketSummary(ctx)
	if err != nil {
		return nil, err
	}
	stages, err := s.listTicketStages(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items, "summary": summary, "stages": stages, "canManage": true}, nil
}

func (s *Server) ticketDetail(ctx context.Context, id string) (map[string]any, error) {
	item, err := s.ticketByID(ctx, id)
	if err != nil {
		return nil, err
	}
	comments, err := s.ticketComments(ctx, id)
	if err != nil {
		return nil, err
	}
	events, err := s.ticketEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	stages, err := s.listTicketStages(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"item": item, "comments": comments, "events": events, "stages": stages}, nil
}

func (s *Server) handleTicketStages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		stages, err := s.listTicketStages(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"stages": stages})
	case http.MethodPost:
		if !canManageBusinessModules(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage ticket stages"})
			return
		}
		var req ticketStageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		stage, err := s.createTicketStage(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"stage": stage})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleTicketStageRoutes(w http.ResponseWriter, r *http.Request, id string) {
	if !canManageBusinessModules(s.role(r.Context())) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage ticket stages"})
		return
	}
	switch r.Method {
	case http.MethodPatch:
		var req ticketStageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		stage, err := s.updateTicketStage(r.Context(), id, req)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, pgx.ErrNoRows) {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"stage": stage})
	case http.MethodDelete:
		if err := s.deleteTicketStage(r.Context(), id); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, pgx.ErrNoRows) {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) ticketByID(ctx context.Context, id string) (map[string]any, error) {
	rows, err := s.db.Query(ctx, ticketListQuery()+`
		WHERE t.id = NULLIF($1, '')::uuid
		  AND t.organization_id = $2
		LIMIT 1
	`, id, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanTicketRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, pgx.ErrNoRows
	}
	return items[0], nil
}

func ticketListQuery() string {
	return `
		SELECT
		  t.id::text,
		  t.ticket_number,
		  t.title,
		  t.description,
		  t.issue_type,
		  t.status,
		  COALESCE(t.stage_id::text, ''),
		  COALESCE(ts.name, ''),
		  COALESCE(ts.status, ''),
		  COALESCE(ts.color, ''),
		  t.priority,
		  t.severity,
		  COALESCE(t.contact_id::text, ''),
		  COALESCE(NULLIF(ct.name, ''), ct.phone, ''),
		  COALESCE(ct.phone, ''),
		  COALESCE(t.conversation_id::text, ''),
		  COALESCE(t.whatsapp_session_id::text, ''),
		  COALESCE(ws.label, ''),
		  COALESCE(ws.provider, 'whatsmeow'),
		  COALESCE(t.ai_agent_id::text, ''),
		  COALESCE(aa.name, ''),
		  COALESCE(t.assigned_to::text, ''),
		  COALESCE(assignee.name, ''),
		  t.source,
		  t.sla_due_at,
		  t.resolved_at,
		  t.closed_at,
		  COALESCE(t.metadata, '{}'::jsonb)::text,
		  t.created_at,
		  t.updated_at,
		  inbound.last_customer_message_at,
		  COALESCE(comment_stats.comment_count, 0)
		FROM tickets t
		LEFT JOIN ticket_stages ts ON ts.id = t.stage_id AND ts.organization_id = t.organization_id
		LEFT JOIN contacts ct ON ct.id = t.contact_id AND ct.organization_id = t.organization_id
		LEFT JOIN conversations c ON c.id = t.conversation_id AND c.organization_id = t.organization_id
		LEFT JOIN whatsapp_sessions ws ON ws.id = t.whatsapp_session_id AND ws.organization_id = t.organization_id
		LEFT JOIN ai_agents aa ON aa.id = t.ai_agent_id AND aa.organization_id = t.organization_id
		LEFT JOIN agents assignee ON assignee.id = t.assigned_to
		LEFT JOIN LATERAL (
		  SELECT MAX(COALESCE(sent_at, created_at)) AS last_customer_message_at
		  FROM messages
		  WHERE conversation_id = c.id
		    AND organization_id = t.organization_id
		    AND direction = 'inbound'
		) inbound ON TRUE
		LEFT JOIN LATERAL (
		  SELECT COUNT(*) AS comment_count
		  FROM ticket_comments
		  WHERE ticket_id = t.id
		    AND organization_id = t.organization_id
		) comment_stats ON TRUE
	`
}

func scanTicketRows(rows pgx.Rows) ([]map[string]any, error) {
	items := []map[string]any{}
	for rows.Next() {
		var id, ticketNumber, title, description, issueType, status, priority, severity string
		var stageID, stageName, stageStatus, stageColor string
		var contactID, contactName, contactPhone, conversationID, whatsappSessionID, whatsappSessionName, whatsappProvider string
		var aiAgentID, aiAgentName, assignedToID, assignedToName, source, metadataJSON string
		var slaDueAt, resolvedAt, closedAt, lastCustomerMessageAt *time.Time
		var createdAt, updatedAt time.Time
		var commentCount int
		if err := rows.Scan(
			&id,
			&ticketNumber,
			&title,
			&description,
			&issueType,
			&status,
			&stageID,
			&stageName,
			&stageStatus,
			&stageColor,
			&priority,
			&severity,
			&contactID,
			&contactName,
			&contactPhone,
			&conversationID,
			&whatsappSessionID,
			&whatsappSessionName,
			&whatsappProvider,
			&aiAgentID,
			&aiAgentName,
			&assignedToID,
			&assignedToName,
			&source,
			&slaDueAt,
			&resolvedAt,
			&closedAt,
			&metadataJSON,
			&createdAt,
			&updatedAt,
			&lastCustomerMessageAt,
			&commentCount,
		); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":                    id,
			"ticketNumber":          ticketNumber,
			"title":                 title,
			"description":           description,
			"issueType":             issueType,
			"status":                status,
			"stageId":               stageID,
			"stageName":             stageName,
			"stageStatus":           stageStatus,
			"stageColor":            stageColor,
			"priority":              priority,
			"severity":              severity,
			"contactId":             contactID,
			"contactName":           contactName,
			"contactPhone":          contactPhone,
			"conversationId":        conversationID,
			"whatsappSessionId":     whatsappSessionID,
			"whatsappSession":       whatsappSessionName,
			"whatsappProvider":      whatsappProvider,
			"aiAgentId":             aiAgentID,
			"aiAgentName":           aiAgentName,
			"assignedToId":          assignedToID,
			"assignedToName":        assignedToName,
			"source":                source,
			"slaDueAt":              slaDueAt,
			"resolvedAt":            resolvedAt,
			"closedAt":              closedAt,
			"metadata":              parseJSONObject(metadataJSON),
			"createdAt":             createdAt,
			"updatedAt":             updatedAt,
			"lastCustomerMessageAt": lastCustomerMessageAt,
			"whatsappWindow":        newWhatsAppCustomerWindowInfo(lastCustomerMessageAt, time.Now()),
			"commentCount":          commentCount,
		})
	}
	return items, rows.Err()
}

func (s *Server) ticketSummary(ctx context.Context) (map[string]any, error) {
	var total, open, overdue, urgent int
	err := s.db.QueryRow(ctx, `
		SELECT
		  COUNT(*),
		  COUNT(*) FILTER (WHERE status NOT IN ('resolved', 'closed', 'cancelled')),
		  COUNT(*) FILTER (WHERE status NOT IN ('resolved', 'closed', 'cancelled') AND sla_due_at IS NOT NULL AND sla_due_at < NOW()),
		  COUNT(*) FILTER (WHERE status NOT IN ('resolved', 'closed', 'cancelled') AND priority = 'urgent')
		FROM tickets
		WHERE organization_id = $1
	`, s.organizationID(ctx)).Scan(&total, &open, &overdue, &urgent)
	if err != nil {
		return nil, err
	}
	return map[string]any{"total": total, "open": open, "overdue": overdue, "urgent": urgent}, nil
}

func (s *Server) ensureDefaultTicketStages(ctx context.Context) error {
	var count int
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM ticket_stages WHERE organization_id = $1`, s.organizationID(ctx)).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for _, stage := range defaultTicketStages {
		if _, err := s.db.Exec(ctx, `
			INSERT INTO ticket_stages (organization_id, name, status, position, color)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT DO NOTHING
		`, s.organizationID(ctx), stage.Name, stage.Status, stage.Position, stage.Color); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) listTicketStages(ctx context.Context) ([]map[string]any, error) {
	if err := s.ensureDefaultTicketStages(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, status, position, color, created_at, updated_at
		FROM ticket_stages
		WHERE organization_id = $1
		ORDER BY position ASC, created_at ASC
	`, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name, status, color string
		var position int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &status, &position, &color, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":        id,
			"name":      name,
			"status":    status,
			"position":  position,
			"color":     color,
			"createdAt": createdAt,
			"updatedAt": updatedAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) ticketStageStatus(ctx context.Context, stageID string) (string, error) {
	stageID = strings.TrimSpace(stageID)
	if stageID == "" {
		return "", errors.New("ticket stage is required")
	}
	if err := s.ensureDefaultTicketStages(ctx); err != nil {
		return "", err
	}
	var status string
	err := s.db.QueryRow(ctx, `
		SELECT status
		FROM ticket_stages
		WHERE id = NULLIF($1, '')::uuid
		  AND organization_id = $2
	`, stageID, s.organizationID(ctx)).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("ticket stage not found")
		}
		return "", err
	}
	return normalizeTicketStatus(status), nil
}

func (s *Server) defaultTicketStage(ctx context.Context, status string) (string, string, error) {
	if err := s.ensureDefaultTicketStages(ctx); err != nil {
		return "", "", err
	}
	status = normalizeTicketStatus(status)
	if status == "" {
		status = "new"
	}
	var id, stageStatus string
	err := s.db.QueryRow(ctx, `
		SELECT id::text, status
		FROM ticket_stages
		WHERE organization_id = $1
		  AND status = $2
		ORDER BY position ASC, created_at ASC
		LIMIT 1
	`, s.organizationID(ctx), status).Scan(&id, &stageStatus)
	if err == nil {
		return id, normalizeTicketStatus(stageStatus), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", "", err
	}
	err = s.db.QueryRow(ctx, `
		SELECT id::text, status
		FROM ticket_stages
		WHERE organization_id = $1
		ORDER BY position ASC, created_at ASC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&id, &stageStatus)
	return id, normalizeTicketStatus(stageStatus), err
}

func (s *Server) createTicketStage(ctx context.Context, req ticketStageRequest) (map[string]any, error) {
	name := strings.TrimSpace(valueOrEmpty(req.Name))
	if name == "" {
		return nil, errors.New("stage name is required")
	}
	status := normalizeTicketStatus(valueOrDefault(req.Status, "new"))
	if status == "" {
		return nil, errors.New("invalid stage status")
	}
	position := 0
	if req.Position != nil {
		position = *req.Position
	}
	if position <= 0 {
		_ = s.db.QueryRow(ctx, `SELECT COALESCE(MAX(position), 0) + 10 FROM ticket_stages WHERE organization_id = $1`, s.organizationID(ctx)).Scan(&position)
	}
	color := normalizeStageColor(valueOrDefault(req.Color, "#2196F3"))
	var id string
	if err := s.db.QueryRow(ctx, `
		INSERT INTO ticket_stages (organization_id, name, status, position, color)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text
	`, s.organizationID(ctx), name, status, position, color).Scan(&id); err != nil {
		return nil, err
	}
	return s.ticketStageByID(ctx, id)
}

func (s *Server) updateTicketStage(ctx context.Context, id string, req ticketStageRequest) (map[string]any, error) {
	current, err := s.ticketStageByID(ctx, id)
	if err != nil {
		return nil, err
	}
	name := stringFromMap(current, "name")
	status := stringFromMap(current, "status")
	position := intFromMap(current, "position")
	color := stringFromMap(current, "color")
	if req.Name != nil {
		name = strings.TrimSpace(valueOrEmpty(req.Name))
	}
	if req.Status != nil {
		status = normalizeTicketStatus(valueOrEmpty(req.Status))
	}
	if req.Position != nil {
		position = *req.Position
	}
	if req.Color != nil {
		color = normalizeStageColor(valueOrEmpty(req.Color))
	}
	if name == "" {
		return nil, errors.New("stage name is required")
	}
	if status == "" {
		return nil, errors.New("invalid stage status")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE ticket_stages
		SET name = $2,
		    status = $3,
		    position = $4,
		    color = $5,
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid
		  AND organization_id = $6
	`, id, name, status, position, color, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	if _, err := tx.Exec(ctx, `
		UPDATE tickets
		SET status = $2,
		    resolved_at = CASE WHEN $2 = 'resolved' AND resolved_at IS NULL THEN NOW() WHEN $2 NOT IN ('resolved', 'closed') THEN NULL ELSE resolved_at END,
		    closed_at = CASE WHEN $2 = 'closed' AND closed_at IS NULL THEN NOW() WHEN $2 <> 'closed' THEN NULL ELSE closed_at END,
		    updated_at = NOW()
		WHERE organization_id = $1
		  AND stage_id = NULLIF($3, '')::uuid
	`, s.organizationID(ctx), status, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ticketStageByID(ctx, id)
}

func (s *Server) deleteTicketStage(ctx context.Context, id string) error {
	if err := s.ensureDefaultTicketStages(ctx); err != nil {
		return err
	}
	var stageCount int
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM ticket_stages WHERE organization_id = $1`, s.organizationID(ctx)).Scan(&stageCount); err != nil {
		return err
	}
	if stageCount <= 1 {
		return errors.New("minimal harus ada satu stage tiket")
	}
	fallbackID, fallbackStatus, err := s.defaultTicketStage(ctx, "new")
	if err != nil {
		return err
	}
	if fallbackID == id {
		err = s.db.QueryRow(ctx, `
			SELECT id::text, status
			FROM ticket_stages
			WHERE organization_id = $1
			  AND id <> NULLIF($2, '')::uuid
			ORDER BY position ASC, created_at ASC
			LIMIT 1
		`, s.organizationID(ctx), id).Scan(&fallbackID, &fallbackStatus)
		if err != nil {
			return err
		}
		fallbackStatus = normalizeTicketStatus(fallbackStatus)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE tickets
		SET stage_id = NULLIF($3, '')::uuid,
		    status = $4,
		    resolved_at = CASE WHEN $4 = 'resolved' AND resolved_at IS NULL THEN NOW() WHEN $4 NOT IN ('resolved', 'closed') THEN NULL ELSE resolved_at END,
		    closed_at = CASE WHEN $4 = 'closed' AND closed_at IS NULL THEN NOW() WHEN $4 <> 'closed' THEN NULL ELSE closed_at END,
		    updated_at = NOW()
		WHERE organization_id = $1
		  AND stage_id = NULLIF($2, '')::uuid
	`, s.organizationID(ctx), id, fallbackID, fallbackStatus)
	if err != nil {
		return err
	}
	_ = tag
	tag, err = tx.Exec(ctx, `
		DELETE FROM ticket_stages
		WHERE id = NULLIF($1, '')::uuid
		  AND organization_id = $2
	`, id, s.organizationID(ctx))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return tx.Commit(ctx)
}

func (s *Server) ticketStageByID(ctx context.Context, id string) (map[string]any, error) {
	var name, status, color string
	var position int
	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(ctx, `
		SELECT name, status, position, color, created_at, updated_at
		FROM ticket_stages
		WHERE id = NULLIF($1, '')::uuid
		  AND organization_id = $2
	`, id, s.organizationID(ctx)).Scan(&name, &status, &position, &color, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":        id,
		"name":      name,
		"status":    status,
		"position":  position,
		"color":     color,
		"createdAt": createdAt,
		"updatedAt": updatedAt,
	}, nil
}

func (s *Server) createTicket(ctx context.Context, req ticketRequest) (map[string]any, error) {
	title := strings.TrimSpace(valueOrEmpty(req.Title))
	if title == "" {
		return nil, errors.New("ticket title is required")
	}
	description := strings.TrimSpace(valueOrEmpty(req.Description))
	issueType := normalizeTicketIssueType(valueOrDefault(req.IssueType, "other"))
	status := normalizeTicketStatus(valueOrDefault(req.Status, "new"))
	stageID := strings.TrimSpace(valueOrEmpty(req.StageID))
	if stageID != "" {
		stageStatus, err := s.ticketStageStatus(ctx, stageID)
		if err != nil {
			return nil, err
		}
		status = stageStatus
	}
	priority := normalizeTicketPriority(valueOrDefault(req.Priority, "normal"))
	severity := normalizeTicketSeverity(valueOrDefault(req.Severity, "minor"))
	if issueType == "" || status == "" || priority == "" || severity == "" {
		return nil, errors.New("invalid ticket classification")
	}
	if stageID == "" {
		var err error
		stageID, _, err = s.defaultTicketStage(ctx, status)
		if err != nil {
			return nil, err
		}
	}
	source := strings.TrimSpace(valueOrDefault(req.Source, "dashboard"))
	contactID, conversationID, assignedToID := valueOrEmpty(req.ContactID), valueOrEmpty(req.ConversationID), valueOrEmpty(req.AssignedToID)
	resolvedContactID, whatsappSessionID, aiAgentID, err := s.resolveTicketLinks(ctx, contactID, conversationID, assignedToID)
	if err != nil {
		return nil, err
	}
	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var ticketID string
	baseSeq, err := s.nextTicketSequenceTx(ctx, tx)
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 6; attempt++ {
		ticketNumber := fmt.Sprintf("TCK-%s-%04d", time.Now().UTC().Format("20060102"), baseSeq+attempt)
		err = tx.QueryRow(ctx, `
			INSERT INTO tickets (
			  organization_id, ticket_number, title, description, issue_type, status, stage_id, priority, severity,
			  contact_id, conversation_id, whatsapp_session_id, ai_agent_id, assigned_to, source, sla_due_at,
			  created_by, updated_by, metadata, resolved_at, closed_at
			)
			VALUES (
			  $1, $2, $3, $4, $5, $6, NULLIF($7, '')::uuid, $8, $9,
			  NULLIF($10, '')::uuid, NULLIF($11, '')::uuid, NULLIF($12, '')::uuid, NULLIF($13, '')::uuid, NULLIF($14, '')::uuid, $15, $16,
			  NULLIF($17, '')::uuid, NULLIF($17, '')::uuid, $18::jsonb,
			  CASE WHEN $6 = 'resolved' THEN NOW() ELSE NULL END,
			  CASE WHEN $6 = 'closed' THEN NOW() ELSE NULL END
			)
			RETURNING id::text
		`, s.organizationID(ctx), ticketNumber, title, description, issueType, status, stageID, priority, severity, resolvedContactID, conversationID, whatsappSessionID, aiAgentID, assignedToID, source, req.SLADueAt, s.agentID(ctx), marshalJSON(metadata)).Scan(&ticketID)
		if err == nil {
			break
		}
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, err
		}
	}
	if ticketID == "" {
		return nil, errors.New("could not allocate ticket number")
	}
	if err := s.insertTicketEventTx(ctx, tx, ticketID, "ticket_created", map[string]any{"source": source, "status": status, "stageId": stageID}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ticketByID(ctx, ticketID)
}

func (s *Server) updateTicket(ctx context.Context, id string, req ticketRequest) (map[string]any, error) {
	current, err := s.ticketByID(ctx, id)
	if err != nil {
		return nil, err
	}
	title := stringFromMap(current, "title")
	description := stringFromMap(current, "description")
	issueType := stringFromMap(current, "issueType")
	status := stringFromMap(current, "status")
	stageID := stringFromMap(current, "stageId")
	priority := stringFromMap(current, "priority")
	severity := stringFromMap(current, "severity")
	contactID := stringFromMap(current, "contactId")
	conversationID := stringFromMap(current, "conversationId")
	assignedToID := stringFromMap(current, "assignedToId")
	source := stringFromMap(current, "source")
	metadata, _ := current["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}
	var slaDueAt *time.Time
	if value, ok := current["slaDueAt"].(*time.Time); ok {
		slaDueAt = value
	}

	if req.Title != nil {
		title = strings.TrimSpace(valueOrEmpty(req.Title))
	}
	if req.Description != nil {
		description = strings.TrimSpace(valueOrEmpty(req.Description))
	}
	if req.IssueType != nil {
		issueType = normalizeTicketIssueType(valueOrEmpty(req.IssueType))
	}
	if req.Status != nil {
		status = normalizeTicketStatus(valueOrEmpty(req.Status))
	}
	if req.StageID != nil {
		stageID = strings.TrimSpace(valueOrEmpty(req.StageID))
		if stageID != "" {
			stageStatus, err := s.ticketStageStatus(ctx, stageID)
			if err != nil {
				return nil, err
			}
			status = stageStatus
		}
	}
	if req.Priority != nil {
		priority = normalizeTicketPriority(valueOrEmpty(req.Priority))
	}
	if req.Severity != nil {
		severity = normalizeTicketSeverity(valueOrEmpty(req.Severity))
	}
	if req.ContactID != nil {
		contactID = valueOrEmpty(req.ContactID)
	}
	if req.ConversationID != nil {
		conversationID = valueOrEmpty(req.ConversationID)
	}
	if req.AssignedToID != nil {
		assignedToID = valueOrEmpty(req.AssignedToID)
	}
	if req.Source != nil {
		source = strings.TrimSpace(valueOrEmpty(req.Source))
	}
	if req.SLADueAt != nil {
		slaDueAt = req.SLADueAt
	}
	if req.Metadata != nil {
		metadata = req.Metadata
	}
	if title == "" {
		return nil, errors.New("ticket title is required")
	}
	if issueType == "" || status == "" || priority == "" || severity == "" {
		return nil, errors.New("invalid ticket classification")
	}
	if stageID == "" || (req.Status != nil && req.StageID == nil) {
		var err error
		stageID, _, err = s.defaultTicketStage(ctx, status)
		if err != nil {
			return nil, err
		}
	}
	resolvedContactID, whatsappSessionID, aiAgentID, err := s.resolveTicketLinks(ctx, contactID, conversationID, assignedToID)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE tickets
		SET title = $2,
		    description = $3,
		    issue_type = $4,
		    status = $5,
		    stage_id = NULLIF($6, '')::uuid,
		    priority = $7,
		    severity = $8,
		    contact_id = NULLIF($9, '')::uuid,
		    conversation_id = NULLIF($10, '')::uuid,
		    whatsapp_session_id = NULLIF($11, '')::uuid,
		    ai_agent_id = NULLIF($12, '')::uuid,
		    assigned_to = NULLIF($13, '')::uuid,
		    source = $14,
		    sla_due_at = $15,
		    metadata = $16::jsonb,
		    resolved_at = CASE WHEN $5 = 'resolved' AND resolved_at IS NULL THEN NOW() WHEN $5 NOT IN ('resolved', 'closed') THEN NULL ELSE resolved_at END,
		    closed_at = CASE WHEN $5 = 'closed' AND closed_at IS NULL THEN NOW() WHEN $5 <> 'closed' THEN NULL ELSE closed_at END,
		    updated_by = NULLIF($17, '')::uuid,
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid
		  AND organization_id = $18
	`, id, title, description, issueType, status, stageID, priority, severity, resolvedContactID, conversationID, whatsappSessionID, aiAgentID, assignedToID, source, slaDueAt, marshalJSON(metadata), s.agentID(ctx), s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	if err := s.insertTicketEventTx(ctx, tx, id, "ticket_updated", map[string]any{"status": status, "stageId": stageID, "priority": priority, "severity": severity}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ticketByID(ctx, id)
}

func (s *Server) createTicketComment(ctx context.Context, ticketID string, req ticketCommentRequest) (map[string]any, []map[string]any, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return nil, nil, errors.New("comment body is required")
	}
	isInternal := true
	if req.IsInternal != nil {
		isInternal = *req.IsInternal
	}
	if _, err := s.ticketByID(ctx, ticketID); err != nil {
		return nil, nil, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		INSERT INTO ticket_comments (organization_id, ticket_id, body, is_internal, created_by)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, NULLIF($5, '')::uuid)
	`, s.organizationID(ctx), ticketID, body, isInternal, s.agentID(ctx))
	if err != nil {
		return nil, nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE tickets SET updated_at = NOW(), updated_by = NULLIF($2, '')::uuid WHERE id = NULLIF($1, '')::uuid AND organization_id = $3`, ticketID, s.agentID(ctx), s.organizationID(ctx)); err != nil {
		return nil, nil, err
	}
	if err := s.insertTicketEventTx(ctx, tx, ticketID, "comment_added", map[string]any{"is_internal": isInternal}); err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	item, err := s.ticketByID(ctx, ticketID)
	if err != nil {
		return nil, nil, err
	}
	comments, err := s.ticketComments(ctx, ticketID)
	return item, comments, err
}

func (s *Server) ticketComments(ctx context.Context, ticketID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT tc.id::text, tc.body, tc.is_internal, COALESCE(a.name, ''), tc.created_at
		FROM ticket_comments tc
		LEFT JOIN agents a ON a.id = tc.created_by
		WHERE tc.organization_id = $1
		  AND tc.ticket_id = NULLIF($2, '')::uuid
		ORDER BY tc.created_at DESC
		LIMIT 80
	`, s.organizationID(ctx), ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, body, authorName string
		var isInternal bool
		var createdAt time.Time
		if err := rows.Scan(&id, &body, &isInternal, &authorName, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"id": id, "body": body, "isInternal": isInternal, "authorName": authorName, "createdAt": createdAt})
	}
	return items, rows.Err()
}

func (s *Server) ticketEvents(ctx context.Context, ticketID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT te.id::text, te.event_type, COALESCE(te.payload, '{}'::jsonb)::text, COALESCE(a.name, ''), te.created_at
		FROM ticket_events te
		LEFT JOIN agents a ON a.id = te.created_by
		WHERE te.organization_id = $1
		  AND te.ticket_id = NULLIF($2, '')::uuid
		ORDER BY te.created_at DESC
		LIMIT 80
	`, s.organizationID(ctx), ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, eventType, payloadJSON, authorName string
		var createdAt time.Time
		if err := rows.Scan(&id, &eventType, &payloadJSON, &authorName, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"id": id, "eventType": eventType, "payload": parseJSONObject(payloadJSON), "authorName": authorName, "createdAt": createdAt})
	}
	return items, rows.Err()
}

func (s *Server) insertTicketEventTx(ctx context.Context, tx pgx.Tx, ticketID, eventType string, payload map[string]any) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO ticket_events (organization_id, ticket_id, event_type, payload, created_by)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4::jsonb, NULLIF($5, '')::uuid)
	`, s.organizationID(ctx), ticketID, eventType, marshalJSON(payload), s.agentID(ctx))
	return err
}

func (s *Server) nextTicketSequenceTx(ctx context.Context, tx pgx.Tx) (int, error) {
	prefix := fmt.Sprintf("TCK-%s-", time.Now().UTC().Format("20060102"))
	var count int
	err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM tickets WHERE organization_id = $1 AND ticket_number LIKE $2`, s.organizationID(ctx), prefix+"%").Scan(&count)
	return count + 1, err
}

func (s *Server) resolveTicketLinks(ctx context.Context, contactID, conversationID, assignedToID string) (string, string, string, error) {
	contactID = strings.TrimSpace(contactID)
	conversationID = strings.TrimSpace(conversationID)
	assignedToID = strings.TrimSpace(assignedToID)
	whatsappSessionID := ""
	aiAgentID := ""
	if conversationID != "" {
		var conversationContactID string
		err := s.db.QueryRow(ctx, `
			SELECT contact_id::text, COALESCE(whatsapp_session_id::text, ''), COALESCE(ai_agent_id::text, '')
			FROM conversations
			WHERE id = NULLIF($1, '')::uuid
			  AND organization_id = $2
		`, conversationID, s.organizationID(ctx)).Scan(&conversationContactID, &whatsappSessionID, &aiAgentID)
		if err != nil {
			return "", "", "", errors.New("conversation not found")
		}
		if contactID == "" {
			contactID = conversationContactID
		} else if contactID != conversationContactID {
			return "", "", "", errors.New("contact does not match conversation")
		}
	}
	if contactID != "" && !s.contactExists(ctx, contactID) {
		return "", "", "", errors.New("contact not found")
	}
	if assignedToID != "" && !s.agentBelongsToOrganization(ctx, assignedToID) {
		return "", "", "", errors.New("assignee not found")
	}
	return contactID, whatsappSessionID, aiAgentID, nil
}

func normalizeTicketStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "new", "triage", "waiting_customer", "waiting_internal", "in_progress", "resolved", "closed", "cancelled":
		return strings.ToLower(strings.TrimSpace(value))
	case "open":
		return "in_progress"
	default:
		return ""
	}
}

func normalizeTicketPriority(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "normal", "high", "urgent":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func normalizeTicketSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "minor", "moderate", "major", "critical":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func normalizeTicketIssueType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "support", "general_support", "customer_support":
		return "other"
	case "question":
		return "product_question"
	}
	for _, item := range ticketIssueTypes {
		if normalized == item {
			return normalized
		}
	}
	return ""
}

func (s *Server) handleTicketAITriage(w http.ResponseWriter, r *http.Request) {
	allowed, err := s.businessModuleAIAllowed(r.Context(), "tickets", "draft")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "enable Ticketing AI draft mode before running AI triage"})
		return
	}
	var req ticketAITriageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.ConversationID = strings.TrimSpace(req.ConversationID)
	req.ContactID = strings.TrimSpace(req.ContactID)
	req.TicketID = strings.TrimSpace(req.TicketID)
	if req.ConversationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversationId is required"})
		return
	}
	if ok, needed, err := s.ensureCreditsAvailable(r.Context(), 1000, 220); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	} else if !ok {
		if err := s.deliverAICreditAlert(r.Context(), needed); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"error": "insufficient AI credits", "neededCredits": needed})
		return
	}
	contextPayload, err := s.ticketTriageConversationContext(r.Context(), req.ConversationID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	triage, err := s.callTicketTriageAI(r.Context(), contextPayload)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	if triage.UsageMetadata == nil {
		triage.UsageMetadata = map[string]any{}
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
		VALUES ($1, NULL, 'answer', $2, $3, $4, NULL, $5::jsonb, $6, $7, $8, $9)
		RETURNING id
	`, req.ConversationID, "Ticket AI triage", nullableString(triage.Summary), triage.Confidence, marshalJSON(map[string]any{
		"mode":       "ticket_triage",
		"ticketId":   req.TicketID,
		"suggestion": ticketSuggestionMap(triage),
	}), triage.ModelName, triage.LatencyMS, aiSettingsUpdatedAt, s.organizationID(r.Context())).Scan(&aiRunID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	decision := aiDecisionResponse{ModelName: triage.ModelName, UsageMetadata: triage.UsageMetadata}
	inputTokens, outputTokens, embeddingTokens, actualCostUSD := usageTokensFromDecision(decision, 1000, 220, 0)
	usageResult, err := s.logCreditUsage(r.Context(), req.ConversationID, "", aiRunID, "ticket_ai_triage", triage.ModelName, inputTokens, outputTokens, embeddingTokens, actualCostUSD)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertAIRunCostSteps(r.Context(), aiRunID, decision, usageResult); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	triage.UsageMetadata["input_tokens"] = inputTokens
	triage.UsageMetadata["output_tokens"] = outputTokens
	triage.UsageMetadata["embedding_tokens"] = embeddingTokens
	triage.UsageMetadata["cost_usd"] = usageResult.CostUSD
	triage.UsageMetadata["cost_idr"] = usageResult.CostIDR
	triage.UsageMetadata["credits_used"] = usageResult.CreditsUsed
	triage.UsageMetadata["creditsUsed"] = usageResult.CreditsUsed
	triage.UsageMetadata["credit_source"] = usageResult.CreditSource
	triage.UsageMetadata["creditSource"] = usageResult.CreditSource
	triage.UsageMetadata["monthly_credits_remaining"] = usageResult.MonthlyRemaining
	triage.UsageMetadata["additional_credits_remaining"] = usageResult.AdditionalRemaining
	writeJSON(w, http.StatusOK, map[string]any{
		"suggestion": ticketSuggestionMap(triage),
		"usage":      triage.UsageMetadata,
		"aiRunId":    aiRunID,
		"wallet": map[string]any{
			"monthlyRemaining":    usageResult.MonthlyRemaining,
			"additionalRemaining": usageResult.AdditionalRemaining,
		},
	})
}

func ticketSuggestionMap(item aiTicketTriageResponse) map[string]any {
	return map[string]any{
		"title":       item.Title,
		"description": item.Description,
		"issueType":   normalizeTicketIssueType(item.IssueType),
		"status":      normalizeTicketStatus(item.Status),
		"priority":    normalizeTicketPriority(item.Priority),
		"severity":    normalizeTicketSeverity(item.Severity),
		"summary":     item.Summary,
		"nextAction":  item.NextAction,
		"confidence":  item.Confidence,
	}
}

func (s *Server) ticketTriageConversationContext(ctx context.Context, conversationID string) (map[string]any, error) {
	var contactID, contactName, contactPhone, aiAgentID string
	err := s.db.QueryRow(ctx, `
		SELECT ct.id::text, COALESCE(NULLIF(ct.name, ''), ct.phone), ct.phone, COALESCE(c.ai_agent_id::text, '')
		FROM conversations c
		JOIN contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
		WHERE c.id = NULLIF($1, '')::uuid
		  AND c.organization_id = $2
	`, conversationID, s.organizationID(ctx)).Scan(&contactID, &contactName, &contactPhone, &aiAgentID)
	if err != nil {
		return nil, errors.New("conversation not found")
	}
	rows, err := s.db.Query(ctx, `
		SELECT direction, COALESCE(text, ''), message_at
		FROM (
		  SELECT direction::text AS direction, COALESCE(text, '') AS text, COALESCE(sent_at, created_at) AS message_at, created_at, id
		  FROM messages
		  WHERE conversation_id = NULLIF($1, '')::uuid
		    AND organization_id = $2
		    AND COALESCE(BTRIM(text), '') <> ''
		  ORDER BY COALESCE(sent_at, created_at) DESC, created_at DESC, id DESC
		  LIMIT 18
		) recent
		ORDER BY message_at ASC, created_at ASC, id ASC
	`, conversationID, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lines := []string{}
	lastMessageText := ""
	for rows.Next() {
		var direction, text string
		var messageAt time.Time
		if err := rows.Scan(&direction, &text, &messageAt); err != nil {
			return nil, err
		}
		speaker := "Agent"
		if direction == "inbound" {
			speaker = "Customer"
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		lastMessageText = text
		lines = append(lines, fmt.Sprintf("%s (%s): %s", speaker, messageAt.Format(time.RFC3339), text))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, errors.New("conversation has no text messages to triage")
	}
	return map[string]any{
		"conversation_id":     conversationID,
		"organization_id":     s.organizationID(ctx),
		"ai_agent_id":         aiAgentID,
		"contact_id":          contactID,
		"customer_name":       contactName,
		"customer_phone":      contactPhone,
		"conversation_text":   strings.Join(lines, "\n"),
		"last_message_text":   lastMessageText,
		"allowed_issue_types": ticketIssueTypes,
		"allowed_statuses":    []string{"new", "triage", "waiting_customer", "waiting_internal", "in_progress"},
		"allowed_priorities":  []string{"low", "normal", "high", "urgent"},
		"allowed_severities":  []string{"minor", "moderate", "major", "critical"},
	}, nil
}

func (s *Server) callTicketTriageAI(ctx context.Context, bodyPayload map[string]any) (aiTicketTriageResponse, error) {
	body, _ := json.Marshal(bodyPayload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.AIServiceBaseURL+"/api/ticket-triage", bytes.NewReader(body))
	if err != nil {
		return aiTicketTriageResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return aiTicketTriageResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return aiTicketTriageResponse{}, fmt.Errorf("ai-service returned %d", resp.StatusCode)
	}
	var payload aiTicketTriageResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return aiTicketTriageResponse{}, err
	}
	payload.IssueType = normalizeTicketIssueType(payload.IssueType)
	if payload.IssueType == "" {
		payload.IssueType = "other"
	}
	payload.Status = normalizeTicketStatus(payload.Status)
	if payload.Status == "" {
		payload.Status = "triage"
	}
	payload.Priority = normalizeTicketPriority(payload.Priority)
	if payload.Priority == "" {
		payload.Priority = "normal"
	}
	payload.Severity = normalizeTicketSeverity(payload.Severity)
	if payload.Severity == "" {
		payload.Severity = "minor"
	}
	return payload, nil
}
