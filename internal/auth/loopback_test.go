package auth

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"testing"
	"time"
)

func stubOpenBrowser(t *testing.T, fn func(context.Context, string) error) {
	t.Helper()
	original := openBrowser
	openBrowser = fn
	t.Cleanup(func() {
		openBrowser = original
	})
}

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
	stubOpenBrowser(t, func(_ context.Context, _ string) error { return nil })

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

	stubOpenBrowser(t, func(_ context.Context, _ string) error { return nil })

	err = flow.Start(context.Background(), "http://example.com/auth")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// simulate callback
	go func() {
		time.Sleep(10 * time.Millisecond)
		resp, err := http.Get(fmt.Sprintf("%s?code=12345&state=abcde", uri))
		if err == nil {
			_ = resp.Body.Close()
		}
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
	stubOpenBrowser(t, func(_ context.Context, _ string) error { return nil })
	err = flow.Start(context.Background(), "http://example.com/auth")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		// Send error from OAuth server
		resp, err := http.Get(fmt.Sprintf("%s?error=invalid_request", uri))
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	_, _, err = flow.WaitForCallback(context.Background())
	if err == nil {
		t.Fatalf("Expected error from callback, got nil")
	}
}

func TestLoopbackServerTimeout(t *testing.T) {
	flow := NewLoopbackFlow()
	_, _ = flow.RedirectURI()
	stubOpenBrowser(t, func(_ context.Context, _ string) error { return nil })
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

func TestStartAndCheckBrowserCommandFastFail(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	if runtime.GOOS == "windows" {
		err = startAndCheckBrowserCommand(ctx, "cmd", "/c", "exit", "1")
	} else {
		err = startAndCheckBrowserCommand(ctx, "sh", "-c", "exit 1")
	}

	if err == nil {
		t.Fatal("expected fast-fail command error, got nil")
	}
}

func TestStartAndCheckBrowserCommandLongRunning(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("long-running shell command test is Unix-specific")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := startAndCheckBrowserCommand(ctx, "sh", "-c", "sleep 2")
	if err != nil {
		t.Fatalf("expected nil for long-running command, got %v", err)
	}
}
