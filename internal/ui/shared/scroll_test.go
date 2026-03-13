package shared

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

func TestRenderItemWithBorderNormalLines(t *testing.T) {
	t.Parallel()

	content := "alpha\nbeta"
	width := 12
	got := RenderItemWithBorder(content, false, width)
	lines := strings.Split(strings.TrimSuffix(got, "\n\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	styledBorder, gap, lineStyle, bgSeq := borderStyles(false, width)

	want0 := styledBorder + gap + lineStyle.Render(stabilizeLine("alpha", bgSeq))
	want1 := styledBorder + gap + lineStyle.Render(stabilizeLine("beta", bgSeq))

	if lines[0] != want0 {
		t.Fatalf("line 0 mismatch\nwant: %q\n got: %q", want0, lines[0])
	}
	if lines[1] != want1 {
		t.Fatalf("line 1 mismatch\nwant: %q\n got: %q", want1, lines[1])
	}
}

func TestRenderItemWithBorderKittyLines(t *testing.T) {
	t.Parallel()

	kittyLine := "left\U0010EEEEright"
	width := 12
	got := RenderItemWithBorder(kittyLine, false, width)
	lines := strings.Split(strings.TrimSuffix(got, "\n\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	styledBorder, gap, _, bgSeq := borderStyles(false, width)
	want := styledBorder + gap + bgSeq + kittyLine
	if lines[0] != want {
		t.Fatalf("kitty line mismatch\nwant: %q\n got: %q", want, lines[0])
	}

	if !strings.Contains(lines[0], "\U0010EEEE") {
		t.Fatalf("expected kitty placeholder rune to be preserved: %q", lines[0])
	}
}

func TestRenderItemWithBorderMixedLines(t *testing.T) {
	t.Parallel()

	normal1 := "first"
	kitty := "mid\U0010EEEEmid"
	normal2 := "last"
	content := normal1 + "\n" + kitty + "\n" + normal2
	width := 12
	got := RenderItemWithBorder(content, true, width)
	lines := strings.Split(strings.TrimSuffix(got, "\n\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	styledBorder, gap, lineStyle, bgSeq := borderStyles(true, width)
	want0 := styledBorder + gap + lineStyle.Render(stabilizeLine(normal1, bgSeq))
	want1 := styledBorder + gap + bgSeq + kitty
	want2 := styledBorder + gap + lineStyle.Render(stabilizeLine(normal2, bgSeq))

	if lines[0] != want0 {
		t.Fatalf("line 0 mismatch\nwant: %q\n got: %q", want0, lines[0])
	}
	if lines[1] != want1 {
		t.Fatalf("line 1 mismatch\nwant: %q\n got: %q", want1, lines[1])
	}
	if lines[2] != want2 {
		t.Fatalf("line 2 mismatch\nwant: %q\n got: %q", want2, lines[2])
	}
}

func borderStyles(selected bool, width int) (string, string, lipgloss.Style, string) {
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

	return styledBorder, gap, lineStyle, bgSeq
}

func stabilizeLine(line, bgSeq string) string {
	stabilized := strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bgSeq)
	stabilized = strings.ReplaceAll(stabilized, "\x1b[m", "\x1b[m"+bgSeq)
	stabilized = strings.ReplaceAll(stabilized, "\x1b[49m", "\x1b[49m"+bgSeq)
	return stabilized
}
