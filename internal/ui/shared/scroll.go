package shared

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// EnsureSelectedVisible returns the scroll offset needed to keep selectedIndex
// visible. renderItem returns the rendered string for height calculation.
func EnsureSelectedVisible(itemCount, selectedIndex, offset, height int, renderItem func(index int) string) int {
	if itemCount == 0 {
		return 0
	}
	if selectedIndex < offset {
		return selectedIndex
	}

	totalHeight := 0
	heights := make([]int, 0, selectedIndex-offset+1)
	for i := offset; i <= selectedIndex && i < itemCount; i++ {
		h := strings.Count(renderItem(i), "\n")
		heights = append(heights, h)
		totalHeight += h
	}

	const margin = 2
	for totalHeight+margin > height && offset < selectedIndex {
		totalHeight -= heights[0]
		heights = heights[1:]
		offset++
	}

	return offset
}

// RenderItemWithBorder wraps content with a left border and separator line.
func RenderItemWithBorder(content string, selected bool, width int) string {
	borderColor := lipgloss.Color("238")
	if selected {
		borderColor = theme.ColorAccent
	}
	styledBorder := lipgloss.NewStyle().Foreground(borderColor).Render("▎ ")

	sep := theme.StyleMuted.Render(strings.Repeat("─", max(1, width-4)))

	allContent := strings.TrimRight(content, "\n") + "\n" + sep
	lines := strings.Split(allContent, "\n")
	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(styledBorder + line)
	}
	result.WriteString("\n")

	return result.String()
}
