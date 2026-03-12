package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestPKCEGeneration(t *testing.T) {
	t.Parallel()
	verifier, err := GeneratePKCEVerifier()
	if err != nil {
		t.Fatalf("GeneratePKCEVerifier() error: %v", err)
	}

	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("Verifier length %d outside [43, 128]", len(verifier))
	}

	challenge := GeneratePKCEChallenge(verifier)
	if challenge == "" {
		t.Error("Challenge should not be empty")
	}

	_, err = base64.RawURLEncoding.DecodeString(challenge)
	if err != nil {
		t.Errorf("Challenge is not valid base64url: %v", err)
	}

	if len(challenge) != 43 {
		t.Errorf("SHA-256 base64url challenge should be 43 chars, got %d", len(challenge))
	}
}

func TestPKCEVerifier(t *testing.T) {
	t.Parallel()
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	c1 := GeneratePKCEChallenge(verifier)
	c2 := GeneratePKCEChallenge(verifier)

	if c1 != c2 {
		t.Errorf("Same verifier should produce same challenge: %q != %q", c1, c2)
	}

	c3 := GeneratePKCEChallenge("different-verifier")
	if c1 == c3 {
		t.Error("Different verifiers should produce different challenges")
	}
}

func TestFullOAuthFlow(t *testing.T) {
	t.Parallel()
	var serverURL string

	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"resource":              serverURL,
			"authorization_servers": []string{serverURL},
		}); err != nil {
			t.Fatalf("encode protected resource metadata: %v", err)
		}
	})

	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(AuthServerMetadata{
			Issuer:                        serverURL,
			AuthorizationEndpoint:         serverURL + "/authorize",
			TokenEndpoint:                 serverURL + "/token",
			ResponseTypesSupported:        []string{"code"},
			GrantTypesSupported:           []string{"authorization_code", "refresh_token"},
			CodeChallengeMethodsSupported: []string{"S256"},
			DPoPSigningAlgValuesSupported: []string{"ES256"},
			ScopesSupported:               []string{"atproto"},
		}); err != nil {
			t.Fatalf("encode auth server metadata: %v", err)
		}
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("DPoP") == "" {
			http.Error(w, "missing DPoP", http.StatusBadRequest)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("Expected authorization_code, got %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") == "" {
			t.Error("Expected code in token request")
		}
		if r.Form.Get("code_verifier") == "" {
			t.Error("Expected code_verifier in token request")
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "access-token-xyz",
			"refresh_token": "refresh-token-xyz",
			"expires_in":    3600,
			"token_type":    "DPoP",
			"sub":           "did:plc:testuser123",
		}); err != nil {
			t.Fatalf("encode token response: %v", err)
		}
	})

	server := httptest.NewServer(mux)
	serverURL = server.URL
	defer server.Close()

	signer, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("NewDPoPSigner: %v", err)
	}

	mockFlow := &testMockFlow{code: "test-auth-code"}

	config := OAuthConfig{
		ClientID:    "http://localhost/client-metadata.json",
		RedirectURI: "http://127.0.0.1:9999/callback",
		Scopes:      []string{"atproto"},
	}

	manager := NewOAuthManager(config, mockFlow, signer)
	manager.HTTPClient = server.Client()
	manager.ResolvePDSURL = func(ctx context.Context, handle string) (string, error) {
		return serverURL, nil
	}

	session, err := manager.Authenticate(context.Background(), "test.bsky.social")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	if session.DID != "did:plc:testuser123" {
		t.Errorf("Expected DID 'did:plc:testuser123', got %q", session.DID)
	}
	if session.Handle != "test.bsky.social" {
		t.Errorf("Expected Handle 'test.bsky.social', got %q", session.Handle)
	}
	if session.Tokens.AccessToken != "access-token-xyz" {
		t.Errorf("Expected AccessToken 'access-token-xyz', got %q", session.Tokens.AccessToken)
	}
	if session.Tokens.RefreshToken != "refresh-token-xyz" {
		t.Errorf("Expected RefreshToken 'refresh-token-xyz', got %q", session.Tokens.RefreshToken)
	}
	if session.TokenManager == nil {
		t.Error("Expected TokenManager to be set")
	}
	if session.DPoP == nil {
		t.Error("Expected DPoP signer to be set")
	}

	if !strings.Contains(mockFlow.capturedURL, "/authorize") {
		t.Errorf("Auth URL should contain /authorize: %q", mockFlow.capturedURL)
	}
	if !strings.Contains(mockFlow.capturedURL, "state=") {
		t.Errorf("Auth URL should contain state parameter: %q", mockFlow.capturedURL)
	}
}

func TestAuthenticateRejectsEmptyCallbackState(t *testing.T) {
	t.Parallel()
	var serverURL string

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"resource":              serverURL,
			"authorization_servers": []string{serverURL},
		}); err != nil {
			t.Fatalf("encode protected resource metadata: %v", err)
		}
	})
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(AuthServerMetadata{
			Issuer:                             serverURL,
			AuthorizationEndpoint:              serverURL + "/authorize",
			TokenEndpoint:                      serverURL + "/token",
			PushedAuthorizationRequestEndpoint: serverURL + "/par",
		}); err != nil {
			t.Fatalf("encode auth server metadata: %v", err)
		}
	})
	mux.HandleFunc("/par", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(map[string]any{"request_uri": "urn:ietf:params:oauth:request_uri:test123"}); err != nil {
			t.Fatalf("encode PAR response: %v", err)
		}
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token-xyz",
			"refresh_token": "refresh-token-xyz",
			"expires_in":    3600,
			"token_type":    "DPoP",
			"sub":           "did:plc:testuser123",
		}); err != nil {
			t.Fatalf("encode token response: %v", err)
		}
	})

	server := httptest.NewServer(mux)
	serverURL = server.URL
	defer server.Close()

	signer, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("NewDPoPSigner: %v", err)
	}

	mockFlow := &testMockFlow{code: "test-auth-code", state: ""}
	manager := NewOAuthManager(OAuthConfig{
		ClientID:    "http://localhost/client-metadata.json",
		RedirectURI: "http://127.0.0.1:9999/callback",
		Scopes:      []string{"atproto"},
	}, mockFlow, signer)
	manager.HTTPClient = server.Client()
	manager.ResolvePDSURL = func(ctx context.Context, handle string) (string, error) { return serverURL, nil }

	_, err = manager.Authenticate(context.Background(), "test.bsky.social")
	if err == nil || !strings.Contains(err.Error(), "state mismatch") {
		t.Fatalf("expected state mismatch error, got %v", err)
	}
}

func TestAuthenticateRejectsOversizedPARResponse(t *testing.T) {
	t.Parallel()
	var serverURL string

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"resource":              serverURL,
			"authorization_servers": []string{serverURL},
		}); err != nil {
			t.Fatalf("encode protected resource metadata: %v", err)
		}
	})
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(AuthServerMetadata{
			Issuer:                             serverURL,
			AuthorizationEndpoint:              serverURL + "/authorize",
			TokenEndpoint:                      serverURL + "/token",
			PushedAuthorizationRequestEndpoint: serverURL + "/par",
		}); err != nil {
			t.Fatalf("encode auth server metadata: %v", err)
		}
	})
	mux.HandleFunc("/par", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Length", strconv.Itoa(maxOAuthResponseBytes+1))
		_, _ = w.Write([]byte(strings.Repeat("x", maxOAuthResponseBytes+1)))
	})

	server := httptest.NewServer(mux)
	serverURL = server.URL
	defer server.Close()

	signer, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("NewDPoPSigner: %v", err)
	}

	mockFlow := &testMockFlow{code: "test-auth-code"}
	manager := NewOAuthManager(OAuthConfig{
		ClientID:    "http://localhost/client-metadata.json",
		RedirectURI: "http://127.0.0.1:9999/callback",
		Scopes:      []string{"atproto"},
	}, mockFlow, signer)
	manager.HTTPClient = server.Client()
	manager.ResolvePDSURL = func(ctx context.Context, handle string) (string, error) { return serverURL, nil }

	_, err = manager.Authenticate(context.Background(), "test.bsky.social")
	want := fmt.Sprintf("PAR response exceeds %d bytes", maxOAuthResponseBytes)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

type testMockFlow struct {
	code        string
	state       string
	capturedURL string
}

func (f *testMockFlow) Start(_ context.Context, authURL string) error {
	f.capturedURL = authURL
	parsed, err := url.Parse(authURL)
	if err == nil {
		state := parsed.Query().Get("state")
		if state != "" {
			f.state = state
		}
	}
	return nil
}

func (f *testMockFlow) WaitForCallback(_ context.Context) (string, string, error) {
	return f.code, f.state, nil
}
