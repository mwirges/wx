package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mwirges/wx/internal/airquality"
	"github.com/mwirges/wx/internal/astro"
	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/nowcast"
	"github.com/mwirges/wx/internal/output"
	"github.com/mwirges/wx/internal/provider"
	_ "github.com/mwirges/wx/internal/provider/nws"
	_ "github.com/mwirges/wx/internal/provider/openmeteo"
)

const (
	// toolSlots is the per-process cap on in-flight weather tool calls.
	// wx_server_info does not take a slot.
	toolSlots = 4
	// toolTimeout bounds one weather tool, including geocode and provider retries.
	toolTimeout = 30 * time.Second
)

// errBusy is the local concurrency limit. It is not an upstream 429.
var errBusy = errors.New("tool concurrency limit reached")

// HTTP 429 from NWS, and nowcast's "upstream error 502", both count.
var httpStatusRe = regexp.MustCompile(`(?:HTTP |upstream error )(\d{3})\b`)

// weatherFunc replaces the live provider path in tests.
type weatherFunc func(ctx context.Context, kind string, loc location.Location, units string, hours int) (json.RawMessage, error)

// toolError is the S6 payload. Error() is the JSON text placed in the MCP
// tool result when isError is set.
type toolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *toolError) Error() string {
	b, err := json.Marshal(e)
	if err != nil {
		return e.Code + ": " + e.Message
	}
	return string(b)
}

type wxArgs struct {
	Location string `json:"location,omitempty" jsonschema:"Optional place. Omit, empty, or default uses the configured default. Otherwise a favorite, ZIP, City ST, or lat,lon."`
	Units    string `json:"units,omitempty" jsonschema:"imperial or metric. Omit to use the configured units."`
}

type hourlyArgs struct {
	Location string `json:"location,omitempty" jsonschema:"Optional place. Omit, empty, or default uses the configured default. Otherwise a favorite, ZIP, City ST, or lat,lon."`
	Units    string `json:"units,omitempty" jsonschema:"imperial or metric. Omit to use the configured units."`
	Hours    int    `json:"hours,omitempty" jsonschema:"How many hourly periods to return, from 1 to 168. Default 24."`
}

func (s *Server) addWeatherTools() {
	if s.slots == nil {
		s.slots = make(chan struct{}, toolSlots)
	}
	const polite = " Be polite to Nominatim and NWS: these calls share a small in-process cap."
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_conditions",
		Description: "Current conditions for the default location or a location argument. JSON matches wx --json conditions. City and ZIP use the shared geocoder; lat,lon does not." + polite,
	}, s.wxConditions)
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_forecast",
		Description: "Daily forecast periods for the default location or a location argument. JSON matches wx --forecast --json forecast." + polite,
	}, s.wxForecast)
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_hourly",
		Description: "Hourly forecast periods. hours defaults to 24. JSON periods match wx hourly --json, capped to hours." + polite,
	}, s.wxHourly)
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_alerts",
		Description: "Active weather alerts for the location. Returns a JSON array. An empty array means none are active." + polite,
	}, s.wxAlerts)
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_nowcast",
		Description: "Quantitative precipitation nowcast. JSON matches wx nowcast --json, including the nowcast object." + polite,
	}, s.wxNowcast)
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_aqi",
		Description: "Air quality, smoke, and the EPA health advisory. JSON matches wx aqi --json, including air_quality." + polite,
	}, s.wxAQI)
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_astro",
		Description: "Sun, moon, and twilight. JSON matches wx astro --json, including astronomy. date is YYYY-MM-DD; omit it for today." + polite,
	}, s.wxAstro)
	s.addRadarTool()
}

func (s *Server) wxConditions(ctx context.Context, _ *sdkmcp.CallToolRequest, in wxArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.finish(s.fetchLive(ctx, "conditions", in.Location, in.Units, 0))
}

func (s *Server) wxForecast(ctx context.Context, _ *sdkmcp.CallToolRequest, in wxArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.finish(s.fetchLive(ctx, "forecast", in.Location, in.Units, 0))
}

func (s *Server) wxHourly(ctx context.Context, _ *sdkmcp.CallToolRequest, in hourlyArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.finish(s.fetchLive(ctx, "hourly", in.Location, in.Units, in.Hours))
}

func (s *Server) wxAlerts(ctx context.Context, _ *sdkmcp.CallToolRequest, in wxArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.finish(s.fetchLive(ctx, "alerts", in.Location, in.Units, 0))
}

type astroArgs struct {
	Location string `json:"location,omitempty" jsonschema:"Optional place. Omit, empty, or default uses the configured default. Otherwise a favorite, ZIP, City ST, or lat,lon."`
	Units    string `json:"units,omitempty" jsonschema:"imperial or metric. Omit to use the configured units. Astronomy JSON includes both and does not change with units."`
	Date     string `json:"date,omitempty" jsonschema:"Calculation date YYYY-MM-DD in local time. Omit for today, matching wx astro --json."`
}

func (s *Server) wxNowcast(ctx context.Context, _ *sdkmcp.CallToolRequest, in wxArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.finish(s.fetchLive(ctx, "nowcast", in.Location, in.Units, 0))
}

func (s *Server) wxAQI(ctx context.Context, _ *sdkmcp.CallToolRequest, in wxArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.finish(s.fetchLive(ctx, "aqi", in.Location, in.Units, 0))
}

func (s *Server) wxAstro(ctx context.Context, _ *sdkmcp.CallToolRequest, in astroArgs) (*sdkmcp.CallToolResult, any, error) {
	when, err := parseAstroDate(in.Date)
	if err != nil {
		return s.finish(nil, err)
	}
	return s.finish(s.fetchProduct(ctx, productReq{
		kind:  "astro",
		loc:   in.Location,
		units: in.Units,
		when:  when,
	}))
}

func parseAstroDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format %q, use YYYY-MM-DD: %w", dateStr, err)
	}
	return parsed, nil
}

func (s *Server) finish(raw json.RawMessage, err error) (*sdkmcp.CallToolResult, any, error) {
	if err != nil {
		return nil, nil, classify(err)
	}
	if len(raw) == 0 {
		raw = json.RawMessage(`null`)
	}
	return nil, json.RawMessage(raw), nil
}

func (s *Server) acquire() error {
	if s.slots == nil {
		return nil
	}
	select {
	case s.slots <- struct{}{}:
		return nil
	default:
		return errBusy
	}
}

func (s *Server) release() {
	if s.slots == nil {
		return
	}
	<-s.slots
}

// productReq is one weather-tool fetch. when is the astro calculation time.
// The zero time means wx astro with no --date: calculate for time.Now().
type productReq struct {
	kind  string
	loc   string
	units string
	hours int
	when  time.Time
}

func (s *Server) fetchLive(ctx context.Context, kind, locArg, unitsArg string, hours int) (json.RawMessage, error) {
	return s.fetchProduct(ctx, productReq{kind: kind, loc: locArg, units: unitsArg, hours: hours})
}

func (s *Server) fetchProduct(ctx context.Context, req productReq) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()

	cfg, err := config.Load()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}
	ch, err := cache.New()
	if err != nil {
		ch = cache.NewNoOp()
	}
	// Slot covers Nominatim, ipinfo, and the live fetch. Fail fast when full.
	if err := s.acquire(); err != nil {
		return nil, err
	}
	defer s.release()

	loc, query, err := resolveToolLocation(ctx, cfg, req.loc, ch)
	if err != nil {
		return nil, err
	}
	units, err := normalizeUnits(cfg, req.units, query, req.loc, loc.DisplayName)
	if err != nil {
		return nil, err
	}
	hours, err := normalizeHours(req.hours)
	if err != nil {
		return nil, err
	}

	if s.weather != nil {
		return s.weather(ctx, req.kind, loc, units, hours)
	}

	switch req.kind {
	case "nowcast", "aqi", "astro":
		return fetchDirect(ctx, req.kind, loc, ch, req.when)
	}

	pref := cfg.GetEffectiveProvider(query, strings.TrimSpace(req.loc), loc.DisplayName)
	prov, err := provider.ForLocationWithPreference(loc, pref)
	if err != nil {
		return nil, err
	}
	raw, ferr := fetchFrom(ctx, req.kind, prov, loc, ch, units, hours)
	if ferr != nil {
		if fb, fbErr := provider.FallbackForLocation(loc, prov.Name()); fbErr == nil && !strings.EqualFold(fb.Name(), prov.Name()) {
			raw, ferr = fetchFrom(ctx, req.kind, fb, loc, ch, units, hours)
		}
	}
	return raw, ferr
}

// fetchDirect is wx nowcast / wx aqi / wx astro. Those commands do not go
// through WeatherProvider. The JSON is the same struct the CLI renders.
func fetchDirect(ctx context.Context, kind string, loc location.Location, ch *cache.Cache, when time.Time) (json.RawMessage, error) {
	switch kind {
	case "nowcast":
		nc, err := nowcast.Fetch(ctx, loc.Lat, loc.Lon, loc.DisplayName, ch)
		if err != nil {
			return nil, err
		}
		if nc == nil {
			return nil, errors.New("weather: empty nowcast")
		}
		return mustJSON(&models.NowcastPayload{Location: loc.DisplayName, Nowcast: nc})
	case "aqi":
		aq, err := airquality.Fetch(ctx, loc.Lat, loc.Lon, ch)
		if err != nil {
			return nil, err
		}
		if aq == nil {
			return nil, errors.New("weather: empty air quality")
		}
		payload := &models.AirQualityPayload{
			Location:   loc.DisplayName,
			Latitude:   loc.Lat,
			Longitude:  loc.Lon,
			FetchedAt:  time.Now().UTC(),
			AirQuality: aq,
		}
		if aq.AQI != nil {
			payload.HealthAdvisory = models.EPAHealthAdvisory(*aq.AQI)
		}
		if aq.PM25 != nil {
			payload.SmokeAdvisory = models.SmokeAdvisory(*aq.PM25)
		}
		return mustJSON(payload)
	case "astro":
		calcTime := when
		if calcTime.IsZero() {
			calcTime = time.Now()
		}
		astronomy, err := astro.Calculate(loc.Lat, loc.Lon, calcTime)
		if err != nil {
			return nil, fmt.Errorf("calculating astronomical ephemeris: %w", err)
		}
		return mustJSON(&models.AstroPayload{
			Location:     loc.DisplayName,
			Latitude:     loc.Lat,
			Longitude:    loc.Lon,
			CalculatedAt: calcTime,
			Astronomy:    astronomy,
		})
	default:
		return nil, fmt.Errorf("unknown product %q", kind)
	}
}

func mustJSON(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func resolveToolLocation(ctx context.Context, cfg *config.Config, arg string, ch location.Cacher) (location.Location, string, error) {
	arg = strings.TrimSpace(arg)
	var query string
	if arg == "" || strings.EqualFold(arg, "default") {
		query = cfg.ResolveLocation("")
	} else {
		query = cfg.ResolveLocation(arg)
	}
	loc, err := location.Resolve(ctx, query, ch)
	if err != nil {
		return location.Location{}, query, err
	}
	return loc, query, nil
}

func normalizeUnits(cfg *config.Config, explicit, resolved, raw, display string) (string, error) {
	u := strings.ToLower(strings.TrimSpace(explicit))
	if u == "" && cfg != nil {
		u = cfg.GetEffectiveUnits(resolved, strings.TrimSpace(raw), display)
	}
	if u == "" {
		u = "imperial"
	}
	if u != "imperial" && u != "metric" {
		return "", fmt.Errorf("units must be imperial or metric")
	}
	return u, nil
}

func normalizeHours(h int) (int, error) {
	if h == 0 {
		return 24, nil
	}
	if h < 1 || h > 168 {
		return 0, fmt.Errorf("hours must be between 1 and 168")
	}
	return h, nil
}

func fetchFrom(ctx context.Context, kind string, p provider.WeatherProvider, loc location.Location, ch *cache.Cache, units string, hours int) (json.RawMessage, error) {
	switch kind {
	case "conditions":
		cond, err := p.CurrentConditions(ctx, loc, ch)
		if err != nil {
			return nil, err
		}
		if cond == nil {
			return nil, errors.New("weather: empty conditions")
		}
		if cond.Astronomy == nil && (loc.Lat != 0 || loc.Lon != 0) {
			obs := cond.ObservedAt
			if obs.IsZero() {
				obs = time.Now()
			}
			if a, aerr := astro.Calculate(loc.Lat, loc.Lon, obs); aerr == nil {
				cond.Astronomy = a
			}
		}
		return output.ConditionsJSON(cond, units)
	case "forecast":
		fc, err := p.Forecast(ctx, loc, false, ch)
		if err != nil {
			return nil, err
		}
		if fc == nil {
			return nil, errors.New("weather: empty forecast")
		}
		return output.ForecastJSON(fc, units)
	case "hourly":
		fc, err := p.Forecast(ctx, loc, true, ch)
		if err != nil {
			return nil, err
		}
		if fc == nil {
			return nil, errors.New("weather: empty hourly forecast")
		}
		return output.HourlyJSON(fc.GeneratedAt, fc.Periods, units, hours)
	case "alerts":
		alerts, err := p.Alerts(ctx, loc, ch)
		if err != nil {
			return nil, err
		}
		return output.AlertsJSON(alerts)
	default:
		return nil, fmt.Errorf("unknown product %q", kind)
	}
}

func classify(err error) error {
	if err == nil {
		return nil
	}
	var already *toolError
	if errors.As(err, &already) {
		return already
	}
	if errors.Is(err, errBusy) {
		return &toolError{Code: "rate_limited", Message: err.Error()}
	}
	if errors.Is(err, location.ErrInvalidLocation) {
		return &toolError{Code: "invalid_location", Message: err.Error()}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &toolError{Code: "upstream_timeout", Message: err.Error()}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &toolError{Code: "upstream_timeout", Message: err.Error()}
	}
	if code := httpCode(err); code == 429 {
		return &toolError{Code: "rate_limited", Message: err.Error()}
	}
	if code := httpCode(err); code >= 400 {
		return &toolError{Code: "upstream_http", Message: err.Error()}
	}
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "not found") {
		return &toolError{Code: "invalid_location", Message: msg}
	}
	return &toolError{Code: "internal", Message: msg}
}

func httpCode(err error) int {
	m := httpStatusRe.FindStringSubmatch(err.Error())
	if len(m) != 2 {
		return 0
	}
	var n int
	for _, r := range m[1] {
		n = n*10 + int(r-'0')
	}
	return n
}
