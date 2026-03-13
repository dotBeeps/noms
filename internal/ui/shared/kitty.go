package shared

import "strings"

// IsKittyPlaceholderLine detects if a line contains the Kitty terminal graphics
// protocol placeholder character (U+10EEEE). This character is used by the Kitty
// terminal to mark lines containing image data, and is extremely unlikely to appear
// in normal text (it's in Supplementary Private Use Area B).
func IsKittyPlaceholderLine(line string) bool {
	return strings.Contains(line, "\U0010EEEE")
}
