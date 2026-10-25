package ui

import (
	"image"
	"testing"
	"time"
)

func TestConditionCascadeMatchesTheMacPhases(t *testing.T) {
	if barOpacity(0, 0, false) != 0.50 || barOpacity(4, 0, false) != 1 {
		t.Fatal("phase 0 is the resting gradient")
	}
	if barOpacity(0, 1, false) != 0 || barOpacity(1, 1, false) != 0.70 {
		t.Fatal("phase 1 drops the outer bar")
	}
	if barOpacity(2, 2, false) != 0.84 || barOpacity(1, 2, false) != 0 {
		t.Fatal("phase 2 collapses inward")
	}
	if barOpacity(3, 3, false) != 0.65 || barOpacity(4, 3, false) != 0.90 {
		t.Fatal("phase 3 holds the inner pair")
	}
	if barOpacity(4, 4, false) != 0.48 || barOpacity(3, 4, false) != 0 {
		t.Fatal("phase 4 is the last bar")
	}
	for i := 0; i < 5; i++ {
		if barOpacity(i, 5, false) != 0 {
			t.Fatalf("phase 5 bar %d still lit", i)
		}
	}
	if barOpacity(0, 6, false) != 0.40 || barOpacity(3, 6, false) != 0.40 || barOpacity(1, 6, false) != 0 {
		t.Fatal("phase 6 pre-charge")
	}
	if barOpacity(0, 7, false) != 0.60 || barOpacity(4, 7, false) != 0 {
		t.Fatal("phase 7 spark")
	}
	if barOpacity(4, 8, false) != 1 || barOpacity(2, 8, false) != 0.75 || barOpacity(0, 8, false) != 0 {
		t.Fatal("phase 8 strobe")
	}
	if barOpacity(0, 8, true) != 0.50 {
		t.Fatal("reduce motion ignores the strobe")
	}
	if conditionPhase(0, bandRed) != 0 || conditionPhase(50*time.Millisecond, bandRed) != 1 {
		t.Fatal("red steps at 50ms")
	}
	if conditionPhase(49*time.Millisecond, bandRed) != 0 {
		t.Fatal("red step is not early")
	}
	if conditionPhase(90*time.Millisecond, bandYellow) != 1 || conditionPhase(50*time.Millisecond, bandYellow) != 0 {
		t.Fatal("yellow steps at 90ms")
	}
	if conditionPhase(8*50*time.Millisecond, bandRed) != 8 {
		t.Fatal("phase 8 is the strobe frame")
	}
	if pulseAlpha(0) < 250 {
		t.Fatalf("pulse starts lit: %d", pulseAlpha(0))
	}
	if dim := pulseAlpha(600 * time.Millisecond); dim < 80 || dim > 100 {
		t.Fatalf("pulse dims to 0.35 at 600ms: %d", dim)
	}
}

func TestConditionBadgeStrobeLightsTheInnerBar(t *testing.T) {
	const w, h = 112, 84
	rest := drawConditionBadge(w, h, bandRed, 5)
	flash := drawConditionBadge(w, h, bandRed, 8)
	top := conditionBarRects(w, h, false)
	inner := top[0] // bottom wing reverses, so rect 0 is the inner bar
	if !lit(flash, inner) {
		t.Fatal("strobe left the inner bar dark")
	}
	if lit(rest, inner) {
		t.Fatal("the dead phase still lit a bar")
	}
	if conditionBandOf("severe") != bandRed || conditionBandOf("minor") != bandYellow || conditionBandOf("") != bandNone {
		t.Fatal("band map")
	}
}

func lit(img *image.NRGBA, rect image.Rectangle) bool {
	rect = rect.Intersect(img.Bounds())
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if c.R > 180 && c.A > 180 {
				return true
			}
		}
	}
	return false
}
