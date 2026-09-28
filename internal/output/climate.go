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

// ClimateOptions controls rendering of climate normals and records reports.
type ClimateOptions struct {
	ForceJSON   bool
	ForcePretty bool
	Units       string // "imperial" or "metric"
}

var (
	styleClimateHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleClimateSub    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleClimateLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleClimateVal    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleClimateCyan   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styleClimateGreen  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	styleClimateYellow = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	styleClimateRed    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	styleClimateOrange = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	styleClimateDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// RenderClimate renders climate normals and records to os.Stdout.
func RenderClimate(payload *models.ClimatePayload, opts ClimateOptions) error {
	return RenderClimateTo(os.Stdout, payload, opts)
}

// RenderClimateTo renders climate normals and records to the given writer.
func RenderClimateTo(w io.Writer, payload *models.ClimatePayload, opts ClimateOptions) error {
	if payload == nil || payload.Climate == nil {
		return fmt.Errorf("no climate data")
	}

	if opts.ForceJSON || (!opts.ForcePretty && !isTTY) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	cl := payload.Climate
	isMetric := strings.ToLower(opts.Units) == "metric"

	// ── Header ──────────────────────────────────────────────────────────
	fmt.Fprintln(w, styleClimateHeader.Render("NOAA 30-Year Climate Normals & Historical Records"))

	var subParts []string
	if cl.StationName != "" {
		stnStr := cl.StationName
		if cl.StationID != "" {
			stnStr += fmt.Sprintf(" (%s)", cl.StationID)
		}
		subParts = append(subParts, stnStr)
	} else if payload.Location != "" {
		subParts = append(subParts, payload.Location)
	}

	if isMetric && cl.ElevationM > 0 {
		subParts = append(subParts, fmt.Sprintf("Elev %.0f m", cl.ElevationM))
	} else if !isMetric && cl.ElevationFt > 0 {
		subParts = append(subParts, fmt.Sprintf("Elev %.0f ft", cl.ElevationFt))
	}

	if !cl.Date.IsZero() {
		subParts = append(subParts, cl.Date.Local().Format("Mon Jan 2"))
	}
	if cl.NormalsPeriod != "" {
		subParts = append(subParts, fmt.Sprintf("Base: %s", cl.NormalsPeriod))
	}

	fmt.Fprintln(w, styleClimateSub.Render(strings.Join(subParts, " · ")))
	fmt.Fprintln(w)

	// ── Departure Anomaly Status Banner ──────────────────────────────────
	if cl.Departure != nil {
		var badge string
		diffF := 0.0
		if cl.Departure.DepartureCurrentF != nil {
			diffF = *cl.Departure.DepartureCurrentF
		}

		if diffF >= 4.0 {
			badge = styleClimateRed.Render("[WARM ANOMALY]")
		} else if diffF <= -4.0 {
			badge = styleClimateCyan.Render("[COLD ANOMALY]")
		} else {
			badge = styleClimateGreen.Render("[NEAR NORMAL]")
		}

		fmt.Fprintf(w, "%s %s\n", badge, styleClimateVal.Render(cl.Departure.Summary))
		fmt.Fprintln(w)
	}

	// ── Daily Normals (1991–2020) ─────────────────────────────────────────
	var highStr, lowStr, meanStr, precipStr string
	if isMetric {
		highStr = fmt.Sprintf("%.1f°C", cl.TodayNormals.NormalHighC)
		lowStr = fmt.Sprintf("%.1f°C", cl.TodayNormals.NormalLowC)
		meanStr = fmt.Sprintf("%.1f°C", cl.TodayNormals.NormalMeanC)
		precipStr = fmt.Sprintf("%.1f mm", cl.TodayNormals.NormalPrecipMM)
	} else {
		highStr = fmt.Sprintf("%.1f°F", cl.TodayNormals.NormalHighF)
		lowStr = fmt.Sprintf("%.1f°F", cl.TodayNormals.NormalLowF)
		meanStr = fmt.Sprintf("%.1f°F", cl.TodayNormals.NormalMeanF)
		precipStr = fmt.Sprintf("%.2f in", cl.TodayNormals.NormalPrecipIn)
	}

	fmt.Fprintln(w, styleClimateHeader.Render("DAILY NORMALS (30-YEAR BASELINE)"))
	fmt.Fprintf(w, "%s %s    %s %s    %s %s    %s %s\n",
		styleClimateLabel.Render("Normal High:"), styleClimateOrange.Render(highStr),
		styleClimateLabel.Render("Normal Low:"), styleClimateCyan.Render(lowStr),
		styleClimateLabel.Render("Normal Mean:"), styleClimateVal.Render(meanStr),
		styleClimateLabel.Render("Normal Precip:"), styleClimateGreen.Render(precipStr),
	)
	fmt.Fprintln(w)

	// ── Historical Extremes & Records ────────────────────────────────────
	rec := cl.Records
	if rec.TotalYearsSampled > 0 || rec.RecordHigh.ValueF > -900 {
		fmt.Fprintln(w, styleClimateHeader.Render("ALL-TIME DAILY RECORDS FOR THIS CALENDAR DAY"))

		var recHighStr, recLowStr, recPrecipStr, coldHighStr, warmLowStr string
		if isMetric {
			recHighStr = formatRecordVal(rec.RecordHigh.ValueC, "°C", rec.RecordHigh.Years)
			recLowStr = formatRecordVal(rec.RecordLow.ValueC, "°C", rec.RecordLow.Years)
			coldHighStr = formatRecordVal(rec.ColdestHigh.ValueC, "°C", rec.ColdestHigh.Years)
			warmLowStr = formatRecordVal(rec.WarmestLow.ValueC, "°C", rec.WarmestLow.Years)
			recPrecipStr = formatRecordVal(rec.RecordPrecip.ValueMM, " mm", rec.RecordPrecip.Years)
		} else {
			recHighStr = formatRecordVal(rec.RecordHigh.ValueF, "°F", rec.RecordHigh.Years)
			recLowStr = formatRecordVal(rec.RecordLow.ValueF, "°F", rec.RecordLow.Years)
			coldHighStr = formatRecordVal(rec.ColdestHigh.ValueF, "°F", rec.ColdestHigh.Years)
			warmLowStr = formatRecordVal(rec.WarmestLow.ValueF, "°F", rec.WarmestLow.Years)
			recPrecipStr = formatRecordVal(rec.RecordPrecip.ValueIn, " in", rec.RecordPrecip.Years)
		}

		fmt.Fprintf(w, "%-22s %s\n", styleClimateLabel.Render("All-Time Record High:"), styleClimateRed.Render(recHighStr))
		fmt.Fprintf(w, "%-22s %s\n", styleClimateLabel.Render("All-Time Record Low:"), styleClimateCyan.Render(recLowStr))
		if rec.ColdestHigh.ValueF < 900 {
			fmt.Fprintf(w, "%-22s %s\n", styleClimateLabel.Render("Coldest Daytime High:"), styleClimateSub.Render(coldHighStr))
		}
		if rec.WarmestLow.ValueF > -900 {
			fmt.Fprintf(w, "%-22s %s\n", styleClimateLabel.Render("Warmest Nighttime Low:"), styleClimateSub.Render(warmLowStr))
		}
		if rec.RecordPrecip.ValueIn >= 0 {
			fmt.Fprintf(w, "%-22s %s\n", styleClimateLabel.Render("Max Daily Precip:"), styleClimateGreen.Render(recPrecipStr))
		}

		if rec.PeriodOfRecord != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, styleClimateDim.Render(fmt.Sprintf("Historical Archive: %s (%d years evaluated)", rec.PeriodOfRecord, rec.TotalYearsSampled)))
		}
		fmt.Fprintln(w)
	}

	// ── Monthly Normals Context ──────────────────────────────────────────
	if cl.MonthlyNormals != nil {
		mn := cl.MonthlyNormals
		var mHighStr, mLowStr, mPrecipStr string
		if isMetric {
			mHighStr = fmt.Sprintf("%.1f°C", mn.NormalAvgHighC)
			mLowStr = fmt.Sprintf("%.1f°C", mn.NormalAvgLowC)
			mPrecipStr = fmt.Sprintf("%.1f mm", mn.NormalTotalPrecipMM)
		} else {
			mHighStr = fmt.Sprintf("%.1f°F", mn.NormalAvgHighF)
			mLowStr = fmt.Sprintf("%.1f°F", mn.NormalAvgLowF)
			mPrecipStr = fmt.Sprintf("%.2f in", mn.NormalTotalPrecipIn)
		}

		fmt.Fprintln(w, styleClimateHeader.Render(fmt.Sprintf("%s MONTHLY NORMALS SUMMARY", strings.ToUpper(mn.MonthName))))
		fmt.Fprintf(w, "%s %s    %s %s    %s %s\n",
			styleClimateLabel.Render("Monthly Avg High:"), styleClimateVal.Render(mHighStr),
			styleClimateLabel.Render("Monthly Avg Low:"), styleClimateVal.Render(mLowStr),
			styleClimateLabel.Render("Total Monthly Precip:"), styleClimateVal.Render(mPrecipStr),
		)
	}

	return nil
}

func formatRecordVal(val float64, unit string, years []int) string {
	valStr := fmt.Sprintf("%.1f%s", val, unit)
	if strings.Contains(unit, "in") {
		valStr = fmt.Sprintf("%.2f%s", val, unit)
	}
	if len(years) == 0 {
		return valStr
	}
	var yearStrs []string
	for _, y := range years {
		yearStrs = append(yearStrs, fmt.Sprintf("%d", y))
	}
	return fmt.Sprintf("%s (%s)", valStr, strings.Join(yearStrs, ", "))
}
