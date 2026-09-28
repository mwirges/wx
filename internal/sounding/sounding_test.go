package sounding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleNSHARP = `%TITLE%
 ILX   260928/0000 

   LEVEL       HGHT       TEMP       DWPT       WDIR       WSPD
-------------------------------------------------------------------
%RAW%
 1000.00,    142.00,     22.00,     14.00,    180.00,     12.00
  925.00,    815.00,     18.00,     12.00,    200.00,     25.00
  850.00,   1530.00,     14.00,      8.00,    220.00,     35.00
  700.00,   3143.00,      6.00,     -2.00,    240.00,     45.00
  500.00,   5820.00,    -12.00,    -20.00,    260.00,     55.00
  300.00,   9610.00,    -38.00,    -50.00,    270.00,     75.00
%END%

----- Parcel Information-----
*** SFC PARCEL ***
SBCAPE:              2450 J/kg
SBCINH:               -45 J/kg
SBLI:                  -6 C

*** 100mb MIXED LAYER PARCEL ***
MLCAPE:              1980 J/kg
MLCINH:               -60 J/kg
MLLI:                  -5 C

*** MU PARCEL IN LOWEST 400mb ***
MUCAPE:              2800 J/kg
MUCINH:               -25 J/kg
MULI:                  -7 C

----- Misc Thermodynamic Parameters -----
Precip Water:    1.65 in
Melting Level:    12500.0
DCAPE:        850.0
700-500mb   18 C      6.8 C/km
850-500mb   26 C      6.1 C/km

----- Vertical Shear -----
0-1 km BWD   22 kt
0-3 km BWD   38 kt
0-6 km BWD   48 kt
0-1 km SRH   185 ms/s2
0-3 km SRH   290 ms/s2

----- Composite Parameters -----
Effective-layer SCP   5.4
Effective-layer STP   2.1
SHIP                  1.8
`

func TestFindClosestStation(t *testing.T) {
	// Norman, OK (35.22, -97.44) should find OUN
	stn, dist := FindClosestStation(35.22, -97.44)
	if stn.ID != "OUN" {
		t.Errorf("expected station OUN, got %s", stn.ID)
	}
	if dist > 30.0 {
		t.Errorf("expected distance < 30km, got %f", dist)
	}

	// Fort Wayne, IN (40.97, -85.20)
	stnFWA, _ := FindClosestStation(40.97, -85.20)
	if stnFWA.ID != "ILN" && stnFWA.ID != "DTX" && stnFWA.ID != "ILX" {
		t.Errorf("expected Midwestern station (ILN/DTX/ILX), got %s", stnFWA.ID)
	}
}

func TestParseNSHARPSounding(t *testing.T) {
	stn := Station{ID: "ILX", Name: "Lincoln", State: "IL"}
	report, err := parseNSHARPSounding(sampleNSHARP, stn, "26092800_OBS")
	if err != nil {
		t.Fatalf("unexpected error parsing NSHARP: %v", err)
	}

	idx := report.Indices
	if idx.SBCAPE == nil || *idx.SBCAPE != 2450.0 {
		t.Errorf("expected SBCAPE 2450, got %v", idx.SBCAPE)
	}
	if idx.MLCAPE == nil || *idx.MLCAPE != 1980.0 {
		t.Errorf("expected MLCAPE 1980, got %v", idx.MLCAPE)
	}
	if idx.MUCAPE == nil || *idx.MUCAPE != 2800.0 {
		t.Errorf("expected MUCAPE 2800, got %v", idx.MUCAPE)
	}
	if idx.SBCIN == nil || *idx.SBCIN != -45.0 {
		t.Errorf("expected SBCIN -45, got %v", idx.SBCIN)
	}
	if idx.SBLI == nil || *idx.SBLI != -6.0 {
		t.Errorf("expected SBLI -6, got %v", idx.SBLI)
	}
	if idx.PWATIn == nil || *idx.PWATIn != 1.65 {
		t.Errorf("expected PWAT 1.65 in, got %v", idx.PWATIn)
	}
	if idx.BulkShear06KT == nil || *idx.BulkShear06KT != 48.0 {
		t.Errorf("expected 0-6km BWD 48 kt, got %v", idx.BulkShear06KT)
	}
	if idx.SRH01 == nil || *idx.SRH01 != 185.0 {
		t.Errorf("expected 0-1km SRH 185, got %v", idx.SRH01)
	}
	if idx.STP == nil || *idx.STP != 2.1 {
		t.Errorf("expected STP 2.1, got %v", idx.STP)
	}
	if idx.SCP == nil || *idx.SCP != 5.4 {
		t.Errorf("expected SCP 5.4, got %v", idx.SCP)
	}
	if idx.LapseRate700_500 == nil || *idx.LapseRate700_500 != 6.8 {
		t.Errorf("expected Lapse Rate 6.8, got %v", idx.LapseRate700_500)
	}

	if len(report.Levels) == 0 {
		t.Fatal("expected non-empty levels table")
	}

	summarizeConvectiveEnvironment(&report.Indices)
	if !strings.Contains(report.Indices.InstabilitySummary, "Strong") {
		t.Errorf("expected Strong Instability, got %q", report.Indices.InstabilitySummary)
	}
	if !strings.Contains(report.Indices.ShearSummary, "Supercells Favored") {
		t.Errorf("expected Supercells Favored, got %q", report.Indices.ShearSummary)
	}
}

func TestClient_Fetch_MockSPC(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><a href='/exper/soundings/26092800_OBS/'><img src='sndgmap.gif'></a></html>`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "ILX.txt") {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(sampleNSHARP))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, ts.URL)
	report, err := client.Fetch(context.Background(), 40.15, -89.34, "Lincoln, IL", "ILX", nil)
	if err != nil {
		t.Fatalf("unexpected fetch error: %v", err)
	}

	if report.StationID != "ILX" {
		t.Errorf("expected station ILX, got %s", report.StationID)
	}
	if report.Indices.SBCAPE == nil || *report.Indices.SBCAPE != 2450.0 {
		t.Errorf("expected CAPE 2450, got %v", report.Indices.SBCAPE)
	}
}

