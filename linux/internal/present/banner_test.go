package present

import (
	"strings"
	"testing"
)

func TestBannerRanksWarningsAheadOfWatches(t *testing.T) {
	payload := map[string]any{
		"alerts": []any{
			map[string]any{"event": "Wind Advisory", "severity": "Minor", "headline": "Breezy"},
			map[string]any{"event": "Flood Watch", "severity": "Moderate", "headline": "Flooding possible"},
			map[string]any{"event": "Special Weather Statement", "severity": "Extreme", "headline": "Not a warning"},
			map[string]any{"event": "Tornado Warning", "severity": "Severe", "expires": "2026-10-18T22:15:00Z"},
			map[string]any{"event": "Tornado Emergency", "severity": "Extreme", "headline": "Take cover"},
		},
	}
	ranked := RankedBannerAlerts(payload)
	if len(ranked) != 3 {
		t.Fatalf("ranked %d, want warning, emergency, watch", len(ranked))
	}
	if ranked[0].Kind != BannerWarning || !strings.Contains(strings.ToLower(AsString(ranked[0].Alert["event"])), "emergency") {
		t.Fatalf("lead %+v", ranked[0])
	}
	if ranked[1].Kind != BannerWarning || AsString(ranked[1].Alert["event"]) != "Tornado Warning" {
		t.Fatalf("second %+v", ranked[1])
	}
	if ranked[2].Kind != BannerWatch {
		t.Fatalf("third %+v", ranked[2])
	}
	if _, ok := BannerKindOf(map[string]any{"event": "Wind Advisory", "severity": "Extreme"}); ok {
		t.Fatal("advisory was promoted")
	}
	if _, ok := BannerKindOf(map[string]any{"event": "Special Weather Statement", "severity": "Extreme"}); ok {
		t.Fatal("statement was promoted from severity")
	}
}

func TestBannerExpiryKeepsTheMacStamp(t *testing.T) {
	got := BannerExpiry("2026-10-18T22:15:00Z")
	if got == "" || got == "2026-10-18T22:15:00Z" {
		t.Fatalf("stamp %q", got)
	}
	if strings.Contains(got, "T22") {
		t.Fatalf("stamp still raw %q", got)
	}
	frac := BannerExpiry("2026-10-18T22:15:00.5Z")
	if frac == "" || strings.Contains(frac, "T22") {
		t.Fatalf("fractional %q", frac)
	}
	if BannerExpiry("until further notice") != "until further notice" {
		t.Fatal("raw expiry was rewritten")
	}
	if BannerExpiry("  ") != "" {
		t.Fatal("blank expiry")
	}
}

func TestAlertDetailOmitsEmptyFields(t *testing.T) {
	fields := AlertDetailFields(map[string]any{
		"event":       "Flood Watch",
		"headline":    "Flooding possible",
		"area":        "Allen, IN",
		"expires":     "2026-10-18T22:15:00Z",
		"description": "  ",
	})
	seen := map[string]string{}
	for _, field := range fields {
		seen[field.Label] = field.Value
	}
	if seen["HEADLINE"] != "Flooding possible" || seen["AREA"] != "Allen, IN" {
		t.Fatalf("fields %+v", fields)
	}
	if _, ok := seen["DESCRIPTION"]; ok {
		t.Fatal("blank description was kept")
	}
	if _, ok := seen["INSTRUCTION"]; ok {
		t.Fatal("missing instruction was kept")
	}
	if seen["EXPIRES"] == "" || strings.Contains(seen["EXPIRES"], "T22") {
		t.Fatalf("expires %q", seen["EXPIRES"])
	}
}
