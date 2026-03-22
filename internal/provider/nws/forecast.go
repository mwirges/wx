package nws

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

type forecastResponse struct {
	Properties struct {
		GeneratedAt string           `json:"generatedAt"`
		Periods     []forecastPeriod `json:"periods"`
	} `json:"properties"`
}

type forecastPeriod struct {
	Name             string `json:"name"`
	StartTime        string `json:"startTime"`
	EndTime          string `json:"endTime"`
	IsDaytime        bool   `json:"isDaytime"`
	Temperature      int    `json:"temperature"`
	TemperatureUnit  string `json:"temperatureUnit"` // "F" or "C"
	WindSpeed        string `json:"windSpeed"`       // "10 mph", "5 to 15 mph"
	WindDirection    string `json:"windDirection"`   // "NW"
	ShortForecast    string `json:"shortForecast"`
	DetailedForecast string `json:"detailedForecast"`
}

// Forecast fetches the 7-day or hourly forecast.
// Cached for 1 hour.
func (p *Provider) Forecast(ctx context.Context, loc location.Location, hourly bool, c *cache.Cache) (*models.Forecast, error) {
	cacheKeySuffix := "forecast"
	if hourly {
		cacheKeySuffix = "forecast-hourly"
	}
	cacheKey := fmt.Sprintf("nws:%s:%.4f,%.4f", cacheKeySuffix, loc.Lat, loc.Lon)

	var fc models.Forecast
	if c.Get(cacheKey, &fc) {
		return &fc, nil
	}

	grid, err := p.getGridInfo(ctx, loc, c)
	if err != nil {
		return nil, err
	}

	fcURL := grid.ForecastURL
	if hourly {
		fcURL = grid.ForecastHourlyURL
	}

	var resp forecastResponse
	if err := p.client.get(ctx, fcURL, &resp); err != nil {
		return nil, fmt.Errorf("nws: forecast: %w", err)
	}

	genAt, _ := time.Parse(time.RFC3339, resp.Properties.GeneratedAt)

	periods := make([]models.Period, 0, len(resp.Properties.Periods))
	for _, fp := range resp.Properties.Periods {
		start, _ := time.Parse(time.RFC3339, fp.StartTime)
		end, _ := time.Parse(time.RFC3339, fp.EndTime)

		tempC := float64(fp.Temperature)
		if fp.TemperatureUnit == "F" {
			tempC = fahrenheitToCelsius(tempC)
		}

		periods = append(periods, models.Period{
			Name:         fp.Name,
			StartTime:    start,
			EndTime:      end,
			IsDaytime:    fp.IsDaytime,
			TempC:        tempC,
			WindKPH:      parseWindKPH(fp.WindSpeed),
			WindDir:      fp.WindDirection,
			ShortDesc:    fp.ShortForecast,
			DetailedDesc: fp.DetailedForecast,
		})
	}

	fc = models.Forecast{
		GeneratedAt: genAt,
		Periods:     periods,
	}

	_ = c.Set(cacheKey, fc, 1*time.Hour)
	return &fc, nil
}

// parseWindKPH extracts a km/h value from an NWS wind speed string.
// NWS strings look like "10 mph", "5 to 15 mph", "Calm".
// Returns the upper bound converted to km/h.
func parseWindKPH(s string) float64 {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "calm" {
		return 0
	}
	// Remove unit suffix
	s = strings.TrimSuffix(s, " mph")
	s = strings.TrimSuffix(s, " km/h")
	s = strings.TrimSuffix(s, " knots")

	// "5 to 15" → take upper bound
	if idx := strings.Index(s, " to "); idx >= 0 {
		s = strings.TrimSpace(s[idx+4:])
	}

	val, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return mphToKPH(val)
}

func fahrenheitToCelsius(f float64) float64 {
	return (f - 32) * 5 / 9
}

func mphToKPH(mph float64) float64 {
	return mph * 1.60934
}
