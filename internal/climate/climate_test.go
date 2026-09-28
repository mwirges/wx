package climate

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/models"
)

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input    any
		expected float64
		ok       bool
	}{
		{72.5, 72.5, true},
		{72, 72.0, true},
		{"72.5", 72.5, true},
		{"72A", 72.0, true},
		{"M", 0.0, false},
		{"", 0.0, false},
		{nil, 0.0, false},
		{"T", 0.001, true},
	}

	for _, tt := range tests {
		val, ok := parseFloat(tt.input)
		if ok != tt.ok {
			t.Errorf("parseFloat(%v) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if ok && val != tt.expected {
			t.Errorf("parseFloat(%v) = %f, want %f", tt.input, val, tt.expected)
		}
	}
}

func TestComputeDeparture(t *testing.T) {
	normals := models.DailyNormals{
		NormalHighF: 70.0,
		NormalLowF:  50.0,
		NormalMeanF: 60.0,
	}

	// Warm anomaly: current 70°F (mean is 60°F -> +10°F)
	tempC := (70.0 - 32.0) * 5.0 / 9.0
	obs := &models.CurrentConditions{
		TempC: &tempC,
	}

	dep := ComputeDeparture(normals, obs)
	if dep == nil {
		t.Fatal("expected non-nil departure")
	}
	if *dep.DepartureCurrentF != 10.0 {
		t.Errorf("expected departure +10.0, got %f", *dep.DepartureCurrentF)
	}
	if !strings.Contains(dep.Summary, "Above Normal") {
		t.Errorf("expected summary to mention Above Normal, got %q", dep.Summary)
	}

	// Cool anomaly: current 50°F (-10°F below mean)
	tempCoolC := (50.0 - 32.0) * 5.0 / 9.0
	obsCool := &models.CurrentConditions{
		TempC: &tempCoolC,
	}
	depCool := ComputeDeparture(normals, obsCool)
	if *depCool.DepartureCurrentF != -10.0 {
		t.Errorf("expected departure -10.0, got %f", *depCool.DepartureCurrentF)
	}
	if !strings.Contains(depCool.Summary, "Below Normal") {
		t.Errorf("expected summary to mention Below Normal, got %q", depCool.Summary)
	}
}

func TestClient_Fetch_Mock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "StnMeta") {
			var resp stnMetaResponse
			resp.Meta = append(resp.Meta, struct {
				Name           string     `json:"name"`
				State          string     `json:"state"`
				Elev           *float64   `json:"elev"`
				LL             []float64  `json:"ll"`
				Sids           []string   `json:"sids"`
				UID            int        `json:"uid"`
				ValidDateRange [][]string `json:"valid_daterange"`
			}{
				Name:           "FORT WAYNE INTL AP",
				State:          "IN",
				LL:             []float64{-85.20, 40.97},
				Sids:           []string{"14827 1", "FWA 3"},
				UID:            31832,
				ValidDateRange: [][]string{{"1897-01-01", "2026-09-28"}},
			})
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "StnData") {
			bodyBytes, _ := io.ReadAll(r.Body)
			bodyStr := string(bodyBytes)

			if strings.Contains(bodyStr, `"interval":[1,0,0]`) {
				// Records response
				var resp stnDataResponse
				resp.Meta.Name = "FORT WAYNE INTL AP"
				resp.Data = [][]any{
					{"1950-09-27", "85", "45", "0.00"},
					{"1985-09-27", "72", "42", "0.50"},
					{"2021-09-27", "89", "55", "0.10"}, // record high 89
					{"2024-09-27", "60", "36", "1.25"}, // record low 36, record precip 1.25
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}

			// Normals response
			var resp stnDataResponse
			resp.Meta.Name = "FORT WAYNE INTL AP"
			resp.Data = [][]any{
				{"2026-09-27", "72.0", "48.0", "0.10"},
				{"2026-09-28", "71.0", "47.0", "0.10"},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	targetDate := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	tempCache, err := cache.NewWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create temp cache: %v", err)
	}

	report, err := client.Fetch(context.Background(), 41.0, -85.1, "Fort Wayne, IN", nil, targetDate, tempCache)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	if report.StationID != "14827 1" {
		t.Errorf("expected station 14827 1, got %q", report.StationID)
	}
	if report.TodayNormals.NormalHighF != 72.0 {
		t.Errorf("expected normal high 72.0, got %f", report.TodayNormals.NormalHighF)
	}
	if report.TodayNormals.NormalLowF != 48.0 {
		t.Errorf("expected normal low 48.0, got %f", report.TodayNormals.NormalLowF)
	}
	if report.Records.RecordHigh.ValueF != 89.0 {
		t.Errorf("expected record high 89.0, got %f", report.Records.RecordHigh.ValueF)
	}
	if len(report.Records.RecordHigh.Years) != 1 || report.Records.RecordHigh.Years[0] != 2021 {
		t.Errorf("expected record high year 2021, got %v", report.Records.RecordHigh.Years)
	}
	if report.Records.RecordLow.ValueF != 36.0 {
		t.Errorf("expected record low 36.0, got %f", report.Records.RecordLow.ValueF)
	}
	if report.Records.RecordPrecip.ValueIn != 1.25 {
		t.Errorf("expected record precip 1.25, got %f", report.Records.RecordPrecip.ValueIn)
	}

	// Secondary call should hit cache
	report2, err := client.Fetch(context.Background(), 41.0, -85.1, "Fort Wayne, IN", nil, targetDate, tempCache)
	if err != nil {
		t.Fatalf("secondary Fetch error: %v", err)
	}
	if report2.TodayNormals.NormalHighF != report.TodayNormals.NormalHighF {
		t.Errorf("cached report mismatch")
	}
}
