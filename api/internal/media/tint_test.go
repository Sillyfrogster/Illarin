package media

import (
	"image"
	"image/color"
	"testing"
)

func TestTintFavorsTheSaturatedHueOverTheGreyMostOfThePictureIs(t *testing.T) {
	t.Parallel()
	picture := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := range 100 {
		for x := range 200 {
			picture.Set(x, y, color.RGBA{R: 120, G: 120, B: 120, A: 255})
		}
	}
	for y := 40; y < 60; y++ {
		for x := 80; x < 120; x++ {
			picture.Set(x, y, color.RGBA{R: 124, G: 58, B: 237, A: 255})
		}
	}
	if got := Tint(picture); got != "#7c3aed" {
		t.Fatalf("tint = %s, want the violet patch", got)
	}
}

func TestTintKeepsAVeryDarkOrVeryLightPictureInTheMiddleLightnessBand(t *testing.T) {
	t.Parallel()
	for shade, want := range map[uint8]string{30: "#666666", 240: "#999999"} {
		picture := image.NewRGBA(image.Rect(0, 0, 10, 10))
		for y := range 10 {
			for x := range 10 {
				picture.Set(x, y, color.RGBA{R: shade, G: shade, B: shade, A: 255})
			}
		}
		if got := Tint(picture); got != want {
			t.Errorf("tint of grey %d = %s, want %s", shade, got, want)
		}
	}
}
