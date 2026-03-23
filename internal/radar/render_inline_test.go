package radar

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func solidTestImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{0, 200, 0, 255})
		}
	}
	return img
}

func TestRenderInlineImage_ITerm2(t *testing.T) {
	img := solidTestImage(4, 4)
	var sb strings.Builder
	if err := renderInlineImage(&sb, img, 80, 20, TermITerm2); err != nil {
		t.Fatalf("renderInlineImage iTerm2: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "\x1b]1337;File=") {
		t.Error("expected iTerm2 escape sequence in output")
	}
	if !strings.Contains(out, "inline=1") {
		t.Error("expected inline=1 in output")
	}
}

func TestRenderInlineImage_Kitty(t *testing.T) {
	img := solidTestImage(4, 4)
	var sb strings.Builder
	if err := renderInlineImage(&sb, img, 80, 20, TermKitty); err != nil {
		t.Fatalf("renderInlineImage Kitty: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "\x1b_Ga=T,f=100") {
		t.Error("expected Kitty graphics escape in output")
	}
}

func TestRenderInlineImage_UnsupportedMode(t *testing.T) {
	img := solidTestImage(4, 4)
	var sb strings.Builder
	err := renderInlineImage(&sb, img, 80, 20, TermHalfBlock)
	if err == nil {
		t.Error("expected error for unsupported mode")
	}
}
