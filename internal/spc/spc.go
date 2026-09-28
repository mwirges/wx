package spc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

const (
	defaultOutlooksMapServer = "https://mapservices.weather.noaa.gov/vector/rest/services/outlooks/SPC_wx_outlks/MapServer"
	defaultMCDMapServer      = "https://mapservices.weather.noaa.gov/vector/rest/services/outlooks/spc_mesoscale_discussion/MapServer/0/query"
	defaultWatchesURL        = "https://api.weather.gov/alerts/active?event=Tornado%20Watch,Severe%20Thunderstorm%20Watch"
	defaultMDBaseURL         = "https://www.spc.noaa.gov/products/md"

	cacheTTLOutlooks = 15 * time.Minute
	cacheTTLMCD      = 3 * time.Minute
	cacheTTLWatches  = 2 * time.Minute
)

var (
	reMCDNumber  = regexp.MustCompile(`(?i)(?:MD|MCD)\s*(\d+)`)
	reAreas      = regexp.MustCompile(`(?i)Areas affected\.\.\.([^\n\r]+)`)
	reConcerning = regexp.MustCompile(`(?i)Concerning\.\.\.([^\n\r]+)`)
	reWatchProb  = regexp.MustCompile(`(?i)Probability of Watch Issuance\.\.\.([^\n\r]+)`)
	reSummary    = regexp.MustCompile(`(?i)SUMMARY\.\.\.([\s\S]*?)(?:DISCUSSION\.\.\.|\.\.|\n\n\n)`)
	reWatchNum   = regexp.MustCompile(`(?i)(?:Tornado Watch|Severe Thunderstorm Watch)\s+(\d+)`)
	reVTECWatch  = regexp.MustCompile(`\.(?:TO|SV)\.A\.(\d{4})\.`)
)

type esriFeature struct {
	Attributes map[string]interface{} `json:"attributes"`
}

type esriQueryResponse struct {
	Features []esriFeature `json:"features"`
}

type geoJSONFeatureCollection struct {
	Features []struct {
		ID       any `json:"id"`
		Geometry struct {
			Type        string `json:"type"`
			Coordinates any    `json:"coordinates"`
		} `json:"geometry"`
		Properties map[string]any `json:"properties"`
	} `json:"features"`
}

type nwsWatchAlertsResponse struct {
	Features []struct {
		ID         string `json:"id"`
		Properties struct {
			Event       string `json:"event"`
			Headline    string `json:"headline"`
			AreaDesc    string `json:"areaDesc"`
			Description string `json:"description"`
			Effective   string `json:"effective"`
			Expires     string `json:"expires"`
			Severity    string `json:"severity"`
			Urgency     string `json:"urgency"`
			Parameters  struct {
				VTEC []string `json:"VTEC"`
			} `json:"parameters"`
			Geocode struct {
				UGC []string `json:"UGC"`
			} `json:"geocode"`
		} `json:"properties"`
	} `json:"features"`
}

// Client queries NOAA SPC MapServers and NWS watch endpoints.
type Client struct {
	httpClient  *http.Client
	outlooksURL string
	mcdURL      string
	watchesURL  string
	mdBaseURL   string
}

// NewClient returns a new Client with standard NOAA endpoints.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		outlooksURL: defaultOutlooksMapServer,
		mcdURL:      defaultMCDMapServer,
		watchesURL:  defaultWatchesURL,
		mdBaseURL:   defaultMDBaseURL,
	}
}

// NewCustomClient returns a client with custom URLs (useful for unit testing).
func NewCustomClient(outlooksURL, mcdURL, watchesURL, mdBaseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		httpClient:  httpClient,
		outlooksURL: outlooksURL,
		mcdURL:      mcdURL,
		watchesURL:  watchesURL,
		mdBaseURL:   mdBaseURL,
	}
}

var defaultClient = NewClient()

// FetchSPCOutlooks queries convective outlooks, active MCDs, and active watches for a location.
func FetchSPCOutlooks(ctx context.Context, loc location.Location, ch *cache.Cache) (*models.SPCPayload, error) {
	return defaultClient.FetchSPCOutlooks(ctx, loc, ch)
}

// FetchActiveMCDs queries active Mesoscale Discussions nationwide.
func FetchActiveMCDs(ctx context.Context, ch *cache.Cache) ([]models.MesoscaleDiscussion, error) {
	return defaultClient.FetchActiveMCDs(ctx, ch)
}

// FetchActiveWatches queries active Tornado and Severe Thunderstorm Watches nationwide.
func FetchActiveWatches(ctx context.Context, ch *cache.Cache) ([]models.SPCWatch, error) {
	return defaultClient.FetchActiveWatches(ctx, ch)
}

// FetchNationalMaxRisk retrieves the highest active SPC convective risk across CONUS.
func FetchNationalMaxRisk(ctx context.Context, ch *cache.Cache) (models.SPCRiskCategory, error) {
	return defaultClient.FetchNationalMaxRisk(ctx, ch)
}

// FetchSPCOutlooks retrieves convective outlooks, active MCDs, and active watches for a location.
func (cl *Client) FetchSPCOutlooks(ctx context.Context, loc location.Location, ch *cache.Cache) (*models.SPCPayload, error) {
	cacheKey := fmt.Sprintf("spc:outlook:v1:%.4f,%.4f", loc.Lat, loc.Lon)
	if ch != nil {
		var cached models.SPCPayload
		if ch.Get(cacheKey, &cached) {
			return &cached, nil
		}
	}

	var (
		day1, day2, day3 models.SPCOutlookItem
		mcds             []models.MesoscaleDiscussion
		watches          []models.SPCWatch
		maxRisk          models.SPCRiskCategory
		wg               sync.WaitGroup
	)

	// Fetch Day 1 outlook
	wg.Add(1)
	go func() {
		defer wg.Done()
		day1 = cl.fetchDay1(ctx, loc.Lat, loc.Lon)
	}()

	// Fetch Day 2 outlook
	wg.Add(1)
	go func() {
		defer wg.Done()
		day2 = cl.fetchDay2(ctx, loc.Lat, loc.Lon)
	}()

	// Fetch Day 3 outlook
	wg.Add(1)
	go func() {
		defer wg.Done()
		day3 = cl.fetchDay3(ctx, loc.Lat, loc.Lon)
	}()

	// Fetch active MCDs
	wg.Add(1)
	go func() {
		defer wg.Done()
		if res, err := cl.FetchActiveMCDs(ctx, ch); err == nil {
			mcds = res
		}
	}()

	// Fetch active watches
	wg.Add(1)
	go func() {
		defer wg.Done()
		if res, err := cl.FetchActiveWatches(ctx, ch); err == nil {
			watches = res
		}
	}()

	// Fetch National Max Risk
	wg.Add(1)
	go func() {
		defer wg.Done()
		if res, err := cl.FetchNationalMaxRisk(ctx, ch); err == nil {
			maxRisk = res
		}
	}()

	wg.Wait()

	summary := synthesizeConvectiveSummary(day1, day2, day3, mcds, watches)

	payload := &models.SPCPayload{
		Location:          loc.DisplayName,
		Coordinates:       [2]float64{loc.Lat, loc.Lon},
		FetchedAt:         time.Now().UTC(),
		Day1:              day1,
		Day2:              day2,
		Day3:              day3,
		ActiveMCDs:        mcds,
		ActiveWatches:     watches,
		MaxNationalRisk:   maxRisk,
		ConvectiveSummary: summary,
	}

	if ch != nil {
		_ = ch.Set(cacheKey, payload, cacheTTLOutlooks)
	}

	return payload, nil
}

// FetchActiveMCDs retrieves currently active SPC Mesoscale Discussions.
func (cl *Client) FetchActiveMCDs(ctx context.Context, ch *cache.Cache) ([]models.MesoscaleDiscussion, error) {
	cacheKey := "spc:mcd:active:v1"
	if ch != nil {
		var cached []models.MesoscaleDiscussion
		if ch.Get(cacheKey, &cached) {
			return cached, nil
		}
	}

	reqURL := fmt.Sprintf("%s?where=1%%3D1&outFields=*&f=geojson", cl.mcdURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCD HTTP %d", resp.StatusCode)
	}

	var data geoJSONFeatureCollection
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var mcds []models.MesoscaleDiscussion
	for _, feat := range data.Features {
		name, _ := feat.Properties["name"].(string)
		title, _ := feat.Properties["folderpath"].(string)
		popupURL, _ := feat.Properties["popupinfo"].(string)
		lat, lon := extractGeoCentroid(feat.Geometry.Type, feat.Geometry.Coordinates)

		mcdNum := 0
		if m := reMCDNumber.FindStringSubmatch(name); len(m) > 1 {
			mcdNum, _ = strconv.Atoi(m[1])
		}

		if popupURL != "" && strings.HasPrefix(popupURL, "http://") {
			popupURL = "https://" + strings.TrimPrefix(popupURL, "http://")
		}

		mcd := models.MesoscaleDiscussion{
			ID:            mcdNum,
			Name:          name,
			Title:         title,
			URL:           popupURL,
			Lat:           lat,
			Lon:           lon,
			AreasAffected: "CONUS Regional Sector",
			Concerning:    "Severe Potential",
		}

		if fileDate, ok := feat.Properties["idp_filedate"].(float64); ok && fileDate > 0 {
			mcd.Sent = time.UnixMilli(int64(fileDate)).UTC()
		}

		// Try to enrich from discussion text
		if mcdNum > 0 {
			cl.enrichMCDFromWeb(ctx, &mcd)
		}

		mcds = append(mcds, mcd)
	}

	sort.Slice(mcds, func(i, j int) bool {
		return mcds[i].ID > mcds[j].ID
	})

	if ch != nil {
		_ = ch.Set(cacheKey, mcds, cacheTTLMCD)
	}

	return mcds, nil
}

func (cl *Client) enrichMCDFromWeb(ctx context.Context, mcd *models.MesoscaleDiscussion) {
	fetchURL := ""
	if cl.mdBaseURL != "" && mcd.ID > 0 {
		fetchURL = fmt.Sprintf("%s/md%04d.html", cl.mdBaseURL, mcd.ID)
	} else if mcd.URL != "" {
		fetchURL = mcd.URL
	}
	if fetchURL == "" {
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return
	}
	text := string(body)

	if m := reAreas.FindStringSubmatch(text); len(m) > 1 {
		mcd.AreasAffected = strings.TrimSpace(m[1])
	}
	if m := reConcerning.FindStringSubmatch(text); len(m) > 1 {
		mcd.Concerning = strings.TrimSpace(m[1])
	}
	if m := reWatchProb.FindStringSubmatch(text); len(m) > 1 {
		mcd.WatchProbability = strings.TrimSpace(m[1])
	}
	if m := reSummary.FindStringSubmatch(text); len(m) > 1 {
		cleanSummary := strings.Join(strings.Fields(m[1]), " ")
		mcd.Summary = cleanSummary
	}
}

// FetchActiveWatches queries active Tornado and Severe Thunderstorm Watches nationwide.
func (cl *Client) FetchActiveWatches(ctx context.Context, ch *cache.Cache) ([]models.SPCWatch, error) {
	cacheKey := "spc:watches:active:v1"
	if ch != nil {
		var cached []models.SPCWatch
		if ch.Get(cacheKey, &cached) {
			return cached, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cl.watchesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")
	req.Header.Set("Accept", "application/geo+json")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("watches HTTP %d", resp.StatusCode)
	}

	var data nwsWatchAlertsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var watches []models.SPCWatch
	seen := make(map[string]bool)

	for _, feat := range data.Features {
		p := feat.Properties
		eff, _ := time.Parse(time.RFC3339, p.Effective)
		exp, _ := time.Parse(time.RFC3339, p.Expires)

		watchNum := 0
		if len(p.Parameters.VTEC) > 0 {
			if m := reVTECWatch.FindStringSubmatch(p.Parameters.VTEC[0]); len(m) > 1 {
				watchNum, _ = strconv.Atoi(m[1])
			}
		}
		if watchNum == 0 {
			if m := reWatchNum.FindStringSubmatch(p.Headline); len(m) > 1 {
				watchNum, _ = strconv.Atoi(m[1])
			}
		}

		key := fmt.Sprintf("%s-%d", p.Event, watchNum)
		if seen[key] && watchNum > 0 {
			continue
		}
		seen[key] = true

		states := extractStatesFromUGC(p.Geocode.UGC)

		watch := models.SPCWatch{
			ID:          feat.ID,
			WatchNumber: watchNum,
			Type:        p.Event,
			Headline:    p.Headline,
			AreaDesc:    p.AreaDesc,
			States:      states,
			Effective:   eff,
			Expires:     exp,
			Active:      true,
			Severity:    p.Severity,
			Urgency:     p.Urgency,
		}
		if watchNum > 0 {
			watch.URL = fmt.Sprintf("https://www.spc.noaa.gov/products/watch/ww%04d.html", watchNum)
		}

		watches = append(watches, watch)
	}

	sort.Slice(watches, func(i, j int) bool {
		return watches[i].WatchNumber > watches[j].WatchNumber
	})

	if ch != nil {
		_ = ch.Set(cacheKey, watches, cacheTTLWatches)
	}

	return watches, nil
}

// FetchNationalMaxRisk retrieves the highest active SPC convective risk across CONUS.
func (cl *Client) FetchNationalMaxRisk(ctx context.Context, ch *cache.Cache) (models.SPCRiskCategory, error) {
	cacheKey := "spc:national:max:v1"
	if ch != nil {
		var cached models.SPCRiskCategory
		if ch.Get(cacheKey, &cached) {
			return cached, nil
		}
	}

	endpoint := fmt.Sprintf("%s/1/query?where=1%%3D1&outFields=*&returnGeometry=false&f=json", cl.outlooksURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return RiskNone(), err
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return RiskNone(), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RiskNone(), fmt.Errorf("outlooks HTTP %d", resp.StatusCode)
	}

	var data esriQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return RiskNone(), err
	}

	maxCat := RiskNone()
	for _, feat := range data.Features {
		cat := parseCategoryAttrs(feat.Attributes)
		if cat.DN > maxCat.DN {
			maxCat = cat
		}
	}

	if ch != nil {
		_ = ch.Set(cacheKey, maxCat, cacheTTLOutlooks)
	}

	return maxCat, nil
}

func (cl *Client) fetchDay1(ctx context.Context, lat, lon float64) models.SPCOutlookItem {
	var (
		cat                   models.SPCRiskCategory
		tornProb, hailP, winP string
		tornSig, hailS, winS  bool
		valid, expire, issue  time.Time
		wg                    sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		cat, valid, expire, issue = cl.queryCategoricalLayer(ctx, 1, lat, lon)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tornProb, tornSig = cl.queryProbLayer(ctx, 3, lat, lon)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		hailP, hailS = cl.queryProbLayer(ctx, 5, lat, lon)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		winP, winS = cl.queryProbLayer(ctx, 7, lat, lon)
	}()

	wg.Wait()

	return models.SPCOutlookItem{
		Day:         1,
		Valid:       valid,
		Expires:     expire,
		Issue:       issue,
		Category:    cat,
		TornadoProb: tornProb,
		TornadoSig:  tornSig,
		HailProb:    hailP,
		HailSig:     hailS,
		WindProb:    winP,
		WindSig:     winS,
	}
}

func (cl *Client) fetchDay2(ctx context.Context, lat, lon float64) models.SPCOutlookItem {
	var (
		cat                   models.SPCRiskCategory
		tornProb, hailP, winP string
		tornSig, hailS, winS  bool
		valid, expire, issue  time.Time
		wg                    sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		cat, valid, expire, issue = cl.queryCategoricalLayer(ctx, 9, lat, lon)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tornProb, tornSig = cl.queryProbLayer(ctx, 11, lat, lon)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		hailP, hailS = cl.queryProbLayer(ctx, 13, lat, lon)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		winP, winS = cl.queryProbLayer(ctx, 15, lat, lon)
	}()

	wg.Wait()

	return models.SPCOutlookItem{
		Day:         2,
		Valid:       valid,
		Expires:     expire,
		Issue:       issue,
		Category:    cat,
		TornadoProb: tornProb,
		TornadoSig:  tornSig,
		HailProb:    hailP,
		HailSig:     hailS,
		WindProb:    winP,
		WindSig:     winS,
	}
}

func (cl *Client) fetchDay3(ctx context.Context, lat, lon float64) models.SPCOutlookItem {
	var (
		cat                  models.SPCRiskCategory
		sevProb              string
		sevSig               bool
		valid, expire, issue time.Time
		wg                   sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		cat, valid, expire, issue = cl.queryCategoricalLayer(ctx, 17, lat, lon)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sevProb, sevSig = cl.queryProbLayer(ctx, 19, lat, lon)
	}()

	wg.Wait()

	return models.SPCOutlookItem{
		Day:        3,
		Valid:      valid,
		Expires:    expire,
		Issue:      issue,
		Category:   cat,
		SevereProb: sevProb,
		SevereSig:  sevSig,
	}
}

func (cl *Client) queryCategoricalLayer(ctx context.Context, layerID int, lat, lon float64) (models.SPCRiskCategory, time.Time, time.Time, time.Time) {
	params := url.Values{}
	params.Set("geometry", fmt.Sprintf("%.4f,%.4f", lon, lat))
	params.Set("geometryType", "esriGeometryPoint")
	params.Set("inSR", "4326")
	params.Set("spatialRel", "esriSpatialRelIntersects")
	params.Set("outFields", "*")
	params.Set("returnGeometry", "false")
	params.Set("f", "json")

	endpoint := fmt.Sprintf("%s/%d/query?%s", cl.outlooksURL, layerID, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return RiskNone(), time.Time{}, time.Time{}, time.Time{}
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return RiskNone(), time.Time{}, time.Time{}, time.Time{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RiskNone(), time.Time{}, time.Time{}, time.Time{}
	}

	var data esriQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return RiskNone(), time.Time{}, time.Time{}, time.Time{}
	}

	if len(data.Features) == 0 {
		return RiskNone(), time.Time{}, time.Time{}, time.Time{}
	}

	maxCat := RiskNone()
	var topAttrs map[string]interface{}
	for _, f := range data.Features {
		cat := parseCategoryAttrs(f.Attributes)
		if cat.DN > maxCat.DN {
			maxCat = cat
			topAttrs = f.Attributes
		}
	}

	var valid, expire, issue time.Time
	if topAttrs != nil {
		valid = parseSPCTime(topAttrs["valid"])
		expire = parseSPCTime(topAttrs["expire"])
		issue = parseSPCTime(topAttrs["issue"])
	}

	return maxCat, valid, expire, issue
}

func (cl *Client) queryProbLayer(ctx context.Context, layerID int, lat, lon float64) (string, bool) {
	params := url.Values{}
	params.Set("geometry", fmt.Sprintf("%.4f,%.4f", lon, lat))
	params.Set("geometryType", "esriGeometryPoint")
	params.Set("inSR", "4326")
	params.Set("spatialRel", "esriSpatialRelIntersects")
	params.Set("outFields", "*")
	params.Set("returnGeometry", "false")
	params.Set("f", "json")

	endpoint := fmt.Sprintf("%s/%d/query?%s", cl.outlooksURL, layerID, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "None", false
	}
	req.Header.Set("User-Agent", "wx/1.0 (github.com/mwirges/wx)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return "None", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "None", false
	}

	var data esriQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "None", false
	}

	if len(data.Features) == 0 {
		return "None", false
	}

	maxProb := 0
	probStr := ""
	isSig := false

	for _, f := range data.Features {
		attrs := f.Attributes
		lbl, _ := attrs["label"].(string)
		lbl2, _ := attrs["label2"].(string)

		combined := strings.ToUpper(lbl + " " + lbl2)
		if strings.Contains(combined, "SIGN") || strings.Contains(combined, "HATCH") {
			isSig = true
		}

		dn := 0
		switch v := attrs["dn"].(type) {
		case float64:
			dn = int(v)
		case int:
			dn = v
		}

		if dn > 0 && dn < 100 {
			if dn > maxProb {
				maxProb = dn
				probStr = fmt.Sprintf("%d%%", dn)
			}
		} else if lbl2 != "" && strings.Contains(lbl2, "%") {
			probStr = lbl2
		} else if lbl != "" && strings.HasPrefix(lbl, "0.") {
			if fl, err := strconv.ParseFloat(lbl, 64); err == nil {
				pct := int(math.Round(fl * 100))
				if pct > maxProb {
					maxProb = pct
					probStr = fmt.Sprintf("%d%%", pct)
				}
			}
		}
	}

	if probStr == "" {
		probStr = "None"
	}
	return probStr, isSig
}

func parseCategoryAttrs(attrs map[string]interface{}) models.SPCRiskCategory {
	if attrs == nil {
		return RiskNone()
	}

	dn := 0
	switch v := attrs["dn"].(type) {
	case float64:
		dn = int(v)
	case int:
		dn = v
	}

	label, _ := attrs["label"].(string)
	label2, _ := attrs["label2"].(string)
	return CategoryFromDN(dn, label, label2)
}

// CategoryFromDN converts an SPC DN value or label into a structured SPCRiskCategory.
func CategoryFromDN(dn int, label, label2 string) models.SPCRiskCategory {
	upperLabel := strings.ToUpper(strings.TrimSpace(label))
	upperLabel2 := strings.ToUpper(strings.TrimSpace(label2))

	switch {
	case dn == 8 || upperLabel == "HIGH" || strings.Contains(upperLabel2, "HIGH RISK"):
		return models.SPCRiskCategory{
			DN:          8,
			Code:        "HIGH",
			Name:        "5 - High Risk",
			Description: "Severe weather outbreak expected; violent/long-track tornadoes or derecho",
			Color:       "#FF00FF",
		}
	case dn == 6 || upperLabel == "MDT" || strings.Contains(upperLabel2, "MODERATE RISK"):
		return models.SPCRiskCategory{
			DN:          6,
			Code:        "MDT",
			Name:        "4 - Moderate Risk",
			Description: "Widespread severe storms likely; long-lived supercells, intense wind/hail/tornadoes",
			Color:       "#FF4500",
		}
	case dn == 5 || upperLabel == "ENH" || strings.Contains(upperLabel2, "ENHANCED RISK"):
		return models.SPCRiskCategory{
			DN:          5,
			Code:        "ENH",
			Name:        "3 - Enhanced Risk",
			Description: "Numerous severe storms likely; more persistent/widespread threats",
			Color:       "#FFA500",
		}
	case dn == 4 || upperLabel == "SLGT" || strings.Contains(upperLabel2, "SLIGHT RISK"):
		return models.SPCRiskCategory{
			DN:          4,
			Code:        "SLGT",
			Name:        "2 - Slight Risk",
			Description: "Scattered severe storms possible; short-lived or isolated supercells",
			Color:       "#E6E600",
		}
	case dn == 3 || upperLabel == "MRGL" || strings.Contains(upperLabel2, "MARGINAL RISK"):
		return models.SPCRiskCategory{
			DN:          3,
			Code:        "MRGL",
			Name:        "1 - Marginal Risk",
			Description: "Isolated severe storms possible; limited duration and intensity",
			Color:       "#7FA57F",
		}
	case dn == 2 || upperLabel == "TSTM" || strings.Contains(upperLabel2, "GENERAL THUNDERSTORM"):
		return models.SPCRiskCategory{
			DN:          2,
			Code:        "TSTM",
			Name:        "General Thunderstorms",
			Description: "Thunderstorms possible; severe storms not expected (<10% probability)",
			Color:       "#55BB55",
		}
	default:
		return RiskNone()
	}
}

// RiskNone returns the default category when no severe threat is expected.
func RiskNone() models.SPCRiskCategory {
	return models.SPCRiskCategory{
		DN:          0,
		Code:        "NONE",
		Name:        "No Severe Expected",
		Description: "No organized severe thunderstorms expected",
		Color:       "#A0A0A0",
	}
}

func parseSPCTime(val interface{}) time.Time {
	if val == nil {
		return time.Time{}
	}
	switch v := val.(type) {
	case string:
		v = strings.TrimSpace(v)
		if len(v) == 12 {
			if t, err := time.Parse("200601021504", v); err == nil {
				return t.UTC()
			}
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UTC()
		}
	case float64:
		if v > 0 {
			return time.UnixMilli(int64(v)).UTC()
		}
	}
	return time.Time{}
}

func extractGeoCentroid(geomType string, rawCoords any) (float64, float64) {
	if rawCoords == nil {
		return 0, 0
	}
	rawBytes, err := json.Marshal(rawCoords)
	if err != nil {
		return 0, 0
	}

	if geomType == "Polygon" {
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
			return sumLat / n, sumLon / n
		}
	} else if geomType == "MultiPolygon" {
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
				return sumLat / count, sumLon / count
			}
		}
	}
	return 0, 0
}

func extractStatesFromUGC(ugcs []string) []string {
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

func synthesizeConvectiveSummary(d1, d2, d3 models.SPCOutlookItem, mcds []models.MesoscaleDiscussion, watches []models.SPCWatch) string {
	var parts []string

	if d1.Category.DN > 0 {
		parts = append(parts, fmt.Sprintf("Day 1: %s (%s)", d1.Category.Name, d1.Category.Description))
	} else {
		parts = append(parts, "Day 1: No organized severe weather expected")
	}

	if d2.Category.DN > 0 {
		parts = append(parts, fmt.Sprintf("Day 2: %s", d2.Category.Name))
	}
	if d3.Category.DN > 0 {
		parts = append(parts, fmt.Sprintf("Day 3: %s", d3.Category.Name))
	}

	if len(watches) > 0 {
		parts = append(parts, fmt.Sprintf("%d active severe watch(es) in effect", len(watches)))
	}

	if len(mcds) > 0 {
		parts = append(parts, fmt.Sprintf("%d active Mesoscale Discussion(s)", len(mcds)))
	}

	return strings.Join(parts, ". ") + "."
}
