package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleTempCold = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // blue
	styleTempMild = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))  // green
	styleTempWarm = lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // yellow
	styleTempHot  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // red

	styleLocation = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleTime     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleLabel    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleValue    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleDesc     = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Italic(true)

	styleAlertWarning = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("196")). // Red
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("196")).
				Padding(0, 1)

	styleAlertWatch = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214")). // Amber / Orange
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")).
			Padding(0, 1)

	styleAlertAdvisory = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39")). // Cyan / Blue
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("39")).
				Padding(0, 1)

	styleForecastHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleForecastName   = lipgloss.NewStyle().Width(16).Foreground(lipgloss.Color("252"))
	styleForecastHigh   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleForecastLow    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	styleForecastDesc   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

func renderPretty(data RenderData, opts RenderOptions) error {
	imperial := opts.Units != "metric"

	// ── Current Conditions ─────────────────────────────────────────
	if data.Conditions != nil {
		c := data.Conditions

		// Location + time header (full width, no icon)
		timeStr := c.ObservedAt.Local().Format("Mon Jan 2, 3:04 PM")
		freshness := data.Freshness
		if freshness.ObservedAt.IsZero() && !c.ObservedAt.IsZero() {
			freshness.ObservedAt = c.ObservedAt
		}
		if ageStr := freshness.AgeString(); ageStr != "" && ageStr != "just now" {
			timeStr += fmt.Sprintf(" (%s)", ageStr)
		}
		ts := styleTime.Render(timeStr)
		fmt.Printf("%s  %s\n\n", styleLocation.Render(c.Location), ts)

		// Build 5 content slots to sit beside the icon.
		var contents [5]string

		// Slot 0: temperature + description
		if c.TempC != nil {
			tf := FormatTemp(*c.TempC, imperial)
			s := TempStyle(*c.TempC, imperial).Bold(true).Render(tf)
			if c.Description != "" {
				s += "  " + styleDesc.Render(c.Description)
			}
			contents[0] = s
		} else if c.Description != "" {
			contents[0] = styleDesc.Render(c.Description)
		}

		// Slot 1: feels like (wind chill or heat index)
		fl := c.FeelsLikeC
		if fl == nil {
			fl = FeelsLikeTemp(c.WindChillC, c.HeatIndexC)
		}
		if fl != nil {
			label := styleLabel.Render("Feels like")
			val := TempStyle(*fl, imperial).Render(FormatTemp(*fl, imperial))
			contents[1] = label + " " + val
		}

		// Slot 2: wind (with optional gusts)
		if c.WindKPH != nil {
			windStr := FormatWind(*c.WindKPH, c.WindDegrees, imperial)
			if c.WindGustKPH != nil {
				windStr += ", gusts " + FormatWindSpeed(*c.WindGustKPH, imperial)
			}
			contents[2] = styleLabel.Render("Wind:") + " " + styleValue.Render(windStr)
		}

		// Slot 3: humidity + dew point
		var humDew []string
		if c.HumidityPct != nil {
			humDew = append(humDew, styleLabel.Render("Humidity:")+" "+styleValue.Render(fmt.Sprintf("%.0f%%", *c.HumidityPct)))
		}
		if c.DewPointC != nil {
			humDew = append(humDew, styleLabel.Render("Dew point:")+" "+styleValue.Render(FormatTemp(*c.DewPointC, imperial)))
		}
		if len(humDew) > 0 {
			contents[3] = strings.Join(humDew, "   ")
		}

		// Slot 4: pressure + visibility
		var presVis []string
		if c.PressureHPA != nil {
			presVis = append(presVis, styleLabel.Render("Pressure:")+" "+styleValue.Render(FormatPressure(*c.PressureHPA, imperial)))
		}
		if c.VisibilityM != nil {
			presVis = append(presVis, styleLabel.Render("Visibility:")+" "+styleValue.Render(FormatVisibility(*c.VisibilityM, imperial)))
		}
		if len(presVis) > 0 {
			contents[4] = strings.Join(presVis, "   ")
		}

		// Render icon lines alongside content slots.
		ic := GetIcon(c.ConditionCode)
		for i := 0; i < 5; i++ {
			iconStr := lipgloss.NewStyle().
				Foreground(ic.Colors[i]).
				Width(IconWidth).
				Render(ic.Lines[i])
			fmt.Printf("  %s  %s\n", iconStr, contents[i])
		}

		if c.Astronomy != nil {
			hasSun := c.Astronomy.IsPolarDay || c.Astronomy.IsPolarNight || (c.Astronomy.Sunrise != nil && c.Astronomy.Sunset != nil)
			hasMoon := c.Astronomy.MoonPhase != ""
			if hasSun || hasMoon {
				fmt.Println()
			}
			if c.Astronomy.IsPolarDay {
				fmt.Printf("  %s  %s\n", styleLabel.Render("Sun:"), styleValue.Render("Polar Day (24h daylight)"))
			} else if c.Astronomy.IsPolarNight {
				fmt.Printf("  %s  %s\n", styleLabel.Render("Sun:"), styleValue.Render("Polar Night (0h daylight)"))
			} else if c.Astronomy.Sunrise != nil && c.Astronomy.Sunset != nil {
				sr := c.Astronomy.Sunrise.Local().Format("3:04 PM")
				ss := c.Astronomy.Sunset.Local().Format("3:04 PM")
				hours := int(c.Astronomy.DayLength.Hours())
				mins := int(c.Astronomy.DayLength.Minutes()) % 60
				dayLenStr := fmt.Sprintf("%dh %dm", hours, mins)

				upArrow := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("↑")   // warm gold
				downArrow := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("↓") // dusk orange
				sunLabel := styleLabel.Render("Sun:")
				srStr := styleValue.Render(sr)
				ssStr := styleValue.Render(ss)
				dlStr := styleDesc.Render(fmt.Sprintf("(%s daylight)", dayLenStr))

				fmt.Printf("  %s  %s %s   %s %s   %s\n", sunLabel, upArrow, srStr, downArrow, ssStr, dlStr)
			}
			if hasMoon {
				moonLabel := styleLabel.Render("Moon:")
				phaseStr := styleValue.Render(c.Astronomy.MoonPhase)
				icon := c.Astronomy.MoonPhaseIcon
				if icon != "" {
					icon = icon + " "
				}
				var details []string
				if c.Astronomy.MoonIlluminationPct != nil {
					details = append(details, c.Astronomy.MoonIlluminationStr()+" illuminated")
				}
				if c.Astronomy.MoonAgeDays != nil {
					details = append(details, c.Astronomy.MoonAgeStr()+" age")
				}
				detailStr := ""
				if len(details) > 0 {
					detailStr = "   " + styleDesc.Render(fmt.Sprintf("(%s)", strings.Join(details, ", ")))
				}
				fmt.Printf("  %s %s%s%s\n", moonLabel, icon, phaseStr, detailStr)
			}
		}

		if c.AirQuality != nil && (c.AirQuality.AQI != nil || c.AirQuality.UVIndex != nil) {
			var aqParts []string
			if c.AirQuality.AQI != nil {
				aqiVal := *c.AirQuality.AQI
				cat := c.AirQuality.Category
				style := AQIStyle(aqiVal)
				aqParts = append(aqParts, styleLabel.Render("Air Quality:")+" "+style.Render(fmt.Sprintf("%d (%s)", aqiVal, cat)))
			}
			if c.AirQuality.UVIndex != nil {
				uvVal := *c.AirQuality.UVIndex
				cat := c.AirQuality.UVCategory
				style := UVStyle(uvVal)
				aqParts = append(aqParts, styleLabel.Render("UV Index:")+" "+style.Render(fmt.Sprintf("%.1f (%s)", uvVal, cat)))
			}
			if len(aqParts) > 0 {
				fmt.Printf("  %s\n", strings.Join(aqParts, "   "))
			}
		}

		fmt.Println()
	}

	// ── Alerts ─────────────────────────────────────────────────────
	if len(data.Alerts) > 0 {
		for _, a := range data.Alerts {
			headline := a.Headline
			if headline == "" {
				headline = a.Event
			}
			bannerStyle := styleAlertAdvisory
			if a.IsWarning() {
				bannerStyle = styleAlertWarning
			} else if a.IsWatch() {
				bannerStyle = styleAlertWatch
			}
			fmt.Println(bannerStyle.Render("⚠  " + headline))
			if a.AreaDesc != "" {
				fmt.Printf("   %s\n", styleLabel.Render(a.AreaDesc))
			}
			if !a.Expires.IsZero() {
				fmt.Printf("   %s %s\n", styleLabel.Render("Expires:"), styleTime.Render(a.Expires.Local().Format("Mon Jan 2, 3:04 PM")))
			}
			fmt.Println()
		}
	}

	// ── Forecast ───────────────────────────────────────────────────
	if data.Forecast != nil && len(data.Forecast.Periods) > 0 {
		if opts.ShowHourly {
			limit := opts.HourlyLimit
			if limit <= 0 {
				limit = 24
			}
			headerTitle := fmt.Sprintf("Hourly Forecast (Next %d Hours)", limit)
			if len(data.Forecast.Periods) < limit {
				limit = len(data.Forecast.Periods)
				headerTitle = fmt.Sprintf("Hourly Forecast (%d Hours)", limit)
			}
			fmt.Println(styleForecastHeader.Render(headerTitle))

			colTime := lipgloss.NewStyle().Width(12).Foreground(lipgloss.Color("244")).Bold(true).Render("TIME")
			colTemp := lipgloss.NewStyle().Width(7).Foreground(lipgloss.Color("244")).Bold(true).Render("TEMP")
			colPrecip := lipgloss.NewStyle().Width(7).Foreground(lipgloss.Color("244")).Bold(true).Render("PRECIP")
			colHum := lipgloss.NewStyle().Width(7).Foreground(lipgloss.Color("244")).Bold(true).Render("HUMID")
			colWind := lipgloss.NewStyle().Width(14).Foreground(lipgloss.Color("244")).Bold(true).Render("WIND")
			colDesc := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Bold(true).Render("FORECAST")

			fmt.Printf("  %s %s  %s  %s  %s  %s\n", colTime, colTemp, colPrecip, colHum, colWind, colDesc)
			fmt.Println(styleLabel.Render(strings.Repeat("─", 75)))

			for _, p := range data.Forecast.Periods[:limit] {
				timeStyled := styleForecastName.Width(12).Render(p.Name)
				tempStr := FormatTemp(p.TempC, imperial)
				tempStyled := TempStyle(p.TempC, imperial).Width(7).Render(tempStr)

				precipStr := "  — "
				if p.ProbabilityOfPrecipitation != nil && *p.ProbabilityOfPrecipitation > 0 {
					precipStr = fmt.Sprintf("%3.0f%%", *p.ProbabilityOfPrecipitation)
				}
				precipStyled := styleForecastLow.Width(7).Render(precipStr)

				humidStr := "  — "
				if p.HumidityPct != nil {
					humidStr = fmt.Sprintf("%3.0f%%", *p.HumidityPct)
				}
				humidStyled := styleLabel.Width(7).Render(humidStr)

				windStr := FormatWind(p.WindKPH, nil, imperial)
				if p.WindDir != "" {
					windStr = p.WindDir + " " + windStr
				}
				windStyled := styleLabel.Width(14).Render(windStr)

				desc := styleForecastDesc.Render(p.ShortDesc)
				fmt.Printf("  %s %s  %s  %s  %s  %s\n", timeStyled, tempStyled, precipStyled, humidStyled, windStyled, desc)
			}
			fmt.Println()
		} else {
			fmt.Println(styleForecastHeader.Render("Forecast"))
			fmt.Println(styleLabel.Render(strings.Repeat("─", 60)))

			for _, p := range data.Forecast.Periods {
				tempStr := FormatTemp(p.TempC, imperial)
				var tempStyled string
				if p.IsDaytime {
					tempStyled = styleForecastHigh.Render("High " + tempStr)
				} else {
					tempStyled = styleForecastLow.Render("Low  " + tempStr)
				}
				desc := styleForecastDesc.Render(p.ShortDesc)
				fmt.Printf("  %s %s   %s\n", styleForecastName.Render(p.Name), tempStyled, desc)
			}
			fmt.Println()
		}
	}

	return nil
}

func formatAge(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	mins := int(d.Minutes())
	if mins < 60 {
		return fmt.Sprintf("%dm ago", mins)
	}
	hours := mins / 60
	remMins := mins % 60
	if remMins == 0 {
		return fmt.Sprintf("%dh ago", hours)
	}
	return fmt.Sprintf("%dh %dm ago", hours, remMins)
}

// FeelsLikeTemp returns WindChillC if set, HeatIndexC if set, otherwise nil.
func FeelsLikeTemp(windChill, heatIndex *float64) *float64 {
	if windChill != nil {
		return windChill
	}
	return heatIndex
}

// TempStyle returns the lipgloss style appropriate for the temperature.
func TempStyle(tempC float64, imperial bool) lipgloss.Style {
	var ref float64
	if imperial {
		ref = CelsiusToFahrenheit(tempC)
	} else {
		ref = tempC
	}
	switch {
	case (!imperial && ref < 0) || (imperial && ref < 32):
		return styleTempCold
	case (!imperial && ref < 18) || (imperial && ref < 65):
		return styleTempMild
	case (!imperial && ref < 29) || (imperial && ref < 85):
		return styleTempWarm
	default:
		return styleTempHot
	}
}

// FormatTemp formats temperature for display with the appropriate unit suffix.
func FormatTemp(tempC float64, imperial bool) string {
	if imperial {
		return fmt.Sprintf("%.0f°F", CelsiusToFahrenheit(tempC))
	}
	return fmt.Sprintf("%.1f°C", tempC)
}

// FormatWind formats wind speed and direction for display.
func FormatWind(kph float64, degrees *float64, imperial bool) string {
	speed := FormatWindSpeed(kph, imperial)
	if degrees != nil {
		return DegreesToCompass(*degrees) + " " + speed
	}
	return speed
}

// FormatWindSpeed formats wind speed only (no direction).
func FormatWindSpeed(kph float64, imperial bool) string {
	if imperial {
		return fmt.Sprintf("%.0f mph", KphToMPH(kph))
	}
	return fmt.Sprintf("%.0f km/h", kph)
}

// FormatPressure formats barometric pressure for display.
func FormatPressure(hpa float64, imperial bool) string {
	if imperial {
		return fmt.Sprintf("%.2f inHg", hpa/33.8639)
	}
	return fmt.Sprintf("%.0f hPa", hpa)
}

// FormatVisibility formats visibility distance for display.
func FormatVisibility(meters float64, imperial bool) string {
	if imperial {
		mi := meters / 1609.344
		if mi >= 10 {
			return fmt.Sprintf("%.0f mi", mi)
		}
		return fmt.Sprintf("%.1f mi", mi)
	}
	km := meters / 1000
	if km >= 10 {
		return fmt.Sprintf("%.0f km", km)
	}
	return fmt.Sprintf("%.1f km", km)
}

// DegreesToCompass converts 0–360 degrees to a compass direction string.
func DegreesToCompass(deg float64) string {
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int((deg+11.25)/22.5) % 16
	return dirs[idx]
}

// CelsiusToFahrenheit converts Celsius to Fahrenheit.
func CelsiusToFahrenheit(c float64) float64 {
	return c*9/5 + 32
}

// KphToMPH converts km/h to miles per hour.
func KphToMPH(kph float64) float64 {
	return kph / 1.60934
}

// FahrenheitToCelsius converts Fahrenheit to Celsius.
func FahrenheitToCelsius(f float64) float64 {
	return (f - 32) * 5 / 9
}

// MphToKPH converts miles per hour to km/h.
func MphToKPH(mph float64) float64 {
	return mph * 1.60934
}

// AQIStyle returns a lipgloss Style corresponding to EPA Air Quality Index bands:
// Good (Green 34), Moderate (Yellow 220), Unhealthy for Sensitive (Orange 208),
// Unhealthy (Red 196), Very Unhealthy (Purple 129), Hazardous (Maroon 88).
func AQIStyle(aqi int) lipgloss.Style {
	switch {
	case aqi <= 50:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
	case aqi <= 100:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	case aqi <= 150:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	case aqi <= 200:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	case aqi <= 300:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("129")).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("88")).Bold(true)
	}
}

// UVStyle returns a lipgloss Style corresponding to WHO UV Index bands:
// Low (Green 34), Moderate (Yellow 220), High (Orange 208), Very High (Red 196), Extreme (Purple 129).
func UVStyle(uv float64) lipgloss.Style {
	switch {
	case uv < 3.0:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
	case uv < 6.0:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	case uv < 8.0:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	case uv < 11.0:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("129")).Bold(true)
	}
}

