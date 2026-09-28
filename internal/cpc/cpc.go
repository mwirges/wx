package cpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

const (
	cpc610TempURL   = "https://mapservices.weather.noaa.gov/vector/rest/services/outlooks/cpc_6_10_day_outlk/MapServer/0/query"
	cpc610PrecipURL = "https://mapservices.weather.noaa.gov/vector/rest/services/outlooks/cpc_6_10_day_outlk/MapServer/1/query"
	cpc814TempURL   = "https://mapservices.weather.noaa.gov/vector/rest/services/outlooks/cpc_8_14_day_outlk/MapServer/0/query"
	cpc814PrecipURL = "https://mapservices.weather.noaa.gov/vector/rest/services/outlooks/cpc_8_14_day_outlk/MapServer/1/query"
	cpcDroughtURL   = "https://mapservices.weather.noaa.gov/vector/rest/services/outlooks/cpc_drought_outlk/MapServer/1/query"

	cacheTTL = 2 * time.Hour
)

// ESRIFeature holds raw attributes from an ESRI MapServer query.
type esriResponse struct {
	Features []struct {
		Attributes map[string]interface{} `json:"attributes"`
	} `json:"features"`
}

// Client queries the NOAA Climate Prediction Center map services.
type Client struct {
	httpClient *http.Client
	c610Temp   string
	c610Precip string
	c814Temp   string
	c814Precip string
	cDrought   string
}

// NewClient returns a new CPC client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		c610Temp:   cpc610TempURL,
		c610Precip: cpc610PrecipURL,
		c814Temp:   cpc814TempURL,
		c814Precip: cpc814PrecipURL,
		cDrought:   cpcDroughtURL,
	}
}

// FetchOutlooks retrieves the 6–10 day, 8–14 day, and drought outlooks for a location.
func (cl *Client) FetchOutlooks(ctx context.Context, loc location.Location, c *cache.Cache) (*models.CPCPayload, error) {
	cacheKey := fmt.Sprintf("cpc:outlooks:v1:%.4f,%.4f", loc.Lat, loc.Lon)
	if c != nil {
		var cached models.CPCPayload
		if c.Get(cacheKey, &cached) {
			return &cached, nil
		}
	}

	type fetchResult struct {
		name string
		cat  string
		prob float64
		s    time.Time
		e    time.Time
		err  error
	}

	resCh := make(chan fetchResult, 5)
	var wg sync.WaitGroup

	queries := []struct {
		name string
		url  string
	}{
		{"610_temp", cl.c610Temp},
		{"610_precip", cl.c610Precip},
		{"814_temp", cl.c814Temp},
		{"814_precip", cl.c814Precip},
	}

	for _, q := range queries {
		wg.Add(1)
		go func(name, queryURL string) {
			defer wg.Done()
			cat, prob, s, e, err := cl.queryESRIMapServer(ctx, queryURL, loc.Lat, loc.Lon)
			resCh <- fetchResult{name: name, cat: cat, prob: prob, s: s, e: e, err: err}
		}(q.name, q.url)
	}

	// Drought query
	var droughtResult *models.CPCDroughtOutlook
	var droughtErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		droughtResult, droughtErr = cl.queryDrought(ctx, cl.cDrought, loc.Lat, loc.Lon)
	}()

	wg.Wait()
	close(resCh)

	results := make(map[string]fetchResult)
	for r := range resCh {
		if r.err == nil {
			results[r.name] = r
		}
	}

	r610T := results["610_temp"]
	r610P := results["610_precip"]
	r814T := results["814_temp"]
	r814P := results["814_precip"]

	// Build 6-10 Day item
	item610 := models.CPCOutlookItem{
		Horizon:           "6-10 Day",
		StartDate:         r610T.s,
		EndDate:           r610T.e,
		TempCategory:      normalizeCategory(r610T.cat),
		TempProbability:   r610T.prob,
		PrecipCategory:    normalizeCategory(r610P.cat),
		PrecipProbability: r610P.prob,
	}
	if item610.StartDate.IsZero() {
		item610.StartDate = r610P.s
		item610.EndDate = r610P.e
	}

	// Build 8-14 Day item
	item814 := models.CPCOutlookItem{
		Horizon:           "8-14 Day",
		StartDate:         r814T.s,
		EndDate:           r814T.e,
		TempCategory:      normalizeCategory(r814T.cat),
		TempProbability:   r814T.prob,
		PrecipCategory:    normalizeCategory(r814P.cat),
		PrecipProbability: r814P.prob,
	}
	if item814.StartDate.IsZero() {
		item814.StartDate = r814P.s
		item814.EndDate = r814P.e
	}

	outlooks := []models.CPCOutlookItem{item610, item814}
	patternShift := AnalyzePatternShift(outlooks)

	payload := &models.CPCPayload{
		Location:     loc.DisplayName,
		Coordinates:  [2]float64{loc.Lat, loc.Lon},
		FetchedAt:    time.Now().UTC(),
		Outlooks:     outlooks,
		Drought:      droughtResult,
		PatternShift: patternShift,
	}
	_ = droughtErr

	if c != nil {
		_ = c.Set(cacheKey, payload, cacheTTL)
	}

	return payload, nil
}

func (cl *Client) queryESRIMapServer(ctx context.Context, endpoint string, lat, lon float64) (string, float64, time.Time, time.Time, error) {
	params := url.Values{}
	// ESRI point geometry uses longitude (X), latitude (Y).
	params.Set("geometry", fmt.Sprintf("%.4f,%.4f", lon, lat))
	params.Set("geometryType", "esriGeometryPoint")
	params.Set("inSR", "4326")
	params.Set("spatialRel", "esriSpatialRelIntersects")
	params.Set("outFields", "*")
	params.Set("returnGeometry", "false")
	params.Set("f", "json")

	fullURL := endpoint + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return "", 0, time.Time{}, time.Time{}, err
	}
	req.Header.Set("User-Agent", "wx-cli/1.0 (github.com/mwirges/wx)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return "", 0, time.Time{}, time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, time.Time{}, time.Time{}, fmt.Errorf("CPC HTTP %d", resp.StatusCode)
	}

	var data esriResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", 0, time.Time{}, time.Time{}, err
	}

	if len(data.Features) == 0 {
		return "EC", 33.3, time.Time{}, time.Time{}, nil
	}

	attrs := data.Features[0].Attributes
	cat, _ := attrs["cat"].(string)
	prob := 0.0
	switch v := attrs["prob"].(type) {
	case float64:
		prob = v
	case int:
		prob = float64(v)
	}

	var startT, endT time.Time
	if sVal, ok := attrs["start_date"].(float64); ok && sVal > 0 {
		startT = time.UnixMilli(int64(sVal)).UTC()
	}
	if eVal, ok := attrs["end_date"].(float64); ok && eVal > 0 {
		endT = time.UnixMilli(int64(eVal)).UTC()
	}

	return cat, prob, startT, endT, nil
}

func (cl *Client) queryDrought(ctx context.Context, endpoint string, lat, lon float64) (*models.CPCDroughtOutlook, error) {
	params := url.Values{}
	params.Set("geometry", fmt.Sprintf("%.4f,%.4f", lon, lat))
	params.Set("geometryType", "esriGeometryPoint")
	params.Set("inSR", "4326")
	params.Set("spatialRel", "esriSpatialRelIntersects")
	params.Set("outFields", "*")
	params.Set("returnGeometry", "false")
	params.Set("f", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "wx-cli/1.0 (github.com/mwirges/wx)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("drought HTTP %d", resp.StatusCode)
	}

	var data esriResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if len(data.Features) == 0 {
		return nil, nil
	}

	attrs := data.Features[0].Attributes
	outlook, _ := attrs["outlook"].(string)
	target, _ := attrs["target"].(string)

	cleanStatus := strings.ReplaceAll(outlook, "_", " ")
	if cleanStatus == "" {
		cleanStatus = "None"
	}

	return &models.CPCDroughtOutlook{
		Status: cleanStatus,
		Target: target,
	}, nil
}

func normalizeCategory(cat string) string {
	switch strings.ToLower(strings.TrimSpace(cat)) {
	case "above", "a":
		return "Above"
	case "below", "b":
		return "Below"
	case "normal", "n":
		return "Normal"
	default:
		return "Equal Chances"
	}
}

// AnalyzePatternShift examines the 6–10 and 8–14 day outlooks to detect atmospheric regime changes.
func AnalyzePatternShift(outlooks []models.CPCOutlookItem) models.CPCPatternShift {
	if len(outlooks) < 2 {
		return models.CPCPatternShift{HasShift: false, Summary: "Insufficient long-range data to project pattern shift."}
	}

	o1 := outlooks[0] // 6–10 Day
	o2 := outlooks[1] // 8–14 Day

	tempShift := "steady"
	precipShift := "steady"
	var notes []string

	// Temperature trend analysis
	switch {
	case o1.TempCategory == "Above" && o2.TempCategory == "Below":
		tempShift = "cooling"
		notes = append(notes, "sharp cool-down from above-normal warmth to below-normal chill")
	case o1.TempCategory == "Above" && (o2.TempCategory == "Normal" || o2.TempCategory == "Equal Chances"):
		tempShift = "cooling"
		notes = append(notes, "temperatures cooling back to seasonal averages")
	case (o1.TempCategory == "Normal" || o1.TempCategory == "Equal Chances") && o2.TempCategory == "Below":
		tempShift = "cooling"
		notes = append(notes, "cooler-than-normal Canadian trough settling in")
	case o1.TempCategory == "Below" && o2.TempCategory == "Above":
		tempShift = "warming"
		notes = append(notes, "strong warm-up flipping from below-normal cool to above-average warmth")
	case o1.TempCategory == "Below" && (o2.TempCategory == "Normal" || o2.TempCategory == "Equal Chances"):
		tempShift = "warming"
		notes = append(notes, "temperatures moderating back towards seasonal norms")
	case (o1.TempCategory == "Normal" || o1.TempCategory == "Equal Chances") && o2.TempCategory == "Above":
		tempShift = "warming"
		notes = append(notes, "warming trend pushing temperatures above normal")
	case o1.TempCategory == "Above" && o2.TempCategory == "Above":
		notes = append(notes, "persistent warmth continuing through Day 14")
	case o1.TempCategory == "Below" && o2.TempCategory == "Below":
		notes = append(notes, "persistent below-average chill locked in through Day 14")
	default:
		notes = append(notes, "seasonable temperatures expected")
	}

	// Precipitation trend analysis
	switch {
	case o1.PrecipCategory == "Above" && o2.PrecipCategory == "Below":
		precipShift = "drying"
		notes = append(notes, "wet start drying out substantially in week 2")
	case o1.PrecipCategory == "Above" && (o2.PrecipCategory == "Normal" || o2.PrecipCategory == "Equal Chances"):
		precipShift = "drying"
		notes = append(notes, "active storm track tapering off toward normal")
	case o1.PrecipCategory == "Below" && o2.PrecipCategory == "Above":
		precipShift = "wetter"
		notes = append(notes, "dry stretch giving way to an active, wetter storm pattern")
	case (o1.PrecipCategory == "Normal" || o1.PrecipCategory == "Equal Chances") && o2.PrecipCategory == "Above":
		precipShift = "wetter"
		notes = append(notes, "increasing precipitation odds into mid-month")
	case (o1.PrecipCategory == "Normal" || o1.PrecipCategory == "Equal Chances") && o2.PrecipCategory == "Below":
		precipShift = "drying"
		notes = append(notes, "drier-than-average high pressure building")
	case o1.PrecipCategory == "Above" && o2.PrecipCategory == "Above":
		notes = append(notes, "prolonged wet pattern with multiple storm chances")
	case o1.PrecipCategory == "Below" && o2.PrecipCategory == "Below":
		notes = append(notes, "extended dry stretch favored")
	}

	hasShift := tempShift != "steady" || precipShift != "steady"
	confidence := "Moderate"
	if o1.TempProbability >= 50 || o2.TempProbability >= 50 || o1.PrecipProbability >= 50 || o2.PrecipProbability >= 50 {
		confidence = "High"
	}

	summary := strings.Join(notes, "; ")
	if len(summary) > 0 {
		summary = strings.ToUpper(summary[:1]) + summary[1:] + "."
	}

	return models.CPCPatternShift{
		HasShift:    hasShift,
		Summary:     summary,
		TempShift:   tempShift,
		PrecipShift: precipShift,
		Confidence:  confidence,
	}
}
