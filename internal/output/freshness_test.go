package output

import (
	"testing"
	"time"
)

func TestFreshness_AgeString(t *testing.T) {
	now := time.Now()

	// Zero time
	fZero := Freshness{}
	if fZero.Age() != 0 {
		t.Errorf("expected 0 age for zero time, got %v", fZero.Age())
	}
	if fZero.AgeString() != "" {
		t.Errorf("expected empty string for zero time, got %q", fZero.AgeString())
	}

	// Just now (< 1 min)
	fJustNow := Freshness{ObservedAt: now.Add(-30 * time.Second)}
	if fJustNow.AgeString() != "just now" {
		t.Errorf("expected 'just now', got %q", fJustNow.AgeString())
	}

	// Minutes
	fMins := Freshness{ObservedAt: now.Add(-12 * time.Minute)}
	if fMins.AgeString() != "12m ago" {
		t.Errorf("expected '12m ago', got %q", fMins.AgeString())
	}

	// Exact hour
	fHour := Freshness{ObservedAt: now.Add(-60 * time.Minute)}
	if fHour.AgeString() != "1h ago" {
		t.Errorf("expected '1h ago', got %q", fHour.AgeString())
	}

	// Hours + minutes
	fHoursMins := Freshness{ObservedAt: now.Add(-135 * time.Minute)}
	if fHoursMins.AgeString() != "2h 15m ago" {
		t.Errorf("expected '2h 15m ago', got %q", fHoursMins.AgeString())
	}
}
