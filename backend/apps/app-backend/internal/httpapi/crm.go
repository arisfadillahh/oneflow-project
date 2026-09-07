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

func (s *Server) handleContactRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/contacts/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if parts[0] == "import" {
		s.handleContactsImport(w, r)
		return
	}
	contactID := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleContactDetail(w, r, contactID)
		case http.MethodPatch, http.MethodPut:
			s.handleContactUpdate(w, r, contactID)
		default:
			http.NotFound(w, r)
		}
		return
	}
	if len(parts) == 2 {
		switch parts[1] {
		case "memories":
			s.handleContactMemories(w, r, contactID)
			return
		case "notes":
			s.handleContactNotes(w, r, contactID)
			return
		case "tags":
			s.handleContactTagsForContact(w, r, contactID)
			return
		}
	}
	if len(parts) == 3 && parts[1] == "memories" {
		s.handleContactMemoryItem(w, r, contactID, parts[2])
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleContactTags(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.listContactTags(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var req struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createContactTag(r.Context(), req.Name, req.Color)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleContactDetail(w http.ResponseWriter, r *http.Request, contactID string) {
	if !s.crmAccess(w, r) {
		return
	}
	contact, err := s.contactDetail(r.Context(), contactID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "contact not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, contact)
}

func (s *Server) handleContactUpdate(w http.ResponseWriter, r *http.Request, contactID string) {
	if !s.crmAccess(w, r) {
		return
	}
	var req struct {
		Name                  *string        `json:"name"`
		Email                 *string        `json:"email"`
		Notes                 *string        `json:"notes"`
		LifecycleStatus       *string        `json:"lifecycleStatus"`
		OwnerAgentID          *string        `json:"ownerAgentId"`
		CustomFields          map[string]any `json:"customFields"`
		LastSummary           *string        `json:"lastSummary"`
		CustomerMemoryEnabled *bool          `json:"customerMemoryEnabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	var current struct {
		Name                  string
		Email                 string
		Notes                 string
		LifecycleStatus       string
		OwnerAgentID          string
		CustomFields          string
		LastSummary           string
		CustomerMemoryEnabled bool
	}
	err := s.db.QueryRow(r.Context(), `
		SELECT COALESCE(name, ''), COALESCE(email, ''), COALESCE(notes, ''),
		       lifecycle_status, COALESCE(owner_agent_id::text, ''),
		       COALESCE(custom_fields, '{}'::jsonb)::text, COALESCE(last_summary, ''),
		       customer_memory_enabled
		FROM contacts
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, contactID, s.organizationID(r.Context())).Scan(
		&current.Name,
		&current.Email,
		&current.Notes,
		&current.LifecycleStatus,
		&current.OwnerAgentID,
		&current.CustomFields,
		&current.LastSummary,
		&current.CustomerMemoryEnabled,
	)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "contact not found"})
		return
	}

	name := current.Name
	email := current.Email
	notes := current.Notes
	lifecycleStatus := current.LifecycleStatus
	ownerAgentID := current.OwnerAgentID
	lastSummary := current.LastSummary
	customFieldsJSON := current.CustomFields
	customerMemoryEnabled := current.CustomerMemoryEnabled

	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.Email != nil {
		email = strings.TrimSpace(*req.Email)
	}
	if req.Notes != nil {
		notes = strings.TrimSpace(*req.Notes)
	}
	if req.LifecycleStatus != nil {
		lifecycleStatus = strings.TrimSpace(*req.LifecycleStatus)
	}
	if req.OwnerAgentID != nil {
		ownerAgentID = strings.TrimSpace(*req.OwnerAgentID)
	}
	if req.CustomFields != nil {
		customFieldsJSON = marshalJSON(req.CustomFields)
	}
	if req.LastSummary != nil {
		lastSummary = strings.TrimSpace(*req.LastSummary)
	}
	if req.CustomerMemoryEnabled != nil {
		customerMemoryEnabled = *req.CustomerMemoryEnabled
	}
	if !validLifecycleStatus(lifecycleStatus) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lifecycle status"})
		return
	}
	if ownerAgentID != "" && !s.agentBelongsToOrganization(r.Context(), ownerAgentID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "owner must be an active team member"})
		return
	}

	_, err = s.db.Exec(r.Context(), `
		UPDATE contacts
		SET name = NULLIF($2, ''),
		    email = NULLIF($3, ''),
		    notes = NULLIF($4, ''),
		    lifecycle_status = $5,
		    owner_agent_id = NULLIF($6, '')::uuid,
		    custom_fields = $7::jsonb,
		    last_summary = NULLIF($8, ''),
		    summary_updated_at = CASE WHEN $9 THEN NOW() ELSE summary_updated_at END,
		    customer_memory_enabled = $10,
		    customer_memory_disabled_at = CASE WHEN $10 THEN NULL ELSE COALESCE(customer_memory_disabled_at, NOW()) END,
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $11
	`, contactID, name, email, notes, lifecycleStatus, ownerAgentID, customFieldsJSON, lastSummary, req.LastSummary != nil, customerMemoryEnabled, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	contact, err := s.contactDetail(r.Context(), contactID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, contact)
}

func (s *Server) handleContactNotes(w http.ResponseWriter, r *http.Request, contactID string) {
	if !s.crmAccess(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "note is required"})
		return
	}
	if !s.contactExists(r.Context(), contactID) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "contact not found"})
		return
	}
	var id string
	var createdAt time.Time
	err := s.db.QueryRow(r.Context(), `
		INSERT INTO contact_notes (organization_id, contact_id, body, created_by)
		VALUES ($1, NULLIF($2, '')::uuid, $3, NULLIF($4, '')::uuid)
		RETURNING id::text, created_at
	`, s.organizationID(r.Context()), contactID, body, s.agentID(r.Context())).Scan(&id, &createdAt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"item": map[string]any{
		"id":        id,
		"body":      body,
		"authorId":  s.agentID(r.Context()),
		"createdAt": createdAt,
	}})
}

func (s *Server) handleContactMemories(w http.ResponseWriter, r *http.Request, contactID string) {
	if !s.crmAccess(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.memoriesForContact(r.Context(), contactID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var req struct {
			MemoryType string `json:"memoryType"`
			Value      string `json:"value"`
			ExpiresAt  string `json:"expiresAt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		memoryType := normalizeContactMemoryType(req.MemoryType)
		if memoryType == "" {
			memoryType = "admin_note"
		}
		if memoryType != "admin_note" && memoryType != "preference" && memoryType != "interest" && memoryType != "open_issue" && memoryType != "communication_preference" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid memory type"})
			return
		}
		value := strings.TrimSpace(req.Value)
		normalized := normalizeContactMemoryValue(value)
		if normalized == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "memory value is required"})
			return
		}
		var expiresAt *time.Time
		if strings.TrimSpace(req.ExpiresAt) != "" {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ExpiresAt))
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expiresAt must be RFC3339"})
				return
			}
			expiresAt = &parsed
		}
		if !s.contactExists(r.Context(), contactID) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "contact not found"})
			return
		}
		tx, err := s.db.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer tx.Rollback(r.Context())

		var draftID string
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO contact_memory_drafts (
			  organization_id, contact_id, raw_text, extracted_fact, memory_type, risk_category,
			  source_type, gate1_result, final_status, confidence, created_by
			)
			VALUES ($1, NULLIF($2, '')::uuid, $3, $4::jsonb, $5, 'safe', 'admin', 'pass', 'approved', 1, NULLIF($6, '')::uuid)
			RETURNING id::text
		`, s.organizationID(r.Context()), contactID, value, marshalJSON(map[string]any{"value": value, "normalizedValue": normalized}), memoryType, s.agentID(r.Context())).Scan(&draftID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		item, err := s.upsertContactMemoryTx(r.Context(), tx, contactID, memoryType, value, normalized, "admin", draftID, "", "", 1, expiresAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleContactMemoryItem(w http.ResponseWriter, r *http.Request, contactID, memoryID string) {
	if !s.crmAccess(w, r) {
		return
	}
	if r.Method != http.MethodDelete {
		http.NotFound(w, r)
		return
	}
	tag, err := s.db.Exec(r.Context(), `
		UPDATE contact_memories
		SET status = 'deleted', updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid
		  AND contact_id = NULLIF($2, '')::uuid
		  AND organization_id = $3
	`, memoryID, contactID, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "memory not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleContactTagsForContact(w http.ResponseWriter, r *http.Request, contactID string) {
	if !s.crmAccess(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req struct {
		TagIDs []string `json:"tagIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if !s.contactExists(r.Context(), contactID) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "contact not found"})
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	if _, err := tx.Exec(r.Context(), `DELETE FROM contact_tag_links WHERE contact_id = NULLIF($1, '')::uuid AND organization_id = $2`, contactID, s.organizationID(r.Context())); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	seen := map[string]bool{}
	for _, tagID := range req.TagIDs {
		tagID = strings.TrimSpace(tagID)
		if tagID == "" || seen[tagID] {
			continue
		}
		seen[tagID] = true
		var exists bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM contact_tags WHERE id = NULLIF($1, '')::uuid AND organization_id = $2)`, tagID, s.organizationID(r.Context())).Scan(&exists); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if !exists {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tag does not belong to this organization"})
			return
		}
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO contact_tag_links (organization_id, contact_id, tag_id)
			VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid)
			ON CONFLICT (contact_id, tag_id) DO NOTHING
		`, s.organizationID(r.Context()), contactID, tagID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	tags, err := s.contactTagsForContact(r.Context(), contactID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tags})
}

func (s *Server) handleContactsImport(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req struct {
		Contacts []struct {
			Phone        string         `json:"phone"`
			Name         string         `json:"name"`
			Email        string         `json:"email"`
			Notes        string         `json:"notes"`
			Tags         []string       `json:"tags"`
			CustomFields map[string]any `json:"customFields"`
		} `json:"contacts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if len(req.Contacts) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "contacts are required"})
		return
	}
	if len(req.Contacts) > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "maximum 500 contacts per import"})
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	imported := 0
	skipped := 0
	for _, item := range req.Contacts {
		phone, _ := normalizeWhatsAppPhoneValue(item.Phone)
		phone = strings.TrimSpace(phone)
		if phone == "" {
			skipped++
			continue
		}
		customFields := "{}"
		if item.CustomFields != nil {
			customFields = marshalJSON(item.CustomFields)
		}
		var contactID string
		err := tx.QueryRow(r.Context(), `
			INSERT INTO contacts (phone, name, email, notes, custom_fields, organization_id)
			VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), $5::jsonb, $6)
			ON CONFLICT (organization_id, phone) DO UPDATE
			SET name = COALESCE(NULLIF(EXCLUDED.name, ''), contacts.name),
			    email = COALESCE(NULLIF(EXCLUDED.email, ''), contacts.email),
			    notes = COALESCE(NULLIF(EXCLUDED.notes, ''), contacts.notes),
			    custom_fields = CASE WHEN EXCLUDED.custom_fields = '{}'::jsonb THEN contacts.custom_fields ELSE EXCLUDED.custom_fields END,
			    updated_at = NOW()
			RETURNING id::text
		`, phone, strings.TrimSpace(item.Name), strings.TrimSpace(item.Email), strings.TrimSpace(item.Notes), customFields, s.organizationID(r.Context())).Scan(&contactID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		for _, tagName := range item.Tags {
			tagID, err := s.upsertContactTagTx(r.Context(), tx, tagName, "")
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if tagID == "" {
				continue
			}
			if _, err := tx.Exec(r.Context(), `
				INSERT INTO contact_tag_links (organization_id, contact_id, tag_id)
				VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid)
				ON CONFLICT (contact_id, tag_id) DO NOTHING
			`, s.organizationID(r.Context()), contactID, tagID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
		imported++
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"imported": imported, "skipped": skipped})
}

func (s *Server) handleFollowUpTasks(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.listFollowUpTasks(r.Context(), r.URL.Query().Get("contactId"), r.URL.Query().Get("status"))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var req crmTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createFollowUpTask(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleFollowUpTaskRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/follow-up-tasks/"), "/")
	if id == "" || (r.Method != http.MethodPatch && r.Method != http.MethodPut) {
		http.NotFound(w, r)
		return
	}
	var req crmTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	item, err := s.updateFollowUpTask(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (s *Server) handleMessageTemplates(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.listMessageTemplates(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var req crmTemplateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createMessageTemplate(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleMessageTemplateRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/message-templates/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPatch, http.MethodPut:
		var req crmTemplateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.updateMessageTemplate(r.Context(), id, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": item})
	case http.MethodDelete:
		if _, err := s.db.Exec(r.Context(), `
			UPDATE whatsapp_message_templates
			SET status = 'archived', updated_by = NULLIF($3, '')::uuid, updated_at = NOW()
			WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
		`, id, s.organizationID(r.Context()), s.agentID(r.Context())); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "archived"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleBroadcasts(w http.ResponseWriter, r *http.Request) {
	if !s.crmAccess(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.listBroadcasts(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var req struct {
			Title          string   `json:"title"`
			TemplateID     string   `json:"templateId"`
			Body           string   `json:"body"`
			ContactIDs     []string `json:"contactIds"`
			Audience       string   `json:"audience"`
			TagIDs         []string `json:"tagIds"`
			DealStageID    string   `json:"dealStageId"`
			Lifecycle      string   `json:"lifecycleStatus"`
			NoFollowUpDays int      `json:"noFollowUpDays"`
			SendNow        bool     `json:"sendNow"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		contactIDs, err := s.resolveBroadcastContactIDs(r.Context(), req.ContactIDs, req.Audience, req.TagIDs, req.DealStageID, req.Lifecycle, req.NoFollowUpDays)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		broadcastID, err := s.createBroadcast(r.Context(), req.Title, req.TemplateID, req.Body, contactIDs)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		result := map[string]any{"id": broadcastID, "status": "draft"}
		if req.SendNow {
			sendResult, err := s.sendBroadcastNow(r.Context(), broadcastID)
			if err != nil {
				writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "id": broadcastID})
				return
			}
			result = sendResult
		}
		writeJSON(w, http.StatusCreated, result)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleConversationWorkflow(w http.ResponseWriter, r *http.Request, id string) {
	if !s.crmAccess(w, r) {
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPatch && r.Method != http.MethodPut {
		http.NotFound(w, r)
		return
	}
	var req struct {
		Status       *string    `json:"status"`
		Priority     *string    `json:"priority"`
		AssignedToID *string    `json:"assignedToId"`
		SLADueAt     *time.Time `json:"slaDueAt"`
		InternalNote *string    `json:"internalNote"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	var currentStatus, currentPriority, currentAssignedTo, currentInternalNote string
	var currentSLADueAt *time.Time
	err := s.db.QueryRow(r.Context(), `
		SELECT status::text, priority, COALESCE(assigned_to::text, ''), sla_due_at, COALESCE(internal_note, '')
		FROM conversations
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, id, s.organizationID(r.Context())).Scan(&currentStatus, &currentPriority, &currentAssignedTo, &currentSLADueAt, &currentInternalNote)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation not found"})
		return
	}
	status := currentStatus
	priority := currentPriority
	assignedToID := currentAssignedTo
	internalNote := currentInternalNote
	slaDueAt := currentSLADueAt
	if req.Status != nil {
		status = strings.TrimSpace(*req.Status)
	}
	if req.Priority != nil {
		priority = strings.TrimSpace(*req.Priority)
	}
	if req.AssignedToID != nil {
		assignedToID = strings.TrimSpace(*req.AssignedToID)
	}
	if req.InternalNote != nil {
		internalNote = strings.TrimSpace(*req.InternalNote)
	}
	if req.SLADueAt != nil {
		slaDueAt = req.SLADueAt
	}
	if !validConversationStatus(status) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid conversation status"})
		return
	}
	if !validPriority(priority) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid priority"})
		return
	}
	if assignedToID != "" && !s.agentBelongsToOrganization(r.Context(), assignedToID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "assignee must be an active team member"})
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
		SET status = $2::conversation_status,
		    priority = $3,
		    assigned_to = NULLIF($4, '')::uuid,
		    sla_due_at = $5,
		    internal_note = NULLIF($6, ''),
		    workflow_updated_at = NOW(),
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $7
	`, id, status, priority, assignedToID, slaDueAt, internalNote, s.organizationID(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.insertEventTx(r.Context(), tx, id, "workflow_updated", s.agentID(r.Context()), map[string]any{
		"status":         status,
		"priority":       priority,
		"assigned_to_id": assignedToID,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.handleConversationDetail(w, r, id)
}

type crmTaskRequest struct {
	Title          *string    `json:"title"`
	Notes          *string    `json:"notes"`
	ContactID      *string    `json:"contactId"`
	ConversationID *string    `json:"conversationId"`
	DueAt          *time.Time `json:"dueAt"`
	Status         *string    `json:"status"`
	Priority       *string    `json:"priority"`
	AssignedToID   *string    `json:"assignedToId"`
}

type crmTemplateRequest struct {
	Name      *string `json:"name"`
	Category  *string `json:"category"`
	Body      *string `json:"body"`
	Variables any     `json:"variables"`
	Status    *string `json:"status"`
}

func (s *Server) contactDetail(ctx context.Context, contactID string) (map[string]any, error) {
	var customFieldsJSON string
	var summaryUpdatedAt *time.Time
	item := map[string]any{}
	var id, phone, name, email, notes, lifecycleStatus, ownerAgentID, ownerName, lastSummary string
	var customerMemoryEnabled bool
	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(ctx, `
		SELECT ct.id::text, ct.phone, COALESCE(ct.name, ''), COALESCE(ct.email, ''),
		       COALESCE(ct.notes, ''), ct.lifecycle_status, COALESCE(ct.owner_agent_id::text, ''),
		       COALESCE(owner.name, ''), COALESCE(ct.custom_fields, '{}'::jsonb)::text,
		       COALESCE(ct.last_summary, ''), ct.customer_memory_enabled,
		       ct.summary_updated_at, ct.created_at, ct.updated_at
		FROM contacts ct
		LEFT JOIN agents owner ON owner.id = ct.owner_agent_id
		WHERE ct.id = NULLIF($1, '')::uuid AND ct.organization_id = $2
	`, contactID, s.organizationID(ctx)).Scan(&id, &phone, &name, &email, &notes, &lifecycleStatus, &ownerAgentID, &ownerName, &customFieldsJSON, &lastSummary, &customerMemoryEnabled, &summaryUpdatedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	tags, err := s.contactTagsForContact(ctx, contactID)
	if err != nil {
		return nil, err
	}
	notesList, err := s.notesForContact(ctx, contactID)
	if err != nil {
		return nil, err
	}
	tasks, err := s.listFollowUpTasks(ctx, contactID, "")
	if err != nil {
		return nil, err
	}
	conversations, err := s.conversationsForContact(ctx, contactID)
	if err != nil {
		return nil, err
	}
	messages, err := s.messagesForContact(ctx, contactID)
	if err != nil {
		return nil, err
	}
	memories, err := s.memoriesForContact(ctx, contactID)
	if err != nil {
		return nil, err
	}
	summary := lastSummary
	if strings.TrimSpace(summary) == "" {
		summary = buildContactSummary(name, phone, messages)
	}
	item["contact"] = map[string]any{
		"id":                    id,
		"phone":                 phone,
		"name":                  name,
		"email":                 email,
		"notes":                 notes,
		"lifecycleStatus":       lifecycleStatus,
		"ownerAgentId":          ownerAgentID,
		"ownerName":             ownerName,
		"customFields":          parseJSONObject(customFieldsJSON),
		"lastSummary":           lastSummary,
		"customerMemoryEnabled": customerMemoryEnabled,
		"summaryUpdatedAt":      summaryUpdatedAt,
		"createdAt":             createdAt,
		"updatedAt":             updatedAt,
	}
	item["tags"] = tags
	item["notes"] = notesList
	item["tasks"] = tasks
	item["conversations"] = conversations
	item["messages"] = messages
	item["memories"] = memories
	item["aiSummary"] = summary
	return item, nil
}

func (s *Server) contactTagsForContact(ctx context.Context, contactID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT t.id::text, t.name, t.color
		FROM contact_tag_links l
		JOIN contact_tags t ON t.id = l.tag_id AND t.organization_id = l.organization_id
		WHERE l.contact_id = NULLIF($1, '')::uuid AND l.organization_id = $2
		ORDER BY t.name ASC
	`, contactID, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name, color string
		if err := rows.Scan(&id, &name, &color); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"id": id, "name": name, "color": color})
	}
	return items, rows.Err()
}

func (s *Server) notesForContact(ctx context.Context, contactID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT n.id::text, n.body, COALESCE(a.name, ''), COALESCE(n.created_by::text, ''), n.created_at
		FROM contact_notes n
		LEFT JOIN agents a ON a.id = n.created_by
		WHERE n.contact_id = NULLIF($1, '')::uuid AND n.organization_id = $2
		ORDER BY n.created_at DESC
		LIMIT 50
	`, contactID, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, body, authorName, authorID string
		var createdAt time.Time
		if err := rows.Scan(&id, &body, &authorName, &authorID, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"id": id, "body": body, "authorName": authorName, "authorId": authorID, "createdAt": createdAt})
	}
	return items, rows.Err()
}

func normalizeContactMemoryType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	switch normalized {
	case "", "admin_note", "note", "adminnote":
		return "admin_note"
	case "preference", "preferences":
		return "preference"
	case "interest", "interests":
		return "interest"
	case "open_issue", "issue", "openissue":
		return "open_issue"
	case "communication_preference", "communication", "communicationpreference":
		return "communication_preference"
	case "context":
		return "context"
	default:
		return ""
	}
}

func normalizeContactMemoryValue(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	normalized = strings.Join(strings.Fields(normalized), " ")
	normalized = strings.Trim(normalized, " \t\r\n.,;:!?\"'`")
	if len(normalized) > 240 {
		normalized = strings.TrimSpace(normalized[:240])
	}
	return normalized
}

func (s *Server) memoriesForContact(ctx context.Context, contactID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, memory_type, value, normalized_value, source_type,
		       COALESCE(source_draft_id::text, ''), COALESCE(source_conversation_id::text, ''),
		       COALESCE(source_message_id::text, ''), confidence, created_at, updated_at,
		       last_seen_at, expires_at
		FROM contact_memories
		WHERE contact_id = NULLIF($1, '')::uuid
		  AND organization_id = $2
		  AND status = 'active'
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY (memory_type = 'admin_note') DESC, confidence DESC, updated_at DESC
		LIMIT 50
	`, contactID, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id, memoryType, value, normalizedValue, sourceType, draftID, conversationID, messageID string
		var confidence float64
		var createdAt, updatedAt time.Time
		var lastSeenAt, expiresAt *time.Time
		if err := rows.Scan(&id, &memoryType, &value, &normalizedValue, &sourceType, &draftID, &conversationID, &messageID, &confidence, &createdAt, &updatedAt, &lastSeenAt, &expiresAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":                   id,
			"memoryType":           memoryType,
			"value":                value,
			"normalizedValue":      normalizedValue,
			"sourceType":           sourceType,
			"sourceDraftId":        draftID,
			"sourceConversationId": conversationID,
			"sourceMessageId":      messageID,
			"confidence":           confidence,
			"createdAt":            createdAt,
			"updatedAt":            updatedAt,
			"lastSeenAt":           lastSeenAt,
			"expiresAt":            expiresAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) upsertContactMemoryTx(ctx context.Context, tx pgx.Tx, contactID, memoryType, value, normalizedValue, sourceType, draftID, conversationID, messageID string, confidence float64, expiresAt *time.Time) (map[string]any, error) {
	memoryType = normalizeContactMemoryType(memoryType)
	value = strings.TrimSpace(value)
	normalizedValue = normalizeContactMemoryValue(normalizedValue)
	if normalizedValue == "" {
		normalizedValue = normalizeContactMemoryValue(value)
	}
	sourceType = strings.TrimSpace(sourceType)
	if sourceType == "" {
		sourceType = "ai_extracted"
	}
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}
	if memoryType == "" || value == "" || normalizedValue == "" {
		return nil, errors.New("memory type and value are required")
	}

	var item struct {
		ID             string
		MemoryType     string
		Value          string
		Normalized     string
		SourceType     string
		DraftID        string
		ConversationID string
		MessageID      string
		Confidence     float64
		CreatedAt      time.Time
		UpdatedAt      time.Time
		LastSeenAt     *time.Time
		ExpiresAt      *time.Time
	}
	err := tx.QueryRow(ctx, `
		INSERT INTO contact_memories (
		  organization_id, contact_id, memory_type, value, normalized_value, source_type,
		  source_draft_id, source_conversation_id, source_message_id, confidence,
		  status, created_by, last_seen_at, expires_at
		)
		VALUES (
		  $1, NULLIF($2, '')::uuid, $3, $4, $5, $6,
		  NULLIF($7, '')::uuid, NULLIF($8, '')::uuid, NULLIF($9, '')::uuid, $10,
		  'active', NULLIF($11, '')::uuid, NOW(), $12
		)
		ON CONFLICT (organization_id, contact_id, memory_type, normalized_value) WHERE status = 'active'
		DO UPDATE SET
		  value = EXCLUDED.value,
		  source_type = EXCLUDED.source_type,
		  source_draft_id = COALESCE(EXCLUDED.source_draft_id, contact_memories.source_draft_id),
		  source_conversation_id = COALESCE(EXCLUDED.source_conversation_id, contact_memories.source_conversation_id),
		  source_message_id = COALESCE(EXCLUDED.source_message_id, contact_memories.source_message_id),
		  confidence = GREATEST(contact_memories.confidence, EXCLUDED.confidence),
		  updated_at = NOW(),
		  last_seen_at = NOW(),
		  expires_at = COALESCE(EXCLUDED.expires_at, contact_memories.expires_at)
		RETURNING id::text, memory_type, value, normalized_value, source_type,
		          COALESCE(source_draft_id::text, ''), COALESCE(source_conversation_id::text, ''),
		          COALESCE(source_message_id::text, ''), confidence, created_at, updated_at,
		          last_seen_at, expires_at
	`, s.organizationID(ctx), contactID, memoryType, value, normalizedValue, sourceType, draftID, conversationID, messageID, confidence, s.agentID(ctx), expiresAt).Scan(
		&item.ID,
		&item.MemoryType,
		&item.Value,
		&item.Normalized,
		&item.SourceType,
		&item.DraftID,
		&item.ConversationID,
		&item.MessageID,
		&item.Confidence,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.LastSeenAt,
		&item.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":                   item.ID,
		"memoryType":           item.MemoryType,
		"value":                item.Value,
		"normalizedValue":      item.Normalized,
		"sourceType":           item.SourceType,
		"sourceDraftId":        item.DraftID,
		"sourceConversationId": item.ConversationID,
		"sourceMessageId":      item.MessageID,
		"confidence":           item.Confidence,
		"createdAt":            item.CreatedAt,
		"updatedAt":            item.UpdatedAt,
		"lastSeenAt":           item.LastSeenAt,
		"expiresAt":            item.ExpiresAt,
	}, nil
}

func (s *Server) conversationsForContact(ctx context.Context, contactID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT c.id::text, c.mode::text, c.status::text, c.priority,
		       c.sla_due_at, COALESCE(c.internal_note, ''), COALESCE(a.name, ''),
		       c.last_message_at, COALESCE(m.text, ''), COALESCE(ws.label, '')
		FROM conversations c
		LEFT JOIN agents a ON a.id = c.assigned_to
		LEFT JOIN messages m ON m.id = c.last_message_id AND m.organization_id = c.organization_id
		LEFT JOIN whatsapp_sessions ws ON ws.id = c.whatsapp_session_id AND ws.organization_id = c.organization_id
		WHERE c.contact_id = NULLIF($1, '')::uuid AND c.organization_id = $2 AND c.channel = 'whatsapp'
		ORDER BY c.last_message_at DESC
	`, contactID, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, mode, status, priority, internalNote, assignedToName, lastMessageText, sessionName string
		var slaDueAt *time.Time
		var lastMessageAt time.Time
		if err := rows.Scan(&id, &mode, &status, &priority, &slaDueAt, &internalNote, &assignedToName, &lastMessageAt, &lastMessageText, &sessionName); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":              id,
			"mode":            mode,
			"status":          status,
			"priority":        priority,
			"slaDueAt":        slaDueAt,
			"internalNote":    internalNote,
			"assignedToName":  assignedToName,
			"lastMessageAt":   lastMessageAt,
			"lastMessageText": lastMessageText,
			"whatsappSession": sessionName,
		})
	}
	return items, rows.Err()
}

func (s *Server) messagesForContact(ctx context.Context, contactID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT c.id::text, m.sender_type::text, m.direction::text, m.content_type::text,
		       COALESCE(m.text, ''), COALESCE(m.sent_at, m.created_at)
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id AND c.organization_id = m.organization_id
		WHERE c.contact_id = NULLIF($1, '')::uuid AND m.organization_id = $2
		ORDER BY COALESCE(m.sent_at, m.created_at) DESC, m.created_at DESC, m.id DESC
		LIMIT 80
	`, contactID, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var conversationID, senderType, direction, contentType, text string
		var createdAt time.Time
		if err := rows.Scan(&conversationID, &senderType, &direction, &contentType, &text, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"conversationId": conversationID,
			"senderType":     senderType,
			"direction":      direction,
			"contentType":    contentType,
			"text":           text,
			"createdAt":      createdAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) listContactTags(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT t.id::text, t.name, t.color, COUNT(l.contact_id) AS contact_count
		FROM contact_tags t
		LEFT JOIN contact_tag_links l ON l.tag_id = t.id AND l.organization_id = t.organization_id
		WHERE t.organization_id = $1
		GROUP BY t.id
		ORDER BY t.name ASC
	`, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name, color string
		var contactCount int
		if err := rows.Scan(&id, &name, &color, &contactCount); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"id": id, "name": name, "color": color, "contactCount": contactCount})
	}
	return items, rows.Err()
}

func (s *Server) createContactTag(ctx context.Context, name, color string) (map[string]any, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	id, err := s.upsertContactTagTx(ctx, tx, name, color)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `SELECT id::text, name, color, 0 FROM contact_tags WHERE id = NULLIF($1, '')::uuid AND organization_id = $2`, id, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var tagID, tagName, tagColor string
		var count int
		if err := rows.Scan(&tagID, &tagName, &tagColor, &count); err != nil {
			return nil, err
		}
		return map[string]any{"id": tagID, "name": tagName, "color": tagColor, "contactCount": count}, nil
	}
	return nil, pgx.ErrNoRows
}

func (s *Server) upsertContactTagTx(ctx context.Context, tx pgx.Tx, name, color string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	if strings.TrimSpace(color) == "" {
		color = "#2563eb"
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO contact_tags (organization_id, name, color)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
		RETURNING id::text
	`, s.organizationID(ctx), name, color).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT id::text FROM contact_tags WHERE organization_id = $1 AND lower(name) = lower($2)`, s.organizationID(ctx), name).Scan(&id)
	}
	return id, err
}

func (s *Server) listFollowUpTasks(ctx context.Context, contactID, status string) ([]map[string]any, error) {
	status = strings.TrimSpace(status)
	rows, err := s.db.Query(ctx, `
		SELECT t.id::text, COALESCE(t.contact_id::text, ''), COALESCE(t.conversation_id::text, ''),
		       t.title, COALESCE(t.notes, ''), t.due_at, t.status, t.priority,
		       COALESCE(t.assigned_to::text, ''), COALESCE(a.name, ''),
		       COALESCE(ct.name, ''), COALESCE(ct.phone, ''), t.created_at, t.completed_at
		FROM follow_up_tasks t
		LEFT JOIN agents a ON a.id = t.assigned_to
		LEFT JOIN contacts ct ON ct.id = t.contact_id AND ct.organization_id = t.organization_id
		WHERE t.organization_id = $1
		  AND ($2 = '' OR t.contact_id = NULLIF($2, '')::uuid)
		  AND ($3 = '' OR t.status = $3)
		ORDER BY
		  CASE WHEN t.status = 'open' THEN 0 ELSE 1 END,
		  t.due_at NULLS LAST,
		  t.created_at DESC
		LIMIT 200
	`, s.organizationID(ctx), strings.TrimSpace(contactID), status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, taskContactID, conversationID, title, notes, taskStatus, priority, assignedToID, assignedToName, contactName, contactPhone string
		var dueAt, completedAt *time.Time
		var createdAt time.Time
		if err := rows.Scan(&id, &taskContactID, &conversationID, &title, &notes, &dueAt, &taskStatus, &priority, &assignedToID, &assignedToName, &contactName, &contactPhone, &createdAt, &completedAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":             id,
			"contactId":      taskContactID,
			"conversationId": conversationID,
			"title":          title,
			"notes":          notes,
			"dueAt":          dueAt,
			"status":         taskStatus,
			"priority":       priority,
			"assignedToId":   assignedToID,
			"assignedToName": assignedToName,
			"contactName":    contactName,
			"contactPhone":   contactPhone,
			"createdAt":      createdAt,
			"completedAt":    completedAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) createFollowUpTask(ctx context.Context, req crmTaskRequest) (map[string]any, error) {
	title := valueOrEmpty(req.Title)
	if title == "" {
		return nil, errors.New("title is required")
	}
	contactID := valueOrEmpty(req.ContactID)
	conversationID := valueOrEmpty(req.ConversationID)
	if contactID == "" && conversationID == "" {
		return nil, errors.New("contact or conversation is required")
	}
	status := valueOrDefault(req.Status, "open")
	priority := valueOrDefault(req.Priority, "normal")
	assignedToID := valueOrEmpty(req.AssignedToID)
	notes := valueOrEmpty(req.Notes)
	if !validTaskStatus(status) || !validPriority(priority) {
		return nil, errors.New("invalid status or priority")
	}
	if assignedToID != "" && !s.agentBelongsToOrganization(ctx, assignedToID) {
		return nil, errors.New("assignee must be an active team member")
	}
	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO follow_up_tasks (organization_id, contact_id, conversation_id, title, notes, due_at, status, priority, assigned_to, created_by)
		VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, NULLIF($5, ''), $6, $7, $8, NULLIF($9, '')::uuid, NULLIF($10, '')::uuid)
		RETURNING id::text
	`, s.organizationID(ctx), contactID, conversationID, title, notes, req.DueAt, status, priority, assignedToID, s.agentID(ctx)).Scan(&id)
	if err != nil {
		return nil, err
	}
	items, err := s.listFollowUpTasks(ctx, contactID, "")
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item["id"] == id {
			return item, nil
		}
	}
	return map[string]any{"id": id, "title": title}, nil
}

func (s *Server) updateFollowUpTask(ctx context.Context, id string, req crmTaskRequest) (map[string]any, error) {
	var current struct {
		ContactID      string
		ConversationID string
		Title          string
		Notes          string
		Status         string
		Priority       string
		AssignedToID   string
		DueAt          *time.Time
	}
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(contact_id::text, ''), COALESCE(conversation_id::text, ''), title,
		       COALESCE(notes, ''), status, priority, COALESCE(assigned_to::text, ''), due_at
		FROM follow_up_tasks
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, id, s.organizationID(ctx)).Scan(&current.ContactID, &current.ConversationID, &current.Title, &current.Notes, &current.Status, &current.Priority, &current.AssignedToID, &current.DueAt)
	if err != nil {
		return nil, err
	}
	title := current.Title
	notes := current.Notes
	status := current.Status
	priority := current.Priority
	assignedToID := current.AssignedToID
	dueAt := current.DueAt
	if req.Title != nil {
		title = valueOrEmpty(req.Title)
	}
	if req.Notes != nil {
		notes = valueOrEmpty(req.Notes)
	}
	if req.Status != nil {
		status = valueOrEmpty(req.Status)
	}
	if req.Priority != nil {
		priority = valueOrEmpty(req.Priority)
	}
	if req.AssignedToID != nil {
		assignedToID = valueOrEmpty(req.AssignedToID)
	}
	if req.DueAt != nil {
		dueAt = req.DueAt
	}
	if title == "" || !validTaskStatus(status) || !validPriority(priority) {
		return nil, errors.New("invalid task")
	}
	if assignedToID != "" && !s.agentBelongsToOrganization(ctx, assignedToID) {
		return nil, errors.New("assignee must be an active team member")
	}
	_, err = s.db.Exec(ctx, `
		UPDATE follow_up_tasks
		SET title = $2,
		    notes = NULLIF($3, ''),
		    due_at = $4,
		    status = $5,
		    priority = $6,
		    assigned_to = NULLIF($7, '')::uuid,
		    completed_at = CASE WHEN $5 = 'done' THEN COALESCE(completed_at, NOW()) ELSE NULL END,
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $8
	`, id, title, notes, dueAt, status, priority, assignedToID, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	items, err := s.listFollowUpTasks(ctx, current.ContactID, "")
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item["id"] == id {
			return item, nil
		}
	}
	return map[string]any{"id": id, "title": title}, nil
}

func (s *Server) listMessageTemplates(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, category, body, variables::text, status, created_at, updated_at
		FROM whatsapp_message_templates
		WHERE organization_id = $1 AND status <> 'archived'
		ORDER BY updated_at DESC
	`, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name, category, body, variablesJSON, status string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &category, &body, &variablesJSON, &status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"id": id, "name": name, "category": category, "body": body, "variables": parseJSONValue(variablesJSON), "status": status, "createdAt": createdAt, "updatedAt": updatedAt})
	}
	return items, rows.Err()
}

func (s *Server) createMessageTemplate(ctx context.Context, req crmTemplateRequest) (map[string]any, error) {
	name := valueOrEmpty(req.Name)
	body := valueOrEmpty(req.Body)
	category := valueOrDefault(req.Category, "follow_up")
	status := valueOrDefault(req.Status, "draft")
	if name == "" || body == "" {
		return nil, errors.New("template name and body are required")
	}
	if !validTemplateStatus(status) {
		return nil, errors.New("invalid template status")
	}
	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO whatsapp_message_templates (organization_id, name, category, body, variables, status, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, NULLIF($7, '')::uuid, NULLIF($7, '')::uuid)
		RETURNING id::text
	`, s.organizationID(ctx), name, category, body, marshalJSON(defaultVariables(req.Variables)), status, s.agentID(ctx)).Scan(&id)
	if err != nil {
		return nil, err
	}
	items, err := s.listMessageTemplates(ctx)
	if err != nil {
		return nil, err
	}
	return firstByID(items, id)
}

func (s *Server) updateMessageTemplate(ctx context.Context, id string, req crmTemplateRequest) (map[string]any, error) {
	name := valueOrEmpty(req.Name)
	body := valueOrEmpty(req.Body)
	category := valueOrDefault(req.Category, "follow_up")
	status := valueOrDefault(req.Status, "draft")
	if name == "" || body == "" || !validTemplateStatus(status) {
		return nil, errors.New("invalid template")
	}
	_, err := s.db.Exec(ctx, `
		UPDATE whatsapp_message_templates
		SET name = $2, category = $3, body = $4, variables = $5::jsonb, status = $6,
		    updated_by = NULLIF($7, '')::uuid, updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $8
	`, id, name, category, body, marshalJSON(defaultVariables(req.Variables)), status, s.agentID(ctx), s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	items, err := s.listMessageTemplates(ctx)
	if err != nil {
		return nil, err
	}
	return firstByID(items, id)
}

func (s *Server) listBroadcasts(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, COALESCE(template_id::text, ''), title, body, status,
		       recipient_count, sent_count, failed_count, sent_at, created_at, updated_at
		FROM whatsapp_broadcasts
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, templateID, title, body, status string
		var recipientCount, sentCount, failedCount int
		var sentAt *time.Time
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &templateID, &title, &body, &status, &recipientCount, &sentCount, &failedCount, &sentAt, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":             id,
			"templateId":     templateID,
			"title":          title,
			"body":           body,
			"status":         status,
			"recipientCount": recipientCount,
			"sentCount":      sentCount,
			"failedCount":    failedCount,
			"sentAt":         sentAt,
			"createdAt":      createdAt,
			"updatedAt":      updatedAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) resolveBroadcastContactIDs(ctx context.Context, explicit []string, audience string, tagIDs []string, dealStageID string, lifecycleStatus string, noFollowUpDays int) ([]string, error) {
	audience = strings.TrimSpace(audience)
	switch audience {
	case "", "selected", "filtered":
		return explicit, nil
	case "tag":
		cleanTagIDs := make([]string, 0, len(tagIDs))
		seen := map[string]bool{}
		for _, tagID := range tagIDs {
			tagID = strings.TrimSpace(tagID)
			if tagID == "" || seen[tagID] {
				continue
			}
			seen[tagID] = true
			cleanTagIDs = append(cleanTagIDs, tagID)
		}
		if len(cleanTagIDs) == 0 {
			return nil, errors.New("select at least one tag")
		}
		rows, err := s.db.Query(ctx, `
			SELECT DISTINCT ct.id::text
			FROM contacts ct
			JOIN contact_tag_links l ON l.contact_id = ct.id AND l.organization_id = ct.organization_id
			WHERE ct.organization_id = $1
			  AND ct.phone <> '__playground__'
			  AND l.tag_id::text = ANY($2)
			LIMIT 100
		`, s.organizationID(ctx), cleanTagIDs)
		if err != nil {
			return nil, err
		}
		return scanBroadcastContactIDs(rows)
	case "lifecycle":
		lifecycleStatus = strings.TrimSpace(lifecycleStatus)
		if !validLifecycleStatus(lifecycleStatus) {
			return nil, errors.New("invalid lifecycle status")
		}
		rows, err := s.db.Query(ctx, `
			SELECT ct.id::text
			FROM contacts ct
			WHERE ct.organization_id = $1
			  AND ct.phone <> '__playground__'
			  AND ct.lifecycle_status = $2
			ORDER BY ct.updated_at DESC
			LIMIT 100
		`, s.organizationID(ctx), lifecycleStatus)
		if err != nil {
			return nil, err
		}
		return scanBroadcastContactIDs(rows)
	case "deal_stage":
		dealStageID = strings.TrimSpace(dealStageID)
		if dealStageID == "" {
			return nil, errors.New("select a deal stage")
		}
		var exists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM deal_stages WHERE id = NULLIF($1, '')::uuid AND organization_id = $2)`, dealStageID, s.organizationID(ctx)).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("deal stage not found")
		}
		rows, err := s.db.Query(ctx, `
			SELECT DISTINCT ct.id::text
			FROM contacts ct
			JOIN deals d ON d.contact_id = ct.id AND d.organization_id = ct.organization_id
			WHERE ct.organization_id = $1
			  AND ct.phone <> '__playground__'
			  AND d.stage_id = NULLIF($2, '')::uuid
			  AND d.status <> 'archived'
			LIMIT 100
		`, s.organizationID(ctx), dealStageID)
		if err != nil {
			return nil, err
		}
		return scanBroadcastContactIDs(rows)
	case "stale_follow_up":
		if noFollowUpDays <= 0 {
			noFollowUpDays = 7
		}
		if noFollowUpDays > 90 {
			noFollowUpDays = 90
		}
		cutoff := time.Now().AddDate(0, 0, -noFollowUpDays)
		rows, err := s.db.Query(ctx, `
			SELECT ct.id::text
			FROM contacts ct
			WHERE ct.organization_id = $1
			  AND ct.phone <> '__playground__'
			  AND NOT EXISTS (
			    SELECT 1
			    FROM follow_up_tasks t
			    WHERE t.contact_id = ct.id
			      AND t.organization_id = ct.organization_id
			      AND GREATEST(t.created_at, t.updated_at, COALESCE(t.completed_at, t.created_at)) >= $2
			  )
			  AND NOT EXISTS (
			    SELECT 1
			    FROM deals d
			    JOIN deal_activities a ON a.deal_id = d.id AND a.organization_id = d.organization_id
			    WHERE d.contact_id = ct.id
			      AND d.organization_id = ct.organization_id
			      AND a.activity_type IN ('note', 'call', 'meeting', 'task')
			      AND a.created_at >= $2
			  )
			ORDER BY ct.updated_at DESC
			LIMIT 100
		`, s.organizationID(ctx), cutoff)
		if err != nil {
			return nil, err
		}
		return scanBroadcastContactIDs(rows)
	default:
		return nil, errors.New("invalid broadcast audience")
	}
}

func scanBroadcastContactIDs(rows pgx.Rows) ([]string, error) {
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	return items, rows.Err()
}

func (s *Server) createBroadcast(ctx context.Context, title, templateID, body string, contactIDs []string) (string, error) {
	title = strings.TrimSpace(title)
	templateID = strings.TrimSpace(templateID)
	body = strings.TrimSpace(body)
	if len(contactIDs) == 0 {
		return "", errors.New("select at least one contact")
	}
	if len(contactIDs) > 100 {
		return "", errors.New("maximum 100 contacts per broadcast")
	}
	if templateID != "" && body == "" {
		var templateName string
		if err := s.db.QueryRow(ctx, `
			SELECT name, body
			FROM whatsapp_message_templates
			WHERE id = NULLIF($1, '')::uuid AND organization_id = $2 AND status <> 'archived'
		`, templateID, s.organizationID(ctx)).Scan(&templateName, &body); err != nil {
			return "", errors.New("template not found")
		}
		if title == "" {
			title = templateName
		}
	}
	if title == "" {
		title = "Broadcast WhatsApp"
	}
	if body == "" {
		return "", errors.New("message body is required")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var broadcastID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO whatsapp_broadcasts (organization_id, template_id, title, body, recipient_count, created_by)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, 0, NULLIF($5, '')::uuid)
		RETURNING id::text
	`, s.organizationID(ctx), templateID, title, body, s.agentID(ctx)).Scan(&broadcastID); err != nil {
		return "", err
	}
	recipientCount := 0
	seen := map[string]bool{}
	for _, contactID := range contactIDs {
		contactID = strings.TrimSpace(contactID)
		if contactID == "" || seen[contactID] {
			continue
		}
		seen[contactID] = true
		var phone string
		if err := tx.QueryRow(ctx, `
			SELECT phone FROM contacts WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
		`, contactID, s.organizationID(ctx)).Scan(&phone); err != nil {
			return "", errors.New("one or more contacts are invalid")
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO whatsapp_broadcast_recipients (organization_id, broadcast_id, contact_id, phone)
			VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4)
			ON CONFLICT (broadcast_id, contact_id) DO NOTHING
		`, s.organizationID(ctx), broadcastID, contactID, phone); err != nil {
			return "", err
		}
		recipientCount++
	}
	if recipientCount == 0 {
		return "", errors.New("select at least one valid contact")
	}
	if _, err := tx.Exec(ctx, `UPDATE whatsapp_broadcasts SET recipient_count = $2, updated_at = NOW() WHERE id = NULLIF($1, '')::uuid AND organization_id = $3`, broadcastID, recipientCount, s.organizationID(ctx)); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return broadcastID, nil
}

func (s *Server) sendBroadcastNow(ctx context.Context, broadcastID string) (map[string]any, error) {
	sessionID, err := s.connectedBroadcastSession(ctx)
	if err != nil {
		return nil, err
	}
	sessionCtx := context.WithValue(ctx, contextWhatsAppSession, sessionID)
	ok, err := s.ensureWhatsAppConnected(sessionCtx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("whatsapp is disconnected")
	}
	var body string
	if err := s.db.QueryRow(ctx, `SELECT body FROM whatsapp_broadcasts WHERE id = NULLIF($1, '')::uuid AND organization_id = $2`, broadcastID, s.organizationID(ctx)).Scan(&body); err != nil {
		return nil, errors.New("broadcast not found")
	}
	rows, err := s.db.Query(ctx, `
		SELECT id::text, contact_id::text
		FROM whatsapp_broadcast_recipients
		WHERE broadcast_id = NULLIF($1, '')::uuid AND organization_id = $2 AND status = 'pending'
		ORDER BY created_at ASC
	`, broadcastID, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	recipients := []struct {
		ID        string
		ContactID string
	}{}
	for rows.Next() {
		var item struct {
			ID        string
			ContactID string
		}
		if err := rows.Scan(&item.ID, &item.ContactID); err != nil {
			rows.Close()
			return nil, err
		}
		recipients = append(recipients, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(recipients) == 0 {
		return nil, errors.New("no pending recipients")
	}
	_, _ = s.db.Exec(ctx, `UPDATE whatsapp_broadcasts SET status = 'sending', updated_at = NOW() WHERE id = NULLIF($1, '')::uuid AND organization_id = $2`, broadcastID, s.organizationID(ctx))
	sentCount := 0
	failedCount := 0
	for _, recipient := range recipients {
		if _, err := s.sendBroadcastRecipient(sessionCtx, broadcastID, recipient.ID, recipient.ContactID, body); err != nil {
			failedCount++
			_, _ = s.db.Exec(ctx, `
				UPDATE whatsapp_broadcast_recipients
				SET status = 'failed', error = $3, updated_at = NOW()
				WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
			`, recipient.ID, s.organizationID(ctx), err.Error())
			continue
		}
		sentCount++
	}
	finalStatus := "sent"
	if failedCount > 0 && sentCount > 0 {
		finalStatus = "partial"
	} else if failedCount > 0 {
		finalStatus = "failed"
	}
	_, err = s.db.Exec(ctx, `
		UPDATE whatsapp_broadcasts
		SET status = $2, sent_count = sent_count + $3, failed_count = failed_count + $4,
		    sent_at = CASE WHEN $3 > 0 THEN NOW() ELSE sent_at END,
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $5
	`, broadcastID, finalStatus, sentCount, failedCount, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": broadcastID, "status": finalStatus, "sentCount": sentCount, "failedCount": failedCount}, nil
}

func (s *Server) sendBroadcastRecipient(ctx context.Context, broadcastID, recipientID, contactID, body string) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var phone, conversationID string
	err = tx.QueryRow(ctx, `SELECT phone FROM contacts WHERE id = NULLIF($1, '')::uuid AND organization_id = $2`, contactID, s.organizationID(ctx)).Scan(&phone)
	if err != nil {
		return "", err
	}
	sessionID := whatsAppSessionIDFromContext(ctx)
	err = tx.QueryRow(ctx, `
		SELECT id::text
		FROM conversations
		WHERE contact_id = NULLIF($1, '')::uuid
		  AND organization_id = $2
		  AND channel = 'whatsapp'
		  AND (whatsapp_session_id = NULLIF($3, '')::uuid OR whatsapp_session_id IS NULL)
		ORDER BY CASE WHEN whatsapp_session_id = NULLIF($3, '')::uuid THEN 0 ELSE 1 END
		LIMIT 1
		FOR UPDATE
	`, contactID, s.organizationID(ctx), sessionID).Scan(&conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO conversations (contact_id, channel, mode, status, last_message_at, organization_id, whatsapp_session_id)
			VALUES (NULLIF($1, '')::uuid, 'whatsapp', 'human', 'open', NOW(), $2, NULLIF($3, '')::uuid)
			RETURNING id::text
		`, contactID, s.organizationID(ctx), sessionID).Scan(&conversationID)
	}
	if err != nil {
		return "", err
	}
	sendResult, err := s.sendWhatsAppText(ctx, phone, body)
	if err != nil {
		return "", err
	}
	externalID := sendResult.ExternalMessageID
	if externalID == "" {
		externalID = fmt.Sprintf("broadcast-%s-%d", broadcastID, time.Now().UnixNano())
	}
	var messageID string
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, external_message_id, sender_type, direction, content_type, text, sender_agent_id, raw_payload, sent_at, organization_id)
		VALUES (NULLIF($1, '')::uuid, $2, 'agent', 'outbound', 'text', $3, NULLIF($4, '')::uuid, $5::jsonb, NOW(), $6)
		RETURNING id::text
	`, conversationID, externalID, body, s.agentID(ctx), marshalJSON(map[string]any{"source": "broadcast", "broadcastId": broadcastID}), s.organizationID(ctx)).Scan(&messageID)
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversations
		SET last_message_id = NULLIF($2, '')::uuid,
		    last_message_at = NOW(),
		    mode = 'human',
		    status = 'open',
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $3
	`, conversationID, messageID, s.organizationID(ctx)); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE whatsapp_broadcast_recipients
		SET status = 'sent', message_id = NULLIF($3, '')::uuid, sent_at = NOW(), updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, recipientID, s.organizationID(ctx), messageID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return messageID, nil
}

func (s *Server) connectedBroadcastSession(ctx context.Context) (string, error) {
	var sessionID string
	err := s.db.QueryRow(ctx, `
		SELECT id::text
		FROM whatsapp_sessions
		WHERE organization_id = $1 AND status = 'connected' AND deleted_at IS NULL
		ORDER BY is_default DESC, updated_at DESC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&sessionID)
	if err != nil {
		return "", errors.New("no connected WhatsApp session")
	}
	return sessionID, nil
}

func (s *Server) crmAccess(w http.ResponseWriter, r *http.Request) bool {
	if !isOpsRole(s.role(r.Context())) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only operational roles can access CRM"})
		return false
	}
	return true
}

func (s *Server) contactExists(ctx context.Context, contactID string) bool {
	var exists bool
	_ = s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM contacts WHERE id = NULLIF($1, '')::uuid AND organization_id = $2)`, contactID, s.organizationID(ctx)).Scan(&exists)
	return exists
}

func (s *Server) agentBelongsToOrganization(ctx context.Context, agentID string) bool {
	var exists bool
	_ = s.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1
		  FROM organization_members
		  WHERE agent_id = NULLIF($1, '')::uuid
		    AND organization_id = $2
		    AND status = 'active'
		)
	`, agentID, s.organizationID(ctx)).Scan(&exists)
	return exists
}

func parseJSONObject(raw string) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func parseJSONValue(raw string) any {
	var out any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func defaultVariables(value any) any {
	if value == nil {
		return []string{}
	}
	return value
}

func firstByID(items []map[string]any, id string) (map[string]any, error) {
	for _, item := range items {
		if item["id"] == id {
			return item, nil
		}
	}
	return nil, pgx.ErrNoRows
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func valueOrDefault(value *string, fallback string) string {
	parsed := valueOrEmpty(value)
	if parsed == "" {
		return fallback
	}
	return parsed
}

func validLifecycleStatus(value string) bool {
	switch value {
	case "lead", "prospect", "customer", "inactive":
		return true
	default:
		return false
	}
}

func validConversationStatus(value string) bool {
	switch value {
	case "open", "pending_human", "resolved":
		return true
	default:
		return false
	}
}

func validTaskStatus(value string) bool {
	switch value {
	case "open", "done", "cancelled":
		return true
	default:
		return false
	}
}

func validPriority(value string) bool {
	switch value {
	case "low", "normal", "high", "urgent":
		return true
	default:
		return false
	}
}

func validTemplateStatus(value string) bool {
	switch value {
	case "draft", "active", "archived":
		return true
	default:
		return false
	}
}

func buildContactSummary(name, phone string, messages []map[string]any) string {
	identity := strings.TrimSpace(name)
	if identity == "" {
		identity = strings.TrimSpace(phone)
	}
	if len(messages) == 0 {
		return "Belum ada riwayat chat yang cukup untuk diringkas."
	}
	customerMessages := 0
	agentMessages := 0
	lastText := ""
	for _, message := range messages {
		if message["direction"] == "inbound" {
			customerMessages++
		} else {
			agentMessages++
		}
		if lastText == "" {
			lastText, _ = message["text"].(string)
		}
	}
	if lastText != "" && len(lastText) > 140 {
		lastText = lastText[:140] + "..."
	}
	return fmt.Sprintf("%s memiliki %d pesan masuk dan %d balasan tim/AI. Pesan terakhir: %s", identity, customerMessages, agentMessages, lastText)
}
