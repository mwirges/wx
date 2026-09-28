package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/nowcast"
)

// NowcastOptions controls rendering of precipitation nowcast reports.
type NowcastOptions struct {
	ForceJSON   bool
	ForcePretty bool
	Units       string // "imperial" or "metric"
}

var (
	styleNowcastHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleNowcastSub    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleNowcastLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleNowcastVal    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleNowcastCyan   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styleNowcastGreen  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	styleNowcastYellow = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	styleNowcastRed    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	styleNowcastBlue   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75"))
	styleNowcastDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// RenderNowcast renders precipitation nowcast telemetry to os.Stdout.
func RenderNowcast(payload *models.NowcastPayload, opts NowcastOptions) error {
	return RenderNowcastTo(os.Stdout, payload, opts)
}

// RenderNowcastTo renders precipitation nowcast telemetry to the given writer.
func RenderNowcastTo(w io.Writer, payload *models.NowcastPayload, opts NowcastOptions) error {
	if payload == nil || payload.Nowcast == nil {
		return fmt.Errorf("no nowcast data")
	}

	if opts.ForceJSON || (!opts.ForcePretty && !isTTY) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	nc := payload.Nowcast
	isMetric := strings.ToLower(opts.Units) == "metric"

	// ── Header ──────────────────────────────────────────────────────────
	fmt.Fprintln(w, styleNowcastHeader.Render("Quantitative Precipitation Nowcast & Rain Timeline"))
	locStr := payload.Location
	if locStr == "" {
		locStr = nc.Location
	}
	if !nc.GeneratedAt.IsZero() {
		locStr += fmt.Sprintf(" · %s", nc.GeneratedAt.Local().Format("Mon Jan 2, 3:04 PM"))
	}
	fmt.Fprintln(w, styleNowcastSub.Render(locStr))
	fmt.Fprintln(w)

	// ── Status Banner ───────────────────────────────────────────────────
	var badge string
	if nc.IsActivePrecip {
		badge = styleNowcastRed.Render("[ACTIVE PRECIPITATION]")
	} else if nc.NextPrecipTime != nil {
		badge = styleNowcastYellow.Render("[PRECIPITATION EXPECTED]")
	} else {
		badge = styleNowcastGreen.Render("[ALL CLEAR · DRY]")
	}

	fmt.Fprintf(w, "%s %s\n", badge, styleNowcastVal.Render(nc.Headline))
	if nc.Summary != "" {
		fmt.Fprintln(w, styleNowcastSub.Render(nc.Summary))
	}
	fmt.Fprintln(w)

	// ── Key Metrics Summary ─────────────────────────────────────────────
	var liquidStr, snowStr, peakRateStr string
	if isMetric {
		liquidStr = fmt.Sprintf("%.1f mm", nc.TotalLiquidMM)
		snowStr = fmt.Sprintf("%.1f cm", nc.TotalSnowCM)
		peakRateStr = fmt.Sprintf("%.1f mm/h", nc.PeakRateMMH)
	} else {
		liquidStr = fmt.Sprintf("%.2f in", nc.TotalLiquidIn)
		snowStr = fmt.Sprintf("%.1f in", nc.TotalSnowIn)
		peakRateStr = fmt.Sprintf("%.2f in/h", nc.PeakRateInH)
	}

	peakTimeStr := "None"
	if !nc.PeakTime.IsZero() && nc.PeakRateMMH > 0.02 {
		peakTimeStr = nc.PeakTime.Local().Format("3:04 PM")
	}

	phaseStr := string(nc.PrimaryPhase)
	if phaseStr == "" || phaseStr == "none" {
		phaseStr = "None (Dry)"
	} else {
		phaseStr = strings.Title(phaseStr)
	}

	fmt.Fprintf(w, "%s %s    %s %s    %s %s (%s)    %s %s\n",
		styleNowcastLabel.Render("Liquid Accum:"), styleNowcastVal.Render(liquidStr),
		styleNowcastLabel.Render("Snow Accum:"), styleNowcastVal.Render(snowStr),
		styleNowcastLabel.Render("Peak Rate:"), styleNowcastCyan.Render(peakRateStr), styleNowcastSub.Render(peakTimeStr),
		styleNowcastLabel.Render("Phase:"), styleNowcastVal.Render(phaseStr),
	)
	fmt.Fprintln(w)

	// ── Sparkline Timeline ──────────────────────────────────────────────
	if len(nc.Intervals) > 0 {
		var sparkRunes strings.Builder
		for _, iv := range nc.Intervals {
			sparkRunes.WriteRune(nowcast.SparklineRune(iv.RateInH))
		}

		spanHours := float64(len(nc.Intervals)) * 15.0 / 60.0
		fmt.Fprintf(w, "%s [%s] %s\n\n",
			styleNowcastLabel.Render(fmt.Sprintf("Trend (Next %.0fh):", spanHours)),
			styleNowcastCyan.Render(sparkRunes.String()),
			styleNowcastDim.Render(" ▂▃▄▅▆▇█"),
		)

		// ── Interval Breakdown Table (15-min / 30-min window) ───────────
		fmt.Fprintf(w, "%-10s  %-18s  %-12s  %-14s  %-12s  %s\n",
			styleNowcastLabel.Render("TIME"),
			styleNowcastLabel.Render("CONDITION"),
			styleNowcastLabel.Render("PROBABILITY"),
			styleNowcastLabel.Render(ternary(isMetric, "RATE (mm/h)", "RATE (in/h)")),
			styleNowcastLabel.Render(ternary(isMetric, "ACCUM (mm)", "ACCUM (in)")),
			styleNowcastLabel.Render("INTENSITY"),
		)
		fmt.Fprintln(w, styleNowcastDim.Render(strings.Repeat("─", 74)))

		// Display up to 16 intervals (4 hours) or all active intervals
		count := 0
		for _, iv := range nc.Intervals {
			count++
			if count > 16 && iv.RateMMH < 0.05 {
				continue
			}
			if count > 24 {
				break
			}

			tStr := iv.StartTime.Local().Format("03:04 PM")
			condStr := iv.Summary
			if condStr == "" {
				condStr = "Clear"
			}
			probStr := fmt.Sprintf("%.0f%%", iv.Probability)

			var rStr, aStr string
			if isMetric {
				rStr = fmt.Sprintf("%.1f mm/h", iv.RateMMH)
				aStr = fmt.Sprintf("%.2f mm", iv.AccumMM)
			} else {
				rStr = fmt.Sprintf("%.2f in/h", iv.RateInH)
				aStr = fmt.Sprintf("%.2f in", iv.AccumIn)
			}

			barRune := nowcast.SparklineRune(iv.RateInH)
			barStr := string(barRune)
			if iv.RateMMH >= 7.5 {
				barStr = styleNowcastRed.Render(barStr)
			} else if iv.RateMMH >= 2.5 {
				barStr = styleNowcastYellow.Render(barStr)
			} else if iv.RateMMH >= 0.05 {
				barStr = styleNowcastCyan.Render(barStr)
			} else {
				barStr = styleNowcastDim.Render(barStr)
			}

			fmt.Fprintf(w, "%-10s  %-18s  %-12s  %-14s  %-12s  %s\n",
				styleNowcastSub.Render(tStr),
				styleNowcastVal.Render(condStr),
				styleNowcastSub.Render(probStr),
				styleNowcastCyan.Render(rStr),
				styleNowcastSub.Render(aStr),
				barStr,
			)
		}
	}

	return nil
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
