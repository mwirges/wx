package openmeteo

import (
	"net/http"
	"time"

	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/provider"
)

const defaultBaseURL = "https://api.open-meteo.com"

// Provider implements provider.WeatherProvider using the Open-Meteo API.
// Open-Meteo is free, keyless, and provides global coverage for coordinates worldwide.
type Provider struct {
	client  *http.Client
	baseURL string
}

// New returns a new Open-Meteo Provider with production endpoints.
func New() *Provider {
	return NewWithBaseURL(defaultBaseURL)
}

// NewWithBaseURL returns a new Provider pointing to a custom base URL (useful for tests).
func NewWithBaseURL(baseURL string) *Provider {
	return &Provider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL: baseURL,
	}
}

// init registers Open-Meteo as the global fallback provider.
func init() {
	provider.RegisterWithPriority(New(), provider.PriorityFallback)
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "openmeteo"
}

// Supports returns true for valid coordinate pairs worldwide.
func (p *Provider) Supports(loc location.Location) bool {
	// Latitude is in [-90, 90], Longitude in [-180, 180].
	if loc.Lat < -90 || loc.Lat > 90 || loc.Lon < -180 || loc.Lon > 180 {
		return false
	}
	// Coordinates must be non-zero (0,0 is Null Island and typically indicates missing coordinates).
	return loc.Lat != 0 || loc.Lon != 0
}
