package models

import "time"

// HistoryDay represents observed weather data for a single day in history.
// All measurements are in SI units (°C, km/h, mm). Pointers allow nil for unreported values.
type HistoryDay struct {
	Date          time.Time
	ConditionCode string
	Description   string
	TempMaxC      *float64
	TempMinC      *float64
	ApparentMaxC  *float64
	ApparentMinC  *float64
	PrecipSumMM   *float64
	WindMaxKPH    *float64
}

// HistorySummary holds aggregated statistics over a historical observation period.
type HistorySummary struct {
	DaysCount     int
	AvgTempMaxC   *float64
	AvgTempMinC   *float64
	TotalPrecipMM *float64
	MaxWindKPH    *float64
}

// HistoricalWeather represents a sequence of past daily observations and location context.
type HistoricalWeather struct {
	Location  string
	Latitude  float64
	Longitude float64
	Elevation *float64
	Days      []HistoryDay
	Summary   HistorySummary
}
