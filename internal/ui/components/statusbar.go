package components

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dotBeeps/noms/internal/ui/theme"
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Background(theme.ColorSecondary).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1)

	connectedStyle = lipgloss.NewStyle().Foreground(theme.ColorSuccess)
	offlineStyle   = lipgloss.NewStyle().Foreground(theme.ColorError)
	badgeStyle     = lipgloss.NewStyle().
			Background(theme.ColorAccent).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Bold(true)
)

type StatusBar struct {
	Width       int
	Handle      string
	DID         string
	Connected   bool
	UnreadCount int
}

func NewStatusBar() StatusBar {
	return StatusBar{}
}

func (m StatusBar) Init() tea.Cmd {
	return nil
}

func (m StatusBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
	}
	return m, nil
}

func (m StatusBar) View() tea.View {
	status := offlineStyle.Render("○ Offline")
	if m.Connected {
		status = connectedStyle.Render("● Connected")
	}

	accountInfo := "Not logged in"
	if m.Handle != "" {
		accountInfo = fmt.Sprintf("%s (%s)", m.Handle, m.DID)
		if len(accountInfo) > 40 {
			accountInfo = accountInfo[:37] + "..."
		}
	}

	badge := ""
	if m.UnreadCount > 0 {
		badge = badgeStyle.Render(fmt.Sprintf("%d", m.UnreadCount)) + " "
	}

	left := statusBarStyle.Render(accountInfo)
	right := statusBarStyle.Render(badge + status)

	// Calculate remaining space
	w := lipgloss.Width(left) + lipgloss.Width(right)
	rem := m.Width - w
	if rem < 0 {
		rem = 0
	}

	middle := statusBarStyle.Width(rem).Render("")

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, middle, right)
	return tea.NewView(content)
}
