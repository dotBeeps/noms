package login

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/dotBeeps/noms/internal/auth"
	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

func errorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorError).Bold(true).Padding(1, 0)
}

type LoginState int

const (
	LoginStateForm LoginState = iota
	LoginStateLoading
	LoginStateError
)

type AuthMethod int

const (
	AuthMethodBrowser AuthMethod = iota
	AuthMethodAppPassword
)

// AuthStepMsg updates the login model's progress display during authentication.
type AuthStepMsg struct {
	Method AuthMethod
	Step   int
}

type LoginSuccessMsg struct {
	Session *auth.Session
}

// AppPasswordLoginSuccessMsg carries the result of an app password login.
// The APIClient is already authenticated — no separate Session needed.
type AppPasswordLoginSuccessMsg struct {
	Client *atclient.APIClient
	DID    string
	Handle string
	PDS    string
}

type LoginErrorMsg struct {
	Err error
}

type StartBrowserAuthMsg struct {
	Handle string
}

type StartAppPasswordAuthMsg struct {
	Handle   string
	Password string
}

// BrowserAuthSteps are the progress steps shown during browser OAuth.
var BrowserAuthSteps = []string{
	"Resolving identity...",
	"Opening browser...",
	"Waiting for authorization...",
	"Exchanging token...",
}

// AppPasswordAuthSteps are the progress steps shown during app password login.
var AppPasswordAuthSteps = []string{
	"Authenticating...",
}

// formValues lives on the heap so huh's bound pointers survive
// Bubble Tea's value-receiver model copies.
type formValues struct {
	handle     string
	authChoice string
	password   string
}

type LoginModel struct {
	form    *huh.Form
	vals    *formValues
	spinner spinner.Model
	state   LoginState
	err     error
	width   int
	height  int

	authMethod AuthMethod
	authStep   int
}

func NewLoginModel() LoginModel {
	vals := &formValues{}
	return LoginModel{
		vals:    vals,
		form:    buildForm(vals),
		spinner: shared.NewNetworkSpinner(),
		state:   LoginStateForm,
	}
}

func buildForm(v *formValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("handle").
				Title("Handle").
				Placeholder("alice.bsky.social").
				Value(&v.handle).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("handle is required")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("auth_method").
				Title("Authentication Method").
				Options(
					huh.NewOption("Login with Browser (OAuth)", "browser"),
					huh.NewOption("Login with App Password", "app_password"),
				).
				Value(&v.authChoice),
		),
		huh.NewGroup(
			huh.NewInput().
				Key("password").
				Title("App Password").
				Placeholder("xxxx-xxxx-xxxx-xxxx").
				EchoMode(huh.EchoModePassword).
				Value(&v.password).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("app password is required")
					}
					return nil
				}),
		).WithHideFunc(func() bool {
			return v.authChoice != "app_password"
		}),
	).
		WithShowHelp(false).
		WithTheme(huh.ThemeFunc(huh.ThemeCharm)).
		WithWidth(40)
}

func (m LoginModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		w := min(50, msg.Width-10)
		m.form = m.form.WithWidth(w)
		return m, nil

	case AuthStepMsg:
		m.authMethod = msg.Method
		m.authStep = msg.Step
		return m, nil

	case LoginErrorMsg:
		m.err = msg.Err
		m.state = LoginStateError
		return m, nil

	case spinner.TickMsg:
		if m.state == LoginStateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.state == LoginStateError {
			switch msg.String() {
			case "enter", "esc", "tab":
				m.err = nil
				return m.resetForm()
			}
			return m, nil
		}
	}

	if m.state == LoginStateForm {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {
			return m.handleFormComplete()
		}

		if m.form.State == huh.StateAborted {
			return m.resetForm()
		}

		return m, cmd
	}

	return m, nil
}

func (m LoginModel) resetForm() (tea.Model, tea.Cmd) {
	m.state = LoginStateForm
	m.vals = &formValues{}
	m.form = buildForm(m.vals)
	return m, m.form.Init()
}

func (m LoginModel) handleFormComplete() (tea.Model, tea.Cmd) {
	handle := strings.TrimSpace(m.vals.handle)
	authChoice := m.vals.authChoice

	if authChoice == "browser" {
		m.state = LoginStateLoading
		m.authMethod = AuthMethodBrowser
		m.authStep = 0
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			return StartBrowserAuthMsg{Handle: handle}
		})
	}

	// App password
	password := strings.TrimSpace(m.vals.password)
	m.state = LoginStateLoading
	m.authMethod = AuthMethodAppPassword
	m.authStep = 0
	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		return StartAppPasswordAuthMsg{
			Handle:   handle,
			Password: password,
		}
	})
}

func (m LoginModel) View() tea.View {
	var content strings.Builder

	content.WriteString(theme.StyleTitle().Render("noms - atproto + voresky TUI client"))
	content.WriteString("\n\n")

	switch m.state {
	case LoginStateForm:
		content.WriteString(m.form.View())

	case LoginStateLoading:
		var steps []string
		switch m.authMethod {
		case AuthMethodBrowser:
			steps = BrowserAuthSteps
		case AuthMethodAppPassword:
			steps = AppPasswordAuthSteps
		}

		for i, label := range steps {
			if i < m.authStep {
				content.WriteString(theme.StyleStepDone().Render("✓ " + label))
			} else if i == m.authStep {
				content.WriteString(theme.StyleStepActive().Render(m.spinner.View() + " " + label))
			} else {
				content.WriteString(theme.StyleStepPending().Render("  " + label))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")
		content.WriteString(theme.StyleHint().Render("Please wait while we log you in."))

	case LoginStateError:
		errMsg := "An error occurred"
		if m.err != nil {
			errMsg = m.err.Error()
		}
		content.WriteString(errorStyle().Render(fmt.Sprintf("Error: %s", errMsg)))
		content.WriteString("\n\n")
		content.WriteString(theme.StyleHint().Render("Press Enter to try again"))
	}

	return tea.NewView(content.String())
}

func (m *LoginModel) SetValue(val string) {
	m.vals.handle = val
}

func (m LoginModel) Value() string {
	return m.vals.handle
}

func (m *LoginModel) SetError(err error) {
	m.err = err
	m.state = LoginStateError
}
