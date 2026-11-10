package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendPersonCredentialResetToken(t *testing.T) {
	tests := []struct {
		name     string
		ttl      int64
		email    string
		wantBody string
	}{
		{name: "defaults", wantBody: `{"ttl":null,"email":null}`},
		{name: "ttl and email", ttl: 86400, email: "alt@example.com", wantBody: `{"ttl":86400,"email":"alt@example.com"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath, gotBody string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.EscapedPath()
				var raw json.RawMessage
				_ = json.NewDecoder(r.Body).Decode(&raw)
				gotBody = string(raw)
				_, _ = w.Write([]byte(`"7f9c1c1e-6f5a-4c39-9b55-1b2a7f2a0c11"`))
			}))
			t.Cleanup(srv.Close)

			id, err := NewClient(srv.URL, "token").SendPersonCredentialResetToken(context.Background(), "alice@example", tt.ttl, tt.email)
			if err != nil {
				t.Fatal(err)
			}
			if gotMethod != http.MethodPost {
				t.Errorf("method = %s", gotMethod)
			}
			if want := "/v1/person/alice@example/_credential/_update_intent_send"; gotPath != want {
				t.Errorf("path = %s, want %s", gotPath, want)
			}
			if gotBody != tt.wantBody {
				t.Errorf("body = %s, want %s", gotBody, tt.wantBody)
			}
			if id != "7f9c1c1e-6f5a-4c39-9b55-1b2a7f2a0c11" {
				t.Errorf("message id = %q", id)
			}
		})
	}
}

func TestSendPersonCredentialResetTokenError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `"NoMatchingEntries"`, http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	if _, err := NewClient(srv.URL, "token").SendPersonCredentialResetToken(context.Background(), "alice", 0, ""); err == nil {
		t.Fatal("expected an error")
	}
}
