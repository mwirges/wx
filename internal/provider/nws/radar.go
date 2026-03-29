package nws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/png" // register PNG decoder
	"image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/radar"
)

const (
	radarWMSBase = "https://opengeo.ncep.noaa.gov/geoserver/conus/ows"
	// IEM radmap is used for historical loop frames — cleaner timestamp API.
	iemRadmapBase = "https://mesonet.agron.iastate.edu/GIS/radmap.php"
	// nwsAPIBase is used for NEXRAD station metadata lookups.
	nwsAPIBase = "https://api.weather.gov"

	// radarFrameInterval is the MRMS update cadence (≈2 min, rounded up).
	radarFrameInterval = 5 * time.Minute

	// currentFrameTTL is how long the most-recent frame is cached on disk.
	currentFrameTTL = 5 * time.Minute

	// historicalFrameTTL is how long past loop frames are cached on disk.
	// Historical radar frames never change, so a long TTL is appropriate.
	historicalFrameTTL = 24 * time.Hour

)

// nwsWMSLayers maps our product codes to NWS MRMS WMS layer names.
// Echo tops uses the MRMS Enhanced Echo Tops national mosaic, composited
// on top of a radmap overlay base map (two-request approach).
var nwsWMSLayers = map[radar.Product]string{
	radar.ProductCompositeReflectivity: "conus_cref_qcd",
	radar.ProductBaseReflectivity:      "conus_bref_qcd",
	radar.ProductEchoTops:              "conus_neet_v18",
}

// iemProducts maps our product codes to IEM radmap product codes.
// Composite reflectivity uses the national mosaic layer, not RIDGE.
// Station products (base refl, SRV) use single-station RIDGE.
// Echo tops uses NWS WMS (not IEM) — see nwsWMSLayers.
var iemProducts = map[radar.Product]string{
	radar.ProductCompositeReflectivity: "N0Q",
	radar.ProductBaseReflectivity:      "N0B",
	radar.ProductStormRelativeVelocity: "N0S",
	radar.ProductEchoTops:              "NET", // used only for product validation
}

// isWMSCompositeProduct returns true if the product uses NWS WMS for radar
// data composited on top of an IEM radmap overlay base map.
func isWMSCompositeProduct(p radar.Product) bool {
	return p == radar.ProductEchoTops
}

// ridgeStationCode converts a 4-letter NEXRAD ID (e.g. "KEAX") to the
// 3-letter code used in IEM's RIDGE archive (e.g. "EAX").
// Non-CONUS prefixes (P, T) are also stripped.
func ridgeStationCode(id string) string {
	if len(id) == 4 {
		prefix := id[0]
		if prefix == 'K' || prefix == 'P' || prefix == 'T' {
			return id[1:]
		}
	}
	return id
}

// compositeOver draws fg on top of bg using alpha compositing.
// Both images must have the same dimensions.
func compositeOver(bg, fg image.Image) image.Image {
	b := bg.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, bg, b.Min, draw.Src)
	draw.Draw(dst, b, fg, fg.Bounds().Min, draw.Over)
	return dst
}

// ── bbox ──────────────────────────────────────────────────────────────────────

type radarBBox struct {
	MinLat, MinLon, MaxLat, MaxLon float64
}

// boundingBox returns a bounding box of radiusKM km around (lat, lon).
func boundingBox(lat, lon, radiusKM float64) radarBBox {
	dLat := radiusKM / 111.0
	dLon := radiusKM / (111.0 * math.Cos(lat*math.Pi/180.0))
	return radarBBox{
		MinLat: lat - dLat, MinLon: lon - dLon,
		MaxLat: lat + dLat, MaxLon: lon + dLon,
	}
}

// ── disk cache serialisation ──────────────────────────────────────────────────

// cachedRadarPNG is the on-disk representation of a radar frame.
// PNG bytes serialise as base64 in JSON, which cache.Cache handles natively.
type cachedRadarPNG struct {
	PNG       []byte        `json:"png"`
	ValidTime time.Time     `json:"valid_time"`
	Product   radar.Product `json:"product"`
	MinLat    float64       `json:"min_lat,omitempty"`
	MinLon    float64       `json:"min_lon,omitempty"`
	MaxLat    float64       `json:"max_lat,omitempty"`
	MaxLon    float64       `json:"max_lon,omitempty"`
}

func (p *RadarProvider) diskGet(c *cache.Cache, key string) (*radar.Frame, bool) {
	var cd cachedRadarPNG
	if !c.Get(key, &cd) {
		return nil, false
	}
	img, _, err := image.Decode(bytes.NewReader(cd.PNG))
	if err != nil {
		return nil, false
	}
	f := &radar.Frame{Img: img, ValidTime: cd.ValidTime, Product: cd.Product}
	if cd.MinLat != 0 || cd.MinLon != 0 || cd.MaxLat != 0 || cd.MaxLon != 0 {
		f.BBox = &radar.BBox{MinLat: cd.MinLat, MinLon: cd.MinLon, MaxLat: cd.MaxLat, MaxLon: cd.MaxLon}
	}
	return f, true
}

func (p *RadarProvider) diskSet(c *cache.Cache, key string, f *radar.Frame, ttl time.Duration) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, f.Img); err != nil {
		return // best-effort; a cache miss on the next run is harmless
	}
	cd := cachedRadarPNG{
		PNG:       buf.Bytes(),
		ValidTime: f.ValidTime,
		Product:   f.Product,
	}
	if f.BBox != nil {
		cd.MinLat = f.BBox.MinLat
		cd.MinLon = f.BBox.MinLon
		cd.MaxLat = f.BBox.MaxLat
		cd.MaxLon = f.BBox.MaxLon
	}
	_ = c.Set(key, cd, ttl)
}

// ── in-process L1 cache ───────────────────────────────────────────────────────

// radarCacheEntry holds a decoded radar frame with an expiry time.
// This L1 cache avoids redundant PNG decoding within a single wx invocation
// (most useful during loop animation where the same frames cycle repeatedly).
type radarCacheEntry struct {
	frame     *radar.Frame
	expiresAt time.Time
}

func (p *RadarProvider) imgGet(key string) *radar.Frame {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.imgCache[key]
	if !ok || (!e.expiresAt.IsZero() && time.Now().After(e.expiresAt)) {
		return nil
	}
	return e.frame
}

func (p *RadarProvider) imgSet(key string, f *radar.Frame, ttl time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	p.imgCache[key] = radarCacheEntry{frame: f, expiresAt: exp}
}

// ── RadarProvider ─────────────────────────────────────────────────────────────

// RadarProvider implements radar.Provider using the NWS MRMS WMS for current
// frames and the Iowa State IEM radmap API for historical loop frames.
type RadarProvider struct {
	wmsBase    string
	iemBase    string
	nwsAPIBase string

	mu       sync.RWMutex
	imgCache map[string]radarCacheEntry
}

func newRadarProvider() *RadarProvider {
	return &RadarProvider{
		wmsBase:    radarWMSBase,
		iemBase:    iemRadmapBase,
		nwsAPIBase: nwsAPIBase,
		imgCache:   make(map[string]radarCacheEntry),
	}
}

func init() {
	radar.Register(newRadarProvider())
}

func (p *RadarProvider) Name() string { return "nws-radar" }

func (p *RadarProvider) Supports(loc location.Location) bool {
	return loc.CountryCode == "US"
}

// ── CurrentFrame ──────────────────────────────────────────────────────────────

// CurrentFrame fetches the most recent radar frame via IEM radmap, using the
// same overlay layers as RecentFrames so that the current image and loop frames
// are rendered consistently (IEM composites radar + geographic features
// server-side, avoiding any transparent-background issues with client-side
// compositing).
func (p *RadarProvider) CurrentFrame(ctx context.Context, loc location.Location, opts radar.Options, c *cache.Cache) (*radar.Frame, error) {
	if _, ok := iemProducts[opts.Product]; !ok {
		return nil, fmt.Errorf("nws radar: unsupported product %q", opts.Product)
	}

	bb := boundingBox(loc.Lat, loc.Lon, opts.RadiusKM)
	key := fmt.Sprintf("nws:radar:cur:v3:%s:%.4f,%.4f:%.0f", opts.Product, loc.Lat, loc.Lon, opts.RadiusKM)

	// L1: in-process (avoids re-decoding PNG within the same invocation)
	if f := p.imgGet(key); f != nil {
		return f, nil
	}
	// L2: disk (persists between wx invocations — 5-minute TTL)
	if f, ok := p.diskGet(c, key); ok {
		p.imgSet(key, f, currentFrameTTL)
		return f, nil
	}

	now := time.Now().UTC().Truncate(radarFrameInterval)

	params := url.Values{}
	p.addRadarLayers(params, loc, opts)
	params.Set("width", "1600")
	params.Set("height", "1600")
	params.Set("bbox", fmt.Sprintf("%.6f,%.6f,%.6f,%.6f",
		bb.MinLon, bb.MinLat, bb.MaxLon, bb.MaxLat))
	params.Set("fmt", "png")
	params.Set("ts", now.Format("200601021504"))

	img, _, err := p.fetchImageURL(ctx, p.iemBase+"?"+params.Encode())
	if err != nil {
		return nil, fmt.Errorf("nws radar current: %w", err)
	}

	// For WMS composite products (echo tops), overlay radar data from NWS WMS.
	if isWMSCompositeProduct(opts.Product) {
		if layer, ok := nwsWMSLayers[opts.Product]; ok {
			radarImg, _, wmsErr := p.fetchWMSFrame(ctx, layer, bb, "")
			if wmsErr == nil {
				img = compositeOver(img, radarImg)
			}
		}
	}

	f := &radar.Frame{
		Img: img, ValidTime: now, Product: opts.Product,
		BBox: &radar.BBox{MinLat: bb.MinLat, MinLon: bb.MinLon, MaxLat: bb.MaxLat, MaxLon: bb.MaxLon},
	}
	p.diskSet(c, key, f, currentFrameTTL)
	p.imgSet(key, f, currentFrameTTL)
	return f, nil
}

// addRadarLayers configures the IEM radmap layers[] and product params.
// Composite reflectivity uses the national mosaic. Station products (base
// reflectivity, SRV) use single-station RIDGE. Echo tops uses only
// overlays here — the radar data is fetched separately via NWS WMS.
func (p *RadarProvider) addRadarLayers(params url.Values, loc location.Location, opts radar.Options) {
	if isWMSCompositeProduct(opts.Product) {
		// Radar data comes from NWS WMS; radmap provides only the base map
		// and geographic overlays. No radar layer added here.
	} else if radar.IsStationProduct(opts.Product) {
		// Single-station RIDGE mode — IEM uses 3-letter station codes.
		station := radar.NearestStation(loc.Lat, loc.Lon)
		params.Add("layers[]", "ridge")
		params.Set("ridge_radar", ridgeStationCode(station.ID))
		params.Set("ridge_product", iemProducts[opts.Product])
	} else {
		// National composite mosaic.
		params.Add("layers[]", "nexrad")
	}

	// Geographic overlay layers (always included).
	params.Add("layers[]", "usstates")
	params.Add("layers[]", "uscounties")
	params.Add("layers[]", "places")
	params.Add("layers[]", "interstates")
}

// ── RecentFrames ──────────────────────────────────────────────────────────────

// RecentFrames fetches n recent NEXRAD frames via IEM radmap at 5-minute
// intervals ending at the current time (truncated to 5 minutes).
func (p *RadarProvider) RecentFrames(ctx context.Context, loc location.Location, opts radar.Options, n int, c *cache.Cache) ([]*radar.Frame, error) {
	if _, ok := iemProducts[opts.Product]; !ok {
		return nil, fmt.Errorf("nws radar: unsupported product %q for loop", opts.Product)
	}

	bb := boundingBox(loc.Lat, loc.Lon, opts.RadiusKM)
	now := time.Now().UTC().Truncate(radarFrameInterval)

	type result struct {
		frame *radar.Frame
		err   error
	}
	results := make([]result, n)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // max 3 concurrent fetches
	for i := 0; i < n; i++ {
		ts := now.Add(-time.Duration(n-1-i) * radarFrameInterval)
		wg.Add(1)
		go func(idx int, ts time.Time) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			f, err := p.fetchIEMFrame(ctx, loc, bb, opts, ts, c)
			results[idx] = result{f, err}
		}(i, ts)
	}
	wg.Wait()

	var frames []*radar.Frame
	for _, r := range results {
		if r.frame != nil {
			frames = append(frames, r.frame)
		}
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("nws radar: no frames available for loop")
	}
	return frames, nil
}

// ── WMS fetch (current frame) ─────────────────────────────────────────────────

func (p *RadarProvider) fetchWMSFrame(ctx context.Context, layer string, bb radarBBox, timeStr string) (image.Image, time.Time, error) {
	params := url.Values{
		"SERVICE":     {"WMS"},
		"VERSION":     {"1.3.0"},
		"REQUEST":     {"GetMap"},
		"FORMAT":      {"image/png"},
		"TRANSPARENT": {"TRUE"},
		"LAYERS":      {layer},
		"CRS":         {"EPSG:4326"},
		"STYLES":      {""},
		"WIDTH":       {"1600"},
		"HEIGHT":      {"1600"},
		// WMS 1.3.0 + EPSG:4326: axis order is lat,lon (south,west,north,east).
		"BBOX": {fmt.Sprintf("%.6f,%.6f,%.6f,%.6f",
			bb.MinLat, bb.MinLon, bb.MaxLat, bb.MaxLon)},
	}
	if timeStr != "" {
		params.Set("TIME", timeStr)
	}

	img, validTime, err := p.fetchImageURL(ctx, p.wmsBase+"?"+params.Encode())
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("nws radar WMS: %w", err)
	}
	if validTime.IsZero() {
		if timeStr != "" {
			validTime, _ = time.Parse(time.RFC3339, timeStr)
		}
		if validTime.IsZero() {
			validTime = time.Now().UTC().Truncate(radarFrameInterval)
		}
	}
	return img, validTime, nil
}

// ── IEM fetch (historical loop frames) ───────────────────────────────────────

func (p *RadarProvider) fetchIEMFrame(ctx context.Context, loc location.Location, bb radarBBox, opts radar.Options, ts time.Time, c *cache.Cache) (*radar.Frame, error) {
	key := fmt.Sprintf("nws:radar:iem:v3:%s:%.4f,%.4f:%.0f:%s",
		opts.Product, (bb.MinLat+bb.MaxLat)/2, (bb.MinLon+bb.MaxLon)/2,
		opts.RadiusKM, ts.Format(time.RFC3339))

	// L1: in-process
	if f := p.imgGet(key); f != nil {
		return f, nil
	}
	// L2: disk (historical frames don't change — 24h TTL)
	if f, ok := p.diskGet(c, key); ok {
		p.imgSet(key, f, 0) // no L1 expiry for historical frames
		return f, nil
	}

	params := url.Values{}
	p.addRadarLayers(params, loc, opts)
	params.Set("width", "1600")
	params.Set("height", "1600")
	params.Set("bbox", fmt.Sprintf("%.6f,%.6f,%.6f,%.6f", bb.MinLon, bb.MinLat, bb.MaxLon, bb.MaxLat))
	params.Set("fmt", "png")
	params.Set("ts", ts.UTC().Format("200601021504"))

	img, _, err := p.fetchImageURL(ctx, p.iemBase+"?"+params.Encode())
	if err != nil {
		return nil, fmt.Errorf("iem radar: %w", err)
	}

	// For WMS composite products (echo tops), overlay radar data from NWS WMS.
	if isWMSCompositeProduct(opts.Product) {
		if layer, ok := nwsWMSLayers[opts.Product]; ok {
			radarImg, _, wmsErr := p.fetchWMSFrame(ctx, layer, bb, ts.Format(time.RFC3339))
			if wmsErr == nil {
				img = compositeOver(img, radarImg)
			}
		}
	}

	f := &radar.Frame{
		Img: img, ValidTime: ts, Product: opts.Product,
		BBox: &radar.BBox{MinLat: bb.MinLat, MinLon: bb.MinLon, MaxLat: bb.MaxLat, MaxLon: bb.MaxLon},
	}
	p.diskSet(c, key, f, historicalFrameTTL)
	p.imgSet(key, f, 0) // no L1 expiry for historical frames
	return f, nil
}

// ── HTTP image fetch ──────────────────────────────────────────────────────────

func (p *RadarProvider) fetchImageURL(ctx context.Context, rawURL string) (image.Image, time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, time.Time{}, err
	}
	req.Header.Set("User-Agent", "wx-cli/1.0 (github.com/mwirges/wx)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, time.Time{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("decode image: %w", err)
	}

	var validTime time.Time
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		validTime, _ = http.ParseTime(lm)
	}
	return img, validTime, nil
}

// ── Station lookup ────────────────────────────────────────────────────────────

// nwsRadarStationResponse is a partial decode of the NWS /radar/stations/{id}
// GeoJSON feature response. Coordinates are [lon, lat, elevation].
type nwsRadarStationResponse struct {
	Properties struct {
		StationIdentifier string `json:"stationIdentifier"`
		Name              string `json:"name"`
	} `json:"properties"`
	Geometry struct {
		Coordinates [3]float64 `json:"coordinates"` // [lon, lat, elev]
	} `json:"geometry"`
}

// LookupStation fetches metadata for the given NEXRAD station ID from the NWS
// API and returns a *radar.Station. Results are cached for 24 hours.
func (p *RadarProvider) LookupStation(ctx context.Context, stationID string, c *cache.Cache) (*radar.Station, error) {
	id := strings.ToUpper(strings.TrimSpace(stationID))
	if id == "" {
		return nil, fmt.Errorf("nws radar: empty station ID")
	}
	key := "nws:radar:station:" + id

	var cached radar.Station
	if c.Get(key, &cached) {
		return &cached, nil
	}

	rawURL := p.nwsAPIBase + "/radar/stations/" + id
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "wx-cli/1.0 (github.com/mwirges/wx)")
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nws radar station %s: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("nws radar: station %q not found", id)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nws radar station %s: HTTP %d", id, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("nws radar station %s: read body: %w", id, err)
	}

	var feature nwsRadarStationResponse
	if err := json.Unmarshal(body, &feature); err != nil {
		return nil, fmt.Errorf("nws radar station %s: decode: %w", id, err)
	}

	st := &radar.Station{
		ID:   id,
		Name: feature.Properties.Name,
		Lon:  feature.Geometry.Coordinates[0],
		Lat:  feature.Geometry.Coordinates[1],
	}
	if st.Name == "" {
		st.Name = id
	}

	_ = c.Set(key, *st, 24*time.Hour)
	return st, nil
}
