package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mwirges/wx/internal/models"
)

// AQIOptions controls rendering of Air Quality reports.
type AQIOptions struct {
	ForceJSON   bool
	ForcePretty bool
}

var (
	styleAQIHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleAQISub    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleAQILabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleAQIVal    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleAQIDesc   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	styleAQIAdv    = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Italic(true)
	styleSmokeBox  = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("208")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("208")).
			Padding(0, 1)
)

// RenderAQI renders air quality telemetry to os.Stdout.
func RenderAQI(payload *models.AirQualityPayload, opts AQIOptions) error {
	return RenderAQITo(os.Stdout, payload, opts)
}

// RenderAQITo renders air quality telemetry to the given writer.
func RenderAQITo(w io.Writer, payload *models.AirQualityPayload, opts AQIOptions) error {
	if payload == nil || payload.AirQuality == nil {
		return fmt.Errorf("no air quality data")
	}

	if opts.ForceJSON || (!opts.ForcePretty && !isTTY) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	aq := payload.AirQuality

	// ── Header ──────────────────────────────────────────────────────────
	fmt.Fprintln(w, styleAQIHeader.Render("Air Quality Index (AQI) & Smoke Plume Console"))
	locStr := payload.Location
	if locStr == "" {
		locStr = fmt.Sprintf("%.2f°, %.2f°", payload.Latitude, payload.Longitude)
	} else {
		locStr = fmt.Sprintf("%s (%.2f°, %.2f°)", locStr, payload.Latitude, payload.Longitude)
	}
	if !payload.FetchedAt.IsZero() {
		locStr += fmt.Sprintf(" · %s", payload.FetchedAt.Local().Format("3:04 PM"))
	}
	fmt.Fprintln(w, styleAQISub.Render(locStr))
	fmt.Fprintln(w)

	// ── Primary AQI Readout ─────────────────────────────────────────────
	if aq.AQI != nil {
		aqiVal := *aq.AQI
		cat := aq.Category
		if cat == "" {
			cat = models.AQICategory(aqiVal)
		}

		badge := AQIStyle(aqiVal).Render(fmt.Sprintf("[%s]", strings.ToUpper(cat)))
		scoreStr := AQIStyle(aqiVal).Render(fmt.Sprintf("%d", aqiVal))
		gauge := renderAQIGauge(aqiVal, 24)

		fmt.Fprintf(w, "  %s  %s  %s\n", styleAQILabel.Render("US EPA AQI:"), scoreStr, badge)
		fmt.Fprintf(w, "  %s  [%s]  %s / 500\n", styleAQILabel.Render("Gauge:     "), gauge, scoreStr)

		advisory := payload.HealthAdvisory
		if advisory == "" {
			advisory = models.EPAHealthAdvisory(aqiVal)
		}
		if advisory != "" {
			fmt.Fprintf(w, "  %s  %s\n", styleAQILabel.Render("Advisory:  "), styleAQIAdv.Render(advisory))
		}
		fmt.Fprintln(w)
	}

	// ── Smoke Plume Alert ───────────────────────────────────────────────
	smokeAdv := payload.SmokeAdvisory
	if smokeAdv == "" && aq.PM25 != nil {
		smokeAdv = models.SmokeAdvisory(*aq.PM25)
	}
	if smokeAdv != "" {
		fmt.Fprintln(w, styleSmokeBox.Render("💨 SMOKE PLUME ADVISORY: "+smokeAdv))
		fmt.Fprintln(w)
	}

	// ── UV Index Readout ────────────────────────────────────────────────
	if aq.UVIndex != nil {
		uvVal := *aq.UVIndex
		uvCat := aq.UVCategory
		if uvCat == "" {
			uvCat = models.UVCategory(uvVal)
		}
		uvBadge := UVStyle(uvVal).Render(fmt.Sprintf("[%s]", strings.ToUpper(uvCat)))
		uvScore := UVStyle(uvVal).Render(fmt.Sprintf("%.1f", uvVal))
		uvGauge := renderUVGauge(uvVal, 16)

		fmt.Fprintf(w, "  %s  %s  %s\n", styleAQILabel.Render("WHO UV Index:"), uvScore, uvBadge)
		fmt.Fprintf(w, "  %s  [%s]  %s / 11+\n", styleAQILabel.Render("Sun Gauge:   "), uvGauge, uvScore)
		fmt.Fprintln(w)
	}

	// ── Detailed Pollutant Breakdown ────────────────────────────────────
	fmt.Fprintln(w, styleAQIHeader.Render("Atmospheric Chemistry & Pollutant Telemetry"))
	fmt.Fprintf(w, "  %-24s %-16s %s\n",
		styleAQILabel.Render("POLLUTANT"),
		styleAQILabel.Render("CONCENTRATION"),
		styleAQILabel.Render("AIR QUALITY STANDARD / IMPACT"),
	)
	fmt.Fprintf(w, "  %-24s %-16s %s\n",
		styleAQISub.Render("────────────────────────"),
		styleAQISub.Render("──────────────"),
		styleAQISub.Render("─────────────────────────────"),
	)

	renderPollutantRow(w, "PM2.5 (Fine Particulates)", aq.PM25, "μg/m³", pm25Impact(aq.PM25))
	renderPollutantRow(w, "PM10 (Inhalable Dust)", aq.PM10, "μg/m³", pm10Impact(aq.PM10))
	renderPollutantRow(w, "Ozone (O3 Ground-Level)", aq.O3, "μg/m³", o3Impact(aq.O3))
	renderPollutantRow(w, "Nitrogen Dioxide (NO2)", aq.NO2, "μg/m³", no2Impact(aq.NO2))
	renderPollutantRow(w, "Carbon Monoxide (CO)", aq.CO, "μg/m³", coImpact(aq.CO))
	renderPollutantRow(w, "Sulphur Dioxide (SO2)", aq.SO2, "μg/m³", so2Impact(aq.SO2))

	fmt.Fprintln(w)

	// ── Attribution ─────────────────────────────────────────────────────
	source := payload.Source
	if source == "" {
		source = "Open-Meteo Atmospheric Model (Copernicus CAMS) · US EPA AirNow Guidelines"
	}
	fmt.Fprintf(w, "  %s %s\n", styleAQISub.Render("Source:"), styleAQISub.Render(source))

	return nil
}

func renderPollutantRow(w io.Writer, name string, val *float64, unit string, impact string) {
	if val == nil {
		fmt.Fprintf(w, "  %-24s %-16s %s\n",
			styleAQIDesc.Render(name),
			styleAQISub.Render("—"),
			styleAQISub.Render("unreported"),
		)
		return
	}
	valStr := fmt.Sprintf("%.1f %s", *val, unit)
	fmt.Fprintf(w, "  %-24s %-16s %s\n",
		styleAQIVal.Render(name),
		styleAQIDesc.Render(valStr),
		styleAQISub.Render(impact),
	)
}

func pm25Impact(v *float64) string {
	if v == nil {
		return ""
	}
	val := *v
	switch {
	case val <= 9.0:
		return "Good · minimal health risk"
	case val <= 35.4:
		return "Moderate · ambient background"
	case val <= 55.4:
		return "Elevated · smoke / haze detected"
	case val <= 125.4:
		return "Unhealthy · active smoke plume"
	default:
		return "Hazardous · heavy wildfire plume"
	}
}

func pm10Impact(v *float64) string {
	if v == nil {
		return ""
	}
	val := *v
	switch {
	case val <= 54.0:
		return "Good · ambient dust"
	case val <= 154.0:
		return "Moderate · blowing dust / haze"
	default:
		return "Unhealthy · elevated particulate matter"
	}
}

func o3Impact(v *float64) string {
	if v == nil {
		return ""
	}
	val := *v
	switch {
	case val <= 100.0:
		return "Good · photochemical balance"
	case val <= 160.0:
		return "Moderate · warm sunny stagnation"
	default:
		return "Unhealthy · photochemical smog"
	}
}

func no2Impact(v *float64) string {
	if v == nil {
		return ""
	}
	val := *v
	switch {
	case val <= 40.0:
		return "Good · clean background"
	case val <= 100.0:
		return "Moderate · vehicle combustion"
	default:
		return "Unhealthy · heavy traffic exhaust"
	}
}

func coImpact(v *float64) string {
	if v == nil {
		return ""
	}
	val := *v
	switch {
	case val <= 4000.0:
		return "Good · clean background"
	default:
		return "Elevated · combustion source"
	}
}

func so2Impact(v *float64) string {
	if v == nil {
		return ""
	}
	val := *v
	switch {
	case val <= 20.0:
		return "Good · clean background"
	default:
		return "Elevated · industrial combustion"
	}
}

func renderAQIGauge(aqi int, width int) string {
	if width <= 0 {
		width = 20
	}
	ratio := float64(aqi) / 500.0
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0 {
		ratio = 0
	}
	filled := int(ratio * float64(width))
	if filled < 1 && aqi > 0 {
		filled = 1
	}
	unfilled := width - filled
	if unfilled < 0 {
		unfilled = 0
	}

	return AQIStyle(aqi).Render(strings.Repeat("■", filled)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("□", unfilled))
}

func renderUVGauge(uv float64, width int) string {
	if width <= 0 {
		width = 12
	}
	ratio := uv / 11.0
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0 {
		ratio = 0
	}
	filled := int(ratio * float64(width))
	if filled < 1 && uv > 0 {
		filled = 1
	}
	unfilled := width - filled
	if unfilled < 0 {
		unfilled = 0
	}

	return UVStyle(uv).Render(strings.Repeat("■", filled)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("□", unfilled))
}
