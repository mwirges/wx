package models

import "time"

// Forecast holds a sequence of forecast periods.
type Forecast struct {
	GeneratedAt time.Time
	Periods     []Period
}

// Period represents one forecast period (e.g. "Tonight", "Monday", "Monday Night").
// Temperature is in °C, wind speed in km/h.
type Period struct {
	Name         string
	StartTime    time.Time
	EndTime      time.Time
	IsDaytime    bool
	TempC        float64
	WindKPH      float64 // upper bound from NWS wind string
	WindDir      string  // compass direction, e.g. "NW"
	ShortDesc    string
	DetailedDesc string

	// Quantitative fields (Phase 1)
	ProbabilityOfPrecipitation *float64 // 0-100, nil if not reported
	DewPointC                  *float64 // °C, nil if not reported
	HumidityPct                *float64 // 0-100, nil if not reported
}
