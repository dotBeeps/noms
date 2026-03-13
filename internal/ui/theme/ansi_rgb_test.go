package theme

import (
	"image/color"
	"testing"
)

func TestANSIToRGB(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected color.RGBA
	}{
		// Basic xterm colors (0-15)
		{"color 0 (black)", "0", color.RGBA{0, 0, 0, 255}},
		{"color 15 (white)", "15", color.RGBA{255, 255, 255, 255}},

		// 6×6×6 cube (16-231)
		{"color 16 (first cube, black)", "16", color.RGBA{0, 0, 0, 255}},
		{"color 196 (bright red)", "196", color.RGBA{255, 0, 0, 255}},
		{"color 231 (last cube, white)", "231", color.RGBA{255, 255, 255, 255}},

		// Grayscale ramp (232-255)
		{"color 232 (first grayscale)", "232", color.RGBA{8, 8, 8, 255}},
		{"color 236 (default Surface)", "236", color.RGBA{48, 48, 48, 255}},
		{"color 255 (last grayscale)", "255", color.RGBA{238, 238, 238, 255}},

		// Invalid inputs
		{"invalid string", "invalid", color.RGBA{0, 0, 0, 255}},
		{"empty string", "", color.RGBA{0, 0, 0, 255}},
		{"negative number", "-1", color.RGBA{0, 0, 0, 255}},
		{"out of range", "256", color.RGBA{0, 0, 0, 255}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ANSIToRGB(tt.code)
			if result != tt.expected {
				t.Errorf("ANSIToRGB(%q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}
