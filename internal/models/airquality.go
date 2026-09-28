package models

import "time"

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

// EPAHealthAdvisory returns the official US EPA health advisory statement for a given AQI.
func EPAHealthAdvisory(aqi int) string {
	switch {
	case aqi <= 50:
		return "Air quality is satisfactory, and air pollution poses little or no risk."
	case aqi <= 100:
		return "Air quality is acceptable; unusually sensitive individuals should consider limiting prolonged outdoor exertion."
	case aqi <= 150:
		return "Members of sensitive groups may experience health effects. The general public is less likely to be affected."
	case aqi <= 200:
		return "Some members of the general public may experience health effects; sensitive groups may experience more serious effects."
	case aqi <= 300:
		return "Health alert: The risk of health effects is increased for everyone. Limit outdoor activities."
	default:
		return "Health warning of emergency conditions: everyone is more likely to be affected. Avoid outdoor exertion."
	}
}

// SmokeAdvisory returns a smoke plume warning when PM2.5 concentrations are elevated.
func SmokeAdvisory(pm25 float64) string {
	switch {
	case pm25 >= 125.5:
		return "Dense wildfire smoke plume detected. Severe particulate hazard; stay indoors with air filtration."
	case pm25 >= 55.5:
		return "Heavy particulate / smoke haze active. Avoid strenuous outdoor activities."
	case pm25 >= 35.5:
		return "Elevated PM2.5 smoke particulates present. Sensitive individuals should reduce outdoor exertion."
	default:
		return ""
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

// AirQualityPayload encapsulates location context, air quality telemetry, and health recommendations.
type AirQualityPayload struct {
	Location       string      `json:"location,omitempty"`
	Latitude       float64     `json:"latitude"`
	Longitude      float64     `json:"longitude"`
	FetchedAt      time.Time   `json:"fetched_at"`
	AirQuality     *AirQuality `json:"air_quality"`
	HealthAdvisory string      `json:"health_advisory,omitempty"`
	SmokeAdvisory  string      `json:"smoke_advisory,omitempty"`
	Source         string      `json:"source,omitempty"`
}

