package airquality

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/models"
)

const sampleAQIResponse = `{
  "latitude": 41.1,
  "longitude": -85.1,
  "current": {
    "time": "2026-09-27T22:00",
    "us_aqi": 37,
    "uv_index": 4.5,
    "pm2_5": 2.4,
    "pm10": 5.1,
    "ozone": 81.0,
    "nitrogen_dioxide": 3.7,
    "carbon_monoxide": 151.0,
    "sulphur_dioxide": 0.8
  }
}`

func TestFetch_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleAQIResponse))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	c := cache.NewNoOp()

	aq, err := client.Fetch(context.Background(), 41.0793, -85.1394, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if aq == nil {
		t.Fatal("expected non-nil AirQuality")
	}
	if aq.AQI == nil || *aq.AQI != 37 {
		t.Errorf("expected AQI 37, got %v", aq.AQI)
	}
	if aq.Category != "Good" {
		t.Errorf("expected category 'Good', got %q", aq.Category)
	}
	if aq.UVIndex == nil || *aq.UVIndex != 4.5 {
		t.Errorf("expected UVIndex 4.5, got %v", aq.UVIndex)
	}
	if aq.UVCategory != "Moderate" {
		t.Errorf("expected UVCategory 'Moderate', got %q", aq.UVCategory)
	}
	if aq.PM25 == nil || *aq.PM25 != 2.4 {
		t.Errorf("expected PM25 2.4, got %v", aq.PM25)
	}
}

func TestCategories(t *testing.T) {
	tests := []struct {
		aqi  int
		want string
	}{
		{25, "Good"},
		{75, "Moderate"},
		{125, "Unhealthy for Sensitive Groups"},
		{175, "Unhealthy"},
		{250, "Very Unhealthy"},
		{400, "Hazardous"},
	}
	for _, tt := range tests {
		got := models.AQICategory(tt.aqi)
		if got != tt.want {
			t.Errorf("AQICategory(%d) = %q, want %q", tt.aqi, got, tt.want)
		}
	}

	uvTests := []struct {
		uv   float64
		want string
	}{
		{1.5, "Low"},
		{4.0, "Moderate"},
		{6.5, "High"},
		{9.0, "Very High"},
		{12.0, "Extreme"},
	}
	for _, tt := range uvTests {
		got := models.UVCategory(tt.uv)
		if got != tt.want {
			t.Errorf("UVCategory(%v) = %q, want %q", tt.uv, got, tt.want)
		}
	}
}
