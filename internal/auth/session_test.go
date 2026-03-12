package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionFromOAuth(t *testing.T) {
	t.Parallel()
	signer, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("NewDPoPSigner: %v", err)
	}

	tokens := &TokenSet{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		ExpiresAt:    time.Now().Add(time.Hour),
		TokenType:    "DPoP",
		Sub:          "did:plc:testuser",
	}

	session := NewSession("did:plc:testuser", "test.bsky.social", "https://pds.example.com", tokens, signer)

	if session.DID != "did:plc:testuser" {
		t.Errorf("Expected DID 'did:plc:testuser', got %q", session.DID)
	}
	if session.Handle != "test.bsky.social" {
		t.Errorf("Expected Handle 'test.bsky.social', got %q", session.Handle)
	}
	if session.PDS != "https://pds.example.com" {
		t.Errorf("Expected PDS 'https://pds.example.com', got %q", session.PDS)
	}
	if session.Tokens.AccessToken != "access-123" {
		t.Errorf("Expected AccessToken 'access-123', got %q", session.Tokens.AccessToken)
	}
	if session.DPoP == nil {
		t.Error("Expected DPoP signer to be set")
	}
}

func TestSessionProvideAuthenticatedClient(t *testing.T) {
	t.Parallel()
	var capturedAuth string
	var capturedDPoP string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedDPoP = r.Header.Get("DPoP")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"ok":true}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	signer, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("NewDPoPSigner: %v", err)
	}

	tokens := &TokenSet{
		AccessToken:  "my-access-token",
		RefreshToken: "my-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		TokenType:    "DPoP",
		Sub:          "did:plc:test",
	}

	session := NewSession("did:plc:test", "test.bsky.social", server.URL, tokens, signer)
	tm := NewTokenManager(server.URL+"/token", "test-client", tokens, signer)
	session.TokenManager = tm

	client := session.AuthenticatedHTTPClient()
	client.Transport.(*dpopTransport).base = server.Client().Transport

	resp, err := client.Get(server.URL + "/xrpc/app.bsky.feed.getTimeline")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("read response body: %v", err)
	}

	if capturedAuth != "DPoP my-access-token" {
		t.Errorf("Expected Authorization 'DPoP my-access-token', got %q", capturedAuth)
	}

	if capturedDPoP == "" {
		t.Error("Expected DPoP header to be set")
	}
}
