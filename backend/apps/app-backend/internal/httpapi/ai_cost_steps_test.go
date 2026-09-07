package httpapi

import "testing"

func TestNormalizeAIRunCostStepLabels(t *testing.T) {
	tests := []struct {
		name     string
		stepType string
		stepName string
		wantType string
		wantName string
	}{
		{
			name:     "intent analyzer",
			stepType: "model_call",
			stepName: "openrouter_intent_analyzer",
			wantType: "intent_analyzer",
			wantName: "intent analyzer",
		},
		{
			name:     "knowledge retrieval",
			stepType: "model_call",
			stepName: "openrouter_knowledge_organizer",
			wantType: "knowledge_retrieval",
			wantName: "knowledge retrieval",
		},
		{
			name:     "answer generation",
			stepType: "model_call",
			stepName: "openrouter_single_agent_rag",
			wantType: "answer_generation",
			wantName: "answer generation",
		},
		{
			name:     "future tool call",
			stepType: "tool",
			stepName: "check_stock_tool",
			wantType: "tool_call",
			wantName: "tool call",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotName := normalizeAIRunCostStepLabels(tt.stepType, tt.stepName)
			if gotType != tt.wantType || gotName != tt.wantName {
				t.Fatalf("normalizeAIRunCostStepLabels() = (%q, %q), want (%q, %q)", gotType, gotName, tt.wantType, tt.wantName)
			}
		})
	}
}
