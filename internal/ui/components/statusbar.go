package components

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dotBeeps/noms/internal/ui/theme"
)

// StatusBarBounceMsg triggers the badge bounce animation.
type StatusBarBounceMsg struct{}

type statusBarTickMsg struct{}

func statusBarTick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return statusBarTickMsg{} })
}

func statusBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(theme.ColorSecondary).Foreground(theme.ColorTextStrong).Padding(0, 1)
}
func connectedStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(theme.ColorSuccess) }
func offlineStyle() lipgloss.Style   { return lipgloss.NewStyle().Foreground(theme.ColorError) }
func badgeStyle(extraPad int) lipgloss.Style {
	return lipgloss.NewStyle().
		Background(theme.ColorAccent).
		Foreground(theme.ColorOnAccent).
		Padding(0, 1+extraPad).
		Bold(true)
}

type StatusBar struct {
	Width       int
	Handle      string
	DID         string
	Connected   bool
	UnreadCount int
	badgeAnim   float64
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

	case StatusBarBounceMsg:
		m.badgeAnim = 1.0
		return m, statusBarTick()

	case statusBarTickMsg:
		m.badgeAnim *= 0.7
		if m.badgeAnim > 0.01 {
			return m, statusBarTick()
		}
		m.badgeAnim = 0
	}
	return m, nil
}

func (m StatusBar) View() tea.View {
	status := offlineStyle().Render("○ Offline")
	if m.Connected {
		status = connectedStyle().Render("● Connected")
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
		extraPad := 0
		if m.badgeAnim > 0.5 {
			extraPad = 1
		}
		badge = badgeStyle(extraPad).Render(fmt.Sprintf("%d", m.UnreadCount)) + " "
	}

	left := statusBarStyle().Render(accountInfo)
	right := statusBarStyle().Render(badge + status)

	w := lipgloss.Width(left) + lipgloss.Width(right)
	rem := m.Width - w
	if rem < 0 {
		rem = 0
	}

	middle := statusBarStyle().Width(rem).Render("")
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, middle, right)
	return tea.NewView(content)
}
