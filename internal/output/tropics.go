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

// TropicsOptions controls rendering of NHC tropics telemetry.
type TropicsOptions struct {
	ForceJSON   bool
	ForcePretty bool
	Units       string // "imperial" or "metric"
}

var (
	styleTropicsHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleTropicsSub    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleTropicsLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleTropicsVal    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleTropicsCyan   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styleTropicsGreen  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	styleTropicsGold   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	styleTropicsAmber  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	styleTropicsRed    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	styleTropicsPurple = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("201"))
	styleTropicsDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// RenderTropics renders tropical cyclone telemetry to os.Stdout.
func RenderTropics(payload *models.TropicsPayload, opts TropicsOptions) error {
	return RenderTropicsTo(os.Stdout, payload, opts)
}

// RenderTropicsTo renders tropical cyclone telemetry to the given writer.
func RenderTropicsTo(w io.Writer, payload *models.TropicsPayload, opts TropicsOptions) error {
	if payload == nil || payload.Tropics == nil {
		return fmt.Errorf("tropics: empty payload")
	}

	if opts.ForceJSON || (!opts.ForcePretty && !isTTY) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	imperial := opts.Units != "metric"
	report := payload.Tropics

	// 1. Header
	fmt.Fprintln(w)
	title := "🌪️  NOAA NATIONAL HURRICANE CENTER // TROPICAL CYCLONE TELEMETRY"
	fmt.Fprintln(w, styleTropicsHeader.Render(title))

	var subText string
	if report.TotalActive > 0 {
		subText = fmt.Sprintf("%d ACTIVE TROPICAL CYCLONE%s DETECTED", report.TotalActive, pluralS(report.TotalActive))
		if report.ReferenceLocation != "" {
			subText += fmt.Sprintf(" · PROXIMITY REF: %s", strings.ToUpper(report.ReferenceLocation))
		}
		fmt.Fprintln(w, styleTropicsAmber.Render(subText))
	} else {
		subText = "ALL BASINS QUIET // NO ACTIVE TROPICAL CYCLONES"
		if report.ReferenceLocation != "" {
			subText += fmt.Sprintf(" · REF: %s", strings.ToUpper(report.ReferenceLocation))
		}
		fmt.Fprintln(w, styleTropicsGreen.Render(subText))
	}
	fmt.Fprintln(w, styleTropicsDim.Render(strings.Repeat("─", 68)))

	// 2. Active Storms
	if len(report.Storms) > 0 {
		for i, storm := range report.Storms {
			badgeStyle := categoryBadgeStyle(storm.Category, storm.Classification)
			badge := badgeStyle.Render(fmt.Sprintf("[%s]", strings.ToUpper(storm.CategoryLabel)))

			stormTitle := fmt.Sprintf("%s %s", storm.ClassificationName, storm.Name)
			if storm.AdvisoryNumber != "" {
				stormTitle += fmt.Sprintf(" #%s", storm.AdvisoryNumber)
			}
			stormTitle += fmt.Sprintf(" (%s)", strings.ToUpper(storm.ID))

			fmt.Fprintf(w, "\n%s  %s\n", badge, styleTropicsHeader.Render(strings.ToUpper(stormTitle)))

			// Telemetry Line
			var windStr string
			if imperial {
				windStr = fmt.Sprintf("%d mph (%d kt)", storm.WindSpeedMph, storm.IntensityKt)
			} else {
				windStr = fmt.Sprintf("%d km/h (%d kt)", storm.WindSpeedKmh, storm.IntensityKt)
			}

			var pressStr string
			if storm.PressureMb > 0 {
				if imperial {
					pressStr = fmt.Sprintf("%d mb (%.2f inHg)", storm.PressureMb, storm.PressureInHg)
				} else {
					pressStr = fmt.Sprintf("%d hPa", storm.PressureMb)
				}
			} else {
				pressStr = "N/A"
			}

			var mvmtStr string
			if storm.MovementSpeedMph > 0 {
				if imperial {
					mvmtStr = fmt.Sprintf("%s (%d°) at %d mph", storm.MovementCompass, storm.MovementDir, storm.MovementSpeedMph)
				} else {
					mvmtStr = fmt.Sprintf("%s (%d°) at %d km/h", storm.MovementCompass, storm.MovementDir, storm.MovementSpeedKmh)
				}
			} else {
				mvmtStr = "Stationary"
			}

			fmt.Fprintf(w, "  %s %s   %s %s   %s %s\n",
				styleTropicsLabel.Render("Winds:"), styleTropicsVal.Render(windStr),
				styleTropicsLabel.Render("Pressure:"), styleTropicsVal.Render(pressStr),
				styleTropicsLabel.Render("Movement:"), styleTropicsVal.Render(mvmtStr),
			)

			// Location & Proximity
			locLine := fmt.Sprintf("  %s %s", styleTropicsLabel.Render("Position:"), styleTropicsVal.Render(storm.LocationText))
			if storm.ProximityText != "" {
				locLine += fmt.Sprintf("  %s", styleTropicsSub.Render("· "+storm.ProximityText))
			}
			fmt.Fprintln(w, locLine)

			// Distance from user location if available
			if storm.DistanceMiles != nil && storm.DistanceKm != nil {
				var distStr string
				if imperial {
					distStr = fmt.Sprintf("%.0f miles away", *storm.DistanceMiles)
				} else {
					distStr = fmt.Sprintf("%.0f km away", *storm.DistanceKm)
				}
				fmt.Fprintf(w, "  %s %s\n", styleTropicsLabel.Render("Distance:"), styleTropicsCyan.Render(distStr))
			}

			// Headline if present
			if storm.Headline != "" {
				fmt.Fprintf(w, "  %s %s\n", styleTropicsAmber.Render("⚡"), styleTropicsSub.Render(storm.Headline))
			}

			// Watches & Warnings if present
			if len(storm.WatchesWarnings) > 0 {
				fmt.Fprintln(w, styleTropicsRed.Render("  ⚠️  Active Watches & Warnings:"))
				for _, ww := range storm.WatchesWarnings {
					fmt.Fprintf(w, "     • %s\n", styleTropicsSub.Render(ww))
				}
			}

			// Resource Links
			if storm.GraphicsURL != "" {
				fmt.Fprintf(w, "  %s %s\n", styleTropicsLabel.Render("Track / Cone:"), styleTropicsCyan.Render(storm.GraphicsURL))
			}
			if storm.PublicAdvisoryURL != "" {
				fmt.Fprintf(w, "  %s %s\n", styleTropicsLabel.Render("Public Advisory:"), styleTropicsSub.Render(storm.PublicAdvisoryURL))
			}

			if i < len(report.Storms)-1 {
				fmt.Fprintln(w, styleTropicsDim.Render("  ┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈"))
			}
		}
	}

	// 3. Disturbances & Invests (Tropical Weather Outlook)
	if len(report.Disturbances) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, styleTropicsHeader.Render("TROPICAL WEATHER OUTLOOK // INVEST DISTURBANCES"))
		fmt.Fprintln(w, styleTropicsDim.Render(strings.Repeat("─", 68)))

		for _, d := range report.Disturbances {
			ch48Style := chanceColor(d.Chance48h)
			ch7dStyle := chanceColor(d.Chance7d)

			header := fmt.Sprintf("[%s] %s (%s Basin)", d.ID, d.Name, d.Basin)
			fmt.Fprintln(w, styleTropicsAmber.Render(header))

			fmt.Fprintf(w, "  48-Hour Formation Chance: %s   7-Day Formation Chance: %s\n",
				ch48Style.Render(fmt.Sprintf("%d%% [%s]", d.Chance48h, strings.ToUpper(d.Category48h))),
				ch7dStyle.Render(fmt.Sprintf("%d%% [%s]", d.Chance7d, strings.ToUpper(d.Category7d))),
			)
			if d.Summary != "" {
				fmt.Fprintf(w, "  %s\n", styleTropicsSub.Render(d.Summary))
			}
		}
	}

	// 4. Basin Maps Footer
	fmt.Fprintln(w)
	fmt.Fprintln(w, styleTropicsLabel.Render("NOAA NHC 7-Day Graphical Tropical Weather Outlook Maps:"))
	if report.AtlanticOutlookURL != "" {
		fmt.Fprintf(w, "  %s %s\n", styleTropicsLabel.Render("Atlantic:"), styleTropicsCyan.Render(report.AtlanticOutlookURL))
	}
	if report.PacificOutlookURL != "" {
		fmt.Fprintf(w, "  %s %s\n", styleTropicsLabel.Render("Eastern Pacific:"), styleTropicsCyan.Render(report.PacificOutlookURL))
	}
	fmt.Fprintln(w)

	return nil
}

func categoryBadgeStyle(cat int, class string) lipgloss.Style {
	c := strings.ToUpper(class)
	switch {
	case cat == 5:
		return styleTropicsPurple
	case cat == 4:
		return styleTropicsRed
	case cat == 3:
		return styleTropicsAmber
	case cat >= 1:
		return styleTropicsGold
	case c == "TS":
		return styleTropicsCyan
	case c == "TD":
		return styleTropicsGreen
	default:
		return styleTropicsSub
	}
}

func chanceColor(chance int) lipgloss.Style {
	switch {
	case chance >= 60:
		return styleTropicsRed
	case chance >= 40:
		return styleTropicsAmber
	case chance >= 20:
		return styleTropicsGold
	default:
		return styleTropicsGreen
	}
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "S"
}
