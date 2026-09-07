package config

import "testing"

func TestMetaWebhookDefaultsToCloudFeatureState(t *testing.T) {
	t.Setenv("META_CLOUD_ENABLED", "true")
	t.Setenv("META_WEBHOOK_ENABLED", "")

	cfg := Load()
	if !cfg.MetaWebhookEnabled {
		t.Fatal("expected webhook to follow enabled Meta Cloud provider when no webhook override is set")
	}
}

func TestMetaWebhookCanRunWithoutCloudOnboarding(t *testing.T) {
	t.Setenv("META_CLOUD_ENABLED", "false")
	t.Setenv("META_WEBHOOK_ENABLED", "true")

	cfg := Load()
	if cfg.MetaCloudEnabled || !cfg.MetaWebhookEnabled {
		t.Fatal("expected receiver-only mode with onboarding disabled")
	}
}
