package nws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
)

// newTestProvider returns a Provider whose HTTP client points at baseURL.
func newTestProvider(baseURL string) *Provider {
	p := New()
	p.client.baseURL = baseURL
	return p
}

// mockNWSServer sets up a minimal NWS API mock covering:
//   /points/{lat},{lon}         → grid info + stations URL
//   /gridpoints/.../stations    → station list
//   /stations/{id}/observations/latest → observation
//   /gridpoints/.../forecast    → forecast
//   /alerts/active              → alerts
func mockNWSServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// /points/
	mux.HandleFunc("/points/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"properties": map[string]any{
				"gridId":              "EAX",
				"gridX":               32,
				"gridY":               50,
				"forecast":            "http://" + r.Host + "/gridpoints/EAX/32,50/forecast",
				"forecastHourly":      "http://" + r.Host + "/gridpoints/EAX/32,50/forecast/hourly",
				"observationStations": "http://" + r.Host + "/gridpoints/EAX/32,50/stations",
				"relativeLocation": map[string]any{
					"properties": map[string]any{
						"city":  "Kansas City",
						"state": "MO",
					},
				},
			},
		})
	})

	// stations list
	mux.HandleFunc("/gridpoints/EAX/32,50/stations", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"features": []map[string]any{
				{
					"properties": map[string]any{
						"stationIdentifier": "KMKC",
						"name":              "Kansas City Downtown Airport",
					},
				},
			},
		})
	})

	// latest observation
	tempC := 17.0
	windKPH := 14.4
	windDeg := 270.0
	humidity := 45.0
	dewPointC := 6.0
	windGustKPH := 25.0
	pressurePa := 101500.0 // 1015 hPa
	visibilityM := 16093.0 // ~10 mi
	windChillC := 14.0
	mux.HandleFunc("/stations/KMKC/observations/latest", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"properties": map[string]any{
				"station":          "KMKC",
				"timestamp":        "2026-03-22T20:00:00Z",
				"textDescription":  "Clear",
				"icon":             "https://api.weather.gov/icons/land/day/skc?size=medium",
				"temperature":      map[string]any{"value": tempC, "unitCode": "wmoUnit:degC"},
				"windSpeed":        map[string]any{"value": windKPH, "unitCode": "wmoUnit:km_h-1"},
				"windDirection":    map[string]any{"value": windDeg, "unitCode": "wmoUnit:degree_(angle)"},
				"windGust":         map[string]any{"value": windGustKPH, "unitCode": "wmoUnit:km_h-1"},
				"relativeHumidity": map[string]any{"value": humidity, "unitCode": "wmoUnit:percent"},
				"dewpoint":         map[string]any{"value": dewPointC, "unitCode": "wmoUnit:degC"},
				"windChill":        map[string]any{"value": windChillC, "unitCode": "wmoUnit:degC"},
				"heatIndex":        map[string]any{"value": nil, "unitCode": "wmoUnit:degC"},
				"seaLevelPressure": map[string]any{"value": pressurePa, "unitCode": "wmoUnit:Pa"},
				"visibility":       map[string]any{"value": visibilityM, "unitCode": "wmoUnit:m"},
			},
		})
	})

	// forecast
	mux.HandleFunc("/gridpoints/EAX/32,50/forecast", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"properties": map[string]any{
				"generatedAt": "2026-03-22T18:00:00Z",
				"periods": []map[string]any{
					{
						"name":             "Tonight",
						"startTime":        "2026-03-22T18:00:00-05:00",
						"endTime":          "2026-03-23T06:00:00-05:00",
						"isDaytime":        false,
						"temperature":      38,
						"temperatureUnit":  "F",
						"windSpeed":        "10 to 15 mph",
						"windDirection":    "NNE",
						"shortForecast":    "Mostly Clear",
						"detailedForecast": "Mostly clear with a low around 38.",
					},
				},
			},
		})
	})

	// alerts
	mux.HandleFunc("/alerts/active", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"features": []map[string]any{
				{
					"properties": map[string]any{
						"event":    "Wind Advisory",
						"headline": "Wind Advisory in effect until 6 PM",
						"severity": "Moderate",
						"urgency":  "Expected",
						"effective": "2026-03-22T12:00:00-05:00",
						"expires":  "2026-03-22T18:00:00-05:00",
						"areaDesc": "Jackson County",
					},
				},
			},
		})
	})

	return httptest.NewServer(mux)
}

func TestCurrentConditions(t *testing.T) {
	srv := mockNWSServer(t)
	defer srv.Close()

	p := newTestProvider(srv.URL)
	c := cache.NewNoOp()
	loc := location.Location{Lat: 39.1004, Lon: -94.5785, CountryCode: "US", DisplayName: "64101"}

	cond, err := p.CurrentConditions(context.Background(), loc, c)
	if err != nil {
		t.Fatalf("CurrentConditions: %v", err)
	}

	if cond.StationID != "KMKC" {
		t.Errorf("StationID = %q, want %q", cond.StationID, "KMKC")
	}
	if cond.TempC == nil || *cond.TempC != 17.0 {
		t.Errorf("TempC = %v, want 17.0", cond.TempC)
	}
	if cond.Description != "Clear" {
		t.Errorf("Description = %q, want %q", cond.Description, "Clear")
	}
	if cond.Location != "Kansas City, MO" {
		t.Errorf("Location = %q, want %q", cond.Location, "Kansas City, MO")
	}
	if cond.DewPointC == nil || *cond.DewPointC != 6.0 {
		t.Errorf("DewPointC = %v, want 6.0", cond.DewPointC)
	}
	if cond.WindGustKPH == nil || *cond.WindGustKPH != 25.0 {
		t.Errorf("WindGustKPH = %v, want 25.0", cond.WindGustKPH)
	}
	if cond.PressureHPA == nil || *cond.PressureHPA != 1015.0 {
		t.Errorf("PressureHPA = %v, want 1015.0", cond.PressureHPA)
	}
	if cond.VisibilityM == nil || *cond.VisibilityM != 16093.0 {
		t.Errorf("VisibilityM = %v, want 16093.0", cond.VisibilityM)
	}
	if cond.WindChillC == nil || *cond.WindChillC != 14.0 {
		t.Errorf("WindChillC = %v, want 14.0", cond.WindChillC)
	}
	if cond.HeatIndexC != nil {
		t.Errorf("HeatIndexC = %v, want nil", cond.HeatIndexC)
	}
	if cond.ConditionCode != "clear-day" {
		t.Errorf("ConditionCode = %q, want %q", cond.ConditionCode, "clear-day")
	}
}

func TestParseConditionCode(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://api.weather.gov/icons/land/day/skc?size=medium", "clear-day"},
		{"https://api.weather.gov/icons/land/night/skc?size=medium", "clear-night"},
		{"https://api.weather.gov/icons/land/day/few?size=medium", "partly-cloudy-day"},
		{"https://api.weather.gov/icons/land/night/sct,20?size=medium", "partly-cloudy-night"},
		{"https://api.weather.gov/icons/land/day/ovc?size=medium", "cloudy"},
		{"https://api.weather.gov/icons/land/day/rain?size=medium", "rain"},
		{"https://api.weather.gov/icons/land/day/tsra,40?size=medium", "thunder"},
		{"https://api.weather.gov/icons/land/day/snow?size=medium", "snow"},
		{"https://api.weather.gov/icons/land/day/fog?size=medium", "fog"},
		{"", ""},
		{"https://api.weather.gov/icons/land/day/unknown?size=medium", ""},
	}
	for _, tc := range cases {
		got := parseConditionCode(tc.url)
		if got != tc.want {
			t.Errorf("parseConditionCode(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

func TestForecast(t *testing.T) {
	srv := mockNWSServer(t)
	defer srv.Close()

	p := newTestProvider(srv.URL)
	c := cache.NewNoOp()
	loc := location.Location{Lat: 39.1004, Lon: -94.5785, CountryCode: "US"}

	fc, err := p.Forecast(context.Background(), loc, false, c)
	if err != nil {
		t.Fatalf("Forecast: %v", err)
	}

	if len(fc.Periods) != 1 {
		t.Fatalf("len(Periods) = %d, want 1", len(fc.Periods))
	}
	p0 := fc.Periods[0]
	if p0.Name != "Tonight" {
		t.Errorf("Period name = %q, want %q", p0.Name, "Tonight")
	}
	// 38°F → 3.33°C
	if p0.TempC < 3.3 || p0.TempC > 3.4 {
		t.Errorf("TempC = %.4f, want ~3.33", p0.TempC)
	}
	// "10 to 15 mph" upper bound → mphToKPH(15)
	wantKPH := mphToKPH(15)
	if p0.WindKPH != wantKPH {
		t.Errorf("WindKPH = %.4f, want %.4f", p0.WindKPH, wantKPH)
	}
}

func TestAlerts(t *testing.T) {
	srv := mockNWSServer(t)
	defer srv.Close()

	p := newTestProvider(srv.URL)
	c := cache.NewNoOp()
	loc := location.Location{Lat: 39.1004, Lon: -94.5785, CountryCode: "US"}

	alerts, err := p.Alerts(context.Background(), loc, c)
	if err != nil {
		t.Fatalf("Alerts: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1", len(alerts))
	}
	if alerts[0].Event != "Wind Advisory" {
		t.Errorf("Event = %q, want %q", alerts[0].Event, "Wind Advisory")
	}
	if alerts[0].Severity != "Moderate" {
		t.Errorf("Severity = %q, want %q", alerts[0].Severity, "Moderate")
	}
}

func TestAlertsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alerts/active":
			json.NewEncoder(w).Encode(map[string]any{"features": []any{}})
		default:
			// reuse mock for points/stations/observations
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Use the full mock for most endpoints, override alerts to return empty
	fullSrv := mockNWSServer(t)
	defer fullSrv.Close()

	// Test directly: empty features → empty slice, no error
	// The minimal server returns 404 for non-alert paths, so only test alerts path
	c := cache.NewNoOp()
	loc := location.Location{Lat: 39.1004, Lon: -94.5785, CountryCode: "US"}

	// Test that an empty features array returns empty slice:
	p2 := &Provider{client: newClient()}
	p2.client.baseURL = srv.URL

	alerts, err := p2.Alerts(context.Background(), loc, c)
	if err != nil {
		t.Fatalf("Alerts (empty): %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alerts))
	}
}
