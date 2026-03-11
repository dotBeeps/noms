package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type OAuthConfig struct {
	ClientID    string
	RedirectURI string
	Scopes      []string
}

type AuthServerMetadata struct {
	Issuer                             string   `json:"issuer"`
	AuthorizationEndpoint              string   `json:"authorization_endpoint"`
	TokenEndpoint                      string   `json:"token_endpoint"`
	PushedAuthorizationRequestEndpoint string   `json:"pushed_authorization_request_endpoint"`
	RequirePushedAuthorizationRequests bool     `json:"require_pushed_authorization_requests"`
	DPoPSigningAlgValuesSupported      []string `json:"dpop_signing_alg_values_supported"`
	ScopesSupported                    []string `json:"scopes_supported"`
	ResponseTypesSupported             []string `json:"response_types_supported"`
	GrantTypesSupported                []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported      []string `json:"code_challenge_methods_supported"`
	RevocationEndpoint                 string   `json:"revocation_endpoint"`
}

type OAuthManager struct {
	Config     OAuthConfig
	Flow       OAuthFlow
	DPoP       *DPoPSigner
	HTTPClient *http.Client

	ResolvePDSURL func(ctx context.Context, handle string) (string, error)
}

func NewOAuthManager(config OAuthConfig, flow OAuthFlow, dpop *DPoPSigner) *OAuthManager {
	return &OAuthManager{
		Config:     config,
		Flow:       flow,
		DPoP:       dpop,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func GeneratePKCEVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating PKCE verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func GeneratePKCEChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func DetectOAuthFlow() OAuthFlow {
	if os.Getenv("SSH_CONNECTION") != "" {
		return NewPasteCodeFlow()
	}
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		return NewLoopbackFlow()
	}
	return NewPasteCodeFlow()
}

func (m *OAuthManager) Authenticate(ctx context.Context, handle string) (*Session, error) {
	redirectURI := m.Config.RedirectURI
	if provider, ok := m.Flow.(interface{ RedirectURI() (string, error) }); ok && redirectURI == "" {
		var err error
		redirectURI, err = provider.RedirectURI()
		if err != nil {
			return nil, fmt.Errorf("getting redirect URI: %w", err)
		}
	}

	pdsURL, err := m.resolvePDS(ctx, handle)
	if err != nil {
		return nil, fmt.Errorf("resolving PDS for %s: %w", handle, err)
	}

	authServerURL, err := m.fetchAuthServerURL(ctx, pdsURL)
	if err != nil {
		return nil, fmt.Errorf("discovering auth server for %s: %w", pdsURL, err)
	}

	meta, err := m.fetchAuthServerMetadata(ctx, authServerURL)
	if err != nil {
		return nil, fmt.Errorf("fetching auth server metadata: %w", err)
	}

	verifier, err := GeneratePKCEVerifier()
	if err != nil {
		return nil, err
	}
	challenge := GeneratePKCEChallenge(verifier)
	state, err := generateState()
	if err != nil {
		return nil, err
	}

	var authURL string
	if meta.PushedAuthorizationRequestEndpoint != "" {
		requestURI, err := m.sendPAR(ctx, meta, redirectURI, challenge, state, handle)
		if err != nil {
			return nil, fmt.Errorf("PAR request: %w", err)
		}
		params := url.Values{
			"client_id":   {m.Config.ClientID},
			"request_uri": {requestURI},
		}
		authURL = meta.AuthorizationEndpoint + "?" + params.Encode()
	} else {
		params := url.Values{
			"client_id":             {m.Config.ClientID},
			"response_type":         {"code"},
			"redirect_uri":          {redirectURI},
			"scope":                 {strings.Join(m.Config.Scopes, " ")},
			"state":                 {state},
			"code_challenge":        {challenge},
			"code_challenge_method": {"S256"},
		}
		authURL = meta.AuthorizationEndpoint + "?" + params.Encode()
	}

	if err := m.Flow.Start(ctx, authURL); err != nil {
		return nil, fmt.Errorf("starting auth flow: %w", err)
	}

	code, receivedState, err := m.Flow.WaitForCallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for callback: %w", err)
	}

	if receivedState != "" && receivedState != state {
		return nil, fmt.Errorf("state mismatch: expected %s, got %s", state, receivedState)
	}

	tokens, err := m.exchangeCode(ctx, meta, code, verifier, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	session := NewSession(tokens.Sub, handle, pdsURL, tokens, m.DPoP)
	tm := NewTokenManager(meta.TokenEndpoint, m.Config.ClientID, tokens, m.DPoP)
	tm.HTTPClient = m.HTTPClient
	session.TokenManager = tm

	return session, nil
}

func (m *OAuthManager) resolvePDS(ctx context.Context, handle string) (string, error) {
	if m.ResolvePDSURL != nil {
		return m.ResolvePDSURL(ctx, handle)
	}
	return defaultResolvePDS(ctx, handle, m.httpClient())
}

func (m *OAuthManager) fetchAuthServerURL(ctx context.Context, pdsURL string) (string, error) {
	metaURL := strings.TrimRight(pdsURL, "/") + "/.well-known/oauth-protected-resource"

	req, err := http.NewRequestWithContext(ctx, "GET", metaURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := m.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("protected resource metadata request failed (HTTP %d)", resp.StatusCode)
	}

	var prm struct {
		AuthorizationServers []string `json:"authorization_servers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prm); err != nil {
		return "", fmt.Errorf("decoding protected resource metadata: %w", err)
	}

	if len(prm.AuthorizationServers) == 0 {
		return "", fmt.Errorf("no authorization servers listed for %s", pdsURL)
	}

	return prm.AuthorizationServers[0], nil
}

func (m *OAuthManager) fetchAuthServerMetadata(ctx context.Context, authServerURL string) (*AuthServerMetadata, error) {
	metaURL := strings.TrimRight(authServerURL, "/") + "/.well-known/oauth-authorization-server"

	req, err := http.NewRequestWithContext(ctx, "GET", metaURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth server metadata request failed (HTTP %d)", resp.StatusCode)
	}

	var meta AuthServerMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("decoding auth server metadata: %w", err)
	}

	return &meta, nil
}

func (m *OAuthManager) sendPAR(ctx context.Context, meta *AuthServerMetadata, redirectURI, challenge, state, loginHint string) (string, error) {
	form := url.Values{
		"client_id":             {m.Config.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {strings.Join(m.Config.Scopes, " ")},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	if loginHint != "" {
		form.Set("login_hint", loginHint)
	}

	parURL := meta.PushedAuthorizationRequestEndpoint

	for attempt := 0; attempt < 2; attempt++ {
		dpopJWT, err := m.DPoP.Sign("POST", parURL, "")
		if err != nil {
			return "", fmt.Errorf("DPoP signing for PAR: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", parURL, strings.NewReader(form.Encode()))
		if err != nil {
			return "", fmt.Errorf("building PAR request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("DPoP", dpopJWT)

		resp, err := m.httpClient().Do(req)
		if err != nil {
			return "", fmt.Errorf("sending PAR request: %w", err)
		}

		if nonce := resp.Header.Get("DPoP-Nonce"); nonce != "" {
			m.DPoP.UpdateNonce(extractHost(parURL), nonce)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("reading PAR response: %w", err)
		}

		if resp.StatusCode == http.StatusBadRequest {
			var errResp struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(body, &errResp) == nil && errResp.Error == "use_dpop_nonce" {
				continue
			}
			return "", fmt.Errorf("PAR failed: %s", string(body))
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return "", fmt.Errorf("PAR failed (HTTP %d): %s", resp.StatusCode, string(body))
		}

		var parResp struct {
			RequestURI string `json:"request_uri"`
			ExpiresIn  int    `json:"expires_in"`
		}
		if err := json.Unmarshal(body, &parResp); err != nil {
			return "", fmt.Errorf("decoding PAR response: %w", err)
		}

		return parResp.RequestURI, nil
	}

	return "", fmt.Errorf("PAR: exhausted DPoP nonce retries")
}

func (m *OAuthManager) exchangeCode(ctx context.Context, meta *AuthServerMetadata, code, verifier, redirectURI string) (*TokenSet, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {m.Config.ClientID},
		"code_verifier": {verifier},
	}

	for attempt := 0; attempt < 2; attempt++ {
		dpopJWT, err := m.DPoP.Sign("POST", meta.TokenEndpoint, "")
		if err != nil {
			return nil, fmt.Errorf("DPoP signing for token exchange: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", meta.TokenEndpoint, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, fmt.Errorf("building token exchange request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("DPoP", dpopJWT)

		resp, err := m.httpClient().Do(req)
		if err != nil {
			return nil, fmt.Errorf("sending token exchange request: %w", err)
		}

		if nonce := resp.Header.Get("DPoP-Nonce"); nonce != "" {
			m.DPoP.UpdateNonce(extractHost(meta.TokenEndpoint), nonce)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading token response: %w", err)
		}

		if resp.StatusCode == http.StatusBadRequest {
			var errResp struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(body, &errResp) == nil && errResp.Error == "use_dpop_nonce" {
				continue
			}
			return nil, fmt.Errorf("token exchange failed: %s", string(body))
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
		}

		var tr struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			TokenType    string `json:"token_type"`
			Sub          string `json:"sub"`
		}
		if err := json.Unmarshal(body, &tr); err != nil {
			return nil, fmt.Errorf("decoding token response: %w", err)
		}

		return &TokenSet{
			AccessToken:  tr.AccessToken,
			RefreshToken: tr.RefreshToken,
			ExpiresAt:    time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
			TokenType:    tr.TokenType,
			Sub:          tr.Sub,
		}, nil
	}

	return nil, fmt.Errorf("token exchange: exhausted DPoP nonce retries")
}

func (m *OAuthManager) httpClient() *http.Client {
	if m.HTTPClient != nil {
		return m.HTTPClient
	}
	return http.DefaultClient
}

func generateState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func defaultResolvePDS(ctx context.Context, handle string, _ *http.Client) (string, error) {
	h, err := syntax.ParseHandle(handle)
	if err != nil {
		return "", fmt.Errorf("invalid handle %q: %w", handle, err)
	}

	dir := identity.DefaultDirectory()
	ident, err := dir.LookupHandle(ctx, h)
	if err != nil {
		return "", fmt.Errorf("resolving identity for %s: %w", handle, err)
	}

	pds := ident.PDSEndpoint()
	if pds == "" {
		return "", fmt.Errorf("no PDS endpoint found for %s", handle)
	}

	return pds, nil
}
