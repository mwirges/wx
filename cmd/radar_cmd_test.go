package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/radar"
)

type mockRadarProvider struct {
	frame *radar.Frame
}

func (m *mockRadarProvider) Name() string { return "mock" }
func (m *mockRadarProvider) Supports(loc location.Location) bool { return true }
func (m *mockRadarProvider) CurrentFrame(ctx context.Context, loc location.Location, opts radar.Options, c *cache.Cache) (*radar.Frame, error) {
	return m.frame, nil
}
func (m *mockRadarProvider) RecentFrames(ctx context.Context, loc location.Location, opts radar.Options, n int, c *cache.Cache) ([]*radar.Frame, error) {
	return []*radar.Frame{m.frame}, nil
}

func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.Bytes()
}

func TestRunRadarJSON(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(0, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})

	frame := &radar.Frame{
		Img:       img,
		ValidTime: time.Date(2026, 9, 25, 20, 0, 0, 0, time.UTC),
		Product:   radar.ProductCompositeReflectivity,
		BBox: &radar.BBox{
			MinLat: 40.0,
			MinLon: -86.0,
			MaxLat: 42.0,
			MaxLon: -84.0,
		},
	}
	prov := &mockRadarProvider{frame: frame}
	loc := location.Location{
		DisplayName: "Fort Wayne, IN",
		Lat:         41.0,
		Lon:         -85.0,
		CountryCode: "US",
	}
	opts := radar.Options{
		Product:  radar.ProductCompositeReflectivity,
		RadiusKM: 200,
	}

	outBytes := captureStdout(t, func() {
		err := runRadarJSON(context.Background(), prov, loc, opts, "KIWX", false, false, 1, cache.NewNoOp())
		if err != nil {
			t.Fatalf("runRadarJSON: %v", err)
		}
	})

	var res radar.JSONRadarOutput
	if err := json.Unmarshal(outBytes, &res); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if res.Product != string(radar.ProductCompositeReflectivity) {
		t.Errorf("product = %q, want %q", res.Product, radar.ProductCompositeReflectivity)
	}
	if res.Location != "Fort Wayne, IN" {
		t.Errorf("location = %q, want 'Fort Wayne, IN'", res.Location)
	}
	if res.Station != "KIWX" {
		t.Errorf("station = %q, want 'KIWX'", res.Station)
	}
	if res.ImageBase64 == "" {
		t.Error("expected non-empty image_base64")
	}
	if len(res.Frames) != 1 {
		t.Fatalf("len(frames) = %d, want 1", len(res.Frames))
	}
}

func TestRunRadarJSON_Loop(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	frame := &radar.Frame{
		Img:       img,
		ValidTime: time.Date(2026, 9, 25, 20, 0, 0, 0, time.UTC),
		Product:   radar.ProductBaseReflectivity,
	}
	prov := &mockRadarProvider{frame: frame}
	loc := location.Location{DisplayName: "Chicago, IL", Lat: 41.8, Lon: -87.6, CountryCode: "US"}
	opts := radar.Options{Product: radar.ProductBaseReflectivity, RadiusKM: 150}

	outBytes := captureStdout(t, func() {
		err := runRadarJSON(context.Background(), prov, loc, opts, "", true, true, 2, cache.NewNoOp())
		if err != nil {
			t.Fatalf("runRadarJSON loop: %v", err)
		}
	})

	var res radar.JSONRadarOutput
	if err := json.Unmarshal(outBytes, &res); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if res.Product != string(radar.ProductBaseReflectivity) {
		t.Errorf("product = %q, want %q", res.Product, radar.ProductBaseReflectivity)
	}
	if len(res.Frames) != 1 {
		t.Errorf("len(frames) = %d, want 1", len(res.Frames))
	}
}
