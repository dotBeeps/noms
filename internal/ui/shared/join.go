package shared

import "strings"

// JoinHorizontalRaw joins two multi-line strings side by side with a separator.
// Unlike lipgloss.JoinHorizontal, this does NOT pad lines to equal width,
// avoiding width miscalculation with Kitty Unicode placeholder characters.
func JoinHorizontalRaw(left, right, sep string) string {
	leftLines := strings.Split(strings.TrimRight(left, "\n"), "\n")
	rightLines := strings.Split(strings.TrimRight(right, "\n"), "\n")

	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	var result strings.Builder
	for i := 0; i < maxLines; i++ {
		if i > 0 {
			result.WriteString("\n")
		}
		l, r := "", ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		result.WriteString(l)
		if r != "" {
			result.WriteString(sep)
			result.WriteString(r)
		}
	}
	return result.String()
}

// JoinWithGutter joins two multi-line strings side by side with a fixed-width left gutter.
// When the left string has fewer lines than the right, remaining rows are indented by
// gutterWidth+len(sep) spaces, creating consistent full-body indentation.
func JoinWithGutter(left, right, sep string, gutterWidth int) string {
	leftTrimmed := strings.TrimRight(left, "\n")
	rightTrimmed := strings.TrimRight(right, "\n")

	var leftLines []string
	if leftTrimmed != "" {
		leftLines = strings.Split(leftTrimmed, "\n")
	}

	var rightLines []string
	if rightTrimmed != "" {
		rightLines = strings.Split(rightTrimmed, "\n")
	}

	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	gutter := strings.Repeat(" ", gutterWidth)

	var result strings.Builder
	for i := 0; i < maxLines; i++ {
		if i > 0 {
			result.WriteString("\n")
		}
		if i < len(leftLines) {
			result.WriteString(leftLines[i])
		} else {
			result.WriteString(gutter)
		}
		if i < len(rightLines) {
			result.WriteString(sep)
			result.WriteString(rightLines[i])
		}
	}
	return result.String()
}
