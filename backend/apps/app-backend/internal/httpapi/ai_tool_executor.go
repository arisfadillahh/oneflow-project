package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

type aiToolExecutionRequest struct {
	ToolName       string
	Args           map[string]any
	IdempotencyKey string
	AIRunID        string
}

func normalizeAIToolName(toolName string) string {
	return strings.ToLower(strings.TrimSpace(toolName))
}

func aiToolModuleKey(toolName string) (string, bool) {
	switch normalizeAIToolName(toolName) {
	case "check_stock":
		return "commerce", true
	case "create_order_draft":
		return "commerce", true
	default:
		return "", false
	}
}

func canExecuteAITool(role, toolName string) bool {
	switch normalizeAIToolName(toolName) {
	case "check_stock":
		return role == "owner" || isOpsRole(role)
	case "create_order_draft":
		return role == "owner" || isOpsRole(role)
	default:
		return false
	}
}

func (s *Server) executeAITool(ctx context.Context, req aiToolExecutionRequest) (map[string]any, error) {
	toolName := normalizeAIToolName(req.ToolName)
	if toolName == "" {
		return nil, errors.New("tool name is required")
	}
	moduleKey, ok := aiToolModuleKey(toolName)
	if !ok {
		return nil, errors.New("unknown AI tool")
	}
	if !canExecuteAITool(s.role(ctx), toolName) {
		return nil, errors.New("permission denied for AI tool")
	}
	minimumMode := "read"
	if toolName == "create_order_draft" {
		minimumMode = "draft"
	}
	enabled, err := s.businessModuleAIAllowed(ctx, moduleKey, minimumMode)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, errors.New(moduleKey + " AI tool mode is not active")
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		if replay, found, err := s.lookupAIToolExecution(ctx, toolName, idempotencyKey); err != nil || found {
			return replay, err
		}
		if err := s.insertAIToolExecutionStarted(ctx, toolName, idempotencyKey, req); err != nil {
			return nil, err
		}
	}

	result, runErr := s.runAITool(ctx, toolName, req.Args)
	if idempotencyKey != "" {
		if recordErr := s.recordAIToolExecutionFinished(ctx, toolName, idempotencyKey, result, runErr); recordErr != nil && runErr == nil {
			return nil, recordErr
		}
	}
	if runErr != nil {
		return nil, runErr
	}
	if err := s.insertAuditLog(ctx, "ai_tool.execute", "ai_tool_execution", "", map[string]any{
		"toolName":       toolName,
		"moduleKey":      moduleKey,
		"idempotencyKey": idempotencyKey,
		"aiRunId":        strings.TrimSpace(req.AIRunID),
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Server) runAITool(ctx context.Context, toolName string, args map[string]any) (map[string]any, error) {
	switch normalizeAIToolName(toolName) {
	case "check_stock":
		query := strings.TrimSpace(stringFromAny(args["query"]))
		items, err := s.listCommerceProducts(ctx, query, "active")
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"toolName": "check_stock",
			"query":    query,
			"items":    items,
		}, nil
	case "create_order_draft":
		draftReq, err := commerceOrderDraftRequestFromToolArgs(args)
		if err != nil {
			return nil, err
		}
		source := "ai_tool"
		status := "draft"
		draftReq.Source = &source
		draftReq.Status = &status
		item, err := s.createCommerceOrderDraft(ctx, draftReq)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"toolName":   "create_order_draft",
			"orderDraft": item,
		}, nil
	default:
		return nil, errors.New("unknown AI tool")
	}
}

func commerceOrderDraftRequestFromToolArgs(args map[string]any) (commerceOrderDraftRequest, error) {
	if args == nil {
		args = map[string]any{}
	}
	req := commerceOrderDraftRequest{
		CustomerName:  optionalStringPointer(stringFromAny(firstNonNil(args["customerName"], args["customer_name"]))),
		CustomerPhone: optionalStringPointer(stringFromAny(firstNonNil(args["customerPhone"], args["customer_phone"]))),
		ContactID:     optionalStringPointer(stringFromAny(firstNonNil(args["contactId"], args["contact_id"]))),
		ConversationID: optionalStringPointer(
			stringFromAny(firstNonNil(args["conversationId"], args["conversation_id"])),
		),
		Notes: optionalStringPointer(stringFromAny(args["notes"])),
	}

	rawItems, ok := args["items"].([]any)
	if !ok || len(rawItems) == 0 {
		return req, errors.New("items are required")
	}
	for _, rawItem := range rawItems {
		itemMap, ok := rawItem.(map[string]any)
		if !ok {
			return req, errors.New("invalid order item")
		}
		quantity, ok := jsonNumberToInt(firstNonNil(itemMap["quantity"], itemMap["qty"]))
		if !ok || quantity <= 0 {
			return req, errors.New("item quantity must be greater than zero")
		}
		item := commerceOrderItemRequest{
			ProductID: optionalStringPointer(
				stringFromAny(firstNonNil(itemMap["productId"], itemMap["product_id"])),
			),
			Quantity: quantity,
		}
		if unitPrice, ok := jsonNumberToFloat(firstNonNil(itemMap["unitPrice"], itemMap["unit_price"])); ok {
			item.UnitPrice = &unitPrice
		}
		req.Items = append(req.Items, item)
	}
	return req, nil
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func optionalStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (s *Server) lookupAIToolExecution(ctx context.Context, toolName, idempotencyKey string) (map[string]any, bool, error) {
	var status, responseText, errorText string
	err := s.db.QueryRow(ctx, `
		SELECT status, COALESCE(response_payload, '{}'::jsonb)::text, COALESCE(error_message, '')
		FROM ai_tool_executions
		WHERE organization_id = $1 AND tool_name = $2 AND idempotency_key = $3
	`, s.organizationID(ctx), toolName, idempotencyKey).Scan(&status, &responseText, &errorText)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if status != "succeeded" {
		if errorText == "" {
			errorText = "AI tool execution already exists with status " + status
		}
		return nil, true, errors.New(errorText)
	}
	var response map[string]any
	if err := json.Unmarshal([]byte(responseText), &response); err != nil {
		return nil, true, err
	}
	response["idempotentReplay"] = true
	return response, true, nil
}

func (s *Server) insertAIToolExecutionStarted(ctx context.Context, toolName, idempotencyKey string, req aiToolExecutionRequest) error {
	requestPayload, err := json.Marshal(req.Args)
	if err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `
		INSERT INTO ai_tool_executions (
		  organization_id, ai_run_id, tool_name, idempotency_key, status, request_payload, created_by
		)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, 'running', $5::jsonb, NULLIF($6, '')::uuid)
		ON CONFLICT (organization_id, tool_name, idempotency_key) WHERE idempotency_key <> '' DO NOTHING
	`, s.organizationID(ctx), strings.TrimSpace(req.AIRunID), toolName, idempotencyKey, string(requestPayload), s.agentID(ctx))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		_, _, err := s.lookupAIToolExecution(ctx, toolName, idempotencyKey)
		if err != nil {
			return err
		}
		return errors.New("AI tool execution already exists")
	}
	return nil
}

func (s *Server) recordAIToolExecutionFinished(ctx context.Context, toolName, idempotencyKey string, result map[string]any, runErr error) error {
	status := "succeeded"
	errorMessage := ""
	if runErr != nil {
		status = "failed"
		errorMessage = runErr.Error()
	}
	responsePayload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		UPDATE ai_tool_executions
		SET status = $4,
		    response_payload = $5::jsonb,
		    error_message = NULLIF($6, ''),
		    finished_at = NOW(),
		    updated_at = NOW()
		WHERE organization_id = $1 AND tool_name = $2 AND idempotency_key = $3
	`, s.organizationID(ctx), toolName, idempotencyKey, status, string(responsePayload), errorMessage)
	return err
}
