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

// SPCOptions controls rendering of SPC convective outlooks.
type SPCOptions struct {
	ForceJSON   bool
	ForcePretty bool
}

var (
	styleSPCHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleSPCHorizon = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styleSPCDates   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleSPCMCDBox  = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")). // Amber
			Padding(0, 1)
	styleSPCWatchBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("196")). // Red
				Padding(0, 1)
)

// RenderSPC renders SPC outlooks to os.Stdout.
func RenderSPC(payload *models.SPCPayload, opts SPCOptions) error {
	return RenderSPCTo(os.Stdout, payload, opts)
}

// RenderSPCTo renders SPC outlooks to the given writer.
func RenderSPCTo(w io.Writer, payload *models.SPCPayload, opts SPCOptions) error {
	if payload == nil {
		return fmt.Errorf("no SPC outlook data")
	}

	if opts.ForceJSON || (!opts.ForcePretty && !isTTY) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	// ── Header ──────────────────────────────────────────────────────────
	fmt.Fprintln(w, styleSPCHeader.Render("NOAA Storm Prediction Center (SPC) Convective Outlook"))
	sub := fmt.Sprintf("%s (%.2f°N, %.2f°W)", payload.Location, payload.Coordinates[0], payload.Coordinates[1])
	fmt.Fprintln(w, styleLocation.Render(sub))

	if payload.MaxNationalRisk.DN > 0 {
		maxRiskBadge := spcRiskBadge(payload.MaxNationalRisk)
		fmt.Fprintf(w, "%s %s\n", styleLabel.Render("CONUS Risk Ceiling:"), maxRiskBadge)
	}
	fmt.Fprintln(w, styleLabel.Render(strings.Repeat("─", 78)))
	fmt.Fprintln(w)

	// ── Active Watches (High-Priority Alert Banner) ─────────────────────
	if len(payload.ActiveWatches) > 0 {
		var watchLines []string
		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render("🚨 ACTIVE SEVERE WEATHER WATCHES")
		watchLines = append(watchLines, title)

		for _, wItem := range payload.ActiveWatches {
			expStr := "active"
			if !wItem.Expires.IsZero() {
				if time.Until(wItem.Expires) > 0 {
					expStr = fmt.Sprintf("expires in %s (%s)", formatDurationShort(time.Until(wItem.Expires)), wItem.Expires.Local().Format("3:04 PM MST"))
				} else {
					expStr = "expired"
				}
			}

			statesStr := strings.Join(wItem.States, ", ")
			if statesStr == "" {
				statesStr = "CONUS"
			}

			watchHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).
				Render(fmt.Sprintf("• Watch #%d — %s (%s)", wItem.WatchNumber, wItem.Type, statesStr))
			watchLines = append(watchLines, watchHeader)
			watchLines = append(watchLines, styleDesc.Render(fmt.Sprintf("  Areas: %s • %s", wItem.AreaDesc, expStr)))
			if wItem.URL != "" {
				watchLines = append(watchLines, styleTime.Render(fmt.Sprintf("  Details: %s", wItem.URL)))
			}
		}

		box := styleSPCWatchBox.Render(strings.Join(watchLines, "\n"))
		fmt.Fprintln(w, box)
		fmt.Fprintln(w)
	}

	// ── Active Mesoscale Discussions (MCD) ──────────────────────────────
	if len(payload.ActiveMCDs) > 0 {
		var mcdLines []string
		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render("⚡ ACTIVE MESOSCALE DISCUSSIONS (MCD)")
		mcdLines = append(mcdLines, title)

		for _, mcd := range payload.ActiveMCDs {
			probBadge := ""
			if mcd.WatchProbability != "" {
				probBadge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).
					Render(fmt.Sprintf("[Watch Probability: %s]", mcd.WatchProbability))
			}

			mcdHeader := fmt.Sprintf("• %s — %s %s", mcd.Name, mcd.Concerning, probBadge)
			mcdLines = append(mcdLines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Render(mcdHeader))
			mcdLines = append(mcdLines, styleDesc.Render(fmt.Sprintf("  Areas: %s", mcd.AreasAffected)))
			if mcd.Summary != "" {
				mcdLines = append(mcdLines, styleValue.Render(fmt.Sprintf("  Summary: %s", mcd.Summary)))
			}
			if mcd.URL != "" {
				mcdLines = append(mcdLines, styleTime.Render(fmt.Sprintf("  Discussion: %s", mcd.URL)))
			}
		}

		box := styleSPCMCDBox.Render(strings.Join(mcdLines, "\n"))
		fmt.Fprintln(w, box)
		fmt.Fprintln(w)
	}

	// ── Convective Outlooks (Day 1 / Day 2 / Day 3) ─────────────────────
	colH := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("250"))
	fmt.Fprintf(w, "%-8s %-26s %-14s %-14s %-14s\n",
		colH.Render("OUTLOOK"),
		colH.Render("CATEGORICAL RISK"),
		colH.Render("TORNADO"),
		colH.Render("HAIL"),
		colH.Render("WIND"),
	)
	fmt.Fprintln(w, styleLabel.Render(strings.Repeat("─", 78)))

	renderDayRow(w, "Day 1", payload.Day1)
	renderDayRow(w, "Day 2", payload.Day2)
	renderDayRow(w, "Day 3", payload.Day3)

	fmt.Fprintln(w)

	// ── Narrative Synthesis ─────────────────────────────────────────────
	if payload.ConvectiveSummary != "" {
		fmt.Fprintf(w, "%s %s\n", styleLabel.Render("Outlook Summary:"), styleValue.Render(payload.ConvectiveSummary))
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, styleLabel.Render(strings.Repeat("─", 78)))
	return nil
}

func renderDayRow(w io.Writer, dayLabel string, o models.SPCOutlookItem) {
	label := styleSPCHorizon.Render(dayLabel)
	riskBadge := spcRiskBadge(o.Category)

	tornStr := o.TornadoProb
	if tornStr == "" || tornStr == "None" {
		tornStr = "None (<2%)"
	}
	if o.TornadoSig {
		tornStr += " [SIG]"
	}

	hailStr := o.HailProb
	if hailStr == "" || hailStr == "None" {
		hailStr = "None (<5%)"
	}
	if o.HailSig {
		hailStr += " [SIG]"
	}

	windStr := o.WindProb
	if windStr == "" || windStr == "None" {
		windStr = "None (<5%)"
	}
	if o.WindSig {
		windStr += " [SIG]"
	}

	if o.Day == 3 && o.SevereProb != "" && o.SevereProb != "None" {
		sevStr := fmt.Sprintf("Severe: %s", o.SevereProb)
		if o.SevereSig {
			sevStr += " [SIG]"
		}
		fmt.Fprintf(w, "%-8s %-26s %-42s\n", label, riskBadge, styleValue.Render(sevStr))
		return
	}

	fmt.Fprintf(w, "%-8s %-26s %-14s %-14s %-14s\n",
		label,
		riskBadge,
		styleProb(tornStr, o.TornadoSig),
		styleProb(hailStr, o.HailSig),
		styleProb(windStr, o.WindSig),
	)
}

func spcRiskBadge(cat models.SPCRiskCategory) string {
	fg := "255"
	bg := "240"
	switch cat.Code {
	case "HIGH":
		bg = "198" // Magenta
		fg = "255"
	case "MDT":
		bg = "196" // Red
		fg = "255"
	case "ENH":
		bg = "208" // Orange
		fg = "232"
	case "SLGT":
		bg = "220" // Yellow
		fg = "232"
	case "MRGL":
		bg = "34" // Dark Green
		fg = "255"
	case "TSTM":
		bg = "77" // Light Green
		fg = "232"
	default:
		bg = "238" // Neutral Dark
		fg = "250"
	}

	badge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg)).
		Padding(0, 1).
		Render(cat.Code)

	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	return fmt.Sprintf("%s %s", badge, nameStyle.Render(cat.Name))
}

func styleProb(val string, sig bool) string {
	if strings.HasPrefix(val, "None") {
		return styleTime.Render(val)
	}
	if sig {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render(val)
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render(val)
}
