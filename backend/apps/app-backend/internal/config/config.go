package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	AppEnv                      string
	AppPort                     string
	PostgresDSN                 string
	JWTSecret                   string
	WebsocketAllowedOrigins     []string
	WAGatewayBaseURL            string
	EscalationGroupID           string
	InternalGatewayToken        string
	AIServiceBaseURL            string
	OpenRouterModelsURL         string
	SupportReportPhone          string
	AppPublicURL                string
	MidtransEnv                 string
	MidtransMerchantID          string
	MidtransClientKey           string
	MidtransServerKey           string
	ObjectStorageEndpoint       string
	ObjectStorageAccessKey      string
	ObjectStorageSecretKey      string
	ObjectStorageBucket         string
	ObjectStorageUseSSL         bool
	ObjectStoragePublicBaseURL  string
	MetaCloudEnabled            bool
	MetaAppID                   string
	MetaAppSecret               string
	MetaConfigurationID         string
	MetaGraphAPIVersion         string
	MetaGraphAPIBaseURL         string
	MetaCredentialKey           string
	MetaTestEnabled             bool
	MetaTestAccessToken         string
	MetaTestWABAID              string
	MetaTestPhoneNumberID       string
	InstagramEnabled            bool
	InstagramAppID              string
	InstagramAppSecret          string
	InstagramRedirectURI        string
	InstagramWebhookVerifyToken string
	InstagramGraphAPIBaseURL    string
}

func Load() Config {
	return Config{
		AppEnv:                      getenv("APP_ENV", "development"),
		AppPort:                     getenv("APP_PORT", "8080"),
		PostgresDSN:                 getenv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/oneflow?sslmode=disable"),
		JWTSecret:                   getenv("JWT_SECRET", "change_me"),
		WebsocketAllowedOrigins:     splitCSV(getenv("WEBSOCKET_ALLOWED_ORIGINS", "http://localhost:3000")),
		WAGatewayBaseURL:            getenv("WA_GATEWAY_BASE_URL", "http://localhost:8090"),
		EscalationGroupID:           getenv("ESCALATION_GROUP_ID", ""),
		InternalGatewayToken:        getenv("INTERNAL_GATEWAY_TOKEN", "local-internal-gateway-token"),
		AIServiceBaseURL:            getenv("AI_SERVICE_BASE_URL", "http://localhost:8000"),
		OpenRouterModelsURL:         getenv("OPENROUTER_MODELS_URL", "https://openrouter.ai/api/v1/models"),
		SupportReportPhone:          getenv("SUPPORT_REPORT_PHONE", "6281212022628"),
		AppPublicURL:                strings.TrimRight(getenv("APP_PUBLIC_URL", getenv("APP_BASE_URL", "http://localhost:3000")), "/"),
		MidtransEnv:                 strings.ToLower(getenv("MIDTRANS_ENV", "sandbox")),
		MidtransMerchantID:          getenv("MIDTRANS_MERCHANT_ID", ""),
		MidtransClientKey:           getenv("MIDTRANS_CLIENT_KEY", ""),
		MidtransServerKey:           getenv("MIDTRANS_SERVER_KEY", ""),
		ObjectStorageEndpoint:       getenv("OBJECT_STORAGE_ENDPOINT", "http://minio:9000"),
		ObjectStorageAccessKey:      getenv("OBJECT_STORAGE_ACCESS_KEY", "minioadmin"),
		ObjectStorageSecretKey:      getenv("OBJECT_STORAGE_SECRET_KEY", "minioadmin"),
		ObjectStorageBucket:         getenv("OBJECT_STORAGE_BUCKET", "oneflow"),
		ObjectStorageUseSSL:         strings.EqualFold(getenv("OBJECT_STORAGE_USE_SSL", "false"), "true"),
		ObjectStoragePublicBaseURL:  getenv("OBJECT_STORAGE_PUBLIC_BASE_URL", "http://localhost:9000"),
		MetaCloudEnabled:            strings.EqualFold(getenv("META_CLOUD_ENABLED", "false"), "true"),
		MetaAppID:                   strings.TrimSpace(getenv("META_APP_ID", "")),
		MetaAppSecret:               strings.TrimSpace(getenv("META_APP_SECRET", "")),
		MetaConfigurationID:         strings.TrimSpace(getenv("META_CONFIGURATION_ID", "")),
		MetaGraphAPIVersion:         strings.TrimSpace(getenv("META_GRAPH_API_VERSION", "")),
		MetaGraphAPIBaseURL:         strings.TrimRight(getenv("META_GRAPH_API_BASE_URL", "https://graph.facebook.com"), "/"),
		MetaCredentialKey:           strings.TrimSpace(getenv("META_CREDENTIAL_ENCRYPTION_KEY", "")),
		MetaTestEnabled:             strings.EqualFold(getenv("META_TEST_ENABLED", "false"), "true"),
		MetaTestAccessToken:         strings.TrimSpace(getenv("META_TEST_ACCESS_TOKEN", "")),
		MetaTestWABAID:              strings.TrimSpace(getenv("META_TEST_WABA_ID", "")),
		MetaTestPhoneNumberID:       strings.TrimSpace(getenv("META_TEST_PHONE_NUMBER_ID", "")),
		InstagramEnabled:            strings.EqualFold(getenv("INSTAGRAM_ENABLED", "false"), "true"),
		InstagramAppID:              strings.TrimSpace(getenv("INSTAGRAM_APP_ID", "")),
		InstagramAppSecret:          strings.TrimSpace(getenv("INSTAGRAM_APP_SECRET", "")),
		InstagramRedirectURI:        strings.TrimSpace(getenv("INSTAGRAM_REDIRECT_URI", "")),
		InstagramWebhookVerifyToken: strings.TrimSpace(getenv("INSTAGRAM_WEBHOOK_VERIFY_TOKEN", "")),
		InstagramGraphAPIBaseURL:    strings.TrimRight(getenv("INSTAGRAM_GRAPH_API_BASE_URL", "https://graph.instagram.com"), "/"),
	}
}

func (c Config) Validate() error {
	if c.MetaCloudEnabled {
		if c.MetaAppID == "" || c.MetaAppSecret == "" || c.MetaConfigurationID == "" || c.MetaGraphAPIVersion == "" {
			return fmt.Errorf("Meta Cloud API requires META_APP_ID, META_APP_SECRET, META_CONFIGURATION_ID, and META_GRAPH_API_VERSION")
		}
		if _, err := base64.StdEncoding.DecodeString(c.MetaCredentialKey); err != nil || lenMustDecodeBase64(c.MetaCredentialKey) != 32 {
			return fmt.Errorf("META_CREDENTIAL_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
		}
	}
	if c.InstagramEnabled {
		if c.InstagramAppID == "" || c.InstagramAppSecret == "" || c.InstagramRedirectURI == "" || c.InstagramWebhookVerifyToken == "" {
			return fmt.Errorf("Instagram API requires INSTAGRAM_APP_ID, INSTAGRAM_APP_SECRET, INSTAGRAM_REDIRECT_URI, and INSTAGRAM_WEBHOOK_VERIFY_TOKEN")
		}
		if _, err := base64.StdEncoding.DecodeString(c.MetaCredentialKey); err != nil || lenMustDecodeBase64(c.MetaCredentialKey) != 32 {
			return fmt.Errorf("META_CREDENTIAL_ENCRYPTION_KEY must be configured for Instagram credentials")
		}
	}
	if !strings.EqualFold(c.AppEnv, "production") {
		return nil
	}
	if len(c.JWTSecret) < 32 || c.JWTSecret == "change_me" {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters and non-default in production")
	}
	if len(c.InternalGatewayToken) < 32 || c.InternalGatewayToken == "local-internal-gateway-token" {
		return fmt.Errorf("INTERNAL_GATEWAY_TOKEN must be at least 32 characters and non-default in production")
	}
	if c.PostgresDSN == "" || strings.Contains(c.PostgresDSN, "postgres:postgres@") {
		return fmt.Errorf("POSTGRES_DSN must use non-default credentials in production")
	}
	if c.MidtransEnv != "sandbox" && c.MidtransEnv != "production" {
		return fmt.Errorf("MIDTRANS_ENV must be sandbox or production")
	}
	if (c.MidtransMerchantID != "" || c.MidtransClientKey != "" || c.MidtransServerKey != "") && (c.MidtransClientKey == "" || c.MidtransServerKey == "") {
		return fmt.Errorf("MIDTRANS_CLIENT_KEY and MIDTRANS_SERVER_KEY must be configured together")
	}
	if c.ObjectStorageAccessKey == "" || c.ObjectStorageAccessKey == "minioadmin" || len(c.ObjectStorageSecretKey) < 16 || c.ObjectStorageSecretKey == "minioadmin" {
		return fmt.Errorf("object storage credentials must be strong and non-default in production")
	}
	for _, origin := range c.WebsocketAllowedOrigins {
		if strings.TrimSpace(origin) == "*" {
			return fmt.Errorf("WEBSOCKET_ALLOWED_ORIGINS must be strict in production")
		}
	}
	return nil
}

func lenMustDecodeBase64(value string) int {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return 0
	}
	return len(decoded)
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
