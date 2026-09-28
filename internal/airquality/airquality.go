package airquality

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/models"
)

const (
	defaultBaseURL = "https://air-quality-api.open-meteo.com"
	cacheTTL       = 30 * time.Minute
)

type airQualityResponse struct {
	Current struct {
		Time            string   `json:"time"`
		USAQI           *int     `json:"us_aqi"`
		UVIndex         *float64 `json:"uv_index"`
		PM25            *float64 `json:"pm2_5"`
		PM10            *float64 `json:"pm10"`
		Ozone           *float64 `json:"ozone"`
		NitrogenDioxide *float64 `json:"nitrogen_dioxide"`
		CarbonMonoxide  *float64 `json:"carbon_monoxide"`
		SulphurDioxide  *float64 `json:"sulphur_dioxide"`
	} `json:"current"`
}

// Client fetches air quality telemetry from Open-Meteo.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient returns a new air quality Client.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

var defaultClient = NewClient("")

// Fetch returns AirQuality telemetry for the given coordinates, checking cache first.
func Fetch(ctx context.Context, lat, lon float64, c *cache.Cache) (*models.AirQuality, error) {
	return defaultClient.Fetch(ctx, lat, lon, c)
}

// Fetch returns AirQuality telemetry for the given coordinates, checking cache first.
func (cl *Client) Fetch(ctx context.Context, lat, lon float64, c *cache.Cache) (*models.AirQuality, error) {
	cacheKey := fmt.Sprintf("airquality:%.4f,%.4f", lat, lon)
	var cached models.AirQuality
	if c != nil && c.Get(cacheKey, &cached) {
		return &cached, nil
	}

	reqURL := fmt.Sprintf(
		"%s/v1/air-quality?latitude=%.4f&longitude=%.4f&current=us_aqi,pm2_5,pm10,carbon_monoxide,nitrogen_dioxide,sulphur_dioxide,ozone,uv_index",
		cl.baseURL, lat, lon,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("airquality: build request: %w", err)
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")
	req.Header.Set("Accept", "application/json")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("airquality: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("airquality: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var data airQualityResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("airquality: decode response: %w", err)
	}

	aq := &models.AirQuality{
		AQI:     data.Current.USAQI,
		UVIndex: data.Current.UVIndex,
		PM25:    data.Current.PM25,
		PM10:    data.Current.PM10,
		O3:      data.Current.Ozone,
		NO2:     data.Current.NitrogenDioxide,
		CO:      data.Current.CarbonMonoxide,
		SO2:     data.Current.SulphurDioxide,
	}

	if aq.AQI != nil {
		aq.Category = models.AQICategory(*aq.AQI)
	}
	if aq.UVIndex != nil {
		aq.UVCategory = models.UVCategory(*aq.UVIndex)
	}

	if c != nil {
		_ = c.Set(cacheKey, aq, cacheTTL)
	}
	return aq, nil
}
