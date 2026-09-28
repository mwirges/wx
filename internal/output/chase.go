package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mwirges/wx/internal/models"
)

// ChaseOptions controls rendering of storm chase clusters.
type ChaseOptions struct {
	ForceJSON   bool
	ForcePretty bool
}

var (
	styleChaseHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleClusterNum   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styleClusterTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	styleClusterMeta  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleHazardTag    = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	styleTopBadge     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")) // Gold
)

// RenderChase renders all active storm clusters.
func RenderChase(payload *models.ChasePayload, opts ChaseOptions) error {
	return RenderChaseTo(os.Stdout, payload, opts)
}

// RenderChaseTo renders all active storm clusters to the given writer.
func RenderChaseTo(w io.Writer, payload *models.ChasePayload, opts ChaseOptions) error {
	if payload == nil {
		return fmt.Errorf("no chase payload data")
	}

	if opts.ForceJSON || (!opts.ForcePretty && !isTTY) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	// ── SPC Convective Hazard Strip ──────────────────────────────────────
	if payload.SPC != nil {
		var spcParts []string
		if payload.SPC.MaxNationalRisk.DN > 0 {
			spcParts = append(spcParts, fmt.Sprintf("CONUS Risk Ceiling: [%s] %s", payload.SPC.MaxNationalRisk.Code, payload.SPC.MaxNationalRisk.Name))
		}
		if len(payload.SPC.ActiveWatches) > 0 {
			spcParts = append(spcParts, fmt.Sprintf("%d Active Watch(es)", len(payload.SPC.ActiveWatches)))
		}
		if len(payload.SPC.ActiveMCDs) > 0 {
			mcdSummaryList := make([]string, 0, len(payload.SPC.ActiveMCDs))
			for _, m := range payload.SPC.ActiveMCDs {
				mcdSummaryList = append(mcdSummaryList, fmt.Sprintf("%s (%s watch prob)", m.Name, m.WatchProbability))
			}
			spcParts = append(spcParts, fmt.Sprintf("Active MCD(s): %s", strings.Join(mcdSummaryList, ", ")))
		}
		if len(spcParts) > 0 {
			spcBanner := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("⚡ SPC CONVECTIVE CONTEXT: ") +
				lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(strings.Join(spcParts, " • "))
			fmt.Fprintln(w, spcBanner)
		}
	}

	if len(payload.Clusters) == 0 {
		fmt.Fprintln(w, styleLocation.Render("No active severe weather clusters detected nationwide."))
		fmt.Fprintln(w, styleTime.Render("Sky is calm across the CONUS. Run 'wx' to check your local forecast or 'wx spc' for convective outlooks."))
		return nil
	}

	header := fmt.Sprintf("Active Severe Weather Systems (%d Alerts Across %d Regional Clusters)", payload.TotalAlerts, payload.TotalClusters)
	fmt.Fprintln(w, styleChaseHeader.Render(header))
	fmt.Fprintln(w, styleLabel.Render(strings.Repeat("─", 78)))
	fmt.Fprintln(w)

	for idx, c := range payload.Clusters {
		numBadge := styleClusterNum.Render(fmt.Sprintf("[%d]", c.ID))
		title := styleClusterTitle.Render(c.Name)
		alertsBadge := lipgloss.NewStyle().Bold(true).Foreground(hazardColor(c.PrimaryHazard)).Render(fmt.Sprintf("(%d Alerts)", c.TotalAlerts))

		topTag := ""
		if idx == 0 && c.Score >= 50 {
			topTag = " " + styleTopBadge.Render("★ HIGHEST IMPACT")
		}

		spcTag := ""
		if c.SPCRisk != "" && c.SPCRisk != "NONE" {
			spcTag = " " + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render(fmt.Sprintf("[%s RISK]", c.SPCRisk))
		}

		fmt.Fprintf(w, "%s %s %s%s%s\n", numBadge, title, alertsBadge, spcTag, topTag)

		statesStr := strings.Join(c.States, ", ")
		if statesStr == "" {
			statesStr = "N/A"
		}
		centerStr := fmt.Sprintf("%.2f, %.2f", c.CenterLat, c.CenterLon)
		radarStr := c.NearestRadar
		if radarStr == "" {
			radarStr = "N/A"
		}
		fmt.Fprintf(w, "    %s\n", styleClusterMeta.Render(fmt.Sprintf("States: %s • Center: %s • Radar: %s", statesStr, centerStr, radarStr)))

		if c.MCDWatch != "" {
			fmt.Fprintf(w, "    %s %s\n", styleLabel.Render("SPC Context:"), lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(c.MCDWatch))
		}

		// Hazard breakdown
		var hazardList []string
		for k, v := range c.HazardsCount {
			hazardList = append(hazardList, fmt.Sprintf("%s (%d)", k, v))
		}
		sort.Strings(hazardList)
		if len(hazardList) > 0 {
			fmt.Fprintf(w, "    %s %s\n", styleLabel.Render("Hazards:"), styleHazardTag.Render(strings.Join(hazardList, " • ")))
		}

		// Top sample cell
		if len(c.Cells) > 0 {
			topCell := c.Cells[0]
			sampleArea := topCell.AreaDesc
			if len(sampleArea) > 55 {
				sampleArea = sampleArea[:52] + "..."
			}
			hazardNote := topCell.HazardText
			if hazardNote == "" {
				hazardNote = topCell.Event
			}
			if len(hazardNote) > 60 {
				hazardNote = hazardNote[:57] + "..."
			}

			expStr := ""
			if !topCell.Expires.IsZero() {
				if time.Until(topCell.Expires) > 0 {
					expStr = fmt.Sprintf(" • Exp: %s", topCell.Expires.Local().Format("3:04 PM MST"))
				}
			}

			fmt.Fprintf(w, "    %s %s — %s%s\n", styleLabel.Render("Top Cell:"), styleValue.Render(sampleArea), styleDesc.Render(hazardNote), styleTime.Render(expStr))
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, styleLabel.Render(strings.Repeat("─", 78)))
	return nil
}

// RenderClusterDetail renders detailed cell-level breakdown for a specific cluster.
func RenderClusterDetail(w io.Writer, c *models.StormCluster, opts ChaseOptions) error {
	if c == nil {
		return fmt.Errorf("cluster is nil")
	}

	if opts.ForceJSON || (!opts.ForcePretty && !isTTY) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(c)
	}

	title := fmt.Sprintf("[%d] %s — %d Active Alerts", c.ID, c.Name, c.TotalAlerts)
	fmt.Fprintln(w, styleChaseHeader.Render(title))
	statesStr := strings.Join(c.States, ", ")
	fmt.Fprintln(w, styleClusterMeta.Render(fmt.Sprintf("States: %s • Center: %.2f, %.2f • Nearest Radar: %s • Severity Score: %d", statesStr, c.CenterLat, c.CenterLon, c.NearestRadar, c.Score)))
	if c.MCDWatch != "" || (c.SPCRisk != "" && c.SPCRisk != "NONE") {
		var spcNotes []string
		if c.SPCRisk != "" && c.SPCRisk != "NONE" {
			spcNotes = append(spcNotes, fmt.Sprintf("Convective Risk: [%s]", c.SPCRisk))
		}
		if c.MCDWatch != "" {
			spcNotes = append(spcNotes, fmt.Sprintf("Active Watch/MCD: %s", c.MCDWatch))
		}
		fmt.Fprintf(w, "%s %s\n", styleLabel.Render("SPC Context:"), lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(strings.Join(spcNotes, " • ")))
	}
	fmt.Fprintln(w, styleLabel.Render(strings.Repeat("─", 78)))
	fmt.Fprintln(w)

	for i, cell := range c.Cells {
		cellNum := styleClusterNum.Render(fmt.Sprintf("[%d.%d]", c.ID, i+1))
		evBadge := lipgloss.NewStyle().Bold(true).Foreground(hazardColor(cell.Event)).Render(cell.Event)
		areaStr := styleClusterTitle.Render(cell.AreaDesc)
		fmt.Fprintf(w, "%s %s — %s\n", cellNum, evBadge, areaStr)

		if cell.SenderName != "" {
			fmt.Fprintf(w, "     %s\n", styleClusterMeta.Render("Office: "+cell.SenderName))
		}
		if cell.HazardText != "" {
			fmt.Fprintf(w, "     %s %s\n", styleLabel.Render("Threat:"), styleValue.Render(cell.HazardText))
		}

		coordsStr := fmt.Sprintf("%.2f, %.2f", cell.Latitude, cell.Longitude)
		expStr := "No expiry"
		if !cell.Expires.IsZero() {
			remain := time.Until(cell.Expires)
			if remain > 0 {
				expStr = fmt.Sprintf("in %s (%s)", formatDurationShort(remain), cell.Expires.Local().Format("3:04 PM MST"))
			} else {
				expStr = "Expired"
			}
		}

		fmt.Fprintf(w, "     %s\n", styleClusterMeta.Render(fmt.Sprintf("Center: %s • Radar: %s • Expires: %s", coordsStr, cell.RadarSite, expStr)))
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, styleLabel.Render(strings.Repeat("─", 78)))
	return nil
}

func hazardColor(event string) lipgloss.Color {
	evLower := strings.ToLower(event)
	switch {
	case strings.Contains(evLower, "tornado"):
		return lipgloss.Color("196") // Red
	case strings.Contains(evLower, "severe thunderstorm"):
		return lipgloss.Color("208") // Orange
	case strings.Contains(evLower, "flash flood"):
		return lipgloss.Color("39") // Blue
	case strings.Contains(evLower, "blizzard") || strings.Contains(evLower, "winter") || strings.Contains(evLower, "ice"):
		return lipgloss.Color("159") // Cyan
	case strings.Contains(evLower, "coastal flood") || strings.Contains(evLower, "gale") || strings.Contains(evLower, "surf"):
		return lipgloss.Color("75") // Light blue
	case strings.Contains(evLower, "high wind") || strings.Contains(evLower, "wind"):
		return lipgloss.Color("220") // Yellow
	default:
		return lipgloss.Color("214") // Amber
	}
}

func formatDurationShort(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	m := int(d.Minutes())
	if m < 60 {
		return fmt.Sprintf("%dm", m)
	}
	h := m / 60
	remM := m % 60
	if remM > 0 {
		return fmt.Sprintf("%dh%dm", h, remM)
	}
	return fmt.Sprintf("%dh", h)
}
