package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func sampleChasePayload() *models.ChasePayload {
	now := time.Now()
	exp := now.Add(45 * time.Minute)

	return &models.ChasePayload{
		GeneratedAt:   now,
		TotalAlerts:   3,
		TotalClusters: 2,
		Clusters: []models.StormCluster{
			{
				ID:           1,
				Name:         "Southern Plains Severe Thunderstorms",
				States:       []string{"OK", "TX"},
				CenterLat:    35.2,
				CenterLon:    -97.4,
				NearestRadar: "KTLX",
				TotalAlerts:  2,
				Score:        140,
				HazardsCount: map[string]int{
					"Tornado Warning":             1,
					"Severe Thunderstorm Warning": 1,
				},
				PrimaryHazard: "Tornado Warning",
				Cells: []models.AlertCell{
					{
						ID:          "cell-1",
						Event:       "Tornado Warning",
						Headline:    "Tornado Warning for Oklahoma County",
						AreaDesc:    "Oklahoma, OK",
						Description: "Radar indicated tornado.",
						Severity:    "Extreme",
						Latitude:    35.25,
						Longitude:   -97.35,
						HazardText:  "Tornado indicated, 1.75 inch hail",
						RadarSite:   "KTLX",
						Effective:   now,
						Expires:     exp,
					},
					{
						ID:          "cell-2",
						Event:       "Severe Thunderstorm Warning",
						Headline:    "Severe Thunderstorm Warning for Cleveland County",
						AreaDesc:    "Cleveland, OK",
						Severity:    "Severe",
						Latitude:    35.15,
						Longitude:   -97.45,
						HazardText:  "60 mph wind gusts",
						RadarSite:   "KTLX",
						Effective:   now,
						Expires:     exp,
					},
				},
			},
			{
				ID:           2,
				Name:         "Mid-Atlantic Coastal Storm (Nor'easter)",
				States:       []string{"NJ", "DE"},
				CenterLat:    39.5,
				CenterLon:    -74.5,
				NearestRadar: "KDIX",
				TotalAlerts:  1,
				Score:        20,
				HazardsCount: map[string]int{
					"Coastal Flood Warning": 1,
				},
				PrimaryHazard: "Coastal Flood Warning",
				Cells: []models.AlertCell{
					{
						ID:          "cell-3",
						Event:       "Coastal Flood Warning",
						Headline:    "Coastal Flood Warning for Cape May",
						AreaDesc:    "Cape May, NJ",
						Severity:    "Severe",
						Latitude:    39.0,
						Longitude:   -74.9,
						HazardText:  "Up to 2 feet of inundation",
						RadarSite:   "KDIX",
						Effective:   now,
						Expires:     exp.Add(3 * time.Hour),
					},
				},
			},
		},
	}
}

func TestRenderChase_JSON(t *testing.T) {
	payload := sampleChasePayload()
	var buf bytes.Buffer
	err := RenderChaseTo(&buf, payload, ChaseOptions{ForceJSON: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if data["total_alerts"] != float64(3) {
		t.Errorf("expected 3 total alerts, got %v", data["total_alerts"])
	}
	clusters, ok := data["clusters"].([]any)
	if !ok || len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %v", data["clusters"])
	}
}

func TestRenderChase_Pretty(t *testing.T) {
	payload := sampleChasePayload()
	var buf bytes.Buffer
	err := RenderChaseTo(&buf, payload, ChaseOptions{ForcePretty: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Southern Plains Severe Thunderstorms") {
		t.Errorf("expected cluster 1 title, got: %s", out)
	}
	if !strings.Contains(out, "Mid-Atlantic Coastal Storm") {
		t.Errorf("expected cluster 2 title, got: %s", out)
	}
	if !strings.Contains(out, "★ HIGHEST IMPACT") {
		t.Errorf("expected highest impact tag on cluster 1, got: %s", out)
	}
}

func TestRenderClusterDetail(t *testing.T) {
	payload := sampleChasePayload()
	var buf bytes.Buffer
	err := RenderClusterDetail(&buf, &payload.Clusters[0], ChaseOptions{ForcePretty: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Oklahoma, OK") {
		t.Errorf("expected cell 1 in detail, got: %s", out)
	}
	if !strings.Contains(out, "Tornado indicated") {
		t.Errorf("expected hazard text in detail, got: %s", out)
	}
}
