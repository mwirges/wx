package chase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/radar"
	"github.com/mwirges/wx/internal/spc"
)

const (
	defaultBaseURL = "https://api.weather.gov"
	cacheTTL       = 2 * time.Minute
	clusterDistKm  = 450.0
)

var (
	reHazard  = regexp.MustCompile(`(?i)(?:HAZARD\.\.\.)\s*([^\n\r]+)`)
	reWind    = regexp.MustCompile(`(?i)(?:WIND(?:\s+GUSTS?)?\.\.\.)\s*([^\n\r]+)`)
	reHail    = regexp.MustCompile(`(?i)(?:HAIL(?:\s+THREAT)?\.\.\.)\s*([^\n\r]+)`)
	reTornado = regexp.MustCompile(`(?i)(?:TORNADO\.\.\.)\s*([^\n\r]+)`)
)

type nwsAlertFeature struct {
	ID       string `json:"id"`
	Geometry struct {
		Type        string `json:"type"`
		Coordinates any    `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		Event       string `json:"event"`
		Headline    string `json:"headline"`
		AreaDesc    string `json:"areaDesc"`
		Description string `json:"description"`
		Severity    string `json:"severity"`
		Urgency     string `json:"urgency"`
		Certainty   string `json:"certainty"`
		SenderName  string `json:"senderName"`
		Sender      string `json:"sender"`
		Geocode     struct {
			UGC []string `json:"UGC"`
		} `json:"geocode"`
		Effective string `json:"effective"`
		Expires   string `json:"expires"`
	} `json:"properties"`
}

type nwsAlertsResponse struct {
	Features []nwsAlertFeature `json:"features"`
}

// Client fetches and analyzes active alert clusters.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient returns a new Client.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

var defaultClient = NewClient("")

// FetchClusters retrieves all active severe weather alerts and groups them into regional storm systems.
func FetchClusters(ctx context.Context, c *cache.Cache) (*models.ChasePayload, error) {
	return defaultClient.FetchClusters(ctx, c)
}

// FetchClusters retrieves all active severe weather alerts and groups them into regional storm systems.
func (cl *Client) FetchClusters(ctx context.Context, c *cache.Cache) (*models.ChasePayload, error) {
	cacheKey := "chase:severe_clusters"
	var cached models.ChasePayload
	if c != nil && c.Get(cacheKey, &cached) {
		return &cached, nil
	}

	reqURL := fmt.Sprintf("%s/alerts/active?status=actual&message_type=alert&severity=Extreme,Severe", cl.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("chase: build request: %w", err)
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")
	req.Header.Set("Accept", "application/geo+json")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chase: fetch alerts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("chase: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var data nwsAlertsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("chase: decode alerts: %w", err)
	}

	cells := make([]models.AlertCell, 0, len(data.Features))
	for _, feat := range data.Features {
		p := feat.Properties
		lat, lon, hasPoly := extractCentroid(feat)
		if lat == 0 && lon == 0 {
			// Fallback to sender / state approximation
			lat, lon = fallbackCoords(p.SenderName, p.Geocode.UGC)
		}

		states := extractStates(p.Geocode.UGC)
		eff, _ := time.Parse(time.RFC3339, p.Effective)
		exp, _ := time.Parse(time.RFC3339, p.Expires)

		radarSite := ""
		if nearest := radar.NearestStation(lat, lon); nearest.ID != "" {
			radarSite = nearest.ID
		}

		cell := models.AlertCell{
			ID:          feat.ID,
			Event:       p.Event,
			Headline:    p.Headline,
			AreaDesc:    p.AreaDesc,
			Description: p.Description,
			Severity:    p.Severity,
			Urgency:     p.Urgency,
			Certainty:   p.Certainty,
			SenderName:  p.SenderName,
			States:      states,
			Latitude:    lat,
			Longitude:   lon,
			HasPolygon:  hasPoly,
			HazardText:  extractHazard(p.Description, p.Headline),
			RadarSite:   radarSite,
			Effective:   eff,
			Expires:     exp,
		}
		cells = append(cells, cell)
	}

	clusters := clusterAlerts(cells)

	// Fetch SPC convective context in parallel
	var (
		spcMCDs    []models.MesoscaleDiscussion
		spcWatches []models.SPCWatch
		spcMaxRisk models.SPCRiskCategory
		spcWG      sync.WaitGroup
	)
	spcWG.Add(3)
	go func() {
		defer spcWG.Done()
		if res, err := spc.FetchActiveMCDs(ctx, c); err == nil {
			spcMCDs = res
		}
	}()
	go func() {
		defer spcWG.Done()
		if res, err := spc.FetchActiveWatches(ctx, c); err == nil {
			spcWatches = res
		}
	}()
	go func() {
		defer spcWG.Done()
		if res, err := spc.FetchNationalMaxRisk(ctx, c); err == nil {
			spcMaxRisk = res
		}
	}()
	spcWG.Wait()

	correlateClustersWithSPC(clusters, spcMCDs, spcWatches, spcMaxRisk)

	payload := &models.ChasePayload{
		GeneratedAt:   time.Now().UTC(),
		TotalAlerts:   len(cells),
		TotalClusters: len(clusters),
		SPC: &models.SPCPayload{
			FetchedAt:       time.Now().UTC(),
			ActiveMCDs:      spcMCDs,
			ActiveWatches:   spcWatches,
			MaxNationalRisk: spcMaxRisk,
		},
		Clusters: clusters,
	}

	if c != nil {
		_ = c.Set(cacheKey, payload, cacheTTL)
	}

	return payload, nil
}

func correlateClustersWithSPC(clusters []models.StormCluster, mcds []models.MesoscaleDiscussion, watches []models.SPCWatch, maxRisk models.SPCRiskCategory) {
	for i := range clusters {
		cl := &clusters[i]
		if maxRisk.DN > 0 {
			cl.SPCRisk = maxRisk.Code
		}

		// Check active watches first
		for _, w := range watches {
			if matchesAnyState(cl.States, w.States) {
				cl.MCDWatch = fmt.Sprintf("Watch #%d (%s)", w.WatchNumber, w.Type)
				break
			}
		}

		// If no watch, check active MCDs
		if cl.MCDWatch == "" && len(mcds) > 0 {
			for _, m := range mcds {
				if m.Lat != 0 && m.Lon != 0 {
					d := haversineKm(cl.CenterLat, cl.CenterLon, m.Lat, m.Lon)
					if d <= 450.0 {
						probText := ""
						if m.WatchProbability != "" {
							probText = fmt.Sprintf(", %s watch prob", m.WatchProbability)
						}
						cl.MCDWatch = fmt.Sprintf("%s (%s%s)", m.Name, m.Concerning, probText)
						break
					}
				}
			}
		}
	}
}

func matchesAnyState(s1, s2 []string) bool {
	set := make(map[string]bool)
	for _, s := range s1 {
		set[s] = true
	}
	for _, s := range s2 {
		if set[s] {
			return true
		}
	}
	return false
}

func extractCentroid(feat nwsAlertFeature) (lat float64, lon float64, hasPoly bool) {
	if feat.Geometry.Coordinates == nil {
		return 0, 0, false
	}

	// Coordinates can be Polygon ([][][]float64) or MultiPolygon ([][][][]float64)
	rawBytes, err := json.Marshal(feat.Geometry.Coordinates)
	if err != nil {
		return 0, 0, false
	}

	if feat.Geometry.Type == "Polygon" {
		var rings [][][]float64
		if err := json.Unmarshal(rawBytes, &rings); err == nil && len(rings) > 0 && len(rings[0]) > 0 {
			var sumLat, sumLon float64
			for _, pt := range rings[0] {
				if len(pt) >= 2 {
					sumLon += pt[0]
					sumLat += pt[1]
				}
			}
			n := float64(len(rings[0]))
			return sumLat / n, sumLon / n, true
		}
	} else if feat.Geometry.Type == "MultiPolygon" {
		var polys [][][][]float64
		if err := json.Unmarshal(rawBytes, &polys); err == nil && len(polys) > 0 {
			var sumLat, sumLon float64
			var count float64
			for _, poly := range polys {
				if len(poly) > 0 {
					for _, pt := range poly[0] {
						if len(pt) >= 2 {
							sumLon += pt[0]
							sumLat += pt[1]
							count++
						}
					}
				}
			}
			if count > 0 {
				return sumLat / count, sumLon / count, true
			}
		}
	}

	return 0, 0, false
}

func extractStates(ugcs []string) []string {
	seen := make(map[string]bool)
	var states []string
	for _, u := range ugcs {
		if len(u) >= 2 {
			st := strings.ToUpper(u[:2])
			if !seen[st] {
				seen[st] = true
				states = append(states, st)
			}
		}
	}
	sort.Strings(states)
	return states
}

func extractHazard(desc, headline string) string {
	var parts []string
	if m := reHazard.FindStringSubmatch(desc); len(m) > 1 {
		parts = append(parts, strings.TrimSpace(m[1]))
	}
	if m := reTornado.FindStringSubmatch(desc); len(m) > 1 && !strings.Contains(strings.ToLower(m[1]), "none") {
		parts = append(parts, "Tornado: "+strings.TrimSpace(m[1]))
	}
	if m := reWind.FindStringSubmatch(desc); len(m) > 1 {
		parts = append(parts, "Wind: "+strings.TrimSpace(m[1]))
	}
	if m := reHail.FindStringSubmatch(desc); len(m) > 1 {
		parts = append(parts, "Hail: "+strings.TrimSpace(m[1]))
	}
	if len(parts) > 0 {
		return strings.Join(parts, " • ")
	}
	if headline != "" {
		return headline
	}
	return ""
}

func fallbackCoords(sender string, ugcs []string) (float64, float64) {
	if coords, ok := wfoCoordinates[sender]; ok {
		return coords[0], coords[1]
	}
	if len(ugcs) > 0 && len(ugcs[0]) >= 2 {
		st := strings.ToUpper(ugcs[0][:2])
		if coords, ok := stateCenters[st]; ok {
			return coords[0], coords[1]
		}
	}
	return 38.0, -97.0 // Geographic center of CONUS
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180.0))*math.Cos(lat2*(math.Pi/180.0))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func scoreAlert(ev string) int {
	evLower := strings.ToLower(ev)
	switch {
	case strings.Contains(evLower, "tornado warning"):
		return 100
	case strings.Contains(evLower, "severe thunderstorm warning"):
		return 40
	case strings.Contains(evLower, "flash flood warning"):
		return 40
	case strings.Contains(evLower, "blizzard warning") || strings.Contains(evLower, "ice storm warning"):
		return 35
	case strings.Contains(evLower, "coastal flood warning") || strings.Contains(evLower, "high wind warning"):
		return 20
	case strings.Contains(evLower, "flood warning"):
		return 15
	case strings.Contains(evLower, "watch"):
		return 10
	default:
		return 5
	}
}

func clusterAlerts(cells []models.AlertCell) []models.StormCluster {
	n := len(cells)
	if n == 0 {
		return nil
	}

	adj := make([][]int, n)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			d := haversineKm(cells[i].Latitude, cells[i].Longitude, cells[j].Latitude, cells[j].Longitude)
			if d <= clusterDistKm {
				adj[i] = append(adj[i], j)
				adj[j] = append(adj[j], i)
			}
		}
	}

	visited := make([]bool, n)
	var rawClusters [][]models.AlertCell

	for i := 0; i < n; i++ {
		if !visited[i] {
			var comp []models.AlertCell
			queue := []int{i}
			visited[i] = true

			for len(queue) > 0 {
				curr := queue[0]
				queue = queue[1:]
				comp = append(comp, cells[curr])

				for _, neighbor := range adj[curr] {
					if !visited[neighbor] {
						visited[neighbor] = true
						queue = append(queue, neighbor)
					}
				}
			}
			rawClusters = append(rawClusters, comp)
		}
	}

	var stormClusters []models.StormCluster
	for idx, group := range rawClusters {
		// Sort cells within cluster by severity score descending
		sort.Slice(group, func(i, j int) bool {
			return scoreAlert(group[i].Event) > scoreAlert(group[j].Event)
		})

		var sumLat, sumLon float64
		totalScore := 0
		hazardsCount := make(map[string]int)
		stateSet := make(map[string]bool)

		for _, c := range group {
			sumLat += c.Latitude
			sumLon += c.Longitude
			totalScore += scoreAlert(c.Event)
			hazardsCount[c.Event]++
			for _, s := range c.States {
				stateSet[s] = true
			}
		}

		centerLat := sumLat / float64(len(group))
		centerLon := sumLon / float64(len(group))

		nearestRadar := ""
		if st := radar.NearestStation(centerLat, centerLon); st.ID != "" {
			nearestRadar = st.ID
		}

		var states []string
		for s := range stateSet {
			states = append(states, s)
		}
		sort.Strings(states)

		primaryHazard := determinePrimaryHazard(hazardsCount)
		name := nameCluster(states, primaryHazard, len(group))

		stormClusters = append(stormClusters, models.StormCluster{
			ID:            idx + 1,
			Name:          name,
			States:        states,
			CenterLat:     centerLat,
			CenterLon:     centerLon,
			NearestRadar:  nearestRadar,
			TotalAlerts:   len(group),
			Score:         totalScore,
			HazardsCount:  hazardsCount,
			PrimaryHazard: primaryHazard,
			Cells:         group,
		})
	}

	// Sort clusters by score descending, then by alert count descending
	sort.Slice(stormClusters, func(i, j int) bool {
		if stormClusters[i].Score != stormClusters[j].Score {
			return stormClusters[i].Score > stormClusters[j].Score
		}
		return stormClusters[i].TotalAlerts > stormClusters[j].TotalAlerts
	})

	// Re-index IDs 1..N based on rank
	for i := range stormClusters {
		stormClusters[i].ID = i + 1
	}

	return stormClusters
}

func determinePrimaryHazard(counts map[string]int) string {
	priority := []string{
		"Tornado Warning",
		"Severe Thunderstorm Warning",
		"Flash Flood Warning",
		"Blizzard Warning",
		"Ice Storm Warning",
		"Coastal Flood Warning",
		"High Wind Warning",
		"Flood Warning",
		"Tornado Watch",
		"Severe Thunderstorm Watch",
		"Flood Watch",
		"High Wind Watch",
	}

	for _, p := range priority {
		if counts[p] > 0 {
			return p
		}
	}
	for k := range counts {
		return k
	}
	return "Weather Alert"
}

func nameCluster(states []string, primaryHazard string, count int) string {
	region := determineRegion(states)

	hazardDesc := "Weather System"
	switch {
	case strings.Contains(primaryHazard, "Tornado"):
		if count > 1 {
			hazardDesc = "Tornadic Supercells"
		} else {
			hazardDesc = "Tornadic Cell"
		}
	case strings.Contains(primaryHazard, "Severe Thunderstorm"):
		if count > 2 {
			hazardDesc = "Severe Thunderstorm Cluster"
		} else {
			hazardDesc = "Severe Thunderstorm Cell"
		}
	case strings.Contains(primaryHazard, "Coastal Flood"):
		hazardDesc = "Coastal Storm (Nor'easter)"
	case strings.Contains(primaryHazard, "Flash Flood"):
		hazardDesc = "Flash Flood Event"
	case strings.Contains(primaryHazard, "Flood"):
		hazardDesc = "Flood & Monsoon System"
	case strings.Contains(primaryHazard, "Blizzard") || strings.Contains(primaryHazard, "Winter"):
		hazardDesc = "Winter Storm System"
	case strings.Contains(primaryHazard, "High Wind"):
		hazardDesc = "High Wind Event"
	}

	if region != "" {
		return fmt.Sprintf("%s %s", region, hazardDesc)
	}
	if len(states) > 0 {
		return fmt.Sprintf("%s %s", strings.Join(states, "/"), hazardDesc)
	}
	return hazardDesc
}

func determineRegion(states []string) string {
	stSet := make(map[string]bool)
	for _, s := range states {
		stSet[s] = true
	}

	hasAny := func(keys ...string) bool {
		for _, k := range keys {
			if stSet[k] {
				return true
			}
		}
		return false
	}

	if hasAny("ME", "NH", "VT", "MA", "RI", "CT") && hasAny("NJ", "DE", "MD", "VA") {
		return "Mid-Atlantic & Northeast"
	}
	if hasAny("NJ", "DE", "MD", "VA", "PA", "DC") {
		return "Mid-Atlantic"
	}
	if hasAny("ME", "NH", "VT", "MA", "RI", "CT") {
		return "New England"
	}
	if hasAny("AZ", "NM", "UT", "CO", "NV") {
		return "Southwest & Four Corners"
	}
	if hasAny("OK", "TX") {
		return "Southern Plains"
	}
	if hasAny("NE", "KS", "SD", "ND", "IA") {
		return "Central Plains"
	}
	if hasAny("IL", "IN", "OH", "MI", "WI") {
		return "Midwest & Great Lakes"
	}
	if hasAny("FL", "GA", "SC", "NC", "AL", "MS", "LA") {
		return "Southeast & Gulf Coast"
	}
	if hasAny("WA", "OR", "CA") {
		return "West Coast"
	}
	if hasAny("MT", "WY", "ID") {
		return "Northern Rockies"
	}
	if hasAny("HI") {
		return "Hawaii"
	}
	if hasAny("AK") {
		return "Alaska"
	}
	if len(states) > 0 {
		return strings.Join(states, "/")
	}
	return "Regional"
}

// Common NWS WFO Coordinates
var wfoCoordinates = map[string][2]float64{
	"NWS Mount Holly NJ":             {39.95, -74.82},
	"NWS Wakefield VA":               {36.98, -76.99},
	"NWS Boston/Norton MA":           {41.95, -71.18},
	"NWS Tucson AZ":                  {32.23, -110.95},
	"NWS Phoenix AZ":                 {33.43, -111.95},
	"NWS Las Vegas NV":               {36.05, -115.18},
	"NWS Albuquerque NM":             {35.04, -106.62},
	"NWS El Paso Tx/Santa Teresa NM": {31.87, -106.70},
	"NWS Pueblo CO":                  {38.28, -104.52},
	"NWS Grand Junction CO":          {39.12, -108.53},
	"NWS Salt Lake City UT":          {40.78, -111.97},
	"NWS Flagstaff AZ":               {35.23, -111.82},
	"NWS Midland/Odessa TX":          {31.94, -102.19},
	"NWS Great Falls MT":             {47.46, -111.38},
	"NWS North Platte NE":            {41.13, -100.68},
	"NWS Key West FL":                {24.55, -81.75},
	"NWS Honolulu HI":                {21.32, -157.92},
	"NWS Chicago IL":                 {41.53, -88.08},
	"NWS Quad Cities IA/IL":          {41.61, -90.58},
	"NWS Topeka KS":                  {39.07, -95.63},
	"NWS Melbourne FL":               {28.11, -80.65},
	"NWS Fort Worth TX":              {32.83, -97.30},
	"NWS Norman OK":                  {35.24, -97.46},
	"NWS Northern Indiana":           {41.36, -85.70},
}

var stateCenters = map[string][2]float64{
	"AL": {32.8, -86.8}, "AK": {64.2, -152.5}, "AZ": {34.0, -111.1},
	"AR": {35.2, -92.4}, "CA": {36.8, -119.4}, "CO": {39.5, -105.8},
	"CT": {41.6, -72.7}, "DE": {39.0, -75.5}, "FL": {27.7, -81.7},
	"GA": {32.2, -82.9}, "HI": {21.3, -157.8}, "ID": {44.1, -114.7},
	"IL": {40.6, -89.4}, "IN": {40.3, -86.1}, "IA": {41.9, -93.4},
	"KS": {38.5, -98.5}, "KY": {37.8, -84.3}, "LA": {31.0, -91.9},
	"ME": {45.3, -69.4}, "MD": {39.0, -76.6}, "MA": {42.4, -71.4},
	"MI": {44.3, -84.5}, "MN": {46.7, -94.7}, "MS": {32.3, -89.4},
	"MO": {37.9, -91.8}, "MT": {46.9, -110.5}, "NE": {41.5, -99.9},
	"NV": {38.8, -116.4}, "NH": {43.2, -71.6}, "NJ": {40.1, -74.5},
	"NM": {34.5, -105.9}, "NY": {43.3, -74.2}, "NC": {35.8, -79.0},
	"ND": {47.5, -100.5}, "OH": {40.4, -82.9}, "OK": {35.6, -96.9},
	"OR": {43.8, -120.6}, "PA": {41.2, -77.2}, "RI": {41.6, -71.5},
	"SC": {33.8, -81.2}, "SD": {43.9, -99.9}, "TN": {35.5, -86.6},
	"TX": {31.0, -99.9}, "UT": {39.3, -111.1}, "VT": {44.6, -72.6},
	"VA": {37.4, -78.7}, "WA": {47.8, -120.7}, "WV": {38.6, -80.5},
	"WI": {43.8, -89.5}, "WY": {43.1, -107.3},
}
