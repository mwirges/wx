package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mwirges/wx/internal/models"
)

// AstroOptions controls rendering of astronomical ephemeris reports.
type AstroOptions struct {
	ForceJSON   bool
	ForcePretty bool
}

var (
	styleAstroHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleAstroSub    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleAstroLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleAstroVal    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleAstroGold   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	styleAstroCyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	styleAstroOrange = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	styleAstroDesc   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	styleAstroDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// RenderAstro renders astronomical ephemeris to os.Stdout.
func RenderAstro(payload *models.AstroPayload, opts AstroOptions) error {
	return RenderAstroTo(os.Stdout, payload, opts)
}

// RenderAstroTo renders astronomical ephemeris to the given writer.
func RenderAstroTo(w io.Writer, payload *models.AstroPayload, opts AstroOptions) error {
	if payload == nil || payload.Astronomy == nil {
		return fmt.Errorf("no astronomy data")
	}

	if opts.ForceJSON || (!opts.ForcePretty && !isTTY) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	a := payload.Astronomy

	// ── Header ──────────────────────────────────────────────────────────
	fmt.Fprintln(w, styleAstroHeader.Render("Solar Arc, Ephemeris & Lunar HUD"))
	locStr := payload.Location
	if locStr == "" {
		locStr = fmt.Sprintf("%.2f°, %.2f°", payload.Latitude, payload.Longitude)
	} else {
		locStr = fmt.Sprintf("%s (%.2f°, %.2f°)", locStr, payload.Latitude, payload.Longitude)
	}
	if !payload.CalculatedAt.IsZero() {
		locStr += fmt.Sprintf(" · %s", payload.CalculatedAt.Local().Format("Mon Jan 2, 3:04 PM"))
	}
	fmt.Fprintln(w, styleAstroSub.Render(locStr))
	fmt.Fprintln(w)

	// ── Solar Positioning & Current Regime ──────────────────────────────
	var elevStr, azStr, periodBadge string
	if a.SolarElevationDeg != nil {
		elev := *a.SolarElevationDeg
		if elev >= 0 {
			elevStr = fmt.Sprintf("+%.1f°", elev)
		} else {
			elevStr = fmt.Sprintf("%.1f°", elev)
		}
	} else {
		elevStr = "—"
	}

	if a.SolarAzimuthDeg != nil {
		az := *a.SolarAzimuthDeg
		compass := AzimuthToCompass(az)
		azStr = fmt.Sprintf("%.0f° (%s)", az, compass)
	} else {
		azStr = "—"
	}

	period := a.CurrentPeriod
	if period == "" {
		period = "Nominal"
	}
	switch period {
	case "Daylight":
		periodBadge = styleAstroGold.Render("[" + strings.ToUpper(period) + "]")
	case "Golden Hour":
		periodBadge = styleAstroOrange.Render("[" + strings.ToUpper(period) + "]")
	case "Civil Twilight", "Nautical Twilight", "Astronomical Twilight":
		periodBadge = styleAstroCyan.Render("[" + strings.ToUpper(period) + "]")
	default:
		periodBadge = styleAstroSub.Render("[" + strings.ToUpper(period) + "]")
	}

	fmt.Fprintf(w, "  %s  %s    %s  %s    %s  %s\n",
		styleAstroLabel.Render("Solar Elevation:"), styleAstroVal.Render(elevStr),
		styleAstroLabel.Render("Azimuth:"), styleAstroVal.Render(azStr),
		styleAstroLabel.Render("Regime:"), periodBadge,
	)

	// Solar Trajectory Gauge
	renderSolarArc(w, a, payload.CalculatedAt)
	fmt.Fprintln(w)

	// ── Solar Horizon Schedule & Twilights ──────────────────────────────
	fmt.Fprintln(w, styleAstroHeader.Render("Solar Schedule & Twilight Horizons"))
	fmt.Fprintf(w, "  %-24s %-12s %s\n",
		styleAstroLabel.Render("EVENT / HORIZON"),
		styleAstroLabel.Render("LOCAL TIME"),
		styleAstroLabel.Render("SOLAR ZENITH / CHARACTERISTICS"),
	)
	fmt.Fprintf(w, "  %-24s %-12s %s\n",
		styleAstroSub.Render("────────────────────────"),
		styleAstroSub.Render("────────────"),
		styleAstroSub.Render("──────────────────────────────────────"),
	)

	renderHorizonRow(w, "Astronomical Dawn", a.AstroDawn, "108° zenith (-18° elev) · First optical sunlight")
	renderHorizonRow(w, "Nautical Dawn", a.NauticalDawn, "102° zenith (-12° elev) · Sea horizon becomes distinct")
	renderHorizonRow(w, "Civil Dawn", a.CivilDawn, "96° zenith (-6° elev)   · Outdoor activities without light")

	if a.Sunrise != nil {
		renderHorizonRow(w, "Sunrise", a.Sunrise, "90.8° zenith (0° elev)  · Upper edge of solar disc crests")
	} else if a.IsPolarNight {
		renderHorizonRow(w, "Sunrise", nil, "Polar Night · Sun remains below horizon")
	} else if a.IsPolarDay {
		renderHorizonRow(w, "Sunrise", nil, "Polar Day · Midnight sun remains above horizon")
	}

	if a.GoldenHourMorningStart != nil && a.GoldenHourMorningEnd != nil {
		ghMorn := fmt.Sprintf("%s – %s", formatTime(a.GoldenHourMorningStart), formatTime(a.GoldenHourMorningEnd))
		fmt.Fprintf(w, "  %-24s %-12s %s\n",
			styleAstroOrange.Render("Morning Golden Hour"),
			styleAstroVal.Render(ghMorn),
			styleAstroDesc.Render("Soft warm directional photography light"),
		)
	}

	if !a.SolarNoon.IsZero() {
		sn := a.SolarNoon
		renderHorizonRow(w, "Solar Noon (Transit)", &sn, "Sun crosses local meridian at maximum elevation")
	}

	if a.GoldenHourEveningStart != nil && a.GoldenHourEveningEnd != nil {
		ghEve := fmt.Sprintf("%s – %s", formatTime(a.GoldenHourEveningStart), formatTime(a.GoldenHourEveningEnd))
		fmt.Fprintf(w, "  %-24s %-12s %s\n",
			styleAstroOrange.Render("Evening Golden Hour"),
			styleAstroVal.Render(ghEve),
			styleAstroDesc.Render("Golden hour sunset rays"),
		)
	}

	if a.Sunset != nil {
		renderHorizonRow(w, "Sunset", a.Sunset, "90.8° zenith (0° elev)  · Upper edge of solar disc dips")
	}
	renderHorizonRow(w, "Civil Dusk", a.CivilDusk, "96° zenith (-6° elev)   · Street lamps illuminate")
	renderHorizonRow(w, "Nautical Dusk", a.NauticalDusk, "102° zenith (-12° elev) · Navigational stars visible")
	renderHorizonRow(w, "Astronomical Dusk", a.AstroDusk, "108° zenith (-18° elev) · Total astronomical night begins")

	if a.DayLength > 0 {
		dayHrs := int(a.DayLength.Hours())
		dayMins := int(a.DayLength.Minutes()) % 60
		dayStr := fmt.Sprintf("%dh %02dm", dayHrs, dayMins)

		nightDur := 24*time.Hour - a.DayLength
		nightHrs := int(nightDur.Hours())
		nightMins := int(nightDur.Minutes()) % 60
		nightStr := fmt.Sprintf("%dh %02dm", nightHrs, nightMins)

		fmt.Fprintf(w, "  %-24s %-12s %s\n",
			styleAstroGold.Render("Daylight Duration"),
			styleAstroVal.Render(dayStr),
			styleAstroDesc.Render(fmt.Sprintf("Night: %s", nightStr)),
		)
	}

	fmt.Fprintln(w)

	// ── Lunar Ephemeris & Moon Phase Cycle ──────────────────────────────
	fmt.Fprintln(w, styleAstroHeader.Render("Lunar Ephemeris & Phase Cycle"))
	icon := a.MoonPhaseIcon
	if icon == "" {
		icon = "🌖"
	}
	phase := a.MoonPhase
	if phase == "" {
		phase = "Unknown"
	}

	illumStr := a.MoonIlluminationStr()
	if illumStr == "" {
		illumStr = "—"
	}
	ageStr := a.MoonAgeStr()
	if ageStr == "" {
		ageStr = "—"
	}

	fmt.Fprintf(w, "  %s  %s %s    %s  %s    %s  %s / 29.53d\n",
		styleAstroLabel.Render("Moon Phase:"), icon, styleAstroVal.Render(phase),
		styleAstroLabel.Render("Illumination:"), styleAstroVal.Render(illumStr),
		styleAstroLabel.Render("Lunar Age:"), styleAstroVal.Render(ageStr),
	)

	renderLunarCycleMeter(w, a.MoonAgeDays)
	fmt.Fprintln(w)

	// Attribution footer
	fmt.Fprintf(w, "  %s %s\n", styleAstroSub.Render("Algorithm:"),
		styleAstroSub.Render("NOAA Solar Calculations & Meeus Astronomical Algorithms · 100% Offline"))

	return nil
}

func renderHorizonRow(w io.Writer, name string, t *time.Time, desc string) {
	timeStr := formatTime(t)
	fmt.Fprintf(w, "  %-24s %-12s %s\n",
		styleAstroVal.Render(name),
		styleAstroCyan.Render(timeStr),
		styleAstroDesc.Render(desc),
	)
}

func formatTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "—"
	}
	return t.Local().Format("3:04 PM")
}

// AzimuthToCompass converts an azimuth angle (0-360) to a 16-point cardinal compass string.
func AzimuthToCompass(az float64) string {
	directions := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int((az + 11.25) / 22.5) % 16
	if idx < 0 {
		idx += 16
	}
	return directions[idx]
}

func renderSolarArc(w io.Writer, a *models.Astronomy, now time.Time) {
	if a.Sunrise == nil || a.Sunset == nil {
		return
	}
	sr := *a.Sunrise
	ss := *a.Sunset
	if now.IsZero() {
		now = time.Now()
	}

	// 36-segment day bar: [Night] [Twilight] [Sunrise] ... [Sunset] [Twilight] [Night]
	var progress float64
	totalDay := ss.Sub(sr).Seconds()
	if totalDay > 0 {
		elapsed := now.Sub(sr).Seconds()
		progress = elapsed / totalDay
	}

	barWidth := 32
	var sunMarkerPos int
	switch {
	case progress < 0:
		sunMarkerPos = -1
	case progress > 1.0:
		sunMarkerPos = barWidth + 1
	default:
		sunMarkerPos = int(progress * float64(barWidth))
	}

	var sb strings.Builder
	for i := 0; i <= barWidth; i++ {
		if i == sunMarkerPos {
			sb.WriteString(styleAstroGold.Render("☀️"))
		} else if i == 0 {
			sb.WriteString(styleAstroCyan.Render("●")) // Sunrise
		} else if i == barWidth {
			sb.WriteString(styleAstroOrange.Render("●")) // Sunset
		} else {
			sb.WriteString(styleAstroDim.Render("─"))
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s  %s  %s  %s\n",
		styleAstroCyan.Render("Sunrise "+sr.Local().Format("3:04 PM")),
		sb.String(),
		styleAstroOrange.Render("Sunset "+ss.Local().Format("3:04 PM")),
		styleAstroDesc.Render("Solar Arc"),
	)
}

func renderLunarCycleMeter(w io.Writer, ageDays *float64) {
	if ageDays == nil {
		return
	}
	age := *ageDays
	cycle := 29.530588853
	ratio := age / cycle
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}

	halfWidth := 12
	totalWidth := halfWidth * 2
	markerPos := int(ratio * float64(totalWidth))

	var part1, part2 strings.Builder
	for i := 0; i < halfWidth; i++ {
		if i == markerPos {
			part1.WriteString(styleAstroGold.Render("●"))
		} else {
			part1.WriteString(styleAstroDim.Render("─"))
		}
	}
	for i := halfWidth; i <= totalWidth; i++ {
		if i == markerPos {
			part2.WriteString(styleAstroGold.Render("●"))
		} else {
			part2.WriteString(styleAstroDim.Render("─"))
		}
	}

	fmt.Fprintf(w, "  Cycle: [ 🌑 %s 🌕 %s 🌑 ]\n", part1.String(), part2.String())
}
