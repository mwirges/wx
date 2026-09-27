package openmeteo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

const (
	forecastTTL       = 1 * time.Hour
	hourlyForecastTTL = 30 * time.Minute
)

type dailyForecastResponse struct {
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	UtcOffsetSeconds int     `json:"utc_offset_seconds"`
	Timezone         string  `json:"timezone"`
	TimezoneAbbr     string  `json:"timezone_abbreviation"`
	Daily            struct {
		Time                        []int64   `json:"time"`
		WeatherCode                 []int     `json:"weather_code"`
		Temperature2mMax            []float64 `json:"temperature_2m_max"`
		Temperature2mMin            []float64 `json:"temperature_2m_min"`
		PrecipitationProbabilityMax []float64 `json:"precipitation_probability_max"`
		WindSpeed10mMax             []float64 `json:"wind_speed_10m_max"`
		WindDirection10mDominant    []float64 `json:"wind_direction_10m_dominant"`
	} `json:"daily"`
}

type hourlyForecastResponse struct {
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	UtcOffsetSeconds int     `json:"utc_offset_seconds"`
	Timezone         string  `json:"timezone"`
	TimezoneAbbr     string  `json:"timezone_abbreviation"`
	Hourly           struct {
		Time                     []int64   `json:"time"`
		Temperature2m            []float64 `json:"temperature_2m"`
		RelativeHumidity2m       []float64 `json:"relative_humidity_2m"`
		DewPoint2m               []float64 `json:"dew_point_2m"`
		ApparentTemperature      []float64 `json:"apparent_temperature"`
		PrecipitationProbability []float64 `json:"precipitation_probability"`
		WeatherCode              []int     `json:"weather_code"`
		WindSpeed10m             []float64 `json:"wind_speed_10m"`
		WindDirection10m         []float64 `json:"wind_direction_10m"`
		IsDay                    []int     `json:"is_day"`
	} `json:"hourly"`
}

// Forecast fetches the 7-day or hourly forecast from Open-Meteo.
func (p *Provider) Forecast(ctx context.Context, loc location.Location, hourly bool, c *cache.Cache) (*models.Forecast, error) {
	if hourly {
		return p.fetchHourlyForecast(ctx, loc, c)
	}
	return p.fetchDailyForecast(ctx, loc, c)
}

func (p *Provider) fetchDailyForecast(ctx context.Context, loc location.Location, c *cache.Cache) (*models.Forecast, error) {
	cacheKey := fmt.Sprintf("openmeteo:forecast:%.4f,%.4f", loc.Lat, loc.Lon)

	var cached models.Forecast
	if c.Get(cacheKey, &cached) {
		return &cached, nil
	}

	reqURL := fmt.Sprintf(
		"%s/v1/forecast?latitude=%.4f&longitude=%.4f&daily=weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max,wind_speed_10m_max,wind_direction_10m_dominant&forecast_days=7&timeformat=unixtime&timezone=auto",
		p.baseURL, loc.Lat, loc.Lon,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("openmeteo: build daily forecast request: %w", err)
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openmeteo: daily forecast: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("openmeteo: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var data dailyForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("openmeteo: decode daily forecast: %w", err)
	}

	tz, err := time.LoadLocation(data.Timezone)
	if err != nil {
		tz = time.FixedZone(data.TimezoneAbbr, data.UtcOffsetSeconds)
	}

	now := time.Now().In(tz)
	periods := make([]models.Period, 0, len(data.Daily.Time)*2)

	for i, unixTime := range data.Daily.Time {
		dayStart := time.Unix(unixTime, 0).In(tz)
		dayEnd := dayStart.Add(18 * time.Hour)    // 6 PM local
		nightEnd := dayStart.Add(30 * time.Hour)  // 6 AM next day

		var pop *float64
		if i < len(data.Daily.PrecipitationProbabilityMax) {
			val := data.Daily.PrecipitationProbabilityMax[i]
			pop = &val
		}

		desc := wmoToDescription(data.Daily.WeatherCode[i])
		windKPH := data.Daily.WindSpeed10mMax[i]
		windDir := degreesToCompass(data.Daily.WindDirection10mDominant[i])

		if i == 0 {
			// Today: if already past 6 PM, only emit Tonight
			if !now.After(dayEnd) {
				periods = append(periods, models.Period{
					Name:                       "Today",
					StartTime:                  dayStart,
					EndTime:                    dayEnd,
					IsDaytime:                  true,
					TempC:                      data.Daily.Temperature2mMax[i],
					WindKPH:                    windKPH,
					WindDir:                    windDir,
					ShortDesc:                  desc,
					DetailedDesc:               fmt.Sprintf("High around %.0f°C. Winds %s at %.0f km/h.", data.Daily.Temperature2mMax[i], windDir, windKPH),
					ProbabilityOfPrecipitation: pop,
				})
			}
			periods = append(periods, models.Period{
				Name:                       "Tonight",
				StartTime:                  dayEnd,
				EndTime:                    nightEnd,
				IsDaytime:                  false,
				TempC:                      data.Daily.Temperature2mMin[i],
				WindKPH:                    windKPH,
				WindDir:                    windDir,
				ShortDesc:                  desc,
				DetailedDesc:               fmt.Sprintf("Low around %.0f°C. Winds %s at %.0f km/h.", data.Daily.Temperature2mMin[i], windDir, windKPH),
				ProbabilityOfPrecipitation: pop,
			})
		} else {
			weekday := dayStart.Format("Monday")
			periods = append(periods, models.Period{
				Name:                       weekday,
				StartTime:                  dayStart,
				EndTime:                    dayEnd,
				IsDaytime:                  true,
				TempC:                      data.Daily.Temperature2mMax[i],
				WindKPH:                    windKPH,
				WindDir:                    windDir,
				ShortDesc:                  desc,
				DetailedDesc:               fmt.Sprintf("High around %.0f°C. Winds %s at %.0f km/h.", data.Daily.Temperature2mMax[i], windDir, windKPH),
				ProbabilityOfPrecipitation: pop,
			})
			periods = append(periods, models.Period{
				Name:                       weekday + " Night",
				StartTime:                  dayEnd,
				EndTime:                    nightEnd,
				IsDaytime:                  false,
				TempC:                      data.Daily.Temperature2mMin[i],
				WindKPH:                    windKPH,
				WindDir:                    windDir,
				ShortDesc:                  desc,
				DetailedDesc:               fmt.Sprintf("Low around %.0f°C. Winds %s at %.0f km/h.", data.Daily.Temperature2mMin[i], windDir, windKPH),
				ProbabilityOfPrecipitation: pop,
			})
		}
	}

	fc := models.Forecast{
		GeneratedAt: now,
		Periods:     periods,
	}

	_ = c.Set(cacheKey, fc, forecastTTL)
	return &fc, nil
}

func (p *Provider) fetchHourlyForecast(ctx context.Context, loc location.Location, c *cache.Cache) (*models.Forecast, error) {
	cacheKey := fmt.Sprintf("openmeteo:forecast-hourly:%.4f,%.4f", loc.Lat, loc.Lon)

	var cached models.Forecast
	if c.Get(cacheKey, &cached) {
		return &cached, nil
	}

	reqURL := fmt.Sprintf(
		"%s/v1/forecast?latitude=%.4f&longitude=%.4f&hourly=temperature_2m,relative_humidity_2m,dew_point_2m,apparent_temperature,precipitation_probability,weather_code,wind_speed_10m,wind_direction_10m,is_day&forecast_hours=48&timeformat=unixtime&timezone=auto",
		p.baseURL, loc.Lat, loc.Lon,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("openmeteo: build hourly forecast request: %w", err)
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openmeteo: hourly forecast: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("openmeteo: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var data hourlyForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("openmeteo: decode hourly forecast: %w", err)
	}

	tz, err := time.LoadLocation(data.Timezone)
	if err != nil {
		tz = time.FixedZone(data.TimezoneAbbr, data.UtcOffsetSeconds)
	}

	now := time.Now().In(tz)
	periods := make([]models.Period, 0, len(data.Hourly.Time))

	for i, unixTime := range data.Hourly.Time {
		t := time.Unix(unixTime, 0).In(tz)

		var pop *float64
		if i < len(data.Hourly.PrecipitationProbability) {
			val := data.Hourly.PrecipitationProbability[i]
			pop = &val
		}

		var dewPoint *float64
		if i < len(data.Hourly.DewPoint2m) {
			val := data.Hourly.DewPoint2m[i]
			dewPoint = &val
		}

		var humidity *float64
		if i < len(data.Hourly.RelativeHumidity2m) {
			val := data.Hourly.RelativeHumidity2m[i]
			humidity = &val
		}

		isDay := true
		if i < len(data.Hourly.IsDay) {
			isDay = data.Hourly.IsDay[i] == 1
		}

		windDir := degreesToCompass(data.Hourly.WindDirection10m[i])
		desc := wmoToDescription(data.Hourly.WeatherCode[i])

		periods = append(periods, models.Period{
			Name:                       t.Format("Mon 3 PM"),
			StartTime:                  t,
			EndTime:                    t.Add(time.Hour),
			IsDaytime:                  isDay,
			TempC:                      data.Hourly.Temperature2m[i],
			WindKPH:                    data.Hourly.WindSpeed10m[i],
			WindDir:                    windDir,
			ShortDesc:                  desc,
			ProbabilityOfPrecipitation: pop,
			DewPointC:                  dewPoint,
			HumidityPct:                humidity,
		})
	}

	fc := models.Forecast{
		GeneratedAt: now,
		Periods:     periods,
	}

	_ = c.Set(cacheKey, fc, hourlyForecastTTL)
	return &fc, nil
}
