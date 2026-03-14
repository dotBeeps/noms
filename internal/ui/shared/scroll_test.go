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

	styledBorder, gap, bgSeq := borderStyles(false, width)
	contentArea := max(1, width-2)

	padRight0 := max(0, contentArea-1-5) // "alpha" visWidth=5
	want0 := styledBorder + gap + bgSeq + " " + StabilizeBg("alpha", bgSeq) + bgSeq + strings.Repeat(" ", padRight0)

	padRight1 := max(0, contentArea-1-4) // "beta" visWidth=4
	want1 := styledBorder + gap + bgSeq + " " + StabilizeBg("beta", bgSeq) + bgSeq + strings.Repeat(" ", padRight1)

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

	styledBorder, gap, bgSeq := borderStyles(false, width)
	kittyWidth := strings.Count(kittyLine, "\U0010EEEE")
	nonKitty := strings.ReplaceAll(kittyLine, "\U0010EEEE", "")
	visWidth := kittyWidth + lipgloss.Width(nonKitty)
	contentArea := max(1, width-2)
	padRight := max(0, contentArea-1-visWidth)
	want := styledBorder + gap + bgSeq + " " + StabilizeBg(kittyLine, bgSeq) + bgSeq + strings.Repeat(" ", padRight)
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

	styledBorder, gap, bgSeq := borderStyles(true, width)
	contentArea := max(1, width-2)

	padRight0 := max(0, contentArea-1-5) // "first" visWidth=5
	want0 := styledBorder + gap + bgSeq + " " + StabilizeBg(normal1, bgSeq) + bgSeq + strings.Repeat(" ", padRight0)

	kw := strings.Count(kitty, "\U0010EEEE")
	nonKitty := strings.ReplaceAll(kitty, "\U0010EEEE", "")
	kittyVisWidth := kw + lipgloss.Width(nonKitty)
	pr := max(0, contentArea-1-kittyVisWidth)
	want1 := styledBorder + gap + bgSeq + " " + StabilizeBg(kitty, bgSeq) + bgSeq + strings.Repeat(" ", pr)

	padRight2 := max(0, contentArea-1-4) // "last" visWidth=4
	want2 := styledBorder + gap + bgSeq + " " + StabilizeBg(normal2, bgSeq) + bgSeq + strings.Repeat(" ", padRight2)

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

func borderStyles(selected bool, width int) (styledBorder, gap, bgSeq string) {
	borderColor := theme.ColorBorder
	panelBg := theme.ColorSurface
	panelBgCode := theme.SurfaceCode()
	if selected {
		borderColor = theme.ColorAccent
		panelBg = theme.ColorSurfaceAlt
		panelBgCode = theme.SurfaceAltCode()
	}

	styledBorder = lipgloss.NewStyle().Foreground(borderColor).Render("▎")
	gap = lipgloss.NewStyle().Background(panelBg).Render(" ")
	bgSeq = fmt.Sprintf("\x1b[48;5;%sm", panelBgCode)
	return
}

func TestRenderItemWithBorderANSIWidth(t *testing.T) {
	t.Parallel()

	// Simulate avatar ANSI + separator + content ANSI (the exact bug pattern)
	ansiLine := "\x1b[38;5;242m[····]\x1b[0m \x1b[38;5;7mHello world\x1b[0m"
	content := ansiLine + "\nplain line\n" + ansiLine
	width := 60

	for _, selected := range []bool{false, true} {
		got := RenderItemWithBorder(content, selected, width)
		lines := strings.Split(strings.TrimSuffix(got, "\n\n"), "\n")
		for i, line := range lines {
			w := lipgloss.Width(line)
			if w != width {
				t.Errorf("selected=%v line %d: got width %d, want %d\nline: %q", selected, i, w, width, line)
			}
		}
	}
}

func TestRenderItemWithBorderExactFit(t *testing.T) {
	t.Parallel()

	// Content that exactly fills the available width should have padRight=0
	// (no overflow). contentArea = width-2 = 10, leading space = 1,
	// so content filling 9 chars → padRight = 10-1-9 = 0.
	content := "123456789"
	width := 12
	got := RenderItemWithBorder(content, false, width)
	lines := strings.Split(strings.TrimSuffix(got, "\n\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	w := lipgloss.Width(lines[0])
	if w != width {
		t.Errorf("exact-fit: got width %d, want %d\nline: %q", w, width, lines[0])
	}
}

func TestRenderItemWithBorderCombinedSGR(t *testing.T) {
	t.Parallel()

	// Combined SGR sequence: reset + set foreground in one sequence.
	// The old literal string replacement would miss this; the regex catches it.
	combinedLine := "\x1b[0;38;5;240msome text\x1b[0m"
	width := 40
	got := RenderItemWithBorder(combinedLine, false, width)

	_, _, bgSeq := borderStyles(false, width)

	// The combined reset \x1b[0;38;5;240m should be followed by bgSeq
	if !strings.Contains(got, "\x1b[0;38;5;240m"+bgSeq) {
		t.Errorf("expected bgSeq after combined SGR reset\ngot: %q", got)
	}
}

func TestRenderItemWithBorderNarrowTerminal(t *testing.T) {
	t.Parallel()

	// On a very narrow terminal, visWidth may exceed contentArea, making
	// padRight negative. max(0, ...) must clamp it — strings.Repeat with a
	// negative count returns "" silently, but the visual output is wrong.
	content := "this is a very long line that overflows"
	width := 4 // extremely narrow: contentArea=2, leading space=1, so any content overflows
	got := RenderItemWithBorder(content, false, width)
	lines := strings.Split(strings.TrimSuffix(got, "\n\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	// Should not panic and must not contain a negative repeat artifact.
	// The line must still start with the border + gap sequence.
	styledBorder, gap, bgSeq := borderStyles(false, width)
	prefix := styledBorder + gap + bgSeq + " "
	if !strings.HasPrefix(lines[0], prefix) {
		t.Errorf("narrow: line missing expected prefix\nwant prefix: %q\n got line:   %q", prefix, lines[0])
	}
}

func TestStabilizeBgCombinedSequences(t *testing.T) {
	t.Parallel()

	bgSeq := "\x1b[48;5;236m"

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full reset",
			input: "\x1b[0m",
			want:  "\x1b[0m" + bgSeq,
		},
		{
			name:  "implicit reset",
			input: "\x1b[m",
			want:  "\x1b[m" + bgSeq,
		},
		{
			name:  "default bg",
			input: "\x1b[49m",
			want:  "\x1b[49m" + bgSeq,
		},
		{
			name:  "combined reset+fg",
			input: "\x1b[0;38;5;240m",
			want:  "\x1b[0;38;5;240m" + bgSeq,
		},
		{
			name:  "double-zero reset",
			input: "\x1b[00m",
			want:  "\x1b[00m" + bgSeq,
		},
		{
			name:  "fg-only no reset",
			input: "\x1b[38;5;240m",
			want:  "\x1b[38;5;240m",
		},
		{
			name:  "bold no reset",
			input: "\x1b[1m",
			want:  "\x1b[1m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := StabilizeBg(tt.input, bgSeq)
			if got != tt.want {
				t.Errorf("StabilizeBg(%q, bgSeq)\n got: %q\nwant: %q", tt.input, got, tt.want)
			}
		})
	}
}

