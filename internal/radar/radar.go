// Package radar defines types and interfaces for fetching and displaying
// radar imagery in the terminal. Providers register themselves via init()
// and are selected by location at runtime.
package radar

import (
	"context"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
)

// Product identifies the type of radar data to fetch.
type Product string

const (
	// ProductCompositeReflectivity shows the maximum reflectivity from all
	// elevation scans — best general-purpose view of precipitation extent.
	// Uses the national CONUS mosaic (all stations merged).
	ProductCompositeReflectivity Product = "composite-reflectivity"

	// ProductBaseReflectivity shows only the lowest radar tilt (0.5°).
	// Uses single-station RIDGE data from the nearest NEXRAD site.
	ProductBaseReflectivity Product = "base-reflectivity"

	// ProductStormRelativeVelocity is velocity adjusted for storm motion,
	// highlighting rotation within storm cells.
	// Single-station RIDGE product.
	ProductStormRelativeVelocity Product = "storm-relative-velocity"

	// ProductEchoTops shows the maximum height of detected radar echoes
	// (in thousands of feet). Indicates storm intensity and vertical extent.
	// Uses NWS MRMS Enhanced Echo Tops national mosaic via WMS.
	ProductEchoTops Product = "echo-tops"

	// ProductPrecipType shows surface precipitation classification
	// (rain, snow, ice pellets, freezing rain) via NWS MRMS national WMS.
	ProductPrecipType Product = "precip-type"

	// ProductOneHourPrecip shows 1-hour precipitation accumulation (QPE).
	// Single-station RIDGE product (N1P).
	ProductOneHourPrecip Product = "one-hour-precip"

	// ProductStormTotalPrecip shows storm total precipitation accumulation.
	// Single-station RIDGE product (NTP).
	ProductStormTotalPrecip Product = "storm-total-precip"
)

// IsStationProduct returns true if the product requires single-station RIDGE
// data rather than the national composite mosaic.
func IsStationProduct(p Product) bool {
	switch p {
	case ProductBaseReflectivity, ProductStormRelativeVelocity, ProductOneHourPrecip, ProductStormTotalPrecip:
		return true
	default:
		return false
	}
}

// NormalizeProduct canonicalizes product strings and common aliases.
func NormalizeProduct(p string) Product {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "composite", "composite-reflectivity", "cref":
		return ProductCompositeReflectivity
	case "base", "base-reflectivity", "bref":
		return ProductBaseReflectivity
	case "srv", "storm-relative-velocity", "velocity":
		return ProductStormRelativeVelocity
	case "echo-tops", "echotops", "neet":
		return ProductEchoTops
	case "precip-type", "precipitation-type", "ptype":
		return ProductPrecipType
	case "1h-precip", "one-hour-precip", "1h", "n1p":
		return ProductOneHourPrecip
	case "storm-total", "storm-total-precip", "total-precip", "stp", "ntp":
		return ProductStormTotalPrecip
	default:
		return Product(p)
	}
}

// Options controls what radar data to fetch.
type Options struct {
	Product  Product
	RadiusKM float64 // km radius around the location center; default 200
	Raw      bool    // fetch raw transparent radar data without background map or labels
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		Product:  ProductCompositeReflectivity,
		RadiusKM: 200,
	}
}

// BBox represents the geographic bounding box of a radar image.
type BBox struct {
	MinLat, MinLon, MaxLat, MaxLon float64
}

// Frame is a single decoded radar image with associated metadata.
type Frame struct {
	Img       image.Image
	ValidTime time.Time
	Product   Product
	BBox      *BBox // geographic extent; used for city label overlay
}

// Station represents a NEXRAD radar station.
type Station struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

// StationLookup is an optional interface implemented by providers that support
// fetching radar centered on a specific NEXRAD station (e.g. "KIWX").
type StationLookup interface {
	LookupStation(ctx context.Context, stationID string, c *cache.Cache) (*Station, error)
}

// Provider fetches radar imagery for a given location.
type Provider interface {
	Name() string
	// Supports returns true if this provider can serve radar for loc.
	Supports(loc location.Location) bool
	// CurrentFrame fetches the most recent radar frame.
	CurrentFrame(ctx context.Context, loc location.Location, opts Options, c *cache.Cache) (*Frame, error)
	// RecentFrames returns up to n recent frames in chronological order.
	// Fewer frames may be returned if fewer are available.
	RecentFrames(ctx context.Context, loc location.Location, opts Options, n int, c *cache.Cache) ([]*Frame, error)
}

var registry []Provider

// Register adds a radar provider to the global registry.
// Typically called from a provider package's init() function.
func Register(p Provider) {
	registry = append(registry, p)
}

// ForLocation returns the first registered radar provider that supports loc.
func ForLocation(loc location.Location) (Provider, error) {
	for _, p := range registry {
		if p.Supports(loc) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no radar provider available for location (country: %q) — only US locations are currently supported", loc.CountryCode)
}
