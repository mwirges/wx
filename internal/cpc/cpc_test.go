package cpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

func TestNormalizeCategory(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Above", "Above"},
		{"a", "Above"},
		{"BELOW", "Below"},
		{"b", "Below"},
		{"Normal", "Normal"},
		{"n", "Normal"},
		{"EC", "Equal Chances"},
		{"", "Equal Chances"},
	}
	for _, tc := range cases {
		got := normalizeCategory(tc.input)
		if got != tc.want {
			t.Errorf("normalizeCategory(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestAnalyzePatternShift(t *testing.T) {
	now := time.Now()
	// Scenario: Summer-to-autumn cold front pattern shift (Above/Wet -> Below/Dry)
	outlooks := []models.CPCOutlookItem{
		{
			Horizon:           "6-10 Day",
			StartDate:         now.Add(6 * 24 * time.Hour),
			EndDate:           now.Add(10 * 24 * time.Hour),
			TempCategory:      "Above",
			TempProbability:   60.0,
			PrecipCategory:    "Above",
			PrecipProbability: 45.0,
		},
		{
			Horizon:           "8-14 Day",
			StartDate:         now.Add(8 * 24 * time.Hour),
			EndDate:           now.Add(14 * 24 * time.Hour),
			TempCategory:      "Below",
			TempProbability:   55.0,
			PrecipCategory:    "Below",
			PrecipProbability: 40.0,
		},
	}

	shift := AnalyzePatternShift(outlooks)
	if !shift.HasShift {
		t.Error("expected HasShift = true")
	}
	if shift.TempShift != "cooling" {
		t.Errorf("TempShift = %q, want cooling", shift.TempShift)
	}
	if shift.PrecipShift != "drying" {
		t.Errorf("PrecipShift = %q, want drying", shift.PrecipShift)
	}
	if shift.Confidence != "High" {
		t.Errorf("Confidence = %q, want High", shift.Confidence)
	}
	if shift.Summary == "" {
		t.Error("Summary should not be empty")
	}
}

func TestFetchOutlooks_MockServer(t *testing.T) {
	tsStart := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC).UnixMilli()
	tsEnd := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC).UnixMilli()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/drought" {
			w.Write([]byte(`{
				"features": [{
					"attributes": {
						"outlook": "No_Drought",
						"target": "Oct 2026"
					}
				}]
			}`))
			return
		}

		// Outlook query
		w.Write([]byte(`{
			"features": [{
				"attributes": {
					"cat": "Below",
					"prob": 40.0,
					"start_date": ` + string(rune('0'+tsStart/1000000000000)) + `
				}
			}]
		}`))
	}))
	defer srv.Close()

	// More realistic mock response
	mockResp := `{
		"features": [{
			"attributes": {
				"cat": "Above",
				"prob": 55.0,
				"start_date": ` + formatMs(tsStart) + `,
				"end_date": ` + formatMs(tsEnd) + `
			}
		}]
	}`

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/drought" {
			w.Write([]byte(`{
				"features": [{
					"attributes": {
						"outlook": "No_Drought",
						"target": "Oct 2026"
					}
				}]
			}`))
			return
		}
		w.Write([]byte(mockResp))
	}))
	defer srv2.Close()

	cl := &Client{
		httpClient: srv2.Client(),
		c610Temp:   srv2.URL + "/610t",
		c610Precip: srv2.URL + "/610p",
		c814Temp:   srv2.URL + "/814t",
		c814Precip: srv2.URL + "/814p",
		cDrought:   srv2.URL + "/drought",
	}

	loc := location.Location{DisplayName: "Fort Wayne, IN", Lat: 41.08, Lon: -85.14, CountryCode: "US"}
	payload, err := cl.FetchOutlooks(context.Background(), loc, cache.NewNoOp())
	if err != nil {
		t.Fatalf("FetchOutlooks: %v", err)
	}

	if len(payload.Outlooks) != 2 {
		t.Fatalf("got %d outlooks, want 2", len(payload.Outlooks))
	}
	if payload.Outlooks[0].TempCategory != "Above" {
		t.Errorf("6-10 temp = %q, want Above", payload.Outlooks[0].TempCategory)
	}
	if payload.Drought == nil || payload.Drought.Status != "No Drought" {
		t.Errorf("drought = %v, want No Drought", payload.Drought)
	}
}

func formatMs(ms int64) string {
	return strconv.FormatInt(ms, 10)
}
