package nws

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

type obsValue struct {
	Value    *float64 `json:"value"`
	UnitCode string   `json:"unitCode"`
}

type observationResponse struct {
	Properties struct {
		Station         string   `json:"station"`
		Timestamp       string   `json:"timestamp"`
		TextDescription string   `json:"textDescription"`
		Icon            string   `json:"icon"`
		Temperature     obsValue `json:"temperature"`
		WindSpeed       obsValue `json:"windSpeed"`
		WindDirection   obsValue `json:"windDirection"`
		WindGust        obsValue `json:"windGust"`
		RelativeHumidity obsValue `json:"relativeHumidity"`
		Dewpoint        obsValue `json:"dewpoint"`
		WindChill       obsValue `json:"windChill"`
		HeatIndex       obsValue `json:"heatIndex"`
		SeaLevelPressure obsValue `json:"seaLevelPressure"`
		Visibility      obsValue `json:"visibility"`
	} `json:"properties"`
}

// CurrentConditions fetches the latest observation from the nearest station.
// Cached for 10 minutes.
func (p *Provider) CurrentConditions(ctx context.Context, loc location.Location, c *cache.Cache) (*models.CurrentConditions, error) {
	cacheKey := fmt.Sprintf("nws:conditions:%.4f,%.4f", loc.Lat, loc.Lon)

	var cond models.CurrentConditions
	if c.Get(cacheKey, &cond) {
		return &cond, nil
	}

	grid, err := p.getGridInfo(ctx, loc, c)
	if err != nil {
		return nil, err
	}

	obsURL := fmt.Sprintf("%s/stations/%s/observations/latest", p.client.baseURL, grid.StationID)
	var obs observationResponse
	if err := p.client.get(ctx, obsURL, &obs); err != nil {
		return nil, fmt.Errorf("nws: current conditions: %w", err)
	}

	t, _ := time.Parse(time.RFC3339, obs.Properties.Timestamp)

	displayName := loc.DisplayName
	if grid.CityState != "" {
		displayName = grid.CityState
	}

	// seaLevelPressure is in Pascals; convert to hPa (÷100).
	var pressureHPA *float64
	if obs.Properties.SeaLevelPressure.Value != nil {
		v := *obs.Properties.SeaLevelPressure.Value / 100
		pressureHPA = &v
	}

	// NWS observations report temperature in Celsius (wmoUnit:degC),
	// wind speed in km/h (wmoUnit:km_h-1), and visibility in meters.
	cond = models.CurrentConditions{
		StationID:     grid.StationID,
		StationName:   grid.StationID,
		ObservedAt:    t,
		Location:      displayName,
		Description:   obs.Properties.TextDescription,
		ConditionCode: parseConditionCode(obs.Properties.Icon),

		TempC:       obs.Properties.Temperature.Value,
		WindChillC:  obs.Properties.WindChill.Value,
		HeatIndexC:  obs.Properties.HeatIndex.Value,

		DewPointC:   obs.Properties.Dewpoint.Value,
		HumidityPct: obs.Properties.RelativeHumidity.Value,

		WindKPH:     obs.Properties.WindSpeed.Value,
		WindGustKPH: obs.Properties.WindGust.Value,
		WindDegrees: obs.Properties.WindDirection.Value,

		PressureHPA: pressureHPA,
		VisibilityM: obs.Properties.Visibility.Value,
	}

	_ = c.Set(cacheKey, cond, 10*time.Minute)
	return &cond, nil
}

// parseConditionCode extracts a normalized condition code from an NWS icon URL.
// URL format: https://api.weather.gov/icons/land/{day|night}/{code}[,{pct}]...
func parseConditionCode(iconURL string) string {
	u, err := url.Parse(iconURL)
	if err != nil || u.Path == "" {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	// Expected: ["icons", "land", "{day|night}", "{code}[,{pct}]"]
	if len(parts) < 4 {
		return ""
	}
	timeOfDay := parts[2]
	codeField := parts[3]
	// Strip probability suffix: "tsra,40" → "tsra"
	if idx := strings.IndexByte(codeField, ','); idx >= 0 {
		codeField = codeField[:idx]
	}
	return mapNWSIconCode(codeField, timeOfDay)
}

func mapNWSIconCode(code, timeOfDay string) string {
	night := timeOfDay == "night"
	switch code {
	case "skc", "hot", "cold":
		if night {
			return "clear-night"
		}
		return "clear-day"
	case "few":
		if night {
			return "partly-cloudy-night"
		}
		return "partly-cloudy-day"
	case "sct":
		if night {
			return "partly-cloudy-night"
		}
		return "partly-cloudy-day"
	case "bkn", "ovc":
		return "cloudy"
	case "wind_skc", "wind_few", "wind_sct", "wind_bkn", "wind_ovc":
		return "wind"
	case "snow", "blizzard":
		return "snow"
	case "rain_snow", "rain_sleet", "snow_sleet", "fzra", "rain_fzra", "snow_fzra", "sleet":
		return "sleet"
	case "rain", "rain_showers", "rain_showers_hi":
		return "rain"
	case "tsra", "tsra_sct", "tsra_hi", "tornado":
		return "thunder"
	case "dust", "smoke", "haze", "fog":
		return "fog"
	default:
		return ""
	}
}
