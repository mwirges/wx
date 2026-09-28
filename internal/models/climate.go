package models

import "time"

// DailyNormals holds 30-year NOAA Climate Normals (1991–2020) for a specific date.
type DailyNormals struct {
	Date          time.Time `json:"date"`
	NormalHighF   float64   `json:"normal_high_f"`
	NormalHighC   float64   `json:"normal_high_c"`
	NormalLowF    float64   `json:"normal_low_f"`
	NormalLowC    float64   `json:"normal_low_c"`
	NormalMeanF   float64   `json:"normal_mean_f"`
	NormalMeanC   float64   `json:"normal_mean_c"`
	NormalPrecipIn float64  `json:"normal_precip_in"`
	NormalPrecipMM float64  `json:"normal_precip_mm"`
}

// DailyRecord represents an all-time station daily record for a calendar day.
type DailyRecord struct {
	ValueF  float64 `json:"value_f"`
	ValueC  float64 `json:"value_c"`
	ValueIn float64 `json:"value_in,omitempty"` // For precipitation records
	ValueMM float64 `json:"value_mm,omitempty"`
	Years   []int   `json:"years"`              // Years record was set/tied, e.g. [2021, 2025]
}

// DailyRecords holds all historical extremes for a calendar day.
type DailyRecords struct {
	RecordHigh        DailyRecord `json:"record_high"`
	RecordLow         DailyRecord `json:"record_low"`
	RecordPrecip      DailyRecord `json:"record_precip"`
	ColdestHigh       DailyRecord `json:"coldest_high"`       // Lowest daytime maximum
	WarmestLow        DailyRecord `json:"warmest_low"`        // Highest nighttime minimum
	PeriodOfRecord    string      `json:"period_of_record"`   // e.g. "1940–2025"
	TotalYearsSampled int         `json:"total_years_sampled"`
}

// ClimateDeparture represents observed departure anomalies relative to 1991–2020 normals.
type ClimateDeparture struct {
	ObservedHighF     *float64 `json:"observed_high_f,omitempty"`
	ObservedHighC     *float64 `json:"observed_high_c,omitempty"`
	DepartureHighF    *float64 `json:"departure_high_f,omitempty"` // e.g. +4.2°F
	DepartureHighC    *float64 `json:"departure_high_c,omitempty"`
	ObservedLowF      *float64 `json:"observed_low_f,omitempty"`
	ObservedLowC      *float64 `json:"observed_low_c,omitempty"`
	DepartureLowF     *float64 `json:"departure_low_f,omitempty"`
	DepartureLowC     *float64 `json:"departure_low_c,omitempty"`
	ObservedCurrentF  *float64 `json:"observed_current_f,omitempty"`
	ObservedCurrentC  *float64 `json:"observed_current_c,omitempty"`
	DepartureCurrentF *float64 `json:"departure_current_f,omitempty"`
	DepartureCurrentC *float64 `json:"departure_current_c,omitempty"`
	Summary           string   `json:"summary"` // e.g. "+5.2°F Above Normal (Warm Departure)"
}

// MonthlyNormals holds monthly summary normals for the current month.
type MonthlyNormals struct {
	MonthName           string  `json:"month_name"`       // "September"
	NormalAvgHighF      float64 `json:"normal_avg_high_f"`
	NormalAvgHighC      float64 `json:"normal_avg_high_c"`
	NormalAvgLowF       float64 `json:"normal_avg_low_f"`
	NormalAvgLowC       float64 `json:"normal_avg_low_c"`
	NormalTotalPrecipIn float64 `json:"normal_total_precip_in"`
	NormalTotalPrecipMM float64 `json:"normal_total_precip_mm"`
}

// ClimateReport aggregates normals, records, departures, and station metadata.
type ClimateReport struct {
	Date           time.Time         `json:"date"`
	Location       string            `json:"location"`
	StationID      string            `json:"station_id"`
	StationName    string            `json:"station_name"`
	Latitude       float64           `json:"latitude"`
	Longitude      float64           `json:"longitude"`
	ElevationFt    float64           `json:"elevation_ft"`
	ElevationM     float64           `json:"elevation_m"`
	NormalsPeriod  string            `json:"normals_period"` // "1991–2020"
	TodayNormals   DailyNormals      `json:"today_normals"`
	Records        DailyRecords      `json:"records"`
	Departure      *ClimateDeparture `json:"departure,omitempty"`
	MonthlyNormals *MonthlyNormals   `json:"monthly_normals,omitempty"`
}

// ClimatePayload wraps ClimateReport for serialization.
type ClimatePayload struct {
	Location string         `json:"location"`
	Climate  *ClimateReport `json:"climate"`
}
