package nws

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/radar"
)

// solidPNG returns a minimal valid PNG (4×4 pixels, all one colour).
func solidPNG(t *testing.T, r, g, b uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestBoundingBox(t *testing.T) {
	bb := boundingBox(39.1, -94.6, 200.0)
	if bb.MinLat >= bb.MaxLat {
		t.Error("MinLat must be < MaxLat")
	}
	if bb.MinLon >= bb.MaxLon {
		t.Error("MinLon must be < MaxLon")
	}
	// 200 km ÷ 111 km/° ≈ 1.8° latitude spread on each side.
	gotHalfLat := (bb.MaxLat - bb.MinLat) / 2
	wantHalfLat := 200.0 / 111.0
	if math.Abs(gotHalfLat-wantHalfLat) > 0.01 {
		t.Errorf("half-latitude = %.4f°, want ~%.4f°", gotHalfLat, wantHalfLat)
	}
}

func TestRadarProvider_CurrentFrame(t *testing.T) {
	pngData := solidPNG(t, 0, 200, 0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CurrentFrame now uses IEM (same as loop frames).
		layers := r.URL.Query()["layers[]"]
		if len(layers) == 0 {
			t.Errorf("expected layers[] params, got none")
		}
		foundNexrad := false
		for _, l := range layers {
			if l == "nexrad" {
				foundNexrad = true
			}
		}
		if !foundNexrad {
			t.Errorf("expected nexrad in layers[], got %v", layers)
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngData)
	}))
	defer srv.Close()

	p := &RadarProvider{wmsBase: srv.URL, iemBase: srv.URL, nwsAPIBase: srv.URL, imgCache: make(map[string]radarCacheEntry)}
	loc := location.Location{Lat: 39.1, Lon: -94.6, CountryCode: "US"}
	opts := radar.DefaultOptions()

	frame, err := p.CurrentFrame(context.Background(), loc, opts, cache.NewNoOp())
	if err != nil {
		t.Fatalf("CurrentFrame: %v", err)
	}
	if frame.Img == nil {
		t.Fatal("expected non-nil image")
	}
	if frame.Product != opts.Product {
		t.Errorf("product = %q, want %q", frame.Product, opts.Product)
	}
}

func TestRadarProvider_CurrentFrame_L1Cached(t *testing.T) {
	pngData := solidPNG(t, 255, 0, 0)
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngData)
	}))
	defer srv.Close()

	p := &RadarProvider{wmsBase: srv.URL, iemBase: srv.URL, nwsAPIBase: srv.URL, imgCache: make(map[string]radarCacheEntry)}
	loc := location.Location{Lat: 39.1, Lon: -94.6, CountryCode: "US"}
	opts := radar.DefaultOptions()
	c := cache.NewNoOp()

	if _, err := p.CurrentFrame(context.Background(), loc, opts, c); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := p.CurrentFrame(context.Background(), loc, opts, c); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if callCount != 1 {
		t.Errorf("HTTP calls = %d, want 1 (second should hit L1 in-process cache)", callCount)
	}
}

func TestRadarProvider_CurrentFrame_DiskCached(t *testing.T) {
	pngData := solidPNG(t, 0, 0, 255)
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngData)
	}))
	defer srv.Close()

	dir := t.TempDir()
	c, err := cache.NewWithDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	loc := location.Location{Lat: 39.1, Lon: -94.6, CountryCode: "US"}
	opts := radar.DefaultOptions()

	// First provider instance — fetches from network, writes disk cache.
	p1 := &RadarProvider{wmsBase: srv.URL, iemBase: srv.URL, nwsAPIBase: srv.URL, imgCache: make(map[string]radarCacheEntry)}
	if _, err := p1.CurrentFrame(context.Background(), loc, opts, c); err != nil {
		t.Fatalf("p1 CurrentFrame: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 HTTP call after p1, got %d", callCount)
	}

	// Second provider instance (simulates a new wx invocation) — should hit disk cache.
	p2 := &RadarProvider{wmsBase: srv.URL, iemBase: srv.URL, nwsAPIBase: srv.URL, imgCache: make(map[string]radarCacheEntry)}
	if _, err := p2.CurrentFrame(context.Background(), loc, opts, c); err != nil {
		t.Fatalf("p2 CurrentFrame: %v", err)
	}
	if callCount != 1 {
		t.Errorf("HTTP calls = %d after p2, want 1 (should hit disk cache)", callCount)
	}
}

func TestRadarProvider_Supports(t *testing.T) {
	p := newRadarProvider()
	us := location.Location{CountryCode: "US"}
	uk := location.Location{CountryCode: "GB"}
	if !p.Supports(us) {
		t.Error("should support US")
	}
	if p.Supports(uk) {
		t.Error("should not support GB")
	}
}

func TestRadarProvider_UnsupportedProduct(t *testing.T) {
	p := newRadarProvider()
	loc := location.Location{Lat: 39.1, Lon: -94.6, CountryCode: "US"}
	_, err := p.CurrentFrame(context.Background(), loc, radar.Options{Product: "velocity"}, cache.NewNoOp())
	if err == nil {
		t.Error("expected error for unsupported product")
	}
}

func TestRadarProvider_RecentFrames(t *testing.T) {
	pngData := solidPNG(t, 0, 100, 200)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngData)
	}))
	defer srv.Close()

	p := &RadarProvider{wmsBase: srv.URL, iemBase: srv.URL, nwsAPIBase: srv.URL, imgCache: make(map[string]radarCacheEntry)}
	loc := location.Location{Lat: 39.1, Lon: -94.6, CountryCode: "US"}
	opts := radar.DefaultOptions()

	frames, err := p.RecentFrames(context.Background(), loc, opts, 3, cache.NewNoOp())
	if err != nil {
		t.Fatalf("RecentFrames: %v", err)
	}
	if len(frames) == 0 {
		t.Error("expected at least one frame")
	}
	// Frames must be in chronological order.
	for i := 1; i < len(frames); i++ {
		if !frames[i].ValidTime.After(frames[i-1].ValidTime) {
			t.Errorf("frame %d ValidTime not after frame %d", i, i-1)
		}
	}
}

func TestRadarProvider_TimestampFormat(t *testing.T) {
	// Verify the IEM timestamp format is YYYYMMDDHHMI.
	ts := time.Date(2026, 3, 22, 18, 5, 0, 0, time.UTC)
	got := ts.UTC().Format("200601021504")
	want := "202603221805"
	if got != want {
		t.Errorf("timestamp format: got %q, want %q", got, want)
	}
}

func TestRadarProvider_LookupStation(t *testing.T) {
	const geoJSON = `{
		"properties": {"stationIdentifier":"KIWX","name":"North Webster / South Bend"},
		"geometry":   {"coordinates":[-85.7,41.36,298.0]}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/radar/stations/KIWX" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/geo+json")
		w.Write([]byte(geoJSON))
	}))
	defer srv.Close()

	p := &RadarProvider{
		wmsBase:    srv.URL,
		iemBase:    srv.URL,
		nwsAPIBase: srv.URL,
		imgCache:   make(map[string]radarCacheEntry),
	}

	st, err := p.LookupStation(context.Background(), "kiwx", cache.NewNoOp())
	if err != nil {
		t.Fatalf("LookupStation: %v", err)
	}
	if st.ID != "KIWX" {
		t.Errorf("ID = %q, want KIWX", st.ID)
	}
	if st.Name != "North Webster / South Bend" {
		t.Errorf("Name = %q", st.Name)
	}
	if st.Lat != 41.36 {
		t.Errorf("Lat = %v, want 41.36", st.Lat)
	}
	if st.Lon != -85.7 {
		t.Errorf("Lon = %v, want -85.7", st.Lon)
	}
}

func TestRadarProvider_LookupStation_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	p := &RadarProvider{
		wmsBase:    srv.URL,
		iemBase:    srv.URL,
		nwsAPIBase: srv.URL,
		imgCache:   make(map[string]radarCacheEntry),
	}

	_, err := p.LookupStation(context.Background(), "KZZZ", cache.NewNoOp())
	if err == nil {
		t.Error("expected error for unknown station")
	}
}

func TestRadarProvider_LookupStation_Cached(t *testing.T) {
	const geoJSON = `{
		"properties": {"stationIdentifier":"KMKE","name":"Milwaukee"},
		"geometry":   {"coordinates":[-87.93,42.95,206.0]}
	}`
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/geo+json")
		w.Write([]byte(geoJSON))
	}))
	defer srv.Close()

	dir := t.TempDir()
	c, err := cache.NewWithDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	p := &RadarProvider{
		wmsBase:    srv.URL,
		iemBase:    srv.URL,
		nwsAPIBase: srv.URL,
		imgCache:   make(map[string]radarCacheEntry),
	}

	if _, err := p.LookupStation(context.Background(), "KMKE", c); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := p.LookupStation(context.Background(), "KMKE", c); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if callCount != 1 {
		t.Errorf("HTTP calls = %d, want 1 (second should hit disk cache)", callCount)
	}
}
