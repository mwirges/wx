package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func TestRenderSPCTo_JSON(t *testing.T) {
	payload := &models.SPCPayload{
		Location:    "Cherry County, NE",
		Coordinates: [2]float64{42.0, -101.5},
		FetchedAt:   time.Now().UTC(),
		Day1: models.SPCOutlookItem{
			Day: 1,
			Category: models.SPCRiskCategory{
				DN:   4,
				Code: "SLGT",
				Name: "2 - Slight Risk",
			},
			TornadoProb: "5%",
			HailProb:    "15%",
			HailSig:     true,
			WindProb:    "15%",
		},
		ActiveMCDs: []models.MesoscaleDiscussion{
			{
				ID:               2329,
				Name:             "MD 2329",
				Concerning:       "Severe potential...Watch unlikely",
				WatchProbability: "20 percent",
				AreasAffected:    "Parts of the NE Sand Hills region",
				Summary:          "An isolated supercell may continue through sunset.",
				URL:              "https://www.spc.noaa.gov/products/md/md2329.html",
			},
		},
		ActiveWatches: []models.SPCWatch{
			{
				WatchNumber: 512,
				Type:        "Severe Thunderstorm Watch",
				AreaDesc:    "West Central Nebraska",
				States:      []string{"NE"},
			},
		},
		ConvectiveSummary: "Day 1: 2 - Slight Risk. 1 active Mesoscale Discussion.",
	}

	var buf bytes.Buffer
	err := RenderSPCTo(&buf, payload, SPCOptions{ForceJSON: true})
	if err != nil {
		t.Fatalf("RenderSPCTo JSON error: %v", err)
	}

	var parsed models.SPCPayload
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if parsed.Day1.Category.Code != "SLGT" {
		t.Errorf("parsed.Day1.Category.Code = %q, want SLGT", parsed.Day1.Category.Code)
	}
	if len(parsed.ActiveMCDs) != 1 || parsed.ActiveMCDs[0].ID != 2329 {
		t.Errorf("parsed.ActiveMCDs unexpected: %+v", parsed.ActiveMCDs)
	}
	if len(parsed.ActiveWatches) != 1 || parsed.ActiveWatches[0].WatchNumber != 512 {
		t.Errorf("parsed.ActiveWatches unexpected: %+v", parsed.ActiveWatches)
	}
}

func TestRenderSPCTo_Pretty(t *testing.T) {
	payload := &models.SPCPayload{
		Location:    "Cherry County, NE",
		Coordinates: [2]float64{42.0, -101.5},
		FetchedAt:   time.Now().UTC(),
		Day1: models.SPCOutlookItem{
			Day: 1,
			Category: models.SPCRiskCategory{
				DN:   4,
				Code: "SLGT",
				Name: "2 - Slight Risk",
			},
			TornadoProb: "5%",
			HailProb:    "15%",
			HailSig:     true,
			WindProb:    "15%",
		},
		Day2: models.SPCOutlookItem{
			Day: 2,
			Category: models.SPCRiskCategory{
				DN:   3,
				Code: "MRGL",
				Name: "1 - Marginal Risk",
			},
		},
		Day3: models.SPCOutlookItem{
			Day: 3,
			Category: models.SPCRiskCategory{
				DN:   2,
				Code: "TSTM",
				Name: "General Thunderstorms",
			},
			SevereProb: "5%",
		},
		ActiveMCDs: []models.MesoscaleDiscussion{
			{
				ID:               2329,
				Name:             "MD 2329",
				Concerning:       "Severe potential...Watch unlikely",
				WatchProbability: "20 percent",
				AreasAffected:    "Parts of the NE Sand Hills region",
				Summary:          "An isolated supercell may continue through sunset.",
				URL:              "https://www.spc.noaa.gov/products/md/md2329.html",
			},
		},
		ActiveWatches: []models.SPCWatch{
			{
				WatchNumber: 512,
				Type:        "Severe Thunderstorm Watch",
				AreaDesc:    "West Central Nebraska",
				States:      []string{"NE"},
			},
		},
		MaxNationalRisk: models.SPCRiskCategory{
			DN:   4,
			Code: "SLGT",
			Name: "2 - Slight Risk",
		},
		ConvectiveSummary: "Day 1: 2 - Slight Risk. 1 active watch(es) in effect.",
	}

	var buf bytes.Buffer
	err := RenderSPCTo(&buf, payload, SPCOptions{ForcePretty: true})
	if err != nil {
		t.Fatalf("RenderSPCTo Pretty error: %v", err)
	}

	out := buf.String()

	mustContain := []string{
		"NOAA Storm Prediction Center (SPC) Convective Outlook",
		"Cherry County, NE",
		"SLGT",
		"ACTIVE SEVERE WEATHER WATCHES",
		"Watch #512",
		"ACTIVE MESOSCALE DISCUSSIONS (MCD)",
		"MD 2329",
		"Watch Probability: 20 percent",
		"Day 1",
		"Day 2",
		"Day 3",
		"Outlook Summary:",
	}

	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("output missing expected substring: %q", s)
		}
	}
}
