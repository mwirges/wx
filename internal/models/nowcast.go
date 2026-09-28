package models

import "time"

// PrecipPhase describes the precipitation type or phase.
type PrecipPhase string

const (
	PhaseNone         PrecipPhase = "none"
	PhaseRain         PrecipPhase = "rain"
	PhaseSnow         PrecipPhase = "snow"
	PhaseFreezingRain PrecipPhase = "freezing-rain"
	PhaseIcePellets   PrecipPhase = "ice-pellets"
	PhaseMixed        PrecipPhase = "mixed"
)

// PrecipInterval represents quantitative precipitation telemetry for a single time slice.
type PrecipInterval struct {
	StartTime   time.Time   `json:"start_time"`
	EndTime     time.Time   `json:"end_time"`
	Probability float64     `json:"probability"` // 0–100%
	RateMMH     float64     `json:"rate_mmh"`     // mm/h intensity rate
	RateInH     float64     `json:"rate_inh"`     // in/h intensity rate
	AccumMM     float64     `json:"accum_mm"`     // incremental accumulation in mm
	AccumIn     float64     `json:"accum_in"`     // incremental accumulation in inches
	Phase       PrecipPhase `json:"phase"`        // rain, snow, freezing-rain, ice-pellets, mixed
	Summary     string      `json:"summary"`      // "Moderate Rain", "Light Snow", "Passing Shower", "Clear"
}

// Nowcast holds quantitative precipitation timeline and summary metrics.
type Nowcast struct {
	GeneratedAt    time.Time        `json:"generated_at"`
	Location       string           `json:"location"`
	Headline       string           `json:"headline"`          // e.g. "Precipitation starting in ~25m", "Rain ending in ~15m", "Dry next 6h"
	IsActivePrecip bool             `json:"is_active_precip"`  // true if precip is occurring at start of window
	Summary        string           `json:"summary"`           // overview narrative
	PrimaryPhase   PrecipPhase      `json:"primary_phase"`     // rain, snow, etc.
	TotalLiquidMM  float64          `json:"total_liquid_mm"`   // total liquid accumulation (mm)
	TotalLiquidIn  float64          `json:"total_liquid_in"`   // total liquid accumulation (inches)
	TotalSnowCM    float64          `json:"total_snow_cm"`     // total snow accumulation (cm)
	TotalSnowIn    float64          `json:"total_snow_in"`     // total snow accumulation (inches)
	PeakRateMMH    float64          `json:"peak_rate_mmh"`     // peak precipitation rate (mm/h)
	PeakRateInH    float64          `json:"peak_rate_inh"`     // peak precipitation rate (in/h)
	PeakTime       time.Time        `json:"peak_time"`         // time of peak rate
	NextPrecipTime *time.Time       `json:"next_precip_time,omitempty"`
	PrecipEndTime  *time.Time       `json:"precip_end_time,omitempty"`
	Intervals      []PrecipInterval `json:"intervals"`
}

// NowcastPayload wraps Nowcast with location metadata.
type NowcastPayload struct {
	Location string   `json:"location"`
	Nowcast  *Nowcast `json:"nowcast"`
}
