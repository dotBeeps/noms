package bluesky

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

// newTestServer creates an httptest server that records requests and returns
// the provided JSON body with the given status code.
func newTestServer(t *testing.T, status int, body any) (*httptest.Server, *[]*http.Request) {
	t.Helper()
	var reqs []*http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs = append(reqs, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Fatalf("encode response: %v", err)
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &reqs
}

// newTestClient creates a Client pointed at the test server.
func newTestClient(srv *httptest.Server) *Client {
	return NewClient(srv.Client(), srv.URL, "did:plc:testuser123")
}

func TestNewClient(t *testing.T) {
	srv, _ := newTestServer(t, 200, nil)
	c := newTestClient(srv)

	if c.DID() != "did:plc:testuser123" {
		t.Errorf("DID() = %q, want %q", c.DID(), "did:plc:testuser123")
	}
	if c.APIClient() == nil {
		t.Error("APIClient() should not be nil")
	}
}

func TestParseRateLimit(t *testing.T) {
	h := http.Header{}
	h.Set("RateLimit-Limit", "100")
	h.Set("RateLimit-Remaining", "42")
	h.Set("RateLimit-Reset", "1700000000")

	rl := parseRateLimit(h)
	if rl == nil {
		t.Fatal("expected non-nil RateLimit")
	}
	if rl.Limit != 100 {
		t.Errorf("Limit = %d, want 100", rl.Limit)
	}
	if rl.Remaining != 42 {
		t.Errorf("Remaining = %d, want 42", rl.Remaining)
	}
	if rl.Reset.Unix() != 1700000000 {
		t.Errorf("Reset = %d, want 1700000000", rl.Reset.Unix())
	}
}

func TestParseRateLimitEmpty(t *testing.T) {
	rl := parseRateLimit(http.Header{})
	if rl != nil {
		t.Error("expected nil for empty headers")
	}
}

func TestIsRateLimited(t *testing.T) {
	err429 := &atclient.APIError{StatusCode: 429, Name: "RateLimited", Message: "too many requests"}
	if !isRateLimited(err429) {
		t.Error("expected 429 to be rate limited")
	}

	err500 := &atclient.APIError{StatusCode: 500, Name: "ServerError"}
	if isRateLimited(err500) {
		t.Error("expected 500 not to be rate limited")
	}

	if isRateLimited(nil) {
		t.Error("expected nil not to be rate limited")
	}
}

func TestWithRetrySuccess(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), 2, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestWithRetryNonRetryable(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), 2, func() error {
		calls++
		return &atclient.APIError{StatusCode: 500, Name: "ServerError"}
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (non-retryable)", calls)
	}
}

func TestWithRetryContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := withRetry(ctx, 2, func() error {
		return &atclient.APIError{StatusCode: 429, Name: "RateLimited"}
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseATURI(t *testing.T) {
	repo, col, rkey, err := parseATURI("at://did:plc:abc/app.bsky.feed.post/xyz123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo != "did:plc:abc" {
		t.Errorf("repo = %q, want %q", repo, "did:plc:abc")
	}
	if col != "app.bsky.feed.post" {
		t.Errorf("collection = %q, want %q", col, "app.bsky.feed.post")
	}
	if rkey != "xyz123" {
		t.Errorf("rkey = %q, want %q", rkey, "xyz123")
	}
}

func TestParseATURIInvalid(t *testing.T) {
	tests := []string{
		"https://example.com",
		"at://did:plc:abc",
		"at://did:plc:abc/collection",
		"",
	}
	for _, uri := range tests {
		_, _, _, err := parseATURI(uri)
		if err == nil {
			t.Errorf("expected error for %q", uri)
		}
	}
}

func TestWrapErr(t *testing.T) {
	err := wrapErr("TestMethod", &atProtoError{message: "something failed"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "TestMethod") {
		t.Errorf("error should contain method name, got: %s", err.Error())
	}
	if wrapErr("foo", nil) != nil {
		t.Error("wrapErr(nil) should return nil")
	}
}

func TestDPoPHeaderPresent(t *testing.T) {
	// This test verifies our client sends whatever auth the http.Client transport provides.
	// We simulate a DPoP transport by using a custom RoundTripper that adds the header.
	var capturedReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"feed": []any{}, "cursor": nil})
	}))
	t.Cleanup(srv.Close)

	httpClient := &http.Client{
		Transport: &fakeDPoPTransport{token: "test-access-token"},
	}

	c := NewClient(httpClient, srv.URL, "did:plc:test")
	_, _, err := c.GetTimeline(context.Background(), "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedReq == nil {
		t.Fatal("no request captured")
	}
	auth := capturedReq.Header.Get("Authorization")
	if auth != "DPoP test-access-token" {
		t.Errorf("Authorization = %q, want %q", auth, "DPoP test-access-token")
	}
}

// fakeDPoPTransport simulates the DPoP transport from our auth module.
type fakeDPoPTransport struct {
	token string
}

func (t *fakeDPoPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "DPoP "+t.token)
	req.Header.Set("DPoP", "fake-dpop-jwt")
	return http.DefaultTransport.RoundTrip(req)
}

func TestRateLimitRetry(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("RateLimit-Limit", "100")
			w.Header().Set("RateLimit-Remaining", "0")
			w.Header().Set("RateLimit-Reset", "9999999999")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(429)
			json.NewEncoder(w).Encode(map[string]string{"error": "RateLimitExceeded", "message": "too many requests"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"count": 5})
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:test")

	count, err := c.GetUnreadCount(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (1 retry)", callCount)
	}
}

func TestAsAPIError(t *testing.T) {
	apiErr := &atclient.APIError{StatusCode: 404, Name: "NotFound"}
	got, ok := asAPIError(apiErr)
	if !ok || got.StatusCode != 404 {
		t.Errorf("asAPIError should extract APIError")
	}

	_, ok = asAPIError(&atProtoError{message: "other"})
	if ok {
		t.Error("asAPIError should not match non-APIError")
	}
}

func TestInterfaceCompliance(t *testing.T) {
	// Verify Client implements BlueskyClient at compile time via the var _ assertion.
	// This test just confirms the assertion file compiles (it does, or the build would fail).
	var _ BlueskyClient = (*Client)(nil)
}

func TestWithRetryBackoffRespected(t *testing.T) {
	// Verify that retries happen with increasing delay.
	start := time.Now()
	calls := 0
	_ = withRetry(context.Background(), 1, func() error {
		calls++
		return &atclient.APIError{StatusCode: 429}
	})
	elapsed := time.Since(start)
	// Should have waited at least ~500ms for one retry
	if elapsed < 400*time.Millisecond {
		t.Errorf("expected at least 400ms delay, got %v", elapsed)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}
