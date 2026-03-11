package login

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestLoginModelInit(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	if cmd := m.Init(); cmd == nil {
		t.Errorf("Expected non-nil cmd from Init (textinput.Blink)")
	}
}

func TestLoginScreenRender(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Handle:") {
		t.Errorf("Expected login screen to contain 'Handle:', got %q", content)
	}

	if !strings.Contains(content, "noms") {
		t.Errorf("Expected login screen to contain 'noms', got %q", content)
	}
}

func TestLoginHandleInput(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()

	m.SetValue("alice.bsky.social")

	if m.Value() != "alice.bsky.social" {
		t.Errorf("Expected handle input to be 'alice.bsky.social', got %q", m.Value())
	}
}

func TestLoginErrorDisplay(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()

	testErr := errors.New("authentication failed")
	m.SetError(testErr)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Error:") {
		t.Errorf("Expected error display to contain 'Error:', got %q", content)
	}

	if !strings.Contains(content, "authentication failed") {
		t.Errorf("Expected error display to contain error message, got %q", content)
	}
}

func TestLoginStateTransitionToChoosing(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.handleInput.SetValue("alice.bsky.social")

	updated, _ := m.Update(tea.KeyPressMsg{})
	m = updated.(LoginModel)

	if m.state != LoginStateInput {
		t.Errorf("Expected state to remain Input for non-enter key, got %v", m.state)
	}
}

func TestLoginChoosingStateNavigation(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.handleInput.SetValue("alice.bsky.social")
	m.state = LoginStateChoosing

	updated, _ := m.Update(tea.KeyPressMsg{})
	m = updated.(LoginModel)

	if m.selectedOption != 0 {
		t.Errorf("Expected selected option to be 0, got %d", m.selectedOption)
	}
}

func TestLoginWindowSizeMsg(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(LoginModel)

	if m.width != 100 {
		t.Errorf("Expected width to be 100, got %d", m.width)
	}

	if m.height != 40 {
		t.Errorf("Expected height to be 40, got %d", m.height)
	}
}

func TestLoginErrorMsg(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()

	testErr := errors.New("test error")
	updated, _ := m.Update(LoginErrorMsg{Err: testErr})
	m = updated.(LoginModel)

	if m.err != testErr {
		t.Errorf("Expected err to be test error, got %v", m.err)
	}

	if m.state != LoginStateError {
		t.Errorf("Expected state to be Error, got %v", m.state)
	}
}

func TestLoginLoadingState(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.state = LoginStateLoading

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Authenticating") {
		t.Errorf("Expected loading screen to contain 'Authenticating', got %q", content)
	}
}
