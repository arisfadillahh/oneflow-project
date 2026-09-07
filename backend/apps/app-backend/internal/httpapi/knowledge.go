package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type knowledgeMetadata struct {
	SourceType   string         `json:"source_type"`
	Priority     int            `json:"priority"`
	Locked       bool           `json:"locked"`
	Topic        string         `json:"topic"`
	Intent       string         `json:"intent"`
	MetadataJSON map[string]any `json:"metadata_json"`
}

func fallbackKnowledgeMetadata(kind string, status string) knowledgeMetadata {
	normalizedKind := strings.ToLower(strings.TrimSpace(kind))
	if normalizedKind != "faq" && normalizedKind != "record" && normalizedKind != "document" {
		normalizedKind = "document"
	}
	isPublished := strings.EqualFold(strings.TrimSpace(status), "published")
	priority := 20
	if isPublished {
		switch normalizedKind {
		case "faq":
			priority = 90
		case "record":
			priority = 88
		default:
			priority = 80
		}
	}
	return knowledgeMetadata{
		SourceType: normalizedKind,
		Priority:   priority,
		Locked:     normalizedKind == "faq" && isPublished,
		MetadataJSON: map[string]any{
			"organizer": "fallback",
			"kind":      normalizedKind,
			"status":    strings.ToLower(strings.TrimSpace(status)),
		},
	}
}

func normalizeKnowledgeMetadata(meta knowledgeMetadata, fallback knowledgeMetadata) knowledgeMetadata {
	allowedSourceTypes := map[string]bool{
		"faq":            true,
		"document":       true,
		"record":         true,
		"admin_approved": true,
		"admin_rules":    true,
	}
	meta.SourceType = strings.ToLower(strings.TrimSpace(meta.SourceType))
	if !allowedSourceTypes[meta.SourceType] {
		meta.SourceType = fallback.SourceType
	}
	if meta.Priority < 0 || meta.Priority > 100 {
		meta.Priority = fallback.Priority
	}
	if meta.Topic == "" {
		meta.Topic = fallback.Topic
	}
	if meta.Intent == "" {
		meta.Intent = fallback.Intent
	}
	if meta.MetadataJSON == nil {
		meta.MetadataJSON = fallback.MetadataJSON
	}
	if meta.Priority < 90 {
		meta.Locked = false
	}
	return meta
}

func (s *Server) organizeKnowledge(ctx context.Context, payload map[string]any, fallback knowledgeMetadata) knowledgeMetadata {
	body, err := json.Marshal(payload)
	if err != nil {
		return fallback
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.AIServiceBaseURL+"/api/knowledge/organize", bytes.NewReader(body))
	if err != nil {
		return fallback
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fallback
	}

	var meta knowledgeMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return fallback
	}
	return normalizeKnowledgeMetadata(meta, fallback)
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func (s *Server) syncJobPositionKnowledgeRecord(ctx context.Context, id string, actorAgentID string, aiAgentID string, title string, status string, location string, workType string, shortDescription string, applyLink string, isActive bool) error {
	recordStatus := "draft"
	if isActive && strings.EqualFold(strings.TrimSpace(status), "published") {
		recordStatus = "published"
	}
	fields := map[string]any{
		"status":            status,
		"location":          location,
		"work_type":         workType,
		"short_description": shortDescription,
		"apply_link":        applyLink,
		"is_active":         isActive,
	}
	metadata := s.organizeKnowledge(
		ctx,
		map[string]any{
			"kind":    "record",
			"title":   title,
			"content": shortDescription,
			"status":  recordStatus,
		},
		fallbackKnowledgeMetadata("record", recordStatus),
	)
	metadata.MetadataJSON["legacy_table"] = "job_positions"
	metadata.MetadataJSON["legacy_id"] = id
	fieldsJSON, _ := json.Marshal(fields)
	metadataJSON, _ := json.Marshal(metadata.MetadataJSON)

	_, err := s.db.Exec(ctx, `
		INSERT INTO knowledge_records (
		  id, record_type, title, content, fields_json, source_type, priority, locked,
		  status, topic, intent, metadata_json, embedding, created_by, updated_by, organization_id, ai_agent_id
		)
		VALUES (
		  $1, 'job_position', $2, $3, $4::jsonb, $5, $6, $7,
		  $8, NULLIF($9, ''), NULLIF($10, ''), $11::jsonb, NULL, $12, $12, $13, NULLIF($14, '')::uuid
		)
		ON CONFLICT (id) DO UPDATE SET
		  organization_id = EXCLUDED.organization_id,
		  ai_agent_id = EXCLUDED.ai_agent_id,
		  title = EXCLUDED.title,
		  content = EXCLUDED.content,
		  fields_json = EXCLUDED.fields_json,
		  source_type = EXCLUDED.source_type,
		  priority = EXCLUDED.priority,
		  locked = EXCLUDED.locked,
		  status = EXCLUDED.status,
		  topic = EXCLUDED.topic,
		  intent = EXCLUDED.intent,
		  metadata_json = EXCLUDED.metadata_json,
		  embedding = NULL,
		  updated_by = EXCLUDED.updated_by,
		  updated_at = NOW()
	`, id, title, shortDescription, string(fieldsJSON), metadata.SourceType, metadata.Priority, metadata.Locked, recordStatus, metadata.Topic, metadata.Intent, string(metadataJSON), actorAgentID, s.organizationID(ctx), aiAgentID)
	return err
}

func (s *Server) handleKnowledgeFAQs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		aiAgentID := strings.TrimSpace(r.URL.Query().Get("aiAgentId"))
		rows, err := s.db.Query(r.Context(), `
			SELECT id, question, answer, status, published_at, updated_at,
			       source_type, priority, locked, COALESCE(topic, ''), COALESCE(intent, ''), metadata_json
			FROM knowledge_faqs
			WHERE organization_id = $1
			  AND ai_agent_id = NULLIF($2, '')::uuid
			ORDER BY updated_at DESC
		`, s.organizationID(r.Context()), aiAgentID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		items := []map[string]any{}
		for rows.Next() {
			var id, question, answer, status, sourceType, topic, intent string
			var priority int
			var locked bool
			var metadataJSON []byte
			var publishedAt, updatedAt *time.Time
			if err := rows.Scan(&id, &question, &answer, &status, &publishedAt, &updatedAt, &sourceType, &priority, &locked, &topic, &intent, &metadataJSON); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			items = append(items, map[string]any{
				"id":          id,
				"question":    question,
				"answer":      answer,
				"status":      status,
				"publishedAt": publishedAt,
				"updatedAt":   updatedAt,
				"sourceType":  sourceType,
				"priority":    priority,
				"locked":      locked,
				"topic":       topic,
				"intent":      intent,
				"metadata":    decodeJSONMap(metadataJSON),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		role := s.role(r.Context())
		if role == "owner" || role == "operator" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		var req struct {
			Question  string `json:"question"`
			Answer    string `json:"answer"`
			Status    string `json:"status"`
			AIAgentID string `json:"aiAgentId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if strings.TrimSpace(req.Question) == "" || strings.TrimSpace(req.Answer) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "question and answer are required"})
			return
		}
		if strings.TrimSpace(req.Status) == "" {
			req.Status = "published"
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

		metadata := s.organizeKnowledge(
			r.Context(),
			map[string]any{
				"kind":     "faq",
				"question": req.Question,
				"answer":   req.Answer,
				"status":   req.Status,
			},
			fallbackKnowledgeMetadata("faq", req.Status),
		)
		metadataJSON, _ := json.Marshal(metadata.MetadataJSON)

		var id string
		err := s.db.QueryRow(r.Context(), `
			INSERT INTO knowledge_faqs (
			  question, answer, status, created_by, updated_by, published_at,
			  source_type, priority, locked, topic, intent, metadata_json, organization_id, ai_agent_id
			)
			VALUES (
			  $1, $2, $3, $4, $4, CASE WHEN $3 = 'published' THEN NOW() ELSE NULL END,
			  $5, $6, $7, NULLIF($8, ''), NULLIF($9, ''), $10::jsonb, $11, $12
			)
			RETURNING id
		`, req.Question, req.Answer, req.Status, s.agentID(r.Context()), metadata.SourceType, metadata.Priority, metadata.Locked, metadata.Topic, metadata.Intent, string(metadataJSON), s.organizationID(r.Context()), req.AIAgentID).Scan(&id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLog(r.Context(), "knowledge.faq_create", "knowledge_faq", id, map[string]any{"status": req.Status}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "id": id})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleKnowledgeFAQRoutes(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role == "owner" || role == "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/knowledge/faqs/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
			Status   string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if strings.TrimSpace(req.Question) == "" || strings.TrimSpace(req.Answer) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "question and answer are required"})
			return
		}
		if strings.TrimSpace(req.Status) == "" {
			req.Status = "published"
		}

		metadata := s.organizeKnowledge(
			r.Context(),
			map[string]any{
				"kind":     "faq",
				"question": req.Question,
				"answer":   req.Answer,
				"status":   req.Status,
			},
			fallbackKnowledgeMetadata("faq", req.Status),
		)
		metadataJSON, _ := json.Marshal(metadata.MetadataJSON)

		tag, err := s.db.Exec(r.Context(), `
			UPDATE knowledge_faqs
			SET question = $2,
			    answer = $3,
			    status = $4,
			    embedding = NULL,
			    source_type = $5,
			    priority = $6,
			    locked = $7,
			    topic = NULLIF($8, ''),
			    intent = NULLIF($9, ''),
			    metadata_json = $10::jsonb,
			    updated_by = $11,
			    updated_at = NOW(),
			    published_at = CASE
			      WHEN $4 = 'published' AND published_at IS NULL THEN NOW()
			      WHEN $4 <> 'published' THEN NULL
			      ELSE published_at
			    END
			WHERE id = $1 AND organization_id = $12
		`, id, req.Question, req.Answer, req.Status, metadata.SourceType, metadata.Priority, metadata.Locked, metadata.Topic, metadata.Intent, string(metadataJSON), s.agentID(r.Context()), s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "faq not found"})
			return
		}
		if err := s.insertAuditLog(r.Context(), "knowledge.faq_update", "knowledge_faq", id, map[string]any{"status": req.Status}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case http.MethodDelete:
		tag, err := s.db.Exec(r.Context(), `DELETE FROM knowledge_faqs WHERE id = $1 AND organization_id = $2`, id, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "faq not found"})
			return
		}
		if err := s.insertAuditLog(r.Context(), "knowledge.faq_delete", "knowledge_faq", id, map[string]any{}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleKnowledgeDocuments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		aiAgentID := strings.TrimSpace(r.URL.Query().Get("aiAgentId"))
		rows, err := s.db.Query(r.Context(), `
			SELECT
			  d.id,
			  d.title,
			  d.knowledge_type,
			  COALESCE(d.raw_text, ''),
			  COALESCE(d.file_url, ''),
			  d.status,
			  d.published_at,
			  d.updated_at,
			  d.source_type,
			  d.priority,
			  d.locked,
			  COALESCE(d.topic, ''),
			  COALESCE(d.intent, ''),
			  d.metadata_json,
			  COALESCE(chunk_stats.chunk_count, 0),
			  COALESCE(chunk_stats.embedded_chunk_count, 0),
			  COALESCE(chunk_stats.pending_embedding_count, 0),
			  COALESCE(chunk_stats.first_chunk_preview, '')
			FROM knowledge_documents d
			LEFT JOIN LATERAL (
			  SELECT
			    COUNT(*) AS chunk_count,
			    COUNT(*) FILTER (WHERE embedding IS NOT NULL) AS embedded_chunk_count,
			    COUNT(*) FILTER (WHERE embedding IS NULL) AS pending_embedding_count,
			    (ARRAY_AGG(chunk_text ORDER BY chunk_index ASC))[1] AS first_chunk_preview
			  FROM knowledge_chunks
			  WHERE knowledge_document_id = d.id
			) chunk_stats ON TRUE
			WHERE d.organization_id = $1
			  AND d.ai_agent_id = NULLIF($2, '')::uuid
			ORDER BY d.updated_at DESC
		`, s.organizationID(r.Context()), aiAgentID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		items := []map[string]any{}
		for rows.Next() {
			var id, title, knowledgeType, rawText, status, fileURL, sourceType, topic, intent string
			var firstChunkPreview string
			var priority int
			var chunkCount, embeddedChunkCount, pendingEmbeddingCount int
			var locked bool
			var metadataJSON []byte
			var publishedAt, updatedAt *time.Time
			if err := rows.Scan(
				&id,
				&title,
				&knowledgeType,
				&rawText,
				&fileURL,
				&status,
				&publishedAt,
				&updatedAt,
				&sourceType,
				&priority,
				&locked,
				&topic,
				&intent,
				&metadataJSON,
				&chunkCount,
				&embeddedChunkCount,
				&pendingEmbeddingCount,
				&firstChunkPreview,
			); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if fileURL != "" && strings.HasPrefix(fileURL, s.cfg.ObjectStorageEndpoint) {
				fileURL = strings.Replace(fileURL, s.cfg.ObjectStorageEndpoint, s.cfg.ObjectStoragePublicBaseURL, 1)
			}
			embeddingStatus := "not_chunked"
			switch {
			case chunkCount > 0 && pendingEmbeddingCount == 0:
				embeddingStatus = "ready"
			case chunkCount > 0 && embeddedChunkCount > 0:
				embeddingStatus = "partial"
			case chunkCount > 0:
				embeddingStatus = "pending"
			}
			items = append(items, map[string]any{
				"id":                    id,
				"title":                 title,
				"knowledgeType":         knowledgeType,
				"rawText":               rawText,
				"fileUrl":               fileURL,
				"downloadUrl":           "/api/knowledge-download?id=" + id,
				"status":                status,
				"publishedAt":           publishedAt,
				"updatedAt":             updatedAt,
				"sourceType":            sourceType,
				"priority":              priority,
				"locked":                locked,
				"topic":                 topic,
				"intent":                intent,
				"metadata":              decodeJSONMap(metadataJSON),
				"chunkCount":            chunkCount,
				"embeddedChunkCount":    embeddedChunkCount,
				"pendingEmbeddingCount": pendingEmbeddingCount,
				"embeddingStatus":       embeddingStatus,
				"chunkPreview":          firstChunkPreview,
			})
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		role := s.role(r.Context())
		if role == "owner" || role == "operator" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		var req struct {
			Title         string `json:"title"`
			KnowledgeType string `json:"knowledgeType"`
			RawText       string `json:"rawText"`
			Status        string `json:"status"`
			AIAgentID     string `json:"aiAgentId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.RawText) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title and rawText are required"})
			return
		}
		if strings.TrimSpace(req.KnowledgeType) == "" {
			req.KnowledgeType = "text"
		}
		if strings.TrimSpace(req.Status) == "" {
			req.Status = "published"
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

		metadata := s.organizeKnowledge(
			r.Context(),
			map[string]any{
				"kind":           "document",
				"title":          req.Title,
				"content":        req.RawText,
				"status":         req.Status,
				"knowledge_type": req.KnowledgeType,
			},
			fallbackKnowledgeMetadata("document", req.Status),
		)
		metadataJSON, _ := json.Marshal(metadata.MetadataJSON)

		var id string
		err := s.db.QueryRow(r.Context(), `
			INSERT INTO knowledge_documents (
			  title, knowledge_type, raw_text, status, created_by, updated_by, published_at,
			  source_type, priority, locked, topic, intent, metadata_json, organization_id, ai_agent_id
			)
			VALUES (
			  $1, $2, $3, $4, $5, $5, CASE WHEN $4 = 'published' THEN NOW() ELSE NULL END,
			  $6, $7, $8, NULLIF($9, ''), NULLIF($10, ''), $11::jsonb, $12, $13
			)
			RETURNING id
		`, req.Title, req.KnowledgeType, req.RawText, req.Status, s.agentID(r.Context()), metadata.SourceType, metadata.Priority, metadata.Locked, metadata.Topic, metadata.Intent, string(metadataJSON), s.organizationID(r.Context()), req.AIAgentID).Scan(&id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLog(r.Context(), "knowledge.document_create", "knowledge_document", id, map[string]any{"status": req.Status, "knowledge_type": req.KnowledgeType}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "id": id})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleKnowledgeDocumentRoutes(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role == "owner" || role == "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/knowledge/documents/"), "/")
	parts := strings.Split(path, "/")
	id := ""
	if len(parts) > 0 {
		id = strings.TrimSpace(parts[0])
	}
	if id == "" || id == "upload" || len(parts) > 2 {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 2 {
		if parts[1] != "reindex" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var exists bool
		if err := s.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM knowledge_documents WHERE id = $1 AND organization_id = $2)`, id, s.organizationID(r.Context())).Scan(&exists); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
			return
		}
		tx, err := s.db.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer tx.Rollback(r.Context())
		if _, err := tx.Exec(r.Context(), `
			DELETE FROM knowledge_chunks
			WHERE knowledge_document_id = $1
			  AND EXISTS (
			    SELECT 1 FROM knowledge_documents d
			    WHERE d.id = knowledge_chunks.knowledge_document_id
			      AND d.organization_id = $2
			  )
		`, id, s.organizationID(r.Context())); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if _, err := tx.Exec(r.Context(), `UPDATE knowledge_documents SET updated_at = NOW(), updated_by = NULLIF($2, '')::uuid WHERE id = $1 AND organization_id = $3`, id, s.agentID(r.Context()), s.organizationID(r.Context())); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLogTx(r.Context(), tx, "knowledge.document_reindex", "knowledge_document", id, map[string]any{"queued": true}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "queued", "id": id})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Title         string `json:"title"`
			KnowledgeType string `json:"knowledgeType"`
			RawText       string `json:"rawText"`
			Status        string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.RawText) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title and rawText are required"})
			return
		}
		if strings.TrimSpace(req.KnowledgeType) == "" {
			req.KnowledgeType = "text"
		}
		if strings.TrimSpace(req.Status) == "" {
			req.Status = "published"
		}

		metadata := s.organizeKnowledge(
			r.Context(),
			map[string]any{
				"kind":           "document",
				"title":          req.Title,
				"content":        req.RawText,
				"status":         req.Status,
				"knowledge_type": req.KnowledgeType,
			},
			fallbackKnowledgeMetadata("document", req.Status),
		)
		metadataJSON, _ := json.Marshal(metadata.MetadataJSON)

		tag, err := s.db.Exec(r.Context(), `
			UPDATE knowledge_documents
			SET title = $2,
			    knowledge_type = $3,
			    raw_text = $4,
			    status = $5,
			    source_type = $6,
			    priority = $7,
			    locked = $8,
			    topic = NULLIF($9, ''),
			    intent = NULLIF($10, ''),
			    metadata_json = $11::jsonb,
			    updated_by = $12,
			    updated_at = NOW(),
			    published_at = CASE
			      WHEN $5 = 'published' AND published_at IS NULL THEN NOW()
			      WHEN $5 <> 'published' THEN NULL
			      ELSE published_at
			    END
			WHERE id = $1 AND organization_id = $13
		`, id, req.Title, req.KnowledgeType, req.RawText, req.Status, metadata.SourceType, metadata.Priority, metadata.Locked, metadata.Topic, metadata.Intent, string(metadataJSON), s.agentID(r.Context()), s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
			return
		}

		_, _ = s.db.Exec(r.Context(), `
			DELETE FROM knowledge_chunks
			WHERE knowledge_document_id = $1
			  AND EXISTS (
			    SELECT 1 FROM knowledge_documents d
			    WHERE d.id = knowledge_chunks.knowledge_document_id
			      AND d.organization_id = $2
			  )
		`, id, s.organizationID(r.Context()))
		if err := s.insertAuditLog(r.Context(), "knowledge.document_update", "knowledge_document", id, map[string]any{"status": req.Status, "knowledge_type": req.KnowledgeType}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case http.MethodDelete:
		var fileURL string
		if err := s.db.QueryRow(r.Context(), `SELECT COALESCE(file_url, '') FROM knowledge_documents WHERE id = $1 AND organization_id = $2`, id, s.organizationID(r.Context())).Scan(&fileURL); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
			return
		}
		tag, err := s.db.Exec(r.Context(), `DELETE FROM knowledge_documents WHERE id = $1 AND organization_id = $2`, id, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
			return
		}
		if objectKey := s.objectKeyFromURL(fileURL); objectKey != "" {
			_ = s.storage.Delete(r.Context(), objectKey)
		}
		if err := s.insertAuditLog(r.Context(), "knowledge.document_delete", "knowledge_document", id, map[string]any{}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleKnowledgeDocumentUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	role := s.role(r.Context())
	if role == "owner" || role == "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxKnowledgeUploadBytes+(1<<20))
	if err := r.ParseMultipartForm(maxKnowledgeUploadBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse multipart form"})
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	knowledgeType := strings.TrimSpace(r.FormValue("knowledgeType"))
	aiAgentID := strings.TrimSpace(r.FormValue("aiAgentId"))
	if aiAgentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "aiAgentId is required"})
		return
	}
	if ok, err := s.aiAgentBelongsToOrganization(r.Context(), aiAgentID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	} else if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid aiAgentId is required"})
		return
	}
	if title == "" {
		title = "Uploaded Knowledge Document"
	}
	if knowledgeType == "" {
		knowledgeType = "file"
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()
	if header.Size > maxKnowledgeUploadBytes {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uploaded file is too large"})
		return
	}

	allowed := map[string]string{
		".txt":  "text/plain",
		".md":   "text/markdown",
		".csv":  "text/csv",
		".json": "application/json",
		".pdf":  "application/pdf",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	contentType, ok := allowed[ext]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "only .txt, .md, .csv, .json, .pdf, and .docx files are supported for now"})
		return
	}

	content, err := io.ReadAll(io.LimitReader(file, maxKnowledgeUploadBytes+1))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if len(content) > maxKnowledgeUploadBytes {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uploaded file is too large"})
		return
	}
	if len(content) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uploaded file is empty"})
		return
	}

	rawText := ""
	if ext == ".txt" || ext == ".md" || ext == ".csv" || ext == ".json" {
		rawText = strings.TrimSpace(string(content))
		if rawText == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uploaded file is empty"})
			return
		}
	}

	objectKey := safeObjectKey(fmt.Sprintf("org/%s/knowledge-documents", s.organizationID(r.Context())), header.Filename)
	fileURL, err := s.storage.UploadTextFile(r.Context(), objectKey, content, contentType)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	metadata := s.organizeKnowledge(
		r.Context(),
		map[string]any{
			"kind":           "document",
			"title":          title,
			"content":        rawText,
			"status":         "published",
			"knowledge_type": knowledgeType,
		},
		fallbackKnowledgeMetadata("document", "published"),
	)
	metadataJSON, _ := json.Marshal(metadata.MetadataJSON)

	var id string
	err = s.db.QueryRow(r.Context(), `
		INSERT INTO knowledge_documents (
		  title, knowledge_type, file_url, raw_text, status, created_by, updated_by, published_at,
		  source_type, priority, locked, topic, intent, metadata_json, organization_id, ai_agent_id
		)
		VALUES (
		  $1, $2, $3, $4, 'published', $5, $5, NOW(),
		  $6, $7, $8, NULLIF($9, ''), NULLIF($10, ''), $11::jsonb, $12, $13
		)
		RETURNING id
	`, title, knowledgeType, fileURL, rawText, s.agentID(r.Context()), metadata.SourceType, metadata.Priority, metadata.Locked, metadata.Topic, metadata.Intent, string(metadataJSON), s.organizationID(r.Context()), aiAgentID).Scan(&id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertAuditLog(r.Context(), "knowledge.document_upload", "knowledge_document", id, map[string]any{"knowledge_type": knowledgeType, "filename": header.Filename}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"id":      id,
		"fileUrl": fileURL,
	})
}

func (s *Server) handleKnowledgeDocumentDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}

	var title, fileURL string
	err := s.db.QueryRow(r.Context(), `
		SELECT title, COALESCE(file_url, '')
		FROM knowledge_documents
		WHERE id = $1 AND organization_id = $2
	`, id, s.organizationID(r.Context())).Scan(&title, &fileURL)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}
	if fileURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "document has no uploaded file"})
		return
	}

	objectKey := s.objectKeyFromURL(fileURL)

	content, err := s.storage.Download(r.Context(), objectKey)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	filename := strings.ReplaceAll(strings.ToLower(title), " ", "-") + ".txt"
	contentType := "application/octet-stream"
	if ext := strings.ToLower(filepath.Ext(objectKey)); ext != "" {
		switch ext {
		case ".txt":
			contentType = "text/plain; charset=utf-8"
		case ".md":
			contentType = "text/markdown; charset=utf-8"
		case ".csv":
			contentType = "text/csv; charset=utf-8"
		case ".json":
			contentType = "application/json"
		case ".pdf":
			contentType = "application/pdf"
		case ".docx":
			contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		}
		filename = strings.ReplaceAll(strings.ToLower(title), " ", "-") + ext
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) handleJobPositions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		aiAgentID := strings.TrimSpace(r.URL.Query().Get("aiAgentId"))
		if aiAgentID != "" {
			if ok, err := s.aiAgentBelongsToOrganization(r.Context(), aiAgentID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			} else if !ok {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid aiAgentId is required"})
				return
			}
		}
		rows, err := s.db.Query(r.Context(), `
			SELECT id, title, status, COALESCE(location, ''), COALESCE(work_type, ''), COALESCE(short_description, ''), COALESCE(apply_link, ''), is_active, updated_at, COALESCE(ai_agent_id::text, '')
			FROM job_positions
			WHERE organization_id = $1
			  AND ($2 = '' OR ai_agent_id = NULLIF($2, '')::uuid)
			ORDER BY updated_at DESC
		`, s.organizationID(r.Context()), aiAgentID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		items := []map[string]any{}
		for rows.Next() {
			var id, title, status, location, workType, shortDescription, applyLink, itemAIAgentID string
			var isActive bool
			var updatedAt time.Time
			if err := rows.Scan(&id, &title, &status, &location, &workType, &shortDescription, &applyLink, &isActive, &updatedAt, &itemAIAgentID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			items = append(items, map[string]any{
				"id":               id,
				"title":            title,
				"status":           status,
				"location":         location,
				"workType":         workType,
				"shortDescription": shortDescription,
				"applyLink":        applyLink,
				"isActive":         isActive,
				"aiAgentId":        itemAIAgentID,
				"updatedAt":        updatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		role := s.role(r.Context())
		if role == "owner" || role == "operator" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		var req struct {
			Title            string `json:"title"`
			Status           string `json:"status"`
			Location         string `json:"location"`
			WorkType         string `json:"workType"`
			ShortDescription string `json:"shortDescription"`
			ApplyLink        string `json:"applyLink"`
			AIAgentID        string `json:"aiAgentId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if strings.TrimSpace(req.Title) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
			return
		}
		if strings.TrimSpace(req.Status) == "" {
			req.Status = "published"
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

		var id string
		err := s.db.QueryRow(r.Context(), `
			INSERT INTO job_positions (
			  title, status, location, work_type, short_description, apply_link, is_active, created_by, updated_by, organization_id, ai_agent_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, TRUE, $7, $7, $8, $9)
			RETURNING id
		`, req.Title, req.Status, req.Location, req.WorkType, req.ShortDescription, req.ApplyLink, s.agentID(r.Context()), s.organizationID(r.Context()), req.AIAgentID).Scan(&id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.syncJobPositionKnowledgeRecord(r.Context(), id, s.agentID(r.Context()), req.AIAgentID, req.Title, req.Status, req.Location, req.WorkType, req.ShortDescription, req.ApplyLink, true); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLog(r.Context(), "job_position.create", "job_position", id, map[string]any{"status": req.Status, "title": req.Title}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "id": id})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleJobPositionRoutes(w http.ResponseWriter, r *http.Request) {
	role := s.role(r.Context())
	if role == "owner" || role == "operator" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/job-positions/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Title            string `json:"title"`
			Status           string `json:"status"`
			Location         string `json:"location"`
			WorkType         string `json:"workType"`
			ShortDescription string `json:"shortDescription"`
			ApplyLink        string `json:"applyLink"`
			IsActive         *bool  `json:"isActive"`
			AIAgentID        string `json:"aiAgentId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if strings.TrimSpace(req.Title) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
			return
		}
		if strings.TrimSpace(req.Status) == "" {
			req.Status = "published"
		}
		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}
		req.AIAgentID = strings.TrimSpace(req.AIAgentID)
		if req.AIAgentID == "" {
			if err := s.db.QueryRow(r.Context(), `SELECT COALESCE(ai_agent_id::text, '') FROM job_positions WHERE id = $1 AND organization_id = $2`, id, s.organizationID(r.Context())).Scan(&req.AIAgentID); err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "position not found"})
				return
			}
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

		tag, err := s.db.Exec(r.Context(), `
			UPDATE job_positions
			SET title = $2,
			    status = $3,
			    location = $4,
			    work_type = $5,
			    short_description = $6,
			    apply_link = $7,
			    is_active = $8,
			    ai_agent_id = $9,
			    embedding = NULL,
			    updated_by = $10,
			    updated_at = NOW()
			WHERE id = $1 AND organization_id = $11
		`, id, req.Title, req.Status, req.Location, req.WorkType, req.ShortDescription, req.ApplyLink, isActive, req.AIAgentID, s.agentID(r.Context()), s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "position not found"})
			return
		}
		if err := s.syncJobPositionKnowledgeRecord(r.Context(), id, s.agentID(r.Context()), req.AIAgentID, req.Title, req.Status, req.Location, req.WorkType, req.ShortDescription, req.ApplyLink, isActive); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.insertAuditLog(r.Context(), "job_position.update", "job_position", id, map[string]any{"status": req.Status, "is_active": isActive, "title": req.Title}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case http.MethodDelete:
		tag, err := s.db.Exec(r.Context(), `DELETE FROM job_positions WHERE id = $1 AND organization_id = $2`, id, s.organizationID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "position not found"})
			return
		}
		_, _ = s.db.Exec(r.Context(), `DELETE FROM knowledge_records WHERE id = $1 AND organization_id = $2`, id, s.organizationID(r.Context()))
		if err := s.insertAuditLog(r.Context(), "job_position.delete", "job_position", id, map[string]any{}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) objectKeyFromURL(fileURL string) string {
	objectKey := strings.TrimSpace(fileURL)
	if objectKey == "" {
		return ""
	}
	if strings.HasPrefix(objectKey, s.cfg.ObjectStoragePublicBaseURL+"/"+s.cfg.ObjectStorageBucket+"/") {
		objectKey = strings.TrimPrefix(objectKey, s.cfg.ObjectStoragePublicBaseURL+"/"+s.cfg.ObjectStorageBucket+"/")
	}
	if strings.HasPrefix(objectKey, s.cfg.ObjectStorageEndpoint+"/"+s.cfg.ObjectStorageBucket+"/") {
		objectKey = strings.TrimPrefix(objectKey, s.cfg.ObjectStorageEndpoint+"/"+s.cfg.ObjectStorageBucket+"/")
	}
	return objectKey
}
