package climate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/models"
)

const (
	defaultACISBaseURL = "https://data.rcc-acis.org"
	cacheTTL           = 6 * time.Hour
)

type stnMetaResponse struct {
	Meta []struct {
		Name  string    `json:"name"`
		State string    `json:"state"`
		Elev  *float64  `json:"elev"`
		LL             []float64  `json:"ll"` // [lon, lat]
		Sids           []string   `json:"sids"`
		UID            int        `json:"uid"`
		ValidDateRange [][]string `json:"valid_daterange"`
	} `json:"meta"`
}

type stnDataResponse struct {
	Meta struct {
		Name  string    `json:"name"`
		State string    `json:"state"`
		Elev  *float64  `json:"elev"`
		LL    []float64 `json:"ll"`
		Sids  []string  `json:"sids"`
		UID   int       `json:"uid"`
	} `json:"meta"`
	Data [][]any `json:"data"`
}

type gridDataResponse struct {
	Data [][]any `json:"data"`
}

// Client fetches NOAA Climate Normals and historical daily extremes.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient returns a new climate client.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultACISBaseURL
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

var defaultClient = NewClient("")

// Fetch returns the ClimateReport for a location and target date.
func Fetch(ctx context.Context, lat, lon float64, locName string, obsCurrent *models.CurrentConditions, targetDate time.Time, c *cache.Cache) (*models.ClimateReport, error) {
	return defaultClient.Fetch(ctx, lat, lon, locName, obsCurrent, targetDate, c)
}

// Fetch returns the ClimateReport for a location and target date.
func (cl *Client) Fetch(ctx context.Context, lat, lon float64, locName string, obsCurrent *models.CurrentConditions, targetDate time.Time, c *cache.Cache) (*models.ClimateReport, error) {
	if targetDate.IsZero() {
		targetDate = time.Now()
	}

	dateKey := targetDate.Format("2006-01-02")
	cacheKey := fmt.Sprintf("climate:%.4f,%.4f:%s", lat, lon, dateKey)
	var cached models.ClimateReport
	if c != nil && c.Get(cacheKey, &cached) {
		// Update departure if live observation is provided
		if obsCurrent != nil {
			cached.Departure = ComputeDeparture(cached.TodayNormals, obsCurrent)
		}
		return &cached, nil
	}

	// 1. Resolve closest primary climate station in ACIS
	sid, stnName, stnElevFt, stnLat, stnLon, err := cl.findClosestStation(ctx, lat, lon)
	useGridFallback := err != nil || sid == ""

	report := &models.ClimateReport{
		Date:          targetDate,
		Location:      locName,
		Latitude:      lat,
		Longitude:     lon,
		NormalsPeriod: "1991–2020",
	}

	if !useGridFallback {
		report.StationID = sid
		report.StationName = stnName
		report.ElevationFt = stnElevFt
		report.ElevationM = stnElevFt * 0.3048
		report.Latitude = stnLat
		report.Longitude = stnLon

		// Fetch Station Normals
		if err := cl.fetchStationNormals(ctx, sid, targetDate, report); err != nil || (report.TodayNormals.NormalHighF == 0 && report.TodayNormals.NormalLowF == 0) {
			_ = cl.fetchGridNormals(ctx, lat, lon, targetDate, report)
		}
		// Fetch Station Historical Records across all years
		_ = cl.fetchStationRecords(ctx, sid, targetDate, report)
	}

	if useGridFallback {
		// GridData fallback
		report.StationID = fmt.Sprintf("GRID(%.2f,%.2f)", lat, lon)
		report.StationName = fmt.Sprintf("NOAA National Grid (1991-2020 Normals)")
		if err := cl.fetchGridNormals(ctx, lat, lon, targetDate, report); err != nil {
			return nil, fmt.Errorf("climate: fetch grid normals: %w", err)
		}
	}

	if obsCurrent != nil {
		report.Departure = ComputeDeparture(report.TodayNormals, obsCurrent)
	}

	if c != nil {
		_ = c.Set(cacheKey, report, cacheTTL)
	}

	return report, nil
}

func (cl *Client) findClosestStation(ctx context.Context, lat, lon float64) (sid, name string, elevFt, stnLat, stnLon float64, err error) {
	delta := 0.65
	bboxStr := fmt.Sprintf("%.4f,%.4f,%.4f,%.4f", lon-delta, lat-delta, lon+delta, lat+delta)

	payload := map[string]any{
		"bbox":  bboxStr,
		"meta":  "name,state,sids,ll,elev,uid,valid_daterange",
		"elems": "maxt",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", 0, 0, 0, err
	}

	url := fmt.Sprintf("%s/StnMeta", cl.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", "", 0, 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "wx-cli/1.0 (climate normals)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return "", "", 0, 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, 0, 0, fmt.Errorf("stnmeta status %d", resp.StatusCode)
	}

	var metaResp stnMetaResponse
	if err := json.NewDecoder(resp.Body).Decode(&metaResp); err != nil {
		return "", "", 0, 0, 0, err
	}

	if len(metaResp.Meta) == 0 {
		return "", "", 0, 0, 0, fmt.Errorf("no stations found in bbox")
	}

	type candidate struct {
		sid      string
		name     string
		elev     float64
		lat      float64
		lon      float64
		dist     float64
		priority int // 1 = Airport/ICAO/WBAN, 2 = COOP, 3 = other
	}

	var candidates []candidate
	for _, m := range metaResp.Meta {
		if len(m.LL) < 2 || len(m.Sids) == 0 {
			continue
		}
		sLon, sLat := m.LL[0], m.LL[1]
		dist := haversineKm(lat, lon, sLat, sLon)

		// Choose primary SID
		chosenSid := m.Sids[0]
		priority := 3
		nameUpper := strings.ToUpper(m.Name)
		isAirport := strings.Contains(nameUpper, "AP") || strings.Contains(nameUpper, "AIRPORT") || strings.Contains(nameUpper, "FIELD")

		for _, s := range m.Sids {
			parts := strings.Fields(s)
			if len(parts) >= 2 {
				sidType := parts[1]
				if sidType == "1" || sidType == "5" || sidType == "3" { // WBAN, ICAO, FAA
					chosenSid = s
					priority = 1
					break
				} else if sidType == "2" || sidType == "6" { // COOP / GHCN
					if priority > 2 {
						chosenSid = s
						priority = 2
					}
				}
			}
		}

		if isAirport && priority > 1 {
			priority = 1
		}

		// Penalize inactive/historic stations that closed before 2015
		isActive := false
		if len(m.ValidDateRange) > 0 && len(m.ValidDateRange[0]) > 1 {
			endYearStr := m.ValidDateRange[0][1]
			if len(endYearStr) >= 4 {
				if endYear, err := strconv.Atoi(endYearStr[:4]); err == nil && endYear >= 2015 {
					isActive = true
				}
			}
		}
		if !isActive {
			priority += 10
		}

		elev := 0.0
		if m.Elev != nil {
			elev = *m.Elev
		}

		candidates = append(candidates, candidate{
			sid:      chosenSid,
			name:     m.Name,
			elev:     elev,
			lat:      sLat,
			lon:      sLon,
			dist:     dist,
			priority: priority,
		})
	}

	if len(candidates) == 0 {
		return "", "", 0, 0, 0, fmt.Errorf("no valid candidates")
	}

	// Sort by priority (asc) then distance (asc)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].priority != candidates[j].priority {
			return candidates[i].priority < candidates[j].priority
		}
		return candidates[i].dist < candidates[j].dist
	})

	best := candidates[0]
	return best.sid, best.name, best.elev, best.lat, best.lon, nil
}

func (cl *Client) fetchStationNormals(ctx context.Context, sid string, targetDate time.Time, report *models.ClimateReport) error {
	year := targetDate.Year()
	month := int(targetDate.Month())
	monthStart := time.Date(year, targetDate.Month(), 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, -1)

	payload := map[string]any{
		"sid": sid,
		"elems": []map[string]any{
			{"name": "maxt", "normal": "1"},
			{"name": "mint", "normal": "1"},
			{"name": "pcpn", "normal": "1"},
		},
		"sdate": monthStart.Format("2006-01-02"),
		"edate": monthEnd.Format("2006-01-02"),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/StnData", cl.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "wx-cli/1.0 (climate normals)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stndata status %d", resp.StatusCode)
	}

	var dataResp stnDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&dataResp); err != nil {
		return err
	}

	if dataResp.Meta.Name != "" && report.StationName == "" {
		report.StationName = dataResp.Meta.Name
	}

	targetDateStr := targetDate.Format("2006-01-02")
	var monthHighSum, monthLowSum, monthPrecipSum float64
	var monthDaysCount int

	for _, row := range dataResp.Data {
		if len(row) < 4 {
			continue
		}
		dateStr, _ := row[0].(string)
		maxF, okMax := parseFloat(row[1])
		minF, okMin := parseFloat(row[2])
		pcpnIn, _ := parseFloat(row[3])

		if okMax && okMin {
			monthHighSum += maxF
			monthLowSum += minF
			monthPrecipSum += pcpnIn
			monthDaysCount++
		}

		if dateStr == targetDateStr && okMax && okMin {
			meanF := (maxF + minF) / 2.0
			report.TodayNormals = models.DailyNormals{
				Date:           targetDate,
				NormalHighF:    maxF,
				NormalHighC:    (maxF - 32.0) * 5.0 / 9.0,
				NormalLowF:     minF,
				NormalLowC:     (minF - 32.0) * 5.0 / 9.0,
				NormalMeanF:    meanF,
				NormalMeanC:    (meanF - 32.0) * 5.0 / 9.0,
				NormalPrecipIn: pcpnIn,
				NormalPrecipMM: pcpnIn * 25.4,
			}
		}
	}

	if monthDaysCount > 0 {
		avgHighF := monthHighSum / float64(monthDaysCount)
		avgLowF := monthLowSum / float64(monthDaysCount)
		report.MonthlyNormals = &models.MonthlyNormals{
			MonthName:           targetDate.Month().String(),
			NormalAvgHighF:      avgHighF,
			NormalAvgHighC:      (avgHighF - 32.0) * 5.0 / 9.0,
			NormalAvgLowF:       avgLowF,
			NormalAvgLowC:       (avgLowF - 32.0) * 5.0 / 9.0,
			NormalTotalPrecipIn: monthPrecipSum,
			NormalTotalPrecipMM: monthPrecipSum * 25.4,
		}
	}

	_ = month

	return nil
}

func (cl *Client) fetchStationRecords(ctx context.Context, sid string, targetDate time.Time, report *models.ClimateReport) error {
	mmdd := targetDate.Format("01-02")
	curYear := targetDate.Year()
	sdate := fmt.Sprintf("1890-%s", mmdd)
	edate := fmt.Sprintf("%d-%s", curYear, mmdd)

	payload := map[string]any{
		"sid": sid,
		"elems": []map[string]any{
			{"name": "maxt", "interval": []int{1, 0, 0}},
			{"name": "mint", "interval": []int{1, 0, 0}},
			{"name": "pcpn", "interval": []int{1, 0, 0}},
		},
		"sdate": sdate,
		"edate": edate,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/StnData", cl.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "wx-cli/1.0 (climate records)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stndata records status %d", resp.StatusCode)
	}

	var dataResp stnDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&dataResp); err != nil {
		return err
	}

	var recordHighF float64 = -999
	var highYears []int
	var recordLowF float64 = 999
	var lowYears []int

	var recordPrecipIn float64 = -1
	var precipYears []int

	var coldestHighF float64 = 999
	var coldHighYears []int
	var warmestLowF float64 = -999
	var warmLowYears []int

	firstYear := 9999
	lastYear := 0
	validYears := 0

	for _, row := range dataResp.Data {
		if len(row) < 4 {
			continue
		}
		dateStr, _ := row[0].(string)
		if len(dateStr) < 4 {
			continue
		}
		year, err := strconv.Atoi(dateStr[:4])
		if err != nil {
			continue
		}

		maxF, okMax := parseFloat(row[1])
		minF, okMin := parseFloat(row[2])
		pcpn, okPcpn := parseFloat(row[3])

		if okMax {
			validYears++
			if year < firstYear {
				firstYear = year
			}
			if year > lastYear {
				lastYear = year
			}

			if maxF > recordHighF {
				recordHighF = maxF
				highYears = []int{year}
			} else if math.Abs(maxF-recordHighF) < 0.05 {
				highYears = append(highYears, year)
			}

			if maxF < coldestHighF {
				coldestHighF = maxF
				coldHighYears = []int{year}
			} else if math.Abs(maxF-coldestHighF) < 0.05 {
				coldHighYears = append(coldHighYears, year)
			}
		}

		if okMin {
			if minF < recordLowF {
				recordLowF = minF
				lowYears = []int{year}
			} else if math.Abs(minF-recordLowF) < 0.05 {
				lowYears = append(lowYears, year)
			}

			if minF > warmestLowF {
				warmestLowF = minF
				warmLowYears = []int{year}
			} else if math.Abs(minF-warmestLowF) < 0.05 {
				warmLowYears = append(warmLowYears, year)
			}
		}

		if okPcpn {
			if pcpn > recordPrecipIn {
				recordPrecipIn = pcpn
				precipYears = []int{year}
			} else if math.Abs(pcpn-recordPrecipIn) < 0.005 {
				precipYears = append(precipYears, year)
			}
		}
	}

	if recordHighF > -900 {
		report.Records.RecordHigh = models.DailyRecord{
			ValueF: recordHighF,
			ValueC: (recordHighF - 32.0) * 5.0 / 9.0,
			Years:  highYears,
		}
	}
	if recordLowF < 900 {
		report.Records.RecordLow = models.DailyRecord{
			ValueF: recordLowF,
			ValueC: (recordLowF - 32.0) * 5.0 / 9.0,
			Years:  lowYears,
		}
	}
	if coldestHighF < 900 {
		report.Records.ColdestHigh = models.DailyRecord{
			ValueF: coldestHighF,
			ValueC: (coldestHighF - 32.0) * 5.0 / 9.0,
			Years:  coldHighYears,
		}
	}
	if warmestLowF > -900 {
		report.Records.WarmestLow = models.DailyRecord{
			ValueF: warmestLowF,
			ValueC: (warmestLowF - 32.0) * 5.0 / 9.0,
			Years:  warmLowYears,
		}
	}
	if recordPrecipIn >= 0 {
		report.Records.RecordPrecip = models.DailyRecord{
			ValueIn: recordPrecipIn,
			ValueMM: recordPrecipIn * 25.4,
			Years:   precipYears,
		}
	}

	if firstYear <= lastYear && firstYear != 9999 {
		report.Records.PeriodOfRecord = fmt.Sprintf("%d–%d", firstYear, lastYear)
		report.Records.TotalYearsSampled = validYears
	}

	return nil
}

func (cl *Client) fetchGridNormals(ctx context.Context, lat, lon float64, targetDate time.Time, report *models.ClimateReport) error {
	dateStr := targetDate.Format("2006-01-02")
	locStr := fmt.Sprintf("%.4f,%.4f", lon, lat)

	payload := map[string]any{
		"loc":  locStr,
		"grid": "1",
		"elems": []map[string]any{
			{"name": "maxt", "normal": "1"},
			{"name": "mint", "normal": "1"},
		},
		"sdate": dateStr,
		"edate": dateStr,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/GridData", cl.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "wx-cli/1.0 (grid normals)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("griddata status %d", resp.StatusCode)
	}

	var gridResp gridDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&gridResp); err != nil {
		return err
	}

	if len(gridResp.Data) > 0 && len(gridResp.Data[0]) >= 3 {
		row := gridResp.Data[0]
		maxF, _ := parseFloat(row[1])
		minF, _ := parseFloat(row[2])
		meanF := (maxF + minF) / 2.0

		report.TodayNormals = models.DailyNormals{
			Date:        targetDate,
			NormalHighF: maxF,
			NormalHighC: (maxF - 32.0) * 5.0 / 9.0,
			NormalLowF:  minF,
			NormalLowC:  (minF - 32.0) * 5.0 / 9.0,
			NormalMeanF: meanF,
			NormalMeanC: (meanF - 32.0) * 5.0 / 9.0,
		}
	}

	return nil
}

// ComputeDeparture calculates the departure anomaly of current conditions against daily normals.
func ComputeDeparture(normals models.DailyNormals, obs *models.CurrentConditions) *models.ClimateDeparture {
	if obs == nil {
		return nil
	}
	dep := &models.ClimateDeparture{}
	if obs.TempC != nil {
		curC := *obs.TempC
		curF := curC*9.0/5.0 + 32.0
		dep.ObservedCurrentC = &curC
		dep.ObservedCurrentF = &curF

		diffF := curF - normals.NormalMeanF
		diffC := diffF * 5.0 / 9.0
		dep.DepartureCurrentF = &diffF
		dep.DepartureCurrentC = &diffC

		sign := "+"
		if diffF < 0 {
			sign = ""
		}
		cat := "Near Normal"
		if diffF >= 8.0 {
			cat = "Significantly Above Normal"
		} else if diffF >= 3.0 {
			cat = "Above Normal"
		} else if diffF <= -8.0 {
			cat = "Significantly Below Normal"
		} else if diffF <= -3.0 {
			cat = "Below Normal"
		}
		dep.Summary = fmt.Sprintf("%s%.1f°F Departure (%s)", sign, diffF, cat)
	}
	return dep
}

func parseFloat(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case string:
		clean := strings.TrimSpace(val)
		if clean == "M" || clean == "" {
			return 0, false
		}
		if clean == "T" {
			return 0.001, true // Trace
		}
		clean = strings.TrimRight(clean, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz*")
		f, err := strconv.ParseFloat(clean, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}
