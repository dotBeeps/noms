package shared

import (
	"charm.land/lipgloss/v2"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// TruncateStr cuts s at maxLen runes and appends "…" if truncated.
func TruncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}

// UniverseStyle returns a style for rendering universe/world names.
func UniverseStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorSecondary)
}
