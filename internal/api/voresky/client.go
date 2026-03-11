package voresky

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// VoreskyError represents an API error returned by the Voresky server.
// It implements the error interface.
type VoreskyError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"error"`
	// Details is populated for Zod validation errors (HTTP 400).
	Details []json.RawMessage `json:"details,omitempty"`
}

func (e *VoreskyError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("voresky API error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("voresky API error %d", e.StatusCode)
}

// ParseError reads the response body and returns a *VoreskyError. The caller
// is responsible for closing resp.Body before calling this function only if
// they have already read it; otherwise ParseError reads and closes it.
func ParseError(resp *http.Response) error {
	if resp == nil {
		return fmt.Errorf("nil response")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read error body: %w", err)
	}

	ve := &VoreskyError{StatusCode: resp.StatusCode}
	// Best-effort JSON decode; ignore parse errors and use empty message.
	_ = json.Unmarshal(body, ve)
	return ve
}

// VoreskyClient is an authenticated HTTP client for the Voresky REST API.
// It injects the session cookie on every request and retries once on 401.
type VoreskyClient struct {
	// BaseURL is the Voresky API base URL, e.g. "https://voresky.app".
	BaseURL string
	auth    *VoreskyAuth
	http    *http.Client
}

// NewVoreskyClient creates a new VoreskyClient backed by the given auth.
func NewVoreskyClient(baseURL string, auth *VoreskyAuth) *VoreskyClient {
	return &VoreskyClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		auth:    auth,
		http:    &http.Client{},
	}
}

// Do executes an authenticated HTTP request.
//
//   - method: HTTP method (GET, POST, DELETE, …)
//   - path:   URL path, e.g. "/api/auth/session"
//   - body:   optional request body; marshalled to JSON if non-nil
//
// If the server responds with 401, Do attempts to revalidate the session and
// retries the request once. The caller is responsible for closing the returned
// response body.
func (c *VoreskyClient) Do(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	resp, err := c.doOnce(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// Drain and close the first response before retrying.
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// Attempt to revalidate; if it fails, surface the auth error.
		if revalErr := c.auth.RefreshOrRevalidate(ctx); revalErr != nil {
			return nil, revalErr
		}

		// Retry once with the (potentially refreshed) cookie.
		resp, err = c.doOnce(ctx, method, path, body)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// Get is a convenience wrapper for Do with method GET and no body.
func (c *VoreskyClient) Get(ctx context.Context, path string) (*http.Response, error) {
	return c.Do(ctx, http.MethodGet, path, nil)
}

// Post is a convenience wrapper for Do with method POST.
func (c *VoreskyClient) Post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.Do(ctx, http.MethodPost, path, body)
}

// Delete is a convenience wrapper for Do with method DELETE and no body.
func (c *VoreskyClient) Delete(ctx context.Context, path string) (*http.Response, error) {
	return c.Do(ctx, http.MethodDelete, path, nil)
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// doOnce builds and executes a single HTTP request with the current cookie.
func (c *VoreskyClient) doOnce(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	cookie := c.auth.GetCookie()
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return resp, nil
}
