package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/output"
)

func TestSliceEToolsRegistered(t *testing.T) {
	s := testServer(t)
	_, cs := connectMCP(t, s)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, tool := range res.Tools {
		got[tool.Name] = true
	}
	for _, name := range []string{"wx_climate", "wx_sounding", "wx_outlook", "wx_spc"} {
		if !got[name] {
			t.Errorf("missing tool %s", name)
		}
	}
}

func TestToolJSONMaxBytes(t *testing.T) {
	if ToolJSONMaxBytes != 256<<10 {
		t.Fatalf("ToolJSONMaxBytes = %d, want %d", ToolJSONMaxBytes, 256<<10)
	}
	fit := paddedJSON(t, ToolJSONMaxBytes)
	got, err := capToolJSON(fit)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != ToolJSONMaxBytes {
		t.Fatalf("fit len = %d", len(got))
	}
	over := paddedJSON(t, ToolJSONMaxBytes+1)
	_, err = capToolJSON(over)
	assertPayloadTooLarge(t, err)
	classified := classify(err)
	var te *toolError
	if !errors.As(classified, &te) || te.Code != "payload_too_large" {
		t.Fatalf("classify remapped cap: %v", classified)
	}
}

func TestSliceEHappyPath(t *testing.T) {
	s := testServer(t)
	var kinds []string
	s.sliceE = func(_ context.Context, kind string, loc location.Location, when time.Time, station string) (json.RawMessage, error) {
		kinds = append(kinds, kind)
		if loc.Lat != 41.07 || loc.Lon != -85.14 {
			t.Errorf("loc = %#v", loc)
		}
		switch kind {
		case sliceEClimate:
			if !when.IsZero() {
				t.Errorf("climate date = %s", when)
			}
			return json.RawMessage(`{"location":"41.07, -85.14","climate":{"station_id":"USW00014827"}}`), nil
		case sliceESounding:
			if station != "" {
				t.Errorf("station = %q", station)
			}
			return json.RawMessage(`{"location":"41.07, -85.14","sounding":{"station_id":"ILN","skewt_image_url":"https://www.spc.noaa.gov/exper/soundings/26010100_OBS/ILN.gif","indices":{"sbcape_jkg":1200}}}`), nil
		case sliceEOutlook:
			return json.RawMessage(`{"location":"41.07, -85.14","outlooks":[{"horizon":"6-10 Day"},{"horizon":"8-14 Day"}]}`), nil
		case sliceESPC:
			return json.RawMessage(`{"location":"41.07, -85.14","day1":{"day":1},"day2":{"day":2},"day3":{"day":3},"active_mcds":[],"active_watches":[]}`), nil
		default:
			t.Errorf("kind %s", kind)
			return nil, errString("unexpected")
		}
	}
	_, cs := connectMCP(t, s)
	loc := map[string]any{"location": "41.07,-85.14"}
	for _, tc := range []struct {
		tool string
		key  string
	}{
		{"wx_climate", "climate"},
		{"wx_sounding", "sounding"},
		{"wx_outlook", "outlooks"},
		{"wx_spc", "day1"},
	} {
		res := call(t, cs, tc.tool, loc)
		if res.IsError {
			t.Fatalf("%s: %s", tc.tool, resultText(res))
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal([]byte(resultText(res)), &body); err != nil {
			t.Fatalf("%s: %v %s", tc.tool, err, resultText(res))
		}
		if _, ok := body[tc.key]; !ok {
			t.Fatalf("%s missing %s in %s", tc.tool, tc.key, resultText(res))
		}
		if tc.tool == "wx_sounding" {
			if strings.Contains(resultText(res), "image_base64") || strings.Contains(resultText(res), "data:image") {
				t.Fatalf("sounding embedded image bytes: %s", resultText(res))
			}
			var snd struct {
				Sounding struct {
					URL string `json:"skewt_image_url"`
				} `json:"sounding"`
			}
			if err := json.Unmarshal([]byte(resultText(res)), &snd); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(snd.Sounding.URL, "https://") {
				t.Fatalf("skewt_image_url = %q", snd.Sounding.URL)
			}
		}
	}
	if strings.Join(kinds, ",") != "climate,sounding,outlook,spc" {
		t.Fatalf("kinds = %v", kinds)
	}
}

func TestSliceEPayloadTooLarge(t *testing.T) {
	s := testServer(t)
	over := paddedJSON(t, ToolJSONMaxBytes+1)
	s.sliceE = func(context.Context, string, location.Location, time.Time, string) (json.RawMessage, error) {
		return over, nil
	}
	_, cs := connectMCP(t, s)
	loc := map[string]any{"location": "41.07,-85.14"}
	for _, tool := range []string{"wx_climate", "wx_sounding", "wx_outlook", "wx_spc"} {
		res := call(t, cs, tool, loc)
		assertCode(t, res, "payload_too_large")
		text := resultText(res)
		if strings.Contains(text, `"pad"`) {
			t.Fatalf("%s leaked oversize JSON: %s", tool, text[:80])
		}
		if len(text) > 512 {
			t.Fatalf("%s error text is %d bytes", tool, len(text))
		}
	}
}

func TestSliceESkewTRejectsEmbeddedBytes(t *testing.T) {
	s := testServer(t)
	s.sliceE = func(context.Context, string, location.Location, time.Time, string) (json.RawMessage, error) {
		return json.RawMessage(`{"location":"x","sounding":{"skewt_image_url":"data:image/gif;base64,AAAA"}}`), nil
	}
	_, cs := connectMCP(t, s)
	res := call(t, cs, "wx_sounding", map[string]any{"location": "41.07,-85.14"})
	assertCode(t, res, "payload_too_large")
	if strings.Contains(resultText(res), "base64") || strings.Contains(resultText(res), "data:image") {
		t.Fatalf("error echoed image bytes: %s", resultText(res))
	}
}

func TestSliceEInvalidLocationAndBadDate(t *testing.T) {
	s := testServer(t)
	s.sliceE = func(context.Context, string, location.Location, time.Time, string) (json.RawMessage, error) {
		t.Fatal("fetch ran")
		return nil, nil
	}
	_, cs := connectMCP(t, s)
	for _, tool := range []string{"wx_climate", "wx_sounding", "wx_outlook", "wx_spc"} {
		res := call(t, cs, tool, map[string]any{"location": "91,0"})
		assertCode(t, res, "invalid_location")
	}
	res := call(t, cs, "wx_climate", map[string]any{"location": "41.07,-85.14", "date": "nope"})
	assertCode(t, res, "internal")
	res = call(t, cs, "wx_climate", map[string]any{"location": "41.07,-85.14", "units": "furlongs"})
	assertCode(t, res, "internal")
}

func TestSliceESharesSlots(t *testing.T) {
	s := testServer(t)
	s.slots = make(chan struct{}, 1)
	s.slots <- struct{}{}
	s.sliceE = func(context.Context, string, location.Location, time.Time, string) (json.RawMessage, error) {
		t.Fatal("fetch ran while the slot was full")
		return nil, nil
	}
	_, cs := connectMCP(t, s)
	res := call(t, cs, "wx_spc", map[string]any{"location": "41.07,-85.14"})
	assertCode(t, res, "rate_limited")
}

func TestSliceEStationWithoutGeocode(t *testing.T) {
	loc, query, station, err := resolveSliceELocation(context.Background(), &config.Config{}, "", "iln", nil)
	if err != nil {
		t.Fatal(err)
	}
	if station != "ILN" || loc.Lat != 39.42 || loc.Lon != -83.82 || query != "Wilmington, OH" {
		t.Fatalf("station resolve = %#v %q %q", loc, query, station)
	}
	_, _, _, err = resolveSliceELocation(context.Background(), &config.Config{}, "91,0", "ILN", nil)
	if !errors.Is(err, location.ErrInvalidLocation) {
		t.Fatalf("coord+station err = %v", err)
	}
}

func TestSliceERejectsNonUS(t *testing.T) {
	loc := location.Location{DisplayName: "Paris", CountryCode: "FR", Lat: 48.8, Lon: 2.3}
	if _, err := liveOutlook(context.Background(), loc, nil); err == nil || !strings.Contains(err.Error(), "only available for US") {
		t.Fatalf("outlook err = %v", err)
	}
	if _, err := liveSPC(context.Background(), loc, nil); err == nil || !strings.Contains(err.Error(), "only available for US") {
		t.Fatalf("spc err = %v", err)
	}
	bare := location.Location{DisplayName: "41.07, -85.14", Lat: 41.07, Lon: -85.14}
	if _, err := liveOutlook(context.Background(), bare, nil); err == nil || !strings.Contains(err.Error(), "detected: ") {
		t.Fatalf("empty country outlook err = %v", err)
	}
}

func TestSliceEJSONMatchesCLIRender(t *testing.T) {
	when := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	high := 4.2
	climatePayload := &models.ClimatePayload{
		Location: "Fort Wayne, IN",
		Climate: &models.ClimateReport{
			Date:          when,
			Location:      "Fort Wayne, IN",
			StationID:     "USW00014827",
			StationName:   "Fort Wayne",
			Latitude:      41.0,
			Longitude:     -85.2,
			NormalsPeriod: "1991–2020",
			TodayNormals:  models.DailyNormals{Date: when, NormalHighF: 60, NormalHighC: 15.6},
			Records: models.DailyRecords{
				RecordHigh: models.DailyRecord{ValueF: 80, ValueC: 26.7, Years: []int{1990}},
			},
			Departure: &models.ClimateDeparture{DepartureHighF: &high, Summary: "warm"},
		},
	}
	got, err := mustJSON(climatePayload)
	if err != nil {
		t.Fatal(err)
	}
	capped, err := capToolJSON(got)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := output.RenderClimateTo(&buf, climatePayload, output.ClimateOptions{ForceJSON: true, Units: "imperial"}); err != nil {
		t.Fatal(err)
	}
	assertSameJSON(t, capped, buf.Bytes())

	cape := 1500.0
	snd := &models.SoundingPayload{
		Location: "Fort Wayne, IN",
		Sounding: &models.SoundingReport{
			Timestamp:     when,
			Location:      "Fort Wayne, IN",
			StationID:     "ILN",
			StationName:   "Wilmington",
			Provider:      "NOAA SPC",
			SkewTImageURL: "https://www.spc.noaa.gov/exper/soundings/26103112_OBS/ILN.gif",
			Indices:       models.ConvectiveIndices{SBCAPE: &cape, InstabilitySummary: "moderate", ShearSummary: "weak", ConvectiveRisk: "marginal"},
			Levels:        []models.SoundingLevel{{PressureHPA: 500}},
		},
	}
	got, err = mustJSON(snd)
	if err != nil {
		t.Fatal(err)
	}
	kept, err := guardSkewT(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(kept), "https://www.spc.noaa.gov/exper/soundings/26103112_OBS/ILN.gif") {
		t.Fatalf("url dropped: %s", kept)
	}
	if strings.Contains(string(kept), "image_base64") || strings.Contains(string(kept), "data:image") {
		t.Fatalf("sounding JSON embedded bytes: %s", kept)
	}
	buf.Reset()
	if err := output.RenderSoundingTo(&buf, snd, output.SoundingOptions{ForceJSON: true, Units: "imperial"}); err != nil {
		t.Fatal(err)
	}
	assertSameJSON(t, kept, buf.Bytes())

	cpcPayload := &models.CPCPayload{
		Location:    "Fort Wayne, IN",
		Coordinates: [2]float64{41.08, -85.14},
		FetchedAt:   when,
		Outlooks: []models.CPCOutlookItem{
			{Horizon: "6-10 Day", StartDate: when, EndDate: when.Add(24 * time.Hour), TempCategory: "Above", TempProbability: 50, PrecipCategory: "Below", PrecipProbability: 40},
			{Horizon: "8-14 Day", StartDate: when, EndDate: when.Add(48 * time.Hour), TempCategory: "Normal", TempProbability: 33, PrecipCategory: "Above", PrecipProbability: 40},
		},
		PatternShift: models.CPCPatternShift{HasShift: true, Summary: "warming"},
	}
	got, err = mustJSON(cpcPayload)
	if err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := output.RenderCPCTo(&buf, cpcPayload, output.CPCOptions{ForceJSON: true}); err != nil {
		t.Fatal(err)
	}
	assertSameJSON(t, got, buf.Bytes())

	spcPayload := &models.SPCPayload{
		Location:    "Fort Wayne, IN",
		Coordinates: [2]float64{41.08, -85.14},
		FetchedAt:   when,
		Day1:        models.SPCOutlookItem{Day: 1, Valid: when, Category: models.SPCRiskCategory{DN: 2, Code: "TSTM", Name: "Thunderstorm"}},
		Day2:        models.SPCOutlookItem{Day: 2, Valid: when, Category: models.SPCRiskCategory{Code: "NONE"}},
		Day3:        models.SPCOutlookItem{Day: 3, Valid: when, Category: models.SPCRiskCategory{Code: "NONE"}},
		ActiveMCDs: []models.MesoscaleDiscussion{{
			ID: 1, Name: "MD 1", Title: "MD 1", Discussion: "short text", Sent: when,
		}},
		ActiveWatches:     []models.SPCWatch{},
		MaxNationalRisk:   models.SPCRiskCategory{DN: 2, Code: "TSTM", Name: "Thunderstorm"},
		ConvectiveSummary: "General thunder",
	}
	got, err = mustJSON(spcPayload)
	if err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := output.RenderSPCTo(&buf, spcPayload, output.SPCOptions{ForceJSON: true}); err != nil {
		t.Fatal(err)
	}
	assertSameJSON(t, got, buf.Bytes())
}

func paddedJSON(t *testing.T, n int) json.RawMessage {
	t.Helper()
	const prefix = `{"pad":"`
	const suffix = `"}`
	inner := n - len(prefix) - len(suffix)
	if inner < 1 {
		t.Fatalf("n %d too small", n)
	}
	body := prefix + strings.Repeat("a", inner) + suffix
	if len(body) != n {
		t.Fatalf("padded len = %d, want %d", len(body), n)
	}
	return json.RawMessage(body)
}
