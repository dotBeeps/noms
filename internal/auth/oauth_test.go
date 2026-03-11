package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPKCEGeneration(t *testing.T) {
	verifier := GeneratePKCEVerifier()

	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("Verifier length %d outside [43, 128]", len(verifier))
	}

	challenge := GeneratePKCEChallenge(verifier)
	if challenge == "" {
		t.Error("Challenge should not be empty")
	}

	_, err := base64.RawURLEncoding.DecodeString(challenge)
	if err != nil {
		t.Errorf("Challenge is not valid base64url: %v", err)
	}

	if len(challenge) != 43 {
		t.Errorf("SHA-256 base64url challenge should be 43 chars, got %d", len(challenge))
	}
}

func TestPKCEVerifier(t *testing.T) {
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

func TestEnvironmentDetection(t *testing.T) {
	t.Run("WithDisplay", func(t *testing.T) {
		t.Setenv("DISPLAY", ":0")
		t.Setenv("WAYLAND_DISPLAY", "")
		t.Setenv("SSH_CONNECTION", "")
		flow := DetectOAuthFlow()
		if _, ok := flow.(*LoopbackFlow); !ok {
			t.Errorf("Expected *LoopbackFlow with DISPLAY set, got %T", flow)
		}
	})

	t.Run("WithWayland", func(t *testing.T) {
		t.Setenv("DISPLAY", "")
		t.Setenv("WAYLAND_DISPLAY", "wayland-0")
		t.Setenv("SSH_CONNECTION", "")
		flow := DetectOAuthFlow()
		if _, ok := flow.(*LoopbackFlow); !ok {
			t.Errorf("Expected *LoopbackFlow with WAYLAND_DISPLAY set, got %T", flow)
		}
	})

	t.Run("WithSSH", func(t *testing.T) {
		t.Setenv("DISPLAY", ":0")
		t.Setenv("SSH_CONNECTION", "1.2.3.4 5678 5.6.7.8 22")
		flow := DetectOAuthFlow()
		if _, ok := flow.(*PasteCodeFlow); !ok {
			t.Errorf("Expected *PasteCodeFlow with SSH_CONNECTION set, got %T", flow)
		}
	})

	t.Run("NoDisplay", func(t *testing.T) {
		t.Setenv("DISPLAY", "")
		t.Setenv("WAYLAND_DISPLAY", "")
		t.Setenv("SSH_CONNECTION", "")
		flow := DetectOAuthFlow()
		if _, ok := flow.(*PasteCodeFlow); !ok {
			t.Errorf("Expected *PasteCodeFlow with no display, got %T", flow)
		}
	})
}

func TestFullOAuthFlow(t *testing.T) {
	var serverURL string

	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthServerMetadata{
			Issuer:                             serverURL,
			AuthorizationEndpoint:              serverURL + "/authorize",
			TokenEndpoint:                      serverURL + "/token",
			PushedAuthorizationRequestEndpoint: serverURL + "/par",
			ResponseTypesSupported:             []string{"code"},
			GrantTypesSupported:                []string{"authorization_code", "refresh_token"},
			CodeChallengeMethodsSupported:      []string{"S256"},
			DPoPSigningAlgValuesSupported:      []string{"ES256"},
			ScopesSupported:                    []string{"atproto"},
		})
	})

	mux.HandleFunc("/par", func(w http.ResponseWriter, r *http.Request) {
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

		if r.Form.Get("code_challenge_method") != "S256" {
			t.Errorf("Expected S256, got %s", r.Form.Get("code_challenge_method"))
		}
		if r.Form.Get("code_challenge") == "" {
			t.Error("Expected code_challenge in PAR request")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"request_uri": "urn:ietf:params:oauth:request_uri:test123",
			"expires_in":  90,
		})
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
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "access-token-xyz",
			"refresh_token": "refresh-token-xyz",
			"expires_in":    3600,
			"token_type":    "DPoP",
			"sub":           "did:plc:testuser123",
		})
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
	if !strings.Contains(mockFlow.capturedURL, "request_uri=") {
		t.Errorf("Auth URL should contain request_uri (PAR): %q", mockFlow.capturedURL)
	}
}

type testMockFlow struct {
	code        string
	capturedURL string
}

func (f *testMockFlow) Start(_ context.Context, authURL string) error {
	f.capturedURL = authURL
	return nil
}

func (f *testMockFlow) WaitForCallback(_ context.Context) (string, string, error) {
	return f.code, "", nil
}
