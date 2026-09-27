package openmeteo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mwirges/wx/internal/astro"
	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

const conditionsTTL = 5 * time.Minute

type currentConditionsResponse struct {
	Latitude            float64 `json:"latitude"`
	Longitude           float64 `json:"longitude"`
	UtcOffsetSeconds    int     `json:"utc_offset_seconds"`
	Timezone            string  `json:"timezone"`
	TimezoneAbbr        string  `json:"timezone_abbreviation"`
	Current             struct {
		Time                int64    `json:"time"`
		Temperature2m       float64  `json:"temperature_2m"`
		RelativeHumidity2m  float64  `json:"relative_humidity_2m"`
		DewPoint2m          float64  `json:"dew_point_2m"`
		ApparentTemperature float64  `json:"apparent_temperature"`
		IsDay               int      `json:"is_day"`
		WeatherCode         int      `json:"weather_code"`
		PressureMSL         float64  `json:"pressure_msl"`
		WindSpeed10m        float64  `json:"wind_speed_10m"`
		WindDirection10m    float64  `json:"wind_direction_10m"`
		WindGusts10m        *float64 `json:"wind_gusts_10m"`
		Visibility          *float64 `json:"visibility"`
	} `json:"current"`
}

// CurrentConditions fetches the latest observed weather conditions from Open-Meteo.
func (p *Provider) CurrentConditions(ctx context.Context, loc location.Location, c *cache.Cache) (*models.CurrentConditions, error) {
	cacheKey := fmt.Sprintf("openmeteo:conditions:%.4f,%.4f", loc.Lat, loc.Lon)

	var cached models.CurrentConditions
	if c.Get(cacheKey, &cached) {
		return &cached, nil
	}

	reqURL := fmt.Sprintf(
		"%s/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,relative_humidity_2m,dew_point_2m,apparent_temperature,is_day,weather_code,pressure_msl,wind_speed_10m,wind_direction_10m,wind_gusts_10m,visibility&timeformat=unixtime&timezone=auto",
		p.baseURL, loc.Lat, loc.Lon,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("openmeteo: build conditions request: %w", err)
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openmeteo: current conditions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("openmeteo: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var data currentConditionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("openmeteo: decode conditions: %w", err)
	}

	obsTime := time.Unix(data.Current.Time, 0)
	tempC := data.Current.Temperature2m
	humidity := data.Current.RelativeHumidity2m
	dewPointC := data.Current.DewPoint2m
	apparentTempC := data.Current.ApparentTemperature
	windKPH := data.Current.WindSpeed10m
	windDeg := data.Current.WindDirection10m
	pressureHPA := data.Current.PressureMSL

	displayName := loc.DisplayName
	if displayName == "" {
		displayName = fmt.Sprintf("%.4f, %.4f", loc.Lat, loc.Lon)
	}

	cond := models.CurrentConditions{
		StationID:     "openmeteo",
		StationName:   "Open-Meteo",
		ObservedAt:    obsTime,
		Location:      displayName,
		Description:   wmoToDescription(data.Current.WeatherCode),
		TempC:         &tempC,
		HumidityPct:   &humidity,
		DewPointC:     &dewPointC,
		FeelsLikeC:    &apparentTempC,
		WindKPH:       &windKPH,
		WindDegrees:   &windDeg,
		PressureHPA:   &pressureHPA,
		VisibilityM:   data.Current.Visibility,
		ConditionCode: wmoToConditionCode(data.Current.WeatherCode, data.Current.IsDay == 1),
	}

	if data.Current.WindGusts10m != nil && *data.Current.WindGusts10m > 0 {
		cond.WindGustKPH = data.Current.WindGusts10m
	}

	// Set WindChill or HeatIndex if conditions warrant
	if tempC <= 10.0 && windKPH > 4.8 {
		cond.WindChillC = &apparentTempC
	} else if tempC >= 26.7 && humidity >= 40.0 {
		cond.HeatIndexC = &apparentTempC
	}

	// Calculate solar ephemeris
	if a, err := astro.Calculate(loc.Lat, loc.Lon, obsTime); err == nil {
		cond.Astronomy = a
	}

	_ = c.Set(cacheKey, cond, conditionsTTL)
	return &cond, nil
}
