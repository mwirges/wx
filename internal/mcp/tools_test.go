package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mwirges/wx/internal/location"
)

func TestWeatherToolCatalog(t *testing.T) {
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
	for _, name := range []string{
		"wx_server_info", "wx_conditions", "wx_forecast", "wx_hourly", "wx_alerts",
		"wx_nowcast", "wx_aqi", "wx_astro", "wx_radar",
		"wx_climate", "wx_sounding", "wx_outlook", "wx_spc",
	} {
		if !got[name] {
			t.Errorf("missing tool %s in %v", name, got)
		}
	}
}

func TestWeatherInvalidLocation(t *testing.T) {
	s := testServer(t)
	s.weather = func(context.Context, string, location.Location, string, int) (json.RawMessage, error) {
		t.Fatal("weather fetch ran for an invalid location")
		return nil, nil
	}
	_, cs := connectMCP(t, s)
	for _, loc := range []string{"91,0", "foo,bar", "41.07,", "41.07 -85.14"} {
		res := call(t, cs, "wx_conditions", map[string]any{"location": loc})
		assertCode(t, res, "invalid_location")
	}
}

func TestWeatherRateLimitAndUpstream(t *testing.T) {
	s := testServer(t)
	s.slots = make(chan struct{}, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	// holding is true only while the first conditions call is inside weather.
	// A second entry sets slipped and returns success, not HTTP 429, so the
	// overlap assertion fails if acquire is removed or the slot is unbounded.
	var holding atomic.Bool
	var slipped atomic.Bool
	s.weather = func(ctx context.Context, kind string, loc location.Location, units string, hours int) (json.RawMessage, error) {
		if holding.Load() {
			slipped.Store(true)
			return json.RawMessage(`{"slipped":true}`), nil
		}
		switch kind {
		case "conditions":
			holding.Store(true)
			defer holding.Store(false)
			once.Do(func() { close(started) })
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return json.RawMessage(`{"ok":true}`), nil
		case "forecast":
			return nil, errString("nws: forecast: HTTP 429: slow down")
		case "hourly":
			if hours != 6 {
				t.Errorf("hours = %d, want 6", hours)
			}
			return nil, errString("nws: forecast: HTTP 503")
		case "alerts":
			return json.RawMessage(`[]`), nil
		default:
			return nil, errString("unexpected " + kind)
		}
	}

	_, cs1 := connectMCP(t, s)
	_, cs2 := connectMCP(t, s)
	errCh := make(chan error, 1)
	go func() {
		_, err := cs1.CallTool(context.Background(), &sdkmcp.CallToolParams{
			Name:      "wx_conditions",
			Arguments: map[string]any{"location": "41.07,-85.14"},
		})
		errCh <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first call did not reach the weather func")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := cs2.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "wx_forecast",
		Arguments: map[string]any{"location": "41.07,-85.14"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if slipped.Load() {
		t.Fatalf("overlap wx_forecast entered weather while the first call held the slot (cap did not reject before fetch); body=%s", resultText(res))
	}
	assertCode(t, res, "rate_limited")
	if !strings.Contains(resultText(res), "tool concurrency limit reached") {
		t.Fatalf("overlap rate_limited was not errBusy: %s", resultText(res))
	}
	close(release)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}

	res = call(t, cs2, "wx_forecast", map[string]any{"location": "41.07,-85.14"})
	assertCode(t, res, "rate_limited")
	if !strings.Contains(resultText(res), "HTTP 429") {
		t.Fatalf("post-release forecast was not the HTTP 429 classify path: %s", resultText(res))
	}
	res = call(t, cs2, "wx_hourly", map[string]any{"location": "41.07,-85.14", "hours": 6})
	assertCode(t, res, "upstream_http")
	if !strings.Contains(resultText(res), "HTTP 503") {
		t.Fatalf("hourly was not the HTTP 503 path: %s", resultText(res))
	}
	res = call(t, cs2, "wx_alerts", map[string]any{"location": "41.07,-85.14"})
	if res.IsError {
		t.Fatalf("alerts error: %s", resultText(res))
	}
	if strings.TrimSpace(resultText(res)) != "[]" {
		t.Fatalf("alerts = %s", resultText(res))
	}
}

func TestWeatherTimeoutAndBadArgs(t *testing.T) {
	s := testServer(t)
	s.weather = func(context.Context, string, location.Location, string, int) (json.RawMessage, error) {
		return nil, context.DeadlineExceeded
	}
	_, cs := connectMCP(t, s)
	res := call(t, cs, "wx_conditions", map[string]any{"location": "41.07,-85.14"})
	assertCode(t, res, "upstream_timeout")

	res = call(t, cs, "wx_hourly", map[string]any{"location": "41.07,-85.14", "hours": 999})
	assertCode(t, res, "internal")
	res = call(t, cs, "wx_conditions", map[string]any{"location": "41.07,-85.14", "units": "furlongs"})
	assertCode(t, res, "internal")
}

type errString string

func (e errString) Error() string { return string(e) }

func connectMCP(t *testing.T, s *Server) (*sdkmcp.Client, *sdkmcp.ClientSession) {
	t.Helper()
	ctx := context.Background()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "wx-test", Version: "0"}, nil)
	st, ct := sdkmcp.NewInMemoryTransports()
	if _, err := s.mcp.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return client, cs
}

func call(t *testing.T, cs *sdkmcp.ClientSession, name string, args map[string]any) *sdkmcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return res
}

func assertCode(t *testing.T, res *sdkmcp.CallToolResult, code string) {
	t.Helper()
	if res == nil || !res.IsError {
		t.Fatalf("isError=false, want code %s; text %s", code, resultText(res))
	}
	text := resultText(res)
	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("result is not JSON code/message: %s", text)
	}
	if payload.Code != code {
		t.Fatalf("code = %q, want %q (%s)", payload.Code, code, text)
	}
	if payload.Message == "" {
		t.Fatalf("empty message: %s", text)
	}
}

func resultText(res *sdkmcp.CallToolResult) string {
	if res == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}
