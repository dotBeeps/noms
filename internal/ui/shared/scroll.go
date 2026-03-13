package shared

import (
	"fmt"
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

func RenderItemWithBorder(content string, selected bool, width int) string {
	borderColor := theme.ColorBorder
	panelBg := theme.ColorSurface
	panelBgCode := theme.SurfaceCode()
	if selected {
		borderColor = theme.ColorAccent
		panelBg = theme.ColorSurfaceAlt
		panelBgCode = theme.SurfaceAltCode()
	}
	styledBorder := lipgloss.NewStyle().Foreground(borderColor).Render("▎")
	gap := lipgloss.NewStyle().Background(panelBg).Render(" ")
	lineStyle := lipgloss.NewStyle().Background(panelBg).Padding(0, 1).Width(max(1, width-2))
	bgSeq := fmt.Sprintf("\x1b[48;5;%sm", panelBgCode)

	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		if IsKittyPlaceholderLine(line) {
			result.WriteString(styledBorder + gap + bgSeq + line)
		} else {
			stabilized := strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bgSeq)
			stabilized = strings.ReplaceAll(stabilized, "\x1b[m", "\x1b[m"+bgSeq)
			stabilized = strings.ReplaceAll(stabilized, "\x1b[49m", "\x1b[49m"+bgSeq)
			result.WriteString(styledBorder + gap + lineStyle.Render(stabilized))
		}
	}
	result.WriteString("\n\n")

	return result.String()
}
