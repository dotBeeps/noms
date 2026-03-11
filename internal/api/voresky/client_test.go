package voresky

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotBeeps/noms/internal/config"
)

// newAuthWithCookie creates a VoreskyAuth with a pre-set cookie, bypassing
// the server validation step. Useful for client tests that only care about
// cookie injection.
func newAuthWithCookie(baseURL, cookie, did string) *VoreskyAuth {
	store := config.NewMemoryStore()
	auth := NewVoreskyAuth(baseURL, store)
	auth.cookie = cookie
	auth.did = did
	auth.handle = testHandle
	return auth
}

// TestVoreskyAuthenticatedRequest verifies that the client injects the Cookie
// header on every request.
func TestVoreskyAuthenticatedRequest(t *testing.T) {
	var receivedCookie string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	auth := newAuthWithCookie(srv.URL, testCookie, testDID)
	client := NewVoreskyClient(srv.URL, auth)

	resp, err := client.Get(context.Background(), "/api/some/endpoint")
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	if receivedCookie != testCookie {
		t.Errorf("Cookie header: got %q, want %q", receivedCookie, testCookie)
	}
}

// TestVoreskyClientRetryOn401 verifies that the client retries once after a
// 401 response. The mock returns 401 on the first call and 200 on the second.
func TestVoreskyClientRetryOn401(t *testing.T) {
	callCount := 0

	// We need a session endpoint for RefreshOrRevalidate to call.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/session" {
			// Revalidation call — return success so retry proceeds.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"authenticated": true,
				"user": map[string]interface{}{
					"did":    testDID,
					"handle": testHandle,
				},
			})
			return
		}

		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"Not authenticated"}`))
			return
		}
		// Second call succeeds.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":"ok"}`))
	}))
	defer srv.Close()

	auth := newAuthWithCookie(srv.URL, testCookie, testDID)
	client := NewVoreskyClient(srv.URL, auth)

	resp, err := client.Get(context.Background(), "/api/protected")
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status after retry: got %d, want 200", resp.StatusCode)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls to protected endpoint, got %d", callCount)
	}
}

// TestVoreskyErrorParsing verifies that ParseError correctly extracts the
// error message and status code from an API error response.
func TestVoreskyErrorParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"Account not in session"}`))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/auth/switch")
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()

	apiErr := ParseError(resp)
	if apiErr == nil {
		t.Fatal("expected error, got nil")
	}

	ve, ok := apiErr.(*VoreskyError)
	if !ok {
		t.Fatalf("expected *VoreskyError, got %T", apiErr)
	}
	if ve.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode: got %d, want %d", ve.StatusCode, http.StatusForbidden)
	}
	if ve.Message != "Account not in session" {
		t.Errorf("Message: got %q, want %q", ve.Message, "Account not in session")
	}
}

// TestVoreskyClientPost verifies that POST requests serialise the body as JSON
// and set the Content-Type header.
func TestVoreskyClientPost(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	auth := newAuthWithCookie(srv.URL, testCookie, testDID)
	client := NewVoreskyClient(srv.URL, auth)

	payload := map[string]string{"did": testDID}
	resp, err := client.Post(context.Background(), "/api/auth/switch", payload)
	if err != nil {
		t.Fatalf("Post: unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}

	if !strings.Contains(receivedContentType, "application/json") {
		t.Errorf("Content-Type: got %q, want application/json", receivedContentType)
	}

	var decoded map[string]string
	if err := json.Unmarshal(receivedBody, &decoded); err != nil {
		t.Fatalf("unmarshal received body: %v", err)
	}
	if decoded["did"] != testDID {
		t.Errorf("body did: got %q, want %q", decoded["did"], testDID)
	}
}

// TestVoreskyClientDelete verifies that DELETE requests are sent correctly.
func TestVoreskyClientDelete(t *testing.T) {
	var receivedMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	auth := newAuthWithCookie(srv.URL, testCookie, testDID)
	client := NewVoreskyClient(srv.URL, auth)

	resp, err := client.Delete(context.Background(), "/api/some/resource")
	if err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if receivedMethod != http.MethodDelete {
		t.Errorf("method: got %q, want DELETE", receivedMethod)
	}
}

// TestVoreskyVoreskyErrorInterface verifies that VoreskyError satisfies the
// error interface and formats correctly.
func TestVoreskyVoreskyErrorInterface(t *testing.T) {
	ve := &VoreskyError{StatusCode: 401, Message: "Not authenticated"}
	got := ve.Error()
	if !strings.Contains(got, "401") {
		t.Errorf("Error() should contain status code, got %q", got)
	}
	if !strings.Contains(got, "Not authenticated") {
		t.Errorf("Error() should contain message, got %q", got)
	}
}
