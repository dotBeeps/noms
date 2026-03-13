package shared

import "testing"

func TestIsKittyPlaceholderLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "Kitty placeholder character alone",
			line:     "\U0010EEEE",
			expected: true,
		},
		{
			name:     "Kitty placeholder in text",
			line:     "hello\U0010EEEEworld",
			expected: true,
		},
		{
			name:     "Empty string",
			line:     "",
			expected: false,
		},
		{
			name:     "Normal text",
			line:     "this is just normal text",
			expected: false,
		},
		{
			name:     "ANSI escape codes without placeholder",
			line:     "\x1b[31mred text\x1b[0m",
			expected: false,
		},
		{
			name:     "ANSI escape codes with placeholder",
			line:     "\x1b[31m\U0010EEEE\x1b[0m",
			expected: true,
		},
		{
			name:     "Multiple placeholder characters",
			line:     "\U0010EEEE\U0010EEEE",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsKittyPlaceholderLine(tt.line)
			if result != tt.expected {
				t.Errorf("IsKittyPlaceholderLine(%q) = %v, want %v", tt.line, result, tt.expected)
			}
		})
	}
}
