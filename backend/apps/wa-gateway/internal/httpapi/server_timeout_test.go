package httpapi

import (
	"testing"
	"time"

	"oneflow/wa-gateway/internal/config"
)

func TestNewAllowsAppBackendAIProcessingWindow(t *testing.T) {
	server := New(config.Config{Mode: "mock"}, nil)

	if server.httpClient.Timeout < 120*time.Second {
		t.Fatalf("downstream timeout = %s, want at least 120s", server.httpClient.Timeout)
	}
}
