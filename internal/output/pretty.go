package output

import (
	"fmt"
	"strings"

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

	styleAlertBanner = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("196")).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("196")).
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
		ts := styleTime.Render(c.ObservedAt.Local().Format("Mon Jan 2, 3:04 PM"))
		fmt.Printf("%s  %s\n\n", styleLocation.Render(c.Location), ts)

		// Build 5 content slots to sit beside the icon.
		var contents [5]string

		// Slot 0: temperature + description
		if c.TempC != nil {
			tf := formatTemp(*c.TempC, imperial)
			s := tempStyle(*c.TempC, imperial).Bold(true).Render(tf)
			if c.Description != "" {
				s += "  " + styleDesc.Render(c.Description)
			}
			contents[0] = s
		} else if c.Description != "" {
			contents[0] = styleDesc.Render(c.Description)
		}

		// Slot 1: feels like (wind chill or heat index)
		if fl := feelsLikeTemp(c.WindChillC, c.HeatIndexC); fl != nil {
			label := styleLabel.Render("Feels like")
			val := tempStyle(*fl, imperial).Render(formatTemp(*fl, imperial))
			contents[1] = label + " " + val
		}

		// Slot 2: wind (with optional gusts)
		if c.WindKPH != nil {
			windStr := formatWind(*c.WindKPH, c.WindDegrees, imperial)
			if c.WindGustKPH != nil {
				windStr += ", gusts " + formatWindSpeed(*c.WindGustKPH, imperial)
			}
			contents[2] = styleLabel.Render("Wind:") + " " + styleValue.Render(windStr)
		}

		// Slot 3: humidity + dew point
		var humDew []string
		if c.HumidityPct != nil {
			humDew = append(humDew, styleLabel.Render("Humidity:")+" "+styleValue.Render(fmt.Sprintf("%.0f%%", *c.HumidityPct)))
		}
		if c.DewPointC != nil {
			humDew = append(humDew, styleLabel.Render("Dew point:")+" "+styleValue.Render(formatTemp(*c.DewPointC, imperial)))
		}
		if len(humDew) > 0 {
			contents[3] = strings.Join(humDew, "   ")
		}

		// Slot 4: pressure + visibility
		var presVis []string
		if c.PressureHPA != nil {
			presVis = append(presVis, styleLabel.Render("Pressure:")+" "+styleValue.Render(formatPressure(*c.PressureHPA, imperial)))
		}
		if c.VisibilityM != nil {
			presVis = append(presVis, styleLabel.Render("Visibility:")+" "+styleValue.Render(formatVisibility(*c.VisibilityM, imperial)))
		}
		if len(presVis) > 0 {
			contents[4] = strings.Join(presVis, "   ")
		}

		// Render icon lines alongside content slots.
		ic := getIcon(c.ConditionCode)
		for i := 0; i < 5; i++ {
			iconStr := lipgloss.NewStyle().
				Foreground(ic.colors[i]).
				Width(iconWidth).
				Render(ic.lines[i])
			fmt.Printf("  %s  %s\n", iconStr, contents[i])
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
			fmt.Println(styleAlertBanner.Render("⚠  " + headline))
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
		fmt.Println(styleForecastHeader.Render("Forecast"))
		fmt.Println(styleLabel.Render(strings.Repeat("─", 60)))

		for _, p := range data.Forecast.Periods {
			tempStr := formatTemp(p.TempC, imperial)
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

	return nil
}

// feelsLikeTemp returns WindChillC if set, HeatIndexC if set, otherwise nil.
// NWS only populates these fields when meteorologically applicable, so at most
// one will be non-nil for a given observation.
func feelsLikeTemp(windChill, heatIndex *float64) *float64 {
	if windChill != nil {
		return windChill
	}
	return heatIndex
}

// tempStyle returns the lipgloss style appropriate for the temperature.
// tempC is in Celsius; imperial flag is used for threshold comparison.
func tempStyle(tempC float64, imperial bool) lipgloss.Style {
	var ref float64
	if imperial {
		ref = celsiusToFahrenheit(tempC)
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

// formatTemp formats temperature for display with the appropriate unit suffix.
func formatTemp(tempC float64, imperial bool) string {
	if imperial {
		return fmt.Sprintf("%.0f°F", celsiusToFahrenheit(tempC))
	}
	return fmt.Sprintf("%.1f°C", tempC)
}

// formatWind formats wind speed and direction for display.
func formatWind(kph float64, degrees *float64, imperial bool) string {
	speed := formatWindSpeed(kph, imperial)
	if degrees != nil {
		return degreesToCompass(*degrees) + " " + speed
	}
	return speed
}

// formatWindSpeed formats wind speed only (no direction).
func formatWindSpeed(kph float64, imperial bool) string {
	if imperial {
		return fmt.Sprintf("%.0f mph", kphToMPH(kph))
	}
	return fmt.Sprintf("%.0f km/h", kph)
}

// formatPressure formats barometric pressure for display.
func formatPressure(hpa float64, imperial bool) string {
	if imperial {
		return fmt.Sprintf("%.2f inHg", hpa/33.8639)
	}
	return fmt.Sprintf("%.0f hPa", hpa)
}

// formatVisibility formats visibility distance for display.
func formatVisibility(meters float64, imperial bool) string {
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

// degreesToCompass converts 0–360 degrees to a compass direction string.
func degreesToCompass(deg float64) string {
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int((deg+11.25)/22.5) % 16
	return dirs[idx]
}

func celsiusToFahrenheit(c float64) float64 {
	return c*9/5 + 32
}

func kphToMPH(kph float64) float64 {
	return kph / 1.60934
}
