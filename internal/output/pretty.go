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
		if fl := FeelsLikeTemp(c.WindChillC, c.HeatIndexC); fl != nil {
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

	return nil
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
