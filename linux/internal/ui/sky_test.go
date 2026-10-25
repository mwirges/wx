package ui

import (
	"image"
	"testing"
)

func TestSkyGlyphUsesTheCondition(t *testing.T) {
	sun := drawSky("clear-day", 28)
	cloud := drawSky("partly-cloudy-day", 28)
	if sun.Bounds().Dx() != 28 || cloud.Bounds().Dx() != 28 {
		t.Fatalf("sizes %v %v", sun.Bounds(), cloud.Bounds())
	}
	if !hasColor(sun, func(r, g, b uint8) bool { return r > 180 && g > 100 && b < 80 }) {
		t.Fatal("clear day has no sun")
	}
	if !hasColor(cloud, func(r, g, b uint8) bool { return r > 100 && g > 100 && b > 100 }) {
		t.Fatal("partly cloudy has no cloud")
	}
	if periodGlyph(map[string]any{"name": "Tonight", "short_description": "Clear", "is_daytime": false}) != "clear-night" {
		t.Fatal("tonight glyph")
	}
	if periodGlyph(map[string]any{"name": "This Afternoon", "short_description": "Showers", "is_daytime": true}) != "rain" {
		t.Fatal("shower glyph")
	}
	if brief("Frost Advisory issued October 18 at 1:04PM EDT until October 19 at 9:00AM EDT", 40) == "" {
		t.Fatal("brief emptied the headline")
	}
}

func hasColor(img *image.NRGBA, match func(r, g, b uint8) bool) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if c.A > 40 && match(c.R, c.G, c.B) {
				return true
			}
		}
	}
	return false
}
