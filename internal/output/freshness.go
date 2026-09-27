package output

import (
	"fmt"
	"time"
)

// Freshness provides information about observation age and caching state.
type Freshness struct {
	ObservedAt time.Time
	FetchedAt  time.Time
	FromCache  bool
}

// Age returns the elapsed time since the observation was recorded.
func (f Freshness) Age() time.Duration {
	if f.ObservedAt.IsZero() {
		return 0
	}
	return time.Since(f.ObservedAt)
}

// AgeString returns a human-friendly representation of data age (e.g. "just now", "15m ago", "2h ago").
func (f Freshness) AgeString() string {
	if f.ObservedAt.IsZero() {
		return ""
	}
	d := f.Age()
	if d < time.Minute {
		return "just now"
	}
	mins := int(d.Minutes())
	if mins < 60 {
		return fmt.Sprintf("%dm ago", mins)
	}
	hours := mins / 60
	remMins := mins % 60
	if remMins == 0 {
		return fmt.Sprintf("%dh ago", hours)
	}
	return fmt.Sprintf("%dh %dm ago", hours, remMins)
}
