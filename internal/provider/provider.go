package provider

import (
	"context"
	"fmt"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

// WeatherProvider is the interface all weather data sources must implement.
type WeatherProvider interface {
	// Name returns a short identifier, e.g. "nws".
	Name() string
	// Supports returns true if this provider can serve data for the given location.
	Supports(loc location.Location) bool
	// CurrentConditions fetches the latest observed conditions.
	CurrentConditions(ctx context.Context, loc location.Location, c *cache.Cache) (*models.CurrentConditions, error)
	// Forecast fetches the forecast. Set hourly=true for hourly periods.
	Forecast(ctx context.Context, loc location.Location, hourly bool, c *cache.Cache) (*models.Forecast, error)
	// Alerts fetches active weather alerts. Returns an empty slice when none exist.
	Alerts(ctx context.Context, loc location.Location, c *cache.Cache) ([]models.Alert, error)
}

var registry []WeatherProvider

// Register adds a provider to the global registry.
// Typically called from a provider package's init() function.
func Register(p WeatherProvider) {
	registry = append(registry, p)
}

// ForLocation returns the first registered provider that supports the given location.
func ForLocation(loc location.Location) (WeatherProvider, error) {
	for _, p := range registry {
		if p.Supports(loc) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no weather provider available for location (country: %q) — only US locations are currently supported", loc.CountryCode)
}
