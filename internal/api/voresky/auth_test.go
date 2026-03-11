package voresky

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dotBeeps/noms/internal/config"
)

const testCookie = "base64payload.hmacSignature"
const testDID = "did:plc:abc123"
const testHandle = "user.bsky.social"

// sessionOKHandler returns a valid authenticated session response when the
// Cookie header is present.
func sessionOKHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/session" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Cookie") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"authenticated": false,
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"user": map[string]interface{}{
				"did":         testDID,
				"handle":      testHandle,
				"displayName": "Test User",
				"avatar":      "https://example.com/avatar.jpg",
			},
		})
	}
}

// session401Handler always returns 401.
func session401Handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Not authenticated"})
	}
}

// TestVoreskyAuthFromCookie verifies that a valid cookie is accepted and the
// session info is parsed correctly.
func TestVoreskyAuthFromCookie(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(sessionOKHandler(t))
	defer srv.Close()

	store := config.NewMemoryStore()
	auth := NewVoreskyAuth(srv.URL, store)

	if err := auth.AuthenticateWithCookie(context.Background(), testCookie); err != nil {
		t.Fatalf("AuthenticateWithCookie: unexpected error: %v", err)
	}

	if auth.GetCookie() != testCookie {
		t.Errorf("cookie: got %q, want %q", auth.GetCookie(), testCookie)
	}
	if auth.GetDID() != testDID {
		t.Errorf("DID: got %q, want %q", auth.GetDID(), testDID)
	}
}

// TestVoreskyAuthInvalidCookie verifies that a 401 from the server causes
// AuthenticateWithCookie to return an error and not store the cookie.
func TestVoreskyAuthInvalidCookie(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(session401Handler(t))
	defer srv.Close()

	store := config.NewMemoryStore()
	auth := NewVoreskyAuth(srv.URL, store)

	err := auth.AuthenticateWithCookie(context.Background(), "bad.cookie")
	if err == nil {
		t.Fatal("expected error for invalid cookie, got nil")
	}

	// Cookie must be cleared after failure.
	if auth.GetCookie() != "" {
		t.Errorf("cookie should be empty after failed auth, got %q", auth.GetCookie())
	}
}

// TestVoreskySessionValidation verifies that ValidateSession returns the
// correct DID and handle from the server response.
func TestVoreskySessionValidation(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(sessionOKHandler(t))
	defer srv.Close()

	store := config.NewMemoryStore()
	auth := NewVoreskyAuth(srv.URL, store)
	auth.cookie = testCookie // inject directly to skip AuthenticateWithCookie

	info, err := auth.ValidateSession(context.Background())
	if err != nil {
		t.Fatalf("ValidateSession: unexpected error: %v", err)
	}

	if info.DID != testDID {
		t.Errorf("DID: got %q, want %q", info.DID, testDID)
	}
	if info.Handle != testHandle {
		t.Errorf("handle: got %q, want %q", info.Handle, testHandle)
	}
	if !info.Active {
		t.Error("expected Active=true")
	}
}

// TestVoreskySessionExpired verifies that a 401 from the session endpoint
// returns ErrSessionExpired.
func TestVoreskySessionExpired(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(session401Handler(t))
	defer srv.Close()

	store := config.NewMemoryStore()
	auth := NewVoreskyAuth(srv.URL, store)
	auth.cookie = testCookie // inject a cookie so we get past the empty check

	_, err := auth.ValidateSession(context.Background())
	if err == nil {
		t.Fatal("expected error for expired session, got nil")
	}
	if err != ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}
}

// TestVoreskyAuthPersistence verifies that a successful authentication
// persists the session to the TokenStore and can be retrieved.
func TestVoreskyAuthPersistence(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(sessionOKHandler(t))
	defer srv.Close()

	store := config.NewMemoryStore()
	auth := NewVoreskyAuth(srv.URL, store)

	if err := auth.AuthenticateWithCookie(context.Background(), testCookie); err != nil {
		t.Fatalf("AuthenticateWithCookie: %v", err)
	}

	// The token store should now have an entry for the DID.
	key := "voresky:" + testDID
	data, err := store.Retrieve(key)
	if err != nil {
		t.Fatalf("Retrieve from store: %v", err)
	}

	var ss storedSession
	if err := json.Unmarshal(data, &ss); err != nil {
		t.Fatalf("unmarshal stored session: %v", err)
	}
	if ss.Cookie != testCookie {
		t.Errorf("stored cookie: got %q, want %q", ss.Cookie, testCookie)
	}
	if ss.DID != testDID {
		t.Errorf("stored DID: got %q, want %q", ss.DID, testDID)
	}
}

// TestVoreskyLoadStoredSession verifies that LoadStoredSession restores a
// previously persisted session.
func TestVoreskyLoadStoredSession(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(sessionOKHandler(t))
	defer srv.Close()

	store := config.NewMemoryStore()

	// Pre-populate the store as if a previous session was saved.
	ss := storedSession{Cookie: testCookie, DID: testDID, Handle: testHandle}
	data, _ := json.Marshal(ss)
	_ = store.Store("voresky:"+testDID, data)

	auth := NewVoreskyAuth(srv.URL, store)
	if err := auth.LoadStoredSession(context.Background()); err != nil {
		t.Fatalf("LoadStoredSession: %v", err)
	}

	if auth.GetCookie() != testCookie {
		t.Errorf("cookie after load: got %q, want %q", auth.GetCookie(), testCookie)
	}
	if auth.GetDID() != testDID {
		t.Errorf("DID after load: got %q, want %q", auth.GetDID(), testDID)
	}
}
