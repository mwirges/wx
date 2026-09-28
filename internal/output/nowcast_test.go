package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func sampleNowcastPayload() *models.NowcastPayload {
	now := time.Date(2026, 9, 27, 20, 0, 0, 0, time.UTC)
	peakTime := now.Add(45 * time.Minute)
	nextPrecip := now.Add(30 * time.Minute)

	return &models.NowcastPayload{
		Location: "Fort Wayne, IN",
		Nowcast: &models.Nowcast{
			GeneratedAt:    now,
			Location:       "Fort Wayne, IN",
			Headline:       "Precipitation starting in ~30 min (Light Rain)",
			IsActivePrecip: false,
			Summary:        "Light Rain begins around 8:30 PM with expected total of 0.35 in liquid equivalent.",
			PrimaryPhase:   models.PhaseRain,
			TotalLiquidMM:  8.89,
			TotalLiquidIn:  0.35,
			TotalSnowCM:    0.0,
			TotalSnowIn:    0.0,
			PeakRateMMH:    6.40,
			PeakRateInH:    0.25,
			PeakTime:       peakTime,
			NextPrecipTime: &nextPrecip,
			Intervals: []models.PrecipInterval{
				{
					StartTime:   now,
					EndTime:     now.Add(15 * time.Minute),
					Probability: 10,
					RateMMH:     0.0,
					RateInH:     0.0,
					AccumMM:     0.0,
					AccumIn:     0.0,
					Phase:       models.PhaseNone,
					Summary:     "Clear",
				},
				{
					StartTime:   now.Add(30 * time.Minute),
					EndTime:     now.Add(45 * time.Minute),
					Probability: 80,
					RateMMH:     2.54,
					RateInH:     0.10,
					AccumMM:     0.64,
					AccumIn:     0.025,
					Phase:       models.PhaseRain,
					Summary:     "Light Rain",
				},
				{
					StartTime:   peakTime,
					EndTime:     peakTime.Add(15 * time.Minute),
					Probability: 95,
					RateMMH:     6.35,
					RateInH:     0.25,
					AccumMM:     1.59,
					AccumIn:     0.062,
					Phase:       models.PhaseRain,
					Summary:     "Moderate Rain",
				},
			},
		},
	}
}

func TestRenderNowcast_JSON(t *testing.T) {
	payload := sampleNowcastPayload()
	var buf bytes.Buffer

	err := RenderNowcastTo(&buf, payload, NowcastOptions{ForceJSON: true})
	if err != nil {
		t.Fatalf("RenderNowcastTo error: %v", err)
	}

	var decoded models.NowcastPayload
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if decoded.Location != "Fort Wayne, IN" {
		t.Errorf("expected location 'Fort Wayne, IN', got %q", decoded.Location)
	}
	if decoded.Nowcast == nil || decoded.Nowcast.Headline != payload.Nowcast.Headline {
		t.Errorf("headline mismatch: got %v", decoded.Nowcast)
	}
	if len(decoded.Nowcast.Intervals) != 3 {
		t.Errorf("expected 3 intervals, got %d", len(decoded.Nowcast.Intervals))
	}
}

func TestRenderNowcast_PrettyImperial(t *testing.T) {
	payload := sampleNowcastPayload()
	var buf bytes.Buffer

	err := RenderNowcastTo(&buf, payload, NowcastOptions{ForcePretty: true, Units: "imperial"})
	if err != nil {
		t.Fatalf("RenderNowcastTo error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Quantitative Precipitation Nowcast & Rain Timeline") {
		t.Errorf("missing header in pretty output: %s", out)
	}
	if !strings.Contains(out, "Fort Wayne, IN") {
		t.Errorf("missing location in pretty output")
	}
	if !strings.Contains(out, "Precipitation starting in ~30 min") {
		t.Errorf("missing headline in pretty output")
	}
	if !strings.Contains(out, "0.35 in") {
		t.Errorf("missing liquid accumulation in pretty output")
	}
	if !strings.Contains(out, "0.25 in/h") {
		t.Errorf("missing peak rate in pretty output")
	}
}

func TestRenderNowcast_PrettyMetric(t *testing.T) {
	payload := sampleNowcastPayload()
	var buf bytes.Buffer

	err := RenderNowcastTo(&buf, payload, NowcastOptions{ForcePretty: true, Units: "metric"})
	if err != nil {
		t.Fatalf("RenderNowcastTo error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "8.9 mm") {
		t.Errorf("missing metric liquid accumulation in pretty output: %s", out)
	}
	if !strings.Contains(out, "6.4 mm/h") {
		t.Errorf("missing metric peak rate in pretty output: %s", out)
	}
}

func TestRenderNowcast_Nil(t *testing.T) {
	var buf bytes.Buffer
	err := RenderNowcastTo(&buf, nil, NowcastOptions{})
	if err == nil {
		t.Errorf("expected error for nil payload, got nil")
	}
}
