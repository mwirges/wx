package sounding

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/models"
)

const (
	defaultSPCBaseURL       = "https://www.spc.noaa.gov/exper/soundings"
	defaultOpenMeteoBaseURL = "https://api.open-meteo.com/v1/forecast"
	cacheTTL                = 45 * time.Minute
)

var defaultClient = NewClient("", "")

// Client retrieves and parses atmospheric sounding data.
type Client struct {
	spcBaseURL       string
	openMeteoBaseURL string
	httpClient       *http.Client
}

// NewClient creates a new sounding client.
func NewClient(spcBase, openMeteoBase string) *Client {
	if spcBase == "" {
		spcBase = defaultSPCBaseURL
	}
	if openMeteoBase == "" {
		openMeteoBase = defaultOpenMeteoBaseURL
	}
	return &Client{
		spcBaseURL:       strings.TrimRight(spcBase, "/"),
		openMeteoBaseURL: strings.TrimRight(openMeteoBase, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Fetch retrieves sounding telemetry for a given location or forced station.
func Fetch(ctx context.Context, lat, lon float64, locName string, forceStation string, c *cache.Cache) (*models.SoundingReport, error) {
	return defaultClient.Fetch(ctx, lat, lon, locName, forceStation, c)
}

// Fetch retrieves sounding telemetry for a given location or forced station.
func (cl *Client) Fetch(ctx context.Context, lat, lon float64, locName string, forceStation string, c *cache.Cache) (*models.SoundingReport, error) {
	// 1. Determine target station if US or forced
	var station Station
	var distKm float64
	var isForced bool

	if forceStation != "" {
		stnID := strings.ToUpper(strings.TrimSpace(forceStation))
		if s, ok := FindStationByID(stnID); ok {
			station = s
			distKm = haversineKm(lat, lon, s.Lat, s.Lon)
			isForced = true
		} else {
			station = Station{ID: stnID, Name: stnID + " Upper-Air Site", State: "US"}
			isForced = true
		}
	} else {
		station, distKm = FindClosestStation(lat, lon)
	}

	cacheKey := fmt.Sprintf("sounding:%.4f,%.4f:%s", lat, lon, station.ID)
	if isForced {
		cacheKey = fmt.Sprintf("sounding:stn:%s", station.ID)
	}

	var cached models.SoundingReport
	if c != nil && c.Get(cacheKey, &cached) {
		return &cached, nil
	}

	// If within 800km of a US RAOB site or forced, attempt NOAA SPC Upper-Air RAOB
	if isForced || distKm <= 800.0 {
		report, err := cl.fetchSPCSounding(ctx, station, lat, lon, locName, distKm)
		if err == nil && report != nil {
			if c != nil {
				_ = c.Set(cacheKey, report, cacheTTL)
			}
			return report, nil
		}
	}

	// Fallback to high-resolution model sounding via Open-Meteo
	report, err := cl.fetchOpenMeteoSounding(ctx, lat, lon, locName)
	if err != nil {
		return nil, fmt.Errorf("sounding: %w", err)
	}

	if c != nil {
		_ = c.Set(cacheKey, report, cacheTTL)
	}
	return report, nil
}

// ── NOAA SPC Upper-Air Sounding (NSHARP) ───────────────────────────────────────

var runCycleRe = regexp.MustCompile(`/exper/soundings/([0-9]{8}_OBS)/`)

func (cl *Client) fetchSPCSounding(ctx context.Context, stn Station, lat, lon float64, locName string, distKm float64) (*models.SoundingReport, error) {
	// 1. Discover recent observation cycles
	cycles, err := cl.discoverSPCCycles(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover spc cycles: %w", err)
	}

	var lastErr error
	for _, cycle := range cycles {
		url := fmt.Sprintf("%s/%s/%s.txt", cl.spcBaseURL, cycle, stn.ID)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "wx/1.0 (US Weather CLI; https://github.com/mwirges/wx)")

		resp, err := cl.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("http status %d from %s", resp.StatusCode, url)
			continue
		}

		rawText, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		report, err := parseNSHARPSounding(string(rawText), stn, cycle)
		if err != nil {
			lastErr = err
			continue
		}

		if locName != "" {
			report.Location = locName
		} else {
			report.Location = fmt.Sprintf("%s, %s", stn.Name, stn.State)
		}
		report.DistanceKM = distKm
		report.DistanceMiles = distKm * 0.621371
		report.SkewTImageURL = fmt.Sprintf("%s/%s/%s.gif", cl.spcBaseURL, cycle, stn.ID)

		summarizeConvectiveEnvironment(&report.Indices)
		return report, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no sounding found for %s across recent cycles", stn.ID)
}

func (cl *Client) discoverSPCCycles(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cl.spcBaseURL+"/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "wx/1.0")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	matches := runCycleRe.FindAllStringSubmatch(string(bodyBytes), -1)
	var cycles []string
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			seen[m[1]] = true
			cycles = append(cycles, m[1])
		}
	}
	if len(cycles) > 0 {
		return cycles, nil
	}

	// Fallback to current UTC cycle calculation (00Z or 12Z today)
	now := time.Now().UTC()
	hour := now.Hour()
	targetHour := 0
	if hour >= 13 {
		targetHour = 12
	}
	cycleStr := fmt.Sprintf("%02d%02d%02d%02d_OBS", now.Year()%100, now.Month(), now.Day(), targetHour)
	return []string{cycleStr}, nil
}

// parseNSHARPSounding parses the NOAA SPC NSHARP text output.
func parseNSHARPSounding(raw string, stn Station, cycle string) (*models.SoundingReport, error) {
	scanner := bufio.NewScanner(strings.NewReader(raw))

	report := &models.SoundingReport{
		StationID:   stn.ID,
		StationName: fmt.Sprintf("%s (%s)", stn.Name, stn.State),
		Provider:    "NOAA SPC Upper-Air RAOB (NSHARP)",
		Timestamp:   parseCycleTimestamp(cycle),
	}

	inRawTable := false
	var levels []models.SoundingLevel

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if line == "%RAW%" {
			inRawTable = true
			continue
		}
		if line == "%END%" {
			inRawTable = false
			continue
		}

		if inRawTable {
			parts := strings.Split(line, ",")
			if len(parts) >= 6 {
				p := parseVal(parts[0])
				h := parseVal(parts[1])
				t := parseVal(parts[2])
				td := parseVal(parts[3])
				wdir := parseVal(parts[4])
				wspd := parseVal(parts[5])

				if p != nil && *p > 0 {
					lvl := models.SoundingLevel{
						PressureHPA: *p,
					}
					if h != nil && *h > -9000 {
						lvl.HeightM = h
						ft := *h * 3.28084
						lvl.HeightFT = &ft
					}
					if t != nil && *t > -9000 {
						lvl.TempC = t
						f := *t*9.0/5.0 + 32.0
						lvl.TempF = &f
					}
					if td != nil && *td > -9000 {
						lvl.DewPointC = td
						f := *td*9.0/5.0 + 32.0
						lvl.DewPointF = &f
					}
					if wdir != nil && *wdir >= 0 && *wdir <= 360 {
						lvl.WindDirDeg = wdir
					}
					if wspd != nil && *wspd >= 0 {
						lvl.WindSpeedKT = wspd
						kph := *wspd * 1.852
						mph := *wspd * 1.15078
						lvl.WindSpeedKPH = &kph
						lvl.WindSpeedMPH = &mph
					}
					if lvl.TempC != nil && lvl.DewPointC != nil {
						rh := calculateRelativeHumidity(*lvl.TempC, *lvl.DewPointC)
						lvl.RHPct = &rh
					}
					levels = append(levels, lvl)
				}
			}
			continue
		}

		// Parse Convective Indices
		parseConvectiveLine(line, &report.Indices)
	}

	report.Levels = filterStandardSoundingLevels(levels)
	return report, nil
}

func parseConvectiveLine(line string, idx *models.ConvectiveIndices) {
	upper := strings.ToUpper(line)

	// Parcel CAPE / CIN / LI
	if strings.HasPrefix(upper, "SBCAPE:") {
		idx.SBCAPE = extractFloatValue(line)
	} else if strings.HasPrefix(upper, "SBCINH:") {
		idx.SBCIN = extractFloatValue(line)
	} else if strings.HasPrefix(upper, "SBLI:") {
		idx.SBLI = extractFloatValue(line)
	} else if strings.HasPrefix(upper, "MLCAPE:") {
		idx.MLCAPE = extractFloatValue(line)
	} else if strings.HasPrefix(upper, "MLCINH:") {
		idx.MLCIN = extractFloatValue(line)
	} else if strings.HasPrefix(upper, "MLLI:") {
		idx.MLLI = extractFloatValue(line)
	} else if strings.HasPrefix(upper, "MUCAPE:") {
		idx.MUCAPE = extractFloatValue(line)
	} else if strings.HasPrefix(upper, "MUCINH:") {
		idx.MUCIN = extractFloatValue(line)
	} else if strings.HasPrefix(upper, "MULI:") {
		idx.MULI = extractFloatValue(line)
	} else if strings.HasPrefix(upper, "PRECIP WATER:") {
		if in := extractFloatValue(line); in != nil {
			idx.PWATIn = in
			mm := *in * 25.4
			idx.PWATMm = &mm
		}
	} else if strings.HasPrefix(upper, "MELTING LEVEL:") {
		if ft := extractFloatValue(line); ft != nil && *ft > 0 {
			idx.FreezingLevelFT = ft
			m := *ft / 3.28084
			idx.FreezingLevelM = &m
		}
	} else if strings.HasPrefix(upper, "DCAPE:") {
		idx.DCAPE = extractFloatValue(line)
	} else if strings.Contains(upper, "0-1 KM BWD") {
		idx.BulkShear01KT = extractFloatValue(line)
	} else if strings.Contains(upper, "0-3 KM BWD") {
		idx.BulkShear03KT = extractFloatValue(line)
	} else if strings.Contains(upper, "0-6 KM BWD") {
		idx.BulkShear06KT = extractFloatValue(line)
	} else if strings.Contains(upper, "0-1 KM SRH") {
		idx.SRH01 = extractFloatValue(line)
	} else if strings.Contains(upper, "0-3 KM SRH") {
		idx.SRH03 = extractFloatValue(line)
	} else if strings.Contains(upper, "EFFECTIVE-LAYER STP") || strings.Contains(upper, "FIXED-LAYER STP") {
		if idx.STP == nil {
			idx.STP = extractFloatValue(line)
		}
	} else if strings.Contains(upper, "EFFECTIVE-LAYER SCP") {
		idx.SCP = extractFloatValue(line)
	} else if strings.HasPrefix(upper, "SHIP") {
		idx.SHIP = extractFloatValue(line)
	} else if strings.Contains(upper, "700-500MB") && strings.Contains(upper, "C/KM") {
		idx.LapseRate700_500 = extractSecondRate(line)
	} else if strings.Contains(upper, "850-500MB") && strings.Contains(upper, "C/KM") {
		idx.LapseRate850_500 = extractSecondRate(line)
	}
}

func extractFloatValue(line string) *float64 {
	// If line has colon, only search after the colon
	if idx := strings.Index(line, ":"); idx != -1 {
		line = line[idx+1:]
	}

	tokens := strings.Fields(line)
	for i := len(tokens) - 1; i >= 0; i-- {
		tok := strings.TrimRight(tokens[i], "%,;")
		if val, err := strconv.ParseFloat(tok, 64); err == nil {
			if val <= -9000 {
				return nil
			}
			return &val
		}
	}
	return nil
}

func extractSecondRate(line string) *float64 {
	// Format: "700-500mb   15 C      5.7 C/km" -> extracts 5.7
	re := regexp.MustCompile(`[-+]?[0-9]*\.?[0-9]+`)
	matches := re.FindAllString(line, -1)
	if len(matches) >= 2 {
		val, err := strconv.ParseFloat(matches[len(matches)-1], 64)
		if err == nil && val > -9000 {
			return &val
		}
	}
	return nil
}

func parseVal(s string) *float64 {
	s = strings.TrimSpace(s)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= -9000 {
		return nil
	}
	return &f
}

func calculateRelativeHumidity(tC, tdC float64) float64 {
	// Magnus formula approximation
	es := 6.112 * math.Exp((17.67*tC)/(tC+243.5))
	e := 6.112 * math.Exp((17.67*tdC)/(tdC+243.5))
	rh := (e / es) * 100.0
	if rh > 100.0 {
		rh = 100.0
	} else if rh < 0.0 {
		rh = 0.0
	}
	return math.Round(rh)
}

func parseCycleTimestamp(cycle string) time.Time {
	// "26092800_OBS" -> 2026-09-28 00:00 UTC
	if len(cycle) >= 8 {
		t, err := time.Parse("06010215", cycle[:8])
		if err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

// filterStandardSoundingLevels trims raw hundreds of radiosonde data points into key mandatory/meteorological pressure levels.
func filterStandardSoundingLevels(raw []models.SoundingLevel) []models.SoundingLevel {
	if len(raw) == 0 {
		return nil
	}

	targets := []float64{1000, 925, 850, 700, 500, 400, 300, 250, 200}
	var res []models.SoundingLevel

	// Always add surface level
	if len(raw) > 0 {
		res = append(res, raw[0])
	}

	for _, target := range targets {
		var closest *models.SoundingLevel
		minDiff := 15.0 // within 15 mb

		for i := range raw {
			diff := math.Abs(raw[i].PressureHPA - target)
			if diff < minDiff {
				minDiff = diff
				closest = &raw[i]
			}
		}

		if closest != nil {
			// Avoid duplicate surface level
			if len(res) > 0 && math.Abs(res[len(res)-1].PressureHPA-closest.PressureHPA) < 5.0 {
				continue
			}
			res = append(res, *closest)
		}
	}

	return res
}

// ── Open-Meteo High-Resolution Model Sounding ─────────────────────────────────

type openMeteoSoundingResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Elevation float64 `json:"elevation"`
	Current   struct {
		Time                 string   `json:"time"`
		CAPE                 *float64 `json:"cape"`
		ConvectiveInhibition *float64 `json:"convective_inhibition"`
		LiftedIndex          *float64 `json:"lifted_index"`
	} `json:"current"`
	Hourly struct {
		Time                             []string   `json:"time"`
		TotalColumnIntegratedWaterVapour []float64  `json:"total_column_integrated_water_vapour"`
		FreezingLevelHeight              []float64  `json:"freezing_level_height"`
		WindSpeed10m                     []float64  `json:"wind_speed_10m"`
		WindDirection10m                 []float64  `json:"wind_direction_10m"`
		WindSpeed925hPa                  []float64  `json:"wind_speed_925hPa"`
		WindDirection925hPa              []float64  `json:"wind_direction_925hPa"`
		WindSpeed850hPa                  []float64  `json:"wind_speed_850hPa"`
		WindDirection850hPa              []float64  `json:"wind_direction_850hPa"`
		WindSpeed700hPa                  []float64  `json:"wind_speed_700hPa"`
		WindDirection700hPa              []float64  `json:"wind_direction_700hPa"`
		WindSpeed500hPa                  []float64  `json:"wind_speed_500hPa"`
		WindDirection500hPa              []float64  `json:"wind_direction_500hPa"`
		WindSpeed300hPa                  []float64  `json:"wind_speed_300hPa"`
		WindDirection300hPa              []float64  `json:"wind_direction_300hPa"`
		WindSpeed250hPa                  []float64  `json:"wind_speed_250hPa"`
		Temperature1000hPa               []*float64 `json:"temperature_1000hPa"`
		DewPoint1000hPa                  []*float64 `json:"dew_point_1000hPa"`
		Temperature925hPa                []*float64 `json:"temperature_925hPa"`
		DewPoint925hPa                   []*float64 `json:"dew_point_925hPa"`
		Temperature850hPa                []*float64 `json:"temperature_850hPa"`
		DewPoint850hPa                   []*float64 `json:"dew_point_850hPa"`
		Temperature700hPa                []*float64 `json:"temperature_700hPa"`
		DewPoint700hPa                   []*float64 `json:"dew_point_700hPa"`
		Temperature500hPa                []*float64 `json:"temperature_500hPa"`
		DewPoint500hPa                   []*float64 `json:"dew_point_500hPa"`
		GeopotentialHeight925hPa         []*float64 `json:"geopotential_height_925hPa"`
		GeopotentialHeight850hPa         []*float64 `json:"geopotential_height_850hPa"`
		GeopotentialHeight700hPa         []*float64 `json:"geopotential_height_700hPa"`
		GeopotentialHeight500hPa         []*float64 `json:"geopotential_height_500hPa"`
	} `json:"hourly"`
}

func (cl *Client) fetchOpenMeteoSounding(ctx context.Context, lat, lon float64, locName string) (*models.SoundingReport, error) {
	params := "current=cape,lifted_index,convective_inhibition&hourly=" +
		"total_column_integrated_water_vapour,freezing_level_height," +
		"wind_speed_10m,wind_direction_10m,wind_speed_925hPa,wind_direction_925hPa," +
		"wind_speed_850hPa,wind_direction_850hPa,wind_speed_700hPa,wind_direction_700hPa," +
		"wind_speed_500hPa,wind_direction_500hPa,wind_speed_300hPa,wind_direction_300hPa,wind_speed_250hPa," +
		"temperature_1000hPa,dew_point_1000hPa,temperature_925hPa,dew_point_925hPa," +
		"temperature_850hPa,dew_point_850hPa,temperature_700hPa,dew_point_700hPa," +
		"temperature_500hPa,dew_point_500hPa,geopotential_height_925hPa,geopotential_height_850hPa," +
		"geopotential_height_700hPa,geopotential_height_500hPa&forecast_days=1"

	url := fmt.Sprintf("%s?latitude=%.4f&longitude=%.4f&%s", cl.openMeteoBaseURL, lat, lon, params)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "wx/1.0 (US Weather CLI; https://github.com/mwirges/wx)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo sounding: http status %d", resp.StatusCode)
	}

	var data openMeteoSoundingResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode open-meteo sounding: %w", err)
	}

	report := &models.SoundingReport{
		Timestamp:   time.Now().UTC(),
		Location:    locName,
		StationID:   fmt.Sprintf("MODEL(%.2f,%.2f)", lat, lon),
		StationName: "Global Numerical Weather Prediction Model Sounding",
		Provider:    "Open-Meteo High-Resolution Atmospheric Profile",
	}

	// 1. Current Convective Indices
	report.Indices.SBCAPE = data.Current.CAPE
	report.Indices.MLCAPE = data.Current.CAPE
	report.Indices.SBCIN = data.Current.ConvectiveInhibition
	report.Indices.SBLI = data.Current.LiftedIndex

	// Extract current or hour 0 metrics
	h := data.Hourly
	if len(h.TotalColumnIntegratedWaterVapour) > 0 {
		kgm2 := h.TotalColumnIntegratedWaterVapour[0]
		mm := kgm2
		in := mm / 25.4
		report.Indices.PWATMm = &mm
		report.Indices.PWATIn = &in
	}
	if len(h.FreezingLevelHeight) > 0 {
		m := h.FreezingLevelHeight[0]
		ft := m * 3.28084
		report.Indices.FreezingLevelM = &m
		report.Indices.FreezingLevelFT = &ft
	}

	// Calculate 0-6km bulk shear from 10m to 500hPa (~5.5km)
	if len(h.WindSpeed10m) > 0 && len(h.WindSpeed500hPa) > 0 {
		s0 := h.WindSpeed10m[0] / 1.852 // to kt
		d0 := h.WindDirection10m[0]
		s6 := h.WindSpeed500hPa[0] / 1.852 // to kt
		d6 := h.WindDirection500hPa[0]

		shear06 := calculateVectorShear(s0, d0, s6, d6)
		report.Indices.BulkShear06KT = &shear06
	}

	// Calculate 0-1km bulk shear from 10m to 925hPa (~800m)
	if len(h.WindSpeed10m) > 0 && len(h.WindSpeed925hPa) > 0 {
		s0 := h.WindSpeed10m[0] / 1.852
		d0 := h.WindDirection10m[0]
		s1 := h.WindSpeed925hPa[0] / 1.852
		d1 := h.WindDirection925hPa[0]

		shear01 := calculateVectorShear(s0, d0, s1, d1)
		report.Indices.BulkShear01KT = &shear01
	}

	// Build Levels Table
	var levels []models.SoundingLevel
	addOMLevel := func(p float64, tArr, tdArr, hArr []*float64, spdArr, dirArr []float64) {
		lvl := models.SoundingLevel{PressureHPA: p}
		if len(tArr) > 0 && tArr[0] != nil {
			lvl.TempC = tArr[0]
			f := *tArr[0]*9.0/5.0 + 32.0
			lvl.TempF = &f
		}
		if len(tdArr) > 0 && tdArr[0] != nil {
			lvl.DewPointC = tdArr[0]
			f := *tdArr[0]*9.0/5.0 + 32.0
			lvl.DewPointF = &f
		}
		if len(hArr) > 0 && hArr[0] != nil {
			lvl.HeightM = hArr[0]
			ft := *hArr[0] * 3.28084
			lvl.HeightFT = &ft
		}
		if len(spdArr) > 0 {
			kph := spdArr[0]
			kt := kph / 1.852
			mph := kph * 0.621371
			lvl.WindSpeedKPH = &kph
			lvl.WindSpeedKT = &kt
			lvl.WindSpeedMPH = &mph
		}
		if len(dirArr) > 0 {
			dir := dirArr[0]
			lvl.WindDirDeg = &dir
		}
		if lvl.TempC != nil && lvl.DewPointC != nil {
			rh := calculateRelativeHumidity(*lvl.TempC, *lvl.DewPointC)
			lvl.RHPct = &rh
		}
		levels = append(levels, lvl)
	}

	addOMLevel(1000, h.Temperature1000hPa, h.DewPoint1000hPa, nil, h.WindSpeed10m, h.WindDirection10m)
	addOMLevel(925, h.Temperature925hPa, h.DewPoint925hPa, h.GeopotentialHeight925hPa, h.WindSpeed925hPa, h.WindDirection925hPa)
	addOMLevel(850, h.Temperature850hPa, h.DewPoint850hPa, h.GeopotentialHeight850hPa, h.WindSpeed850hPa, h.WindDirection850hPa)
	addOMLevel(700, h.Temperature700hPa, h.DewPoint700hPa, h.GeopotentialHeight700hPa, h.WindSpeed700hPa, h.WindDirection700hPa)
	addOMLevel(500, h.Temperature500hPa, h.DewPoint500hPa, h.GeopotentialHeight500hPa, h.WindSpeed500hPa, h.WindDirection500hPa)

	report.Levels = levels
	summarizeConvectiveEnvironment(&report.Indices)
	return report, nil
}

func calculateVectorShear(spd0, dir0, spd1, dir1 float64) float64 {
	rad0 := dir0 * math.Pi / 180.0
	rad1 := dir1 * math.Pi / 180.0

	u0 := -spd0 * math.Sin(rad0)
	v0 := -spd0 * math.Cos(rad0)

	u1 := -spd1 * math.Sin(rad1)
	v1 := -spd1 * math.Cos(rad1)

	du := u1 - u0
	dv := v1 - v0
	return math.Round(math.Sqrt(du*du+dv*dv)*10) / 10
}

func summarizeConvectiveEnvironment(idx *models.ConvectiveIndices) {
	capeVal := 0.0
	if idx.SBCAPE != nil {
		capeVal = *idx.SBCAPE
	} else if idx.MLCAPE != nil {
		capeVal = *idx.MLCAPE
	}

	if capeVal >= 3500 {
		idx.InstabilitySummary = "Extreme Instability"
	} else if capeVal >= 2000 {
		idx.InstabilitySummary = "Strong Instability"
	} else if capeVal >= 1000 {
		idx.InstabilitySummary = "Moderate Instability"
	} else if capeVal >= 300 {
		idx.InstabilitySummary = "Weak Instability"
	} else {
		idx.InstabilitySummary = "Stable / Marginal Instability"
	}

	shearVal := 0.0
	if idx.BulkShear06KT != nil {
		shearVal = *idx.BulkShear06KT
	}

	if shearVal >= 50 {
		idx.ShearSummary = "Intense Deep-Layer Shear (Fast Supercells / Bow Echoes)"
	} else if shearVal >= 35 {
		idx.ShearSummary = "Strong Deep-Layer Shear (Supercells Favored)"
	} else if shearVal >= 20 {
		idx.ShearSummary = "Moderate Shear (Multicells / Clusters Favored)"
	} else {
		idx.ShearSummary = "Weak Shear (Pulse / Unorganized Convection)"
	}

	if capeVal >= 1500 && shearVal >= 35 {
		idx.ConvectiveRisk = "High Severe Convective Potential (Supercells / Tornado / Large Hail)"
	} else if capeVal >= 800 && shearVal >= 25 {
		idx.ConvectiveRisk = "Elevated Storm Potential (Multicell Clusters / Damaging Winds)"
	} else if capeVal >= 400 {
		idx.ConvectiveRisk = "Isolated Pulse Thunderstorm Potential"
	} else {
		idx.ConvectiveRisk = "Minimal Severe Thunderstorm Risk"
	}
}
