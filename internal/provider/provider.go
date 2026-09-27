package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

// WeatherProvider is the interface all weather data sources must implement.
type WeatherProvider interface {
	// Name returns a short identifier, e.g. "nws", "openmeteo".
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

// Priority levels for provider registration.
const (
	PrioritySpecialized = 100 // High-fidelity national/regional services (e.g. NWS for US)
	PriorityStandard    = 50  // Standard or secondary providers
	PriorityFallback    = 10  // Broad global fallback sources (e.g. Open-Meteo)
)

type providerEntry struct {
	provider WeatherProvider
	priority int
}

var (
	registryMu sync.RWMutex
	registry   []providerEntry
)

// Register adds a provider to the global registry with PriorityStandard.
// Typically called from a provider package's init() function.
func Register(p WeatherProvider) {
	RegisterWithPriority(p, PriorityStandard)
}

// RegisterWithPriority adds or updates a provider with an explicit priority tier.
// Providers with higher priority values are evaluated first when selecting for a location.
func RegisterWithPriority(p WeatherProvider, priority int) {
	registryMu.Lock()
	defer registryMu.Unlock()

	for i, entry := range registry {
		if strings.EqualFold(entry.provider.Name(), p.Name()) {
			registry[i] = providerEntry{provider: p, priority: priority}
			sortRegistryLocked()
			return
		}
	}
	registry = append(registry, providerEntry{provider: p, priority: priority})
	sortRegistryLocked()
}

func sortRegistryLocked() {
	sort.SliceStable(registry, func(i, j int) bool {
		return registry[i].priority > registry[j].priority
	})
}

// Get returns the registered provider with the given name (case-insensitive), if found.
func Get(name string) (WeatherProvider, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	for _, entry := range registry {
		if strings.EqualFold(entry.provider.Name(), name) {
			return entry.provider, true
		}
	}
	return nil, false
}

// All returns a slice of all registered providers in descending priority order.
func All() []WeatherProvider {
	registryMu.RLock()
	defer registryMu.RUnlock()

	res := make([]WeatherProvider, len(registry))
	for i, entry := range registry {
		res[i] = entry.provider
	}
	return res
}

// ForLocation returns the highest-priority registered provider that supports the given location.
func ForLocation(loc location.Location) (WeatherProvider, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	for _, entry := range registry {
		if entry.provider.Supports(loc) {
			return entry.provider, nil
		}
	}
	return nil, fmt.Errorf("no weather provider available for location (country: %q)", loc.CountryCode)
}

// ForLocationWithPreference returns the preferred provider if specified and supported,
// or falls back to automatic priority-based provider selection for the location.
func ForLocationWithPreference(loc location.Location, preferredName string) (WeatherProvider, error) {
	preferredName = strings.TrimSpace(preferredName)
	if preferredName == "" {
		return ForLocation(loc)
	}

	p, ok := Get(preferredName)
	if !ok {
		return nil, fmt.Errorf("unknown weather provider %q", preferredName)
	}
	if !p.Supports(loc) {
		return nil, fmt.Errorf("weather provider %q does not support location (country: %q)", p.Name(), loc.CountryCode)
	}
	return p, nil
}

// FallbackForLocation finds another provider that supports the location, excluding the failed provider.
func FallbackForLocation(loc location.Location, failedProviderName string) (WeatherProvider, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	for _, entry := range registry {
		if !strings.EqualFold(entry.provider.Name(), failedProviderName) && entry.provider.Supports(loc) {
			return entry.provider, nil
		}
	}
	return nil, fmt.Errorf("no fallback weather provider available for location (country: %q)", loc.CountryCode)
}

// ClearRegistry removes all registered providers (primarily for tests).
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = nil
}
