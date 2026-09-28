package models

import "time"

// SoundingLevel represents atmospheric state at a specific pressure level.
type SoundingLevel struct {
	PressureHPA  float64  `json:"pressure_hpa"`
	HeightM      *float64 `json:"height_m,omitempty"`
	HeightFT     *float64 `json:"height_ft,omitempty"`
	TempC        *float64 `json:"temp_c,omitempty"`
	TempF        *float64 `json:"temp_f,omitempty"`
	DewPointC    *float64 `json:"dewpoint_c,omitempty"`
	DewPointF    *float64 `json:"dewpoint_f,omitempty"`
	WindDirDeg   *float64 `json:"wind_dir_deg,omitempty"`
	WindSpeedKT  *float64 `json:"wind_speed_kt,omitempty"`
	WindSpeedKPH *float64 `json:"wind_speed_kph,omitempty"`
	WindSpeedMPH *float64 `json:"wind_speed_mph,omitempty"`
	RHPct        *float64 `json:"rh_pct,omitempty"`
}

// ConvectiveIndices contains thermodynamic and kinematic stability parameters.
type ConvectiveIndices struct {
	// Thermodynamic Instability
	SBCAPE *float64 `json:"sbcape_jkg,omitempty"` // Surface-based CAPE (J/kg)
	MLCAPE *float64 `json:"mlcape_jkg,omitempty"` // Mixed-layer CAPE (J/kg)
	MUCAPE *float64 `json:"mucape_jkg,omitempty"` // Most-unstable CAPE (J/kg)
	SBCIN  *float64 `json:"sbcin_jkg,omitempty"`  // Surface-based CIN (J/kg)
	MLCIN  *float64 `json:"mlcin_jkg,omitempty"`  // Mixed-layer CIN (J/kg)
	MUCIN  *float64 `json:"mucin_jkg,omitempty"`  // Most-unstable CIN (J/kg)
	SBLI   *float64 `json:"sbli_c,omitempty"`     // Surface Lifted Index (°C)
	MLLI   *float64 `json:"mlli_c,omitempty"`     // Mixed-layer Lifted Index (°C)
	MULI   *float64 `json:"muli_c,omitempty"`     // Most-unstable Lifted Index (°C)

	// Moisture & Melting
	PWATIn         *float64 `json:"pwat_in,omitempty"`          // Precipitable Water (inches)
	PWATMm         *float64 `json:"pwat_mm,omitempty"`          // Precipitable Water (mm)
	FreezingLevelM *float64 `json:"freezing_level_m,omitempty"` // Freezing / Melting Level (meters)
	FreezingLevelFT *float64 `json:"freezing_level_ft,omitempty"`// Freezing / Melting Level (feet)
	DCAPE          *float64 `json:"dcape_jkg,omitempty"`        // Downdraft CAPE (J/kg)

	// Kinematics & Vertical Wind Shear
	BulkShear01KT *float64 `json:"bulk_shear_0_1km_kt,omitempty"` // 0-1km Bulk Wind Difference (knots)
	BulkShear03KT *float64 `json:"bulk_shear_0_3km_kt,omitempty"` // 0-3km Bulk Wind Difference (knots)
	BulkShear06KT *float64 `json:"bulk_shear_0_6km_kt,omitempty"` // 0-6km Deep-Layer Bulk Shear (knots)
	SRH01         *float64 `json:"srh_0_1km_m2s2,omitempty"`      // 0-1km Storm-Relative Helicity (m²/s²)
	SRH03         *float64 `json:"srh_0_3km_m2s2,omitempty"`      // 0-3km Storm-Relative Helicity (m²/s²)

	// Severe Weather Composite Indices
	STP  *float64 `json:"stp,omitempty"`  // Significant Tornado Parameter
	SCP  *float64 `json:"scp,omitempty"`  // Supercell Composite Parameter
	SHIP *float64 `json:"ship,omitempty"` // Significant Hail Parameter

	// Mid-Level Lapse Rates
	LapseRate700_500 *float64 `json:"lapse_rate_700_500_c_km,omitempty"` // 700-500mb Lapse Rate (°C/km)
	LapseRate850_500 *float64 `json:"lapse_rate_850_500_c_km,omitempty"` // 850-500mb Lapse Rate (°C/km)

	// Qualitative Summary
	InstabilitySummary string `json:"instability_summary"`
	ShearSummary       string `json:"shear_summary"`
	ConvectiveRisk     string `json:"convective_risk"`
}

// SoundingReport encapsulates the complete upper-air observation or model profile.
type SoundingReport struct {
	Timestamp     time.Time         `json:"timestamp"`
	Location      string            `json:"location"`
	StationID     string            `json:"station_id"`
	StationName   string            `json:"station_name"`
	DistanceKM    float64           `json:"distance_km"`
	DistanceMiles float64           `json:"distance_miles"`
	Provider      string            `json:"provider"`
	SkewTImageURL string            `json:"skewt_image_url,omitempty"`
	Indices       ConvectiveIndices `json:"indices"`
	Levels        []SoundingLevel   `json:"levels"`
}

// SoundingPayload is the top-level payload for CLI and JSON output.
type SoundingPayload struct {
	Location string          `json:"location"`
	Sounding *SoundingReport `json:"sounding"`
}
