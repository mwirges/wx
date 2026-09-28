package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func TestRenderAQITo_JSON(t *testing.T) {
	aqi := 42
	uv := 2.5
	pm25 := 8.4
	pm10 := 12.0
	o3 := 45.0
	no2 := 10.0
	co := 200.0
	so2 := 1.5

	payload := &models.AirQualityPayload{
		Location:  "Fort Wayne, IN",
		Latitude:  41.08,
		Longitude: -85.14,
		FetchedAt: time.Date(2026, 9, 27, 18, 0, 0, 0, time.UTC),
		AirQuality: &models.AirQuality{
			AQI:        &aqi,
			Category:   "Good",
			UVIndex:    &uv,
			UVCategory: "Low",
			PM25:       &pm25,
			PM10:       &pm10,
			O3:         &o3,
			NO2:        &no2,
			CO:         &co,
			SO2:        &so2,
		},
		HealthAdvisory: models.EPAHealthAdvisory(aqi),
	}

	var buf bytes.Buffer
	err := RenderAQITo(&buf, payload, AQIOptions{ForceJSON: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded models.AirQualityPayload
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if decoded.Location != "Fort Wayne, IN" {
		t.Errorf("expected location 'Fort Wayne, IN', got %q", decoded.Location)
	}
	if decoded.AirQuality == nil || decoded.AirQuality.AQI == nil || *decoded.AirQuality.AQI != 42 {
		t.Errorf("expected AQI 42, got %v", decoded.AirQuality)
	}
}

func TestRenderAQITo_Pretty(t *testing.T) {
	aqi := 165
	uv := 8.5
	pm25 := 62.0

	payload := &models.AirQualityPayload{
		Location:  "Sacramento, CA",
		Latitude:  38.58,
		Longitude: -121.49,
		FetchedAt: time.Date(2026, 9, 27, 18, 0, 0, 0, time.UTC),
		AirQuality: &models.AirQuality{
			AQI:        &aqi,
			Category:   models.AQICategory(aqi),
			UVIndex:    &uv,
			UVCategory: models.UVCategory(uv),
			PM25:       &pm25,
		},
		HealthAdvisory: models.EPAHealthAdvisory(aqi),
		SmokeAdvisory:  models.SmokeAdvisory(pm25),
	}

	var buf bytes.Buffer
	err := RenderAQITo(&buf, payload, AQIOptions{ForcePretty: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Air Quality Index (AQI)") {
		t.Errorf("expected header, got:\n%s", out)
	}
	if !strings.Contains(out, "Sacramento, CA") {
		t.Errorf("expected Sacramento, CA, got:\n%s", out)
	}
	if !strings.Contains(out, "165") {
		t.Errorf("expected 165 AQI, got:\n%s", out)
	}
	if !strings.Contains(out, "SMOKE PLUME ADVISORY") {
		t.Errorf("expected smoke plume advisory, got:\n%s", out)
	}
}

func TestRenderAQITo_Nil(t *testing.T) {
	var buf bytes.Buffer
	err := RenderAQITo(&buf, nil, AQIOptions{ForcePretty: true})
	if err == nil {
		t.Error("expected error for nil payload")
	}
}
