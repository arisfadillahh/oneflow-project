package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"oneflow/app-backend/internal/config"
)

func TestSubscribeMetaWABARequiresConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		body      string
		wantError bool
	}{
		{"confirmed", 200, `{"success":true}`, false},
		{"rejected", 200, `{"success":false}`, true},
		{"missing confirmation", 200, `{}`, true},
		{"malformed", 200, `invalid`, true},
		{"permission denied", 403, `{"error":{"message":"denied"}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/v23.0/123/subscribed_apps" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing bearer token")
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer upstream.Close()
			s := &Server{cfg: config.Config{MetaGraphAPIBaseURL: upstream.URL, MetaGraphAPIVersion: "v23.0"}, httpClient: upstream.Client()}
			err := s.subscribeMetaWABA(context.Background(), "123", "test-token")
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, want error %v", err, tc.wantError)
			}
		})
	}
}

func TestVerifyMetaPhoneOwnership(t *testing.T) {
	for _, tc := range []struct {
		name      string
		bodies    []string
		status    int
		wantError bool
	}{
		{"matched", []string{`{"data":[{"id":"456"}]}`}, 200, false},
		{"wrong WABA", []string{`{"data":[{"id":"999"}]}`}, 200, true},
		{"denied", []string{`{"error":{}}`}, 403, true},
		{"malformed", []string{`invalid`}, 200, true},
		{"later page", []string{`{"data":[{"id":"999"}],"paging":{"next":"https://untrusted.invalid/next","cursors":{"after":"cursor1"}}}`, `{"data":[{"id":"456"}]}`}, 200, false},
		{"invalid pagination", []string{`{"data":[],"paging":{"next":"https://untrusted.invalid/next"}}`}, 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v23.0/123/phone_numbers" || r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("unexpected ownership request")
				}
				if calls > 0 && r.URL.Query().Get("after") != "cursor1" {
					t.Error("missing pagination cursor")
				}
				index := calls
				calls++
				if index >= len(tc.bodies) {
					t.Error("unexpected extra request")
					w.WriteHeader(500)
					return
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.bodies[index]))
			}))
			defer upstream.Close()
			s := &Server{cfg: config.Config{MetaGraphAPIBaseURL: upstream.URL, MetaGraphAPIVersion: "v23.0"}, httpClient: upstream.Client()}
			err := s.verifyMetaPhoneOwnership(context.Background(), "123", "456", "test-token")
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, want error %v", err, tc.wantError)
			}
		})
	}
}
