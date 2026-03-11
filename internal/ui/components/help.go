package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	helpOverlayStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("236")).
				Padding(1, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))
)

type KeyBinding struct {
	Key         string
	Description string
}

type HelpContext int

const (
	HelpContextLogin HelpContext = iota
	HelpContextMain
)

var loginKeyBindings = []KeyBinding{
	{Key: "Enter", Description: "Submit handle"},
	{Key: "Tab", Description: "Switch focus"},
	{Key: "Ctrl+C", Description: "Quit"},
}

var mainKeyBindings = []KeyBinding{
	{Key: "1-4", Description: "Switch tabs"},
	{Key: "?", Description: "Toggle help"},
	{Key: "q", Description: "Quit"},
	{Key: "Ctrl+C", Description: "Force quit"},
	{Key: "j/k", Description: "Navigate up/down"},
	{Key: "Enter", Description: "Select/Open"},
	{Key: "Esc", Description: "Back/Close modal"},
}

type HelpModel struct {
	Visible bool
	Width   int
	Height  int
	Context HelpContext
}

func NewHelpModel() HelpModel {
	return HelpModel{
		Context: HelpContextLogin,
	}
}

func (m HelpModel) Init() tea.Cmd {
	return nil
}

func (m HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}
	return m, nil
}

func (m HelpModel) View() tea.View {
	if !m.Visible {
		return tea.NewView("")
	}

	var bindings []KeyBinding
	var title string

	switch m.Context {
	case HelpContextLogin:
		bindings = loginKeyBindings
		title = "Login Help"
	case HelpContextMain:
		bindings = mainKeyBindings
		title = "Keyboard Shortcuts"
	default:
		bindings = mainKeyBindings
		title = "Help"
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")).Render(title))
	lines = append(lines, "")

	maxKeyLen := 0
	for _, kb := range bindings {
		if len(kb.Key) > maxKeyLen {
			maxKeyLen = len(kb.Key)
		}
	}

	for _, kb := range bindings {
		key := helpKeyStyle.Render(fmt.Sprintf("%-*s", maxKeyLen, kb.Key))
		desc := helpDescStyle.Render(kb.Description)
		lines = append(lines, fmt.Sprintf("  %s  %s", key, desc))
	}

	content := strings.Join(lines, "\n")
	rendered := helpOverlayStyle.Render(content)

	return tea.NewView(rendered)
}

func (m *HelpModel) Toggle() {
	m.Visible = !m.Visible
}

func (m *HelpModel) SetContext(ctx HelpContext) {
	m.Context = ctx
}

func (m *HelpModel) Show() {
	m.Visible = true
}

func (m *HelpModel) Hide() {
	m.Visible = false
}
