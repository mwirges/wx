package spc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
)

func TestCategoryFromDN(t *testing.T) {
	tests := []struct {
		dn       int
		label    string
		label2   string
		wantCode string
		wantDN   int
	}{
		{0, "", "", "NONE", 0},
		{2, "TSTM", "General Thunderstorms Risk", "TSTM", 2},
		{3, "MRGL", "1 - Marginal Risk", "MRGL", 3},
		{4, "SLGT", "2 - Slight Risk", "SLGT", 4},
		{5, "ENH", "3 - Enhanced Risk", "ENH", 5},
		{6, "MDT", "4 - Moderate Risk", "MDT", 6},
		{8, "HIGH", "5 - High Risk", "HIGH", 8},
		{0, "ENH", "", "ENH", 5},
	}

	for _, tt := range tests {
		cat := CategoryFromDN(tt.dn, tt.label, tt.label2)
		if cat.Code != tt.wantCode {
			t.Errorf("CategoryFromDN(%d, %q, %q).Code = %q, want %q", tt.dn, tt.label, tt.label2, cat.Code, tt.wantCode)
		}
		if cat.DN != tt.wantDN {
			t.Errorf("CategoryFromDN(%d, %q, %q).DN = %d, want %d", tt.dn, tt.label, tt.label2, cat.DN, tt.wantDN)
		}
	}
}

func TestFetchSPCOutlooks(t *testing.T) {
	// Mock MapServer handler
	mapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case path == "/1/query": // Day 1 Categorical
			fmt.Fprint(w, `{
				"features": [
					{
						"attributes": {
							"dn": 4,
							"label": "SLGT",
							"label2": "2 - Slight Risk",
							"valid": "202609272000",
							"expire": "202609281200",
							"issue": "202609271957"
						}
					}
				]
			}`)
		case path == "/3/query": // Day 1 Tornado
			fmt.Fprint(w, `{
				"features": [
					{
						"attributes": {
							"dn": 5,
							"label": "0.05",
							"label2": "5%"
						}
					}
				]
			}`)
		case path == "/5/query": // Day 1 Hail
			fmt.Fprint(w, `{
				"features": [
					{
						"attributes": {
							"dn": 15,
							"label": "SIGN",
							"label2": "Significant Hail"
						}
					}
				]
			}`)
		case path == "/7/query": // Day 1 Wind
			fmt.Fprint(w, `{
				"features": [
					{
						"attributes": {
							"dn": 15,
							"label": "0.15",
							"label2": "15%"
						}
					}
				]
			}`)
		case path == "/9/query": // Day 2 Categorical
			fmt.Fprint(w, `{
				"features": [
					{
						"attributes": {
							"dn": 3,
							"label": "MRGL",
							"label2": "1 - Marginal Risk"
						}
					}
				]
			}`)
		case path == "/17/query": // Day 3 Categorical
			fmt.Fprint(w, `{
				"features": [
					{
						"attributes": {
							"dn": 2,
							"label": "TSTM",
							"label2": "General Thunderstorms"
						}
					}
				]
			}`)
		default:
			fmt.Fprint(w, `{"features": []}`)
		}
	}))
	defer mapServer.Close()

	// Mock MCD MapServer handler
	mcdServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"type": "FeatureCollection",
			"features": [
				{
					"geometry": {
						"type": "Polygon",
						"coordinates": [[[-101.8, 42.6], [-100.9, 42.1], [-101.4, 41.7], [-101.8, 42.6]]]
					},
					"properties": {
						"objectid": 1,
						"name": "MD 2329",
						"folderpath": "MD 2329 Active Till 0100 UTC",
						"popupinfo": "http://spc.example.com/md/md2329.html",
						"idp_filedate": 1790552857000
					}
				}
			]
		}`)
	}))
	defer mcdServer.Close()

	// Mock MCD Web Server handler
	mdWebServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><pre>
Mesoscale Discussion 2329
NWS Storm Prediction Center Norman OK

Areas affected...Parts of the NE Sand Hills region

Concerning...Severe potential...Watch unlikely 

Valid 272327Z - 280100Z

Probability of Watch Issuance...20 percent

SUMMARY...An isolated supercell may continue through sunset, with additional development possible.
</pre></body></html>`)
	}))
	defer mdWebServer.Close()

	// Mock Watches server handler
	watchesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"features": [
				{
					"id": "urn:oid:watch-512",
					"properties": {
						"event": "Severe Thunderstorm Watch",
						"headline": "Severe Thunderstorm Watch 512 is in effect",
						"areaDesc": "West Central Nebraska",
						"effective": "2026-09-27T20:00:00Z",
						"expires": "2026-09-28T02:00:00Z",
						"severity": "Severe",
						"urgency": "Expected",
						"parameters": {
							"VTEC": ["/O.NEW.KWNS.SV.A.0512.000000T0000Z-260928T0200Z/"]
						},
						"geocode": {
							"UGC": ["NEC031", "NEC033"]
						}
					}
				}
			]
		}`)
	}))
	defer watchesServer.Close()

	client := NewCustomClient(
		mapServer.URL,
		mcdServer.URL,
		watchesServer.URL,
		mdWebServer.URL,
		&http.Client{},
	)

	loc := location.Location{
		Lat:         42.0,
		Lon:         -101.5,
		DisplayName: "Cherry County, NE",
		CountryCode: "US",
	}

	payload, err := client.FetchSPCOutlooks(context.Background(), loc, cache.NewNoOp())
	if err != nil {
		t.Fatalf("FetchSPCOutlooks error: %v", err)
	}

	if payload.Day1.Category.Code != "SLGT" {
		t.Errorf("Day1 category code = %q, want SLGT", payload.Day1.Category.Code)
	}
	if payload.Day1.TornadoProb != "5%" {
		t.Errorf("Day1 tornado prob = %q, want 5%%", payload.Day1.TornadoProb)
	}
	if !payload.Day1.HailSig {
		t.Errorf("Day1 hail sig = false, want true")
	}
	if payload.Day2.Category.Code != "MRGL" {
		t.Errorf("Day2 category code = %q, want MRGL", payload.Day2.Category.Code)
	}
	if payload.Day3.Category.Code != "TSTM" {
		t.Errorf("Day3 category code = %q, want TSTM", payload.Day3.Category.Code)
	}

	// Verify MCDs
	if len(payload.ActiveMCDs) != 1 {
		t.Fatalf("len(ActiveMCDs) = %d, want 1", len(payload.ActiveMCDs))
	}
	mcd := payload.ActiveMCDs[0]
	if mcd.ID != 2329 {
		t.Errorf("MCD ID = %d, want 2329", mcd.ID)
	}
	if mcd.Concerning != "Severe potential...Watch unlikely" {
		t.Errorf("MCD Concerning = %q, want 'Severe potential...Watch unlikely'", mcd.Concerning)
	}
	if mcd.WatchProbability != "20 percent" {
		t.Errorf("MCD WatchProbability = %q, want '20 percent'", mcd.WatchProbability)
	}
	if mcd.AreasAffected != "Parts of the NE Sand Hills region" {
		t.Errorf("MCD AreasAffected = %q, want 'Parts of the NE Sand Hills region'", mcd.AreasAffected)
	}

	// Verify Watches
	if len(payload.ActiveWatches) != 1 {
		t.Fatalf("len(ActiveWatches) = %d, want 1", len(payload.ActiveWatches))
	}
	watch := payload.ActiveWatches[0]
	if watch.WatchNumber != 512 {
		t.Errorf("WatchNumber = %d, want 512", watch.WatchNumber)
	}
	if watch.Type != "Severe Thunderstorm Watch" {
		t.Errorf("Watch Type = %q, want 'Severe Thunderstorm Watch'", watch.Type)
	}
	if len(watch.States) == 0 || watch.States[0] != "NE" {
		t.Errorf("Watch States = %v, want ['NE']", watch.States)
	}

	// Verify Summary
	if payload.ConvectiveSummary == "" {
		t.Errorf("ConvectiveSummary is empty")
	}
}

func TestFetchSPCOutlooks_Quiet(t *testing.T) {
	emptyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "query") {
			fmt.Fprint(w, `{"features": []}`)
		} else {
			fmt.Fprint(w, `{"type": "FeatureCollection", "features": []}`)
		}
	}))
	defer emptyServer.Close()

	client := NewCustomClient(
		emptyServer.URL,
		emptyServer.URL+"/mcd",
		emptyServer.URL+"/watches",
		emptyServer.URL+"/md",
		&http.Client{},
	)

	loc := location.Location{Lat: 40.0, Lon: -85.0, DisplayName: "Indiana"}
	payload, err := client.FetchSPCOutlooks(context.Background(), loc, cache.NewNoOp())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if payload.Day1.Category.Code != "NONE" {
		t.Errorf("Day1 code = %q, want NONE", payload.Day1.Category.Code)
	}
	if payload.Day1.TornadoProb != "None" {
		t.Errorf("Day1 tornado prob = %q, want None", payload.Day1.TornadoProb)
	}
	if len(payload.ActiveMCDs) != 0 {
		t.Errorf("len(ActiveMCDs) = %d, want 0", len(payload.ActiveMCDs))
	}
	if len(payload.ActiveWatches) != 0 {
		t.Errorf("len(ActiveWatches) = %d, want 0", len(payload.ActiveWatches))
	}
}

func TestFetchNationalMaxRisk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"features": [
				{"attributes": {"dn": 2, "label": "TSTM"}},
				{"attributes": {"dn": 5, "label": "ENH", "label2": "3 - Enhanced Risk"}},
				{"attributes": {"dn": 4, "label": "SLGT"}}
			]
		}`)
	}))
	defer server.Close()

	client := NewCustomClient(
		server.URL,
		server.URL,
		server.URL,
		server.URL,
		&http.Client{},
	)

	maxRisk, err := client.FetchNationalMaxRisk(context.Background(), cache.NewNoOp())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if maxRisk.Code != "ENH" {
		t.Errorf("maxRisk.Code = %q, want ENH", maxRisk.Code)
	}
	if maxRisk.DN != 5 {
		t.Errorf("maxRisk.DN = %d, want 5", maxRisk.DN)
	}
}

