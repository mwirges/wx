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

	// Solar twilight & golden hour horizons
	CivilDawn              *time.Time `json:"civil_dawn,omitempty"`
	CivilDusk              *time.Time `json:"civil_dusk,omitempty"`
	NauticalDawn           *time.Time `json:"nautical_dawn,omitempty"`
	NauticalDusk           *time.Time `json:"nautical_dusk,omitempty"`
	AstroDawn              *time.Time `json:"astro_dawn,omitempty"`
	AstroDusk              *time.Time `json:"astro_dusk,omitempty"`
	GoldenHourMorningStart *time.Time `json:"golden_hour_morning_start,omitempty"`
	GoldenHourMorningEnd   *time.Time `json:"golden_hour_morning_end,omitempty"`
	GoldenHourEveningStart *time.Time `json:"golden_hour_evening_start,omitempty"`
	GoldenHourEveningEnd   *time.Time `json:"golden_hour_evening_end,omitempty"`

	// Instantaneous solar positioning
	SolarElevationDeg *float64 `json:"solar_elevation_deg,omitempty"`
	SolarAzimuthDeg   *float64 `json:"solar_azimuth_deg,omitempty"`
	CurrentPeriod     string   `json:"current_period,omitempty"` // "Day", "Golden Hour", "Civil Twilight", "Nautical Twilight", "Astronomical Twilight", "Night"

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

// AstroPayload encapsulates location context and calculated astronomical ephemeris.
type AstroPayload struct {
	Location     string     `json:"location,omitempty"`
	Latitude     float64    `json:"latitude"`
	Longitude    float64    `json:"longitude"`
	CalculatedAt time.Time  `json:"calculated_at"`
	Astronomy    *Astronomy `json:"astronomy"`
}

