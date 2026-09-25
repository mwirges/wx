package radar

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestEncodePNGAndBase64(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	raw, err := EncodePNG(img)
	if err != nil {
		t.Fatalf("EncodePNG failed: %v", err)
	}

	decoded, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("png.Decode failed: %v", err)
	}
	if decoded.Bounds().Dx() != 10 || decoded.Bounds().Dy() != 10 {
		t.Errorf("bounds = %v, want 10x10", decoded.Bounds())
	}

	b64, err := EncodeBase64PNG(img)
	if err != nil {
		t.Fatalf("EncodeBase64PNG failed: %v", err)
	}
	fromB64, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("DecodeString failed: %v", err)
	}
	if !bytes.Equal(raw, fromB64) {
		t.Error("base64 payload does not match raw PNG bytes")
	}
}

func TestRenderJSON(t *testing.T) {
	out := JSONRadarOutput{
		Product:      string(ProductCompositeReflectivity),
		ProductLabel: "Composite Reflectivity",
		Location:     "Fort Wayne, IN",
		Station:      "KIWX",
		ValidTime:    "2026-09-25T18:00:00Z",
		ImageBase64:  "dGVzdA==",
		RadiusKM:     200,
		Frames: []JSONFrame{
			{ValidTime: "2026-09-25T18:00:00Z", ImageBase64: "dGVzdA=="},
		},
	}

	var buf bytes.Buffer
	if err := RenderJSON(&buf, out); err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}

	var decoded JSONRadarOutput
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Product != out.Product {
		t.Errorf("product = %q, want %q", decoded.Product, out.Product)
	}
	if decoded.ProductLabel != out.ProductLabel {
		t.Errorf("product_label = %q, want %q", decoded.ProductLabel, out.ProductLabel)
	}
	if decoded.Location != out.Location {
		t.Errorf("location = %q, want %q", decoded.Location, out.Location)
	}
	if decoded.Station != out.Station {
		t.Errorf("station = %q, want %q", decoded.Station, out.Station)
	}
	if decoded.ImageBase64 != out.ImageBase64 {
		t.Errorf("image_base64 = %q, want %q", decoded.ImageBase64, out.ImageBase64)
	}
	if len(decoded.Frames) != 1 {
		t.Errorf("len(frames) = %d, want 1", len(decoded.Frames))
	}
}
