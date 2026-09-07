package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"oneflow/wa-gateway/internal/config"
	"testing"
)

func TestLegacyTransportIsUnavailable(t *testing.T) {
	s := New(config.Config{Mode: "mock"}, nil)
	if _, ok := s.transport.(unavailableTransport); !ok {
		t.Fatal("legacy transport must never initialize, including mock fallback")
	}
	if _, err := s.transport.Connect(context.Background()); err == nil {
		t.Fatal("legacy connect must fail closed")
	}
	if _, err := s.transport.SendText(context.Background(), "dummy", "QA"); err == nil {
		t.Fatal("legacy sending must fail closed")
	}
}

func TestLegacyPairingEndpointsAreGone(t *testing.T) {
	s := New(config.Config{Mode: "real"}, nil)
	for _, path := range []string{"/api/qr", "/api/qr-image", "/api/connect", "/api/pair-phone"} {
		response := httptest.NewRecorder()
		s.Routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusGone {
			t.Fatalf("%s returned %d, want 410", path, response.Code)
		}
	}
}
