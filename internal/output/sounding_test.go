package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func sampleSoundingPayload() *models.SoundingPayload {
	cape := 2450.0
	cin := -45.0
	li := -6.2
	pwat := 1.65
	shear := 48.0
	srh := 185.0
	stp := 2.1
	t1000 := 22.0
	td1000 := 14.0
	ws1000 := 12.0
	wd1000 := 180.0
	rh1000 := 60.0

	return &models.SoundingPayload{
		Location: "Lincoln, Illinois",
		Sounding: &models.SoundingReport{
			Timestamp:     time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC),
			Location:      "Lincoln, Illinois",
			StationID:     "ILX",
			StationName:   "Lincoln (IL)",
			DistanceKM:    42.0,
			DistanceMiles: 26.1,
			Provider:      "NOAA SPC Upper-Air RAOB (NSHARP)",
			SkewTImageURL: "https://www.spc.noaa.gov/exper/soundings/26092800_OBS/ILX.gif",
			Indices: models.ConvectiveIndices{
				SBCAPE:             &cape,
				SBCIN:              &cin,
				SBLI:               &li,
				PWATIn:             &pwat,
				BulkShear06KT:      &shear,
				SRH01:              &srh,
				STP:                &stp,
				InstabilitySummary: "Strong Instability",
				ShearSummary:       "Strong Deep-Layer Shear (Supercells Favored)",
				ConvectiveRisk:     "High Severe Convective Potential",
			},
			Levels: []models.SoundingLevel{
				{
					PressureHPA: 1000.0,
					TempC:       &t1000,
					DewPointC:   &td1000,
					WindSpeedKT: &ws1000,
					WindDirDeg:  &wd1000,
					RHPct:       &rh1000,
				},
			},
		},
	}
}

func TestRenderSounding_Pretty(t *testing.T) {
	var buf bytes.Buffer
	p := sampleSoundingPayload()
	err := RenderSoundingTo(&buf, p, SoundingOptions{ForcePretty: true, Units: "imperial"})
	if err != nil {
		t.Fatalf("unexpected error rendering pretty sounding: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ATMOSPHERIC SOUNDING & CONVECTIVE INSTABILITY") {
		t.Errorf("expected title in output, got:\n%s", out)
	}
	if !strings.Contains(out, "ILX") {
		t.Errorf("expected station ID ILX in output, got:\n%s", out)
	}
	if !strings.Contains(out, "2450 J/kg") {
		t.Errorf("expected CAPE 2450 in output, got:\n%s", out)
	}
	if !strings.Contains(out, "48 kt") {
		t.Errorf("expected shear 48 kt in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1.65 in") {
		t.Errorf("expected PWAT 1.65 in in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1000 hPa") {
		t.Errorf("expected 1000 hPa level in output, got:\n%s", out)
	}
}

func TestRenderSounding_JSON(t *testing.T) {
	var buf bytes.Buffer
	p := sampleSoundingPayload()
	err := RenderSoundingTo(&buf, p, SoundingOptions{ForceJSON: true})
	if err != nil {
		t.Fatalf("unexpected error rendering json sounding: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"station_id": "ILX"`) {
		t.Errorf("expected json to contain station_id ILX, got:\n%s", out)
	}
	if !strings.Contains(out, `"sbcape_jkg": 2450`) {
		t.Errorf("expected json to contain sbcape_jkg 2450, got:\n%s", out)
	}
	if !strings.Contains(out, `"bulk_shear_0_6km_kt": 48`) {
		t.Errorf("expected json to contain bulk_shear_0_6km_kt 48, got:\n%s", out)
	}
}
