package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type commerceProductRequest struct {
	SKU               *string  `json:"sku"`
	Name              *string  `json:"name"`
	Description       *string  `json:"description"`
	UnitPrice         *float64 `json:"unitPrice"`
	StockQuantity     *int     `json:"stockQuantity"`
	LowStockThreshold *int     `json:"lowStockThreshold"`
	Status            *string  `json:"status"`
}

type commerceOrderDraftRequest struct {
	ContactID       *string                    `json:"contactId"`
	ConversationID  *string                    `json:"conversationId"`
	CustomerName    *string                    `json:"customerName"`
	CustomerPhone   *string                    `json:"customerPhone"`
	StageID         *string                    `json:"stageId"`
	Status          *string                    `json:"status"`
	Source          *string                    `json:"source"`
	Notes           *string                    `json:"notes"`
	FulfillmentType *string                    `json:"fulfillmentType"`
	RecipientName   *string                    `json:"recipientName"`
	RecipientPhone  *string                    `json:"recipientPhone"`
	AddressLine     *string                    `json:"addressLine"`
	AddressArea     *string                    `json:"addressArea"`
	AddressNotes    *string                    `json:"addressNotes"`
	Items           []commerceOrderItemRequest `json:"items"`
}

type commerceOrderItemRequest struct {
	ProductID *string  `json:"productId"`
	Quantity  int      `json:"quantity"`
	UnitPrice *float64 `json:"unitPrice"`
}

type commerceOrderPipelineRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsDefault   *bool   `json:"isDefault"`
}

type commerceOrderStageRequest struct {
	PipelineID *string `json:"pipelineId"`
	Name       *string `json:"name"`
	Position   *int    `json:"position"`
	StageType  *string `json:"type"`
	Color      *string `json:"color"`
}

type commerceOrderSettingsRequest struct {
	EnabledFulfillmentTypes []string `json:"enabledFulfillmentTypes"`
}

func canManageCommerceCatalog(role string) bool {
	return role == "owner" || role == "super_admin" || role == "admin"
}

func canCreateCommerceDraft(role string) bool {
	return role == "owner" || isOpsRole(role)
}

func (s *Server) handleCommerceProducts(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "commerce") {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.listCommerceProducts(r.Context(), strings.TrimSpace(r.URL.Query().Get("query")), strings.TrimSpace(r.URL.Query().Get("status")))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "canManage": canManageCommerceCatalog(s.role(r.Context()))})
	case http.MethodPost:
		if !canManageCommerceCatalog(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage products"})
			return
		}
		var req commerceProductRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createCommerceProduct(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCommerceProductRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "commerce") {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/commerce/products/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		if !canManageCommerceCatalog(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage products"})
			return
		}
		var req commerceProductRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.updateCommerceProduct(r.Context(), id, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": item})
	case http.MethodDelete:
		if !canManageCommerceCatalog(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage products"})
			return
		}
		if err := s.deleteCommerceProduct(r.Context(), id); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, pgx.ErrNoRows) {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCommerceOrderDrafts(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "commerce") {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.listCommerceOrderDrafts(r.Context(), strings.TrimSpace(r.URL.Query().Get("status")))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		if !canCreateCommerceDraft(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only operational roles can create order drafts"})
			return
		}
		var req commerceOrderDraftRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createCommerceOrderDraft(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCommerceOrderDraftRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "commerce") {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/commerce/order-drafts/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		var req commerceOrderDraftRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.updateCommerceOrderDraft(r.Context(), id, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCommerceOrderPipelines(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "commerce") {
		return
	}
	switch r.Method {
	case http.MethodGet:
		payload, err := s.commerceOrderPipelineBoard(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, payload)
	case http.MethodPost:
		if !canManageCommerceCatalog(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage order pipelines"})
			return
		}
		var req commerceOrderPipelineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createCommerceOrderPipeline(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCommerceOrderPipelineRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "commerce") {
		return
	}
	if !canManageCommerceCatalog(s.role(r.Context())) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage order pipelines"})
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/commerce/order-pipelines/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		var req commerceOrderPipelineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.updateCommerceOrderPipeline(r.Context(), id, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCommerceOrderStages(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "commerce") {
		return
	}
	if !canManageCommerceCatalog(s.role(r.Context())) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage order stages"})
		return
	}
	switch r.Method {
	case http.MethodPost:
		var req commerceOrderStageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createCommerceOrderStage(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCommerceOrderStageRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "commerce") {
		return
	}
	if !canManageCommerceCatalog(s.role(r.Context())) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage order stages"})
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/commerce/order-stages/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		var req commerceOrderStageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.updateCommerceOrderStage(r.Context(), id, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": item})
	case http.MethodDelete:
		if err := s.deleteCommerceOrderStage(r.Context(), id); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, pgx.ErrNoRows) {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCommerceOrderSettings(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "commerce") {
		return
	}
	switch r.Method {
	case http.MethodGet:
		settings, err := s.commerceOrderSettings(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"settings": settings, "options": commerceFulfillmentOptions()})
	case http.MethodPatch:
		if !canManageCommerceCatalog(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage order settings"})
			return
		}
		var req commerceOrderSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		settings, err := s.updateCommerceOrderSettings(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"settings": settings, "options": commerceFulfillmentOptions()})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) listCommerceProducts(ctx context.Context, query, status string) ([]map[string]any, error) {
	if status == "" {
		status = "active"
	}
	limit := 150
	rows, err := s.db.Query(ctx, `
		SELECT id::text, COALESCE(sku, ''), name, COALESCE(description, ''), unit_price::float8,
		       stock_quantity, reserved_quantity, low_stock_threshold, status, created_at, updated_at
		FROM commerce_products
		WHERE organization_id = $1
		  AND ($2 = 'all' OR status = $2)
		  AND (
		    $3 = ''
		    OR lower(name) LIKE '%' || lower($3) || '%'
		    OR lower(COALESCE(sku, '')) LIKE '%' || lower($3) || '%'
		  )
		ORDER BY updated_at DESC
		LIMIT $4
	`, s.organizationID(ctx), status, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, sku, name, description, itemStatus string
		var unitPrice float64
		var stockQuantity, reservedQuantity, lowStockThreshold int
		var createdAt, updatedAt any
		if err := rows.Scan(&id, &sku, &name, &description, &unitPrice, &stockQuantity, &reservedQuantity, &lowStockThreshold, &itemStatus, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		available := stockQuantity - reservedQuantity
		if available < 0 {
			available = 0
		}
		items = append(items, map[string]any{
			"id":                id,
			"sku":               sku,
			"name":              name,
			"description":       description,
			"unitPrice":         unitPrice,
			"stockQuantity":     stockQuantity,
			"reservedQuantity":  reservedQuantity,
			"availableStock":    available,
			"lowStockThreshold": lowStockThreshold,
			"lowStock":          lowStockThreshold > 0 && available <= lowStockThreshold,
			"status":            itemStatus,
			"createdAt":         createdAt,
			"updatedAt":         updatedAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) createCommerceProduct(ctx context.Context, req commerceProductRequest) (map[string]any, error) {
	name := strings.TrimSpace(valueOrEmpty(req.Name))
	if name == "" {
		return nil, errors.New("name is required")
	}
	status := normalizeCommerceProductStatus(valueOrEmpty(req.Status))
	stockQuantity := 0
	if req.StockQuantity != nil {
		stockQuantity = *req.StockQuantity
	}
	if stockQuantity < 0 {
		return nil, errors.New("stock cannot be negative")
	}
	lowStockThreshold := 0
	if req.LowStockThreshold != nil {
		lowStockThreshold = *req.LowStockThreshold
	}
	if lowStockThreshold < 0 {
		return nil, errors.New("low stock threshold cannot be negative")
	}
	unitPrice := 0.0
	if req.UnitPrice != nil {
		unitPrice = *req.UnitPrice
	}
	if unitPrice < 0 {
		return nil, errors.New("unit price cannot be negative")
	}
	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO commerce_products (
		  organization_id, sku, name, description, unit_price, stock_quantity, low_stock_threshold, status, created_by, updated_by
		)
		VALUES ($1, NULLIF($2, ''), $3, NULLIF($4, ''), $5, $6, $7, $8, NULLIF($9, '')::uuid, NULLIF($9, '')::uuid)
		RETURNING id::text
	`, s.organizationID(ctx), strings.TrimSpace(valueOrEmpty(req.SKU)), name, strings.TrimSpace(valueOrEmpty(req.Description)), unitPrice, stockQuantity, lowStockThreshold, status, s.agentID(ctx)).Scan(&id)
	if err != nil {
		return nil, err
	}
	if stockQuantity > 0 {
		_ = s.insertCommerceStockMovement(ctx, id, "", "manual_adjustment", stockQuantity, 0, 0, stockQuantity, 0, 0, "Stok awal produk")
	}
	_ = s.insertAuditLog(ctx, "commerce.product_create", "commerce_product", id, map[string]any{"name": name, "stock": stockQuantity})
	return s.commerceProductByID(ctx, id)
}

func (s *Server) updateCommerceProduct(ctx context.Context, id string, req commerceProductRequest) (map[string]any, error) {
	current, err := s.commerceProductByID(ctx, id)
	if err != nil {
		return nil, errors.New("product not found")
	}
	sku := strings.TrimSpace(stringFromMap(current, "sku"))
	name := strings.TrimSpace(stringFromMap(current, "name"))
	description := strings.TrimSpace(stringFromMap(current, "description"))
	unitPrice := floatFromMap(current, "unitPrice")
	stockQuantity := intFromMap(current, "stockQuantity")
	previousStockQuantity := stockQuantity
	previousReservedQuantity := intFromMap(current, "reservedQuantity")
	lowStockThreshold := intFromMap(current, "lowStockThreshold")
	status := strings.TrimSpace(stringFromMap(current, "status"))
	if req.SKU != nil {
		sku = strings.TrimSpace(*req.SKU)
	}
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
	}
	if req.UnitPrice != nil {
		unitPrice = *req.UnitPrice
	}
	if req.StockQuantity != nil {
		stockQuantity = *req.StockQuantity
	}
	if req.LowStockThreshold != nil {
		lowStockThreshold = *req.LowStockThreshold
	}
	if req.Status != nil {
		status = normalizeCommerceProductStatus(*req.Status)
	}
	if name == "" {
		return nil, errors.New("name is required")
	}
	if unitPrice < 0 || stockQuantity < 0 {
		return nil, errors.New("price and stock cannot be negative")
	}
	if lowStockThreshold < 0 {
		return nil, errors.New("low stock threshold cannot be negative")
	}
	_, err = s.db.Exec(ctx, `
		UPDATE commerce_products
		SET sku = NULLIF($2, ''),
		    name = $3,
		    description = NULLIF($4, ''),
		    unit_price = $5,
		    stock_quantity = $6,
		    low_stock_threshold = $7,
		    status = $8,
		    updated_by = NULLIF($9, '')::uuid,
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $10
	`, id, sku, name, description, unitPrice, stockQuantity, lowStockThreshold, status, s.agentID(ctx), s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	if previousStockQuantity != stockQuantity {
		_ = s.insertCommerceStockMovement(ctx, id, "", "manual_adjustment", stockQuantity-previousStockQuantity, 0, previousStockQuantity, stockQuantity, previousReservedQuantity, previousReservedQuantity, "Penyesuaian stok manual")
	}
	_ = s.insertAuditLog(ctx, "commerce.product_update", "commerce_product", id, map[string]any{"name": name, "status": status})
	return s.commerceProductByID(ctx, id)
}

func (s *Server) deleteCommerceProduct(ctx context.Context, id string) error {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM commerce_products
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, id, s.organizationID(ctx))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	_ = s.insertAuditLog(ctx, "commerce.product_delete", "commerce_product", id, map[string]any{})
	return nil
}

func (s *Server) commerceProductByID(ctx context.Context, id string) (map[string]any, error) {
	var sku, name, description, status string
	var unitPrice float64
	var stockQuantity, reservedQuantity, lowStockThreshold int
	var createdAt, updatedAt any
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(sku, ''), name, COALESCE(description, ''), unit_price::float8,
		       stock_quantity, reserved_quantity, low_stock_threshold, status, created_at, updated_at
		FROM commerce_products
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, id, s.organizationID(ctx)).Scan(&sku, &name, &description, &unitPrice, &stockQuantity, &reservedQuantity, &lowStockThreshold, &status, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	available := stockQuantity - reservedQuantity
	if available < 0 {
		available = 0
	}
	return map[string]any{
		"id":                id,
		"sku":               sku,
		"name":              name,
		"description":       description,
		"unitPrice":         unitPrice,
		"stockQuantity":     stockQuantity,
		"reservedQuantity":  reservedQuantity,
		"availableStock":    available,
		"lowStockThreshold": lowStockThreshold,
		"lowStock":          lowStockThreshold > 0 && available <= lowStockThreshold,
		"status":            status,
		"createdAt":         createdAt,
		"updatedAt":         updatedAt,
	}, nil
}

func (s *Server) listCommerceOrderDrafts(ctx context.Context, status string) ([]map[string]any, error) {
	if status == "" {
		status = "all"
	}
	if defaultStageID, _, err := s.defaultCommerceOrderStage(ctx); err == nil && strings.TrimSpace(defaultStageID) != "" {
		_, _ = s.db.Exec(ctx, `
			UPDATE commerce_order_drafts
			SET stage_id = NULLIF($2, '')::uuid, updated_at = updated_at
			WHERE organization_id = $1 AND stage_id IS NULL
		`, s.organizationID(ctx), defaultStageID)
	}
	rows, err := s.db.Query(ctx, `
		SELECT id::text, COALESCE(customer_name, ''), COALESCE(customer_phone, ''), status, source,
		       total_amount::float8, COALESCE(notes, ''), created_at, updated_at
		FROM commerce_order_drafts
		WHERE organization_id = $1
		  AND ($2 = 'all' OR status = $2)
		ORDER BY created_at DESC
		LIMIT 100
	`, s.organizationID(ctx), status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, customerName, customerPhone, itemStatus, source, notes string
		var totalAmount float64
		var createdAt, updatedAt any
		if err := rows.Scan(&id, &customerName, &customerPhone, &itemStatus, &source, &totalAmount, &notes, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		item, err := s.commerceOrderDraftByID(ctx, id)
		if err != nil {
			return nil, err
		}
		item["customerName"] = customerName
		item["customerPhone"] = customerPhone
		item["status"] = itemStatus
		item["source"] = source
		item["totalAmount"] = totalAmount
		item["notes"] = notes
		item["createdAt"] = createdAt
		item["updatedAt"] = updatedAt
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) createCommerceOrderDraft(ctx context.Context, req commerceOrderDraftRequest) (map[string]any, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("at least one item is required")
	}
	status := normalizeCommerceOrderStatus(valueOrEmpty(req.Status))
	if status == "confirmed" {
		status = "awaiting_confirmation"
	}
	enabledFulfillmentTypes, err := s.enabledCommerceFulfillmentTypes(ctx)
	if err != nil {
		return nil, err
	}
	fulfillmentType := defaultCommerceFulfillmentType(enabledFulfillmentTypes)
	if req.FulfillmentType != nil {
		fulfillmentType = normalizeCommerceFulfillmentType(*req.FulfillmentType)
	}
	addressLine := strings.TrimSpace(valueOrEmpty(req.AddressLine))
	if !commerceFulfillmentTypeEnabled(enabledFulfillmentTypes, fulfillmentType) {
		return nil, errors.New("cara terima pesanan ini tidak aktif")
	}
	if err := validateCommerceFulfillment(fulfillmentType, addressLine); err != nil {
		return nil, err
	}
	source := strings.TrimSpace(valueOrEmpty(req.Source))
	if source == "" {
		source = "dashboard"
	}
	contactID := strings.TrimSpace(valueOrEmpty(req.ContactID))
	if contactID != "" && !s.contactExists(ctx, contactID) {
		return nil, errors.New("contact not found")
	}
	conversationID := strings.TrimSpace(valueOrEmpty(req.ConversationID))
	if conversationID != "" && !s.conversationBelongsToOrganization(ctx, conversationID) {
		return nil, errors.New("conversation not found")
	}
	stageID := strings.TrimSpace(valueOrEmpty(req.StageID))
	if stageID == "" {
		var err error
		stageID, _, err = s.defaultCommerceOrderStage(ctx)
		if err != nil {
			return nil, err
		}
	} else if !s.commerceOrderStageBelongsToOrganization(ctx, stageID) {
		return nil, errors.New("order stage not found")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	preparedItems, total, err := s.prepareCommerceOrderItems(ctx, req.Items)
	if err != nil {
		return nil, err
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO commerce_order_drafts (
		  organization_id, contact_id, conversation_id, customer_name, customer_phone,
		  stage_id, status, source, total_amount, notes, fulfillment_type,
		  recipient_name, recipient_phone, address_line, address_area, address_notes, created_by
		)
		VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, NULLIF($4, ''), NULLIF($5, ''),
		        NULLIF($6, '')::uuid, $7, $8, $9, NULLIF($10, ''), $11,
		        NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, ''), NULLIF($16, ''), NULLIF($17, '')::uuid)
		RETURNING id::text
	`, s.organizationID(ctx), contactID, conversationID, strings.TrimSpace(valueOrEmpty(req.CustomerName)), strings.TrimSpace(valueOrEmpty(req.CustomerPhone)), stageID, status, source, total, strings.TrimSpace(valueOrEmpty(req.Notes)), fulfillmentType, strings.TrimSpace(valueOrEmpty(req.RecipientName)), strings.TrimSpace(valueOrEmpty(req.RecipientPhone)), addressLine, strings.TrimSpace(valueOrEmpty(req.AddressArea)), strings.TrimSpace(valueOrEmpty(req.AddressNotes)), s.agentID(ctx)).Scan(&id)
	if err != nil {
		return nil, err
	}
	for _, item := range preparedItems {
		_, err = tx.Exec(ctx, `
			INSERT INTO commerce_order_items (
			  order_draft_id, organization_id, product_id, product_name, sku, quantity, unit_price, line_total
			)
			VALUES (NULLIF($1, '')::uuid, $2, NULLIF($3, '')::uuid, $4, NULLIF($5, ''), $6, $7, $8)
		`, id, s.organizationID(ctx), item["productId"], item["productName"], item["sku"], item["quantity"], item["unitPrice"], item["lineTotal"])
		if err != nil {
			return nil, err
		}
	}
	if err := s.insertAuditLogTx(ctx, tx, "commerce.order_draft_create", "commerce_order_draft", id, map[string]any{"totalAmount": total, "source": source}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	item, err := s.commerceOrderDraftByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.notifyBusinessPluginGroup(ctx, groupNotificationTriggerCommerceOrderCreated, item)
	return item, nil
}

func (s *Server) prepareCommerceOrderItems(ctx context.Context, items []commerceOrderItemRequest) ([]map[string]any, float64, error) {
	preparedItems := []map[string]any{}
	total := 0.0
	quantityByProduct := map[string]int{}
	for _, item := range items {
		productID := strings.TrimSpace(valueOrEmpty(item.ProductID))
		if productID == "" {
			return nil, 0, errors.New("productId is required")
		}
		quantity := item.Quantity
		if quantity <= 0 {
			return nil, 0, errors.New("quantity must be greater than zero")
		}
		product, err := s.commerceProductByID(ctx, productID)
		if err != nil {
			return nil, 0, errors.New("product not found")
		}
		if stringFromMap(product, "status") != "active" {
			return nil, 0, errors.New("product is inactive")
		}
		available := intFromMap(product, "availableStock")
		quantityByProduct[productID] += quantity
		if quantityByProduct[productID] > available {
			return nil, 0, errors.New("stock is not enough")
		}
		unitPrice := floatFromMap(product, "unitPrice")
		if item.UnitPrice != nil {
			unitPrice = *item.UnitPrice
		}
		if unitPrice < 0 {
			return nil, 0, errors.New("unit price cannot be negative")
		}
		lineTotal := unitPrice * float64(quantity)
		total += lineTotal
		preparedItems = append(preparedItems, map[string]any{
			"productId":   productID,
			"productName": stringFromMap(product, "name"),
			"sku":         stringFromMap(product, "sku"),
			"quantity":    quantity,
			"unitPrice":   unitPrice,
			"lineTotal":   lineTotal,
		})
	}
	return preparedItems, total, nil
}

func (s *Server) updateCommerceOrderDraft(ctx context.Context, id string, req commerceOrderDraftRequest) (map[string]any, error) {
	current, err := s.commerceOrderDraftByID(ctx, id)
	if err != nil {
		return nil, errors.New("order draft not found")
	}
	currentStatus := stringFromMap(current, "status")
	status := currentStatus
	notes := stringFromMap(current, "notes")
	customerName := stringFromMap(current, "customerName")
	customerPhone := stringFromMap(current, "customerPhone")
	stageID := stringFromMap(current, "stageId")
	fulfillmentType := stringFromMap(current, "fulfillmentType")
	recipientName := stringFromMap(current, "recipientName")
	recipientPhone := stringFromMap(current, "recipientPhone")
	addressLine := stringFromMap(current, "addressLine")
	addressArea := stringFromMap(current, "addressArea")
	addressNotes := stringFromMap(current, "addressNotes")
	total := floatFromMap(current, "totalAmount")
	updateItems := req.Items != nil
	preparedItems := []map[string]any{}

	if req.Status != nil {
		status = normalizeCommerceOrderStatus(*req.Status)
	}
	if req.Notes != nil {
		notes = strings.TrimSpace(*req.Notes)
	}
	if req.CustomerName != nil {
		customerName = strings.TrimSpace(*req.CustomerName)
	}
	if req.CustomerPhone != nil {
		customerPhone = strings.TrimSpace(*req.CustomerPhone)
	}
	if req.StageID != nil {
		stageID = strings.TrimSpace(*req.StageID)
		if stageID != "" && !s.commerceOrderStageBelongsToOrganization(ctx, stageID) {
			return nil, errors.New("order stage not found")
		}
	}
	if stageID == "" {
		var err error
		stageID, _, err = s.defaultCommerceOrderStage(ctx)
		if err != nil {
			return nil, err
		}
	}
	if req.FulfillmentType != nil {
		fulfillmentType = normalizeCommerceFulfillmentType(*req.FulfillmentType)
		enabledFulfillmentTypes, err := s.enabledCommerceFulfillmentTypes(ctx)
		if err != nil {
			return nil, err
		}
		if !commerceFulfillmentTypeEnabled(enabledFulfillmentTypes, fulfillmentType) {
			return nil, errors.New("cara terima pesanan ini tidak aktif")
		}
	}
	if req.RecipientName != nil {
		recipientName = strings.TrimSpace(*req.RecipientName)
	}
	if req.RecipientPhone != nil {
		recipientPhone = strings.TrimSpace(*req.RecipientPhone)
	}
	if req.AddressLine != nil {
		addressLine = strings.TrimSpace(*req.AddressLine)
	}
	if req.AddressArea != nil {
		addressArea = strings.TrimSpace(*req.AddressArea)
	}
	if req.AddressNotes != nil {
		addressNotes = strings.TrimSpace(*req.AddressNotes)
	}
	if err := validateCommerceFulfillment(fulfillmentType, addressLine); err != nil {
		return nil, err
	}
	if updateItems {
		if currentStatus == "confirmed" {
			return nil, errors.New("confirmed order items cannot be edited")
		}
		if len(req.Items) == 0 {
			return nil, errors.New("at least one item is required")
		}
		preparedItems, total, err = s.prepareCommerceOrderItems(ctx, req.Items)
		if err != nil {
			return nil, err
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if currentStatus == "confirmed" && status != "confirmed" {
		if err := s.releaseCommerceOrderDraftReservationTx(ctx, tx, id); err != nil {
			return nil, err
		}
	}
	if updateItems {
		if _, err := tx.Exec(ctx, `
			DELETE FROM commerce_order_items
			WHERE order_draft_id = NULLIF($1, '')::uuid AND organization_id = $2
		`, id, s.organizationID(ctx)); err != nil {
			return nil, err
		}
		for _, item := range preparedItems {
			_, err = tx.Exec(ctx, `
				INSERT INTO commerce_order_items (
				  order_draft_id, organization_id, product_id, product_name, sku, quantity, unit_price, line_total
				)
				VALUES (NULLIF($1, '')::uuid, $2, NULLIF($3, '')::uuid, $4, NULLIF($5, ''), $6, $7, $8)
			`, id, s.organizationID(ctx), item["productId"], item["productName"], item["sku"], item["quantity"], item["unitPrice"], item["lineTotal"])
			if err != nil {
				return nil, err
			}
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE commerce_order_drafts
		SET customer_name = NULLIF($2, ''),
		    customer_phone = NULLIF($3, ''),
		    stage_id = NULLIF($4, '')::uuid,
		    status = $5,
		    total_amount = $6,
		    notes = NULLIF($7, ''),
		    fulfillment_type = $8,
		    recipient_name = NULLIF($9, ''),
		    recipient_phone = NULLIF($10, ''),
		    address_line = NULLIF($11, ''),
		    address_area = NULLIF($12, ''),
		    address_notes = NULLIF($13, ''),
		    confirmed_at = CASE WHEN $5 = 'confirmed' AND confirmed_at IS NULL THEN NOW() ELSE confirmed_at END,
		    cancelled_at = CASE WHEN $5 = 'cancelled' AND cancelled_at IS NULL THEN NOW() ELSE cancelled_at END,
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $14
	`, id, customerName, customerPhone, stageID, status, total, notes, fulfillmentType, recipientName, recipientPhone, addressLine, addressArea, addressNotes, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	if currentStatus != "confirmed" && status == "confirmed" {
		if err := s.reserveCommerceOrderDraftTx(ctx, tx, id); err != nil {
			return nil, err
		}
	}
	if err := s.insertAuditLogTx(ctx, tx, "commerce.order_draft_update", "commerce_order_draft", id, map[string]any{
		"previousStatus": currentStatus,
		"status":         status,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	item, err := s.commerceOrderDraftByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if currentStatus != "confirmed" && status == "confirmed" {
		s.notifyBusinessPluginGroup(ctx, groupNotificationTriggerCommerceOrderConfirmed, item)
	}
	return item, nil
}

func (s *Server) reserveCommerceOrderDraftTx(ctx context.Context, tx pgx.Tx, orderDraftID string) error {
	type reservationItem struct {
		productID   string
		productName string
		quantity    int
	}
	rows, err := tx.Query(ctx, `
		SELECT COALESCE(product_id::text, ''), product_name, quantity
		FROM commerce_order_items
		WHERE order_draft_id = NULLIF($1, '')::uuid AND organization_id = $2
	`, orderDraftID, s.organizationID(ctx))
	if err != nil {
		return err
	}
	defer rows.Close()

	items := make([]reservationItem, 0)
	for rows.Next() {
		var item reservationItem
		if err := rows.Scan(&item.productID, &item.productName, &item.quantity); err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	for _, item := range items {
		if strings.TrimSpace(item.productID) == "" {
			return errors.New("order item is not linked to a product")
		}
		var updatedID string
		var stockQuantity, reservedAfter int
		err := tx.QueryRow(ctx, `
			UPDATE commerce_products
			SET reserved_quantity = reserved_quantity + $3,
			    updated_at = NOW()
			WHERE id = NULLIF($1, '')::uuid
			  AND organization_id = $2
			  AND status = 'active'
			  AND stock_quantity - reserved_quantity >= $3
			RETURNING id::text, stock_quantity, reserved_quantity
		`, item.productID, s.organizationID(ctx), item.quantity).Scan(&updatedID, &stockQuantity, &reservedAfter)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errors.New("stock is not enough for " + item.productName)
			}
			return err
		}
		if err := s.insertCommerceStockMovementTx(ctx, tx, item.productID, orderDraftID, "reserve_order", 0, item.quantity, stockQuantity, stockQuantity, reservedAfter-item.quantity, reservedAfter, "Reserve stok untuk pesanan"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) releaseCommerceOrderDraftReservationTx(ctx context.Context, tx pgx.Tx, orderDraftID string) error {
	type reservationItem struct {
		productID string
		quantity  int
	}
	rows, err := tx.Query(ctx, `
		SELECT COALESCE(product_id::text, ''), quantity
		FROM commerce_order_items
		WHERE order_draft_id = NULLIF($1, '')::uuid AND organization_id = $2
	`, orderDraftID, s.organizationID(ctx))
	if err != nil {
		return err
	}
	defer rows.Close()

	items := make([]reservationItem, 0)
	for rows.Next() {
		var item reservationItem
		if err := rows.Scan(&item.productID, &item.quantity); err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	for _, item := range items {
		if strings.TrimSpace(item.productID) == "" {
			continue
		}
		var stockQuantity, reservedAfter int
		if err := tx.QueryRow(ctx, `
			UPDATE commerce_products
			SET reserved_quantity = GREATEST(0, reserved_quantity - $3),
			    updated_at = NOW()
			WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
			RETURNING stock_quantity, reserved_quantity
		`, item.productID, s.organizationID(ctx), item.quantity).Scan(&stockQuantity, &reservedAfter); err != nil {
			return err
		}
		if err := s.insertCommerceStockMovementTx(ctx, tx, item.productID, orderDraftID, "release_order", 0, -item.quantity, stockQuantity, stockQuantity, reservedAfter+item.quantity, reservedAfter, "Lepas reserve pesanan"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) commerceOrderDraftByID(ctx context.Context, id string) (map[string]any, error) {
	var contactID, conversationID, customerName, customerPhone, stageID, stageName, stageType, stageColor, status, source, notes string
	var fulfillmentType, recipientName, recipientPhone, addressLine, addressArea, addressNotes string
	var totalAmount float64
	var createdAt, updatedAt any
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(contact_id::text, ''), COALESCE(conversation_id::text, ''), COALESCE(customer_name, ''),
		       COALESCE(customer_phone, ''), COALESCE(stage_id::text, ''), COALESCE(st.name, ''), COALESCE(st.stage_type, ''),
		       COALESCE(st.color, ''), status, source, total_amount::float8, COALESCE(notes, ''),
		       fulfillment_type, COALESCE(recipient_name, ''), COALESCE(recipient_phone, ''),
		       COALESCE(address_line, ''), COALESCE(address_area, ''), COALESCE(address_notes, ''),
		       d.created_at, d.updated_at
		FROM commerce_order_drafts d
		LEFT JOIN commerce_order_stages st ON st.id = d.stage_id AND st.organization_id = d.organization_id
		WHERE d.id = NULLIF($1, '')::uuid AND d.organization_id = $2
	`, id, s.organizationID(ctx)).Scan(&contactID, &conversationID, &customerName, &customerPhone, &stageID, &stageName, &stageType, &stageColor, &status, &source, &totalAmount, &notes, &fulfillmentType, &recipientName, &recipientPhone, &addressLine, &addressArea, &addressNotes, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT id::text, COALESCE(product_id::text, ''), product_name, COALESCE(sku, ''), quantity, unit_price::float8, line_total::float8
		FROM commerce_order_items
		WHERE order_draft_id = NULLIF($1, '')::uuid AND organization_id = $2
		ORDER BY created_at ASC
	`, id, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var itemID, productID, productName, sku string
		var quantity int
		var unitPrice, lineTotal float64
		if err := rows.Scan(&itemID, &productID, &productName, &sku, &quantity, &unitPrice, &lineTotal); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":          itemID,
			"productId":   productID,
			"productName": productName,
			"sku":         sku,
			"quantity":    quantity,
			"unitPrice":   unitPrice,
			"lineTotal":   lineTotal,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":              id,
		"contactId":       contactID,
		"conversationId":  conversationID,
		"customerName":    customerName,
		"customerPhone":   customerPhone,
		"stageId":         stageID,
		"stageName":       stageName,
		"stageType":       stageType,
		"stageColor":      stageColor,
		"status":          status,
		"source":          source,
		"totalAmount":     totalAmount,
		"notes":           notes,
		"fulfillmentType": fulfillmentType,
		"recipientName":   recipientName,
		"recipientPhone":  recipientPhone,
		"addressLine":     addressLine,
		"addressArea":     addressArea,
		"addressNotes":    addressNotes,
		"items":           items,
		"createdAt":       createdAt,
		"updatedAt":       updatedAt,
	}, nil
}

func (s *Server) commerceOrderPipelineBoard(ctx context.Context) (map[string]any, error) {
	activePipelineID, err := s.ensureDefaultCommerceOrderPipeline(ctx)
	if err != nil {
		return nil, err
	}
	pipelines, err := s.listCommerceOrderPipelines(ctx)
	if err != nil {
		return nil, err
	}
	stages, err := s.listCommerceOrderStages(ctx, activePipelineID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"activePipelineId": activePipelineID,
		"pipelines":        pipelines,
		"stages":           stages,
	}, nil
}

func (s *Server) ensureDefaultCommerceOrderPipeline(ctx context.Context) (string, error) {
	var id string
	err := s.db.QueryRow(ctx, `
		SELECT id::text
		FROM commerce_order_pipelines
		WHERE organization_id = $1 AND is_default = TRUE
		ORDER BY created_at ASC
		LIMIT 1
	`, s.organizationID(ctx)).Scan(&id)
	if err == nil {
		return id, s.ensureDefaultCommerceOrderStages(ctx, id)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	err = s.db.QueryRow(ctx, `
		INSERT INTO commerce_order_pipelines (organization_id, name, description, is_default, created_by)
		VALUES ($1, 'Alur Pesanan', 'Alur operasional pesanan default.', TRUE, NULLIF($2, '')::uuid)
		RETURNING id::text
	`, s.organizationID(ctx), s.agentID(ctx)).Scan(&id)
	if err != nil {
		lookupErr := s.db.QueryRow(ctx, `
			SELECT id::text
			FROM commerce_order_pipelines
			WHERE organization_id = $1
			  AND (is_default = TRUE OR lower(name) = lower('Alur Pesanan'))
			ORDER BY is_default DESC, created_at ASC
			LIMIT 1
		`, s.organizationID(ctx)).Scan(&id)
		if lookupErr != nil {
			return "", err
		}
	}
	return id, s.ensureDefaultCommerceOrderStages(ctx, id)
}

func (s *Server) ensureDefaultCommerceOrderStages(ctx context.Context, pipelineID string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO commerce_order_stages (organization_id, pipeline_id, name, position, stage_type, color)
		VALUES
		  ($1, NULLIF($2, '')::uuid, 'Baru', 10, 'open', '#2196F3'),
		  ($1, NULLIF($2, '')::uuid, 'Diproses', 20, 'open', '#F97316'),
		  ($1, NULLIF($2, '')::uuid, 'Siap dikirim/diambil', 30, 'open', '#10B981'),
		  ($1, NULLIF($2, '')::uuid, 'Selesai', 40, 'completed', '#059669'),
		  ($1, NULLIF($2, '')::uuid, 'Batal', 50, 'cancelled', '#EF4444')
		ON CONFLICT DO NOTHING
	`, s.organizationID(ctx), pipelineID)
	return err
}

func (s *Server) listCommerceOrderPipelines(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, COALESCE(description, ''), is_default, created_at, updated_at
		FROM commerce_order_pipelines
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
		var createdAt, updatedAt any
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

func (s *Server) listCommerceOrderStages(ctx context.Context, pipelineID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, pipeline_id::text, name, position, stage_type, color, created_at, updated_at
		FROM commerce_order_stages
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
		var position int
		var createdAt, updatedAt any
		if err := rows.Scan(&id, &nextPipelineID, &name, &position, &stageType, &color, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":         id,
			"pipelineId": nextPipelineID,
			"name":       name,
			"position":   position,
			"type":       stageType,
			"color":      color,
			"createdAt":  createdAt,
			"updatedAt":  updatedAt,
		})
	}
	return items, rows.Err()
}

func (s *Server) createCommerceOrderPipeline(ctx context.Context, req commerceOrderPipelineRequest) (map[string]any, error) {
	name := strings.TrimSpace(valueOrEmpty(req.Name))
	if name == "" {
		return nil, errors.New("pipeline name is required")
	}
	isDefault := req.IsDefault != nil && *req.IsDefault
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if isDefault {
		if _, err := tx.Exec(ctx, `UPDATE commerce_order_pipelines SET is_default = FALSE, updated_at = NOW() WHERE organization_id = $1`, s.organizationID(ctx)); err != nil {
			return nil, err
		}
	}
	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO commerce_order_pipelines (organization_id, name, description, is_default, created_by)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, '')::uuid)
		RETURNING id::text
	`, s.organizationID(ctx), name, strings.TrimSpace(valueOrEmpty(req.Description)), isDefault, s.agentID(ctx)).Scan(&id); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	_ = s.ensureDefaultCommerceOrderStages(ctx, id)
	return s.commerceOrderPipelineByID(ctx, id)
}

func (s *Server) updateCommerceOrderPipeline(ctx context.Context, id string, req commerceOrderPipelineRequest) (map[string]any, error) {
	current, err := s.commerceOrderPipelineByID(ctx, id)
	if err != nil {
		return nil, errors.New("pipeline not found")
	}
	name := strings.TrimSpace(stringFromMap(current, "name"))
	description := strings.TrimSpace(stringFromMap(current, "description"))
	isDefault, _ := current["isDefault"].(bool)
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
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
		if _, err := tx.Exec(ctx, `UPDATE commerce_order_pipelines SET is_default = FALSE, updated_at = NOW() WHERE organization_id = $1 AND id <> NULLIF($2, '')::uuid`, s.organizationID(ctx), id); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE commerce_order_pipelines
		SET name = $2, description = NULLIF($3, ''), is_default = $4, updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $5
	`, id, name, description, isDefault, s.organizationID(ctx)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.commerceOrderPipelineByID(ctx, id)
}

func (s *Server) commerceOrderPipelineByID(ctx context.Context, id string) (map[string]any, error) {
	var name, description string
	var isDefault bool
	var createdAt, updatedAt any
	err := s.db.QueryRow(ctx, `
		SELECT name, COALESCE(description, ''), is_default, created_at, updated_at
		FROM commerce_order_pipelines
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, id, s.organizationID(ctx)).Scan(&name, &description, &isDefault, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "name": name, "description": description, "isDefault": isDefault, "createdAt": createdAt, "updatedAt": updatedAt}, nil
}

func (s *Server) createCommerceOrderStage(ctx context.Context, req commerceOrderStageRequest) (map[string]any, error) {
	pipelineID := strings.TrimSpace(valueOrEmpty(req.PipelineID))
	if pipelineID == "" || !s.commerceOrderPipelineBelongsToOrganization(ctx, pipelineID) {
		return nil, errors.New("pipeline not found")
	}
	name := strings.TrimSpace(valueOrEmpty(req.Name))
	if name == "" {
		return nil, errors.New("stage name is required")
	}
	position := 0
	if req.Position != nil {
		position = *req.Position
	}
	if position <= 0 {
		_ = s.db.QueryRow(ctx, `SELECT COALESCE(MAX(position), 0) + 10 FROM commerce_order_stages WHERE organization_id = $1 AND pipeline_id = NULLIF($2, '')::uuid`, s.organizationID(ctx), pipelineID).Scan(&position)
	}
	stageType := normalizeCommerceOrderStageType(valueOrEmpty(req.StageType))
	color := normalizeStageColor(valueOrDefault(req.Color, "#2196F3"))
	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO commerce_order_stages (organization_id, pipeline_id, name, position, stage_type, color)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6)
		RETURNING id::text
	`, s.organizationID(ctx), pipelineID, name, position, stageType, color).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.commerceOrderStageByID(ctx, id)
}

func (s *Server) updateCommerceOrderStage(ctx context.Context, id string, req commerceOrderStageRequest) (map[string]any, error) {
	current, err := s.commerceOrderStageByID(ctx, id)
	if err != nil {
		return nil, errors.New("stage not found")
	}
	name := strings.TrimSpace(stringFromMap(current, "name"))
	position := intFromMap(current, "position")
	stageType := strings.TrimSpace(stringFromMap(current, "type"))
	color := strings.TrimSpace(stringFromMap(current, "color"))
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.Position != nil {
		position = *req.Position
	}
	if req.StageType != nil {
		stageType = normalizeCommerceOrderStageType(*req.StageType)
	}
	if req.Color != nil {
		color = normalizeStageColor(*req.Color)
	}
	if name == "" || position <= 0 {
		return nil, errors.New("invalid stage")
	}
	if _, err := s.db.Exec(ctx, `
		UPDATE commerce_order_stages
		SET name = $2, position = $3, stage_type = $4, color = $5, updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $6
	`, id, name, position, stageType, color, s.organizationID(ctx)); err != nil {
		return nil, err
	}
	return s.commerceOrderStageByID(ctx, id)
}

func (s *Server) deleteCommerceOrderStage(ctx context.Context, id string) error {
	current, err := s.commerceOrderStageByID(ctx, id)
	if err != nil {
		return pgx.ErrNoRows
	}
	pipelineID := stringFromMap(current, "pipelineId")
	var stageCount int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM commerce_order_stages
		WHERE organization_id = $1 AND pipeline_id = NULLIF($2, '')::uuid
	`, s.organizationID(ctx), pipelineID).Scan(&stageCount); err != nil {
		return err
	}
	if stageCount <= 1 {
		return errors.New("minimal harus ada satu stage pesanan")
	}
	var fallbackStageID string
	if err := s.db.QueryRow(ctx, `
		SELECT id::text
		FROM commerce_order_stages
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
		UPDATE commerce_order_drafts
		SET stage_id = NULLIF($3, '')::uuid, updated_at = updated_at
		WHERE organization_id = $1 AND stage_id = NULLIF($2, '')::uuid
	`, s.organizationID(ctx), id, fallbackStageID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		DELETE FROM commerce_order_stages
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, id, s.organizationID(ctx))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if err := s.insertAuditLogTx(ctx, tx, "commerce.order_stage_delete", "commerce_order_stage", id, map[string]any{"pipelineId": pipelineID, "fallbackStageId": fallbackStageID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) commerceOrderStageByID(ctx context.Context, id string) (map[string]any, error) {
	var pipelineID, name, stageType, color string
	var position int
	var createdAt, updatedAt any
	err := s.db.QueryRow(ctx, `
		SELECT pipeline_id::text, name, position, stage_type, color, created_at, updated_at
		FROM commerce_order_stages
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, id, s.organizationID(ctx)).Scan(&pipelineID, &name, &position, &stageType, &color, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "pipelineId": pipelineID, "name": name, "position": position, "type": stageType, "color": color, "createdAt": createdAt, "updatedAt": updatedAt}, nil
}

func (s *Server) defaultCommerceOrderStage(ctx context.Context) (string, string, error) {
	pipelineID, err := s.ensureDefaultCommerceOrderPipeline(ctx)
	if err != nil {
		return "", "", err
	}
	var id, stageType string
	err = s.db.QueryRow(ctx, `
		SELECT id::text, stage_type
		FROM commerce_order_stages
		WHERE organization_id = $1 AND pipeline_id = NULLIF($2, '')::uuid
		ORDER BY CASE WHEN stage_type = 'open' THEN 0 ELSE 1 END, position ASC
		LIMIT 1
	`, s.organizationID(ctx), pipelineID).Scan(&id, &stageType)
	return id, stageType, err
}

func (s *Server) commerceOrderPipelineBelongsToOrganization(ctx context.Context, pipelineID string) bool {
	var exists bool
	_ = s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM commerce_order_pipelines WHERE id = NULLIF($1, '')::uuid AND organization_id = $2)`, pipelineID, s.organizationID(ctx)).Scan(&exists)
	return exists
}

func (s *Server) commerceOrderStageBelongsToOrganization(ctx context.Context, stageID string) bool {
	var exists bool
	_ = s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM commerce_order_stages WHERE id = NULLIF($1, '')::uuid AND organization_id = $2)`, stageID, s.organizationID(ctx)).Scan(&exists)
	return exists
}

func (s *Server) insertCommerceStockMovement(ctx context.Context, productID, orderDraftID, movementType string, stockDelta, reservedDelta, stockBefore, stockAfter, reservedBefore, reservedAfter int, notes string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO commerce_stock_movements (
		  organization_id, product_id, order_draft_id, movement_type, stock_delta, reserved_delta,
		  stock_quantity_before, stock_quantity_after, reserved_quantity_before, reserved_quantity_after,
		  notes, created_by
		)
		VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10, NULLIF($11, ''), NULLIF($12, '')::uuid)
	`, s.organizationID(ctx), productID, orderDraftID, movementType, stockDelta, reservedDelta, stockBefore, stockAfter, reservedBefore, reservedAfter, notes, s.agentID(ctx))
	return err
}

func (s *Server) insertCommerceStockMovementTx(ctx context.Context, tx pgx.Tx, productID, orderDraftID, movementType string, stockDelta, reservedDelta, stockBefore, stockAfter, reservedBefore, reservedAfter int, notes string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO commerce_stock_movements (
		  organization_id, product_id, order_draft_id, movement_type, stock_delta, reserved_delta,
		  stock_quantity_before, stock_quantity_after, reserved_quantity_before, reserved_quantity_after,
		  notes, created_by
		)
		VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10, NULLIF($11, ''), NULLIF($12, '')::uuid)
	`, s.organizationID(ctx), productID, orderDraftID, movementType, stockDelta, reservedDelta, stockBefore, stockAfter, reservedBefore, reservedAfter, notes, s.agentID(ctx))
	return err
}

func commerceFulfillmentOptions() []map[string]any {
	return []map[string]any{
		{"type": "pickup", "label": "Ambil di tempat", "requiresAddress": false},
		{"type": "delivery", "label": "Diantar kurir lokal", "requiresAddress": true},
		{"type": "shipping", "label": "Dikirim ekspedisi", "requiresAddress": true},
		{"type": "digital", "label": "Online / digital", "requiresAddress": false},
		{"type": "onsite_service", "label": "Layanan ke alamat pelanggan", "requiresAddress": true},
	}
}

func defaultCommerceFulfillmentTypes() []string {
	return []string{"pickup", "delivery", "shipping", "digital", "onsite_service"}
}

func normalizeCommerceFulfillmentTypes(values []string) []string {
	allowed := map[string]bool{}
	for _, option := range commerceFulfillmentOptions() {
		allowed[option["type"].(string)] = true
	}
	seen := map[string]bool{}
	items := []string{}
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if !allowed[normalized] || seen[normalized] {
			continue
		}
		seen[normalized] = true
		items = append(items, normalized)
	}
	return items
}

func defaultCommerceFulfillmentType(enabledTypes []string) string {
	normalized := normalizeCommerceFulfillmentTypes(enabledTypes)
	if len(normalized) == 0 {
		return "pickup"
	}
	return normalized[0]
}

func commerceFulfillmentTypeEnabled(enabledTypes []string, fulfillmentType string) bool {
	normalized := normalizeCommerceFulfillmentType(fulfillmentType)
	for _, item := range normalizeCommerceFulfillmentTypes(enabledTypes) {
		if item == normalized {
			return true
		}
	}
	return false
}

func commerceOrderSettingsPayload(enabledTypes []string) map[string]any {
	normalized := normalizeCommerceFulfillmentTypes(enabledTypes)
	if len(normalized) == 0 {
		normalized = defaultCommerceFulfillmentTypes()
	}
	return map[string]any{"enabledFulfillmentTypes": normalized}
}

func (s *Server) enabledCommerceFulfillmentTypes(ctx context.Context) ([]string, error) {
	settings, err := s.commerceOrderSettings(ctx)
	if err != nil {
		return nil, err
	}
	if values, ok := settings["enabledFulfillmentTypes"].([]string); ok && len(values) > 0 {
		return values, nil
	}
	return defaultCommerceFulfillmentTypes(), nil
}

func (s *Server) commerceOrderSettings(ctx context.Context) (map[string]any, error) {
	var raw string
	err := s.db.QueryRow(ctx, `
		SELECT array_to_string(enabled_fulfillment_types, ',')
		FROM commerce_order_settings
		WHERE organization_id = $1
	`, s.organizationID(ctx)).Scan(&raw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return commerceOrderSettingsPayload(defaultCommerceFulfillmentTypes()), nil
		}
		return nil, err
	}
	values := []string{}
	for _, item := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return commerceOrderSettingsPayload(values), nil
}

func (s *Server) updateCommerceOrderSettings(ctx context.Context, req commerceOrderSettingsRequest) (map[string]any, error) {
	enabledTypes := normalizeCommerceFulfillmentTypes(req.EnabledFulfillmentTypes)
	if len(enabledTypes) == 0 {
		return nil, errors.New("pilih minimal satu cara pelanggan menerima pesanan")
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO commerce_order_settings (organization_id, enabled_fulfillment_types, updated_by, updated_at)
		VALUES ($1, $2, NULLIF($3, '')::uuid, NOW())
		ON CONFLICT (organization_id) DO UPDATE
		SET enabled_fulfillment_types = EXCLUDED.enabled_fulfillment_types,
		    updated_by = EXCLUDED.updated_by,
		    updated_at = EXCLUDED.updated_at
	`, s.organizationID(ctx), enabledTypes, s.agentID(ctx))
	if err != nil {
		return nil, err
	}
	return commerceOrderSettingsPayload(enabledTypes), nil
}

func normalizeCommerceFulfillmentType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "delivery", "shipping", "digital", "onsite_service":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "pickup"
	}
}

func validateCommerceFulfillment(fulfillmentType, addressLine string) error {
	switch fulfillmentType {
	case "delivery", "shipping", "onsite_service":
		if strings.TrimSpace(addressLine) == "" {
			return errors.New("alamat lengkap wajib diisi untuk cara terima pesanan ini")
		}
	}
	return nil
}

func normalizeCommerceOrderStageType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "completed", "cancelled":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "open"
	}
}

func normalizeStageColor(value string) string {
	color := strings.TrimSpace(value)
	if strings.HasPrefix(color, "#") && (len(color) == 7 || len(color) == 4) {
		return color
	}
	return "#2196F3"
}

func normalizeCommerceProductStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "inactive", "archived":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "active"
	}
}

func normalizeCommerceOrderStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "awaiting_confirmation", "confirmed", "cancelled":
		return strings.ToLower(strings.TrimSpace(value))
	case "canceled":
		return "cancelled"
	default:
		return "draft"
	}
}

func floatFromMap(item map[string]any, key string) float64 {
	if item == nil {
		return 0
	}
	switch value := item[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		parsed, _ := strconv.ParseFloat(value.String(), 64)
		return parsed
	default:
		return 0
	}
}

func stringFromAny(value any) string {
	switch parsed := value.(type) {
	case string:
		return parsed
	case json.Number:
		return parsed.String()
	case int:
		return strconv.Itoa(parsed)
	case int64:
		return strconv.FormatInt(parsed, 10)
	case float64:
		return strconv.FormatFloat(parsed, 'f', -1, 64)
	default:
		return ""
	}
}

func (s *Server) executeBusinessTool(ctx context.Context, toolName string, args map[string]any) (map[string]any, error) {
	switch strings.TrimSpace(toolName) {
	case "check_stock":
		enabled, err := s.businessModuleEnabled(ctx, "commerce")
		if err != nil {
			return nil, err
		}
		if !enabled {
			return nil, errors.New("commerce tool is not active")
		}
		query := strings.TrimSpace(stringFromAny(args["query"]))
		items, err := s.listCommerceProducts(ctx, query, "active")
		if err != nil {
			return nil, err
		}
		return map[string]any{"items": items, "query": query}, nil
	case "create_order_draft":
		enabled, err := s.businessModuleEnabled(ctx, "commerce")
		if err != nil {
			return nil, err
		}
		if !enabled {
			return nil, errors.New("commerce tool is not active")
		}
		return nil, errors.New("create_order_draft tool requires typed request binding")
	default:
		return nil, errors.New("unknown business tool")
	}
}
