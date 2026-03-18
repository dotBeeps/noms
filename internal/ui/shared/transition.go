package shared

import "strings"

// SliceContent extracts lines [from, to) from rendered content.
// Handles out-of-bounds gracefully.
func SliceContent(content string, from, to int) string {
	lines := strings.Split(content, "\n")
	if from < 0 {
		from = 0
	}
	if to > len(lines) {
		to = len(lines)
	}
	if from >= to {
		return ""
	}
	return strings.Join(lines[from:to], "\n")
}

// PadToHeight ensures content has exactly height lines by padding with empty lines.
func PadToHeight(content string, height int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:height], "\n")
}
