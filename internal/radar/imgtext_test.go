package radar

import (
	"image"
	"image/color"
	"testing"
)

func TestDrawCityLabels_NilBBox(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	result := drawCityLabels(img, nil)
	if result != img {
		t.Error("nil BBox should return original image unchanged")
	}
}

func TestDrawCityLabels_NoCitiesInBBox(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// Middle of the Pacific — no cities.
	bb := &BBox{MinLat: 0, MinLon: -170, MaxLat: 1, MaxLon: -169}
	result := drawCityLabels(img, bb)
	if result != img {
		t.Error("empty city list should return original image unchanged")
	}
}

func TestDrawCityLabels_DrawsOnImage(t *testing.T) {
	// Create a black 200×200 image covering the Chicago area.
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.Black)
		}
	}
	// BBox that contains Chicago (41.8781, -87.6298).
	bb := &BBox{MinLat: 41, MinLon: -89, MaxLat: 43, MaxLon: -87}
	result := drawCityLabels(img, bb)

	// The result should be an *image.RGBA with at least some non-black pixels
	// (from the label text and dot).
	rgba, ok := result.(*image.RGBA)
	if !ok {
		t.Fatal("expected *image.RGBA result")
	}
	nonBlack := 0
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			r, g, b, _ := rgba.At(x, y).RGBA()
			if r > 0 || g > 0 || b > 0 {
				nonBlack++
			}
		}
	}
	if nonBlack == 0 {
		t.Error("expected some non-black pixels from city label drawing")
	}
}

func TestDrawBitmapText(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 30))
	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 200}
	drawBitmapText(img, 5, 5, "Test", white, black)

	// Check that some white pixels were drawn.
	whiteCount := 0
	for y := 0; y < 30; y++ {
		for x := 0; x < 200; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r == 0xffff && g == 0xffff && b == 0xffff {
				whiteCount++
			}
		}
	}
	if whiteCount == 0 {
		t.Error("expected white pixels from text rendering")
	}
}

func TestBitmapFontCoverage(t *testing.T) {
	// Verify all city names can be rendered (every rune has a glyph or falls
	// back to '?').
	for _, c := range majorCities {
		for _, ch := range c.Name {
			if _, ok := bitmapFont[ch]; !ok {
				t.Errorf("city %q has character %q (U+%04X) missing from bitmap font", c.Name, string(ch), ch)
			}
		}
	}
}
