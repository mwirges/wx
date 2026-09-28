package history

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mwirges/wx/internal/cache"
)

const sampleHistoryResponse = `{
  "latitude": 41.07,
  "longitude": -85.02,
  "elevation": 232.0,
  "daily": {
    "time": ["2026-09-25", "2026-09-26"],
    "temperature_2m_max": [19.6, 21.4],
    "temperature_2m_min": [8.1, 8.0],
    "apparent_temperature_max": [18.7, 20.5],
    "apparent_temperature_min": [6.7, 6.4],
    "precipitation_sum": [0.0, 5.2],
    "weather_code": [3, 61],
    "wind_speed_10m_max": [12.3, 18.5]
  }
}`

func TestFetch_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleHistoryResponse))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	c := cache.NewNoOp()

	h, err := client.Fetch(context.Background(), 41.07, -85.02, 7, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if h == nil {
		t.Fatal("expected non-nil HistoricalWeather")
	}
	if len(h.Days) != 2 {
		t.Fatalf("expected 2 days, got %d", len(h.Days))
	}

	d1 := h.Days[0]
	if d1.Date.Format("2006-01-02") != "2026-09-25" {
		t.Errorf("expected date 2026-09-25, got %s", d1.Date.Format("2006-01-02"))
	}
	if d1.TempMaxC == nil || *d1.TempMaxC != 19.6 {
		t.Errorf("expected TempMaxC 19.6, got %v", d1.TempMaxC)
	}
	if d1.TempMinC == nil || *d1.TempMinC != 8.1 {
		t.Errorf("expected TempMinC 8.1, got %v", d1.TempMinC)
	}
	if d1.ConditionCode != "cloudy" {
		t.Errorf("expected ConditionCode cloudy, got %q", d1.ConditionCode)
	}
	if d1.Description != "Overcast" {
		t.Errorf("expected Description Overcast, got %q", d1.Description)
	}

	d2 := h.Days[1]
	if d2.ConditionCode != "rain" {
		t.Errorf("expected ConditionCode rain, got %q", d2.ConditionCode)
	}
	if d2.PrecipSumMM == nil || *d2.PrecipSumMM != 5.2 {
		t.Errorf("expected PrecipSumMM 5.2, got %v", d2.PrecipSumMM)
	}

	// Verify Summary
	s := h.Summary
	if s.DaysCount != 2 {
		t.Errorf("expected DaysCount 2, got %d", s.DaysCount)
	}
	if s.AvgTempMaxC == nil || *s.AvgTempMaxC != 20.5 {
		t.Errorf("expected AvgTempMaxC 20.5, got %v", s.AvgTempMaxC)
	}
	if s.AvgTempMinC == nil || *s.AvgTempMinC != 8.05 {
		t.Errorf("expected AvgTempMinC 8.05, got %v", s.AvgTempMinC)
	}
	if s.TotalPrecipMM == nil || *s.TotalPrecipMM != 5.2 {
		t.Errorf("expected TotalPrecipMM 5.2, got %v", s.TotalPrecipMM)
	}
	if s.MaxWindKPH == nil || *s.MaxWindKPH != 18.5 {
		t.Errorf("expected MaxWindKPH 18.5, got %v", s.MaxWindKPH)
	}
}

func TestFetch_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	c := cache.NewNoOp()

	_, err := client.Fetch(context.Background(), 41.07, -85.02, 7, c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
