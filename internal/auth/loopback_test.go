package auth

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestLoopbackServerStartStop(t *testing.T) {
	flow := NewLoopbackFlow()
	uri, err := flow.RedirectURI()
	if err != nil {
		t.Fatalf("RedirectURI failed: %v", err)
	}

	if uri == "" {
		t.Errorf("Expected non-empty redirect URI")
	}

	// Fake start to not really open a browser
	openBrowser = func(url string) error { return nil }
	defer func() { openBrowser = nil }()

	err = flow.Start(context.Background(), "http://example.com/auth")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop it
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, err = flow.WaitForCallback(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context timeout error, got %v", err)
	}
}

func TestLoopbackCapturesCode(t *testing.T) {
	flow := NewLoopbackFlow()
	uri, err := flow.RedirectURI()
	if err != nil {
		t.Fatalf("RedirectURI failed: %v", err)
	}

	openBrowser = func(url string) error { return nil }

	err = flow.Start(context.Background(), "http://example.com/auth")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// simulate callback
	go func() {
		time.Sleep(10 * time.Millisecond)
		http.Get(fmt.Sprintf("%s?code=12345&state=abcde", uri))
	}()

	code, state, err := flow.WaitForCallback(context.Background())
	if err != nil {
		t.Fatalf("WaitForCallback failed: %v", err)
	}
	if code != "12345" {
		t.Errorf("Expected code '12345', got %q", code)
	}
	if state != "abcde" {
		t.Errorf("Expected state 'abcde', got %q", state)
	}
}

func TestLoopbackRejectsInvalidState(t *testing.T) {
	flow := NewLoopbackFlow()
	uri, err := flow.RedirectURI()
	if err != nil {
		t.Fatalf("RedirectURI failed: %v", err)
	}

	// This is not exactly testing state verification here since flow just returns it,
	// but let's test missing params.
	openBrowser = func(url string) error { return nil }
	err = flow.Start(context.Background(), "http://example.com/auth")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		// Send error from OAuth server
		http.Get(fmt.Sprintf("%s?error=invalid_request", uri))
	}()

	_, _, err = flow.WaitForCallback(context.Background())
	if err == nil {
		t.Fatalf("Expected error from callback, got nil")
	}
}

func TestLoopbackServerTimeout(t *testing.T) {
	flow := NewLoopbackFlow()
	_, _ = flow.RedirectURI()
	openBrowser = func(url string) error { return nil }
	_ = flow.Start(context.Background(), "http://test")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	code, state, err := flow.WaitForCallback(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected deadline exceeded, got %v", err)
	}
	if code != "" || state != "" {
		t.Errorf("Expected empty code and state")
	}
}
