package location

import (
	"context"
	"regexp"
	"time"
)

// Location holds resolved geographic coordinates and metadata.
type Location struct {
	Lat         float64
	Lon         float64
	DisplayName string // "Kansas City, MO", "64101", "Chicago, Illinois"
	CountryCode string // ISO 3166-1 alpha-2, e.g. "US"
	City        string
	State       string // full state name from geocoder
}

// Cacher is the minimal cache interface used by location resolution.
// Satisfied by *cache.Cache (real or no-op).
type Cacher interface {
	Get(key string, v any) bool
	Set(key string, v any, ttl time.Duration) error
}

var zipRe = regexp.MustCompile(`^\d{5}(-\d{4})?$`)

func isZip(s string) bool {
	return zipRe.MatchString(s)
}

// Resolve returns a Location from the input string.
//   - Empty input → auto-detect via IP geolocation (ipinfo.io)
//   - 5-digit zip (optionally +4) → Nominatim postal code lookup
//   - Anything else → treated as "City, ST" free-text Nominatim lookup
func Resolve(ctx context.Context, input string, c Cacher) (Location, error) {
	if input == "" {
		return resolveByIP(ctx, c)
	}
	if isZip(input) {
		return resolveByZip(ctx, input, c)
	}
	return resolveByCityState(ctx, input, c)
}
