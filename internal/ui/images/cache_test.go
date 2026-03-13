package images

import (
	"image"
	"image/color"
	"testing"
)

func TestApplyRoundedCornersBackground(t *testing.T) {
	srcColor := color.RGBA{200, 100, 50, 255}
	bgColor := color.RGBA{48, 48, 48, 255}

	src := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			src.Set(x, y, srcColor)
		}
	}

	rounded := applyRoundedCorners(src, 0.3, bgColor)
	got := color.RGBAModel.Convert(rounded.At(0, 0)).(color.RGBA)

	if got != bgColor {
		t.Fatalf("corner pixel = %#v, want %#v", got, bgColor)
	}
}

func TestApplyRoundedCornersCenter(t *testing.T) {
	srcColor := color.RGBA{200, 100, 50, 255}
	bgColor := color.RGBA{48, 48, 48, 255}

	src := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			src.Set(x, y, srcColor)
		}
	}

	rounded := applyRoundedCorners(src, 0.3, bgColor)
	got := color.RGBAModel.Convert(rounded.At(5, 5)).(color.RGBA)

	if got != srcColor {
		t.Fatalf("center pixel = %#v, want %#v", got, srcColor)
	}
}
