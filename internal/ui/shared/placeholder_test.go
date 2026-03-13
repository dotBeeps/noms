package shared

import (
	"regexp"
	"strings"
	"testing"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsi(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestRenderPlaceholderAvatarSize(t *testing.T) {
	t.Parallel()
	result := RenderPlaceholder(6, 3)
	stripped := stripAnsi(result)
	lines := strings.Split(stripped, "\n")
	if len(lines) != 3 {
		t.Errorf("RenderPlaceholder(6, 3) expected 3 lines, got %d: %q", len(lines), stripped)
		return
	}
	for i, line := range lines {
		w := len([]rune(line))
		if w != 6 {
			t.Errorf("RenderPlaceholder(6, 3) line %d: expected 6 visible columns, got %d: %q", i, w, line)
		}
	}
}

func TestRenderPlaceholderLargeSize(t *testing.T) {
	t.Parallel()
	result := RenderPlaceholder(20, 5)
	stripped := stripAnsi(result)
	lines := strings.Split(stripped, "\n")
	if len(lines) != 5 {
		t.Errorf("RenderPlaceholder(20, 5) expected 5 lines, got %d: %q", len(lines), stripped)
	}
}

func TestRenderPlaceholderZero(t *testing.T) {
	t.Parallel()
	if got := RenderPlaceholder(0, 0); got != "" {
		t.Errorf("RenderPlaceholder(0, 0) expected empty string, got %q", got)
	}
}
