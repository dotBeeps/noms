package login

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dotBeeps/noms/internal/auth"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true).
			Padding(1, 0)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Padding(1, 0)

	optionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")).
			Padding(0, 2)

	selectedOptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("62")).
				Bold(true).
				Padding(0, 2)

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Padding(1, 0)
)

type LoginState int

const (
	LoginStateInput LoginState = iota
	LoginStateChoosing
	LoginStateLoading
	LoginStateError
)

type LoginSuccessMsg struct {
	Session *auth.Session
}

type LoginErrorMsg struct {
	Err error
}

type StartBrowserAuthMsg struct {
	Handle string
}

type StartPasteCodeAuthMsg struct {
	Handle string
}

type LoginModel struct {
	handleInput    textinput.Model
	state          LoginState
	selectedOption int
	err            error
	width          int
	height         int
}

func NewLoginModel() LoginModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Enter your handle (e.g., alice.bsky.social)"
	ti.Focus()
	ti.SetWidth(40)
	ti.SetStyles(textinput.DefaultDarkStyles())

	return LoginModel{
		handleInput:    ti,
		state:          LoginStateInput,
		selectedOption: 0,
	}
}

func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.handleInput.SetWidth(min(40, msg.Width-10))
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case LoginErrorMsg:
		m.err = msg.Err
		m.state = LoginStateError
		return m, nil
	}

	if m.state == LoginStateInput {
		var cmd tea.Cmd
		m.handleInput, cmd = m.handleInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m LoginModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case LoginStateInput:
		return m.handleInputState(msg)
	case LoginStateChoosing:
		return m.handleChoosingState(msg)
	case LoginStateError:
		return m.handleErrorState(msg)
	}
	return m, nil
}

func (m LoginModel) handleInputState(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if strings.TrimSpace(m.handleInput.Value()) != "" {
			m.state = LoginStateChoosing
			return m, nil
		}
	case "tab":
		m.state = LoginStateChoosing
		return m, nil
	}

	var cmd tea.Cmd
	m.handleInput, cmd = m.handleInput.Update(msg)
	return m, cmd
}

func (m LoginModel) handleChoosingState(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedOption > 0 {
			m.selectedOption--
		}
		return m, nil
	case "down", "j":
		if m.selectedOption < 1 {
			m.selectedOption++
		}
		return m, nil
	case "enter":
		handle := strings.TrimSpace(m.handleInput.Value())
		if m.selectedOption == 0 {
			m.state = LoginStateLoading
			return m, func() tea.Msg {
				return StartBrowserAuthMsg{Handle: handle}
			}
		}
		m.state = LoginStateLoading
		return m, func() tea.Msg {
			return StartPasteCodeAuthMsg{Handle: handle}
		}
	case "tab", "esc":
		m.state = LoginStateInput
		return m, nil
	}
	return m, nil
}

func (m LoginModel) handleErrorState(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "tab":
		m.err = nil
		m.state = LoginStateInput
		return m, nil
	}
	return m, nil
}

func (m LoginModel) View() tea.View {
	var content strings.Builder

	content.WriteString(titleStyle.Render("noms - atproto + voresky TUI client"))
	content.WriteString("\n\n")

	switch m.state {
	case LoginStateInput, LoginStateChoosing:
		content.WriteString(inputStyle.Render("Handle:"))
		content.WriteString("\n")
		content.WriteString(m.handleInput.View())
		content.WriteString("\n\n")

		if m.state == LoginStateChoosing {
			content.WriteString(optionStyle.Render("Choose authentication method:"))
			content.WriteString("\n")

			options := []string{"Login with Browser", "Paste Authorization Code"}
			for i, opt := range options {
				style := optionStyle
				if i == m.selectedOption {
					style = selectedOptionStyle
				}
				content.WriteString(style.Render(fmt.Sprintf("  %s", opt)))
				content.WriteString("\n")
			}
			content.WriteString("\n")
			content.WriteString(optionStyle.Render("Press Enter to select, Esc to go back"))
		} else {
			content.WriteString(optionStyle.Render("Press Enter to continue or Tab to choose auth method"))
		}

	case LoginStateLoading:
		content.WriteString(loadingStyle.Render("Authenticating..."))
		content.WriteString("\n")
		content.WriteString(optionStyle.Render("Please complete the OAuth flow in your browser."))

	case LoginStateError:
		errMsg := "An error occurred"
		if m.err != nil {
			errMsg = m.err.Error()
		}
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %s", errMsg)))
		content.WriteString("\n\n")
		content.WriteString(optionStyle.Render("Press Enter to try again"))
	}

	return tea.NewView(content.String())
}

func (m *LoginModel) SetValue(val string) {
	m.handleInput.SetValue(val)
}

func (m LoginModel) Value() string {
	return m.handleInput.Value()
}

func (m *LoginModel) SetError(err error) {
	m.err = err
	m.state = LoginStateError
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
