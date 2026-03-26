package components

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"

	"github.com/dotBeeps/noms/internal/ui/theme"
)

// ConnectionStatus represents the current API connection state.
type ConnectionStatus int

const (
	StatusNormal      ConnectionStatus = iota
	StatusRateLimited                  // API returned 429
	StatusOffline                      // network error or timeout
)

// StatusUpdateMsg is emitted by screens when API calls succeed or fail.
// The App routes it to the StatusBar.
type StatusUpdateMsg struct {
	Status ConnectionStatus
}

// StatusBarBounceMsg triggers the badge bounce animation.
type StatusBarBounceMsg struct{}

type statusBarTickMsg struct{}
type statusClearTickMsg struct{}

func statusBarTick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return statusBarTickMsg{} })
}

func identityStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(theme.ColorPrimary).Foreground(theme.ColorOnPrimary).Bold(true).Padding(0, 1)
}
func statusZoneStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(theme.ColorSecondary).Foreground(theme.ColorTextStrong).Padding(0, 1)
}
func connectedStyle() lipgloss.Style   { return lipgloss.NewStyle().Foreground(theme.ColorSuccess) }
func offlineStyle() lipgloss.Style     { return lipgloss.NewStyle().Foreground(theme.ColorError) }
func rateLimitedStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(theme.ColorWarning) }
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
	UnreadCount int
	connStatus  ConnectionStatus
	badgeAnim   float64
	badgeVel    float64
	badgeSpring harmonica.Spring
}

func NewStatusBar() StatusBar {
	return StatusBar{
		badgeSpring: harmonica.NewSpring(harmonica.FPS(30), 8.0, 0.5),
	}
}

func (m StatusBar) Init() tea.Cmd {
	return nil
}

func (m StatusBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width

	case StatusUpdateMsg:
		m.connStatus = msg.Status
		if msg.Status == StatusRateLimited {
			// Auto-clear rate-limited status after 30 seconds
			return m, tea.Tick(30*time.Second, func(time.Time) tea.Msg {
				return statusClearTickMsg{}
			})
		}
		return m, nil

	case statusClearTickMsg:
		// Only clear if we're still in rate-limited state (not already recovered)
		if m.connStatus == StatusRateLimited {
			m.connStatus = StatusNormal
		}
		return m, nil

	case StatusBarBounceMsg:
		m.badgeAnim = 1.0
		m.badgeVel = 0
		return m, statusBarTick()

	case statusBarTickMsg:
		m.badgeAnim, m.badgeVel = m.badgeSpring.Update(m.badgeAnim, m.badgeVel, 0)
		if m.badgeAnim > 0.01 || m.badgeAnim < -0.01 {
			return m, statusBarTick()
		}
		m.badgeAnim = 0
	}
	return m, nil
}

func (m StatusBar) View() tea.View {
	var status string
	switch m.connStatus {
	case StatusRateLimited:
		status = rateLimitedStyle().Render("⚡ Rate limited")
	case StatusOffline:
		status = offlineStyle().Render("○ Offline")
	default:
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
		countText := fmt.Sprintf("%d", m.UnreadCount)
		if m.UnreadCount > 99 {
			countText = "99+"
		}
		badge = badgeStyle(extraPad).Render(countText) + " "
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
