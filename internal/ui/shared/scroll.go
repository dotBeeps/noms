package shared

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// LeftAccent is a border definition with only a left-side accent character.
var LeftAccent = lipgloss.Border{Left: "▎"}

func RenderItemWithBorder(content string, selected bool, width int) string {
	borderColor := theme.ColorBorder
	panelBg := theme.ColorSurface
	panelBgCode := theme.SurfaceCode()
	if selected {
		borderColor = theme.ColorAccent
		panelBg = theme.ColorSurfaceAlt
		panelBgCode = theme.SurfaceAltCode()
	}
	// NOTE: This uses ANSI-256 escape syntax directly. All terminals that
	// support Bubble Tea + Kitty graphics also support 256 colors, so this
	// is safe in practice. If noms ever needs 16-color terminal support,
	// this should be rendered through lipgloss's color profile system.
	bgSeq := fmt.Sprintf("\x1b[48;5;%sm", panelBgCode)

	// Single lipgloss style owns the entire panel: border, padding, background, width.
	// Width(w) includes border in v2, so content area = w - border(1) - paddingLeft(1).
	panelStyle := lipgloss.NewStyle().
		Border(LeftAccent, false, false, false, true).
		BorderLeftForeground(borderColor).
		Width(width).
		Background(panelBg).
		PaddingLeft(1)
	contentWidth := max(0, width-2)

	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	// Block-level Kitty detection: if ANY line has a placeholder, use manual
	// padding for ALL lines so avatar rows and text rows align identically.
	hasKitty := slices.ContainsFunc(lines, IsKittyPlaceholderLine)

	if !hasKitty {
		// Happy path: stabilize bg, truncate, let lipgloss handle everything.
		var processed strings.Builder
		for i, line := range lines {
			if i > 0 {
				processed.WriteString("\n")
			}
			stabilized := StabilizeBg(line, bgSeq)
			truncated := ansi.Truncate(stabilized, contentWidth, "")
			processed.WriteString(truncated)
		}
		return panelStyle.Render(processed.String()) + "\n\n"
	}

	// Kitty path: manual padding for ALL lines in block (consistent alignment).
	styledBorder := lipgloss.NewStyle().Foreground(borderColor).Render("▎")
	gap := lipgloss.NewStyle().Background(panelBg).Render(" ")

	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		stabilized := StabilizeBg(line, bgSeq)

		if IsKittyPlaceholderLine(line) {
			kittyWidth := strings.Count(line, "\U0010EEEE")
			nonKitty := strings.ReplaceAll(line, "\U0010EEEE", "")
			visWidth := kittyWidth + ansi.StringWidth(nonKitty)
			padRight := max(0, contentWidth-visWidth)
			result.WriteString(styledBorder + gap + bgSeq + stabilized + bgSeq + strings.Repeat(" ", padRight))
		} else {
			truncated := ansi.Truncate(stabilized, contentWidth, "")
			padRight := max(0, contentWidth-ansi.StringWidth(truncated))
			result.WriteString(styledBorder + gap + bgSeq + truncated + bgSeq + strings.Repeat(" ", padRight))
		}
	}
	result.WriteString("\x1b[0m\n\n")

	return result.String()
}
