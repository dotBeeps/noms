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
