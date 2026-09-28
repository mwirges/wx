package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func sampleCPCPayload() *models.CPCPayload {
	t1 := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	t4 := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)

	return &models.CPCPayload{
		Location:    "Fort Wayne, IN",
		Coordinates: [2]float64{41.08, -85.14},
		FetchedAt:   time.Now().UTC(),
		Outlooks: []models.CPCOutlookItem{
			{
				Horizon:           "6-10 Day",
				StartDate:         t1,
				EndDate:           t2,
				TempCategory:      "Above",
				TempProbability:   60.0,
				PrecipCategory:    "Above",
				PrecipProbability: 45.0,
			},
			{
				Horizon:           "8-14 Day",
				StartDate:         t3,
				EndDate:           t4,
				TempCategory:      "Below",
				TempProbability:   55.0,
				PrecipCategory:    "Below",
				PrecipProbability: 40.0,
			},
		},
		Drought: &models.CPCDroughtOutlook{
			Status: "No Drought",
			Target: "Oct 2026",
		},
		PatternShift: models.CPCPatternShift{
			HasShift:    true,
			Summary:     "Sharp cool-down from above-normal warmth to below-normal chill; wet start drying out substantially in week 2.",
			TempShift:   "cooling",
			PrecipShift: "drying",
			Confidence:  "High",
		},
	}
}

func TestRenderCPCTo_Pretty(t *testing.T) {
	payload := sampleCPCPayload()
	var buf bytes.Buffer
	opts := CPCOptions{ForcePretty: true}

	if err := RenderCPCTo(&buf, payload, opts); err != nil {
		t.Fatalf("RenderCPCTo: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Fort Wayne, IN") {
		t.Error("expected location in output")
	}
	if !strings.Contains(out, "6-10 Day") || !strings.Contains(out, "8-14 Day") {
		t.Error("expected horizons in output")
	}
	if !strings.Contains(out, "Pattern Shift Heads-Up") {
		t.Error("expected pattern shift section in output")
	}
	if !strings.Contains(out, "Sharp cool-down") {
		t.Error("expected summary in output")
	}
	if !strings.Contains(out, "No Drought") {
		t.Error("expected drought status in output")
	}
}

func TestRenderCPCTo_JSON(t *testing.T) {
	payload := sampleCPCPayload()
	var buf bytes.Buffer
	opts := CPCOptions{ForceJSON: true}

	if err := RenderCPCTo(&buf, payload, opts); err != nil {
		t.Fatalf("RenderCPCTo JSON: %v", err)
	}

	var decoded models.CPCPayload
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}

	if decoded.Location != payload.Location {
		t.Errorf("location = %q, want %q", decoded.Location, payload.Location)
	}
	if len(decoded.Outlooks) != 2 {
		t.Errorf("got %d outlooks, want 2", len(decoded.Outlooks))
	}
	if !decoded.PatternShift.HasShift {
		t.Error("expected HasShift = true")
	}
}

func TestRenderCPCTo_Nil(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderCPCTo(&buf, nil, CPCOptions{}); err == nil {
		t.Error("expected error for nil payload")
	}
}
