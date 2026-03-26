package shared

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

// RenderEmptyState renders a centered empty-state block with a decorative
// divider, a styled message, and an optional key hint.
func RenderEmptyState(width, height int, message, hint string) string {
	dividerLen := min(24, width-4)
	if dividerLen < 4 {
		dividerLen = 4
	}
	divider := theme.StyleMuted().Render(strings.Repeat("·", dividerLen))
	msg := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Bold(true).
		Render(message)

	parts := []string{divider, "", msg}
	if hint != "" {
		parts = append(parts, "", theme.StyleMuted().Render(hint))
	}
	parts = append(parts, "", divider)

	block := lipgloss.JoinVertical(lipgloss.Center, parts...)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, block)
}

// RenderEndDivider returns a styled "─── end ───" divider centered to width.
// Emits an ANSI reset first to prevent background color bleed from preceding items.
func RenderEndDivider(width int) string {
	divider := theme.StyleMuted().Render("─── end ───")
	return ansi.ResetStyle + "\n" + lipgloss.PlaceHorizontal(width, lipgloss.Center, divider)
}

// RenderMoreIndicator returns a subtle "↓ more" hint centered to width.
// Emits an ANSI reset first to prevent background color bleed from preceding items.
func RenderMoreIndicator(width int) string {
	hint := theme.StyleMuted().Render("↓ more")
	return ansi.ResetStyle + "\n" + lipgloss.PlaceHorizontal(width, lipgloss.Center, hint)
}

// RenderErrorBox renders an error message inside a rounded bordered box,
// centered in the available space. The box contains the styled error text
// and a muted hint line (e.g. key instructions).
func RenderErrorBox(width, height int, errMsg, hint string) string {
	content := theme.StyleError().Render("Error: " + errMsg)
	if hint != "" {
		content += "\n\n" + theme.StyleMuted().Render(hint)
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorError).
		Padding(1, 2).
		Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// RenderLoadingPill renders a spinner+text loading indicator as a styled pill,
// centered horizontally within the given width.
func RenderLoadingPill(spinnerView, text string, width int) string {
	pill := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Background(theme.ColorSurfaceAlt).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorMuted).
		Render(spinnerView + " " + text)
	return ansi.ResetStyle + "\n" + lipgloss.PlaceHorizontal(width, lipgloss.Center, pill)
}
