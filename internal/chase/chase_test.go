package chase

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mwirges/wx/internal/cache"
)

const sampleAlertsResponse = `{
  "features": [
    {
      "id": "urn:oid:2.49.0.1.840.0.tornado1",
      "geometry": {
        "type": "Polygon",
        "coordinates": [[
          [-97.4, 35.2],
          [-97.2, 35.2],
          [-97.2, 35.4],
          [-97.4, 35.4],
          [-97.4, 35.2]
        ]]
      },
      "properties": {
        "event": "Tornado Warning",
        "headline": "Tornado Warning for Oklahoma County",
        "areaDesc": "Oklahoma, OK; Cleveland, OK",
        "description": "At 400 PM, a tornado was reported. HAZARD...Damaging tornado and quarter size hail. SOURCE...Radar indicated.",
        "severity": "Extreme",
        "urgency": "Immediate",
        "certainty": "Observed",
        "senderName": "NWS Norman OK",
        "geocode": {
          "UGC": ["OKC109", "OKC027"]
        },
        "effective": "2026-09-27T21:00:00Z",
        "expires": "2026-09-27T21:45:00Z"
      }
    },
    {
      "id": "urn:oid:2.49.0.1.840.0.severe1",
      "geometry": {
        "type": "Polygon",
        "coordinates": [[
          [-97.1, 35.5],
          [-96.9, 35.5],
          [-96.9, 35.7],
          [-97.1, 35.7],
          [-97.1, 35.5]
        ]]
      },
      "properties": {
        "event": "Severe Thunderstorm Warning",
        "headline": "Severe Thunderstorm Warning for Lincoln County",
        "areaDesc": "Lincoln, OK",
        "description": "HAZARD...70 mph wind gusts and golf ball size hail. WIND GUST...70MPH. HAIL THREAT...1.75IN.",
        "severity": "Severe",
        "urgency": "Immediate",
        "certainty": "Observed",
        "senderName": "NWS Norman OK",
        "geocode": {
          "UGC": ["OKC081"]
        },
        "effective": "2026-09-27T21:10:00Z",
        "expires": "2026-09-27T22:00:00Z"
      }
    },
    {
      "id": "urn:oid:2.49.0.1.840.0.coastal1",
      "geometry": null,
      "properties": {
        "event": "Coastal Flood Warning",
        "headline": "Coastal Flood Warning for Cape May",
        "areaDesc": "Cape May, NJ",
        "description": "Up to 2 feet of inundation above ground level expected.",
        "severity": "Severe",
        "urgency": "Expected",
        "certainty": "Likely",
        "senderName": "NWS Mount Holly NJ",
        "geocode": {
          "UGC": ["NJZ024"]
        },
        "effective": "2026-09-27T18:00:00Z",
        "expires": "2026-09-28T02:00:00Z"
      }
    }
  ]
}`

func TestFetchClusters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleAlertsResponse))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	c := cache.NewNoOp()

	payload, err := client.FetchClusters(context.Background(), c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if payload.TotalAlerts != 3 {
		t.Errorf("expected 3 alerts, got %d", payload.TotalAlerts)
	}
	if payload.TotalClusters != 2 {
		t.Fatalf("expected 2 clusters (Oklahoma severe cluster + New Jersey coastal cluster), got %d", payload.TotalClusters)
	}

	// Cluster 1 should be the Oklahoma severe cluster (Score = 100 + 40 = 140)
	c1 := payload.Clusters[0]
	if c1.Score < 100 {
		t.Errorf("expected high score for Oklahoma tornadic cluster, got %d", c1.Score)
	}
	if !strings.Contains(c1.Name, "Tornad") && !strings.Contains(c1.Name, "Severe") {
		t.Errorf("expected severe/tornadic cluster name, got %q", c1.Name)
	}
	if len(c1.Cells) != 2 {
		t.Errorf("expected 2 cells in Oklahoma cluster, got %d", len(c1.Cells))
	}
	if c1.NearestRadar == "" {
		t.Error("expected non-empty nearest radar")
	}

	// Cell hazard extraction
	cell1 := c1.Cells[0]
	if cell1.Event != "Tornado Warning" {
		t.Errorf("expected top cell to be Tornado Warning, got %q", cell1.Event)
	}
	if !strings.Contains(cell1.HazardText, "Damaging tornado") {
		t.Errorf("expected hazard text with tornado, got %q", cell1.HazardText)
	}

	cell2 := c1.Cells[1]
	if !strings.Contains(cell2.HazardText, "70 mph") {
		t.Errorf("expected hazard text with 70 mph wind, got %q", cell2.HazardText)
	}

	// Cluster 2 should be the NJ Coastal Flood cluster
	c2 := payload.Clusters[1]
	if len(c2.Cells) != 1 {
		t.Errorf("expected 1 cell in NJ cluster, got %d", len(c2.Cells))
	}
	if !strings.Contains(c2.Name, "Coastal") && !strings.Contains(c2.Name, "Nor'easter") {
		t.Errorf("expected coastal cluster name, got %q", c2.Name)
	}
}

func TestHaversineKm(t *testing.T) {
	// NYC to Philadelphia ~130 km
	d := haversineKm(40.7128, -74.0060, 39.9526, -75.1652)
	if d < 120 || d > 140 {
		t.Errorf("expected distance ~130 km, got %.1f", d)
	}
}

func TestScoreAlert(t *testing.T) {
	tests := []struct {
		event string
		want  int
	}{
		{"Tornado Warning", 100},
		{"Severe Thunderstorm Warning", 40},
		{"Flash Flood Warning", 40},
		{"Coastal Flood Warning", 20},
		{"Flood Warning", 15},
		{"Tornado Watch", 10},
		{"Flood Watch", 10},
	}
	for _, tt := range tests {
		got := scoreAlert(tt.event)
		if got != tt.want {
			t.Errorf("scoreAlert(%q) = %d, want %d", tt.event, got, tt.want)
		}
	}
}
