package radar

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
)

func TestEncodeGIF(t *testing.T) {
	// Create two simple test frames
	f1 := image.NewRGBA(image.Rect(0, 0, 40, 40))
	for x := 0; x < 40; x++ {
		for y := 0; y < 40; y++ {
			f1.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	f2 := image.NewRGBA(image.Rect(0, 0, 40, 40))
	for x := 0; x < 40; x++ {
		for y := 0; y < 40; y++ {
			f2.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	data, err := EncodeGIF([]image.Image{f1, f2}, 400)
	if err != nil {
		t.Fatalf("EncodeGIF failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("expected non-empty GIF data")
	}

	// Verify GIF magic header
	if !bytes.HasPrefix(data, []byte("GIF89a")) && !bytes.HasPrefix(data, []byte("GIF87a")) {
		t.Errorf("expected GIF header prefix, got %q", string(data[:6]))
	}

	// Verify decoding
	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to decode generated GIF: %v", err)
	}

	if len(decoded.Image) != 2 {
		t.Errorf("expected 2 frames, got %d", len(decoded.Image))
	}

	if decoded.Delay[0] != 40 {
		t.Errorf("expected frame 0 delay 40, got %d", decoded.Delay[0])
	}
	if decoded.Delay[1] != 80 { // 2x dwell on final frame
		t.Errorf("expected frame 1 dwell delay 80, got %d", decoded.Delay[1])
	}
}

func TestEncodeGIF_Empty(t *testing.T) {
	_, err := EncodeGIF(nil, 400)
	if err == nil {
		t.Error("expected error for empty images slice, got nil")
	}
}
