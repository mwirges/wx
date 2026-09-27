package openmeteo

import "math"

// wmoToConditionCode maps WMO weather interpretation codes to wx normalized condition codes.
func wmoToConditionCode(code int, isDay bool) string {
	switch code {
	case 0, 1:
		if isDay {
			return "clear-day"
		}
		return "clear-night"
	case 2:
		if isDay {
			return "partly-cloudy-day"
		}
		return "partly-cloudy-night"
	case 3:
		return "cloudy"
	case 45, 48:
		return "fog"
	case 51, 53, 55, 61, 63, 80, 81:
		return "rain"
	case 65, 82:
		return "heavy-rain"
	case 56, 57, 66, 67:
		return "sleet"
	case 71, 73, 75, 77, 85, 86:
		return "snow"
	case 95, 96, 99:
		return "thunder"
	default:
		if isDay {
			return "partly-cloudy-day"
		}
		return "partly-cloudy-night"
	}
}

// wmoToDescription maps WMO weather interpretation codes to human-readable descriptions.
func wmoToDescription(code int) string {
	switch code {
	case 0:
		return "Clear"
	case 1:
		return "Mainly Clear"
	case 2:
		return "Partly Cloudy"
	case 3:
		return "Overcast"
	case 45:
		return "Fog"
	case 48:
		return "Depositing Rime Fog"
	case 51:
		return "Light Drizzle"
	case 53:
		return "Moderate Drizzle"
	case 55:
		return "Dense Drizzle"
	case 56:
		return "Light Freezing Drizzle"
	case 57:
		return "Dense Freezing Drizzle"
	case 61:
		return "Slight Rain"
	case 63:
		return "Moderate Rain"
	case 65:
		return "Heavy Rain"
	case 66:
		return "Light Freezing Rain"
	case 67:
		return "Heavy Freezing Rain"
	case 71:
		return "Slight Snow Fall"
	case 73:
		return "Moderate Snow Fall"
	case 75:
		return "Heavy Snow Fall"
	case 77:
		return "Snow Grains"
	case 80:
		return "Slight Rain Showers"
	case 81:
		return "Moderate Rain Showers"
	case 82:
		return "Violent Rain Showers"
	case 85:
		return "Slight Snow Showers"
	case 86:
		return "Heavy Snow Showers"
	case 95:
		return "Thunderstorm"
	case 96:
		return "Thunderstorm with Hail"
	case 99:
		return "Thunderstorm with Heavy Hail"
	default:
		return "Fair"
	}
}

// degreesToCompass converts a wind direction in degrees (0-360) to a 16-point compass abbreviation.
func degreesToCompass(deg float64) string {
	directions := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	normalized := math.Mod(deg, 360)
	if normalized < 0 {
		normalized += 360
	}
	index := int(math.Floor((normalized+11.25)/22.5)) % 16
	return directions[index]
}
