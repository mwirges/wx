package location

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ipgeoDo dials ipinfo.io. Production uses http.DefaultClient.
var ipgeoDo = http.DefaultClient.Do

type ipInfoResponse struct {
	IP      string `json:"ip"`
	City    string `json:"city"`
	Region  string `json:"region"`  // full state name, e.g. "Missouri"
	Country string `json:"country"` // ISO code, e.g. "US"
	Postal  string `json:"postal"`
	Loc     string `json:"loc"` // "lat,lon"
}

const ipGeoTTL = 1 * time.Hour

func resolveByIP(ctx context.Context, c Cacher) (Location, error) {
	const cacheKey = "ipgeo:v1"

	var loc Location
	if c.Get(cacheKey, &loc) {
		return loc, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ipinfo.io/json", nil)
	if err != nil {
		return Location{}, fmt.Errorf("location: ipgeo: build request: %w", err)
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")
	req.Header.Set("Accept", "application/json")

	body, err := readRetry(ctx, ipgeoDo, req)
	if err != nil {
		return Location{}, fmt.Errorf("location: ipgeo: %w", err)
	}

	var info ipInfoResponse
	if err := json.Unmarshal(body, &info); err != nil {
		return Location{}, fmt.Errorf("location: ipgeo: decode: %w", err)
	}

	parts := strings.SplitN(info.Loc, ",", 2)
	if len(parts) != 2 {
		return Location{}, fmt.Errorf("location: ipgeo: unexpected loc format %q", info.Loc)
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return Location{}, fmt.Errorf("location: ipgeo: parse lat: %w", err)
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return Location{}, fmt.Errorf("location: ipgeo: parse lon: %w", err)
	}

	display := info.City
	if info.Region != "" {
		display += ", " + info.Region
	}

	loc = Location{
		Lat:         lat,
		Lon:         lon,
		DisplayName: display,
		CountryCode: strings.ToUpper(info.Country),
		City:        info.City,
		State:       info.Region,
	}

	_ = c.Set(cacheKey, loc, ipGeoTTL)
	return loc, nil
}
