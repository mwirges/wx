package nowcast

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/models"
)

func TestSparklineRune(t *testing.T) {
	tests := []struct {
		rate     float64
		expected rune
	}{
		{0.0, ' '},
		{0.001, ' '},
		{0.02, ' '},
		{0.05, '▂'},
		{0.10, '▃'},
		{0.20, '▄'},
		{0.35, '▅'},
		{0.50, '▆'},
		{0.75, '▇'},
		{1.20, '█'},
	}

	for _, tt := range tests {
		got := SparklineRune(tt.rate)
		if got != tt.expected {
			t.Errorf("SparklineRune(%f) = %c, want %c", tt.rate, got, tt.expected)
		}
	}
}

func TestParseNowcast_Dry(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	var resp openMeteoResponse

	// Generate 16 15-minute intervals (4 hours) of 0 precip
	for i := 0; i < 16; i++ {
		step := now.Add(time.Duration(i*15) * time.Minute)
		resp.Minutely15.Time = append(resp.Minutely15.Time, step.Format("2006-01-02T15:04"))
		resp.Minutely15.Precipitation = append(resp.Minutely15.Precipitation, 0.0)
		resp.Minutely15.Rain = append(resp.Minutely15.Rain, 0.0)
		resp.Minutely15.Snowfall = append(resp.Minutely15.Snowfall, 0.0)
		resp.Minutely15.WeatherCode = append(resp.Minutely15.WeatherCode, 0)
	}

	n := ParseNowcast(resp, "Fort Wayne, IN", now)
	if n == nil {
		t.Fatal("expected non-nil nowcast")
	}
	if n.IsActivePrecip {
		t.Errorf("expected IsActivePrecip = false, got true")
	}
	if n.TotalLiquidMM != 0 {
		t.Errorf("expected TotalLiquidMM = 0, got %f", n.TotalLiquidMM)
	}
	if !strings.Contains(n.Headline, "Clear") {
		t.Errorf("expected headline to mention Clear, got %q", n.Headline)
	}
	if n.PrimaryPhase != models.PhaseNone {
		t.Errorf("expected PrimaryPhase none, got %v", n.PrimaryPhase)
	}
}

func TestParseNowcast_ActiveRainStopping(t *testing.T) {
	now := time.Date(2026, 9, 27, 14, 0, 0, 0, time.UTC)
	var resp openMeteoResponse

	// Active rain for first 2 intervals (30 min: 14:00, 14:15), then stops at 14:30
	for i := 0; i < 8; i++ {
		step := now.Add(time.Duration(i*15) * time.Minute)
		resp.Minutely15.Time = append(resp.Minutely15.Time, step.Format("2006-01-02T15:04"))
		if i < 2 {
			resp.Minutely15.Precipitation = append(resp.Minutely15.Precipitation, 1.5) // 6.0 mm/h
			resp.Minutely15.Rain = append(resp.Minutely15.Rain, 1.5)
			resp.Minutely15.Snowfall = append(resp.Minutely15.Snowfall, 0.0)
			resp.Minutely15.WeatherCode = append(resp.Minutely15.WeatherCode, 61)
		} else {
			resp.Minutely15.Precipitation = append(resp.Minutely15.Precipitation, 0.0)
			resp.Minutely15.Rain = append(resp.Minutely15.Rain, 0.0)
			resp.Minutely15.Snowfall = append(resp.Minutely15.Snowfall, 0.0)
			resp.Minutely15.WeatherCode = append(resp.Minutely15.WeatherCode, 3)
		}
	}

	n := ParseNowcast(resp, "Chicago, IL", now)
	if n == nil {
		t.Fatal("expected non-nil nowcast")
	}
	if !n.IsActivePrecip {
		t.Errorf("expected IsActivePrecip = true, got false")
	}
	if n.PrecipEndTime == nil {
		t.Fatal("expected PrecipEndTime to be non-nil")
	}
	if !strings.Contains(n.Headline, "stopping in ~30 min") {
		t.Errorf("expected headline to note stopping in ~30 min, got %q", n.Headline)
	}
	if n.PrimaryPhase != models.PhaseRain {
		t.Errorf("expected PrimaryPhase = rain, got %v", n.PrimaryPhase)
	}
	if n.PeakRateMMH != 6.0 {
		t.Errorf("expected PeakRateMMH = 6.0, got %f", n.PeakRateMMH)
	}
}

func TestParseNowcast_RainStarting(t *testing.T) {
	now := time.Date(2026, 9, 27, 16, 0, 0, 0, time.UTC)
	var resp openMeteoResponse

	// Dry for 16:00, 16:15, then starts at 16:30 (+30m)
	for i := 0; i < 8; i++ {
		step := now.Add(time.Duration(i*15) * time.Minute)
		resp.Minutely15.Time = append(resp.Minutely15.Time, step.Format("2006-01-02T15:04"))
		if i >= 2 {
			resp.Minutely15.Precipitation = append(resp.Minutely15.Precipitation, 2.0)
			resp.Minutely15.Rain = append(resp.Minutely15.Rain, 2.0)
			resp.Minutely15.Snowfall = append(resp.Minutely15.Snowfall, 0.0)
			resp.Minutely15.WeatherCode = append(resp.Minutely15.WeatherCode, 63)
		} else {
			resp.Minutely15.Precipitation = append(resp.Minutely15.Precipitation, 0.0)
			resp.Minutely15.Rain = append(resp.Minutely15.Rain, 0.0)
			resp.Minutely15.Snowfall = append(resp.Minutely15.Snowfall, 0.0)
			resp.Minutely15.WeatherCode = append(resp.Minutely15.WeatherCode, 2)
		}
	}

	n := ParseNowcast(resp, "Indianapolis, IN", now)
	if n == nil {
		t.Fatal("expected non-nil nowcast")
	}
	if n.IsActivePrecip {
		t.Errorf("expected IsActivePrecip = false, got true")
	}
	if n.NextPrecipTime == nil {
		t.Fatal("expected NextPrecipTime to be non-nil")
	}
	if !strings.Contains(n.Headline, "starting in ~30 min") {
		t.Errorf("expected headline to mention starting in ~30 min, got %q", n.Headline)
	}
}

func TestParseNowcast_Snowfall(t *testing.T) {
	now := time.Date(2026, 12, 15, 8, 0, 0, 0, time.UTC)
	var resp openMeteoResponse

	for i := 0; i < 6; i++ {
		step := now.Add(time.Duration(i*15) * time.Minute)
		resp.Minutely15.Time = append(resp.Minutely15.Time, step.Format("2006-01-02T15:04"))
		resp.Minutely15.Precipitation = append(resp.Minutely15.Precipitation, 0.8)
		resp.Minutely15.Rain = append(resp.Minutely15.Rain, 0.0)
		resp.Minutely15.Snowfall = append(resp.Minutely15.Snowfall, 1.2) // 1.2 cm per 15 min
		resp.Minutely15.WeatherCode = append(resp.Minutely15.WeatherCode, 73)
	}

	n := ParseNowcast(resp, "Minneapolis, MN", now)
	if n.PrimaryPhase != models.PhaseSnow {
		t.Errorf("expected primary phase snow, got %v", n.PrimaryPhase)
	}
	if n.TotalSnowCM <= 0 {
		t.Errorf("expected TotalSnowCM > 0, got %f", n.TotalSnowCM)
	}
	if n.TotalSnowIn <= 0 {
		t.Errorf("expected TotalSnowIn > 0, got %f", n.TotalSnowIn)
	}
	if !strings.Contains(n.Headline, "Snow") {
		t.Errorf("expected headline to mention Snow, got %q", n.Headline)
	}
}

func TestClient_Fetch(t *testing.T) {
	now := time.Now().UTC()
	var mockResp openMeteoResponse
	for i := 0; i < 8; i++ {
		step := now.Add(time.Duration(i*15) * time.Minute)
		mockResp.Minutely15.Time = append(mockResp.Minutely15.Time, step.Format("2006-01-02T15:04"))
		mockResp.Minutely15.Precipitation = append(mockResp.Minutely15.Precipitation, 0.5)
		mockResp.Minutely15.Rain = append(mockResp.Minutely15.Rain, 0.5)
		mockResp.Minutely15.Snowfall = append(mockResp.Minutely15.Snowfall, 0.0)
		mockResp.Minutely15.WeatherCode = append(mockResp.Minutely15.WeatherCode, 61)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer ts.Close()

	c := cache.NewNoOp()
	client := NewClient(ts.URL)

	n, err := client.Fetch(context.Background(), 41.0, -85.0, "Fort Wayne, IN", c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n == nil {
		t.Fatal("expected non-nil nowcast")
	}
	if len(n.Intervals) == 0 {
		t.Errorf("expected intervals, got 0")
	}

	// Test cache hit with an isolated temp cache
	tempCache, err := cache.NewWithDir(t.TempDir())
	if err == nil {
		calls := 0
		tsCount := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mockResp)
		}))
		defer tsCount.Close()

		cl := NewClient(tsCount.URL)
		_, err = cl.Fetch(context.Background(), 40.0, -80.0, "Pittsburgh, PA", tempCache)
		if err != nil {
			t.Fatalf("first fetch failed: %v", err)
		}
		if calls != 1 {
			t.Errorf("expected 1 call, got %d", calls)
		}

		// Second fetch should hit cache
		_, err = cl.Fetch(context.Background(), 40.0, -80.0, "Pittsburgh, PA", tempCache)
		if err != nil {
			t.Fatalf("second fetch failed: %v", err)
		}
		if calls != 1 {
			t.Errorf("expected still 1 call (cache hit), got %d", calls)
		}
	}
}
