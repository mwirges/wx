package nowcast

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/models"
)

const (
	defaultBaseURL = "https://api.open-meteo.com"
	cacheTTL       = 15 * time.Minute // Short-term precipitation nowcast TTL
)

type openMeteoResponse struct {
	Minutely15 struct {
		Time          []string  `json:"time"`
		Precipitation []float64 `json:"precipitation"`
		Rain          []float64 `json:"rain"`
		Snowfall      []float64 `json:"snowfall"`
		WeatherCode   []int     `json:"weather_code"`
	} `json:"minutely_15"`
	Hourly struct {
		Time                     []string  `json:"time"`
		PrecipitationProbability []int     `json:"precipitation_probability"`
		Precipitation            []float64 `json:"precipitation"`
		Rain                     []float64 `json:"rain"`
		Snowfall                 []float64 `json:"snowfall"`
		WeatherCode              []int     `json:"weather_code"`
	} `json:"hourly"`
}

// Client fetches quantitative precipitation nowcast telemetry.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient returns a new nowcast Client.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

var defaultClient = NewClient("")

// Fetch returns precipitation nowcast telemetry for given coordinates and location name.
func Fetch(ctx context.Context, lat, lon float64, locName string, c *cache.Cache) (*models.Nowcast, error) {
	return defaultClient.Fetch(ctx, lat, lon, locName, c)
}

// Fetch returns precipitation nowcast telemetry for given coordinates and location name.
func (cl *Client) Fetch(ctx context.Context, lat, lon float64, locName string, c *cache.Cache) (*models.Nowcast, error) {
	cacheKey := fmt.Sprintf("nowcast:%.4f,%.4f", lat, lon)
	var cached models.Nowcast
	if c != nil && c.Get(cacheKey, &cached) {
		return &cached, nil
	}

	url := fmt.Sprintf(
		"%s/v1/forecast?latitude=%.4f&longitude=%.4f&minutely_15=precipitation,rain,snowfall,weather_code&hourly=precipitation_probability,precipitation,rain,snowfall,weather_code&forecast_days=2&timezone=auto",
		cl.baseURL, lat, lon,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("nowcast: request: %w", err)
	}
	req.Header.Set("User-Agent", "wx-cli/1.0 (weather nowcast)")

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nowcast: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("nowcast: upstream error %d: %s", resp.StatusCode, string(body))
	}

	var data openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("nowcast: parse json: %w", err)
	}

	nowcast := ParseNowcast(data, locName, time.Now())
	if c != nil {
		_ = c.Set(cacheKey, nowcast, cacheTTL)
	}
	return nowcast, nil
}

// ParseNowcast parses Open-Meteo 15-minute and hourly telemetry into a Nowcast model.
func ParseNowcast(data openMeteoResponse, locName string, now time.Time) *models.Nowcast {
	res := &models.Nowcast{
		GeneratedAt:  now,
		Location:     locName,
		PrimaryPhase: models.PhaseNone,
		Intervals:    make([]models.PrecipInterval, 0),
	}

	// First pass: Build 15-minute intervals starting near current time up to ~6-8 hours ahead
	times := data.Minutely15.Time
	precip := data.Minutely15.Precipitation
	rain := data.Minutely15.Rain
	snow := data.Minutely15.Snowfall
	codes := data.Minutely15.WeatherCode

	// Map hourly probability by hour timestamp string
	hourlyProbMap := make(map[string]float64)
	for i, hTime := range data.Hourly.Time {
		if i < len(data.Hourly.PrecipitationProbability) {
			hourlyProbMap[hTime] = float64(data.Hourly.PrecipitationProbability[i])
		}
	}

	cutoff := now.Add(-15 * time.Minute)
	maxForward := now.Add(8 * time.Hour)

	var totalLiquidMM float64
	var totalSnowCM float64
	var peakRateMMH float64
	var peakTime time.Time
	rainCount := 0
	snowCount := 0

	for i := 0; i < len(times); i++ {
		t, err := time.Parse("2006-01-02T15:04", times[i])
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			continue
		}
		if t.After(maxForward) {
			break
		}

		pMM := 0.0
		if i < len(precip) {
			pMM = math.Max(0, precip[i])
		}
		rMM := 0.0
		if i < len(rain) {
			rMM = math.Max(0, rain[i])
		}
		sCM := 0.0
		if i < len(snow) {
			sCM = math.Max(0, snow[i])
		}
		code := 0
		if i < len(codes) {
			code = codes[i]
		}

		// Rate in mm/h (15-min interval accumulated value multiplied by 4)
		rateMMH := pMM * 4.0
		rateInH := rateMMH / 25.4
		accumIn := pMM / 25.4

		phase, summary := classifyPhaseAndSummary(code, rMM, sCM, rateMMH)
		if phase == models.PhaseRain {
			rainCount++
		} else if phase == models.PhaseSnow {
			snowCount++
		}

		// Probability: lookup hourly matching hour
		hourKey := t.Format("2006-01-02T15:00")
		prob := 0.0
		if p, ok := hourlyProbMap[hourKey]; ok {
			prob = p
		} else if rateMMH > 0.05 {
			prob = math.Min(100, 50.0+rateMMH*15)
		}

		interval := models.PrecipInterval{
			StartTime:   t,
			EndTime:     t.Add(15 * time.Minute),
			Probability: prob,
			RateMMH:     rateMMH,
			RateInH:     rateInH,
			AccumMM:     pMM,
			AccumIn:     accumIn,
			Phase:       phase,
			Summary:     summary,
		}
		res.Intervals = append(res.Intervals, interval)

		totalLiquidMM += pMM
		totalSnowCM += sCM
		if rateMMH > peakRateMMH {
			peakRateMMH = rateMMH
			peakTime = t
		}
	}

	res.TotalLiquidMM = totalLiquidMM
	res.TotalLiquidIn = totalLiquidMM / 25.4
	res.TotalSnowCM = totalSnowCM
	res.TotalSnowIn = totalSnowCM / 2.54
	res.PeakRateMMH = peakRateMMH
	res.PeakRateInH = peakRateMMH / 25.4
	res.PeakTime = peakTime

	if snowCount > rainCount && snowCount > 0 {
		res.PrimaryPhase = models.PhaseSnow
	} else if rainCount > 0 {
		res.PrimaryPhase = models.PhaseRain
	} else {
		res.PrimaryPhase = models.PhaseNone
	}

	computeHeadlinesAndStatus(res, now)
	return res
}

func classifyPhaseAndSummary(code int, rainMM, snowCM, rateMMH float64) (models.PrecipPhase, string) {
	if rateMMH < 0.02 && code <= 3 {
		return models.PhaseNone, "Clear"
	}

	// Freezing rain
	if code == 66 || code == 67 || code == 56 || code == 57 {
		if rateMMH > 2.5 {
			return models.PhaseFreezingRain, "Heavy Freezing Rain"
		}
		return models.PhaseFreezingRain, "Freezing Rain"
	}

	// Snow
	if code >= 71 && code <= 77 || code == 85 || code == 86 || snowCM > 0.05 {
		if rateMMH > 4.0 {
			return models.PhaseSnow, "Heavy Snow"
		} else if rateMMH > 1.5 {
			return models.PhaseSnow, "Moderate Snow"
		}
		return models.PhaseSnow, "Light Snow"
	}

	// Thunderstorm / hail
	if code >= 95 {
		return models.PhaseRain, "Thunderstorms"
	}

	// Rain
	if code >= 51 && code <= 65 || code >= 80 && code <= 82 || rainMM > 0.02 || rateMMH >= 0.05 {
		if rateMMH > 7.5 {
			return models.PhaseRain, "Heavy Rain"
		} else if rateMMH > 2.5 {
			return models.PhaseRain, "Moderate Rain"
		} else if rateMMH > 0.5 {
			return models.PhaseRain, "Light Rain"
		}
		return models.PhaseRain, "Drizzle"
	}

	return models.PhaseNone, "Overcast"
}

func computeHeadlinesAndStatus(res *models.Nowcast, now time.Time) {
	if len(res.Intervals) == 0 {
		res.Headline = "No precipitation telemetry available"
		res.Summary = "Telemetry unavailable for current window."
		return
	}

	first := res.Intervals[0]
	res.IsActivePrecip = first.RateMMH >= 0.05

	phaseLabel := "Rain"
	if res.PrimaryPhase == models.PhaseSnow {
		phaseLabel = "Snow"
	} else if res.PrimaryPhase == models.PhaseFreezingRain {
		phaseLabel = "Freezing Rain"
	}

	if res.IsActivePrecip {
		// Active precipitation right now -> search for cessation
		var endTime *time.Time
		for _, iv := range res.Intervals {
			if iv.RateMMH < 0.05 {
				t := iv.StartTime
				endTime = &t
				break
			}
		}
		res.PrecipEndTime = endTime
		if endTime != nil {
			mins := int(math.Max(5, math.Round(endTime.Sub(now).Minutes())))
			if mins < 60 {
				res.Headline = fmt.Sprintf("%s stopping in ~%d min", phaseLabel, mins)
			} else {
				res.Headline = fmt.Sprintf("%s ending around %s", phaseLabel, endTime.Local().Format("3:04 PM"))
			}
		} else {
			res.Headline = fmt.Sprintf("%s continuing through next %d hours", phaseLabel, len(res.Intervals)/4)
		}
		res.Summary = fmt.Sprintf(
			"Active %s with peak intensity %.2f in/hr (%.1f mm/h). Total expected accumulation: %.2f in.",
			phaseLabel, res.PeakRateInH, res.PeakRateMMH, res.TotalLiquidIn,
		)
	} else {
		// No precip right now -> search for next onset
		var startTime *time.Time
		var startInterval *models.PrecipInterval
		for i := range res.Intervals {
			if res.Intervals[i].RateMMH >= 0.05 {
				t := res.Intervals[i].StartTime
				startTime = &t
				startInterval = &res.Intervals[i]
				break
			}
		}
		res.NextPrecipTime = startTime
		if startTime != nil {
			mins := int(math.Round(startTime.Sub(now).Minutes()))
			if mins <= 60 && mins > 0 {
				res.Headline = fmt.Sprintf("Precipitation starting in ~%d min (%s)", mins, startInterval.Summary)
			} else if mins <= 0 {
				res.Headline = fmt.Sprintf("%s imminent", startInterval.Summary)
			} else {
				res.Headline = fmt.Sprintf("%s expected around %s (%.0f%% chance)", startInterval.Summary, startTime.Local().Format("3:04 PM"), startInterval.Probability)
			}
			res.Summary = fmt.Sprintf(
				"%s begins around %s with expected total of %.2f in liquid equivalent.",
				startInterval.Summary, startTime.Local().Format("3:04 PM"), res.TotalLiquidIn,
			)
		} else {
			hours := len(res.Intervals) / 4
			if hours < 1 {
				hours = 6
			}
			res.Headline = fmt.Sprintf("Clear · No precipitation expected next %d hours", hours)
			res.Summary = fmt.Sprintf("Conditions remain dry and clear across the %d-hour synoptic window.", hours)
		}
	}
}

// SparklineRune returns a single Unicode block character representing precipitation intensity.
func SparklineRune(rateInH float64) rune {
	switch {
	case rateInH <= 0.001:
		return ' '
	case rateInH < 0.03:
		return ' '
	case rateInH < 0.08:
		return '▂'
	case rateInH < 0.15:
		return '▃'
	case rateInH < 0.25:
		return '▄'
	case rateInH < 0.40:
		return '▅'
	case rateInH < 0.60:
		return '▆'
	case rateInH < 0.90:
		return '▇'
	default:
		return '█'
	}
}
