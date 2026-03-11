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

func TestLoginPasswordStateTransition(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.handleInput.SetValue("alice.bsky.social")

	updated, _ := m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updated.(LoginModel)
	if m.state != LoginStateChoosing {
		t.Fatalf("Expected LoginStateChoosing after enter, got %d", m.state)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Text: "down"})
	m = updated.(LoginModel)
	if m.selectedOption != 1 {
		t.Fatalf("Expected selectedOption 1 after down, got %d", m.selectedOption)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updated.(LoginModel)
	if m.state != LoginStatePassword {
		t.Errorf("Expected LoginStatePassword after selecting app password, got %d", m.state)
	}
}

func TestLoginPasswordStateRender(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.handleInput.SetValue("alice.bsky.social")
	m.state = LoginStatePassword

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "alice.bsky.social") {
		t.Error("Expected handle to be shown in password state")
	}
	if !strings.Contains(content, "App Password") {
		t.Error("Expected 'App Password' label in password state")
	}
	if !strings.Contains(content, "Press Enter to login") {
		t.Error("Expected 'Press Enter to login' instruction in password state")
	}
}

func TestLoginPasswordSubmit(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.handleInput.SetValue("alice.bsky.social")
	m.state = LoginStatePassword
	m.passwordInput.SetValue("xxxx-xxxx-xxxx-xxxx")

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updated.(LoginModel)

	if m.state != LoginStateLoading {
		t.Errorf("Expected LoginStateLoading after password submit, got %d", m.state)
	}
	if cmd == nil {
		t.Fatal("Expected command to be returned after password submit")
	}

	msg := cmd()
	authMsg, ok := msg.(StartAppPasswordAuthMsg)
	if !ok {
		t.Fatalf("Expected StartAppPasswordAuthMsg, got %T", msg)
	}
	if authMsg.Handle != "alice.bsky.social" {
		t.Errorf("Expected handle 'alice.bsky.social', got %q", authMsg.Handle)
	}
	if authMsg.Password != "xxxx-xxxx-xxxx-xxxx" {
		t.Errorf("Expected password 'xxxx-xxxx-xxxx-xxxx', got %q", authMsg.Password)
	}
}

func TestLoginPasswordEscapeGoesBack(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.handleInput.SetValue("alice.bsky.social")
	m.state = LoginStatePassword

	updated, _ := m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updated.(LoginModel)

	if m.state != LoginStateChoosing {
		t.Errorf("Expected LoginStateChoosing after esc from password, got %d", m.state)
	}
}

func TestLoginPasswordEmptyReject(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.handleInput.SetValue("alice.bsky.social")
	m.state = LoginStatePassword

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updated.(LoginModel)

	if m.state != LoginStatePassword {
		t.Errorf("Expected to stay in LoginStatePassword with empty password, got %d", m.state)
	}
	if cmd != nil {
		t.Error("Expected no command when submitting empty password")
	}
}

func TestLoginTabRequiresHandle(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()

	updated, _ := m.Update(tea.KeyPressMsg{Text: "tab"})
	m = updated.(LoginModel)

	if m.state != LoginStateInput {
		t.Errorf("Expected to stay in LoginStateInput with empty handle on tab, got %d", m.state)
	}
}

func TestAppPasswordLoginSuccessMsg(t *testing.T) {
	t.Parallel()
	msg := AppPasswordLoginSuccessMsg{
		DID:    "did:plc:test123",
		Handle: "alice.bsky.social",
		PDS:    "https://bsky.social",
	}

	if msg.DID != "did:plc:test123" {
		t.Errorf("Expected DID 'did:plc:test123', got %q", msg.DID)
	}
	if msg.Handle != "alice.bsky.social" {
		t.Errorf("Expected Handle 'alice.bsky.social', got %q", msg.Handle)
	}
	if msg.PDS != "https://bsky.social" {
		t.Errorf("Expected PDS 'https://bsky.social', got %q", msg.PDS)
	}
	if msg.Client != nil {
		t.Error("Expected Client to be nil when not set")
	}
}
