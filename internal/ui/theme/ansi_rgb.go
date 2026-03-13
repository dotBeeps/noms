package theme

import (
	"image/color"
	"strconv"
)

// xterm16 contains the RGB values for ANSI colors 0-15
var xterm16 = [16]color.RGBA{
	{0x00, 0x00, 0x00, 0xFF}, // 0: black
	{0x80, 0x00, 0x00, 0xFF}, // 1: maroon
	{0x00, 0x80, 0x00, 0xFF}, // 2: green
	{0x80, 0x80, 0x00, 0xFF}, // 3: olive
	{0x00, 0x00, 0x80, 0xFF}, // 4: navy
	{0x80, 0x00, 0x80, 0xFF}, // 5: purple
	{0x00, 0x80, 0x80, 0xFF}, // 6: teal
	{0xC0, 0xC0, 0xC0, 0xFF}, // 7: silver
	{0x80, 0x80, 0x80, 0xFF}, // 8: gray
	{0xFF, 0x00, 0x00, 0xFF}, // 9: red
	{0x00, 0xFF, 0x00, 0xFF}, // 10: lime
	{0xFF, 0xFF, 0x00, 0xFF}, // 11: yellow
	{0x00, 0x00, 0xFF, 0xFF}, // 12: blue
	{0xFF, 0x00, 0xFF, 0xFF}, // 13: magenta
	{0x00, 0xFF, 0xFF, 0xFF}, // 14: cyan
	{0xFF, 0xFF, 0xFF, 0xFF}, // 15: white
}

// cube6 contains the 6 values used in the 6×6×6 color cube (colors 16-231)
var cube6 = [6]uint8{0x00, 0x5F, 0x87, 0xAF, 0xD7, 0xFF}

// ANSIToRGB converts an ANSI 256-color code string to a Go color.RGBA value.
// Handles:
//   - Colors 0-15: xterm defaults
//   - Colors 16-231: 6×6×6 RGB cube
//   - Colors 232-255: grayscale ramp
//   - Invalid/unparseable: returns opaque black RGBA{0, 0, 0, 255}
func ANSIToRGB(code string) color.RGBA {
	// Parse the code string to an integer
	num, err := strconv.Atoi(code)
	if err != nil || num < 0 || num > 255 {
		return color.RGBA{0, 0, 0, 255}
	}

	// Colors 0-15: xterm defaults
	if num < 16 {
		return xterm16[num]
	}

	// Colors 16-231: 6×6×6 RGB cube
	if num < 232 {
		n := num - 16
		r := cube6[n/36]
		g := cube6[(n%36)/6]
		b := cube6[n%6]
		return color.RGBA{r, g, b, 255}
	}

	// Colors 232-255: grayscale ramp
	// v = 8 + (code - 232) * 10
	v := uint8(8 + (num-232)*10)
	return color.RGBA{v, v, v, 255}
}
