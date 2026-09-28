package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func TestRenderTropicsPretty(t *testing.T) {
	distMiles := 450.0
	distKm := 724.0

	payload := &models.TropicsPayload{
		Location: "Miami, FL",
		Tropics: &models.TropicsReport{
			GeneratedAt:        time.Now().UTC(),
			TotalActive:        1,
			ReferenceLocation:  "Miami, FL",
			AtlanticOutlookURL: "https://www.nhc.noaa.gov/xgtwo/two_atl_7d0.png",
			PacificOutlookURL:  "https://www.nhc.noaa.gov/xgtwo/two_pac_7d0.png",
			Storms: []models.TropicalStorm{
				{
					ID:                 "ep172026",
					BinNumber:          "EP2",
					Name:               "Polo",
					Classification:     "HU",
					ClassificationName: "Hurricane",
					Category:           3,
					CategoryLabel:      "Category 3 (Major Hurricane)",
					IntensityKt:        100,
					WindSpeedMph:       115,
					WindSpeedKmh:       185,
					PressureMb:         958,
					PressureInHg:       28.29,
					Latitude:           23.6,
					Longitude:          -113.8,
					LocationText:       "23.6N 113.8W",
					MovementDir:        15,
					MovementCompass:    "NNE",
					MovementSpeedMph:   10,
					MovementSpeedKmh:   17,
					Headline:           "LIFE-THREATENING WINDS AND FLASH FLOODS EXPECTED",
					ProximityText:      "About 125 mi SW of Cabo San Lazaro Mexico",
					DistanceKm:         &distKm,
					DistanceMiles:      &distMiles,
					WatchesWarnings: []string{
						"Hurricane Warning is in effect for Baja California Sur",
					},
					AdvisoryNumber:    "031",
					PublicAdvisoryURL: "https://www.nhc.noaa.gov/text/MIATCPEP2.shtml",
					GraphicsURL:       "https://www.nhc.noaa.gov/graphics_ep2.shtml",
				},
			},
			Disturbances: []models.TropicalDisturbance{
				{
					ID:          "AL91",
					Basin:       "Atlantic",
					Name:        "Central Subtropical Atlantic",
					Chance48h:   50,
					Category48h: "medium",
					Chance7d:    60,
					Category7d:  "medium",
					Summary:     "A trough of low pressure located several hundred miles east-northeast of Bermuda.",
				},
			},
		},
	}

	var buf bytes.Buffer
	opts := TropicsOptions{ForcePretty: true, Units: "imperial"}
	if err := RenderTropicsTo(&buf, payload, opts); err != nil {
		t.Fatalf("RenderTropicsTo failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "HURRICANE POLO #031") {
		t.Errorf("expected storm Polo in output, got:\n%s", out)
	}
	if !strings.Contains(out, "115 mph") {
		t.Errorf("expected 115 mph in output, got:\n%s", out)
	}
	if !strings.Contains(out, "958 mb") {
		t.Errorf("expected 958 mb in output, got:\n%s", out)
	}
	if !strings.Contains(out, "AL91") {
		t.Errorf("expected disturbance AL91 in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Hurricane Warning is in effect") {
		t.Errorf("expected warning in output, got:\n%s", out)
	}
	if !strings.Contains(out, "450 miles away") {
		t.Errorf("expected distance in output, got:\n%s", out)
	}
}

func TestRenderTropicsJSON(t *testing.T) {
	payload := &models.TropicsPayload{
		Location: "Miami, FL",
		Tropics: &models.TropicsReport{
			GeneratedAt: time.Now().UTC(),
			TotalActive: 0,
			Storms:      []models.TropicalStorm{},
		},
	}

	var buf bytes.Buffer
	opts := TropicsOptions{ForceJSON: true}
	if err := RenderTropicsTo(&buf, payload, opts); err != nil {
		t.Fatalf("RenderTropicsTo JSON failed: %v", err)
	}

	var decoded models.TropicsPayload
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON output: %v", err)
	}

	if decoded.Location != "Miami, FL" {
		t.Errorf("expected location Miami, FL, got %s", decoded.Location)
	}
	if decoded.Tropics.TotalActive != 0 {
		t.Errorf("expected 0 active storms, got %d", decoded.Tropics.TotalActive)
	}
}
