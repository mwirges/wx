package present

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mwirges/wx/linux/internal/fixture"
	"github.com/mwirges/wx/linux/internal/pixmap"
	"github.com/mwirges/wx/linux/internal/prefs"
)

func TestMenuBarFormatsMatchMac(t *testing.T) {
	if got := StatusText(fixture.Payload, "imperial", "compact", ""); got != "🔴 72°" {
		t.Fatalf("compact %q", got)
	}
	if got := StatusText(fixture.Payload, "imperial", "standard", ""); got != " 🔴 72°" {
		t.Fatalf("standard %q", got)
	}
	if got := StatusText(fixture.Payload, "imperial", "tactical", ""); got != "🔴 [FWA] 72° ↘12mph" {
		t.Fatalf("tactical %q", got)
	}
	if got := StatusText(fixture.Quiet, "imperial", "compact", ""); got != "68°" {
		t.Fatalf("quiet compact %q", got)
	}
	if got := StatusText(fixture.Quiet, "imperial", "standard", ""); got != " 68°" {
		t.Fatalf("quiet standard %q", got)
	}
	if got := StatusText(fixture.Watch, "imperial", "compact", ""); got != "🟠 80°" {
		t.Fatalf("watch %q", got)
	}
	if got := StatusText(nil, "imperial", "compact", "missing"); got != " wx?" {
		t.Fatalf("error %q", got)
	}
}

func TestWarningPipBeatsWatchAndDoesNotBlink(t *testing.T) {
	both := map[string]any{
		"conditions": map[string]any{"temperature_f": 70.0, "station": "KFWA"},
		"alerts": []any{
			map[string]any{"event": "Wind Advisory", "severity": "Minor"},
			map[string]any{"event": "Tornado Warning", "severity": "Extreme"},
		},
	}
	first := StatusText(both, "imperial", "compact", "")
	second := StatusText(both, "imperial", "compact", "")
	if first != second || len(first) < 4 || first[:len("🔴 ")] != "🔴 " {
		t.Fatalf("pip %q", first)
	}
	if contains(first, "⭕") {
		t.Fatalf("blink %q", first)
	}
}

func TestStationTagAndOpenMeteoFallback(t *testing.T) {
	if got := TacticalStationTag(fixture.Payload); got != "FWA" {
		t.Fatalf("tag %q", got)
	}
	openMeteo := map[string]any{"conditions": map[string]any{"station": "OPENMETEO", "location": "Fort Wayne, IN", "temperature_f": 70.0}}
	if got := TacticalStationTag(openMeteo); got != "FOR" {
		t.Fatalf("open-meteo tag %q", got)
	}
	if got := TacticalStatus(openMeteo, "imperial"); got != "[FOR] 70°" {
		t.Fatalf("open-meteo status %q", got)
	}
}

func TestMetricTempAndWind(t *testing.T) {
	if got := DisplayTemp(fixture.Payload, "metric"); got != "22°" {
		t.Fatalf("metric temp %q", got)
	}
	if got := TacticalWind(fixture.Payload, "metric"); got != "↘19km/h" {
		t.Fatalf("metric wind %q", got)
	}
	if got := WindLine(fixture.Payload, "imperial"); got != "Wind NW 12 mph" {
		t.Fatalf("wind line %q", got)
	}
	if got := WindLine(fixture.Payload, "metric"); got != "Wind NW 19 km/h" {
		t.Fatalf("metric wind line %q", got)
	}
}

func TestClockLabelKeepsMinutes(t *testing.T) {
	five := ClockLabel(map[string]any{"start_time": "2026-10-10T21:00:00Z"})
	quarter := ClockLabel(map[string]any{"start_time": "2026-10-10T21:15:00Z"})
	if five == quarter {
		t.Fatalf("same clock label %q", five)
	}
	if !contains(five, ":00") || !contains(quarter, ":15") {
		t.Fatalf("labels %q %q", five, quarter)
	}
}

func TestLocalStampUsesLocalClock(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	if got := formatStamp("2026-10-10T21:20:00Z", loc, "Jan 2, 3:04 PM"); got != "Oct 10, 5:20 PM" {
		t.Fatalf("stamp %q", got)
	}
	if got := LocalStamp("not-a-time"); got != "not-a-time" {
		t.Fatalf("passthrough %q", got)
	}
}

func TestAdvisoryStatementIsAmber(t *testing.T) {
	payload := map[string]any{"conditions": map[string]any{"temperature_f": 60.0}, "alerts": []any{map[string]any{"event": "Special Weather Statement"}}}
	if PipKind(payload) != "watch" {
		t.Fatalf("kind %q", PipKind(payload))
	}
	if PipGlyph(payload) != "🟠 " {
		t.Fatalf("glyph %q", PipGlyph(payload))
	}
}

func TestPixmapPipIsSteadyColor(t *testing.T) {
	_, _, warn := pixmap.RenderStatus("72°", "warning")
	_, _, watch := pixmap.RenderStatus("[FWA] 72° ↘12mph", "watch")
	_, _, quiet := pixmap.RenderStatus("72°", "")
	if !pixmap.HasPipColor(warn, "warning") {
		t.Fatal("warning pixmap has no red pip")
	}
	if !pixmap.HasPipColor(watch, "watch") {
		t.Fatal("watch pixmap has no amber pip")
	}
	if pixmap.HasPipColor(quiet, "warning") {
		t.Fatal("quiet pixmap has a warning pip")
	}
	if len(warn) <= 16 {
		t.Fatalf("pixmap length %d", len(warn))
	}
}

func TestUserConfigPathIsNotTheDefaultUnderOverride(t *testing.T) {
	defer fixture.GuardHome(t)()
	t.Setenv("WX_CONFIG", "/tmp/wx-linux-not-home.json")
	if prefs.ConfigPath() != "/tmp/wx-linux-not-home.json" {
		t.Fatalf("path %s", prefs.ConfigPath())
	}
	if filepath.Clean(prefs.ConfigPath()) == filepath.Join(homeStub(), ".config", "wx", "config.json") {
		t.Fatal("override fell through to the user config")
	}
}

func contains(text, needle string) bool {
	return len(text) >= len(needle) && (text == needle || len(needle) == 0 || indexOf(text, needle) >= 0)
}

func indexOf(text, needle string) int {
	for i := 0; i+len(needle) <= len(text); i++ {
		if text[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func homeStub() string { return "/home" }
