package output

import (
	"fmt"
	"math"
	"strings"
)

// FormatShort returns a single-line compact weather summary.
func FormatShort(data RenderData, opts RenderOptions) string {
	imperial := opts.Units != "metric"

	var parts []string

	if data.Conditions != nil {
		c := data.Conditions

		// 1. Primary: Location, Temperature, (Feels like), Description
		var leadParts []string
		loc := c.Location
		if loc == "" {
			loc = c.StationID
		}

		var tempDesc string
		if c.TempC != nil {
			tempDesc = FormatTemp(*c.TempC, imperial)

			// Feels like
			if fl := FeelsLikeTemp(c.WindChillC, c.HeatIndexC); fl != nil {
				actual := *c.TempC
				if imperial {
					actual = CelsiusToFahrenheit(actual)
					flVal := CelsiusToFahrenheit(*fl)
					if math.Abs(flVal-actual) >= 1.0 {
						tempDesc += fmt.Sprintf(" (feels %.0f°)", flVal)
					}
				} else {
					if math.Abs(*fl-actual) >= 1.0 {
						tempDesc += fmt.Sprintf(" (feels %.1f°)", *fl)
					}
				}
			}

			if c.Description != "" {
				tempDesc += " " + c.Description
			}
		} else if c.Description != "" {
			tempDesc = c.Description
		}

		if loc != "" && tempDesc != "" {
			leadParts = append(leadParts, fmt.Sprintf("%s: %s", loc, tempDesc))
		} else if loc != "" {
			leadParts = append(leadParts, loc)
		} else if tempDesc != "" {
			leadParts = append(leadParts, tempDesc)
		}

		if len(leadParts) > 0 {
			parts = append(parts, leadParts...)
		}

		// 2. Wind
		if c.WindKPH != nil {
			wStr := FormatWind(*c.WindKPH, c.WindDegrees, imperial)
			if c.WindGustKPH != nil {
				wStr += fmt.Sprintf(" (gusts %s)", FormatWindSpeed(*c.WindGustKPH, imperial))
			}
			parts = append(parts, wStr)
		}

		// 3. Humidity & Precipitation
		var humPrecip []string
		if c.HumidityPct != nil {
			humPrecip = append(humPrecip, fmt.Sprintf("%.0f%% hum", *c.HumidityPct))
		}
		if data.Forecast != nil && len(data.Forecast.Periods) > 0 {
			p := data.Forecast.Periods[0]
			if p.ProbabilityOfPrecipitation != nil && *p.ProbabilityOfPrecipitation > 0 {
				humPrecip = append(humPrecip, fmt.Sprintf("%.0f%% precip", *p.ProbabilityOfPrecipitation))
			}
		}
		if len(humPrecip) > 0 {
			parts = append(parts, strings.Join(humPrecip, " · "))
		}

		// 4. Astronomy
		if c.Astronomy != nil {
			astro := c.Astronomy
			if astro.IsPolarDay {
				parts = append(parts, "24h sun")
			} else if astro.IsPolarNight {
				parts = append(parts, "polar night")
			} else if astro.Sunrise != nil && astro.Sunset != nil {
				sr := astro.Sunrise.Local().Format("3:04 PM")
				ss := astro.Sunset.Local().Format("3:04 PM")
				parts = append(parts, fmt.Sprintf("↑%s ↓%s", sr, ss))
			}
		}
	} else if data.Forecast != nil && len(data.Forecast.Periods) > 0 {
		p := data.Forecast.Periods[0]
		part := fmt.Sprintf("%s: %s", p.Name, FormatTemp(p.TempC, imperial))
		if p.ShortDesc != "" {
			part += " " + p.ShortDesc
		}
		parts = append(parts, part)

		if p.WindKPH > 0 {
			parts = append(parts, FormatWind(p.WindKPH, nil, imperial))
		}
		if p.ProbabilityOfPrecipitation != nil && *p.ProbabilityOfPrecipitation > 0 {
			parts = append(parts, fmt.Sprintf("%.0f%% precip", *p.ProbabilityOfPrecipitation))
		}
	}

	// 5. Alerts
	if len(data.Alerts) > 0 {
		if len(data.Alerts) == 1 {
			parts = append(parts, fmt.Sprintf("⚠️ %s", data.Alerts[0].Event))
		} else {
			parts = append(parts, fmt.Sprintf("⚠️ %d alerts", len(data.Alerts)))
		}
	}

	return strings.Join(parts, " · ")
}

func renderShort(data RenderData, opts RenderOptions) error {
	out := FormatShort(data, opts)
	if out == "" {
		return fmt.Errorf("no weather data available")
	}
	fmt.Println(out)
	return nil
}
