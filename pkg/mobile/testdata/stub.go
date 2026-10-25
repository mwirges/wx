// Command stub serves fixture HTTP for the mobile bridge test.
// WX_STUB_ORIGIN points wx at this process. Paths keep the original host.
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
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
	if body, ok := fixture(r); ok {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
		return
	}
	if os.Getenv("WX_STUB_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "stub: 404 %s %s?%s\n", r.Method, r.URL.Path, r.URL.RawQuery)
	}
	http.NotFound(w, r)
}

func fixture(r *http.Request) ([]byte, bool) {
	p := r.URL.Path
	switch {
	case strings.Contains(p, "open-meteo.com"):
		// Nowcast calls api.open-meteo.com/v1/forecast. That must not hit the
		// NWS forecast fixture. AQI is the only open-meteo payload we serve.
		if strings.Contains(p, "air-quality") {
			return air, true
		}
		return nil, false
	case strings.Contains(p, "nominatim.openstreetmap.org/search"):
		q := r.URL.Query().Get("q")
		if !strings.Contains(q, "New Haven") {
			return nil, false
		}
		return nominatim, true
	case strings.Contains(p, "/points/"):
		return points, true
	case strings.Contains(p, "observations"):
		return observation, true
	case strings.Contains(p, "/stations"):
		return stations, true
	case strings.Contains(p, "forecast/hourly"):
		return hourly, true
	case strings.Contains(p, "/forecast"):
		return daily, true
	case strings.Contains(p, "/alerts/"):
		return alerts, true
	default:
		return nil, false
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
