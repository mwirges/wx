package location

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const geocodeTTL = 24 * time.Hour

type nominatimResult struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
	Address     struct {
		City        string `json:"city"`
		Town        string `json:"town"`
		Village     string `json:"village"`
		State       string `json:"state"`
		PostCode    string `json:"postcode"`
		CountryCode string `json:"country_code"` // lowercase "us"
	} `json:"address"`
}

func resolveByZip(ctx context.Context, zip string, c Cacher) (Location, error) {
	cacheKey := "geocode:zip:" + zip

	var loc Location
	if c.Get(cacheKey, &loc) {
		return loc, nil
	}

	params := url.Values{
		"format":       {"json"},
		"postalcode":   {zip},
		"countrycodes": {"us"},
		"addressdetails": {"1"},
		"limit":        {"1"},
	}

	results, err := nominatimSearch(ctx, params)
	if err != nil {
		return Location{}, fmt.Errorf("location: geocode zip %q: %w", zip, err)
	}
	if len(results) == 0 {
		return Location{}, fmt.Errorf("location: zip code %q not found", zip)
	}

	loc = resultToLocation(results[0])
	if loc.DisplayName == "" {
		loc.DisplayName = zip
	}

	_ = c.Set(cacheKey, loc, geocodeTTL)
	return loc, nil
}

func resolveByCityState(ctx context.Context, query string, c Cacher) (Location, error) {
	normalized := strings.ToLower(strings.TrimSpace(query))
	cacheKey := "geocode:city:" + normalized

	var loc Location
	if c.Get(cacheKey, &loc) {
		return loc, nil
	}

	params := url.Values{
		"format":         {"json"},
		"q":              {query},
		"countrycodes":   {"us"},
		"addressdetails": {"1"},
		"limit":          {"1"},
	}

	results, err := nominatimSearch(ctx, params)
	if err != nil {
		return Location{}, fmt.Errorf("location: geocode %q: %w", query, err)
	}
	if len(results) == 0 {
		return Location{}, fmt.Errorf("location: %q not found", query)
	}

	loc = resultToLocation(results[0])
	if loc.DisplayName == "" {
		loc.DisplayName = query
	}

	_ = c.Set(cacheKey, loc, geocodeTTL)
	return loc, nil
}

func nominatimSearch(ctx context.Context, params url.Values) ([]nominatimResult, error) {
	reqURL := "https://nominatim.openstreetmap.org/search?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var results []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return results, nil
}

func resultToLocation(r nominatimResult) Location {
	lat, _ := strconv.ParseFloat(r.Lat, 64)
	lon, _ := strconv.ParseFloat(r.Lon, 64)

	city := r.Address.City
	if city == "" {
		city = r.Address.Town
	}
	if city == "" {
		city = r.Address.Village
	}

	display := city
	if r.Address.State != "" {
		if display != "" {
			display += ", " + r.Address.State
		} else {
			display = r.Address.State
		}
	}

	return Location{
		Lat:         lat,
		Lon:         lon,
		DisplayName: display,
		CountryCode: strings.ToUpper(r.Address.CountryCode),
		City:        city,
		State:       r.Address.State,
	}
}
