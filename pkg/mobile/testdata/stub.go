// Command stub serves fixture HTTP for the mobile bridge test.
// WX_STUB_ORIGIN points wx at this process. Paths keep the original host.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(ln.Addr().String())
	if err := http.Serve(ln, http.HandlerFunc(handle)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if payload, ctype, ok := fixture(r, body); ok {
		if ctype != "" {
			w.Header().Set("Content-Type", ctype)
		}
		_, _ = w.Write(payload)
		return
	}
	if os.Getenv("WX_STUB_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "stub: 404 %s %s?%s\n", r.Method, r.URL.Path, r.URL.RawQuery)
	}
	http.NotFound(w, r)
}

func fixture(r *http.Request, body []byte) ([]byte, string, bool) {
	p := r.URL.Path
	switch {
	case strings.Contains(p, "open-meteo.com"):
		// Nowcast calls api.open-meteo.com/v1/forecast. That must not hit the
		// NWS forecast fixture. AQI is a different host and path.
		if strings.Contains(p, "air-quality") {
			return air, "application/json", true
		}
		if strings.Contains(p, "/v1/forecast") {
			return nowcastBody, "application/json", true
		}
		return nil, "", false
	case strings.Contains(p, "nominatim.openstreetmap.org/search"):
		q := r.URL.Query().Get("q")
		if !strings.Contains(q, "New Haven") {
			return nil, "", false
		}
		return nominatim, "application/json", true
	case strings.Contains(p, "data.rcc-acis.org"):
		return acisFixture(p, body)
	case strings.Contains(p, "nhc.noaa.gov"):
		return nhcFixture(p)
	case strings.Contains(p, "mesonet.agron.iastate.edu") || strings.Contains(p, "opengeo.ncep.noaa.gov"):
		return radarPNG, "image/png", true
	case strings.Contains(p, "/points/"):
		return points, "application/json", true
	case strings.Contains(p, "observations"):
		return observation, "application/json", true
	case strings.Contains(p, "/stations"):
		return stations, "application/json", true
	case strings.Contains(p, "forecast/hourly"):
		return hourly, "application/json", true
	case strings.Contains(p, "/forecast"):
		return daily, "application/json", true
	case strings.Contains(p, "/alerts/"):
		return alerts, "application/json", true
	default:
		return nil, "", false
	}
}

func acisFixture(path string, body []byte) ([]byte, string, bool) {
	switch {
	case strings.Contains(path, "StnMeta"):
		return acisMeta, "application/json", true
	case strings.Contains(path, "StnData"):
		if strings.Contains(string(body), `"interval"`) {
			return acisRecords, "application/json", true
		}
		return acisNormals, "application/json", true
	case strings.Contains(path, "GridData"):
		return acisGrid, "application/json", true
	default:
		return nil, "", false
	}
}

func nhcFixture(path string) ([]byte, string, bool) {
	switch {
	case strings.Contains(path, "CurrentStorms.json"):
		return nhcStorms, "application/json", true
	case strings.Contains(path, "MIATWOAT"):
		return nhcAtlantic, "text/html; charset=utf-8", true
	case strings.Contains(path, "MIATWOEP"):
		return nhcPacific, "text/html; charset=utf-8", true
	default:
		return nil, "", false
	}
}

var nominatim = []byte(`[{
  "lat": "41.0",
  "lon": "-85.0",
  "display_name": "New Haven, Allen County, Indiana, United States",
  "address": {
    "city": "New Haven",
    "state": "IN",
    "country": "United States",
    "country_code": "us"
  }
}]`)

var points = []byte(`{
  "properties": {
    "gridId": "IWX",
    "gridX": 32,
    "gridY": 48,
    "forecast": "https://api.weather.gov/gridpoints/IWX/32,48/forecast",
    "forecastHourly": "https://api.weather.gov/gridpoints/IWX/32,48/forecast/hourly",
    "observationStations": "https://api.weather.gov/gridpoints/IWX/32,48/stations",
    "relativeLocation": {
      "properties": {"city": "New Haven", "state": "IN"}
    }
  }
}`)

var stations = []byte(`{
  "features": [{
    "properties": {
      "stationIdentifier": "KFWA",
      "name": "Fort Wayne International"
    }
  }]
}`)

var observation = []byte(`{
  "properties": {
    "station": "https://api.weather.gov/stations/KFWA",
    "timestamp": "2026-06-21T18:00:00Z",
    "textDescription": "Clear",
    "icon": "https://api.weather.gov/icons/land/day/skc",
    "temperature": {"value": 20.0, "unitCode": "wmoUnit:degC"},
    "windSpeed": {"value": 16.0, "unitCode": "wmoUnit:km_h-1"},
    "windDirection": {"value": 180},
    "windGust": {"value": null},
    "relativeHumidity": {"value": 40},
    "dewpoint": {"value": 6.0, "unitCode": "wmoUnit:degC"},
    "windChill": {"value": null},
    "heatIndex": {"value": null},
    "seaLevelPressure": {"value": 101325},
    "visibility": {"value": 16093.44}
  }
}`)

var daily = []byte(`{
  "properties": {
    "generatedAt": "2026-06-21T17:00:00Z",
    "periods": [{
      "name": "This Afternoon",
      "startTime": "2026-06-21T18:00:00Z",
      "endTime": "2026-06-22T00:00:00Z",
      "isDaytime": true,
      "temperature": 75,
      "temperatureUnit": "F",
      "windSpeed": "10 mph",
      "windDirection": "S",
      "shortForecast": "Sunny",
      "detailedForecast": "Sunny and warm.",
      "probabilityOfPrecipitation": {"unitCode": "wmoUnit:percent", "value": 0},
      "dewpoint": {"unitCode": "wmoUnit:degC", "value": 6},
      "relativeHumidity": {"unitCode": "wmoUnit:percent", "value": 40}
    }]
  }
}`)

var hourly = []byte(`{
  "properties": {
    "generatedAt": "2026-06-21T17:00:00Z",
    "periods": [
      {
        "name": "6 PM",
        "startTime": "2026-06-21T22:00:00Z",
        "endTime": "2026-06-21T23:00:00Z",
        "isDaytime": true,
        "temperature": 74,
        "temperatureUnit": "F",
        "windSpeed": "8 mph",
        "windDirection": "S",
        "shortForecast": "Sunny",
        "detailedForecast": "Sunny.",
        "probabilityOfPrecipitation": {"unitCode": "wmoUnit:percent", "value": 0},
        "dewpoint": {"unitCode": "wmoUnit:degC", "value": 6},
        "relativeHumidity": {"unitCode": "wmoUnit:percent", "value": 40}
      },
      {
        "name": "7 PM",
        "startTime": "2026-06-21T23:00:00Z",
        "endTime": "2026-06-22T00:00:00Z",
        "isDaytime": true,
        "temperature": 72,
        "temperatureUnit": "F",
        "windSpeed": "7 mph",
        "windDirection": "SW",
        "shortForecast": "Clear",
        "detailedForecast": "Clear.",
        "probabilityOfPrecipitation": {"unitCode": "wmoUnit:percent", "value": 0},
        "dewpoint": {"unitCode": "wmoUnit:degC", "value": 7},
        "relativeHumidity": {"unitCode": "wmoUnit:percent", "value": 45}
      }
    ]
  }
}`)

var alerts = []byte(`{
  "features": [{
    "id": "urn:alert:1",
    "properties": {
      "event": "Heat Advisory",
      "headline": "Heat Advisory for New Haven",
      "description": "Hot.",
      "instruction": "Drink water.",
      "severity": "Moderate",
      "urgency": "Expected",
      "effective": "2026-06-21T12:00:00Z",
      "expires": "2026-06-22T00:00:00Z",
      "areaDesc": "Allen, IN"
    }
  }]
}`)

var air = []byte(`{
  "current": {
    "time": "2026-06-21T18:00",
    "us_aqi": 42,
    "uv_index": 3.5,
    "pm2_5": 8.2,
    "pm10": 12.0,
    "ozone": 60.0,
    "nitrogen_dioxide": 5.0,
    "carbon_monoxide": 200.0,
    "sulphur_dioxide": 1.0
  }
}`)

// nowcastBody is a dry 4-hour window that stays inside the parser's
// [now-15m, now+8h] range for the life of this process. Both the CLI and the
// bridge therefore see the same intervals.
var nowcastBody = func() []byte {
	start := time.Now().UTC().Truncate(time.Minute).Add(time.Hour)
	const n = 16
	times := make([]string, n)
	zeros := make([]float64, n)
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		times[i] = start.Add(time.Duration(i) * 15 * time.Minute).Format("2006-01-02T15:04")
	}
	payload := map[string]any{
		"minutely_15": map[string]any{
			"time":          times,
			"precipitation": zeros,
			"rain":          zeros,
			"snowfall":      zeros,
			"weather_code":  codes,
		},
		"hourly": map[string]any{
			"time":                      []string{start.Format("2006-01-02T15:00")},
			"precipitation_probability": []int{0},
			"precipitation":             []float64{0},
			"rain":                      []float64{0},
			"snowfall":                  []float64{0},
			"weather_code":              []int{0},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return b
}()

var acisMeta = []byte(`{
  "meta": [{
    "name": "FORT WAYNE INTL AP",
    "state": "IN",
    "elev": 791,
    "ll": [-85.01, 41.01],
    "sids": ["14827 1", "FWA 3"],
    "uid": 31832,
    "valid_daterange": [["1940-01-01", "2026-12-31"]]
  }]
}`)

var acisNormals = []byte(`{
  "meta": {"name": "FORT WAYNE INTL AP"},
  "data": [
    ["2026-06-21", "80.0", "60.0", "0.12"]
  ]
}`)

var acisRecords = []byte(`{
  "meta": {"name": "FORT WAYNE INTL AP"},
  "data": [
    ["1990-06-21", "95", "40", "1.50"],
    ["2012-06-21", "90", "48", "0.25"]
  ]
}`)

var acisGrid = []byte(`{
  "data": [["2026-06-21", "80.0", "60.0"]]
}`)

var nhcStorms = []byte(`{"activeStorms":[]}`)

var nhcAtlantic = []byte(`<html><body><pre>
Tropical Weather Outlook
NWS National Hurricane Center Miami FL
200 AM EDT Mon Jun 21 2026

Central Subtropical Atlantic (AL91):
A trough of low pressure is moving east-northeast.
* Formation chance through 48 hours...medium...50 percent.
* Formation chance through 7 days...medium...60 percent.
</pre></body></html>`)

var nhcPacific = []byte(`<html><body><pre>
Tropical Weather Outlook
No tropical cyclones are expected during the next 7 days.
</pre></body></html>`)

// radarPNG is a small opaque image. Radar re-encodes whatever the server
// returns, so the CLI and the bridge match without fetching a 1600px mosaic.
var radarPNG = func() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+0] = 20
		img.Pix[i+1] = 40
		img.Pix[i+2] = 90
		img.Pix[i+3] = 255
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}()
