package components

import (
	"strings"
	"testing"
)

func TestStatusBarInit(t *testing.T) {
	sb := NewStatusBar()
	if cmd := sb.Init(); cmd != nil {
		t.Errorf("Expected nil cmd, got %v", cmd)
	}
}

func TestStatusBarRender(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 100
	sb.Handle = "alice"
	sb.DID = "did:plc:123"

	v := sb.View()
	content := v.Content

	if !strings.Contains(content, "alice (did:plc:123)") {
		t.Errorf("Expected status bar to contain account info, got %q", content)
	}
}

func TestStatusBarUnreadCount(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 100
	sb.UnreadCount = 42

	v := sb.View()
	content := v.Content

	if !strings.Contains(content, "42") {
		t.Errorf("Expected status bar to contain unread badge '42', got %q", content)
	}
}
