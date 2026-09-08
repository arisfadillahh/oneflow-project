package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidInstagramSignature(t *testing.T) {
	body := []byte(`{"object":"instagram"}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !validInstagramSignature(body, signature, "secret") {
		t.Fatal("expected a valid signature")
	}
	if validInstagramSignature([]byte("changed"), signature, "secret") {
		t.Fatal("accepted a signature for a different payload")
	}
	if validInstagramSignature(body, "sha1=invalid", "secret") {
		t.Fatal("accepted an unsupported signature format")
	}
}

func TestInstagramRecipientID(t *testing.T) {
	tests := []struct {
		name      string
		identity  string
		sessionID string
		want      string
		ok        bool
	}{
		{name: "valid", identity: "ig:session-1:178414000", sessionID: "session-1", want: "178414000", ok: true},
		{name: "wrong session", identity: "ig:session-2:178414000", sessionID: "session-1"},
		{name: "missing recipient", identity: "ig:session-1:", sessionID: "session-1"},
		{name: "WhatsApp identity", identity: "628123456789", sessionID: "session-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := instagramRecipientID(tt.identity, tt.sessionID)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("instagramRecipientID() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestCanManageInstagram(t *testing.T) {
	for _, role := range []string{"admin", "super_admin"} {
		if !canManageInstagram(role) {
			t.Fatalf("expected %s to manage Instagram", role)
		}
	}
	for _, role := range []string{"agent", "operator", "owner", ""} {
		if canManageInstagram(role) {
			t.Fatalf("did not expect %s to manage Instagram", role)
		}
	}
}

func TestCheckInstagramGraphEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK},
		{name: "Meta rejection", statusCode: http.StatusForbidden, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"data":[]}`))
			}))
			defer server.Close()
			s := &Server{httpClient: server.Client()}
			err := s.checkInstagramGraphEndpoint(context.Background(), server.URL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkInstagramGraphEndpoint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
