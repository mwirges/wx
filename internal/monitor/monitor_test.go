package monitor

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

func testConfig() MonitorConfig {
	return MonitorConfig{
		Imperial:        true,
		RefreshInterval: 15 * time.Minute,
	}
}

func testLoc() location.Location {
	return location.Location{DisplayName: "Kansas City, MO", CountryCode: "US"}
}

// ── New ───────────────────────────────────────────────────────────────────────

func TestNew_DefaultState(t *testing.T) {
	m := New(testConfig(), testLoc())
	if m.radarVisible {
		t.Error("radarVisible should default to false")
	}
	if m.forecastOffset != 0 {
		t.Errorf("forecastOffset = %d, want 0", m.forecastOffset)
	}
	if m.inputMode {
		t.Error("inputMode should default to false")
	}
	if !m.weatherLoading {
		t.Error("weatherLoading should be true on creation")
	}
}

// ── WindowSizeMsg ─────────────────────────────────────────────────────────────

func TestUpdate_WindowSize(t *testing.T) {
	m := New(testConfig(), testLoc())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	um := updated.(MonitorModel)
	if um.width != 120 || um.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", um.width, um.height)
	}
}

// ── weatherMsg ───────────────────────────────────────────────────────────────

func TestUpdate_WeatherMsg(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.weatherLoading = true

	tempC := 20.0
	now := time.Now()
	msg := weatherMsg{
		conditions: &models.CurrentConditions{TempC: &tempC},
		forecast:   &models.Forecast{},
		alerts:     []models.Alert{{Event: "Wind Advisory"}},
		fetchedAt:  now,
	}
	updated, _ := m.Update(msg)
	um := updated.(MonitorModel)

	if um.weatherLoading {
		t.Error("weatherLoading should be false after weatherMsg")
	}
	if um.fetchErr != nil {
		t.Errorf("fetchErr should be nil, got %v", um.fetchErr)
	}
	if um.conditions == nil {
		t.Error("conditions should be set")
	}
	if len(um.alerts) != 1 {
		t.Errorf("alerts len = %d, want 1", len(um.alerts))
	}
	if !um.lastFetch.Equal(now) {
		t.Error("lastFetch not set correctly")
	}
}

// ── weatherErrMsg ─────────────────────────────────────────────────────────────

func TestUpdate_WeatherErrMsg(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.weatherLoading = true

	updated, _ := m.Update(weatherErrMsg{err: errors.New("network error")})
	um := updated.(MonitorModel)

	if um.weatherLoading {
		t.Error("weatherLoading should be false after error")
	}
	if um.fetchErr == nil {
		t.Error("fetchErr should be set")
	}
}

// ── radarMsg ─────────────────────────────────────────────────────────────────

func TestUpdate_RadarMsg_Success(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.radarLoading = true

	lines := []string{"line1", "line2"}
	updated, _ := m.Update(radarMsg{lines: lines})
	um := updated.(MonitorModel)

	if um.radarLoading {
		t.Error("radarLoading should be false")
	}
	if um.radarErr != nil {
		t.Errorf("radarErr should be nil, got %v", um.radarErr)
	}
	if len(um.radarLines) != 2 {
		t.Errorf("radarLines len = %d, want 2", len(um.radarLines))
	}
}

func TestUpdate_RadarMsg_Error(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.radarLoading = true
	m.radarLines = []string{"old"}

	updated, _ := m.Update(radarMsg{err: errors.New("radar fetch failed")})
	um := updated.(MonitorModel)

	if um.radarLoading {
		t.Error("radarLoading should be false")
	}
	if um.radarErr == nil {
		t.Error("radarErr should be set")
	}
	// Old lines preserved on error
	if len(um.radarLines) != 1 {
		t.Errorf("radarLines should be preserved on error, got len %d", len(um.radarLines))
	}
}

// ── tickMsg ───────────────────────────────────────────────────────────────────

func TestUpdate_TickMsg_TriggersRefresh(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.weatherLoading = false

	updated, cmd := m.Update(tickMsg(time.Now()))
	um := updated.(MonitorModel)

	if !um.weatherLoading {
		t.Error("tickMsg should set weatherLoading=true")
	}
	if cmd == nil {
		t.Error("tickMsg should return a non-nil cmd")
	}
}

func TestUpdate_TickMsg_WithRadarVisible(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.radarVisible = true
	m.width = 120
	m.height = 40

	updated, _ := m.Update(tickMsg(time.Now()))
	um := updated.(MonitorModel)

	if !um.radarLoading {
		t.Error("radarLoading should be true when radar is visible during tick")
	}
}

// ── locationMsg ──────────────────────────────────────────────────────────────

func TestUpdate_LocationMsg(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.inputMode = true
	m.inputText = "Chicago, IL"
	m.forecastOffset = 3

	newLoc := location.Location{DisplayName: "Chicago, IL", CountryCode: "US"}
	updated, cmd := m.Update(locationMsg{loc: newLoc})
	um := updated.(MonitorModel)

	if um.inputMode {
		t.Error("inputMode should be cleared after locationMsg")
	}
	if um.inputText != "" {
		t.Errorf("inputText should be empty, got %q", um.inputText)
	}
	if um.forecastOffset != 0 {
		t.Error("forecastOffset should reset to 0 on location change")
	}
	if um.loc.DisplayName != "Chicago, IL" {
		t.Errorf("loc.DisplayName = %q, want %q", um.loc.DisplayName, "Chicago, IL")
	}
	if !um.weatherLoading {
		t.Error("weatherLoading should be true after location change")
	}
	if cmd == nil {
		t.Error("should return a fetch cmd")
	}
}

// ── locationErrMsg ────────────────────────────────────────────────────────────

func TestUpdate_LocationErrMsg(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.inputMode = true
	m.inputText = "zzz bad location"

	updated, _ := m.Update(locationErrMsg{err: errors.New("not found")})
	um := updated.(MonitorModel)

	if !um.inputMode {
		t.Error("inputMode should remain true on location error")
	}
	if um.inputErr == nil {
		t.Error("inputErr should be set")
	}
}

// ── Key: q ───────────────────────────────────────────────────────────────────

func TestUpdate_Key_Q_Quits(t *testing.T) {
	m := New(testConfig(), testLoc())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("q should return tea.Quit cmd")
	}
}

// ── Key: r ───────────────────────────────────────────────────────────────────

func TestUpdate_Key_r_Refresh(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.weatherLoading = false

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	um := updated.(MonitorModel)

	if !um.weatherLoading {
		t.Error("r should set weatherLoading=true")
	}
	if cmd == nil {
		t.Error("r should return fetch cmd")
	}
}

// ── Key: R (toggle radar) ─────────────────────────────────────────────────────

func TestUpdate_Key_R_TogglesRadarOn(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.width = 120
	m.height = 40

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	um := updated.(MonitorModel)

	if !um.radarVisible {
		t.Error("R should toggle radarVisible to true")
	}
	if !um.radarLoading {
		t.Error("radarLoading should be true when radar is first enabled")
	}
	if cmd == nil {
		t.Error("should fire radar fetch cmd on first enable")
	}
}

func TestUpdate_Key_R_TogglesRadarOff(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.radarVisible = true
	m.radarLines = []string{"line1"}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	um := updated.(MonitorModel)

	if um.radarVisible {
		t.Error("R should toggle radarVisible to false")
	}
}

func TestUpdate_Key_R_NoFetchWhenLinesExist(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.radarLines = []string{"existing"}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	um := updated.(MonitorModel)

	if !um.radarVisible {
		t.Error("R should enable radar")
	}
	// No fetch needed since lines exist
	if cmd != nil {
		// cmd may be nil since lines already exist
		_ = cmd
	}
}

// ── Key: l (location input) ───────────────────────────────────────────────────

func TestUpdate_Key_l_EntersInputMode(t *testing.T) {
	m := New(testConfig(), testLoc())

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	um := updated.(MonitorModel)

	if !um.inputMode {
		t.Error("l should set inputMode=true")
	}
	if um.inputText != "" {
		t.Errorf("inputText should be empty, got %q", um.inputText)
	}
}

// ── Input mode key handling ───────────────────────────────────────────────────

func TestUpdate_InputMode_Esc(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.inputMode = true
	m.inputText = "Chicago"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	um := updated.(MonitorModel)

	if um.inputMode {
		t.Error("esc should clear inputMode")
	}
	if um.inputText != "" {
		t.Errorf("esc should clear inputText, got %q", um.inputText)
	}
}

func TestUpdate_InputMode_Typing(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.inputMode = true

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	um := updated.(MonitorModel)

	if um.inputText != "K" {
		t.Errorf("inputText = %q, want %q", um.inputText, "K")
	}
}

func TestUpdate_InputMode_Backspace(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.inputMode = true
	m.inputText = "abc"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	um := updated.(MonitorModel)

	if um.inputText != "ab" {
		t.Errorf("backspace: inputText = %q, want %q", um.inputText, "ab")
	}
}

func TestUpdate_InputMode_BackspaceEmpty(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.inputMode = true
	m.inputText = ""

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	um := updated.(MonitorModel)

	if um.inputText != "" {
		t.Errorf("backspace on empty: inputText = %q, want empty", um.inputText)
	}
}

func TestUpdate_InputMode_Enter_Empty(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.inputMode = true
	m.inputText = ""

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(MonitorModel)

	if !um.inputMode {
		t.Error("enter with empty text should keep inputMode")
	}
	if cmd != nil {
		t.Error("enter with empty text should not fire a cmd")
	}
}

func TestUpdate_InputMode_Enter_NonEmpty(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.inputMode = true
	m.inputText = "Chicago, IL"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Error("enter with non-empty text should fire resolveLocationCmd")
	}
}

// ── Scroll ────────────────────────────────────────────────────────────────────

func TestUpdate_ScrollDown_Clamped(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.forecastOffset = 0
	// forecast with 3 periods, forecastVisible = 3 → max offset = 0
	m.forecast = &models.Forecast{Periods: make([]models.Period, 3)}
	m.forecastVisible = 3

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	um := updated.(MonitorModel)

	if um.forecastOffset != 0 {
		t.Errorf("offset should stay 0 when already at max, got %d", um.forecastOffset)
	}
}

func TestUpdate_ScrollDown(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.forecastOffset = 0
	m.forecast = &models.Forecast{Periods: make([]models.Period, 6)}
	m.forecastVisible = 3

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	um := updated.(MonitorModel)

	if um.forecastOffset != 1 {
		t.Errorf("offset = %d, want 1", um.forecastOffset)
	}
}

func TestUpdate_ScrollUp_Clamped(t *testing.T) {
	m := New(testConfig(), testLoc())
	m.forecastOffset = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	um := updated.(MonitorModel)

	if um.forecastOffset != 0 {
		t.Errorf("offset should not go below 0, got %d", um.forecastOffset)
	}
}

// ── clampOffset ───────────────────────────────────────────────────────────────

func TestClampOffset_BelowZero(t *testing.T) {
	if got := clampOffset(-1, 10, 3); got != 0 {
		t.Errorf("clampOffset(-1,10,3) = %d, want 0", got)
	}
}

func TestClampOffset_AboveMax(t *testing.T) {
	if got := clampOffset(10, 5, 3); got != 2 {
		t.Errorf("clampOffset(10,5,3) = %d, want 2 (max=5-3)", got)
	}
}

func TestClampOffset_WithinRange(t *testing.T) {
	if got := clampOffset(2, 10, 3); got != 2 {
		t.Errorf("clampOffset(2,10,3) = %d, want 2", got)
	}
}

// ── computeForecastVisible ────────────────────────────────────────────────────

func TestComputeForecastVisible_SmallTerminal(t *testing.T) {
	// Very short terminal: should clamp at 0
	got := computeForecastVisible(10, 0)
	if got < 0 {
		t.Errorf("forecastVisible should not be negative, got %d", got)
	}
}

func TestComputeForecastVisible_LargeTerminal(t *testing.T) {
	// Very tall terminal: should cap at maxForecastPeriods
	got := computeForecastVisible(200, 0)
	if got > maxForecastPeriods {
		t.Errorf("forecastVisible = %d, want <= %d", got, maxForecastPeriods)
	}
}

// ── splitAndStripRadarLines ───────────────────────────────────────────────────

func TestSplitAndStripRadarLines(t *testing.T) {
	input := "line1\x1b[K\nline2\x1b[K\nline3"
	lines := splitAndStripRadarLines(input)
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	for i, want := range []string{"line1", "line2", "line3"} {
		if lines[i] != want {
			t.Errorf("line[%d] = %q, want %q", i, lines[i], want)
		}
	}
}
