package monitor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mwirges/wx/internal/output"
)

// Layout constants.
const (
	headerLines        = 3  // title + location/status line + separator
	helpBarLines       = 1
	conditionsBlockLines = 7  // 5 icon rows + 2 spacer/extra rows
	maxAlertLines      = 3
	maxForecastPeriods = 10 // 5 days × day+night
)

// Styles.
var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleLoc      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleUpdated  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleSep      = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	styleLabel    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleValue    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleDesc     = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Italic(true)
	styleInputKey = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	styleInputVal = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleInputErr = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleErr      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleAlert    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("160"))
	styleHelpKey  = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	styleHelpDim  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	styleLoading  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)

	styleForecastHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleForecastHigh   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleForecastLow    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	styleForecastDesc   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleForecastName   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

// ── View ──────────────────────────────────────────────────────────────────────

func (m MonitorModel) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return styleLoading.Render("Loading…")
	}

	contentH := m.height - headerLines - helpBarLines
	if contentH < 1 {
		contentH = 1
	}

	var sb strings.Builder
	sb.WriteString(m.renderHeader())
	sb.WriteString("\n")
	if m.radarVisible {
		sb.WriteString(m.renderSplit(contentH))
	} else {
		sb.WriteString(m.renderFullWidth(contentH))
	}
	sb.WriteString("\n")
	sb.WriteString(m.renderHelpBar())

	return sb.String()
}

// ── Header (3 lines) ──────────────────────────────────────────────────────────

func (m MonitorModel) renderHeader() string {
	// Line 1: "wx monitor" title
	line1 := styleTitle.Render("wx monitor")

	// Line 2: location / input / status
	var line2 string
	if m.inputMode {
		prompt := styleInputKey.Render("Location: ") +
			styleInputVal.Render(m.inputText) +
			styleInputVal.Render("█")
		if m.inputErr != nil {
			prompt += "  " + styleInputErr.Render(m.inputErr.Error())
		}
		line2 = prompt
	} else {
		locPart := styleLoc.Render(m.loc.DisplayName)
		var statusPart string
		if m.weatherLoading {
			statusPart = styleUpdated.Render(" · refreshing…")
		} else if m.fetchErr != nil {
			statusPart = styleErr.Render(" · error: " + m.fetchErr.Error())
		} else if !m.lastFetch.IsZero() {
			statusPart = styleUpdated.Render(" · updated " + m.lastFetch.Local().Format("3:04 PM"))
		}
		line2 = locPart + statusPart
	}

	// Line 3: separator
	line3 := styleSep.Render(strings.Repeat("─", m.width))

	return strings.Join([]string{line1, line2, line3}, "\n")
}

// ── Full-width layout ─────────────────────────────────────────────────────────

func (m MonitorModel) renderFullWidth(contentH int) string {
	lines := m.buildWeatherLines(m.width, contentH)
	return strings.Join(normalizeLines(lines, contentH, m.width), "\n")
}

// ── Split layout ──────────────────────────────────────────────────────────────

func (m MonitorModel) renderSplit(contentH int) string {
	leftW := m.width / 2
	rightW := m.width - leftW

	leftLines := normalizeLines(m.buildWeatherLines(leftW, contentH), contentH, leftW)
	rightLines := normalizeLines(m.buildRadarLines(rightW, contentH), contentH, rightW)

	rows := make([]string, contentH)
	for i := 0; i < contentH; i++ {
		rows[i] = padToWidth(leftLines[i], leftW) + rightLines[i]
	}
	return strings.Join(rows, "\n")
}

// ── Weather panel ─────────────────────────────────────────────────────────────

func (m MonitorModel) buildWeatherLines(w, maxH int) []string {
	var lines []string

	// Alerts (loud red banners, up to maxAlertLines)
	lines = append(lines, m.renderAlerts(w)...)
	if len(lines) > 0 {
		lines = append(lines, "")
	}

	// Conditions block
	lines = append(lines, m.renderConditions(w)...)
	lines = append(lines, "")

	// Forecast
	forecastMax := maxH - len(lines) - 1 // -1 for forecast header
	if forecastMax > maxForecastPeriods {
		forecastMax = maxForecastPeriods
	}
	if forecastMax > 0 {
		lines = append(lines, m.renderForecast(w, forecastMax)...)
	}

	return lines
}

func (m MonitorModel) renderAlerts(w int) []string {
	if len(m.alerts) == 0 {
		return nil
	}
	alertStyle := styleAlert.Width(w)
	var lines []string
	for i, a := range m.alerts {
		if i >= maxAlertLines {
			break
		}
		headline := a.Headline
		if headline == "" {
			headline = a.Event
		}
		text := "⚠  " + a.Event
		if headline != a.Event && headline != "" {
			text += ": " + headline
		}
		lines = append(lines, alertStyle.Render(truncateStr(text, w)))
	}
	return lines
}

func (m MonitorModel) renderConditions(w int) []string {
	if m.conditions == nil {
		if m.weatherLoading {
			return padLines([]string{styleLoading.Render("Fetching weather…")}, conditionsBlockLines, w)
		}
		return padLines([]string{styleErr.Render("No conditions data")}, conditionsBlockLines, w)
	}

	c := m.conditions
	imperial := m.cfg.Imperial
	ic := output.GetIcon(c.ConditionCode)

	// Right-side content slots (5 to match icon height)
	slots := [5]string{}

	// Slot 0: temperature + description
	if c.TempC != nil {
		ts := output.TempStyle(*c.TempC, imperial).Bold(true).Render(output.FormatTemp(*c.TempC, imperial))
		if c.Description != "" {
			ts += "  " + styleDesc.Render(c.Description)
		}
		slots[0] = ts
	} else if c.Description != "" {
		slots[0] = styleDesc.Render(c.Description)
	}

	// Slot 1: feels like
	if fl := output.FeelsLikeTemp(c.WindChillC, c.HeatIndexC); fl != nil {
		slots[1] = styleLabel.Render("Feels like ") +
			output.TempStyle(*fl, imperial).Render(output.FormatTemp(*fl, imperial))
	}

	// Slot 2: wind
	if c.WindKPH != nil {
		windStr := output.FormatWind(*c.WindKPH, c.WindDegrees, imperial)
		if c.WindGustKPH != nil {
			windStr += "  gusts " + output.FormatWindSpeed(*c.WindGustKPH, imperial)
		}
		slots[2] = styleLabel.Render("Wind: ") + styleValue.Render(windStr)
	}

	// Slot 3: humidity + dew point
	var humDew []string
	if c.HumidityPct != nil {
		humDew = append(humDew, styleLabel.Render("Humidity: ")+styleValue.Render(fmt.Sprintf("%.0f%%", *c.HumidityPct)))
	}
	if c.DewPointC != nil {
		humDew = append(humDew, styleLabel.Render("Dew point: ")+styleValue.Render(output.FormatTemp(*c.DewPointC, imperial)))
	}
	if len(humDew) > 0 {
		slots[3] = strings.Join(humDew, "   ")
	}

	// Slot 4: pressure + visibility
	var presVis []string
	if c.PressureHPA != nil {
		presVis = append(presVis, styleLabel.Render("Pressure: ")+styleValue.Render(output.FormatPressure(*c.PressureHPA, imperial)))
	}
	if c.VisibilityM != nil {
		presVis = append(presVis, styleLabel.Render("Visibility: ")+styleValue.Render(output.FormatVisibility(*c.VisibilityM, imperial)))
	}
	if len(presVis) > 0 {
		slots[4] = strings.Join(presVis, "   ")
	}

	// Build 5 icon+content lines
	out := make([]string, 5)
	for i := 0; i < 5; i++ {
		iconStr := lipgloss.NewStyle().
			Foreground(ic.Colors[i]).
			Width(output.IconWidth).
			Render(ic.Lines[i])
		out[i] = "  " + iconStr + "  " + slots[i]
	}

	// Pad to conditionsBlockLines (7)
	return padLines(out, conditionsBlockLines, w)
}

func (m MonitorModel) renderForecast(w, maxLines int) []string {
	var lines []string

	// Scroll indicator header
	total := forecastLen(m.forecast)
	lines = append(lines, renderScrollHeader(m.forecastOffset, total, m.forecastVisible, w))

	if m.forecast == nil || total == 0 {
		lines = append(lines, styleUpdated.Render("  No forecast data"))
		return lines
	}

	end := m.forecastOffset + maxLines
	if end > total {
		end = total
	}
	periods := m.forecast.Periods[m.forecastOffset:end]

	for _, p := range periods {
		name := styleForecastName.Width(16).Render(p.Name)
		tempStr := output.FormatTemp(p.TempC, m.cfg.Imperial)
		var tempStyled string
		if p.IsDaytime {
			tempStyled = styleForecastHigh.Width(9).Render("↑ " + tempStr)
		} else {
			tempStyled = styleForecastLow.Width(9).Render("↓ " + tempStr)
		}
		desc := styleForecastDesc.Render(p.ShortDesc)
		line := "  " + name + " " + tempStyled + "  " + desc
		lines = append(lines, truncateStr(line, w))
	}

	return lines
}

func renderScrollHeader(offset, total, visible, w int) string {
	if total == 0 {
		return styleForecastHeader.Render("── Forecast" + strings.Repeat("─", max(0, w-11)))
	}

	upArrow := " "
	downArrow := " "
	if offset > 0 {
		upArrow = "▲"
	}
	if offset+visible < total {
		downArrow = "▼"
	}

	label := fmt.Sprintf("── Forecast %s %d–%d/%d %s ", upArrow, offset+1, min(offset+visible, total), total, downArrow)
	rest := w - lipgloss.Width(label)
	if rest > 0 {
		label += strings.Repeat("─", rest)
	}
	return styleForecastHeader.Render(label)
}

// ── Radar panel ───────────────────────────────────────────────────────────────

func (m MonitorModel) buildRadarLines(w, h int) []string {
	if m.radarLoading && len(m.radarLines) == 0 {
		return []string{styleLoading.Width(w).Render(" Loading radar…")}
	}
	if m.radarErr != nil {
		msg := styleErr.Width(w).Render(" Radar error: " + m.radarErr.Error())
		return []string{msg}
	}
	if len(m.radarLines) == 0 {
		return []string{styleLoading.Width(w).Render(" No radar data")}
	}
	return m.radarLines
}

// ── Help bar ──────────────────────────────────────────────────────────────────

func (m MonitorModel) renderHelpBar() string {
	dim := styleHelpDim
	key := styleHelpKey

	var parts []string
	if m.inputMode {
		parts = []string{
			key.Render("enter") + dim.Render(":confirm"),
			key.Render("esc") + dim.Render(":cancel"),
		}
	} else {
		radarLabel := "radar on"
		if m.radarVisible {
			radarLabel = "radar off"
		}
		parts = []string{
			key.Render("r") + dim.Render(":refresh"),
			key.Render("R") + dim.Render(":" + radarLabel),
			key.Render("l") + dim.Render(":location"),
			key.Render("↑↓") + dim.Render(":scroll"),
			key.Render("q") + dim.Render(":quit"),
		}
	}
	bar := strings.Join(parts, dim.Render("  │  "))
	return bar + "\x1b[K"
}

// ── Layout helpers ────────────────────────────────────────────────────────────

// padToWidth pads s with spaces to reach exactly w visible columns.
// Uses lipgloss.Width for ANSI-aware measurement.
func padToWidth(s string, w int) string {
	vis := lipgloss.Width(s)
	if vis >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vis)
}

// normalizeLines ensures lines is exactly h entries, padding with blanks or truncating.
func normalizeLines(lines []string, h, w int) []string {
	blank := strings.Repeat(" ", w)
	for len(lines) < h {
		lines = append(lines, blank)
	}
	return lines[:h]
}

// padLines pads a slice to exactly h lines by appending empty strings.
func padLines(lines []string, h, _ int) []string {
	for len(lines) < h {
		lines = append(lines, "")
	}
	return lines
}

// truncateStr truncates s to at most maxW visible characters.
func truncateStr(s string, maxW int) string {
	if lipgloss.Width(s) <= maxW {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > maxW {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
