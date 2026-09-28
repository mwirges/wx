package models

// AirQuality represents atmospheric air quality and UV telemetry.
// AQI follows the US EPA Air Quality Index scale (0-500).
// UVIndex follows the WHO Global Solar UV Index scale (0-11+).
type AirQuality struct {
	AQI        *int     `json:"aqi,omitempty"`         // US EPA AQI (0–500)
	Category   string   `json:"category,omitempty"`    // "Good", "Moderate", "Unhealthy for Sensitive Groups", etc.
	UVIndex    *float64 `json:"uv_index,omitempty"`    // WHO UV Index (0–11+)
	UVCategory string   `json:"uv_category,omitempty"` // "Low", "Moderate", "High", "Very High", "Extreme"
	PM25       *float64 `json:"pm2_5,omitempty"`       // μg/m³
	PM10       *float64 `json:"pm10,omitempty"`        // μg/m³
	O3         *float64 `json:"ozone,omitempty"`       // μg/m³
	NO2        *float64 `json:"no2,omitempty"`         // μg/m³
	CO         *float64 `json:"co,omitempty"`          // μg/m³
	SO2        *float64 `json:"so2,omitempty"`         // μg/m³
}

// AQICategory returns the EPA category string for a given US AQI value.
func AQICategory(aqi int) string {
	switch {
	case aqi <= 50:
		return "Good"
	case aqi <= 100:
		return "Moderate"
	case aqi <= 150:
		return "Unhealthy for Sensitive Groups"
	case aqi <= 200:
		return "Unhealthy"
	case aqi <= 300:
		return "Very Unhealthy"
	default:
		return "Hazardous"
	}
}

// UVCategory returns the WHO category string for a given UV Index value.
func UVCategory(uv float64) string {
	switch {
	case uv < 3.0:
		return "Low"
	case uv < 6.0:
		return "Moderate"
	case uv < 8.0:
		return "High"
	case uv < 11.0:
		return "Very High"
	default:
		return "Extreme"
	}
}
