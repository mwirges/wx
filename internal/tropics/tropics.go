package tropics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

const (
	defaultTimeout       = 15 * time.Second
	nhcCurrentStormsURL  = "https://www.nhc.noaa.gov/CurrentStorms.json"
	nhcAtlanticTWOURL    = "https://www.nhc.noaa.gov/text/MIATWOAT.shtml"
	nhcEastPacificTWOURL = "https://www.nhc.noaa.gov/text/MIATWOEP.shtml"
	atlanticMapURL       = "https://www.nhc.noaa.gov/xgtwo/two_atl_7d0.png"
	pacificMapURL        = "https://www.nhc.noaa.gov/xgtwo/two_pac_7d0.png"
)

// Client handles querying the NOAA National Hurricane Center feeds.
type Client struct {
	httpClient       *http.Client
	currentStormsURL string
	atlanticTWOURL   string
	pacificTWOURL    string
}

// NewClient returns a new NHC Tropics client.
func NewClient(client *http.Client) *Client {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{
		httpClient:       client,
		currentStormsURL: nhcCurrentStormsURL,
		atlanticTWOURL:   nhcAtlanticTWOURL,
		pacificTWOURL:    nhcEastPacificTWOURL,
	}
}

// WithURLs overrides the default URLs (primarily used for unit testing).
func (c *Client) WithURLs(currentStorms, atlanticTWO, pacificTWO string) *Client {
	if currentStorms != "" {
		c.currentStormsURL = currentStorms
	}
	if atlanticTWO != "" {
		c.atlanticTWOURL = atlanticTWO
	}
	if pacificTWO != "" {
		c.pacificTWOURL = pacificTWO
	}
	return c
}

// nhcRawPayload matches the top-level CurrentStorms.json response from NHC.
type nhcRawPayload struct {
	ActiveStorms []nhcRawStorm `json:"activeStorms"`
}

type nhcRawStorm struct {
	ID               string   `json:"id"`
	BinNumber        string   `json:"binNumber"`
	Name             string   `json:"name"`
	Classification   string   `json:"classification"`
	Intensity        string   `json:"intensity"`
	Pressure         string   `json:"pressure"`
	Latitude         string   `json:"latitude"`
	Longitude        string   `json:"longitude"`
	LatitudeNumeric  *float64 `json:"latitudeNumeric"`
	LongitudeNumeric *float64 `json:"longitudeNumeric"`
	MovementDir      *int     `json:"movementDir"`
	MovementSpeed    *int     `json:"movementSpeed"`
	LastUpdate       string   `json:"lastUpdate"`
	PublicAdvisory   *struct {
		AdvNum   string `json:"advNum"`
		Issuance string `json:"issuance"`
		URL      string `json:"url"`
	} `json:"publicAdvisory"`
	ForecastDiscussion *struct {
		URL string `json:"url"`
	} `json:"forecastDiscussion"`
	ForecastGraphics *struct {
		URL string `json:"url"`
	} `json:"forecastGraphics"`
	TrackCone *struct {
		KMZFile string `json:"kmzFile"`
	} `json:"trackCone"`
}

// FetchTropics retrieves current tropical cyclones and tropical weather outlook disturbances.
func (c *Client) FetchTropics(ctx context.Context, loc *location.Location) (*models.TropicsReport, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.currentStormsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NHC storms request: %w", err)
	}
	req.Header.Set("User-Agent", "wx-cli/1.0 (US Weather CLI; https://github.com/mwirges/wx)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NHC current storms: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NHC current storms returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read NHC response: %w", err)
	}

	var raw nhcRawPayload
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode NHC JSON: %w", err)
	}

	report := &models.TropicsReport{
		GeneratedAt:        time.Now().UTC(),
		AtlanticOutlookURL: atlanticMapURL,
		PacificOutlookURL:  pacificMapURL,
		Storms:             make([]models.TropicalStorm, 0, len(raw.ActiveStorms)),
		Disturbances:       make([]models.TropicalDisturbance, 0),
	}

	if loc != nil && loc.DisplayName != "" {
		report.ReferenceLocation = loc.DisplayName
	}

	// Concurrently process active storms and scrape advisories
	var wg sync.WaitGroup
	stormChan := make(chan models.TropicalStorm, len(raw.ActiveStorms))

	for _, s := range raw.ActiveStorms {
		wg.Add(1)
		go func(rawStorm nhcRawStorm) {
			defer wg.Done()
			storm := c.buildStorm(ctx, rawStorm, loc)
			stormChan <- storm
		}(s)
	}

	wg.Wait()
	close(stormChan)

	for storm := range stormChan {
		report.Storms = append(report.Storms, storm)
	}
	report.TotalActive = len(report.Storms)

	// Ingest Atlantic and East Pacific Tropical Weather Outlooks
	atlanticDisturbances := c.fetchDisturbances(ctx, c.atlanticTWOURL, "Atlantic")
	pacificDisturbances := c.fetchDisturbances(ctx, c.pacificTWOURL, "Eastern Pacific")
	report.Disturbances = append(report.Disturbances, atlanticDisturbances...)
	report.Disturbances = append(report.Disturbances, pacificDisturbances...)

	return report, nil
}

func (c *Client) buildStorm(ctx context.Context, raw nhcRawStorm, loc *location.Location) models.TropicalStorm {
	intensityKt, _ := strconv.Atoi(raw.Intensity)
	pressureMb, _ := strconv.Atoi(raw.Pressure)
	mvmtDir := 0
	if raw.MovementDir != nil {
		mvmtDir = *raw.MovementDir
	}
	mvmtSpeed := 0
	if raw.MovementSpeed != nil {
		mvmtSpeed = *raw.MovementSpeed
	}

	lat := 0.0
	lon := 0.0
	if raw.LatitudeNumeric != nil {
		lat = *raw.LatitudeNumeric
	}
	if raw.LongitudeNumeric != nil {
		lon = *raw.LongitudeNumeric
	}

	category, catLabel := classifyIntensity(raw.Classification, intensityKt)
	className := classificationFullName(raw.Classification)

	storm := models.TropicalStorm{
		ID:                 raw.ID,
		BinNumber:          raw.BinNumber,
		Name:               raw.Name,
		Classification:     raw.Classification,
		ClassificationName: className,
		Category:           category,
		CategoryLabel:      catLabel,
		IntensityKt:        intensityKt,
		WindSpeedMph:       int(math.Round(float64(intensityKt) * 1.15078)),
		WindSpeedKmh:       int(math.Round(float64(intensityKt) * 1.852)),
		PressureMb:         pressureMb,
		PressureInHg:       math.Round(float64(pressureMb)*0.029530*100) / 100,
		Latitude:           lat,
		Longitude:          lon,
		LocationText:       fmt.Sprintf("%s %s", raw.Latitude, raw.Longitude),
		MovementDir:        mvmtDir,
		MovementCompass:    degreesToCompass(mvmtDir),
		MovementSpeedMph:   mvmtSpeed,
		MovementSpeedKmh:   int(math.Round(float64(mvmtSpeed) * 1.60934)),
		LastUpdate:         time.Now().UTC(),
	}

	if raw.LastUpdate != "" {
		if t, err := time.Parse(time.RFC3339, raw.LastUpdate); err == nil {
			storm.LastUpdate = t
		}
	}

	if raw.PublicAdvisory != nil {
		storm.AdvisoryNumber = raw.PublicAdvisory.AdvNum
		storm.AdvisoryTime = raw.PublicAdvisory.Issuance
		storm.PublicAdvisoryURL = raw.PublicAdvisory.URL
	}
	if raw.ForecastDiscussion != nil {
		storm.ForecastDiscussion = raw.ForecastDiscussion.URL
	}
	if raw.ForecastGraphics != nil {
		storm.GraphicsURL = raw.ForecastGraphics.URL
	}
	if raw.TrackCone != nil {
		storm.TrackConeKMZ = raw.TrackCone.KMZFile
	}

	// Calculate distance to reference location if available
	if loc != nil && (loc.Lat != 0 || loc.Lon != 0) && (lat != 0 || lon != 0) {
		dKm := haversineDistance(loc.Lat, loc.Lon, lat, lon)
		dMi := dKm * 0.621371
		storm.DistanceKm = &dKm
		storm.DistanceMiles = &dMi
	}

	// Enhance with public advisory bulletin details if URL available
	if storm.PublicAdvisoryURL != "" {
		c.enrichWithAdvisory(ctx, &storm)
	}

	return storm
}

func (c *Client) enrichWithAdvisory(ctx context.Context, storm *models.TropicalStorm) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, storm.PublicAdvisoryURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "wx-cli/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	text := string(bodyBytes)

	// Extract content inside <pre> tag if present
	preStart := strings.Index(text, "<pre>")
	if preStart != -1 {
		preEnd := strings.Index(text[preStart:], "</pre>")
		if preEnd != -1 {
			text = text[preStart+5 : preStart+preEnd]
		}
	}

	lines := strings.Split(text, "\n")
	var headlines []string
	var proximities []string
	inWatches := false
	var watches []string

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Headlines like ...FAY PERSISTS IN THE CENTRAL ATLANTIC...
		if strings.HasPrefix(line, "...") && strings.HasSuffix(line, "...") {
			trimmed := strings.Trim(line, ". ")
			if !strings.Contains(trimmed, "INFORMATION") && !strings.Contains(trimmed, "WATCHES") {
				headlines = append(headlines, trimmed)
			}
		}

		// Proximity like ABOUT 125 MI...200 KM SW OF CABO SAN LAZARO MEXICO
		if strings.HasPrefix(line, "ABOUT ") {
			proximities = append(proximities, line)
		}

		// Watches and warnings section
		if strings.HasPrefix(line, "WATCHES AND WARNINGS") {
			inWatches = true
			continue
		}
		if inWatches {
			if strings.HasPrefix(line, "DISCUSSION AND OUTLOOK") || strings.HasPrefix(line, "HAZARDS AFFECTING LAND") {
				inWatches = false
				continue
			}
			if strings.Contains(line, "Warning is in effect") || strings.Contains(line, "Watch is in effect") || strings.Contains(line, "warning is in effect") {
				watches = append(watches, line)
			}
		}
	}

	if len(headlines) > 0 {
		storm.Headline = strings.Join(headlines, " // ")
	}
	if len(proximities) > 0 {
		storm.ProximityText = proximities[0]
	}
	if len(watches) > 0 {
		storm.WatchesWarnings = watches
	}
}

var twoBlockRegex = regexp.MustCompile(`(?s)([A-Za-z0-9\s]+)\(([A-Z]{2}\d{2})\):\s*(.+?)\*\s*Formation chance through 48 hours\.\.\.([a-zA-Z]+)\.\.\.(\d+)\s*percent.*?Formation chance through 7 days\.\.\.([a-zA-Z]+)\.\.\.(\d+)\s*percent`)

func (c *Client) fetchDisturbances(ctx context.Context, url, basin string) []models.TropicalDisturbance {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "wx-cli/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	text := string(bodyBytes)

	matches := twoBlockRegex.FindAllStringSubmatch(text, -1)
	var disturbances []models.TropicalDisturbance

	for _, m := range matches {
		if len(m) < 8 {
			continue
		}
		name := strings.TrimSpace(m[1])
		id := strings.TrimSpace(m[2])
		summary := cleanSummary(m[3])
		cat48 := strings.ToLower(strings.TrimSpace(m[4]))
		ch48, _ := strconv.Atoi(m[5])
		cat7d := strings.ToLower(strings.TrimSpace(m[6]))
		ch7d, _ := strconv.Atoi(m[7])

		disturbances = append(disturbances, models.TropicalDisturbance{
			ID:          id,
			Basin:       basin,
			Name:        name,
			Summary:     summary,
			Chance48h:   ch48,
			Category48h: cat48,
			Chance7d:    ch7d,
			Category7d:  cat7d,
		})
	}

	return disturbances
}

func cleanSummary(text string) string {
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t != "" && !strings.HasPrefix(t, "*") {
			cleaned = append(cleaned, t)
		}
	}
	return strings.Join(cleaned, " ")
}

func classifyIntensity(class string, intensityKt int) (int, string) {
	c := strings.ToUpper(class)
	switch {
	case intensityKt >= 137:
		return 5, "Category 5 (Major Hurricane)"
	case intensityKt >= 113:
		return 4, "Category 4 (Major Hurricane)"
	case intensityKt >= 96:
		return 3, "Category 3 (Major Hurricane)"
	case intensityKt >= 83:
		return 2, "Category 2 (Hurricane)"
	case intensityKt >= 64:
		return 1, "Category 1 (Hurricane)"
	case c == "PTC":
		return 0, "Potential Tropical Cyclone"
	case c == "SS":
		return 0, "Subtropical Storm"
	case c == "SD":
		return 0, "Subtropical Depression"
	case c == "EX" || c == "POST":
		return 0, "Post-Tropical Cyclone"
	case c == "LO":
		return 0, "Remnant Low"
	case c == "TS" || (intensityKt >= 34 && intensityKt < 64):
		return 0, "Tropical Storm"
	default:
		return 0, "Tropical Depression"
	}
}

func classificationFullName(class string) string {
	switch strings.ToUpper(class) {
	case "HU":
		return "Hurricane"
	case "TS":
		return "Tropical Storm"
	case "TD":
		return "Tropical Depression"
	case "PTC":
		return "Potential Tropical Cyclone"
	case "SS":
		return "Subtropical Storm"
	case "SD":
		return "Subtropical Depression"
	case "POST", "EX":
		return "Post-Tropical / Extratropical"
	case "LO":
		return "Remnant Low"
	case "DB":
		return "Disturbance"
	default:
		return class
	}
}

func degreesToCompass(deg int) string {
	deg = (deg%360 + 360) % 360
	compass := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int(math.Floor((float64(deg) + 11.25) / 22.5))
	return compass[idx%16]
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return r * c
}
