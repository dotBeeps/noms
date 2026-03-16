package shared

import (
	"strings"

	"github.com/dotBeeps/noms/internal/ui/theme"
)

// Avatar placeholder dimensions
const (
	AvatarCols = 6
	AvatarRows = 3
)

// RenderPlaceholder returns a static loading placeholder of exactly cols columns wide
// and rows lines tall. For small sizes like avatars (4x2), it renders a compact
// placeholder. For larger sizes, it renders a box with ⋯ centered.
func RenderPlaceholder(cols, rows int) string {
	if cols <= 0 || rows <= 0 {
		return ""
	}

	// For avatar-sized placeholders (6x3), use compact format
	if cols == 6 && rows == 3 {
		return theme.StyleMuted().Render("[····]") + "\n" + theme.StyleMuted().Render("[····]") + "\n" + theme.StyleMuted().Render("[····]")
	}

	// For larger placeholders, render a box with ⋯ centered
	var lines []string

	// Top border
	topBorder := "┌" + strings.Repeat("─", max(0, cols-2)) + "┐"
	lines = append(lines, theme.StyleMuted().Render(topBorder))

	// Middle lines with ⋯ centered on the middle row
	middleRow := rows / 2
	for i := 1; i < rows-1; i++ {
		if i == middleRow {
			// Center the ⋯ indicator
			padding := max(0, cols-4) / 2
			line := "│" + strings.Repeat(" ", padding) + "⋯" + strings.Repeat(" ", max(0, cols-4-padding-1)) + "│"
			lines = append(lines, theme.StyleMuted().Render(line))
		} else {
			line := "│" + strings.Repeat(" ", max(0, cols-2)) + "│"
			lines = append(lines, theme.StyleMuted().Render(line))
		}
	}

	// Bottom border
	bottomBorder := "└" + strings.Repeat("─", max(0, cols-2)) + "┘"
	lines = append(lines, theme.StyleMuted().Render(bottomBorder))

	return strings.Join(lines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
