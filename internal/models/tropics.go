package models

import "time"

// TropicalClassification represents the official NHC cyclone classification.
type TropicalClassification string

const (
	ClassHurricane            TropicalClassification = "HU"
	ClassTropicalStorm        TropicalClassification = "TS"
	ClassTropicalDepression   TropicalClassification = "TD"
	ClassPotentialCyclone     TropicalClassification = "PTC"
	ClassSubtropicalStorm     TropicalClassification = "SS"
	ClassSubtropicalDepr      TropicalClassification = "SD"
	ClassPostTropical         TropicalClassification = "POST"
	ClassExtratropical        TropicalClassification = "EX"
	ClassRemnantLow           TropicalClassification = "LO"
	ClassDisturbance          TropicalClassification = "DB"
)

// TropicalStorm represents an active or tracked tropical cyclone from the National Hurricane Center.
type TropicalStorm struct {
	ID                 string    `json:"id"`
	BinNumber          string    `json:"bin_number"`
	Name               string    `json:"name"`
	Classification     string    `json:"classification"`
	ClassificationName string    `json:"classification_name"`
	Category           int       `json:"category"`             // 1-5 for Saffir-Simpson, 0 for TS/TD
	CategoryLabel      string    `json:"category_label"`       // e.g., "Category 3 (Major)", "Tropical Storm"
	IntensityKt        int       `json:"intensity_kt"`         // Knots
	WindSpeedMph       int       `json:"wind_speed_mph"`       // MPH
	WindSpeedKmh       int       `json:"wind_speed_kmh"`       // KM/H
	PressureMb         int       `json:"pressure_mb"`          // Millibars / hPa
	PressureInHg       float64   `json:"pressure_inhg"`        // Inches of mercury
	Latitude           float64   `json:"latitude"`
	Longitude          float64   `json:"longitude"`
	LocationText       string    `json:"location_text"`        // e.g. "23.6N 113.8W"
	MovementDir        int       `json:"movement_dir"`         // Heading in degrees (0-360)
	MovementCompass    string    `json:"movement_compass"`     // Compass cardinal e.g. "NNE"
	MovementSpeedMph   int       `json:"movement_speed_mph"`
	MovementSpeedKmh   int       `json:"movement_speed_kmh"`
	Headline           string    `json:"headline,omitempty"`
	ProximityText      string    `json:"proximity_text,omitempty"` // e.g. "About 125 mi SW of Cabo San Lazaro Mexico"
	DistanceKm         *float64  `json:"distance_km,omitempty"`
	DistanceMiles      *float64  `json:"distance_miles,omitempty"`
	WatchesWarnings    []string  `json:"watches_warnings,omitempty"`
	AdvisoryNumber     string    `json:"advisory_number,omitempty"`
	AdvisoryTime       string    `json:"advisory_time,omitempty"`
	PublicAdvisoryURL  string    `json:"public_advisory_url,omitempty"`
	ForecastDiscussion string    `json:"forecast_discussion_url,omitempty"`
	GraphicsURL        string    `json:"graphics_url,omitempty"`
	TrackConeKMZ       string    `json:"track_cone_kmz,omitempty"`
	LastUpdate         time.Time `json:"last_update"`
}

// TropicalDisturbance represents an invest area or area of interest in the Tropical Weather Outlook.
type TropicalDisturbance struct {
	ID          string `json:"id"`                    // e.g. "AL91"
	Basin       string `json:"basin"`                 // "Atlantic", "Eastern Pacific", "Central Pacific"
	Name        string `json:"name"`                  // e.g. "Central Subtropical Atlantic"
	Chance48h   int    `json:"chance_48h"`            // Percent (0-100)
	Category48h string `json:"category_48h"`         // "low", "medium", "high"
	Chance7d    int    `json:"chance_7d"`             // Percent (0-100)
	Category7d  string `json:"category_7d"`          // "low", "medium", "high"
	Summary     string `json:"summary,omitempty"`
}

// TropicsReport aggregates all active NHC tropical cyclones and tropical weather outlook disturbances.
type TropicsReport struct {
	GeneratedAt        time.Time             `json:"generated_at"`
	TotalActive        int                   `json:"total_active"`
	Storms             []TropicalStorm       `json:"storms"`
	Disturbances       []TropicalDisturbance `json:"disturbances,omitempty"`
	AtlanticOutlookURL string                `json:"atlantic_outlook_url"`
	PacificOutlookURL  string                `json:"pacific_outlook_url"`
	ReferenceLocation  string                `json:"reference_location,omitempty"`
}

// TropicsPayload is the JSON wrapper returned for CLI `--json` output.
type TropicsPayload struct {
	Location string         `json:"location,omitempty"`
	Tropics  *TropicsReport `json:"tropics"`
}
