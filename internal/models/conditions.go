package models

import "time"

// CurrentConditions holds the latest observed weather. All measurements are SI units:
// temperature/dew point/wind chill/heat index in °C, wind speed in km/h,
// pressure in hPa, visibility in meters, humidity as a percentage 0–100.
// Fields that the station did not report are nil.
type CurrentConditions struct {
	StationID   string
	StationName string
	ObservedAt  time.Time
	Location    string   // human-readable location name
	Description string   // "Partly Cloudy", "Light Rain", etc.

	TempC       *float64 // Celsius
	WindChillC  *float64 // Celsius; set by NWS when temp ≤ 50°F and wind > 3 mph
	HeatIndexC  *float64 // Celsius; set by NWS when temp ≥ 80°F and humidity ≥ 40%

	DewPointC   *float64 // Celsius
	HumidityPct *float64 // 0–100

	WindKPH     *float64 // km/h
	WindGustKPH *float64 // km/h; nil if no gusts reported
	WindDegrees *float64 // 0–360

	PressureHPA *float64 // hPa (sea-level); nil if unavailable
	VisibilityM *float64 // meters; nil if unavailable

	// ConditionCode is a normalized icon key (e.g. "clear-day", "rain", "snow").
	// Set by the provider; empty string means unknown.
	ConditionCode string
}
