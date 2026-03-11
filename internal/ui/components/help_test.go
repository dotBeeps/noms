package components

import (
	"strings"
	"testing"
)

func TestHelpModelInit(t *testing.T) {
	t.Parallel()
	h := NewHelpModel()
	if cmd := h.Init(); cmd != nil {
		t.Errorf("Expected nil cmd, got %v", cmd)
	}
}

func TestHelpOverlayToggle(t *testing.T) {
	t.Parallel()
	h := NewHelpModel()
	h.Visible = false

	h.Toggle()
	if !h.Visible {
		t.Errorf("Expected help to be visible after toggle, got %v", h.Visible)
	}

	h.Toggle()
	if h.Visible {
		t.Errorf("Expected help to be hidden after second toggle, got %v", h.Visible)
	}
}

func TestHelpModelShowHide(t *testing.T) {
	t.Parallel()
	h := NewHelpModel()

	h.Show()
	if !h.Visible {
		t.Errorf("Expected help to be visible after Show, got %v", h.Visible)
	}

	h.Hide()
	if h.Visible {
		t.Errorf("Expected help to be hidden after Hide, got %v", h.Visible)
	}
}

func TestHelpModelContextSwitch(t *testing.T) {
	t.Parallel()
	h := NewHelpModel()
	h.Context = HelpContextLogin

	h.SetContext(HelpContextMain)
	if h.Context != HelpContextMain {
		t.Errorf("Expected context to be Main, got %v", h.Context)
	}
}

func TestHelpModelViewWhenHidden(t *testing.T) {
	t.Parallel()
	h := NewHelpModel()
	h.Visible = false

	v := h.View()
	if v.Content != "" {
		t.Errorf("Expected empty view when hidden, got %q", v.Content)
	}
}

func TestHelpModelViewWhenVisibleLoginContext(t *testing.T) {
	t.Parallel()
	h := NewHelpModel()
	h.Visible = true
	h.Context = HelpContextLogin

	v := h.View()
	content := v.Content

	if !strings.Contains(content, "Login Help") {
		t.Errorf("Expected help to contain 'Login Help' title, got %q", content)
	}

	if !strings.Contains(content, "Enter") {
		t.Errorf("Expected help to contain 'Enter' key binding, got %q", content)
	}
}

func TestHelpModelViewWhenVisibleMainContext(t *testing.T) {
	t.Parallel()
	h := NewHelpModel()
	h.Visible = true
	h.Context = HelpContextMain

	v := h.View()
	content := v.Content

	if !strings.Contains(content, "Feed") {
		t.Errorf("Expected help to contain 'Feed' title, got %q", content)
	}

	if !strings.Contains(content, "1-4") {
		t.Errorf("Expected help to contain '1-4' key binding, got %q", content)
	}
}
