// Package voresky provides authentication and HTTP client functionality for
// the Voresky REST API. Voresky uses cookie-based session authentication;
// this package handles cookie import, session validation, and persistence.
package voresky

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dotBeeps/noms/internal/config"
)

// SessionInfo holds the authenticated user's identity returned by
// GET /api/auth/session.
type SessionInfo struct {
	DID             string `json:"did"`
	Handle          string `json:"handle"`
	DisplayName     string `json:"displayName"`
	Avatar          string `json:"avatar"`
	MainCharacterID string `json:"mainCharacterId"`
	Active          bool   `json:"active"`
}

// sessionResponse is the raw JSON shape returned by GET /api/auth/session.
type sessionResponse struct {
	Authenticated bool `json:"authenticated"`
	User          *struct {
		DID             string `json:"did"`
		Handle          string `json:"handle"`
		DisplayName     string `json:"displayName"`
		Avatar          string `json:"avatar"`
		MainCharacterID string `json:"mainCharacterId"`
	} `json:"user"`
}

// storedSession is the JSON structure persisted to the TokenStore.
type storedSession struct {
	Cookie string `json:"cookie"`
	DID    string `json:"did"`
	Handle string `json:"handle"`
}

// ErrNotAuthenticated is returned when the session cookie is missing, invalid,
// or has expired (server returns 401).
var ErrNotAuthenticated = errors.New("not authenticated")

// ErrSessionExpired is returned when a previously valid session is no longer
// accepted by the server.
var ErrSessionExpired = errors.New("session expired")

// VoreskyAuth manages authentication state for the Voresky API.
// It stores a session cookie obtained either via cookie import or browser
// redirect, validates it against the server, and persists it using a
// TokenStore.
type VoreskyAuth struct {
	// BaseURL is the Voresky API base URL, e.g. "https://voresky.app".
	BaseURL    string
	httpClient *http.Client
	cookie     string // raw session cookie value
	did        string // active DID
	handle     string // active handle (cached)
	tokenStore config.TokenStore
}

// NewVoreskyAuth creates a new VoreskyAuth. It does not attempt to load a
// stored session; call LoadStoredSession to restore a previous session.
func NewVoreskyAuth(baseURL string, tokenStore config.TokenStore) *VoreskyAuth {
	return &VoreskyAuth{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		tokenStore: tokenStore,
	}
}

// AuthenticateWithCookie validates the provided cookie against
// GET /api/auth/session and, if valid, stores it for future use.
func (a *VoreskyAuth) AuthenticateWithCookie(ctx context.Context, cookie string) error {
	if cookie == "" {
		return fmt.Errorf("%w: cookie is empty", ErrNotAuthenticated)
	}

	// Temporarily set the cookie so ValidateSession can use it.
	a.cookie = cookie

	info, err := a.ValidateSession(ctx)
	if err != nil {
		a.cookie = "" // roll back on failure
		return err
	}

	a.did = info.DID
	a.handle = info.Handle

	return a.persistSession()
}

// ValidateSession calls GET /api/auth/session and returns the current user's
// identity. Returns ErrNotAuthenticated if the server responds with 401 or
// reports authenticated=false.
func (a *VoreskyAuth) ValidateSession(ctx context.Context) (*SessionInfo, error) {
	if a.cookie == "" {
		return nil, ErrNotAuthenticated
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.BaseURL+"/api/auth/session", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Cookie", a.cookieHeader())

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("session request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrSessionExpired
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("session check failed with status %d", resp.StatusCode)
	}

	var sr sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decode session response: %w", err)
	}

	if !sr.Authenticated || sr.User == nil {
		return nil, ErrNotAuthenticated
	}

	return &SessionInfo{
		DID:             sr.User.DID,
		Handle:          sr.User.Handle,
		DisplayName:     sr.User.DisplayName,
		Avatar:          sr.User.Avatar,
		MainCharacterID: sr.User.MainCharacterID,
		Active:          true,
	}, nil
}

// RefreshOrRevalidate checks whether the current session is still valid.
// Returns ErrNotAuthenticated or ErrSessionExpired if it is not.
func (a *VoreskyAuth) RefreshOrRevalidate(ctx context.Context) error {
	if a.cookie == "" {
		return ErrNotAuthenticated
	}
	_, err := a.ValidateSession(ctx)
	return err
}

// GetCookie returns the raw session cookie value. An empty string means no
// session is loaded.
func (a *VoreskyAuth) GetCookie() string {
	return a.cookie
}

// GetDID returns the active DID for the current session.
func (a *VoreskyAuth) GetDID() string {
	return a.did
}

// LoadStoredSession attempts to restore a previously persisted session from
// the TokenStore and validates it against the server. Returns nil if no stored
// session exists.
func (a *VoreskyAuth) LoadStoredSession(ctx context.Context) error {
	// Try each stored key that looks like a voresky session.
	accounts, err := a.tokenStore.ListAccounts()
	if err != nil {
		return fmt.Errorf("list accounts: %w", err)
	}

	for _, key := range accounts {
		if !strings.HasPrefix(key, "voresky:") {
			continue
		}

		data, err := a.tokenStore.Retrieve(key)
		if err != nil {
			continue
		}

		var ss storedSession
		if err := json.Unmarshal(data, &ss); err != nil {
			continue
		}

		a.cookie = ss.Cookie
		a.did = ss.DID
		a.handle = ss.Handle

		// Validate the restored session.
		if _, err := a.ValidateSession(ctx); err != nil {
			// Session is stale — clear it.
			a.cookie = ""
			a.did = ""
			a.handle = ""
			_ = a.tokenStore.Delete(key)
			continue
		}

		return nil
	}

	return nil
}

// Logout clears the in-memory session and removes the persisted token.
func (a *VoreskyAuth) Logout() error {
	if a.did == "" {
		return nil
	}
	key := a.storeKey()
	a.cookie = ""
	a.handle = ""
	a.did = ""
	return a.tokenStore.Delete(key)
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// cookieHeader returns the Cookie header value for the current session.
// Voresky accepts the raw cookie value; the cookie name is handled by the
// browser but for API calls we send the value directly.
func (a *VoreskyAuth) cookieHeader() string {
	return a.cookie
}

// storeKey returns the TokenStore key for the current session.
func (a *VoreskyAuth) storeKey() string {
	return "voresky:" + a.did
}

// persistSession serialises the current session and writes it to the
// TokenStore.
func (a *VoreskyAuth) persistSession() error {
	if a.did == "" {
		return nil
	}
	ss := storedSession{
		Cookie: a.cookie,
		DID:    a.did,
		Handle: a.handle,
	}
	data, err := json.Marshal(ss)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	return a.tokenStore.Store(a.storeKey(), data)
}
