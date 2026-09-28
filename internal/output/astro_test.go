package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func TestRenderAstroTo_JSON(t *testing.T) {
	sr := time.Date(2026, 9, 27, 11, 30, 0, 0, time.UTC)
	ss := time.Date(2026, 9, 27, 23, 30, 0, 0, time.UTC)
	sn := time.Date(2026, 9, 27, 17, 30, 0, 0, time.UTC)
	elev := 38.5
	az := 195.0
	illum := 95.0
	age := 15.5

	payload := &models.AstroPayload{
		Location:     "Fort Wayne, IN",
		Latitude:     41.08,
		Longitude:    -85.14,
		CalculatedAt: time.Date(2026, 9, 27, 18, 0, 0, 0, time.UTC),
		Astronomy: &models.Astronomy{
			Sunrise:             &sr,
			Sunset:              &ss,
			SolarNoon:           sn,
			DayLength:           12 * time.Hour,
			SolarElevationDeg:   &elev,
			SolarAzimuthDeg:     &az,
			CurrentPeriod:       "Daylight",
			MoonPhase:           "Full Moon",
			MoonPhaseIcon:       "🌕",
			MoonIlluminationPct: &illum,
			MoonAgeDays:         &age,
		},
	}

	var buf bytes.Buffer
	err := RenderAstroTo(&buf, payload, AstroOptions{ForceJSON: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded models.AstroPayload
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if decoded.Location != "Fort Wayne, IN" {
		t.Errorf("expected location 'Fort Wayne, IN', got %q", decoded.Location)
	}
	if decoded.Astronomy == nil || decoded.Astronomy.SolarElevationDeg == nil || *decoded.Astronomy.SolarElevationDeg != 38.5 {
		t.Errorf("expected SolarElevationDeg 38.5, got %v", decoded.Astronomy)
	}
}

func TestRenderAstroTo_Pretty(t *testing.T) {
	sr := time.Date(2026, 9, 27, 11, 30, 0, 0, time.UTC)
	ss := time.Date(2026, 9, 27, 23, 30, 0, 0, time.UTC)
	sn := time.Date(2026, 9, 27, 17, 30, 0, 0, time.UTC)
	elev := 42.0
	az := 180.0
	illum := 88.0
	age := 12.0

	payload := &models.AstroPayload{
		Location:     "New Haven, IN",
		Latitude:     41.07,
		Longitude:    -85.02,
		CalculatedAt: time.Date(2026, 9, 27, 17, 30, 0, 0, time.UTC),
		Astronomy: &models.Astronomy{
			Sunrise:             &sr,
			Sunset:              &ss,
			SolarNoon:           sn,
			DayLength:           12 * time.Hour,
			SolarElevationDeg:   &elev,
			SolarAzimuthDeg:     &az,
			CurrentPeriod:       "Daylight",
			MoonPhase:           "Waxing Gibbous",
			MoonPhaseIcon:       "🌔",
			MoonIlluminationPct: &illum,
			MoonAgeDays:         &age,
		},
	}

	var buf bytes.Buffer
	err := RenderAstroTo(&buf, payload, AstroOptions{ForcePretty: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Solar Arc, Ephemeris & Lunar HUD") {
		t.Errorf("expected header, got:\n%s", out)
	}
	if !strings.Contains(out, "New Haven, IN") {
		t.Errorf("expected New Haven, IN, got:\n%s", out)
	}
	if !strings.Contains(out, "Solar Schedule") {
		t.Errorf("expected Solar Schedule, got:\n%s", out)
	}
	if !strings.Contains(out, "Waxing Gibbous") {
		t.Errorf("expected Waxing Gibbous, got:\n%s", out)
	}
}

func TestAzimuthToCompass(t *testing.T) {
	tests := []struct {
		az   float64
		want string
	}{
		{0.0, "N"},
		{90.0, "E"},
		{180.0, "S"},
		{270.0, "W"},
		{45.0, "NE"},
		{225.0, "SW"},
	}

	for _, tt := range tests {
		got := AzimuthToCompass(tt.az)
		if got != tt.want {
			t.Errorf("AzimuthToCompass(%v) = %q, want %q", tt.az, got, tt.want)
		}
	}
}
