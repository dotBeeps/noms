package login

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/dotBeeps/noms/internal/ui/testutil"
)

func TestLoginModelInit(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	if cmd := m.Init(); cmd == nil {
		t.Errorf("Expected non-nil cmd from Init (form init)")
	}
}

func TestLoginScreenRender(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "noms") {
		t.Errorf("Expected login screen to contain 'noms', got %q", content)
	}

	// Form fields only render after Init() cmd is processed by the runtime,
	// so we just verify the title and form state here.
	if m.state != LoginStateForm {
		t.Errorf("Expected initial state to be LoginStateForm, got %v", m.state)
	}
}

func TestLoginSetAndGetValue(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()

	m.SetValue("alice.bsky.social")

	if m.Value() != "alice.bsky.social" {
		t.Errorf("Expected handle to be 'alice.bsky.social', got %q", m.Value())
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

func TestLoginErrorStateRecoversOnEnter(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.SetError(errors.New("something broke"))

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updated.(LoginModel)

	if m.state != LoginStateForm {
		t.Errorf("Expected state to be Form after enter in error, got %v", m.state)
	}
	if m.err != nil {
		t.Errorf("Expected err to be nil after recovery, got %v", m.err)
	}
	if cmd == nil {
		t.Error("Expected form init cmd after error recovery")
	}
}

func TestLoginLoadingStateBrowser(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.state = LoginStateLoading
	m.authMethod = AuthMethodBrowser
	m.authStep = 0

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Resolving identity") {
		t.Errorf("Expected loading screen to contain 'Resolving identity', got %q", content)
	}
	if !strings.Contains(content, "Opening browser") {
		t.Errorf("Expected loading screen to show pending 'Opening browser' step, got %q", content)
	}
}

func TestLoginLoadingStateAppPassword(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.state = LoginStateLoading
	m.authMethod = AuthMethodAppPassword
	m.authStep = 0

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Authenticating") {
		t.Errorf("Expected loading screen to contain 'Authenticating', got %q", content)
	}
}

func TestLoginLoadingStateStepAdvance(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.state = LoginStateLoading
	m.authMethod = AuthMethodBrowser
	m.authStep = 0

	updated, _ := m.Update(AuthStepMsg{Method: AuthMethodBrowser, Step: 2})
	m = updated.(LoginModel)

	if m.authStep != 2 {
		t.Errorf("Expected authStep to be 2, got %d", m.authStep)
	}

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "\u2713") {
		t.Errorf("Expected completed steps to show checkmarks, got %q", content)
	}
	if !strings.Contains(content, "Waiting for authorization") {
		t.Errorf("Expected current step 'Waiting for authorization', got %q", content)
	}
}

func TestLoginFormCompleteBrowser(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.vals.handle = "alice.bsky.social"
	m.vals.authChoice = "browser"

	model, cmd := m.handleFormComplete()
	m = model.(LoginModel)

	if m.state != LoginStateLoading {
		t.Errorf("Expected LoginStateLoading, got %d", m.state)
	}
	if m.authMethod != AuthMethodBrowser {
		t.Errorf("Expected AuthMethodBrowser, got %d", m.authMethod)
	}
	if cmd == nil {
		t.Fatal("Expected command after form complete")
	}

	msgs := testutil.ExecBatch(cmd)
	found := false
	for _, msg := range msgs {
		if authMsg, ok := msg.(StartBrowserAuthMsg); ok {
			if authMsg.Handle != "alice.bsky.social" {
				t.Errorf("Expected handle 'alice.bsky.social', got %q", authMsg.Handle)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Expected StartBrowserAuthMsg in batch")
	}
}

func TestLoginFormCompleteAppPassword(t *testing.T) {
	t.Parallel()
	m := NewLoginModel()
	m.vals.handle = "alice.bsky.social"
	m.vals.authChoice = "app_password"
	m.vals.password = "xxxx-xxxx-xxxx-xxxx"

	model, cmd := m.handleFormComplete()
	m = model.(LoginModel)

	if m.state != LoginStateLoading {
		t.Errorf("Expected LoginStateLoading, got %d", m.state)
	}
	if m.authMethod != AuthMethodAppPassword {
		t.Errorf("Expected AuthMethodAppPassword, got %d", m.authMethod)
	}
	if cmd == nil {
		t.Fatal("Expected command after form complete")
	}

	msgs := testutil.ExecBatch(cmd)
	found := false
	for _, msg := range msgs {
		if authMsg, ok := msg.(StartAppPasswordAuthMsg); ok {
			if authMsg.Handle != "alice.bsky.social" {
				t.Errorf("Expected handle 'alice.bsky.social', got %q", authMsg.Handle)
			}
			if authMsg.Password != "xxxx-xxxx-xxxx-xxxx" {
				t.Errorf("Expected password 'xxxx-xxxx-xxxx-xxxx', got %q", authMsg.Password)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Expected StartAppPasswordAuthMsg in batch")
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
