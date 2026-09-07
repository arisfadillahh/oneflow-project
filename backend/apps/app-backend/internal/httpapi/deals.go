package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type dealRequest struct {
	Title             *string `json:"title"`
	ContactID         *string `json:"contactId"`
	ConversationID    *string `json:"conversationId"`
	PipelineID        *string `json:"pipelineId"`
	StageID           *string `json:"stageId"`
	ValueAmount       *int64  `json:"valueAmount"`
	Currency          *string `json:"currency"`
	Status            *string `json:"status"`
	Priority          *string `json:"priority"`
	OwnerAgentID      *string `json:"ownerAgentId"`
	ExpectedCloseDate *string `json:"expectedCloseDate"`
	Source            *string `json:"source"`
	Notes             *string `json:"notes"`
	LossReason        *string `json:"lossReason"`
}

type dealActivityRequest struct {
	Type         *string    `json:"type"`
	Title        *string    `json:"title"`
	Body         *string    `json:"body"`
	DueAt        *time.Time `json:"dueAt"`
	Done         *bool      `json:"done"`
	AssignedToID *string    `json:"assignedToId"`
}

type dealPipelineRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsDefault   *bool   `json:"isDefault"`
}

type dealStageRequest struct {
	PipelineID  *string `json:"pipelineId"`
	Name        *string `json:"name"`
	Probability *int    `json:"probability"`
	Position    *int    `json:"position"`
	Type        *string `json:"type"`
	Color       *string `json:"color"`
}

func (s *Server) handleDeals(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	if !s.requireBusinessModule(w, r, "prospects") {
		return
	}
	switch r.Method {
	case http.MethodGet:
		board, err := s.dealBoard(r.Context(), r.URL.Query().Get("pipelineId"))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, board)
	case http.MethodPost:
		var req dealRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createDeal(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleDealRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	if !s.requireBusinessModule(w, r, "prospects") {
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/deals/"), "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(path, "/")
	dealID := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodPatch:
			var req dealRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
				return
			}
			item, err := s.updateDeal(r.Context(), dealID, req)
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
	if len(parts) == 2 && parts[1] == "activities" {
		switch r.Method {
		case http.MethodGet:
			items, err := s.listDealActivities(r.Context(), dealID)
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, pgx.ErrNoRows) {
					status = http.StatusNotFound
				}
				writeJSON(w, status, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"items": items})
		case http.MethodPost:
			var req dealActivityRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
				return
			}
			item, err := s.createDealActivity(r.Context(), dealID, req)
			if err != nil {
				status := http.StatusBadRequest
				if errors.Is(err, pgx.ErrNoRows) {
					status = http.StatusNotFound
				}
				writeJSON(w, status, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"item": item})
		default:
			http.NotFound(w, r)
		}
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleDealPipelines(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		if _, err := s.ensureDefaultDealPipeline(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		items, err := s.listDealPipelines(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var req dealPipelineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createDealPipeline(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleDealPipelineRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/deal-pipelines/"), "/")
	if id == "" || r.Method != http.MethodPatch {
		http.NotFound(w, r)
		return
	}
	var req dealPipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	item, err := s.updateDealPipeline(r.Context(), id, req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (s *Server) handleDealStages(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req dealStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	item, err := s.createDealStage(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (s *Server) handleDealStageRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/deal-stages/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodDelete {
		if err := s.deleteDealStage(r.Context(), id); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, pgx.ErrNoRows) {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
		return
	}
	if r.Method != http.MethodPatch {
		http.NotFound(w, r)
		return
	}
	var req dealStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	item, err := s.updateDealStage(r.Context(), id, req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (s *Server) handleDealActivityRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/deal-activities/"), "/")
	if id == "" || r.Method != http.MethodPatch {
		http.NotFound(w, r)
		return
	}
	var req dealActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	item, err := s.updateDealActivity(r.Context(), id, req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (s *Server) dealBoard(ctx context.Context, pipelineID string) (map[string]any, error) {
	pipelineID = strings.TrimSpace(pipelineID)
	if pipelineID == "" {
		var err error
		pipelineID, err = s.ensureDefaultDealPipeline(ctx)
		if err != nil {
			return nil, err
		}
	} else if !s.pipelineBelongsToOrganization(ctx, pipelineID) {
		return nil, errors.New("pipeline not found")
	}
	pipelines, err := s.listDealPipelines(ctx)
	if err != nil {
		return nil, err
	}
	stages, err := s.listDealStages(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	deals, err := s.listDeals(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	summary, err := s.dealSummary(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"activePipelineId": pipelineID,
		"pipelines":        pipelines,
		"stages":           stages,
		"deals":            deals,
		"summary":          summary,
	}, nil
}

func (s *Server) ensureDefaultDealPipeline(ctx context.Context) (string, error) {
	var id string
	err := s.db.QueryRow(ctx, `
		SELECT id::text
		FROM deal_pipelines
		WHERE organization_id = $1 AND is_default = TRUE
		ORDER BY created_at ASC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&id)
	if err == nil {
		return id, s.ensureDefaultDealStages(ctx, id)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO deal_pipelines (organization_id, name, description, is_default, created_by)
		VALUES ($1, 'Sales Pipeline', 'Pipeline default untuk prospek dan penjualan.', TRUE, NULLIF($2, '')::uuid)
		ON CONFLICT DO NOTHING
		RETURNING id::text
	`, s.organizationID(ctx), s.agentID(ctx)).Scan(&id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if id == "" {
		err = tx.QueryRow(ctx, `
			SELECT id::text
			FROM deal_pipelines
			WHERE organization_id = $1 AND (is_default = TRUE OR lower(name) = lower('Sales Pipeline'))
			ORDER BY is_default DESC, created_at ASC
			LIMIT 1
		`, s.organizationID(ctx)).Scan(&id)
		if err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE deal_pipelines p
			SET is_default = TRUE, updated_at = NOW()
			WHERE p.id = NULLIF($1, '')::uuid
			  AND p.organization_id = $2
			  AND NOT EXISTS (
			    SELECT 1 FROM deal_pipelines other
			    WHERE other.organization_id = p.organization_id
			      AND other.is_default = TRUE
			      AND other.id <> p.id
			  )
		`, id, s.organizationID(ctx)); err != nil {
			return "", err
		}
	}
	if err := insertDefaultDealStages(ctx, tx, s.organizationID(ctx), id); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Server) ensureDefaultDealStages(ctx context.Context, pipelineID string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := insertDefaultDealStages(ctx, tx, s.organizationID(ctx), pipelineID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func insertDefaultDealStages(ctx context.Context, tx pgx.Tx, organizationID string, pipelineID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO deal_stages (organization_id, pipeline_id, name, probability, position, stage_type, color)
		VALUES
		  ($1, NULLIF($2, '')::uuid, 'Lead Baru', 10, 10, 'open', '#2196F3'),
		  ($1, NULLIF($2, '')::uuid, 'Qualified', 30, 20, 'open', '#2563EB'),
		  ($1, NULLIF($2, '')::uuid, 'Proposal', 55, 30, 'open', '#7C4DFF'),
		  ($1, NULLIF($2, '')::uuid, 'Negotiation', 75, 40, 'open', '#EA580C'),
		  ($1, NULLIF($2, '')::uuid, 'Won', 100, 90, 'won', '#16A34A'),
		  ($1, NULLIF($2, '')::uuid, 'Lost', 0, 100, 'lost', '#DC2626')
		ON CONFLICT DO NOTHING
	`, organizationID, pipelineID)
	return err
}

func (s *Server) listDealPipelines(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, COALESCE(description, ''), is_default, created_at, updated_at
		FROM deal_pipelines
		WHERE organization_id = $1
		ORDER BY is_default DESC, created_at ASC
	`, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name, description string
		var isDefault bool
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &description, &isDefault, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":          id,
			"name":        name,
			"description": description,
			"isDefault":   isDefault,
			"createdAt":   createdAt,
			"updatedAt":   updatedAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) createDealPipeline(ctx context.Context, req dealPipelineRequest) (map[string]any, error) {
	name := valueOrEmpty(req.Name)
	description := valueOrEmpty(req.Description)
	if name == "" {
		return nil, errors.New("pipeline name is required")
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if isDefault {
		if _, err := tx.Exec(ctx, `UPDATE deal_pipelines SET is_default = FALSE, updated_at = NOW() WHERE organization_id = $1`, s.organizationID(ctx)); err != nil {
			return nil, err
		}
	}
	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO deal_pipelines (organization_id, name, description, is_default, created_by)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, '')::uuid)
		RETURNING id::text
	`, s.organizationID(ctx), name, description, isDefault, s.agentID(ctx)).Scan(&id); err != nil {
		return nil, err
	}
	if err := insertDefaultDealStages(ctx, tx, s.organizationID(ctx), id); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.dealPipelineByID(ctx, id)
}

func (s *Server) updateDealPipeline(ctx context.Context, id string, req dealPipelineRequest) (map[string]any, error) {
	current, err := s.dealPipelineByID(ctx, id)
	if err != nil {
		return nil, err
	}
	name := stringFromMap(current, "name")
	description := stringFromMap(current, "description")
	isDefault, _ := current["isDefault"].(bool)
	if req.Name != nil {
		name = valueOrEmpty(req.Name)
	}
	if req.Description != nil {
		description = valueOrEmpty(req.Description)
	}
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}
	if name == "" {
		return nil, errors.New("pipeline name is required")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if isDefault {
		if _, err := tx.Exec(ctx, `UPDATE deal_pipelines SET is_default = FALSE, updated_at = NOW() WHERE organization_id = $1 AND id <> NULLIF($2, '')::uuid`, s.organizationID(ctx), id); err != nil {
			return nil, err
		}
	}
	tag, err := tx.Exec(ctx, `
		UPDATE deal_pipelines
		SET name = $2, description = NULLIF($3, ''), is_default = $4, updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $5
	`, id, name, description, isDefault, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.dealPipelineByID(ctx, id)
}

func (s *Server) dealPipelineByID(ctx context.Context, id string) (map[string]any, error) {
	var name, description string
	var isDefault bool
	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(ctx, `
		SELECT name, COALESCE(description, ''), is_default, created_at, updated_at
		FROM deal_pipelines
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, id, s.organizationID(ctx)).Scan(&name, &description, &isDefault, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "name": name, "description": description, "isDefault": isDefault, "createdAt": createdAt, "updatedAt": updatedAt}, nil
}

func (s *Server) listDealStages(ctx context.Context, pipelineID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, pipeline_id::text, name, probability, position, stage_type, color, created_at, updated_at
		FROM deal_stages
		WHERE organization_id = $1 AND pipeline_id = NULLIF($2, '')::uuid
		ORDER BY position ASC, created_at ASC
	`, s.organizationID(ctx), pipelineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, nextPipelineID, name, stageType, color string
		var probability, position int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &nextPipelineID, &name, &probability, &position, &stageType, &color, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":          id,
			"pipelineId":  nextPipelineID,
			"name":        name,
			"probability": probability,
			"position":    position,
			"type":        stageType,
			"color":       color,
			"createdAt":   createdAt,
			"updatedAt":   updatedAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) createDealStage(ctx context.Context, req dealStageRequest) (map[string]any, error) {
	pipelineID := valueOrEmpty(req.PipelineID)
	if pipelineID == "" || !s.pipelineBelongsToOrganization(ctx, pipelineID) {
		return nil, errors.New("pipeline not found")
	}
	name := valueOrEmpty(req.Name)
	if name == "" {
		return nil, errors.New("stage name is required")
	}
	stageType := valueOrDefault(req.Type, "open")
	if !validDealStageType(stageType) {
		return nil, errors.New("invalid stage type")
	}
	probability := 20
	if req.Probability != nil {
		probability = *req.Probability
	}
	if probability < 0 || probability > 100 {
		return nil, errors.New("probability must be 0-100")
	}
	position := 0
	if req.Position != nil {
		position = *req.Position
	}
	if position <= 0 {
		_ = s.db.QueryRow(ctx, `SELECT COALESCE(MAX(position), 0) + 10 FROM deal_stages WHERE organization_id = $1 AND pipeline_id = NULLIF($2, '')::uuid`, s.organizationID(ctx), pipelineID).Scan(&position)
	}
	color := valueOrDefault(req.Color, "#2196F3")
	var id string
	if err := s.db.QueryRow(ctx, `
		INSERT INTO deal_stages (organization_id, pipeline_id, name, probability, position, stage_type, color)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7)
		RETURNING id::text
	`, s.organizationID(ctx), pipelineID, name, probability, position, stageType, color).Scan(&id); err != nil {
		return nil, err
	}
	return s.dealStageByID(ctx, id)
}

func (s *Server) updateDealStage(ctx context.Context, id string, req dealStageRequest) (map[string]any, error) {
	current, err := s.dealStageByID(ctx, id)
	if err != nil {
		return nil, err
	}
	name := stringFromMap(current, "name")
	stageType := stringFromMap(current, "type")
	color := stringFromMap(current, "color")
	probability := intFromMap(current, "probability")
	position := intFromMap(current, "position")
	if req.Name != nil {
		name = valueOrEmpty(req.Name)
	}
	if req.Type != nil {
		stageType = valueOrEmpty(req.Type)
	}
	if req.Color != nil {
		color = valueOrEmpty(req.Color)
	}
	if req.Probability != nil {
		probability = *req.Probability
	}
	if req.Position != nil {
		position = *req.Position
	}
	if name == "" || !validDealStageType(stageType) || probability < 0 || probability > 100 || position <= 0 {
		return nil, errors.New("invalid stage")
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE deal_stages
		SET name = $2, probability = $3, position = $4, stage_type = $5, color = $6, updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $7
	`, id, name, probability, position, stageType, color, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return s.dealStageByID(ctx, id)
}

func (s *Server) deleteDealStage(ctx context.Context, id string) error {
	current, err := s.dealStageByID(ctx, id)
	if err != nil {
		return pgx.ErrNoRows
	}
	pipelineID := stringFromMap(current, "pipelineId")
	var stageCount int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM deal_stages
		WHERE organization_id = $1 AND pipeline_id = NULLIF($2, '')::uuid
	`, s.organizationID(ctx), pipelineID).Scan(&stageCount); err != nil {
		return err
	}
	if stageCount <= 1 {
		return errors.New("minimal harus ada satu stage follow-up")
	}
	var fallbackStageID string
	if err := s.db.QueryRow(ctx, `
		SELECT id::text
		FROM deal_stages
		WHERE organization_id = $1
		  AND pipeline_id = NULLIF($2, '')::uuid
		  AND id <> NULLIF($3, '')::uuid
		ORDER BY position ASC
		LIMIT 1
	`, s.organizationID(ctx), pipelineID, id).Scan(&fallbackStageID); err != nil {
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE deals
		SET stage_id = NULLIF($3, '')::uuid, updated_at = NOW()
		WHERE organization_id = $1 AND stage_id = NULLIF($2, '')::uuid
	`, s.organizationID(ctx), id, fallbackStageID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		DELETE FROM deal_stages
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, id, s.organizationID(ctx))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if err := s.insertAuditLogTx(ctx, tx, "deal_stage.delete", "deal_stage", id, map[string]any{"pipelineId": pipelineID, "fallbackStageId": fallbackStageID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) dealStageByID(ctx context.Context, id string) (map[string]any, error) {
	var pipelineID, name, stageType, color string
	var probability, position int
	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(ctx, `
		SELECT pipeline_id::text, name, probability, position, stage_type, color, created_at, updated_at
		FROM deal_stages
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, id, s.organizationID(ctx)).Scan(&pipelineID, &name, &probability, &position, &stageType, &color, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":          id,
		"pipelineId":  pipelineID,
		"name":        name,
		"probability": probability,
		"position":    position,
		"type":        stageType,
		"color":       color,
		"createdAt":   createdAt,
		"updatedAt":   updatedAt,
	}, nil
}

func (s *Server) listDeals(ctx context.Context, pipelineID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
		  d.id::text,
		  d.pipeline_id::text,
		  d.stage_id::text,
		  COALESCE(st.name, ''),
		  st.stage_type,
		  COALESCE(d.contact_id::text, ''),
		  COALESCE(ct.name, ''),
		  COALESCE(ct.phone, ''),
		  COALESCE(d.conversation_id::text, ''),
		  d.title,
		  d.value_amount,
		  d.currency,
		  d.status,
		  d.priority,
		  COALESCE(d.owner_agent_id::text, ''),
		  COALESCE(owner.name, ''),
		  d.expected_close_date,
		  COALESCE(d.source, ''),
		  COALESCE(d.notes, ''),
		  COALESCE(d.loss_reason, ''),
		  d.won_at,
		  d.lost_at,
		  d.created_at,
		  d.updated_at,
		  activity_stats.last_activity_at,
		  COALESCE(activity_stats.open_activity_count, 0)
		FROM deals d
		JOIN deal_stages st ON st.id = d.stage_id AND st.organization_id = d.organization_id
		LEFT JOIN contacts ct ON ct.id = d.contact_id AND ct.organization_id = d.organization_id
		LEFT JOIN agents owner ON owner.id = d.owner_agent_id
		LEFT JOIN LATERAL (
		  SELECT
		    MAX(created_at) AS last_activity_at,
		    COUNT(*) FILTER (WHERE activity_type = 'task' AND completed_at IS NULL) AS open_activity_count
		  FROM deal_activities
		  WHERE deal_id = d.id AND organization_id = d.organization_id
		) activity_stats ON TRUE
		WHERE d.organization_id = $1
		  AND d.pipeline_id = NULLIF($2, '')::uuid
		  AND d.status <> 'archived'
		ORDER BY st.position ASC, d.updated_at DESC
	`, s.organizationID(ctx), pipelineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, nextPipelineID, stageID, stageName, stageType, contactID, contactName, contactPhone, conversationID string
		var title, currency, status, priority, ownerAgentID, ownerName, source, notes, lossReason string
		var valueAmount int64
		var expectedCloseDate, wonAt, lostAt, lastActivityAt *time.Time
		var createdAt, updatedAt time.Time
		var openActivityCount int
		if err := rows.Scan(
			&id,
			&nextPipelineID,
			&stageID,
			&stageName,
			&stageType,
			&contactID,
			&contactName,
			&contactPhone,
			&conversationID,
			&title,
			&valueAmount,
			&currency,
			&status,
			&priority,
			&ownerAgentID,
			&ownerName,
			&expectedCloseDate,
			&source,
			&notes,
			&lossReason,
			&wonAt,
			&lostAt,
			&createdAt,
			&updatedAt,
			&lastActivityAt,
			&openActivityCount,
		); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":                id,
			"pipelineId":        nextPipelineID,
			"stageId":           stageID,
			"stageName":         stageName,
			"stageType":         stageType,
			"contactId":         contactID,
			"contactName":       contactName,
			"contactPhone":      contactPhone,
			"conversationId":    conversationID,
			"title":             title,
			"valueAmount":       valueAmount,
			"currency":          currency,
			"status":            status,
			"priority":          priority,
			"ownerAgentId":      ownerAgentID,
			"ownerName":         ownerName,
			"expectedCloseDate": formatDateValue(expectedCloseDate),
			"source":            source,
			"notes":             notes,
			"lossReason":        lossReason,
			"wonAt":             wonAt,
			"lostAt":            lostAt,
			"createdAt":         createdAt,
			"updatedAt":         updatedAt,
			"lastActivityAt":    lastActivityAt,
			"openActivityCount": openActivityCount,
		})
	}
	return items, rows.Err()
}

func (s *Server) dealSummary(ctx context.Context, pipelineID string) (map[string]any, error) {
	var openCount, wonCount, lostCount, closingSoon int
	var openValue, weightedValue, wonValue int64
	err := s.db.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE d.status = 'open'),
		  COALESCE(SUM(d.value_amount) FILTER (WHERE d.status = 'open'), 0),
		  COALESCE(SUM((d.value_amount * st.probability) / 100) FILTER (WHERE d.status = 'open'), 0),
		  COUNT(*) FILTER (WHERE d.status = 'won' AND d.won_at >= date_trunc('month', NOW())),
		  COALESCE(SUM(d.value_amount) FILTER (WHERE d.status = 'won' AND d.won_at >= date_trunc('month', NOW())), 0),
		  COUNT(*) FILTER (WHERE d.status = 'lost'),
		  COUNT(*) FILTER (WHERE d.status = 'open' AND d.expected_close_date IS NOT NULL AND d.expected_close_date <= CURRENT_DATE + INTERVAL '14 days')
		FROM deals d
		JOIN deal_stages st ON st.id = d.stage_id AND st.organization_id = d.organization_id
		WHERE d.organization_id = $1
		  AND d.pipeline_id = NULLIF($2, '')::uuid
		  AND d.status <> 'archived'
	`, s.organizationID(ctx), pipelineID).Scan(&openCount, &openValue, &weightedValue, &wonCount, &wonValue, &lostCount, &closingSoon)
	if err != nil {
		return nil, err
	}
	var overdueActivities, dueTodayActivities int
	if err := s.db.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE a.due_at < NOW()),
		  COUNT(*) FILTER (WHERE a.due_at >= date_trunc('day', NOW()) AND a.due_at < date_trunc('day', NOW()) + INTERVAL '1 day')
		FROM deal_activities a
		JOIN deals d ON d.id = a.deal_id AND d.organization_id = a.organization_id
		WHERE a.organization_id = $1
		  AND d.pipeline_id = NULLIF($2, '')::uuid
		  AND d.status = 'open'
		  AND a.activity_type = 'task'
		  AND a.completed_at IS NULL
		  AND a.due_at IS NOT NULL
	`, s.organizationID(ctx), pipelineID).Scan(&overdueActivities, &dueTodayActivities); err != nil {
		return nil, err
	}
	return map[string]any{
		"openCount":          openCount,
		"openValue":          openValue,
		"weightedValue":      weightedValue,
		"wonCount":           wonCount,
		"wonValue":           wonValue,
		"lostCount":          lostCount,
		"closingSoon":        closingSoon,
		"overdueActivities":  overdueActivities,
		"dueTodayActivities": dueTodayActivities,
	}, nil
}

func (s *Server) createDeal(ctx context.Context, req dealRequest) (map[string]any, error) {
	pipelineID := valueOrEmpty(req.PipelineID)
	if pipelineID == "" {
		var err error
		pipelineID, err = s.ensureDefaultDealPipeline(ctx)
		if err != nil {
			return nil, err
		}
	} else if !s.pipelineBelongsToOrganization(ctx, pipelineID) {
		return nil, errors.New("pipeline not found")
	}
	stageID := valueOrEmpty(req.StageID)
	var stageType string
	var err error
	if stageID == "" {
		stageID, stageType, err = s.defaultDealStage(ctx, pipelineID)
	} else {
		pipelineID, stageType, err = s.dealStageInfo(ctx, stageID)
	}
	if err != nil {
		return nil, err
	}

	title := valueOrDefault(req.Title, "Deal baru")
	contactID := valueOrEmpty(req.ContactID)
	conversationID := valueOrEmpty(req.ConversationID)
	if contactID == "" && conversationID != "" {
		contactID, _ = s.contactIDForConversation(ctx, conversationID)
	}
	if contactID != "" && !s.contactExists(ctx, contactID) {
		return nil, errors.New("contact not found")
	}
	if conversationID != "" && !s.conversationBelongsToOrganization(ctx, conversationID) {
		return nil, errors.New("conversation not found")
	}
	valueAmount := int64(0)
	if req.ValueAmount != nil {
		valueAmount = *req.ValueAmount
	}
	if valueAmount < 0 {
		return nil, errors.New("deal value cannot be negative")
	}
	currency := strings.ToUpper(valueOrDefault(req.Currency, "IDR"))
	status := valueOrDefault(req.Status, dealStatusForStage(stageType))
	priority := valueOrDefault(req.Priority, "normal")
	ownerAgentID := valueOrEmpty(req.OwnerAgentID)
	expectedCloseDate := valueOrEmpty(req.ExpectedCloseDate)
	source := valueOrEmpty(req.Source)
	notes := valueOrEmpty(req.Notes)
	lossReason := valueOrEmpty(req.LossReason)
	if !validDealStatus(status) || !validPriority(priority) || !validDealCurrency(currency) || !validDateInput(expectedCloseDate) {
		return nil, errors.New("invalid deal payload")
	}
	if ownerAgentID != "" && !s.agentBelongsToOrganization(ctx, ownerAgentID) {
		return nil, errors.New("owner must be an active team member")
	}

	var id string
	err = s.db.QueryRow(ctx, `
		INSERT INTO deals (
		  organization_id, pipeline_id, stage_id, contact_id, conversation_id, title,
		  value_amount, currency, status, priority, owner_agent_id, expected_close_date,
		  source, notes, loss_reason, won_at, lost_at, created_by, updated_by
		)
		VALUES (
		  $1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid, NULLIF($5, '')::uuid, $6,
		  $7, $8, $9, $10, NULLIF($11, '')::uuid, NULLIF($12, '')::date,
		  NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, ''),
		  CASE WHEN $9 = 'won' THEN NOW() ELSE NULL END,
		  CASE WHEN $9 = 'lost' THEN NOW() ELSE NULL END,
		  NULLIF($16, '')::uuid, NULLIF($16, '')::uuid
		)
		RETURNING id::text
	`, s.organizationID(ctx), pipelineID, stageID, contactID, conversationID, title, valueAmount, currency, status, priority, ownerAgentID, expectedCloseDate, source, notes, lossReason, s.agentID(ctx)).Scan(&id)
	if err != nil {
		return nil, err
	}
	_ = s.insertDealActivity(ctx, id, "note", "Deal dibuat", "", nil, false, "")
	return s.dealByID(ctx, id)
}

func (s *Server) updateDeal(ctx context.Context, id string, req dealRequest) (map[string]any, error) {
	current, err := s.dealByID(ctx, id)
	if err != nil {
		return nil, err
	}
	pipelineID := stringFromMap(current, "pipelineId")
	stageID := stringFromMap(current, "stageId")
	previousStageID := stageID
	stageType := stringFromMap(current, "stageType")
	title := stringFromMap(current, "title")
	contactID := stringFromMap(current, "contactId")
	conversationID := stringFromMap(current, "conversationId")
	valueAmount := int64FromMap(current, "valueAmount")
	currency := stringFromMap(current, "currency")
	status := stringFromMap(current, "status")
	priority := stringFromMap(current, "priority")
	ownerAgentID := stringFromMap(current, "ownerAgentId")
	expectedCloseDate := stringFromMap(current, "expectedCloseDate")
	source := stringFromMap(current, "source")
	notes := stringFromMap(current, "notes")
	lossReason := stringFromMap(current, "lossReason")

	if req.PipelineID != nil {
		nextPipelineID := valueOrEmpty(req.PipelineID)
		if nextPipelineID == "" || !s.pipelineBelongsToOrganization(ctx, nextPipelineID) {
			return nil, errors.New("pipeline not found")
		}
		pipelineID = nextPipelineID
		if req.StageID == nil {
			stageID, stageType, err = s.defaultDealStage(ctx, pipelineID)
			if err != nil {
				return nil, err
			}
		}
	}
	if req.StageID != nil {
		stageID = valueOrEmpty(req.StageID)
		var nextPipelineID string
		nextPipelineID, stageType, err = s.dealStageInfo(ctx, stageID)
		if err != nil {
			return nil, err
		}
		pipelineID = nextPipelineID
		if req.Status == nil {
			status = dealStatusForStage(stageType)
		}
	}
	if req.Title != nil {
		title = valueOrEmpty(req.Title)
	}
	if req.ContactID != nil {
		contactID = valueOrEmpty(req.ContactID)
	}
	if req.ConversationID != nil {
		conversationID = valueOrEmpty(req.ConversationID)
	}
	if contactID == "" && conversationID != "" {
		contactID, _ = s.contactIDForConversation(ctx, conversationID)
	}
	if req.ValueAmount != nil {
		valueAmount = *req.ValueAmount
	}
	if req.Currency != nil {
		currency = strings.ToUpper(valueOrEmpty(req.Currency))
	}
	if req.Status != nil {
		status = valueOrEmpty(req.Status)
	}
	if req.Priority != nil {
		priority = valueOrEmpty(req.Priority)
	}
	if req.OwnerAgentID != nil {
		ownerAgentID = valueOrEmpty(req.OwnerAgentID)
	}
	if req.ExpectedCloseDate != nil {
		expectedCloseDate = valueOrEmpty(req.ExpectedCloseDate)
	}
	if req.Source != nil {
		source = valueOrEmpty(req.Source)
	}
	if req.Notes != nil {
		notes = valueOrEmpty(req.Notes)
	}
	if req.LossReason != nil {
		lossReason = valueOrEmpty(req.LossReason)
	}
	if title == "" || valueAmount < 0 || !validDealStatus(status) || !validPriority(priority) || !validDealCurrency(currency) || !validDateInput(expectedCloseDate) {
		return nil, errors.New("invalid deal payload")
	}
	if contactID != "" && !s.contactExists(ctx, contactID) {
		return nil, errors.New("contact not found")
	}
	if conversationID != "" && !s.conversationBelongsToOrganization(ctx, conversationID) {
		return nil, errors.New("conversation not found")
	}
	if ownerAgentID != "" && !s.agentBelongsToOrganization(ctx, ownerAgentID) {
		return nil, errors.New("owner must be an active team member")
	}

	_, err = s.db.Exec(ctx, `
		UPDATE deals
		SET pipeline_id = NULLIF($2, '')::uuid,
		    stage_id = NULLIF($3, '')::uuid,
		    contact_id = NULLIF($4, '')::uuid,
		    conversation_id = NULLIF($5, '')::uuid,
		    title = $6,
		    value_amount = $7,
		    currency = $8,
		    status = $9,
		    priority = $10,
		    owner_agent_id = NULLIF($11, '')::uuid,
		    expected_close_date = NULLIF($12, '')::date,
		    source = NULLIF($13, ''),
		    notes = NULLIF($14, ''),
		    loss_reason = NULLIF($15, ''),
		    won_at = CASE WHEN $9 = 'won' THEN COALESCE(won_at, NOW()) ELSE NULL END,
		    lost_at = CASE WHEN $9 = 'lost' THEN COALESCE(lost_at, NOW()) ELSE NULL END,
		    updated_by = NULLIF($16, '')::uuid,
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $17
	`, id, pipelineID, stageID, contactID, conversationID, title, valueAmount, currency, status, priority, ownerAgentID, expectedCloseDate, source, notes, lossReason, s.agentID(ctx), s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	if previousStageID != stageID {
		nextStageName := s.dealStageName(ctx, stageID)
		if nextStageName == "" {
			nextStageName = "stage baru"
		}
		_ = s.insertDealActivity(ctx, id, "stage_change", "Stage diperbarui", fmt.Sprintf("Deal dipindahkan ke %s.", nextStageName), nil, false, "")
	}
	return s.dealByID(ctx, id)
}

func (s *Server) dealByID(ctx context.Context, id string) (map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
		  d.id::text,
		  d.pipeline_id::text,
		  d.stage_id::text,
		  COALESCE(st.name, ''),
		  st.stage_type,
		  COALESCE(d.contact_id::text, ''),
		  COALESCE(ct.name, ''),
		  COALESCE(ct.phone, ''),
		  COALESCE(d.conversation_id::text, ''),
		  d.title,
		  d.value_amount,
		  d.currency,
		  d.status,
		  d.priority,
		  COALESCE(d.owner_agent_id::text, ''),
		  COALESCE(owner.name, ''),
		  d.expected_close_date,
		  COALESCE(d.source, ''),
		  COALESCE(d.notes, ''),
		  COALESCE(d.loss_reason, ''),
		  d.won_at,
		  d.lost_at,
		  d.created_at,
		  d.updated_at,
		  activity_stats.last_activity_at,
		  COALESCE(activity_stats.open_activity_count, 0)
		FROM deals d
		JOIN deal_stages st ON st.id = d.stage_id AND st.organization_id = d.organization_id
		LEFT JOIN contacts ct ON ct.id = d.contact_id AND ct.organization_id = d.organization_id
		LEFT JOIN agents owner ON owner.id = d.owner_agent_id
		LEFT JOIN LATERAL (
		  SELECT
		    MAX(created_at) AS last_activity_at,
		    COUNT(*) FILTER (WHERE activity_type = 'task' AND completed_at IS NULL) AS open_activity_count
		  FROM deal_activities
		  WHERE deal_id = d.id AND organization_id = d.organization_id
		) activity_stats ON TRUE
		WHERE d.organization_id = $1 AND d.id = NULLIF($2, '')::uuid
	`, s.organizationID(ctx), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanDealRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, pgx.ErrNoRows
	}
	return items[0], nil
}

func scanDealRows(rows pgx.Rows) ([]map[string]any, error) {
	items := []map[string]any{}
	for rows.Next() {
		var id, pipelineID, stageID, stageName, stageType, contactID, contactName, contactPhone, conversationID string
		var title, currency, status, priority, ownerAgentID, ownerName, source, notes, lossReason string
		var valueAmount int64
		var expectedCloseDate, wonAt, lostAt, lastActivityAt *time.Time
		var createdAt, updatedAt time.Time
		var openActivityCount int
		if err := rows.Scan(
			&id,
			&pipelineID,
			&stageID,
			&stageName,
			&stageType,
			&contactID,
			&contactName,
			&contactPhone,
			&conversationID,
			&title,
			&valueAmount,
			&currency,
			&status,
			&priority,
			&ownerAgentID,
			&ownerName,
			&expectedCloseDate,
			&source,
			&notes,
			&lossReason,
			&wonAt,
			&lostAt,
			&createdAt,
			&updatedAt,
			&lastActivityAt,
			&openActivityCount,
		); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":                id,
			"pipelineId":        pipelineID,
			"stageId":           stageID,
			"stageName":         stageName,
			"stageType":         stageType,
			"contactId":         contactID,
			"contactName":       contactName,
			"contactPhone":      contactPhone,
			"conversationId":    conversationID,
			"title":             title,
			"valueAmount":       valueAmount,
			"currency":          currency,
			"status":            status,
			"priority":          priority,
			"ownerAgentId":      ownerAgentID,
			"ownerName":         ownerName,
			"expectedCloseDate": formatDateValue(expectedCloseDate),
			"source":            source,
			"notes":             notes,
			"lossReason":        lossReason,
			"wonAt":             wonAt,
			"lostAt":            lostAt,
			"createdAt":         createdAt,
			"updatedAt":         updatedAt,
			"lastActivityAt":    lastActivityAt,
			"openActivityCount": openActivityCount,
		})
	}
	return items, rows.Err()
}

func (s *Server) listDealActivities(ctx context.Context, dealID string) ([]map[string]any, error) {
	if _, err := s.dealByID(ctx, dealID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT
		  a.id::text,
		  a.activity_type,
		  a.title,
		  COALESCE(a.body, ''),
		  a.due_at,
		  a.completed_at,
		  COALESCE(a.assigned_to::text, ''),
		  COALESCE(assignee.name, ''),
		  COALESCE(a.created_by::text, ''),
		  COALESCE(agent.name, ''),
		  a.created_at
		FROM deal_activities a
		LEFT JOIN agents assignee ON assignee.id = a.assigned_to
		LEFT JOIN agents agent ON agent.id = a.created_by
		WHERE a.organization_id = $1 AND a.deal_id = NULLIF($2, '')::uuid
		ORDER BY a.created_at DESC
		LIMIT 80
	`, s.organizationID(ctx), dealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, activityType, title, body, assignedToID, assignedToName, createdByID, createdByName string
		var dueAt, completedAt *time.Time
		var createdAt time.Time
		if err := rows.Scan(&id, &activityType, &title, &body, &dueAt, &completedAt, &assignedToID, &assignedToName, &createdByID, &createdByName, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":             id,
			"type":           activityType,
			"title":          title,
			"body":           body,
			"dueAt":          dueAt,
			"completedAt":    completedAt,
			"assignedToId":   assignedToID,
			"assignedToName": assignedToName,
			"createdById":    createdByID,
			"createdByName":  createdByName,
			"createdAt":      createdAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) createDealActivity(ctx context.Context, dealID string, req dealActivityRequest) (map[string]any, error) {
	if _, err := s.dealByID(ctx, dealID); err != nil {
		return nil, err
	}
	activityType := valueOrDefault(req.Type, "note")
	title := valueOrDefault(req.Title, "Catatan deal")
	body := valueOrEmpty(req.Body)
	done := false
	if req.Done != nil {
		done = *req.Done
	}
	assignedToID := valueOrEmpty(req.AssignedToID)
	if assignedToID != "" && !s.agentBelongsToOrganization(ctx, assignedToID) {
		return nil, errors.New("assignee must be an active team member")
	}
	if title == "" || !validDealActivityType(activityType) {
		return nil, errors.New("invalid activity")
	}
	if err := s.insertDealActivity(ctx, dealID, activityType, title, body, req.DueAt, done, assignedToID); err != nil {
		return nil, err
	}
	items, err := s.listDealActivities(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return map[string]any{"title": title, "type": activityType}, nil
	}
	return items[0], nil
}

func (s *Server) updateDealActivity(ctx context.Context, id string, req dealActivityRequest) (map[string]any, error) {
	current, err := s.dealActivityByID(ctx, id)
	if err != nil {
		return nil, err
	}
	activityType := stringFromMap(current, "type")
	title := stringFromMap(current, "title")
	body := stringFromMap(current, "body")
	dueAt := timePtrFromMap(current, "dueAt")
	done := timePtrFromMap(current, "completedAt") != nil
	assignedToID := stringFromMap(current, "assignedToId")
	if req.Type != nil {
		activityType = valueOrEmpty(req.Type)
	}
	if req.Title != nil {
		title = valueOrEmpty(req.Title)
	}
	if req.Body != nil {
		body = valueOrEmpty(req.Body)
	}
	if req.DueAt != nil {
		dueAt = req.DueAt
	}
	if req.Done != nil {
		done = *req.Done
	}
	if req.AssignedToID != nil {
		assignedToID = valueOrEmpty(req.AssignedToID)
	}
	if assignedToID != "" && !s.agentBelongsToOrganization(ctx, assignedToID) {
		return nil, errors.New("assignee must be an active team member")
	}
	if title == "" || !validDealActivityType(activityType) {
		return nil, errors.New("invalid activity")
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE deal_activities
		SET activity_type = $2,
		    title = $3,
		    body = NULLIF($4, ''),
		    due_at = $5,
		    completed_at = CASE WHEN $6 THEN COALESCE(completed_at, NOW()) ELSE NULL END,
		    assigned_to = NULLIF($7, '')::uuid
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $8
	`, id, activityType, title, body, dueAt, done, assignedToID, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return s.dealActivityByID(ctx, id)
}

func (s *Server) dealActivityByID(ctx context.Context, id string) (map[string]any, error) {
	var activityID, activityType, title, body, dealID, assignedToID, assignedToName, createdByID, createdByName string
	var dueAt, completedAt *time.Time
	var createdAt time.Time
	err := s.db.QueryRow(ctx, `
		SELECT
		  a.id::text,
		  a.activity_type,
		  a.title,
		  COALESCE(a.body, ''),
		  a.deal_id::text,
		  a.due_at,
		  a.completed_at,
		  COALESCE(a.assigned_to::text, ''),
		  COALESCE(assignee.name, ''),
		  COALESCE(a.created_by::text, ''),
		  COALESCE(agent.name, ''),
		  a.created_at
		FROM deal_activities a
		LEFT JOIN agents assignee ON assignee.id = a.assigned_to
		LEFT JOIN agents agent ON agent.id = a.created_by
		WHERE a.id = NULLIF($1, '')::uuid AND a.organization_id = $2
	`, id, s.organizationID(ctx)).Scan(&activityID, &activityType, &title, &body, &dealID, &dueAt, &completedAt, &assignedToID, &assignedToName, &createdByID, &createdByName, &createdAt)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":             activityID,
		"type":           activityType,
		"title":          title,
		"body":           body,
		"dealId":         dealID,
		"dueAt":          dueAt,
		"completedAt":    completedAt,
		"assignedToId":   assignedToID,
		"assignedToName": assignedToName,
		"createdById":    createdByID,
		"createdByName":  createdByName,
		"createdAt":      createdAt,
	}, nil
}

func (s *Server) insertDealActivity(ctx context.Context, dealID string, activityType string, title string, body string, dueAt *time.Time, done bool, assignedToID string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO deal_activities (organization_id, deal_id, activity_type, title, body, due_at, completed_at, assigned_to, created_by)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, NULLIF($5, ''), $6, CASE WHEN $7 THEN NOW() ELSE NULL END, NULLIF($8, '')::uuid, NULLIF($9, '')::uuid)
	`, s.organizationID(ctx), dealID, activityType, title, body, dueAt, done, assignedToID, s.agentID(ctx))
	return err
}

func (s *Server) pipelineBelongsToOrganization(ctx context.Context, pipelineID string) bool {
	var exists bool
	_ = s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM deal_pipelines WHERE id = NULLIF($1, '')::uuid AND organization_id = $2)`, pipelineID, s.organizationID(ctx)).Scan(&exists)
	return exists
}

func (s *Server) conversationBelongsToOrganization(ctx context.Context, conversationID string) bool {
	var exists bool
	_ = s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM conversations WHERE id = NULLIF($1, '')::uuid AND organization_id = $2)`, conversationID, s.organizationID(ctx)).Scan(&exists)
	return exists
}

func (s *Server) contactIDForConversation(ctx context.Context, conversationID string) (string, error) {
	var contactID string
	err := s.db.QueryRow(ctx, `
		SELECT contact_id::text
		FROM conversations
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, conversationID, s.organizationID(ctx)).Scan(&contactID)
	return contactID, err
}

func (s *Server) defaultDealStage(ctx context.Context, pipelineID string) (string, string, error) {
	var id, stageType string
	err := s.db.QueryRow(ctx, `
		SELECT id::text, stage_type
		FROM deal_stages
		WHERE organization_id = $1 AND pipeline_id = NULLIF($2, '')::uuid AND stage_type = 'open'
		ORDER BY position ASC
		LIMIT 1
	`, s.organizationID(ctx), pipelineID).Scan(&id, &stageType)
	if err != nil {
		return "", "", err
	}
	return id, stageType, nil
}

func (s *Server) dealStageInfo(ctx context.Context, stageID string) (string, string, error) {
	var pipelineID, stageType string
	err := s.db.QueryRow(ctx, `
		SELECT pipeline_id::text, stage_type
		FROM deal_stages
		WHERE organization_id = $1 AND id = NULLIF($2, '')::uuid
	`, s.organizationID(ctx), stageID).Scan(&pipelineID, &stageType)
	return pipelineID, stageType, err
}

func (s *Server) dealStageName(ctx context.Context, stageID string) string {
	var name string
	_ = s.db.QueryRow(ctx, `
		SELECT name
		FROM deal_stages
		WHERE organization_id = $1 AND id = NULLIF($2, '')::uuid
	`, s.organizationID(ctx), stageID).Scan(&name)
	return name
}

func dealStatusForStage(stageType string) string {
	if stageType == "won" || stageType == "lost" {
		return stageType
	}
	return "open"
}

func validDealStatus(value string) bool {
	switch value {
	case "open", "won", "lost", "archived":
		return true
	default:
		return false
	}
}

func validDealStageType(value string) bool {
	switch value {
	case "open", "won", "lost":
		return true
	default:
		return false
	}
}

func validDealCurrency(value string) bool {
	if value == "" || len(value) > 8 {
		return false
	}
	return value == strings.ToUpper(value)
}

func validDateInput(value string) bool {
	if value == "" {
		return true
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func validDealActivityType(value string) bool {
	switch value {
	case "note", "call", "meeting", "task", "stage_change":
		return true
	default:
		return false
	}
}

func formatDateValue(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}

func stringFromMap(item map[string]any, key string) string {
	return stringFromMapOr(item, key, "")
}

func stringFromMapOr(item map[string]any, key string, fallback string) string {
	if value, ok := item[key].(string); ok {
		return value
	}
	return fallback
}

func int64FromMap(item map[string]any, key string) int64 {
	switch value := item[key].(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	default:
		return 0
	}
}

func intFromMap(item map[string]any, key string) int {
	switch value := item[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func timePtrFromMap(item map[string]any, key string) *time.Time {
	switch value := item[key].(type) {
	case *time.Time:
		return value
	case time.Time:
		return &value
	default:
		return nil
	}
}
