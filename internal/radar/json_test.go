package radar

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/location"
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
		Raw:          true,
		BBox: &JSONBBox{
			MinLat: 39.0,
			MinLon: -86.0,
			MaxLat: 42.0,
			MaxLon: -83.0,
		},
		Center: &JSONCenter{
			Lat: 41.0,
			Lon: -85.0,
		},
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

	if !decoded.Raw {
		t.Error("expected raw to be true")
	}
	if decoded.BBox == nil || decoded.BBox.MinLat != 39.0 {
		t.Errorf("bbox = %v, want minLat 39.0", decoded.BBox)
	}
	if decoded.Center == nil || decoded.Center.Lat != 41.0 {
		t.Errorf("center = %v, want lat 41.0", decoded.Center)
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

func TestDescribeFrameOmitsImage(t *testing.T) {
	loc := location.Location{DisplayName: "Fort Wayne, IN", Lat: 41, Lon: -85}
	box := &BBox{MinLat: 40, MinLon: -85.001, MaxLat: 42, MaxLon: -85}
	frame := &Frame{
		ValidTime:   time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC),
		Product:     ProductCompositeReflectivity,
		IsComposite: true,
		BBox:        box,
	}
	opts := Options{Product: ProductCompositeReflectivity, RadiusKM: 200}
	out := DescribeFrame(loc, opts, "KIWX", frame)
	if out.ImageBase64 != "" || len(out.Frames) != 0 {
		t.Fatalf("image leaked: image %q frames %d", out.ImageBase64, len(out.Frames))
	}
	if out.RadiusKM != 200 || out.Station != "KIWX" || out.ProductLabel != "Composite Reflectivity" {
		t.Fatalf("meta = %+v", out)
	}
	if !out.IsComposite || out.ValidTime != "2026-10-31T18:00:00Z" {
		t.Fatalf("flags/time = %+v", out)
	}
	opts.BBox = box
	out = DescribeFrame(loc, opts, "", frame)
	if math.Abs(out.RadiusKM-111) > 0.01 {
		t.Fatalf("bbox radius = %v, want 111", out.RadiusKM)
	}
	if out.Center == nil || math.Abs(out.Center.Lat-41) > 0.0001 {
		t.Fatalf("center = %+v", out.Center)
	}
}
