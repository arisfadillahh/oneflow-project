package httpapi

import "testing"

func TestTrialPlanLimitsDoNotIncludeWhatsApp(t *testing.T) {
	limits := trialPlanLimits()

	if limits.PlanKey != "trial" {
		t.Fatalf("trial plan key = %q, want trial", limits.PlanKey)
	}
	if limits.MaxWhatsAppSessions != 0 {
		t.Fatalf("trial max whatsapp sessions = %d, want 0", limits.MaxWhatsAppSessions)
	}
	if limits.MaxAIAgents != 1 {
		t.Fatalf("trial max ai agents = %d, want 1", limits.MaxAIAgents)
	}
	if limits.MaxHumanUsers != 1 {
		t.Fatalf("trial max human users = %d, want 1", limits.MaxHumanUsers)
	}
}
