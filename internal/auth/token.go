package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type TokenSet struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	DPoPNonce    string
	TokenType    string
	Sub          string
}

type TokenManager struct {
	Tokens         *TokenSet
	TokenEndpoint  string
	ClientID       string
	DPoPSigner     *DPoPSigner
	HTTPClient     *http.Client
	OnTokenRefresh func(*TokenSet)

	mu sync.Mutex
}

func NewTokenManager(endpoint, clientID string, tokens *TokenSet, signer *DPoPSigner) *TokenManager {
	return &TokenManager{
		Tokens:        tokens,
		TokenEndpoint: endpoint,
		ClientID:      clientID,
		DPoPSigner:    signer,
		HTTPClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

// AccessToken returns the current access token in a thread-safe manner.
func (m *TokenManager) AccessToken() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Tokens == nil {
		return ""
	}
	return m.Tokens.AccessToken
}

func (m *TokenManager) IsExpired() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Tokens == nil {
		return true
	}
	// Buffer of 1 minute
	return time.Now().Add(time.Minute).After(m.Tokens.ExpiresAt)
}

func (m *TokenManager) Refresh(ctx context.Context) (*TokenSet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check if another goroutine already refreshed it while we were waiting
	if m.Tokens != nil && time.Now().Add(time.Minute).Before(m.Tokens.ExpiresAt) {
		return m.Tokens, nil
	}

	if m.Tokens == nil || m.Tokens.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	for retries := 0; retries < 2; retries++ {
		form := url.Values{}
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", m.Tokens.RefreshToken)
		form.Set("client_id", m.ClientID)

		req, err := http.NewRequestWithContext(ctx, "POST", m.TokenEndpoint, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, fmt.Errorf("building token refresh request: %w", err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		dpopJWT, err := m.DPoPSigner.Sign("POST", m.TokenEndpoint, "")
		if err != nil {
			return nil, fmt.Errorf("failed to sign DPoP: %w", err)
		}
		req.Header.Set("DPoP", dpopJWT)

		resp, err := m.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("executing token refresh request: %w", err)
		}

		// Update nonce if provided
		if nonce := resp.Header.Get("DPoP-Nonce"); nonce != "" {
			host := extractHost(m.TokenEndpoint)
			m.DPoPSigner.UpdateNonce(host, nonce)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read token response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			// Handle use_dpop_nonce error
			if resp.StatusCode == http.StatusBadRequest {
				var errResp struct {
					Error string `json:"error"`
				}
				if json.Unmarshal(body, &errResp) == nil && errResp.Error == "use_dpop_nonce" {
					if retries == 0 {
						continue // Retry with new nonce
					}
				}
			}
			return nil, fmt.Errorf("token refresh failed: %s", string(body))
		}

		var tr struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			TokenType    string `json:"token_type"`
			Sub          string `json:"sub"`
		}

		if err := json.Unmarshal(body, &tr); err != nil {
			return nil, fmt.Errorf("failed to decode token response: %w", err)
		}

		m.Tokens.AccessToken = tr.AccessToken
		if tr.RefreshToken != "" {
			m.Tokens.RefreshToken = tr.RefreshToken
		}
		if tr.ExpiresIn > 0 {
			m.Tokens.ExpiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
		}
		m.Tokens.TokenType = tr.TokenType
		if tr.Sub != "" {
			m.Tokens.Sub = tr.Sub
		}

		if m.OnTokenRefresh != nil {
			m.OnTokenRefresh(m.Tokens)
		}

		return m.Tokens, nil
	}

	return nil, fmt.Errorf("exhausted retries for DPoP nonce during token refresh")
}
