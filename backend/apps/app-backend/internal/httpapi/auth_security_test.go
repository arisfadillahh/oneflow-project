package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"oneflow/app-backend/internal/config"
)

func TestAuthFailureTrackerLocksAndClears(t *testing.T) {
	tracker := newAuthFailureTracker(5, time.Minute)
	key := "auth:login:127.0.0.1:user"

	for i := 0; i < 4; i++ {
		if _, locked := tracker.RecordFailure(key); locked {
			t.Fatalf("attempt %d locked too early", i+1)
		}
	}
	if _, locked := tracker.RecordFailure(key); !locked {
		t.Fatal("fifth failed attempt did not lock")
	}
	if _, locked := tracker.Locked(key); !locked {
		t.Fatal("tracker did not report locked key")
	}

	tracker.Clear(key)
	if _, locked := tracker.Locked(key); locked {
		t.Fatal("cleared key is still locked")
	}
}

func TestAuthFailureTrackerExpiresLock(t *testing.T) {
	tracker := newAuthFailureTracker(1, time.Millisecond)
	key := "auth:register:127.0.0.1:-"

	if _, locked := tracker.RecordFailure(key); !locked {
		t.Fatal("first failure should lock when maxFailures is one")
	}
	time.Sleep(2 * time.Millisecond)
	if _, locked := tracker.Locked(key); locked {
		t.Fatal("lock did not expire")
	}
}

func TestAuthRateLimitReturnsRetryAfter(t *testing.T) {
	server := New(config.Config{}, nil, nil)
	handler := server.Routes()

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d hit rate limit too early", i+1)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After header")
	}
}

func TestClientIPPrefersCloudflareConnectingIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.20")
	req.Header.Set("CF-Connecting-IP", "203.0.113.30")

	if got := clientIP(req); got != "203.0.113.30" {
		t.Fatalf("clientIP = %q, want CF-Connecting-IP", got)
	}
}
