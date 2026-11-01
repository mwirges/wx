package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/climate"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/cpc"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/provider"
	"github.com/mwirges/wx/internal/sounding"
	"github.com/mwirges/wx/internal/spc"
)

const (
	sliceEClimate  = "climate"
	sliceESounding = "sounding"
	sliceEOutlook  = "outlook"
	sliceESPC      = "spc"
)

// ToolJSONMaxBytes is the hard cap on serialized MCP JSON for wx_climate,
// wx_sounding, wx_outlook, and wx_spc. Sounding levels, SPC discussion
// text, and outlook fields share this limit. Over the cap, the tool
// result is isError with code payload_too_large and the body is not returned.
// 256 KiB.
const ToolJSONMaxBytes = 256 << 10

// sliceEFunc replaces the live climate, sounding, outlook, and SPC fetches
// in tests. The handler still enforces ToolJSONMaxBytes and still refuses
// a sounding whose skewt_image_url is not an http(s) URL.
type sliceEFunc func(ctx context.Context, kind string, loc location.Location, when time.Time, station string) (json.RawMessage, error)

type climateArgs struct {
	Location string `json:"location,omitempty" jsonschema:"Optional place. Omit, empty, or default uses the configured default. Otherwise a favorite, ZIP, City ST, or lat,lon."`
	Units    string `json:"units,omitempty" jsonschema:"imperial or metric. Omit to use the configured units. JSON includes both, matching wx climate --json."`
	Date     string `json:"date,omitempty" jsonschema:"Target date YYYY-MM-DD. Omit for today, matching wx climate --json."`
}

type soundingArgs struct {
	Location string `json:"location,omitempty" jsonschema:"Optional place. Omit or empty with station set uses that station. Otherwise a favorite, ZIP, City ST, lat,lon, or default."`
	Units    string `json:"units,omitempty" jsonschema:"imperial or metric. Omit to use the configured units. JSON includes both, matching wx sounding --json."`
	Station  string `json:"station,omitempty" jsonschema:"NOAA upper-air station id such as ILN. Omit to use the nearest station. skewt_image_url is an http(s) URL. The skew-T image bytes are not embedded."`
}

type outlookArgs struct {
	Location string `json:"location,omitempty" jsonschema:"Optional place. Omit, empty, or default uses the configured default. Otherwise a favorite, ZIP, City ST, or lat,lon. NOAA CPC covers US locations."`
}

func (s *Server) addSliceETools() {
	const polite = " Be polite to NOAA and Nominatim: these calls share a small in-process cap. JSON over 256 KiB returns payload_too_large."
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_climate",
		Description: "NOAA 30-year climate normals, daily records, and departure. JSON matches wx climate --json." + polite,
	}, s.wxClimate)
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_sounding",
		Description: "Upper-air sounding, CAPE/CIN, and shear. JSON matches wx sounding --json. skewt_image_url stays an http(s) URL. Skew-T image bytes are not embedded." + polite,
	}, s.wxSounding)
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_outlook",
		Description: "NOAA Climate Prediction Center 6-10 and 8-14 day outlooks. JSON matches wx outlook --json." + polite,
	}, s.wxOutlook)
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_spc",
		Description: "Storm Prediction Center Day 1-3 convective outlooks, mesoscale discussions, and watches. JSON matches wx spc --json." + polite,
	}, s.wxSPC)
}

func (s *Server) wxClimate(ctx context.Context, _ *sdkmcp.CallToolRequest, in climateArgs) (*sdkmcp.CallToolResult, any, error) {
	when, err := parseAstroDate(in.Date)
	if err != nil {
		return s.finish(nil, err)
	}
	return s.finish(s.fetchSliceE(ctx, sliceEClimate, in.Location, in.Units, when, ""))
}

func (s *Server) wxSounding(ctx context.Context, _ *sdkmcp.CallToolRequest, in soundingArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.finish(s.fetchSliceE(ctx, sliceESounding, in.Location, in.Units, time.Time{}, in.Station))
}

func (s *Server) wxOutlook(ctx context.Context, _ *sdkmcp.CallToolRequest, in outlookArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.finish(s.fetchSliceE(ctx, sliceEOutlook, in.Location, "", time.Time{}, ""))
}

func (s *Server) wxSPC(ctx context.Context, _ *sdkmcp.CallToolRequest, in outlookArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.finish(s.fetchSliceE(ctx, sliceESPC, in.Location, "", time.Time{}, ""))
}

func (s *Server) fetchSliceE(ctx context.Context, kind, locArg, unitsArg string, when time.Time, station string) (json.RawMessage, error) {
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
	if err := s.acquire(); err != nil {
		return nil, err
	}
	defer s.release()

	loc, query, station, err := resolveSliceELocation(ctx, cfg, locArg, station, ch)
	if err != nil {
		return nil, err
	}
	if kind == sliceEClimate || kind == sliceESounding {
		if _, err := normalizeUnits(cfg, unitsArg, query, locArg, loc.DisplayName); err != nil {
			return nil, err
		}
	}

	var raw json.RawMessage
	if s.sliceE != nil {
		raw, err = s.sliceE(ctx, kind, loc, when, station)
	} else {
		raw, err = liveSliceE(ctx, kind, loc, when, station, ch)
	}
	if err != nil {
		return nil, err
	}
	if kind == sliceESounding {
		raw, err = guardSkewT(raw)
		if err != nil {
			return nil, err
		}
	}
	return capToolJSON(raw)
}

func liveSliceE(ctx context.Context, kind string, loc location.Location, when time.Time, station string, ch *cache.Cache) (json.RawMessage, error) {
	switch kind {
	case sliceEClimate:
		return liveClimate(ctx, loc, when, ch)
	case sliceESounding:
		return liveSounding(ctx, loc, station, ch)
	case sliceEOutlook:
		return liveOutlook(ctx, loc, ch)
	case sliceESPC:
		return liveSPC(ctx, loc, ch)
	default:
		return nil, fmt.Errorf("unknown product %q", kind)
	}
}

func liveClimate(ctx context.Context, loc location.Location, targetDate time.Time, ch *cache.Cache) (json.RawMessage, error) {
	if targetDate.IsZero() {
		targetDate = time.Now()
	}
	var obs *models.CurrentConditions
	if targetDate.Format("2006-01-02") == time.Now().Format("2006-01-02") {
		if p, err := provider.ForLocation(loc); err == nil {
			obs, _ = p.CurrentConditions(ctx, loc, ch)
		}
	}
	report, err := climate.Fetch(ctx, loc.Lat, loc.Lon, loc.DisplayName, obs, targetDate, ch)
	if err != nil {
		return nil, fmt.Errorf("fetching climate normals: %w", err)
	}
	if report == nil {
		return nil, fmt.Errorf("no climate data")
	}
	return mustJSON(&models.ClimatePayload{Location: loc.DisplayName, Climate: report})
}

func liveSounding(ctx context.Context, loc location.Location, station string, ch *cache.Cache) (json.RawMessage, error) {
	report, err := sounding.Fetch(ctx, loc.Lat, loc.Lon, loc.DisplayName, station, ch)
	if err != nil {
		return nil, fmt.Errorf("sounding fetch: %w", err)
	}
	if report == nil {
		return nil, fmt.Errorf("sounding: empty payload")
	}
	return mustJSON(&models.SoundingPayload{Location: report.Location, Sounding: report})
}

func liveOutlook(ctx context.Context, loc location.Location, ch *cache.Cache) (json.RawMessage, error) {
	if loc.CountryCode != "US" {
		return nil, fmt.Errorf("NOAA CPC outlooks are only available for US locations (detected: %s)", loc.CountryCode)
	}
	payload, err := cpc.NewClient().FetchOutlooks(ctx, loc, ch)
	if err != nil {
		return nil, fmt.Errorf("fetching CPC outlooks: %w", err)
	}
	if payload == nil {
		return nil, fmt.Errorf("no CPC outlook data")
	}
	return mustJSON(payload)
}

func liveSPC(ctx context.Context, loc location.Location, ch *cache.Cache) (json.RawMessage, error) {
	if loc.CountryCode != "US" {
		return nil, fmt.Errorf("NOAA SPC outlooks are only available for US locations (detected: %s)", loc.CountryCode)
	}
	payload, err := spc.FetchSPCOutlooks(ctx, loc, ch)
	if err != nil {
		return nil, fmt.Errorf("fetching SPC outlooks: %w", err)
	}
	if payload == nil {
		return nil, fmt.Errorf("no SPC outlook data")
	}
	return mustJSON(payload)
}

// resolveSliceELocation applies S1, with the CLI sounding exception:
// an omitted location plus a station id uses that station and does not
// geocode the configured default.
func resolveSliceELocation(ctx context.Context, cfg *config.Config, locArg, station string, ch location.Cacher) (location.Location, string, string, error) {
	station = strings.ToUpper(strings.TrimSpace(station))
	if strings.TrimSpace(locArg) == "" && station != "" {
		if st, ok := sounding.FindStationByID(station); ok {
			loc := location.Location{
				Lat:         st.Lat,
				Lon:         st.Lon,
				DisplayName: fmt.Sprintf("%s, %s", st.Name, st.State),
			}
			return loc, loc.DisplayName, station, nil
		}
		return location.Location{}, "", station, nil
	}
	loc, query, err := resolveToolLocation(ctx, cfg, locArg, ch)
	if err != nil {
		return location.Location{}, query, station, err
	}
	return loc, query, station, nil
}

// guardSkewT refuses a sounding result whose skewt_image_url is not an
// http(s) URL. An empty URL is the CLI omitempty case and is allowed.
// Image bytes are not part of the sounding JSON.
func guardSkewT(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var probe struct {
		Sounding *struct {
			SkewT string `json:"skewt_image_url"`
		} `json:"sounding"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	if probe.Sounding == nil {
		return raw, nil
	}
	u := strings.TrimSpace(probe.Sounding.SkewT)
	if u == "" {
		return raw, nil
	}
	lower := strings.ToLower(u)
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
		return raw, nil
	}
	return nil, &toolError{
		Code:    "payload_too_large",
		Message: "skew-T image bytes are not returned over MCP; skewt_image_url must be an http(s) URL",
	}
}

// capToolJSON enforces ToolJSONMaxBytes on the serialized tool JSON.
// A payload of exactly the cap is returned. One byte over is payload_too_large.
func capToolJSON(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) > ToolJSONMaxBytes {
		return nil, &toolError{
			Code:    "payload_too_large",
			Message: fmt.Sprintf("tool JSON exceeds %d byte cap (%d bytes)", ToolJSONMaxBytes, len(raw)),
		}
	}
	return raw, nil
}
