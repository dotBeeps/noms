package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenExchange(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected form content type")
		}
		if r.Header.Get("DPoP") == "" {
			t.Errorf("Expected DPoP header")
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("Expected grant_type refresh_token, got %s", r.Form.Get("grant_type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
			"expires_in":    3600,
			"token_type":    "DPoP",
			"sub":           "did:plc:test",
		})
	}))
	defer server.Close()

	signer, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("NewDPoPSigner: %v", err)
	}

	tokens := &TokenSet{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Hour),
		TokenType:    "DPoP",
	}

	tm := NewTokenManager(server.URL, "test-client", tokens, signer)
	tm.HTTPClient = server.Client()

	result, err := tm.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if result.AccessToken != "new-access" {
		t.Errorf("Expected access_token 'new-access', got %q", result.AccessToken)
	}
	if result.RefreshToken != "new-refresh" {
		t.Errorf("Expected refresh_token 'new-refresh', got %q", result.RefreshToken)
	}
	if result.TokenType != "DPoP" {
		t.Errorf("Expected token_type 'DPoP', got %q", result.TokenType)
	}
	if result.Sub != "did:plc:test" {
		t.Errorf("Expected sub 'did:plc:test', got %q", result.Sub)
	}
}

func TestTokenExchangeError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"Refresh token expired"}`))
	}))
	defer server.Close()

	signer, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("NewDPoPSigner: %v", err)
	}

	tokens := &TokenSet{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Hour),
	}

	tm := NewTokenManager(server.URL, "test-client", tokens, signer)
	tm.HTTPClient = server.Client()

	_, err = tm.Refresh(context.Background())
	if err == nil {
		t.Fatal("Expected error from Refresh, got nil")
	}
}

func TestTokenRefreshMutex(t *testing.T) {
	t.Parallel()
	var refreshCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCount.Add(1)
		time.Sleep(50 * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "refreshed",
			"refresh_token": "new-refresh",
			"expires_in":    3600,
			"token_type":    "DPoP",
		})
	}))
	defer server.Close()

	signer, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("NewDPoPSigner: %v", err)
	}

	tokens := &TokenSet{
		AccessToken:  "old",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Hour),
	}

	tm := NewTokenManager(server.URL, "test-client", tokens, signer)
	tm.HTTPClient = server.Client()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = tm.Refresh(context.Background())
		}()
	}
	wg.Wait()

	count := refreshCount.Load()
	if count != 1 {
		t.Errorf("Expected exactly 1 refresh request (mutex+double-check), got %d", count)
	}
}

func TestTokenRefreshBeforeExpiry(t *testing.T) {
	t.Parallel()
	signer, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("NewDPoPSigner: %v", err)
	}

	tokens := &TokenSet{
		AccessToken:  "token",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(30 * time.Second),
	}

	tm := NewTokenManager("http://example.com/token", "test", tokens, signer)

	if !tm.IsExpired() {
		t.Error("Token expiring in 30s should be considered expired (1min buffer)")
	}

	tokens.ExpiresAt = time.Now().Add(5 * time.Minute)
	if tm.IsExpired() {
		t.Error("Token expiring in 5min should NOT be considered expired")
	}

	tm.Tokens = nil
	if !tm.IsExpired() {
		t.Error("Nil tokens should be considered expired")
	}
}
