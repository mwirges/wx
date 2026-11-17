package monitor

import (
	"context"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/radar"
)

// ── padToWidth ────────────────────────────────────────────────────────────────

func TestPadToWidth_Plain(t *testing.T) {
	s := padToWidth("hello", 10)
	vis := lipgloss.Width(s)
	if vis != 10 {
		t.Errorf("padToWidth visual width = %d, want 10", vis)
	}
	if !strings.HasPrefix(s, "hello") {
		t.Errorf("padToWidth should preserve original content, got %q", s)
	}
}

func TestPadToWidth_AlreadyWide(t *testing.T) {
	s := padToWidth("hello world", 5)
	// Should not truncate — returns as-is when already >= w
	if s != "hello world" {
		t.Errorf("padToWidth should not truncate, got %q", s)
	}
}

func TestPadToWidth_AnsiString(t *testing.T) {
	// lipgloss-styled string: visual width != byte length
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("hi")
	padded := padToWidth(styled, 10)
	vis := lipgloss.Width(padded)
	if vis != 10 {
		t.Errorf("ANSI padToWidth visual width = %d, want 10", vis)
	}
}

// ── normalizeLines ────────────────────────────────────────────────────────────

func TestNormalizeLines_Pad(t *testing.T) {
	lines := []string{"a", "b"}
	result := normalizeLines(lines, 5, 10)
	if len(result) != 5 {
		t.Errorf("len = %d, want 5", len(result))
	}
}

func TestNormalizeLines_Truncate(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	result := normalizeLines(lines, 3, 10)
	if len(result) != 3 {
		t.Errorf("len = %d, want 3", len(result))
	}
}

func TestNormalizeLines_Exact(t *testing.T) {
	lines := []string{"a", "b", "c"}
	result := normalizeLines(lines, 3, 10)
	if len(result) != 3 {
		t.Errorf("len = %d, want 3", len(result))
	}
}

// ── truncateStr ───────────────────────────────────────────────────────────────

func TestTruncateStr_ShortString(t *testing.T) {
	s := truncateStr("hello", 20)
	if s != "hello" {
		t.Errorf("truncateStr should not modify short string, got %q", s)
	}
}

func TestTruncateStr_LongString(t *testing.T) {
	long := strings.Repeat("x", 100)
	result := truncateStr(long, 10)
	if lipgloss.Width(result) > 10 {
		t.Errorf("truncateStr result width = %d, want <= 10", lipgloss.Width(result))
	}
}

// ── renderScrollHeader ────────────────────────────────────────────────────────

func TestRenderScrollHeader_NoScroll(t *testing.T) {
	// All periods fit: no arrows
	s := renderScrollHeader(0, 3, 5, 40)
	if strings.Contains(s, "▲") || strings.Contains(s, "▼") {
		t.Errorf("no-scroll header should have no arrows, got %q", s)
	}
}

func TestRenderScrollHeader_CanScrollDown(t *testing.T) {
	// More periods than visible, at top
	s := renderScrollHeader(0, 10, 5, 60)
	if !strings.Contains(s, "▼") {
		t.Errorf("expected ▼ arrow when can scroll down, got %q", s)
	}
	if strings.Contains(s, "▲") {
		t.Errorf("should not have ▲ when at top, got %q", s)
	}
}

func TestRenderScrollHeader_CanScrollUp(t *testing.T) {
	// More periods than visible, scrolled down
	s := renderScrollHeader(5, 10, 5, 60)
	if !strings.Contains(s, "▲") {
		t.Errorf("expected ▲ arrow when can scroll up, got %q", s)
	}
}

func TestRenderScrollHeader_AtBottom(t *testing.T) {
	// Scrolled to the bottom
	s := renderScrollHeader(5, 10, 5, 60)
	// offset(5) + visible(5) = 10 = total → no down arrow
	if strings.Contains(s, "▼") {
		t.Errorf("should not have ▼ when at bottom, got %q", s)
	}
}

// ── View ─────────────────────────────────────────────────────────────────────

func TestView_ZeroSizeReturnsLoading(t *testing.T) {
	m := New(MonitorConfig{Imperial: true}, testLoc())
	// width/height = 0 → loading message
	v := m.View()
	if v == "" {
		t.Error("View should return loading message when size is 0")
	}
}

func TestView_WithSize(t *testing.T) {
	m := New(MonitorConfig{Imperial: true}, testLoc())
	m.width = 100
	m.height = 30

	v := m.View()
	if v == "" {
		t.Error("View should return non-empty string with valid size")
	}
	// Should contain the location name
	if !strings.Contains(v, "Kansas City, MO") {
		t.Errorf("View should contain location name, got:\n%s", v)
	}
}

func TestView_HelpBar_Normal(t *testing.T) {
	m := New(MonitorConfig{Imperial: true}, testLoc())
	m.width = 100
	m.height = 30

	v := m.renderHelpBar()
	if !strings.Contains(v, "refresh") {
		t.Errorf("help bar should contain 'refresh', got %q", v)
	}
	if !strings.Contains(v, "quit") {
		t.Errorf("help bar should contain 'quit', got %q", v)
	}
}

func TestView_HelpBar_InputMode(t *testing.T) {
	m := New(MonitorConfig{Imperial: true}, testLoc())
	m.inputMode = true

	v := m.renderHelpBar()
	if !strings.Contains(v, "confirm") {
		t.Errorf("input mode help bar should contain 'confirm', got %q", v)
	}
	if !strings.Contains(v, "cancel") {
		t.Errorf("input mode help bar should contain 'cancel', got %q", v)
	}
}

// ── max/min ───────────────────────────────────────────────────────────────────

func TestMax(t *testing.T) {
	if max(3, 5) != 5 {
		t.Error("max(3,5) should be 5")
	}
	if max(7, 2) != 7 {
		t.Error("max(7,2) should be 7")
	}
}

func TestMin(t *testing.T) {
	if min(3, 5) != 3 {
		t.Error("min(3,5) should be 3")
	}
	if min(7, 2) != 2 {
		t.Error("min(7,2) should be 2")
	}
}

func TestRenderConditions_WithAirQuality(t *testing.T) {
	temp := 20.0
	aqi := 45
	pm25 := 8.5
	m := New(MonitorConfig{Imperial: true}, testLoc())
	m.conditions = &models.CurrentConditions{
		TempC:         &temp,
		ConditionCode: "clear-day",
		Description:   "Clear",
		AirQuality: &models.AirQuality{
			AQI:      &aqi,
			Category: "Good",
			PM25:     &pm25,
		},
	}

	lines := m.renderConditions(80)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Air:") {
		t.Errorf("expected conditions to contain 'Air:', got:\n%s", joined)
	}
	if !strings.Contains(joined, "45") {
		t.Errorf("expected conditions to contain AQI '45', got:\n%s", joined)
	}
	if !strings.Contains(joined, "Good") {
		t.Errorf("expected conditions to contain 'Good', got:\n%s", joined)
	}
	if !strings.Contains(joined, "PM2.5: 8.5") {
		t.Errorf("expected conditions to contain 'PM2.5: 8.5', got:\n%s", joined)
	}
}

func TestRenderConditions_WithNowcast(t *testing.T) {
	temp := 18.0
	m := New(MonitorConfig{Imperial: true}, testLoc())
	m.conditions = &models.CurrentConditions{
		TempC:         &temp,
		ConditionCode: "rain",
		Description:   "Rain",
		Nowcast: &models.Nowcast{
			IsActivePrecip: true,
			Headline:       "Rain stopping in ~25 min",
			Intervals: []models.PrecipInterval{
				{RateInH: 0.15},
				{RateInH: 0.08},
			},
		},
	}

	lines := m.renderConditions(80)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Precip:") {
		t.Errorf("expected conditions to contain 'Precip:', got:\n%s", joined)
	}
	if !strings.Contains(joined, "Rain stopping in ~25 min") {
		t.Errorf("expected conditions to contain nowcast headline, got:\n%s", joined)
	}
}

func TestRenderConditions_WithClimate(t *testing.T) {
	temp := 22.0
	depF := 5.4
	depC := 3.0
	m := New(MonitorConfig{Imperial: true}, testLoc())
	m.conditions = &models.CurrentConditions{
		TempC:         &temp,
		ConditionCode: "clear-day",
		Description:   "Sunny",
		Climate: &models.ClimateReport{
			TodayNormals: models.DailyNormals{
				NormalHighF: 72.0,
				NormalLowF:  50.0,
			},
			Departure: &models.ClimateDeparture{
				DepartureCurrentF: &depF,
				DepartureCurrentC: &depC,
				Summary:           "+5.4°F Departure (Above Normal)",
			},
		},
	}

	lines := m.renderConditions(80)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Climate:") {
		t.Errorf("expected conditions to contain 'Climate:', got:\n%s", joined)
	}
	if !strings.Contains(joined, "+5.4°F") {
		t.Errorf("expected conditions to contain departure '+5.4°F', got:\n%s", joined)
	}
	if !strings.Contains(joined, "Above Normal") {
		t.Errorf("expected conditions to contain summary 'Above Normal', got:\n%s", joined)
	}
	if !strings.Contains(joined, "72°F / 50°F") {
		t.Errorf("expected conditions to contain normals '72°F / 50°F', got:\n%s", joined)
	}
}

func TestRenderSplit_HasColumnDividerAndExactRowCount(t *testing.T) {
	m := New(MonitorConfig{Imperial: true}, testLoc())
	m.width = 80
	m.height = 24
	m.radarVisible = true

	split := m.renderSplit(20)
	rows := strings.Split(split, "\n")
	if len(rows) != 20 {
		t.Fatalf("renderSplit row count = %d, want 20", len(rows))
	}
	for i, row := range rows {
		if !strings.Contains(row, "│") {
			t.Fatalf("row %d missing column divider │: %q", i, row)
		}
		if got := lipgloss.Width(row); got > 80 {
			t.Fatalf("row %d visual width = %d, want <= 80", i, got)
		}
	}
}

type stubSplitRadarProvider struct{}

func (stubSplitRadarProvider) Name() string                      { return "stub" }
func (stubSplitRadarProvider) Supports(_ location.Location) bool { return true }
func (stubSplitRadarProvider) CurrentFrame(_ context.Context, _ location.Location, opts radar.Options, _ *cache.Cache) (*radar.Frame, error) {
	return &radar.Frame{
		Img:         image.NewRGBA(image.Rect(0, 0, 64, 64)),
		ValidTime:   time.Date(2026, 10, 25, 18, 0, 0, 0, time.UTC),
		Product:     opts.Product,
		IsComposite: true,
	}, nil
}
func (stubSplitRadarProvider) RecentFrames(ctx context.Context, loc location.Location, opts radar.Options, _ int, ch *cache.Cache) ([]*radar.Frame, error) {
	f, err := stubSplitRadarProvider{}.CurrentFrame(ctx, loc, opts, ch)
	if err != nil {
		return nil, err
	}
	return []*radar.Frame{f}, nil
}

func TestView_SplitWithLiveRadarNeverExceedsTerminalWidth(t *testing.T) {
	loc := location.Location{
		DisplayName: "Truth or Consequences, New Mexico",
		CountryCode: "US",
		Lat:         33.1284,
		Lon:         -107.2528,
	}
	cfg := MonitorConfig{
		Imperial:  true,
		RadarProv: stubSplitRadarProvider{},
	}

	for _, width := range []int{80, 81, 60} {
		m := New(cfg, loc)
		m.width = width
		m.height = 24
		m.radarVisible = true

		msg := fetchRadarCmd(cfg, loc, width, 24)().(radarMsg)
		if msg.err != nil {
			t.Fatalf("width=%d fetchRadarCmd failed: %v", width, msg.err)
		}
		m.radarLines = msg.lines
		m.radarLoading = false

		view := m.View()
		rows := strings.Split(view, "\n")
		if len(rows) != 24 {
			t.Fatalf("width=%d View() row count = %d, want 24", width, len(rows))
		}
		for i, row := range rows {
			clean := strings.TrimSuffix(row, "\x1b[K")
			if got := lipgloss.Width(clean); got > width {
				t.Fatalf("width=%d row %d visual width = %d, want <= %d: %q", width, i, got, width, clean)
			}
		}

		// Also verify shrinking terminal before next radarMsg arrives clamps stale radarLines.
		m.width = width - 12
		shrunk := m.View()
		for i, row := range strings.Split(shrunk, "\n") {
			clean := strings.TrimSuffix(row, "\x1b[K")
			if got := lipgloss.Width(clean); got > m.width {
				t.Fatalf("shrunk width=%d row %d visual width = %d, want <= %d", m.width, i, got, m.width)
			}
		}
	}
}



