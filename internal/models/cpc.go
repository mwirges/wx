package models

import "time"

// CPCOutlookItem represents a single CPC horizon outlook (6-10 Day or 8-14 Day).
type CPCOutlookItem struct {
	Horizon           string    `json:"horizon"`            // "6-10 Day", "8-14 Day"
	StartDate         time.Time `json:"start_date"`
	EndDate           time.Time `json:"end_date"`
	TempCategory      string    `json:"temp_category"`      // "Above", "Below", "Normal", "Equal Chances"
	TempProbability   float64   `json:"temp_probability"`   // percentage 33..100
	PrecipCategory    string    `json:"precip_category"`    // "Above", "Below", "Normal", "Equal Chances"
	PrecipProbability float64   `json:"precip_probability"` // percentage 33..100
}

// CPCDroughtOutlook represents the monthly/seasonal drought status from CPC.
type CPCDroughtOutlook struct {
	Status string `json:"status"` // e.g. "No Drought", "Drought Persists", "Drought Improves", "Drought Develops"
	Target string `json:"target"` // e.g. "Sep 2026"
}

// CPCPatternShift highlights upcoming shifts in the synoptic weather pattern.
type CPCPatternShift struct {
	HasShift    bool   `json:"has_shift"`
	Summary     string `json:"summary"`
	TempShift   string `json:"temp_shift,omitempty"`   // "cooling", "warming", "steady"
	PrecipShift string `json:"precip_shift,omitempty"` // "drying", "wetter", "steady"
	Confidence  string `json:"confidence,omitempty"`   // "Moderate", "High"
}

// CPCPayload is the complete CPC outlook response.
type CPCPayload struct {
	Location     string             `json:"location"`
	Coordinates  [2]float64         `json:"coordinates"` // [lat, lon]
	FetchedAt    time.Time          `json:"fetched_at"`
	Outlooks     []CPCOutlookItem   `json:"outlooks"`
	Drought      *CPCDroughtOutlook `json:"drought,omitempty"`
	PatternShift CPCPatternShift    `json:"pattern_shift"`
}
