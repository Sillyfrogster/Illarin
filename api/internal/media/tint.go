package media

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

const (
	tintSamples = 128
	tintHueBins = 24
	// A hue has to carry this share of the picture's weight before it counts as its color.
	tintColorFloor = 0.02
	// The tint's lightness stays in this band so the page it colors reads the same in both themes.
	tintDarkest  = 0.4
	tintLightest = 0.6
)

// Tint names the color a picture reads as, taking its most saturated hue over its greys.
func Tint(source image.Image) string {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 {
		return ""
	}
	stepX := max(1, width/tintSamples)
	stepY := max(1, height/tintSamples)

	type bin struct{ weight, r, g, b float64 }
	var bins [tintHueBins]bin
	var plain bin
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, a := source.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			red, green, blue := float64(r>>8), float64(g>>8), float64(b>>8)
			plain.weight++
			plain.r += red
			plain.g += green
			plain.b += blue
			hue, saturation, lightness := hsl(red, green, blue)
			weight := saturation * (1 - math.Abs(2*lightness-1))
			if weight < 0.05 {
				continue
			}
			at := int(hue/360*tintHueBins) % tintHueBins
			bins[at].weight += weight
			bins[at].r += red * weight
			bins[at].g += green * weight
			bins[at].b += blue * weight
		}
	}
	if plain.weight == 0 {
		return ""
	}
	chosen := bins[0]
	for _, candidate := range bins[1:] {
		if candidate.weight > chosen.weight {
			chosen = candidate
		}
	}
	if chosen.weight < plain.weight*tintColorFloor {
		chosen = plain
	}
	hue, saturation, lightness := hsl(chosen.r/chosen.weight, chosen.g/chosen.weight, chosen.b/chosen.weight)
	tint := fromHSL(hue, saturation, min(max(lightness, tintDarkest), tintLightest))
	return fmt.Sprintf("#%02x%02x%02x", tint.R, tint.G, tint.B)
}

func fromHSL(hue, saturation, lightness float64) color.RGBA {
	chroma := (1 - math.Abs(2*lightness-1)) * saturation
	second := chroma * (1 - math.Abs(math.Mod(hue/60, 2)-1))
	var r, g, b float64
	switch {
	case hue < 60:
		r, g = chroma, second
	case hue < 120:
		r, g = second, chroma
	case hue < 180:
		g, b = chroma, second
	case hue < 240:
		g, b = second, chroma
	case hue < 300:
		r, b = second, chroma
	default:
		r, b = chroma, second
	}
	lift := lightness - chroma/2
	channel := func(value float64) uint8 { return uint8(math.Round((value + lift) * 255)) }
	return color.RGBA{R: channel(r), G: channel(g), B: channel(b), A: 255}
}

func hsl(red, green, blue float64) (hue, saturation, lightness float64) {
	r, g, b := red/255, green/255, blue/255
	high := math.Max(r, math.Max(g, b))
	low := math.Min(r, math.Min(g, b))
	lightness = (high + low) / 2
	spread := high - low
	if spread == 0 {
		return 0, 0, lightness
	}
	saturation = spread / (1 - math.Abs(2*lightness-1))
	switch high {
	case r:
		hue = math.Mod((g-b)/spread, 6)
	case g:
		hue = (b-r)/spread + 2
	default:
		hue = (r-g)/spread + 4
	}
	hue *= 60
	if hue < 0 {
		hue += 360
	}
	return hue, saturation, lightness
}
