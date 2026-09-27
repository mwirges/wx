package models

import (
	"fmt"
	"math"
	"time"
)

// Astronomy contains calculated solar and lunar ephemeris for a location and date.
type Astronomy struct {
	Sunrise      *time.Time    `json:"sunrise,omitempty"`
	Sunset       *time.Time    `json:"sunset,omitempty"`
	SolarNoon    time.Time     `json:"solar_noon"`
	DayLength    time.Duration `json:"day_length"`
	IsPolarDay   bool          `json:"is_polar_day,omitempty"`
	IsPolarNight bool          `json:"is_polar_night,omitempty"`

	// Lunar ephemeris
	MoonPhase           string   `json:"moon_phase,omitempty"`           // e.g. "Waxing Gibbous", "Full Moon"
	MoonPhaseIcon       string   `json:"moon_phase_icon,omitempty"`      // Unicode emoji: 🌑, 🌒, 🌓, 🌔, 🌕, 🌖, 🌗, 🌘
	MoonIlluminationPct *float64 `json:"moon_illumination_pct,omitempty"` // 0.0 - 100.0%
	MoonAgeDays         *float64 `json:"moon_age_days,omitempty"`        // 0.0 - 29.53 days
}

// MoonIlluminationStr returns formatted illumination like "85%".
func (a *Astronomy) MoonIlluminationStr() string {
	if a == nil || a.MoonIlluminationPct == nil {
		return ""
	}
	return fmt.Sprintf("%.0f%%", math.Round(*a.MoonIlluminationPct))
}

// MoonAgeStr returns formatted moon age like "12.4d".
func (a *Astronomy) MoonAgeStr() string {
	if a == nil || a.MoonAgeDays == nil {
		return ""
	}
	return fmt.Sprintf("%.1fd", *a.MoonAgeDays)
}
