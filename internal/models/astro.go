package models

import "time"

// Astronomy contains calculated solar times and day length for a location and date.
type Astronomy struct {
	Sunrise      *time.Time    `json:"sunrise,omitempty"`
	Sunset       *time.Time    `json:"sunset,omitempty"`
	SolarNoon    time.Time     `json:"solar_noon"`
	DayLength    time.Duration `json:"day_length"`
	IsPolarDay   bool          `json:"is_polar_day,omitempty"`
	IsPolarNight bool          `json:"is_polar_night,omitempty"`
}
