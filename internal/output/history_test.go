package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func sampleHistoricalWeather() *models.HistoricalWeather {
	t1 := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)

	high1, low1 := 20.0, 10.0
	appHigh1, appLow1 := 19.0, 9.0
	precip1 := 0.0
	wind1 := 15.0

	high2, low2 := 22.0, 12.0
	appHigh2, appLow2 := 21.0, 11.0
	precip2 := 25.4 // 1.0 inch
	wind2 := 20.0

	avgHigh := 21.0
	avgLow := 11.0
	totalPrecip := 25.4
	maxWind := 20.0

	return &models.HistoricalWeather{
		Location:  "Fort Wayne, IN",
		Latitude:  41.07,
		Longitude: -85.14,
		Days: []models.HistoryDay{
			{
				Date:          t1,
				ConditionCode: "partly-cloudy-day",
				Description:   "Partly Cloudy",
				TempMaxC:      &high1,
				TempMinC:      &low1,
				ApparentMaxC:  &appHigh1,
				ApparentMinC:  &appLow1,
				PrecipSumMM:   &precip1,
				WindMaxKPH:    &wind1,
			},
			{
				Date:          t2,
				ConditionCode: "rain",
				Description:   "Moderate Rain",
				TempMaxC:      &high2,
				TempMinC:      &low2,
				ApparentMaxC:  &appHigh2,
				ApparentMinC:  &appLow2,
				PrecipSumMM:   &precip2,
				WindMaxKPH:    &wind2,
			},
		},
		Summary: models.HistorySummary{
			DaysCount:     2,
			AvgTempMaxC:   &avgHigh,
			AvgTempMinC:   &avgLow,
			TotalPrecipMM: &totalPrecip,
			MaxWindKPH:    &maxWind,
		},
	}
}

func TestRenderHistory_JSON(t *testing.T) {
	h := sampleHistoricalWeather()
	var buf bytes.Buffer
	err := RenderHistoryTo(&buf, h, HistoryOptions{ForceJSON: true, Units: "imperial"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if data["location"] != "Fort Wayne, IN" {
		t.Errorf("expected location 'Fort Wayne, IN', got %v", data["location"])
	}

	days, ok := data["days"].([]any)
	if !ok || len(days) != 2 {
		t.Fatalf("expected 2 days, got %v", data["days"])
	}

	d2 := days[1].(map[string]any)
	if d2["description"] != "Moderate Rain" {
		t.Errorf("expected description 'Moderate Rain', got %v", d2["description"])
	}
	if d2["temperature_max_f"] == nil || d2["precipitation_in"] == nil {
		t.Errorf("expected imperial conversions in day 2: %v", d2)
	}

	summary, ok := data["summary"].(map[string]any)
	if !ok {
		t.Fatalf("missing summary: %v", data)
	}
	if summary["days_count"] != float64(2) {
		t.Errorf("expected days_count 2, got %v", summary["days_count"])
	}
	if summary["total_precipitation_in"] == nil {
		t.Error("expected total_precipitation_in in summary")
	}
}

func TestRenderHistory_Pretty(t *testing.T) {
	h := sampleHistoricalWeather()
	var buf bytes.Buffer
	err := renderHistoryPretty(&buf, h, HistoryOptions{Units: "imperial"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Fort Wayne, IN — Past 2 Days") {
		t.Errorf("expected header in output, got: %s", out)
	}
	if !strings.Contains(out, "Moderate Rain") {
		t.Errorf("expected 'Moderate Rain' in output, got: %s", out)
	}
	if !strings.Contains(out, "SUMMARY") {
		t.Errorf("expected 'SUMMARY' in output, got: %s", out)
	}
	if !strings.Contains(out, "1.00 in") {
		t.Errorf("expected '1.00 in' precip in output, got: %s", out)
	}
}
