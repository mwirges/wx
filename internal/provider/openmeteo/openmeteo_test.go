package openmeteo_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/provider/openmeteo"
)

func TestProviderNameAndSupports(t *testing.T) {
	p := openmeteo.New()
	if p.Name() != "openmeteo" {
		t.Errorf("Name() = %q, want %q", p.Name(), "openmeteo")
	}

	tests := []struct {
		name string
		loc  location.Location
		want bool
	}{
		{"US location", location.Location{Lat: 41.0, Lon: -85.0, CountryCode: "US"}, true},
		{"Toronto Canada", location.Location{Lat: 43.7, Lon: -79.4, CountryCode: "CA"}, true},
		{"London UK", location.Location{Lat: 51.5, Lon: -0.12, CountryCode: "GB"}, true},
		{"Tokyo Japan", location.Location{Lat: 35.6, Lon: 139.6, CountryCode: "JP"}, true},
		{"Null Island (0,0)", location.Location{Lat: 0.0, Lon: 0.0}, false},
		{"Invalid latitude", location.Location{Lat: 95.0, Lon: 10.0}, false},
		{"Invalid longitude", location.Location{Lat: 10.0, Lon: 200.0}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := p.Supports(tc.loc)
			if got != tc.want {
				t.Errorf("Supports(%+v) = %v, want %v", tc.loc, got, tc.want)
			}
		})
	}
}

func TestCurrentConditions(t *testing.T) {
	fakeObsTime := time.Date(2026, 9, 27, 14, 0, 0, 0, time.UTC).Unix()
	mockResp := map[string]any{
		"latitude":              43.7,
		"longitude":             -79.4,
		"utc_offset_seconds":    -14400,
		"timezone":              "America/Toronto",
		"timezone_abbreviation": "EDT",
		"current": map[string]any{
			"time":                 fakeObsTime,
			"temperature_2m":        18.5,
			"relative_humidity_2m": 62.0,
			"dew_point_2m":         11.0,
			"apparent_temperature": 17.8,
			"is_day":               1,
			"weather_code":         61, // Slight rain
			"pressure_msl":         1015.2,
			"wind_speed_10m":       19.5,
			"wind_direction_10m":   330.0,
			"wind_gusts_10m":       32.0,
			"visibility":           25000.0,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			http.Error(w, "missing User-Agent", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer srv.Close()

	p := openmeteo.NewWithBaseURL(srv.URL)
	c := cache.NewNoOp()
	loc := location.Location{Lat: 43.7, Lon: -79.4, DisplayName: "Toronto, Ontario", CountryCode: "CA"}

	cond, err := p.CurrentConditions(context.Background(), loc, c)
	if err != nil {
		t.Fatalf("CurrentConditions failed: %v", err)
	}

	if cond.StationID != "openmeteo" {
		t.Errorf("StationID = %q, want %q", cond.StationID, "openmeteo")
	}
	if cond.Location != "Toronto, Ontario" {
		t.Errorf("Location = %q, want %q", cond.Location, "Toronto, Ontario")
	}
	if cond.TempC == nil || *cond.TempC != 18.5 {
		t.Errorf("TempC = %v, want 18.5", cond.TempC)
	}
	if cond.HumidityPct == nil || *cond.HumidityPct != 62.0 {
		t.Errorf("HumidityPct = %v, want 62.0", cond.HumidityPct)
	}
	if cond.DewPointC == nil || *cond.DewPointC != 11.0 {
		t.Errorf("DewPointC = %v, want 11.0", cond.DewPointC)
	}
	if cond.FeelsLikeC == nil || *cond.FeelsLikeC != 17.8 {
		t.Errorf("FeelsLikeC = %v, want 17.8", cond.FeelsLikeC)
	}
	if cond.PressureHPA == nil || *cond.PressureHPA != 1015.2 {
		t.Errorf("PressureHPA = %v, want 1015.2", cond.PressureHPA)
	}
	if cond.WindKPH == nil || *cond.WindKPH != 19.5 {
		t.Errorf("WindKPH = %v, want 19.5", cond.WindKPH)
	}
	if cond.WindGustKPH == nil || *cond.WindGustKPH != 32.0 {
		t.Errorf("WindGustKPH = %v, want 32.0", cond.WindGustKPH)
	}
	if cond.VisibilityM == nil || *cond.VisibilityM != 25000.0 {
		t.Errorf("VisibilityM = %v, want 25000.0", cond.VisibilityM)
	}
	if cond.ConditionCode != "rain" {
		t.Errorf("ConditionCode = %q, want %q", cond.ConditionCode, "rain")
	}
	if cond.Description != "Slight Rain" {
		t.Errorf("Description = %q, want %q", cond.Description, "Slight Rain")
	}
	if cond.Astronomy == nil {
		t.Error("Astronomy is nil, expected calculated ephemeris")
	}
}

func TestDailyForecast(t *testing.T) {
	now := time.Now().Truncate(24 * time.Hour)
	day0 := now.Unix()
	day1 := now.Add(24 * time.Hour).Unix()

	mockResp := map[string]any{
		"latitude":              43.7,
		"longitude":             -79.4,
		"utc_offset_seconds":    0,
		"timezone":              "UTC",
		"timezone_abbreviation": "UTC",
		"daily": map[string]any{
			"time":                         []int64{day0, day1},
			"weather_code":                 []int{0, 61},
			"temperature_2m_max":            []float64{22.0, 18.0},
			"temperature_2m_min":            []float64{12.0, 10.0},
			"precipitation_probability_max": []float64{5.0, 65.0},
			"wind_speed_10m_max":             []float64{15.0, 25.0},
			"wind_direction_10m_dominant":    []float64{180.0, 315.0},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer srv.Close()

	p := openmeteo.NewWithBaseURL(srv.URL)
	c := cache.NewNoOp()
	loc := location.Location{Lat: 43.7, Lon: -79.4, CountryCode: "CA"}

	fc, err := p.Forecast(context.Background(), loc, false, c)
	if err != nil {
		t.Fatalf("Daily forecast failed: %v", err)
	}

	if len(fc.Periods) < 2 {
		t.Fatalf("expected at least 2 periods, got %d", len(fc.Periods))
	}

	// Verify periods exist with reasonable attributes
	p0 := fc.Periods[0]
	if p0.Name == "" {
		t.Error("Period name is empty")
	}
	if p0.ShortDesc == "" {
		t.Error("Period ShortDesc is empty")
	}
	if p0.ProbabilityOfPrecipitation == nil {
		t.Error("ProbabilityOfPrecipitation is nil")
	}
}

func TestHourlyForecast(t *testing.T) {
	now := time.Now().Truncate(time.Hour)
	t0 := now.Unix()
	t1 := now.Add(time.Hour).Unix()
	t2 := now.Add(2 * time.Hour).Unix()

	mockResp := map[string]any{
		"latitude":              43.7,
		"longitude":             -79.4,
		"utc_offset_seconds":    0,
		"timezone":              "UTC",
		"timezone_abbreviation": "UTC",
		"hourly": map[string]any{
			"time":                      []int64{t0, t1, t2},
			"temperature_2m":            []float64{15.0, 16.0, 17.0},
			"relative_humidity_2m":       []float64{70.0, 65.0, 60.0},
			"dew_point_2m":              []float64{9.0, 9.5, 9.0},
			"apparent_temperature":      []float64{14.0, 15.0, 16.0},
			"precipitation_probability": []float64{10.0, 20.0, 0.0},
			"weather_code":              []int{0, 2, 3},
			"wind_speed_10m":            []float64{12.0, 14.0, 10.0},
			"wind_direction_10m":        []float64{90.0, 120.0, 150.0},
			"is_day":                    []int{1, 1, 1},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer srv.Close()

	p := openmeteo.NewWithBaseURL(srv.URL)
	c := cache.NewNoOp()
	loc := location.Location{Lat: 43.7, Lon: -79.4, CountryCode: "CA"}

	fc, err := p.Forecast(context.Background(), loc, true, c)
	if err != nil {
		t.Fatalf("Hourly forecast failed: %v", err)
	}

	if len(fc.Periods) != 3 {
		t.Fatalf("expected 3 hourly periods, got %d", len(fc.Periods))
	}

	hp0 := fc.Periods[0]
	if hp0.TempC != 15.0 {
		t.Errorf("TempC = %v, want 15.0", hp0.TempC)
	}
	if hp0.WindDir != "E" {
		t.Errorf("WindDir = %q, want %q", hp0.WindDir, "E")
	}
	if hp0.DewPointC == nil || *hp0.DewPointC != 9.0 {
		t.Errorf("DewPointC = %v, want 9.0", hp0.DewPointC)
	}
}

func TestAlerts(t *testing.T) {
	p := openmeteo.New()
	c := cache.NewNoOp()
	loc := location.Location{Lat: 43.7, Lon: -79.4, CountryCode: "CA"}

	alerts, err := p.Alerts(context.Background(), loc, c)
	if err != nil {
		t.Fatalf("Alerts failed: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alerts))
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := openmeteo.NewWithBaseURL(srv.URL)
	c := cache.NewNoOp()
	loc := location.Location{Lat: 43.7, Lon: -79.4, CountryCode: "CA"}

	_, err := p.CurrentConditions(context.Background(), loc, c)
	if err == nil {
		t.Error("expected error on HTTP 500, got nil")
	}

	_, err = p.Forecast(context.Background(), loc, false, c)
	if err == nil {
		t.Error("expected error on HTTP 500 for forecast, got nil")
	}
}

func TestCurrentConditions_Cache(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"latitude":  43.7,
			"longitude": -79.4,
			"current": map[string]any{
				"time":           time.Now().Unix(),
				"temperature_2m": 20.0,
			},
		})
	}))
	defer srv.Close()

	p := openmeteo.NewWithBaseURL(srv.URL)
	c, err := cache.NewWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("cache.NewWithDir failed: %v", err)
	}
	loc := location.Location{Lat: 43.7, Lon: -79.4, CountryCode: "CA"}

	// First call -> hits HTTP
	cond1, err := p.CurrentConditions(context.Background(), loc, c)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if requests != 1 {
		t.Fatalf("expected 1 request, got %d", requests)
	}

	// Second call -> hits cache
	cond2, err := p.CurrentConditions(context.Background(), loc, c)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if requests != 1 {
		t.Fatalf("expected still 1 request (cached), got %d", requests)
	}

	if *cond1.TempC != *cond2.TempC {
		t.Errorf("temp mismatch: %v vs %v", *cond1.TempC, *cond2.TempC)
	}
}

