package components

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// StatusBarBounceMsg triggers the badge bounce animation.
type StatusBarBounceMsg struct{}

type statusBarTickMsg struct{}

func statusBarTick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return statusBarTickMsg{} })
}

func identityStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(theme.ColorPrimary).Foreground(theme.ColorOnPrimary).Bold(true).Padding(0, 1)
}
func statusZoneStyle() lipgloss.Style {
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
		var still bool
		m.badgeAnim, still = shared.Decay(m.badgeAnim, 0.7, 0.01)
		if still {
			return m, statusBarTick()
		}
	}
	return m, nil
}

func (m StatusBar) View() tea.View {
	status := offlineStyle().Render("○ Offline")
	if m.Connected {
		status = connectedStyle().Render("● Connected")
	}

	handleText := "Not logged in"
	if m.Handle != "" {
		handleText = "@" + m.Handle
	}

	badge := ""
	if m.UnreadCount > 0 {
		extraPad := 0
		if m.badgeAnim > 0.5 {
			extraPad = 1
		}
		badge = badgeStyle(extraPad).Render(fmt.Sprintf("%d", m.UnreadCount)) + " "
	}

	left := identityStyle().Render(handleText)
	right := statusZoneStyle().Render(badge + status)

	w := lipgloss.Width(left) + lipgloss.Width(right)
	rem := m.Width - w
	if rem < 0 {
		rem = 0
	}

	middle := statusZoneStyle().Width(rem).Render("")
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, middle, right)
	return tea.NewView(content)
}
