package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	AppEnv                 string
	Port                   string
	PostgresDSN            string
	EscalationGroupID      string
	Mode                   string
	AllowedOrigins         []string
	AppBackendBaseURL      string
	InternalGatewayToken   string
	SessionDialect         string
	SessionDSN             string
	MetaCloudEnabled       bool
	MetaWebhookEnabled     bool
	MetaAppSecret          string
	MetaWebhookVerifyToken string
	MetaGraphAPIVersion    string
	MetaGraphAPIBaseURL    string
	MetaCredentialKey      string
}

func Load() Config {
	return Config{
		AppEnv:                 getenv("APP_ENV", getenv("WA_ENV", "development")),
		Port:                   getenv("WA_PORT", "8090"),
		PostgresDSN:            getenv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/oneflow?sslmode=disable"),
		EscalationGroupID:      getenv("ESCALATION_GROUP_ID", ""),
		Mode:                   getenv("WA_MODE", "mock"),
		AllowedOrigins:         splitCSV(getenv("WA_ALLOWED_ORIGINS", "http://localhost:3000")),
		AppBackendBaseURL:      getenv("APP_BACKEND_BASE_URL", "http://app-backend:8080"),
		InternalGatewayToken:   getenv("INTERNAL_GATEWAY_TOKEN", "local-internal-gateway-token"),
		SessionDialect:         getenv("WA_SESSION_DIALECT", "postgres"),
		SessionDSN:             getenv("WA_SESSION_DSN", getenv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/oneflow?sslmode=disable")),
		MetaCloudEnabled:       strings.EqualFold(getenv("META_CLOUD_ENABLED", "false"), "true"),
		MetaWebhookEnabled:     strings.EqualFold(getenv("META_WEBHOOK_ENABLED", getenv("META_CLOUD_ENABLED", "false")), "true"),
		MetaAppSecret:          strings.TrimSpace(getenv("META_APP_SECRET", "")),
		MetaWebhookVerifyToken: strings.TrimSpace(getenv("META_WEBHOOK_VERIFY_TOKEN", "")),
		MetaGraphAPIVersion:    strings.TrimSpace(getenv("META_GRAPH_API_VERSION", "")),
		MetaGraphAPIBaseURL:    strings.TrimRight(getenv("META_GRAPH_API_BASE_URL", "https://graph.facebook.com"), "/"),
		MetaCredentialKey:      strings.TrimSpace(getenv("META_CREDENTIAL_ENCRYPTION_KEY", "")),
	}
}

func (c Config) Validate() error {
	if c.MetaWebhookEnabled && (c.MetaAppSecret == "" || c.MetaWebhookVerifyToken == "") {
		return fmt.Errorf("Meta webhook requires META_APP_SECRET and META_WEBHOOK_VERIFY_TOKEN")
	}
	if c.MetaCloudEnabled {
		key, err := base64.StdEncoding.DecodeString(c.MetaCredentialKey)
		if c.MetaAppSecret == "" || c.MetaWebhookVerifyToken == "" || c.MetaGraphAPIVersion == "" {
			return fmt.Errorf("Meta Cloud API requires META_APP_SECRET, META_WEBHOOK_VERIFY_TOKEN, and META_GRAPH_API_VERSION")
		}
		if err != nil || len(key) != 32 {
			return fmt.Errorf("META_CREDENTIAL_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
		}
	}
	if !strings.EqualFold(c.AppEnv, "production") {
		return nil
	}
	if len(c.InternalGatewayToken) < 32 || c.InternalGatewayToken == "local-internal-gateway-token" {
		return fmt.Errorf("INTERNAL_GATEWAY_TOKEN must be at least 32 characters and non-default in production")
	}
	if c.PostgresDSN == "" || strings.Contains(c.PostgresDSN, "postgres:postgres@") {
		return fmt.Errorf("POSTGRES_DSN must use non-default credentials in production")
	}
	for _, origin := range c.AllowedOrigins {
		if strings.TrimSpace(origin) == "*" {
			return fmt.Errorf("WA_ALLOWED_ORIGINS must be strict in production")
		}
	}
	return nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
