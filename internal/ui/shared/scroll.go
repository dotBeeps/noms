package shared

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// LeftAccent is a border definition with only a left-side accent character.
var LeftAccent = lipgloss.Border{Left: "▎"}

// GutterWidth is the avatar column width + 1 separator space.
// Content in post bodies should indent to this column for visual rhythm.
var GutterWidth = AvatarCols + 1

func RenderItemWithBorder(content string, selected bool, width int) string {
	borderColor := theme.ColorSurface
	panelBg := theme.ColorSurface
	panelBgCode := theme.SurfaceCode()
	if selected {
		borderColor = theme.ColorAccent
		panelBg = theme.ColorSurfaceAlt
		panelBgCode = theme.SurfaceAltCode()
	}

	opts := borderOpts{
		borderColor: borderColor,
		panelBg:     panelBg,
		panelBgCode: panelBgCode,
		width:       width,
	}
	if selected {
		opts.blendBorder = true
	}
	return renderBorderedPanel(content, opts)
}

// RenderItemWithBorderColor renders a bordered panel with a specific border color (used for delete flash).
func RenderItemWithBorderColor(content string, width int, borderColor color.Color) string {
	return renderBorderedPanel(content, borderOpts{
		borderColor: borderColor,
		panelBg:     theme.ColorSurface,
		panelBgCode: theme.SurfaceCode(),
		width:       width,
	})
}

// RenderItemWithBorderMuted renders a panel with a muted border (for staggered entrance).
func RenderItemWithBorderMuted(content string, selected bool, width int) string {
	panelBg := theme.ColorSurface
	panelBgCode := theme.SurfaceCode()
	if selected {
		panelBg = theme.ColorSurfaceAlt
		panelBgCode = theme.SurfaceAltCode()
	}
	return renderBorderedPanel(content, borderOpts{
		borderColor: theme.ColorMuted,
		panelBg:     panelBg,
		panelBgCode: panelBgCode,
		width:       width,
	})
}

type borderOpts struct {
	borderColor color.Color
	panelBg     color.Color
	panelBgCode string
	width       int
	blendBorder bool // use BorderForegroundBlend (accent+primary) instead of solid color
}

func renderBorderedPanel(content string, opts borderOpts) string {
	// NOTE: This uses ANSI-256 escape syntax directly. All terminals that
	// support Bubble Tea + Kitty graphics also support 256 colors, so this
	// is safe in practice. If noms ever needs 16-color terminal support,
	// this should be rendered through lipgloss's color profile system.
	bgSeq := fmt.Sprintf("\x1b[48;5;%sm", opts.panelBgCode)

	panelStyle := lipgloss.NewStyle().
		Border(LeftAccent, false, false, false, true).
		Width(opts.width).
		Background(opts.panelBg).
		PaddingLeft(1)
	if opts.blendBorder {
		panelStyle = panelStyle.BorderForegroundBlend(theme.ColorAccent, theme.ColorPrimary)
	} else {
		panelStyle = panelStyle.BorderLeftForeground(opts.borderColor)
	}
	contentWidth := max(0, opts.width-2)

	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	// Block-level Kitty detection: if ANY line has a placeholder, use manual
	// padding for ALL lines so avatar rows and text rows align identically.
	hasKitty := slices.ContainsFunc(lines, IsKittyPlaceholderLine)

	if !hasKitty {
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
	borderStyle := lipgloss.NewStyle().Foreground(opts.borderColor)
	if opts.blendBorder {
		borderStyle = borderStyle.Foreground(theme.ColorAccent)
	}
	styledBorder := borderStyle.Render("▎")
	gap := lipgloss.NewStyle().Background(opts.panelBg).Render(" ")

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
	result.WriteString(ansi.ResetStyle + "\n\n")

	return result.String()
}
