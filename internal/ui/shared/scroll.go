package shared

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

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
	// NOTE: This uses ANSI-256 escape syntax directly. All terminals that
	// support Bubble Tea + Kitty graphics also support 256 colors, so this
	// is safe in practice. If noms ever needs 16-color terminal support,
	// this should be rendered through lipgloss's color profile system.
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
		// Calculate visible width: for lines with kitty placeholders, measure
		// kitty chars separately (they may confuse lipgloss.Width in some contexts),
		// then add the visible width of the remaining text portion.
		var visWidth int
		if IsKittyPlaceholderLine(line) {
			kittyWidth := strings.Count(line, "\U0010EEEE")
			nonKitty := strings.ReplaceAll(line, "\U0010EEEE", "")
			visWidth = kittyWidth + lipgloss.Width(nonKitty)
		} else {
			visWidth = lipgloss.Width(line)
		}
		contentArea := max(1, width-2)
		padRight := max(0, contentArea-1-visWidth)
		// Re-apply background color after any SGR sequence that resets the
		// background (full reset, combined reset+set, or explicit \x1b[49m).
		stabilized := StabilizeBg(line, bgSeq)
		result.WriteString(styledBorder + gap + bgSeq + " " + stabilized + bgSeq + strings.Repeat(" ", padRight))
	}
	result.WriteString("\n\n")

	return result.String()
}
