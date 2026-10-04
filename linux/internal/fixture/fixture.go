// Package fixture holds CLI-shaped JSON for shell tests. It is not a weather source.
package fixture

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Payload is a warning observation the desk can render.
var Payload = map[string]any{
	"conditions": map[string]any{
		"station":        "KFWA",
		"observed_at":    "2026-10-04T15:00:00Z",
		"location":       "Fort Wayne, IN",
		"description":    "Partly Cloudy",
		"condition_code": "partly-cloudy-day",
		"temperature_c":  22.2,
		"temperature_f":  72.0,
		"feels_like_f":   72.0,
		"feels_like_c":   22.2,
		"humidity_pct":   55.0,
		"wind_mph":       12.0,
		"wind_kph":       19.3,
		"wind_direction": "NW",
		"pressure_inhg":  30.05,
		"pressure_hpa":   1017.0,
		"visibility_mi":  10.0,
		"visibility_m":   16093.0,
		"nowcast": map[string]any{
			"headline":         "Dry next 6h",
			"is_active_precip": false,
			"summary":          "No precipitation expected.",
			"primary_phase":    "none",
			"intervals": []any{
				map[string]any{
					"start_time":  "2026-10-04T15:00:00Z",
					"summary":     "Clear",
					"probability": 0.0,
					"phase":       "none",
				},
			},
		},
	},
	"forecast": map[string]any{
		"periods": []any{
			map[string]any{
				"name":                         "This Afternoon",
				"start_time":                   "2026-10-04T18:00:00Z",
				"is_daytime":                   true,
				"temperature_f":                74.0,
				"temperature_c":                23.0,
				"short_description":            "Partly Sunny",
				"probability_of_precipitation": 10.0,
			},
		},
		"hourly": []any{
			map[string]any{
				"name":                         "3 PM",
				"start_time":                   "2026-10-04T19:00:00Z",
				"is_daytime":                   true,
				"temperature_f":                72.0,
				"temperature_c":                22.0,
				"short_description":            "Partly Cloudy",
				"probability_of_precipitation": 5.0,
			},
		},
	},
	"alerts": []any{
		map[string]any{
			"event":       "Severe Thunderstorm Warning",
			"headline":    "Storm near New Haven",
			"severity":    "Severe",
			"urgency":     "Immediate",
			"description": "Hail and wind.",
		},
	},
}

// Quiet is a clear observation with no alerts.
var Quiet = map[string]any{
	"conditions": map[string]any{
		"station":        "KFWA",
		"location":       "Fort Wayne, IN",
		"temperature_f":  68.0,
		"temperature_c":  20.0,
		"condition_code": "clear-day",
	},
	"forecast": map[string]any{"periods": []any{}, "hourly": []any{}},
	"alerts":   []any{},
}

// Watch is a flood watch.
var Watch = map[string]any{
	"conditions": map[string]any{
		"station":       "KORD",
		"location":      "Chicago, IL",
		"temperature_f": 80.0,
		"temperature_c": 26.7,
	},
	"alerts": []any{
		map[string]any{"event": "Flood Watch", "severity": "Moderate", "headline": "Flooding possible"},
	},
}

// Fake is a wx CLI stand-in. It records calls and does not write radar files.
type Fake struct {
	mu          sync.Mutex
	Calls       [][]any
	CancelCount int
	Weather     map[string]any
	Fail        error
}

// Add records one CLI call.
func (f *Fake) Add(parts ...any) {
	f.mu.Lock()
	f.Calls = append(f.Calls, parts)
	f.mu.Unlock()
}

// Names returns the call kinds in order.
func (f *Fake) Names() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.Calls))
	for i, call := range f.Calls {
		out[i], _ = call[0].(string)
	}
	return out
}

// Cancels is how many times radar was cancelled.
func (f *Fake) Cancels() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.CancelCount
}

// FetchWeather records a weather call.
func (f *Fake) FetchWeather(location, units string, hourly bool) (map[string]any, error) {
	f.Add("weather", location, units, hourly)
	if f.Fail != nil {
		return nil, f.Fail
	}
	if f.Weather != nil {
		return f.Weather, nil
	}
	return Payload, nil
}

// FetchRadar records a radar call and returns two frames.
func (f *Fake) FetchRadar(location, product string, radius float64, bbox string, frames int) (map[string]any, error) {
	f.Add("radar", location, product, radius, frames)
	return map[string]any{
		"product":       product,
		"product_label": "Composite Reflectivity",
		"location":      location,
		"valid_time":    "2026-10-04T15:00:00Z",
		"image_base64":  "aGVsbG8=",
		"frames": []any{
			map[string]any{"valid_time": "2026-10-04T14:50:00Z", "image_base64": "aGVsbG8="},
			map[string]any{"valid_time": "2026-10-04T15:00:00Z", "image_base64": "d29ybGQ="},
		},
	}, nil
}

// CancelRadar records a radar cancel.
func (f *Fake) CancelRadar() {
	f.mu.Lock()
	f.CancelCount++
	f.mu.Unlock()
}

// FetchOutlook records an outlook call.
func (f *Fake) FetchOutlook(location string) (map[string]any, error) {
	f.Add("outlook", location)
	return map[string]any{"location": location, "outlooks": []any{}}, nil
}

// FetchChase records a chase call.
func (f *Fake) FetchChase() (map[string]any, error) {
	f.Add("chase")
	return map[string]any{"total_clusters": 0, "clusters": []any{}}, nil
}

// FetchClimate records a climate call.
func (f *Fake) FetchClimate(location, units string) (map[string]any, error) {
	f.Add("climate", location, units)
	return map[string]any{"location": location}, nil
}

// FetchHistory records a history call.
func (f *Fake) FetchHistory(location, units string, days int) (map[string]any, error) {
	f.Add("history", location, units, days)
	return map[string]any{"days": []any{}}, nil
}

// FetchTropics records a tropics call.
func (f *Fake) FetchTropics(location, units, storm string) (map[string]any, error) {
	f.Add("tropics", location)
	return map[string]any{"tropics": map[string]any{"storms": []any{}, "total_active": 0}}, nil
}

// ExportPNG records a save and does not create the file.
func (f *Fake) ExportPNG(path, location, product string, radius float64) (string, error) {
	f.Add("export-png", path)
	return "saved", nil
}

// ExportGIF records a gif save and does not create the file.
func (f *Fake) ExportGIF(path, location, product string, radius float64, frames int) (string, error) {
	f.Add("export-gif", path, frames)
	return "saved", nil
}

// GuardHome fails the test if ~/.config/wx/config.json changes.
func GuardHome(t *testing.T) func() {
	t.Helper()
	path := filepath.Join(homeDir(), ".config", "wx", "config.json")
	before, err := os.ReadFile(path)
	missing := err != nil
	return func() {
		t.Helper()
		after, readErr := os.ReadFile(path)
		if missing {
			if readErr == nil {
				t.Fatalf("created user config %s", path)
			}
			return
		}
		if readErr != nil {
			t.Fatalf("user config disappeared: %s", path)
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("user config changed: %s", path)
		}
	}
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
