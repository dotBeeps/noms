// Package vsetup provides a BubbleTea model for importing a Voresky session
// cookie. It guides the user through copying the cookie from their browser
// DevTools, validates it via the parent App, and supports skipping entirely.
package vsetup

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ─── Messages emitted to the parent App ─────────────────────────────────────

// CookieSubmitMsg is emitted when the user submits a cookie for validation.
type CookieSubmitMsg struct{ Cookie string }

// SkipMsg is emitted when the user chooses to skip Voresky setup.
type SkipMsg struct{}

// ─── Messages received from the parent App ──────────────────────────────────

// AuthErrorMsg is sent back when cookie validation fails.
type AuthErrorMsg struct{ Err error }

// VoreskyConnectedMsg is sent when a stored session loaded successfully.
type VoreskyConnectedMsg struct{}

// ─── Model ──────────────────────────────────────────────────────────────────

type state int

const (
	stateInput state = iota
	stateValidating
	stateError
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true).
			Padding(1, 0)

	instructionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("246")).
				Padding(0, 2)

	stepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 4)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Padding(1, 2)

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Padding(1, 2)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 2)
)

// Model is the BubbleTea model for the Voresky cookie setup screen.
type Model struct {
	cookieInput textinput.Model
	state       state
	err         error
	width       int
	height      int
}

// New creates a new vsetup Model.
func New() Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Paste cookie value here"
	ti.Focus()
	ti.SetWidth(60)
	ti.EchoMode = textinput.EchoPassword

	return Model{
		cookieInput: ti,
		state:       stateInput,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		w := min(60, msg.Width-10)
		if w < 10 {
			w = 10
		}
		m.cookieInput.SetWidth(w)
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case AuthErrorMsg:
		m.err = msg.Err
		m.state = stateError
		m.cookieInput.Focus()
		return m, textinput.Blink

	case VoreskyConnectedMsg:
		// Stored session loaded — parent handles screen transition.
		return m, nil
	}

	if m.state == stateInput {
		var cmd tea.Cmd
		m.cookieInput, cmd = m.cookieInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateInput:
		switch msg.String() {
		case "enter":
			cookie := strings.TrimSpace(m.cookieInput.Value())
			if cookie != "" {
				m.state = stateValidating
				m.cookieInput.Blur()
				return m, func() tea.Msg {
					return CookieSubmitMsg{Cookie: cookie}
				}
			}
		case "esc":
			return m, func() tea.Msg { return SkipMsg{} }
		}

		var cmd tea.Cmd
		m.cookieInput, cmd = m.cookieInput.Update(msg)
		return m, cmd

	case stateValidating:
		return m, nil

	case stateError:
		switch msg.String() {
		case "enter":
			m.err = nil
			m.state = stateInput
			m.cookieInput.SetValue("")
			m.cookieInput.Focus()
			return m, textinput.Blink
		case "esc":
			return m, func() tea.Msg { return SkipMsg{} }
		}
	}

	return m, nil
}

func (m Model) View() tea.View {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Voresky Connection"))
	b.WriteString("\n\n")

	switch m.state {
	case stateInput, stateError:
		b.WriteString(instructionStyle.Render("Paste your Voresky session cookie to connect."))
		b.WriteString("\n\n")
		b.WriteString(stepStyle.Render("1. Open voresky.app in your browser and log in"))
		b.WriteString("\n")
		b.WriteString(stepStyle.Render("2. Open DevTools (F12) → Application → Cookies"))
		b.WriteString("\n")
		b.WriteString(stepStyle.Render("3. Copy the value of \"__Host-voresky_session\""))
		b.WriteString("\n\n")

		if m.state == stateError && m.err != nil {
			b.WriteString(errorStyle.Render("Error: " + m.err.Error()))
			b.WriteString("\n\n")
			b.WriteString(hintStyle.Render("Press Enter to try again, Esc to skip"))
		} else {
			b.WriteString(inputLabelStyle.Render("Cookie:"))
			b.WriteString("\n")
			b.WriteString("  " + m.cookieInput.View())
			b.WriteString("\n\n")
			b.WriteString(hintStyle.Render("Press Enter to connect, Esc to skip"))
		}

	case stateValidating:
		b.WriteString(loadingStyle.Render("Validating cookie..."))
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("Checking session with Voresky server"))
	}

	return tea.NewView(b.String())
}

// NormalizeCookie converts various cookie input formats into the Cookie
// header value expected by the Voresky API.
func NormalizeCookie(raw string) string {
	raw = strings.TrimSpace(raw)

	// Strip "Cookie: " prefix if someone copied from request headers.
	for _, prefix := range []string{"Cookie: ", "cookie: ", "Cookie:", "cookie:"} {
		if strings.HasPrefix(raw, prefix) {
			raw = strings.TrimPrefix(raw, prefix)
			break
		}
	}
	raw = strings.TrimSpace(raw)

	// If it already contains a cookie name, use as-is.
	if strings.Contains(raw, "voresky_session=") {
		return raw
	}

	// Assume bare cookie value; prepend the production cookie name.
	return "__Host-voresky_session=" + raw
}
