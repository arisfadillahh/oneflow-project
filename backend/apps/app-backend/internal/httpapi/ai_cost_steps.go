package httpapi

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
)

type aiRunCostStep struct {
	StepIndex      int
	StepType       string
	StepName       string
	Provider       string
	ModelName      string
	InputTokens    int
	OutputTokens   int
	EmbeddingToken int
	CostUSD        *float64
	Metadata       map[string]any
}

func (s *Server) insertAIRunCostSteps(ctx context.Context, aiRunID string, decision aiDecisionResponse, usageResult creditUsageResult) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := s.insertAIRunCostStepsTx(ctx, tx, aiRunID, decision, usageResult); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) insertAIRunCostStepsTx(ctx context.Context, tx pgx.Tx, aiRunID string, decision aiDecisionResponse, usageResult creditUsageResult) error {
	if strings.TrimSpace(aiRunID) == "" {
		return nil
	}
	steps := usageStepsFromDecision(decision, usageResult)
	if len(steps) == 0 {
		return nil
	}
	snapshot, err := s.activeCreditPricingTx(ctx, tx)
	if err != nil {
		return err
	}
	for index, step := range steps {
		stepIndex := step.StepIndex
		if stepIndex <= 0 {
			stepIndex = index + 1
		}
		stepType := strings.TrimSpace(step.StepType)
		if stepType == "" {
			stepType = "model_call"
		}
		stepName := strings.TrimSpace(step.StepName)
		if stepName == "" {
			stepName = stepType
		}
		stepType, stepName = normalizeAIRunCostStepLabels(stepType, stepName)
		var costIDR *float64
		if step.CostUSD != nil {
			value := *step.CostUSD * snapshot.USDToIDRRate
			costIDR = &value
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO ai_run_cost_steps (
			  organization_id,
			  ai_run_id,
			  step_index,
			  step_type,
			  step_name,
			  provider,
			  model_name,
			  input_tokens,
			  output_tokens,
			  embedding_tokens,
			  cost_usd,
			  cost_idr,
			  metadata
			)
			VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, $9, $10, $11, $12, $13::jsonb)
			ON CONFLICT (ai_run_id, step_index) DO NOTHING
		`,
			s.organizationID(ctx),
			aiRunID,
			stepIndex,
			stepType,
			stepName,
			step.Provider,
			step.ModelName,
			step.InputTokens,
			step.OutputTokens,
			step.EmbeddingToken,
			step.CostUSD,
			costIDR,
			marshalJSON(step.Metadata),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func usageStepsFromDecision(decision aiDecisionResponse, usageResult creditUsageResult) []aiRunCostStep {
	raw := decision.UsageMetadata["steps"]
	if raw == nil {
		raw = decision.UsageMetadata["usage_steps"]
	}
	steps := parseAIRunCostSteps(raw, decision.ModelName)
	if len(steps) > 0 {
		return steps
	}
	costUSD := usageResult.CostUSD
	return []aiRunCostStep{
		{
			StepIndex:      1,
			StepType:       "model_call",
			StepName:       "aggregate_ai_run",
			Provider:       "openrouter",
			ModelName:      decision.ModelName,
			InputTokens:    intFromUsageMetadata(decision.UsageMetadata, "input_tokens"),
			OutputTokens:   intFromUsageMetadata(decision.UsageMetadata, "output_tokens"),
			EmbeddingToken: intFromUsageMetadata(decision.UsageMetadata, "embedding_tokens"),
			CostUSD:        &costUSD,
			Metadata: map[string]any{
				"source":      decision.UsageMetadata["source"],
				"cost_source": decision.UsageMetadata["cost_source"],
				"fallback":    true,
			},
		},
	}
}

func parseAIRunCostSteps(raw any, fallbackModel string) []aiRunCostStep {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	steps := make([]aiRunCostStep, 0, len(items))
	for idx, item := range items {
		payload, ok := item.(map[string]any)
		if !ok {
			continue
		}
		stepIndex, _ := jsonNumberToInt(payload["step_index"])
		if stepIndex <= 0 {
			stepIndex = idx + 1
		}
		var costUSD *float64
		if value, ok := jsonNumberToFloat(payload["cost_usd"]); ok && value >= 0 {
			costUSD = &value
		}
		metadata := map[string]any{}
		if rawMetadata, ok := payload["metadata"].(map[string]any); ok {
			metadata = rawMetadata
		}
		if source := strings.TrimSpace(stringFromAny(payload["source"])); source != "" {
			metadata["source"] = source
		}
		if costSource := strings.TrimSpace(stringFromAny(payload["cost_source"])); costSource != "" {
			metadata["cost_source"] = costSource
		}
		modelName := strings.TrimSpace(stringFromAny(payload["model_name"]))
		if modelName == "" {
			modelName = fallbackModel
		}
		steps = append(steps, aiRunCostStep{
			StepIndex:      stepIndex,
			StepType:       strings.TrimSpace(stringFromAny(payload["step_type"])),
			StepName:       strings.TrimSpace(stringFromAny(payload["step_name"])),
			Provider:       strings.TrimSpace(stringFromAny(payload["provider"])),
			ModelName:      modelName,
			InputTokens:    intFromUsageStep(payload, "input_tokens"),
			OutputTokens:   intFromUsageStep(payload, "output_tokens"),
			EmbeddingToken: intFromUsageStep(payload, "embedding_tokens"),
			CostUSD:        costUSD,
			Metadata:       metadata,
		})
	}
	return steps
}

func normalizeAIRunCostStepLabels(stepType, stepName string) (string, string) {
	rawType := strings.ToLower(strings.TrimSpace(stepType))
	rawName := strings.ToLower(strings.TrimSpace(stepName))
	source := rawName
	if source == "" {
		source = rawType
	}

	switch {
	case strings.Contains(source, "intent"):
		return "intent_analyzer", "intent analyzer"
	case strings.Contains(source, "knowledge") || strings.Contains(source, "retrieval") || strings.Contains(source, "organizer"):
		return "knowledge_retrieval", "knowledge retrieval"
	case strings.Contains(source, "tool"):
		return "tool_call", "tool call"
	case strings.Contains(source, "verifier") || strings.Contains(source, "verification"):
		return "answer_verification", "answer verification"
	case strings.Contains(source, "answer") ||
		strings.Contains(source, "rag") ||
		strings.Contains(source, "revision") ||
		strings.Contains(source, "summary") ||
		strings.Contains(source, "image") ||
		rawType == "model_call":
		return "answer_generation", "answer generation"
	default:
		if rawType == "" {
			rawType = "model_call"
		}
		if rawName == "" {
			rawName = rawType
		}
		return rawType, rawName
	}
}

func intFromUsageMetadata(metadata map[string]any, key string) int {
	if metadata == nil {
		return 0
	}
	value, _ := jsonNumberToInt(metadata[key])
	return value
}

func intFromUsageStep(metadata map[string]any, key string) int {
	value, _ := jsonNumberToInt(metadata[key])
	return value
}

func marshalJSONMap(value map[string]any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
