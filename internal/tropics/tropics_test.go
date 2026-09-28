package tropics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

const mockStormsJSON = `{
  "activeStorms": [
    {
      "id": "al062026",
      "binNumber": "AT1",
      "name": "Fay",
      "classification": "TD",
      "intensity": "30",
      "pressure": "1012",
      "latitude": "27.9N",
      "longitude": "44.1W",
      "latitudeNumeric": 27.9,
      "longitudeNumeric": -44.1,
      "movementDir": 210,
      "movementSpeed": 7,
      "lastUpdate": "2026-09-28T09:00:00.000Z",
      "publicAdvisory": {
        "advNum": "033",
        "issuance": "2026-09-28T09:00:00.000Z",
        "url": "%s/fay.html"
      }
    },
    {
      "id": "ep172026",
      "binNumber": "EP2",
      "name": "Polo",
      "classification": "HU",
      "intensity": "100",
      "pressure": "958",
      "latitude": "23.6N",
      "longitude": "113.8W",
      "latitudeNumeric": 23.6,
      "longitudeNumeric": -113.8,
      "movementDir": 15,
      "movementSpeed": 10,
      "lastUpdate": "2026-09-28T09:00:00.000Z",
      "publicAdvisory": {
        "advNum": "031",
        "issuance": "2026-09-28T09:00:00.000Z",
        "url": "%s/polo.html"
      }
    }
  ]
}`

const mockPoloAdvisory = `<html><body><pre>
BULLETIN
Hurricane Polo Advisory Number  31
NWS National Hurricane Center Miami FL       EP172026
200 AM MST Mon Sep 28 2026

...LIFE-THREATENING WINDS AND FLASH FLOODS EXPECTED OVER PORTIONS OF BAJA CALIFORNIA SUR...

SUMMARY OF 200 AM MST...0900 UTC...INFORMATION
----------------------------------------------
LOCATION...23.6N 113.8W
ABOUT 125 MI...200 KM SW OF CABO SAN LAZARO MEXICO
MAXIMUM SUSTAINED WINDS...115 MPH...185 KM/H
PRESENT MOVEMENT...NNE OR 15 DEGREES AT 10 MPH...17 KM/H
MINIMUM CENTRAL PRESSURE...958 MB...28.29 INCHES

WATCHES AND WARNINGS
--------------------
A Hurricane Warning is in effect for...
* The east coast of Baja California Sur
</pre></body></html>`

const mockFayAdvisory = `<html><body><pre>
BULLETIN
Tropical Depression Fay Advisory Number  33
NWS National Hurricane Center Miami FL       AL062026
900 AM GMT Mon Sep 28 2026

...FAY PERSISTS IN THE CENTRAL ATLANTIC...

SUMMARY OF 900 AM GMT...0900 UTC...INFORMATION
----------------------------------------------
LOCATION...27.9N 44.1W
ABOUT 1210 MI...1945 KM WSW OF THE AZORES
MAXIMUM SUSTAINED WINDS...35 MPH...55 KM/H
PRESENT MOVEMENT...SSW OR 210 DEGREES AT 7 MPH...11 KM/H
MINIMUM CENTRAL PRESSURE...1012 MB...29.89 INCHES

WATCHES AND WARNINGS
--------------------
There are no coastal watches or warnings in effect.
</pre></body></html>`

const mockAtlanticTWO = `<html><body><pre>
Tropical Weather Outlook
NWS National Hurricane Center Miami FL
200 AM EDT Mon Sep 28 2026

Central Subtropical Atlantic (AL91):
A trough of low pressure located several hundred miles east-northeast of Bermuda is moving quickly east-northeastward.
* Formation chance through 48 hours...medium...50 percent.
* Formation chance through 7 days...medium...60 percent.
</pre></body></html>`

func TestFetchTropics(t *testing.T) {
	mux := http.NewServeMux()

	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/storms.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, mockStormsJSON, server.URL, server.URL)
	})
	mux.HandleFunc("/polo.html", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(mockPoloAdvisory))
	})
	mux.HandleFunc("/fay.html", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(mockFayAdvisory))
	})
	mux.HandleFunc("/atlantic_two.html", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(mockAtlanticTWO))
	})
	mux.HandleFunc("/pacific_two.html", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(""))
	})

	client := NewClient(server.Client()).WithURLs(
		server.URL+"/storms.json",
		server.URL+"/atlantic_two.html",
		server.URL+"/pacific_two.html",
	)

	loc := &location.Location{
		DisplayName: "Miami, FL",
		Lat:         25.7617,
		Lon:         -80.1918,
	}

	report, err := client.FetchTropics(context.Background(), loc)
	if err != nil {
		t.Fatalf("FetchTropics failed: %v", err)
	}

	if report.TotalActive != 2 {
		t.Errorf("expected 2 active storms, got %d", report.TotalActive)
	}

	var polo *models.TropicalStorm
	var fay *models.TropicalStorm
	for i := range report.Storms {
		if report.Storms[i].Name == "Polo" {
			polo = &report.Storms[i]
		} else if report.Storms[i].Name == "Fay" {
			fay = &report.Storms[i]
		}
	}

	if polo == nil {
		t.Fatal("expected storm Polo in report")
	}
	if polo.Category != 3 {
		t.Errorf("expected Polo to be Category 3, got %d", polo.Category)
	}
	if polo.IntensityKt != 100 {
		t.Errorf("expected Polo intensity 100kt, got %d", polo.IntensityKt)
	}
	if polo.MovementCompass != "NNE" {
		t.Errorf("expected Polo movement compass NNE, got %s", polo.MovementCompass)
	}
	if polo.DistanceMiles == nil || *polo.DistanceMiles <= 0 {
		t.Errorf("expected valid distance in miles for Polo, got %v", polo.DistanceMiles)
	}

	if fay == nil {
		t.Fatal("expected storm Fay in report")
	}
	if fay.Category != 0 {
		t.Errorf("expected Fay category 0 (TD), got %d", fay.Category)
	}
	if fay.IntensityKt != 30 {
		t.Errorf("expected Fay intensity 30kt, got %d", fay.IntensityKt)
	}
	if fay.MovementCompass != "SSW" {
		t.Errorf("expected Fay movement compass SSW, got %s", fay.MovementCompass)
	}

	if len(report.Disturbances) != 1 {
		t.Errorf("expected 1 disturbance (AL91), got %d", len(report.Disturbances))
	} else {
		d := report.Disturbances[0]
		if d.ID != "AL91" {
			t.Errorf("expected disturbance ID AL91, got %s", d.ID)
		}
		if d.Chance48h != 50 || d.Chance7d != 60 {
			t.Errorf("expected 50%% / 60%% chances, got %d%% / %d%%", d.Chance48h, d.Chance7d)
		}
	}
}

func TestClassifyIntensity(t *testing.T) {
	tests := []struct {
		class    string
		kt       int
		wantCat  int
		wantName string
	}{
		{"HU", 140, 5, "Category 5 (Major Hurricane)"},
		{"HU", 120, 4, "Category 4 (Major Hurricane)"},
		{"HU", 100, 3, "Category 3 (Major Hurricane)"},
		{"HU", 90, 2, "Category 2 (Hurricane)"},
		{"HU", 70, 1, "Category 1 (Hurricane)"},
		{"TS", 50, 0, "Tropical Storm"},
		{"TD", 30, 0, "Tropical Depression"},
		{"PTC", 35, 0, "Potential Tropical Cyclone"},
	}

	for _, tt := range tests {
		cat, label := classifyIntensity(tt.class, tt.kt)
		if cat != tt.wantCat || label != tt.wantName {
			t.Errorf("classifyIntensity(%s, %d) = (%d, %q), want (%d, %q)", tt.class, tt.kt, cat, label, tt.wantCat, tt.wantName)
		}
	}
}

func TestDegreesToCompass(t *testing.T) {
	tests := []struct {
		deg  int
		want string
	}{
		{0, "N"},
		{15, "NNE"},
		{45, "NE"},
		{90, "E"},
		{180, "S"},
		{210, "SSW"},
		{270, "W"},
		{315, "NW"},
		{360, "N"},
	}

	for _, tt := range tests {
		if got := degreesToCompass(tt.deg); got != tt.want {
			t.Errorf("degreesToCompass(%d) = %q, want %q", tt.deg, got, tt.want)
		}
	}
}
