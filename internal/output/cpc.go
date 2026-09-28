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

// CPCOptions controls rendering of CPC outlooks.
type CPCOptions struct {
	ForceJSON   bool
	ForcePretty bool
}

var (
	styleCPCHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleCPCHorizon = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styleCPCDates   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleShiftBox   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")). // Warm amber/orange
			Padding(0, 1)
	styleShiftTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
)

// RenderCPC renders CPC outlooks to os.Stdout.
func RenderCPC(payload *models.CPCPayload, opts CPCOptions) error {
	return RenderCPCTo(os.Stdout, payload, opts)
}

// RenderCPCTo renders CPC outlooks to the given writer.
func RenderCPCTo(w io.Writer, payload *models.CPCPayload, opts CPCOptions) error {
	if payload == nil {
		return fmt.Errorf("no CPC outlook data")
	}

	if opts.ForceJSON || (!opts.ForcePretty && !isTTY) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	// ── Header ──────────────────────────────────────────────────────────
	fmt.Fprintln(w, styleCPCHeader.Render("NOAA Climate Prediction Center (CPC) Long-Range Outlook"))
	sub := fmt.Sprintf("%s (%.2f°N, %.2f°W)", payload.Location, payload.Coordinates[0], payload.Coordinates[1])
	fmt.Fprintln(w, styleLocation.Render(sub))

	if payload.Drought != nil && payload.Drought.Status != "" {
		droughtStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		if payload.Drought.Status != "No Drought" && payload.Drought.Status != "None" {
			droughtStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
		}
		targetStr := ""
		if payload.Drought.Target != "" {
			targetStr = fmt.Sprintf(" (%s)", payload.Drought.Target)
		}
		fmt.Fprintf(w, "%s %s\n", styleLabel.Render("Drought Assessment:"), droughtStyle.Render(payload.Drought.Status+targetStr))
	}
	fmt.Fprintln(w, styleLabel.Render(strings.Repeat("─", 78)))
	fmt.Fprintln(w)

	// ── Outlooks Table ──────────────────────────────────────────────────
	// Table Headers
	colH := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("250"))
	fmt.Fprintf(w, "%-12s %-20s %-22s %-22s\n",
		colH.Render("HORIZON"),
		colH.Render("VALID DATES"),
		colH.Render("TEMPERATURE"),
		colH.Render("PRECIPITATION"),
	)
	fmt.Fprintln(w, styleLabel.Render(strings.Repeat("─", 78)))

	for _, o := range payload.Outlooks {
		horizon := styleCPCHorizon.Render(o.Horizon)
		dates := "N/A"
		if !o.StartDate.IsZero() && !o.EndDate.IsZero() {
			dates = fmt.Sprintf("%s – %s", o.StartDate.Format("Jan 2"), o.EndDate.Format("Jan 2"))
		}
		datesStr := styleCPCDates.Render(dates)

		tempBadge := cpcCategoryBadge(o.TempCategory, o.TempProbability, true)
		precipBadge := cpcCategoryBadge(o.PrecipCategory, o.PrecipProbability, false)

		fmt.Fprintf(w, "%-12s %-20s %-22s %-22s\n", horizon, datesStr, tempBadge, precipBadge)
	}
	fmt.Fprintln(w)

	// ── Pattern Shift Box ───────────────────────────────────────────────
	if payload.PatternShift.Summary != "" {
		confBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).
			Render(fmt.Sprintf("[%s Confidence]", payload.PatternShift.Confidence))

		title := styleShiftTitle.Render("⚡ Pattern Shift Heads-Up") + " " + confBadge
		body := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render(payload.PatternShift.Summary)

		content := fmt.Sprintf("%s\n%s", title, body)
		fmt.Fprintln(w, styleShiftBox.Render(content))
		fmt.Fprintln(w)
	}

	return nil
}

func cpcCategoryBadge(cat string, prob float64, isTemp bool) string {
	var (
		color lipgloss.Color
		label string
	)

	switch strings.ToLower(cat) {
	case "above":
		if isTemp {
			color = lipgloss.Color("208") // Warm Orange
		} else {
			color = lipgloss.Color("35") // Green/Teal (Wet)
		}
		label = fmt.Sprintf("Above Normal (%.0f%%)", prob)
	case "below":
		if isTemp {
			color = lipgloss.Color("39") // Cool Cyan/Blue
		} else {
			color = lipgloss.Color("178") // Brown/Gold (Dry)
		}
		label = fmt.Sprintf("Below Normal (%.0f%%)", prob)
	case "normal":
		color = lipgloss.Color("42") // Mild green
		label = fmt.Sprintf("Near Normal (%.0f%%)", prob)
	default:
		color = lipgloss.Color("244") // Gray
		label = "Equal Chances (33%)"
	}

	return lipgloss.NewStyle().Bold(true).Foreground(color).Render(label)
}
