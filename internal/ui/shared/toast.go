package shared

import (
	"image/color"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"

	"github.com/dotBeeps/noms/internal/ui/theme"
)

// ToastMsg triggers a toast notification above the status bar.
type ToastMsg struct {
	Text    string
	IsError bool
}

type toastTickMsg struct{}

func toastTick() tea.Cmd {
	return tea.Tick(time.Second/30, func(time.Time) tea.Msg { return toastTickMsg{} })
}

// ToastModel manages a single-line animated toast notification.
type ToastModel struct {
	text    string
	isError bool
	anim    float64 // 1.0 → 0.0 via spring
	animVel float64
	spring  harmonica.Spring
}

func NewToast() ToastModel {
	return ToastModel{
		spring: harmonica.NewSpring(harmonica.FPS(30), 6.0, 0.8),
	}
}

func (m ToastModel) Update(msg tea.Msg) (ToastModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ToastMsg:
		if msg.IsError {
			m.text = "✗ " + msg.Text
		} else {
			m.text = "✓ " + msg.Text
		}
		m.isError = msg.IsError
		m.anim = 1.0
		m.animVel = 0
		return m, toastTick()

	case toastTickMsg:
		if m.anim <= 0 {
			return m, nil
		}
		m.anim, m.animVel = m.spring.Update(m.anim, m.animVel, 0)
		if m.anim < 0.01 {
			m.anim = 0
			m.text = ""
			return m, nil
		}
		return m, toastTick()
	}
	return m, nil
}

// View renders the toast as a single styled line, or empty string if inactive.
func (m ToastModel) View(width int) string {
	if m.anim <= 0 || m.text == "" {
		return ""
	}

	var fg color.Color
	if m.isError {
		fg = theme.ColorError
	} else if m.anim > 0.6 {
		fg = theme.ColorHighlight
	} else if m.anim > 0.3 {
		fg = theme.ColorAccent
	} else {
		fg = theme.ColorMuted
	}

	style := lipgloss.NewStyle().
		Foreground(fg).
		Width(width).
		Align(lipgloss.Center)

	return style.Render(m.text)
}

// Active returns true if a toast is currently visible.
func (m ToastModel) Active() bool {
	return m.anim > 0 && m.text != ""
}
