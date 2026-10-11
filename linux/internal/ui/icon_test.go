package ui

import (
	"bytes"
	"image/png"
	"testing"
)

func TestAppIconIsPNG(t *testing.T) {
	icon := AppIcon()
	if icon.Name() != "wx.png" {
		t.Fatalf("name %q", icon.Name())
	}
	img, err := png.Decode(bytes.NewReader(icon.Content()))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() < 64 || b.Dy() < 64 {
		t.Fatalf("size %dx%d", b.Dx(), b.Dy())
	}
}
