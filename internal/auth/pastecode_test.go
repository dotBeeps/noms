package auth

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestPasteCodeFlow(t *testing.T) {
	t.Parallel()
	in := bytes.NewBufferString("  my-auth-code  \n")
	out := &bytes.Buffer{}

	flow := &PasteCodeFlow{
		In:  in,
		Out: out,
	}

	uri, err := flow.RedirectURI()
	if err != nil {
		t.Fatalf("RedirectURI failed: %v", err)
	}
	if uri != "urn:ietf:wg:oauth:2.0:oob" {
		t.Errorf("Expected urn:ietf:wg:oauth:2.0:oob, got %q", uri)
	}

	err = flow.Start(context.Background(), "http://example.com/auth")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !strings.Contains(out.String(), "http://example.com/auth") {
		t.Errorf("Output did not contain auth URL: %s", out.String())
	}

	code, state, err := flow.WaitForCallback(context.Background())
	if err != nil {
		t.Fatalf("WaitForCallback failed: %v", err)
	}

	if code != "my-auth-code" {
		t.Errorf("Expected code 'my-auth-code', got %q", code)
	}
	if state != "" {
		t.Errorf("Expected empty state, got %q", state)
	}
}

func TestPasteCodeFlow_Empty(t *testing.T) {
	t.Parallel()
	in := bytes.NewBufferString("    \n")
	out := &bytes.Buffer{}

	flow := &PasteCodeFlow{
		In:  in,
		Out: out,
	}

	_, _, err := flow.WaitForCallback(context.Background())
	if err == nil {
		t.Fatalf("Expected error for empty code, got nil")
	}
}
