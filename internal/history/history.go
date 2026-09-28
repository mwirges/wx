package history

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/provider/openmeteo"
)

const (
	defaultBaseURL = "https://api.open-meteo.com"
	cacheTTL       = 1 * time.Hour
	maxPastDays    = 92
	defaultDays    = 7
)

type openMeteoHistoryResponse struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Elevation *float64 `json:"elevation"`
	Daily     struct {
		Time             []string   `json:"time"`
		Temperature2mMax []*float64 `json:"temperature_2m_max"`
		Temperature2mMin []*float64 `json:"temperature_2m_min"`
		ApparentTempMax  []*float64 `json:"apparent_temperature_max"`
		ApparentTempMin  []*float64 `json:"apparent_temperature_min"`
		PrecipitationSum []*float64 `json:"precipitation_sum"`
		WeatherCode      []*int     `json:"weather_code"`
		WindSpeed10mMax  []*float64 `json:"wind_speed_10m_max"`
	} `json:"daily"`
}

// Client fetches historical weather observations from Open-Meteo.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient returns a new history Client.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

var defaultClient = NewClient("")

// Fetch returns historical weather observations for the given coordinates and past days.
func Fetch(ctx context.Context, lat, lon float64, days int, c *cache.Cache) (*models.HistoricalWeather, error) {
	return defaultClient.Fetch(ctx, lat, lon, days, c)
}

// Fetch returns historical weather observations for the given coordinates and past days, checking cache first.
func (cl *Client) Fetch(ctx context.Context, lat, lon float64, days int, c *cache.Cache) (*models.HistoricalWeather, error) {
	if days <= 0 {
		days = defaultDays
	}
	if days > maxPastDays {
		days = maxPastDays
	}

	cacheKey := fmt.Sprintf("history:%d:%.4f,%.4f", days, lat, lon)
	var cached models.HistoricalWeather
	if c != nil && c.Get(cacheKey, &cached) {
		return &cached, nil
	}

	reqURL := fmt.Sprintf(
		"%s/v1/forecast?latitude=%.4f&longitude=%.4f&past_days=%d&forecast_days=0&daily=temperature_2m_max,temperature_2m_min,apparent_temperature_max,apparent_temperature_min,precipitation_sum,weather_code,wind_speed_10m_max&timezone=auto",
		cl.baseURL, lat, lon, days,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("history: build request: %w", err)
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")
	req.Header.Set("Accept", "application/json")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("history: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("history: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var data openMeteoHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("history: decode response: %w", err)
	}

	n := len(data.Daily.Time)
	historyDays := make([]models.HistoryDay, 0, n)

	for i := 0; i < n; i++ {
		t, _ := time.Parse("2006-01-02", data.Daily.Time[i])

		day := models.HistoryDay{
			Date: t,
		}

		if i < len(data.Daily.Temperature2mMax) {
			day.TempMaxC = data.Daily.Temperature2mMax[i]
		}
		if i < len(data.Daily.Temperature2mMin) {
			day.TempMinC = data.Daily.Temperature2mMin[i]
		}
		if i < len(data.Daily.ApparentTempMax) {
			day.ApparentMaxC = data.Daily.ApparentTempMax[i]
		}
		if i < len(data.Daily.ApparentTempMin) {
			day.ApparentMinC = data.Daily.ApparentTempMin[i]
		}
		if i < len(data.Daily.PrecipitationSum) {
			day.PrecipSumMM = data.Daily.PrecipitationSum[i]
		}
		if i < len(data.Daily.WindSpeed10mMax) {
			day.WindMaxKPH = data.Daily.WindSpeed10mMax[i]
		}
		if i < len(data.Daily.WeatherCode) && data.Daily.WeatherCode[i] != nil {
			code := *data.Daily.WeatherCode[i]
			day.ConditionCode = openmeteo.WMOToConditionCode(code, true)
			day.Description = openmeteo.WMOToDescription(code)
		}

		historyDays = append(historyDays, day)
	}

	summary := computeSummary(historyDays)

	res := &models.HistoricalWeather{
		Latitude:  data.Latitude,
		Longitude: data.Longitude,
		Elevation: data.Elevation,
		Days:      historyDays,
		Summary:   summary,
	}

	if c != nil {
		_ = c.Set(cacheKey, res, cacheTTL)
	}

	return res, nil
}

func computeSummary(days []models.HistoryDay) models.HistorySummary {
	var (
		sumHigh, sumLow, sumPrecip float64
		countHigh, countLow        int
		hasPrecip                  bool
		maxWind                    *float64
	)

	for _, d := range days {
		if d.TempMaxC != nil {
			sumHigh += *d.TempMaxC
			countHigh++
		}
		if d.TempMinC != nil {
			sumLow += *d.TempMinC
			countLow++
		}
		if d.PrecipSumMM != nil {
			sumPrecip += *d.PrecipSumMM
			hasPrecip = true
		}
		if d.WindMaxKPH != nil {
			if maxWind == nil || *d.WindMaxKPH > *maxWind {
				val := *d.WindMaxKPH
				maxWind = &val
			}
		}
	}

	summary := models.HistorySummary{
		DaysCount:  len(days),
		MaxWindKPH: maxWind,
	}

	if countHigh > 0 {
		avg := sumHigh / float64(countHigh)
		summary.AvgTempMaxC = &avg
	}
	if countLow > 0 {
		avg := sumLow / float64(countLow)
		summary.AvgTempMinC = &avg
	}
	if hasPrecip {
		summary.TotalPrecipMM = &sumPrecip
	}

	return summary
}
