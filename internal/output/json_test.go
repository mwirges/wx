package output

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

// captureStdout redirects os.Stdout to a buffer for the duration of fn.
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

func TestRenderJSON_Conditions(t *testing.T) {
	tempC := 17.0
	windKPH := 16.0934
	deg := 270.0
	humidity := 45.0

	data := RenderData{
		Conditions: &models.CurrentConditions{
			StationID:   "KMKC",
			ObservedAt:  time.Date(2026, 3, 22, 20, 0, 0, 0, time.UTC),
			Location:    "Kansas City, MO",
			Description: "Clear",
			TempC:       &tempC,
			WindKPH:     &windKPH,
			WindDegrees: &deg,
			HumidityPct: &humidity,
		},
	}
	opts := RenderOptions{Units: "imperial"}

	out := captureStdout(t, func() {
		if err := renderJSON(data, opts); err != nil {
			t.Errorf("renderJSON: %v", err)
		}
	})

	var result struct {
		Conditions struct {
			Station     string   `json:"station"`
			TempF       *float64 `json:"temperature_f"`
			TempC       *float64 `json:"temperature_c"`
			WindMPH     *float64 `json:"wind_mph"`
			WindDir     string   `json:"wind_direction"`
			HumidityPct *float64 `json:"humidity_pct"`
		} `json:"conditions"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("unmarshal output: %v\nraw: %s", err, out)
	}

	if result.Conditions.Station != "KMKC" {
		t.Errorf("station = %q, want %q", result.Conditions.Station, "KMKC")
	}
	if result.Conditions.TempF == nil || (*result.Conditions.TempF < 62 || *result.Conditions.TempF > 63) {
		t.Errorf("temperature_f = %v, want ~62.6", result.Conditions.TempF)
	}
	if result.Conditions.WindDir != "W" {
		t.Errorf("wind_direction = %q, want %q", result.Conditions.WindDir, "W")
	}
}

func TestRenderJSON_ExtendedFields(t *testing.T) {
	tempC := 17.0
	windChillC := 14.0
	dewPointC := 6.0
	gustKPH := 25.0
	pressureHPA := 1015.0
	visibilityM := 16093.0

	data := RenderData{
		Conditions: &models.CurrentConditions{
			StationID:     "KMKC",
			ObservedAt:    time.Date(2026, 3, 22, 20, 0, 0, 0, time.UTC),
			Location:      "Kansas City, MO",
			TempC:         &tempC,
			WindChillC:    &windChillC,
			DewPointC:     &dewPointC,
			WindGustKPH:   &gustKPH,
			PressureHPA:   &pressureHPA,
			VisibilityM:   &visibilityM,
			ConditionCode: "clear-day",
		},
	}

	out := captureStdout(t, func() {
		if err := renderJSON(data, RenderOptions{Units: "imperial"}); err != nil {
			t.Errorf("renderJSON: %v", err)
		}
	})

	var result struct {
		Conditions struct {
			FeelsLikeF    *float64 `json:"feels_like_f"`
			DewPointF     *float64 `json:"dew_point_f"`
			WindGustMPH   *float64 `json:"wind_gust_mph"`
			PressureInHg  *float64 `json:"pressure_inhg"`
			VisibilityMi  *float64 `json:"visibility_mi"`
			ConditionCode string   `json:"condition_code"`
		} `json:"conditions"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, out)
	}
	c := result.Conditions
	if c.FeelsLikeF == nil || (*c.FeelsLikeF < 57 || *c.FeelsLikeF > 58) {
		t.Errorf("feels_like_f = %v, want ~57.2 (14°C)", c.FeelsLikeF)
	}
	if c.DewPointF == nil || (*c.DewPointF < 42 || *c.DewPointF > 43) {
		t.Errorf("dew_point_f = %v, want ~42.8 (6°C)", c.DewPointF)
	}
	if c.WindGustMPH == nil || (*c.WindGustMPH < 15 || *c.WindGustMPH > 16) {
		t.Errorf("wind_gust_mph = %v, want ~15.5 (25 km/h)", c.WindGustMPH)
	}
	if c.PressureInHg == nil || (*c.PressureInHg < 29 || *c.PressureInHg > 31) {
		t.Errorf("pressure_inhg = %v, want ~29.97 (1015 hPa)", c.PressureInHg)
	}
	if c.VisibilityMi == nil || (*c.VisibilityMi < 9 || *c.VisibilityMi > 11) {
		t.Errorf("visibility_mi = %v, want ~10 (16093 m)", c.VisibilityMi)
	}
	if c.ConditionCode != "clear-day" {
		t.Errorf("condition_code = %q, want %q", c.ConditionCode, "clear-day")
	}
}

func TestRenderJSON_NoConditions(t *testing.T) {
	out := captureStdout(t, func() {
		renderJSON(RenderData{}, RenderOptions{Units: "imperial"})
	})

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, out)
	}
	if _, ok := result["conditions"]; ok {
		t.Error("expected no 'conditions' key when nil")
	}
}

func TestRenderJSON_Metric(t *testing.T) {
	tempC := 20.0
	data := RenderData{
		Conditions: &models.CurrentConditions{
			StationID: "KORD",
			TempC:     &tempC,
		},
	}

	out := captureStdout(t, func() {
		renderJSON(data, RenderOptions{Units: "metric"})
	})

	var result struct {
		Conditions struct {
			TempF *float64 `json:"temperature_f"`
			TempC *float64 `json:"temperature_c"`
		} `json:"conditions"`
	}
	json.Unmarshal(out, &result)

	if result.Conditions.TempF != nil {
		t.Error("temperature_f should be omitted in metric mode")
	}
	if result.Conditions.TempC == nil || *result.Conditions.TempC != 20.0 {
		t.Errorf("temperature_c = %v, want 20.0", result.Conditions.TempC)
	}
}

func TestRenderJSON_Alerts(t *testing.T) {
	data := RenderData{
		Alerts: []models.Alert{
			{
				Event:    "Wind Advisory",
				Severity: "Moderate",
				AreaDesc: "Jackson County",
			},
		},
	}

	out := captureStdout(t, func() {
		renderJSON(data, RenderOptions{Units: "imperial"})
	})

	var result struct {
		Alerts []struct {
			Event    string `json:"event"`
			Severity string `json:"severity"`
		} `json:"alerts"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1", len(result.Alerts))
	}
	if result.Alerts[0].Event != "Wind Advisory" {
		t.Errorf("event = %q, want %q", result.Alerts[0].Event, "Wind Advisory")
	}
}
