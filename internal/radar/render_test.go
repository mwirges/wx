package radar

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"time"
)

func TestScaleImage_Dimensions(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	dst := scaleImage(src, 20, 10)
	b := dst.Bounds()
	if b.Dx() != 20 || b.Dy() != 10 {
		t.Errorf("scaleImage: got %dx%d, want 20x10", b.Dx(), b.Dy())
	}
}

func TestScaleImage_ColorPreserved(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	want := color.RGBA{200, 100, 50, 255}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			src.Set(x, y, want)
		}
	}
	dst := scaleImage(src, 2, 2)
	got := dst.At(0, 0).(color.RGBA)
	if got != want {
		t.Errorf("color preserved: got %v, want %v", got, want)
	}
}

func TestScaleImage_TransparentToBlack(t *testing.T) {
	// Default image.RGBA is all zeros (transparent).
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	dst := scaleImage(src, 4, 4)
	got := dst.At(0, 0).(color.RGBA)
	if got.R != 0 || got.G != 0 || got.B != 0 || got.A != 255 {
		t.Errorf("transparent pixel: got %v, want opaque black", got)
	}
}

func TestProductLabel(t *testing.T) {
	cases := []struct {
		p    Product
		want string
	}{
		{ProductCompositeReflectivity, "Composite Reflectivity"},
		{ProductBaseReflectivity, "Base Reflectivity"},
		{"custom-product", "custom-product"},
	}
	for _, tc := range cases {
		if got := ProductLabel(tc.p); got != tc.want {
			t.Errorf("ProductLabel(%q) = %q, want %q", tc.p, got, tc.want)
		}
	}
}

func TestRenderFrame_LineCount(t *testing.T) {
	// Build a small solid-green image.
	img := image.NewRGBA(image.Rect(0, 0, 80, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 80; x++ {
			img.Set(x, y, color.RGBA{0, 200, 0, 255})
		}
	}
	frame := &Frame{
		Img:       img,
		ValidTime: time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC),
		Product:   ProductCompositeReflectivity,
	}
	opts := RenderOptions{TermWidth: 80, TermHeight: 24}

	var sb strings.Builder
	if err := RenderFrame(&sb, frame, "Kansas City, MO", opts); err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	lines := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")
	// imgH = (termH - header - footer) * 2  →  imgH/2 image rows rendered.
	// Total = header(2) + imgH/2 + footer(1) = termH.
	wantLines := opts.TermHeight
	if len(lines) != wantLines {
		t.Errorf("output lines = %d, want %d", len(lines), wantLines)
	}
}

func TestRenderFrame_TooSmall(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	frame := &Frame{Img: img, ValidTime: time.Now(), Product: ProductCompositeReflectivity}
	err := RenderFrame(nil, frame, "Loc", RenderOptions{TermWidth: 5, TermHeight: 3})
	if err == nil {
		t.Error("expected error for too-small terminal")
	}
}
