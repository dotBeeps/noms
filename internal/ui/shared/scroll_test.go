package shared

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != width {
			t.Errorf("line %d: got width %d, want %d\nline: %q", i, w, width, line)
		}
	}
	if !strings.Contains(got, "alpha") {
		t.Error("expected output to contain 'alpha'")
	}
	if !strings.Contains(got, "beta") {
		t.Error("expected output to contain 'beta'")
	}
}

func TestRenderItemWithBorderKittyLines(t *testing.T) {
	t.Parallel()

	kittyLine := "left\U0010EEEEright"
	width := 12
	got := RenderItemWithBorder(kittyLine, false, width)
	lines := strings.Split(strings.TrimSuffix(got, ansi.ResetStyle+"\n\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	// Kitty path uses manual padding — verify exact output matches manual construction.
	styledBorder, gap, bgSeq := borderStyles(false, width)
	kittyWidth := strings.Count(kittyLine, "\U0010EEEE")
	nonKitty := strings.ReplaceAll(kittyLine, "\U0010EEEE", "")
	visWidth := kittyWidth + lipgloss.Width(nonKitty)
	contentWidth := max(0, width-2)
	padRight := max(0, contentWidth-visWidth)
	want := styledBorder + gap + bgSeq + StabilizeBg(kittyLine, bgSeq) + bgSeq + strings.Repeat(" ", padRight)
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
	lines := strings.Split(strings.TrimSuffix(got, ansi.ResetStyle+"\n\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// Block-level Kitty detection: ALL lines use manual path.
	// All lines must have identical lipgloss.Width.
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != width {
			t.Errorf("line %d: got width %d, want %d\nline: %q", i, w, width, line)
		}
	}

	// Kitty line: verify exact output (manual path)
	styledBorder, gap, bgSeq := borderStyles(true, width)
	kw := strings.Count(kitty, "\U0010EEEE")
	nonKitty := strings.ReplaceAll(kitty, "\U0010EEEE", "")
	kittyVisWidth := kw + lipgloss.Width(nonKitty)
	contentWidth := max(0, width-2)
	pr := max(0, contentWidth-kittyVisWidth)
	want1 := styledBorder + gap + bgSeq + StabilizeBg(kitty, bgSeq) + bgSeq + strings.Repeat(" ", pr)
	if lines[1] != want1 {
		t.Fatalf("kitty line mismatch\nwant: %q\n got: %q", want1, lines[1])
	}

	// Content preserved
	if !strings.Contains(got, "first") || !strings.Contains(got, "last") {
		t.Error("expected normal content to be preserved")
	}
	if !strings.Contains(got, "\U0010EEEE") {
		t.Error("expected kitty placeholder to be preserved")
	}
}

func TestRenderItemWithBorderMixedKittyAndPlainLines(t *testing.T) {
	t.Parallel()

	// Block with Kitty line 1 and plain lines 2-3.
	// All lines should have identical lipgloss.Width due to block-level detection.
	kittyLine := "avatar\U0010EEEE\U0010EEEE"
	plain1 := "Author Name"
	plain2 := "Post body text"
	content := kittyLine + "\n" + plain1 + "\n" + plain2
	width := 30

	got := RenderItemWithBorder(content, false, width)
	lines := strings.Split(strings.TrimSuffix(got, ansi.ResetStyle+"\n\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != width {
			t.Errorf("line %d: got width %d, want %d\nline: %q", i, w, width, line)
		}
	}
}

func borderStyles(selected bool, _ int) (styledBorder, gap, bgSeq string) {
	borderColor := theme.ColorSurface
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

	combinedLine := "\x1b[0;38;5;240msome text\x1b[0m"
	width := 40
	got := RenderItemWithBorder(combinedLine, false, width)

	_, _, bgSeq := borderStyles(false, width)

	// The combined reset \x1b[0;38;5;240m should be followed by bgSeq
	if !strings.Contains(got, "\x1b[0;38;5;240m"+bgSeq) {
		t.Errorf("expected bgSeq after combined SGR reset\ngot: %q", got)
	}

	// Width must still be exact
	lines := strings.Split(strings.TrimSuffix(got, "\n\n"), "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != width {
			t.Errorf("line %d: got width %d, want %d", i, w, width)
		}
	}
}

func TestRenderItemWithBorderNarrowTerminal(t *testing.T) {
	t.Parallel()

	content := "this is a very long line that overflows"
	width := 4

	got := RenderItemWithBorder(content, false, width)
	lines := strings.Split(strings.TrimSuffix(got, "\n\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	// Should not panic. Overflow content should be truncated.
	w := lipgloss.Width(lines[0])
	if w != width {
		t.Errorf("narrow: got width %d, want %d\nline: %q", w, width, lines[0])
	}
}

func TestRenderItemWithBorderOverflowTruncated(t *testing.T) {
	t.Parallel()

	// Content wider than the panel must be truncated to exact width.
	content := "abcdefghijklmnopqrstuvwxyz0123456789"
	width := 20

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

func TestRenderItemWithBorderVariableWidthLines(t *testing.T) {
	t.Parallel()

	// Simulates JoinWithGutter output: lines of varying widths.
	content := "short\na medium length line here\nhi\nthis one is quite a bit longer than the others"
	width := 40

	for _, selected := range []bool{false, true} {
		got := RenderItemWithBorder(content, selected, width)
		lines := strings.Split(strings.TrimSuffix(got, "\n\n"), "\n")
		if len(lines) != 4 {
			t.Fatalf("selected=%v: expected 4 lines, got %d", selected, len(lines))
		}
		for i, line := range lines {
			w := lipgloss.Width(line)
			if w != width {
				t.Errorf("selected=%v line %d: got width %d, want %d\nline: %q", selected, i, w, width, line)
			}
		}
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
