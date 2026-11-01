package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/astro"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/output"
	"github.com/mwirges/wx/internal/radar"
)

func TestSliceDNowcastAQIAstro(t *testing.T) {
	s := testServer(t)
	var kinds []string
	s.weather = func(_ context.Context, kind string, loc location.Location, units string, hours int) (json.RawMessage, error) {
		kinds = append(kinds, kind)
		if loc.Lat == 0 || units != "imperial" {
			t.Errorf("loc/units = %v %q", loc, units)
		}
		switch kind {
		case "nowcast":
			return json.RawMessage(`{"location":"41.07, -85.14","nowcast":{"headline":"Dry"}}`), nil
		case "aqi":
			return json.RawMessage(`{"location":"41.07, -85.14","air_quality":{"aqi":12}}`), nil
		case "astro":
			if hours != 24 {
				t.Errorf("astro hours = %d", hours)
			}
			return json.RawMessage(`{"location":"41.07, -85.14","astronomy":{"moon_phase":"Full Moon"}}`), nil
		default:
			t.Errorf("unexpected kind %s", kind)
			return nil, errString("unexpected")
		}
	}
	_, cs := connectMCP(t, s)
	loc := map[string]any{"location": "41.07,-85.14", "units": "imperial"}
	for _, tc := range []struct {
		tool string
		key  string
	}{
		{"wx_nowcast", "nowcast"},
		{"wx_aqi", "air_quality"},
		{"wx_astro", "astronomy"},
	} {
		res := call(t, cs, tc.tool, loc)
		if res.IsError {
			t.Fatalf("%s: %s", tc.tool, resultText(res))
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal([]byte(resultText(res)), &body); err != nil {
			t.Fatalf("%s body: %v %s", tc.tool, err, resultText(res))
		}
		if _, ok := body[tc.key]; !ok {
			t.Fatalf("%s missing %s in %s", tc.tool, tc.key, resultText(res))
		}
	}
	if strings.Join(kinds, ",") != "nowcast,aqi,astro" {
		t.Fatalf("kinds = %v", kinds)
	}

	s.weather = func(context.Context, string, location.Location, string, int) (json.RawMessage, error) {
		t.Fatal("weather ran for a bad astro date")
		return nil, nil
	}
	_, cs = connectMCP(t, s)
	res := call(t, cs, "wx_astro", map[string]any{"location": "41.07,-85.14", "date": "nope"})
	assertCode(t, res, "internal")
}

func TestDirectJSONMatchesCLIRender(t *testing.T) {
	now := time.Date(2026, 9, 27, 20, 0, 0, 0, time.UTC)
	nc := &models.Nowcast{
		GeneratedAt:  now,
		Location:     "Fort Wayne, IN",
		Headline:     "Dry next 6h",
		PrimaryPhase: models.PhaseNone,
		Intervals:    []models.PrecipInterval{},
	}
	nowPayload := &models.NowcastPayload{Location: "Fort Wayne, IN", Nowcast: nc}
	got, err := mustJSON(nowPayload)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := output.RenderNowcastTo(&buf, nowPayload, output.NowcastOptions{ForceJSON: true, Units: "imperial"}); err != nil {
		t.Fatal(err)
	}
	assertSameJSON(t, got, buf.Bytes())

	aqi := 42
	pm := 8.4
	aqPayload := &models.AirQualityPayload{
		Location:       "Fort Wayne, IN",
		Latitude:       41.08,
		Longitude:      -85.14,
		FetchedAt:      now,
		AirQuality:     &models.AirQuality{AQI: &aqi, Category: "Good", PM25: &pm},
		HealthAdvisory: models.EPAHealthAdvisory(aqi),
	}
	got, err = mustJSON(aqPayload)
	if err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := output.RenderAQITo(&buf, aqPayload, output.AQIOptions{ForceJSON: true}); err != nil {
		t.Fatal(err)
	}
	assertSameJSON(t, got, buf.Bytes())
	var aqMap map[string]json.RawMessage
	if err := json.Unmarshal(got, &aqMap); err != nil {
		t.Fatal(err)
	}
	if _, ok := aqMap["air_quality"]; !ok {
		t.Fatal("aqi JSON missing air_quality")
	}

	loc := location.Location{DisplayName: "Fort Wayne, IN", Lat: 41.08, Lon: -85.14}
	when := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	got, err = fetchDirect(context.Background(), "astro", loc, nil, when)
	if err != nil {
		t.Fatal(err)
	}
	astronomy, err := astro.Calculate(loc.Lat, loc.Lon, when)
	if err != nil {
		t.Fatal(err)
	}
	astroPayload := &models.AstroPayload{
		Location:     loc.DisplayName,
		Latitude:     loc.Lat,
		Longitude:    loc.Lon,
		CalculatedAt: when,
		Astronomy:    astronomy,
	}
	buf.Reset()
	if err := output.RenderAstroTo(&buf, astroPayload, output.AstroOptions{ForceJSON: true}); err != nil {
		t.Fatal(err)
	}
	assertSameJSON(t, got, buf.Bytes())
	var astroMap map[string]json.RawMessage
	if err := json.Unmarshal(got, &astroMap); err != nil {
		t.Fatal(err)
	}
	if _, ok := astroMap["astronomy"]; !ok {
		t.Fatal("astro JSON missing astronomy")
	}
}

func TestRadarImageMaxBytes(t *testing.T) {
	if RadarImageMaxBytes != 256<<10 {
		t.Fatalf("RadarImageMaxBytes = %d, want %d", RadarImageMaxBytes, 256<<10)
	}
	// Largest PNG whose base64 form is exactly the cap. One more byte is over.
	fit := 196608
	if base64.StdEncoding.EncodedLen(fit) != RadarImageMaxBytes {
		t.Fatalf("EncodedLen(%d) = %d, want %d", fit, base64.StdEncoding.EncodedLen(fit), RadarImageMaxBytes)
	}
	meta := sampleRadarOut()
	raw, err := radarToolJSON(radarShot{Out: meta, PNG: bytes.Repeat([]byte{1}, fit)}, true)
	if err != nil {
		t.Fatal(err)
	}
	var under map[string]any
	if err := json.Unmarshal(raw, &under); err != nil {
		t.Fatal(err)
	}
	frames, _ := under["frames"].([]any)
	if len(frames) != 1 {
		t.Fatalf("boundary fit frames = %d", len(frames))
	}
	_, err = radarToolJSON(radarShot{Out: meta, PNG: bytes.Repeat([]byte{1}, fit+1)}, true)
	assertPayloadTooLarge(t, err)
	_, err = radarToolJSON(radarShot{Out: meta, PNG: bytes.Repeat([]byte{2}, RadarImageMaxBytes+1)}, true)
	assertPayloadTooLarge(t, err)
}

func TestRadarToolMetadataAndCap(t *testing.T) {
	s := testServer(t)
	png := []byte("one-frame-png")
	s.radar = func(context.Context, location.Location, radar.Options, string) (radarShot, error) {
		out := sampleRadarOut()
		out.ImageBase64 = "SHOULD_NOT_LEAK"
		out.Frames = make([]radar.JSONFrame, 6)
		for i := range out.Frames {
			out.Frames[i] = radar.JSONFrame{ValidTime: "loop", ImageBase64: "frame-loop"}
		}
		return radarShot{Out: out, PNG: png}, nil
	}
	_, cs := connectMCP(t, s)
	res := call(t, cs, "wx_radar", map[string]any{"location": "41.07,-85.14"})
	if res.IsError {
		t.Fatalf("default: %s", resultText(res))
	}
	text := resultText(res)
	if strings.Contains(text, "image_base64") || strings.Contains(text, "frames") || strings.Contains(text, "SHOULD_NOT_LEAK") || strings.Contains(text, "frame-loop") {
		t.Fatalf("default leaked image or frames: %s", text)
	}
	var meta map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &meta); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"product", "product_label", "stations", "bbox", "center", "radius_km", "valid_time", "location", "is_composite", "raw"} {
		if _, ok := meta[key]; !ok {
			t.Errorf("default missing %s in %s", key, text)
		}
	}
	if _, ok := meta["image_base64"]; ok {
		t.Fatal("image_base64 key present")
	}
	if _, ok := meta["frames"]; ok {
		t.Fatal("frames key present")
	}

	res = call(t, cs, "wx_radar", map[string]any{"location": "41.07,-85.14", "include_image": true})
	if res.IsError {
		t.Fatalf("under cap: %s", resultText(res))
	}
	var with map[string]any
	if err := json.Unmarshal([]byte(resultText(res)), &with); err != nil {
		t.Fatal(err)
	}
	frames, _ := with["frames"].([]any)
	if len(frames) != 1 {
		t.Fatalf("frames = %d, want 1 (%s)", len(frames), resultText(res))
	}
	if with["image_base64"] != base64.StdEncoding.EncodeToString(png) {
		t.Fatalf("image_base64 = %v", with["image_base64"])
	}
	one, _ := frames[0].(map[string]any)
	if one["image_base64"] != with["image_base64"] {
		t.Fatalf("frame image differs from top-level")
	}
	if with["product"] != "composite-reflectivity" {
		t.Fatalf("product = %v", with["product"])
	}

	s.radar = func(context.Context, location.Location, radar.Options, string) (radarShot, error) {
		return radarShot{Out: sampleRadarOut(), PNG: bytes.Repeat([]byte{9}, RadarImageMaxBytes+1)}, nil
	}
	_, cs = connectMCP(t, s)
	res = call(t, cs, "wx_radar", map[string]any{"location": "41.07,-85.14", "include_image": true})
	assertCode(t, res, "payload_too_large")
	if strings.Contains(resultText(res), "image_base64") {
		t.Fatalf("over-cap result leaked image: %s", resultText(res))
	}

	s.radar = func(context.Context, location.Location, radar.Options, string) (radarShot, error) {
		t.Fatal("radar fetch ran for an invalid location")
		return radarShot{}, nil
	}
	_, cs = connectMCP(t, s)
	res = call(t, cs, "wx_radar", map[string]any{"location": "91,0", "include_image": true})
	assertCode(t, res, "invalid_location")
}

func TestClassifyUpstreamErrorStatus(t *testing.T) {
	err := classify(errString("nowcast: upstream error 502: bad gateway"))
	var te *toolError
	if !errors.As(err, &te) {
		t.Fatalf("%v", err)
	}
	if te.Code != "upstream_http" {
		t.Fatalf("code = %s", te.Code)
	}
	err = classify(&toolError{Code: "payload_too_large", Message: "radar image exceeds cap"})
	if !errors.As(err, &te) || te.Code != "payload_too_large" {
		t.Fatalf("classify remapped payload_too_large: %v", err)
	}
}

func sampleRadarOut() radar.JSONRadarOutput {
	return radar.JSONRadarOutput{
		Product:      "composite-reflectivity",
		ProductLabel: "Composite Reflectivity",
		Location:     "41.07, -85.14",
		ValidTime:    "2026-10-31T18:00:00Z",
		RadiusKM:     200,
		Raw:          true,
		IsComposite:  true,
		Stations:     []string{"KIWX"},
		BBox:         &radar.JSONBBox{MinLat: 40, MinLon: -86, MaxLat: 42, MaxLon: -84},
		Center:       &radar.JSONCenter{Lat: 41.07, Lon: -85.14},
	}
}

func assertPayloadTooLarge(t *testing.T, err error) {
	t.Helper()
	var te *toolError
	if !errors.As(err, &te) || te.Code != "payload_too_large" {
		t.Fatalf("err = %v, want payload_too_large", err)
	}
	if te.Message == "" {
		t.Fatal("empty payload_too_large message")
	}
}

func assertSameJSON(t *testing.T, got, want []byte) {
	t.Helper()
	var a, b any
	if err := json.Unmarshal(got, &a); err != nil {
		t.Fatalf("got: %v", err)
	}
	if err := json.Unmarshal(want, &b); err != nil {
		t.Fatalf("want: %v", err)
	}
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	if string(aj) != string(bj) {
		t.Fatalf("json mismatch\n got %s\n want %s", aj, bj)
	}
}
