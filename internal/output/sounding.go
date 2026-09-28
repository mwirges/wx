package output

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mwirges/wx/internal/models"
)

// SoundingOptions controls rendering of atmospheric sounding reports.
type SoundingOptions struct {
	ForceJSON   bool
	ForcePretty bool
	Units       string // "imperial" or "metric"
}

var (
	styleSoundingHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleSoundingSub    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleSoundingLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleSoundingVal    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleSoundingCyan   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styleSoundingGreen  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	styleSoundingYellow = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	styleSoundingOrange = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	styleSoundingRed    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	styleSoundingDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// RenderSounding renders atmospheric sounding and convective indices to os.Stdout.
func RenderSounding(payload *models.SoundingPayload, opts SoundingOptions) error {
	return RenderSoundingTo(os.Stdout, payload, opts)
}

// RenderSoundingTo renders atmospheric sounding and convective indices to the given writer.
func RenderSoundingTo(w io.Writer, payload *models.SoundingPayload, opts SoundingOptions) error {
	if payload == nil || payload.Sounding == nil {
		return fmt.Errorf("sounding: empty payload")
	}

	if opts.ForceJSON || (!opts.ForcePretty && !isTTY) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	imperial := opts.Units != "metric"
	snd := payload.Sounding
	idx := snd.Indices

	// 1. Header Bar
	fmt.Fprintln(w)
	title := fmt.Sprintf("ATMOSPHERIC SOUNDING & CONVECTIVE INSTABILITY · %s", strings.ToUpper(snd.StationID))
	fmt.Fprintln(w, styleSoundingHeader.Render(title))

	var subMeta []string
	if snd.StationName != "" {
		subMeta = append(subMeta, snd.StationName)
	}
	if snd.DistanceKM > 0 {
		if imperial {
			subMeta = append(subMeta, fmt.Sprintf("%.0f mi away", snd.DistanceMiles))
		} else {
			subMeta = append(subMeta, fmt.Sprintf("%.0f km away", snd.DistanceKM))
		}
	}
	subMeta = append(subMeta, snd.Timestamp.Format("2006-01-02 15:04 MST"))
	subMeta = append(subMeta, snd.Provider)
	fmt.Fprintln(w, styleSoundingSub.Render(strings.Join(subMeta, " · ")))
	fmt.Fprintln(w)

	// 2. Convective Environment Assessment Badge
	var riskBadge lipgloss.Style
	capeVal := 0.0
	if idx.SBCAPE != nil {
		capeVal = *idx.SBCAPE
	}
	shearVal := 0.0
	if idx.BulkShear06KT != nil {
		shearVal = *idx.BulkShear06KT
	}

	if capeVal >= 2000 && shearVal >= 35 {
		riskBadge = styleSoundingRed
	} else if capeVal >= 1000 || shearVal >= 25 {
		riskBadge = styleSoundingOrange
	} else if capeVal >= 300 {
		riskBadge = styleSoundingYellow
	} else {
		riskBadge = styleSoundingGreen
	}

	fmt.Fprintf(w, "  %s  %s\n",
		riskBadge.Render(fmt.Sprintf("[%s]", strings.ToUpper(idx.InstabilitySummary))),
		styleSoundingVal.Render(idx.ConvectiveRisk),
	)
	if idx.ShearSummary != "" {
		fmt.Fprintf(w, "  %s  %s\n",
			styleSoundingCyan.Render("Kinematic Regime:"),
			styleSoundingSub.Render(idx.ShearSummary),
		)
	}
	fmt.Fprintln(w)

	// 3. Convective Telemetry & Kinematics KPI Grid
	fmt.Fprintln(w, styleSoundingHeader.Render("CONVECTIVE INDICES & STABILITY TELEMETRY"))
	fmt.Fprintln(w, styleSoundingDim.Render(strings.Repeat("─", 74)))

	renderKPI := func(label, val string, alertStyle lipgloss.Style) string {
		return fmt.Sprintf("%s %s", styleSoundingLabel.Render(fmt.Sprintf("%-16s", label)), alertStyle.Render(val))
	}

	// Format values
	formatCAPE := func(v *float64) string {
		if v == nil {
			return "--"
		}
		return fmt.Sprintf("%.0f J/kg", *v)
	}
	formatCIN := func(v *float64) string {
		if v == nil {
			return "--"
		}
		return fmt.Sprintf("%.0f J/kg", *v)
	}
	formatLI := func(v *float64) string {
		if v == nil {
			return "--"
		}
		return fmt.Sprintf("%+.1f °C", *v)
	}
	formatShear := func(v *float64) string {
		if v == nil {
			return "--"
		}
		return fmt.Sprintf("%.0f kt", *v)
	}
	formatSRH := func(v *float64) string {
		if v == nil {
			return "--"
		}
		return fmt.Sprintf("%.0f m²/s²", *v)
	}

	// Rows: 4 paired columns
	row1Left := renderKPI("Surface CAPE:", formatCAPE(idx.SBCAPE), styleSoundingVal)
	row1Right := renderKPI("0-1km Bulk Shear:", formatShear(idx.BulkShear01KT), styleSoundingVal)
	fmt.Fprintf(w, "  %-36s   %s\n", row1Left, row1Right)

	row2Left := renderKPI("Mixed-Layer CAPE:", formatCAPE(idx.MLCAPE), styleSoundingVal)
	row2Right := renderKPI("0-3km Bulk Shear:", formatShear(idx.BulkShear03KT), styleSoundingVal)
	fmt.Fprintf(w, "  %-36s   %s\n", row2Left, row2Right)

	row3Left := renderKPI("Most-Unstable CAPE:", formatCAPE(idx.MUCAPE), styleSoundingVal)
	row3Right := renderKPI("0-6km Deep Shear:", formatShear(idx.BulkShear06KT), styleSoundingCyan)
	fmt.Fprintf(w, "  %-36s   %s\n", row3Left, row3Right)

	row4Left := renderKPI("Surface CIN:", formatCIN(idx.SBCIN), styleSoundingSub)
	row4Right := renderKPI("0-1km SRH:", formatSRH(idx.SRH01), styleSoundingVal)
	fmt.Fprintf(w, "  %-36s   %s\n", row4Left, row4Right)

	row5Left := renderKPI("Surface Lifted Idx:", formatLI(idx.SBLI), styleSoundingVal)
	row5Right := renderKPI("0-3km SRH:", formatSRH(idx.SRH03), styleSoundingVal)
	fmt.Fprintf(w, "  %-36s   %s\n", row5Left, row5Right)

	// Additional thermodynamic parameters
	pwatStr := "--"
	if imperial && idx.PWATIn != nil {
		pwatStr = fmt.Sprintf("%.2f in", *idx.PWATIn)
	} else if !imperial && idx.PWATMm != nil {
		pwatStr = fmt.Sprintf("%.1f mm", *idx.PWATMm)
	}
	row6Left := renderKPI("Precip Water (PW):", pwatStr, styleSoundingGreen)

	stpStr := "--"
	if idx.STP != nil {
		stpStr = fmt.Sprintf("%.1f", *idx.STP)
	}
	row6Right := renderKPI("Sig Tornado (STP):", stpStr, styleSoundingYellow)
	fmt.Fprintf(w, "  %-36s   %s\n", row6Left, row6Right)

	if idx.DCAPE != nil || idx.SCP != nil {
		dcapeStr := "--"
		if idx.DCAPE != nil {
			dcapeStr = fmt.Sprintf("%.0f J/kg", *idx.DCAPE)
		}
		scpStr := "--"
		if idx.SCP != nil {
			scpStr = fmt.Sprintf("%.1f", *idx.SCP)
		}
		row7Left := renderKPI("Downdraft CAPE:", dcapeStr, styleSoundingSub)
		row7Right := renderKPI("Supercell (SCP):", scpStr, styleSoundingSub)
		fmt.Fprintf(w, "  %-36s   %s\n", row7Left, row7Right)
	}

	if idx.LapseRate700_500 != nil || idx.FreezingLevelFT != nil {
		lapseStr := "--"
		if idx.LapseRate700_500 != nil {
			lapseStr = fmt.Sprintf("%.1f °C/km", *idx.LapseRate700_500)
		}
		fzStr := "--"
		if imperial && idx.FreezingLevelFT != nil {
			fzStr = fmt.Sprintf("%.0f ft", *idx.FreezingLevelFT)
		} else if !imperial && idx.FreezingLevelM != nil {
			fzStr = fmt.Sprintf("%.0f m", *idx.FreezingLevelM)
		}
		row8Left := renderKPI("700-500mb Lapse:", lapseStr, styleSoundingSub)
		row8Right := renderKPI("Freezing Level:", fzStr, styleSoundingSub)
		fmt.Fprintf(w, "  %-36s   %s\n", row8Left, row8Right)
	}

	fmt.Fprintln(w)

	// 4. Vertical Atmospheric Profile Table
	if len(snd.Levels) > 0 {
		fmt.Fprintln(w, styleSoundingHeader.Render("ATMOSPHERIC VERTICAL PROFILE (MANDATORY LEVELS)"))
		fmt.Fprintln(w, styleSoundingDim.Render(strings.Repeat("─", 74)))

		hAlt := "ALT (FT)"
		hTemp := "TEMP (°F)"
		hDew := "DEW (°F)"
		hWind := "WIND (KT)"
		if !imperial {
			hAlt = "ALT (M)"
			hTemp = "TEMP (°C)"
			hDew = "DEW (°C)"
			hWind = "WIND (KM/H)"
		}

		fmt.Fprintf(w, "  %-10s %-12s %-12s %-12s %-14s %s\n",
			styleSoundingLabel.Render("LEVEL"),
			styleSoundingLabel.Render(hAlt),
			styleSoundingLabel.Render(hTemp),
			styleSoundingLabel.Render(hDew),
			styleSoundingLabel.Render(hWind),
			styleSoundingLabel.Render("RELATIVE HUMIDITY"),
		)
		fmt.Fprintln(w, styleSoundingDim.Render(strings.Repeat("─", 74)))

		for _, lvl := range snd.Levels {
			lvlStr := fmt.Sprintf("%.0f hPa", lvl.PressureHPA)

			altStr := "--"
			if imperial && lvl.HeightFT != nil {
				altStr = fmt.Sprintf("%.0f ft", *lvl.HeightFT)
			} else if !imperial && lvl.HeightM != nil {
				altStr = fmt.Sprintf("%.0f m", *lvl.HeightM)
			}

			tStr := "--"
			var tStyle lipgloss.Style = styleSoundingVal
			if imperial && lvl.TempF != nil {
				tStr = fmt.Sprintf("%+.1f°F", *lvl.TempF)
				if *lvl.TempF <= 32.0 {
					tStyle = styleSoundingCyan
				}
			} else if !imperial && lvl.TempC != nil {
				tStr = fmt.Sprintf("%+.1f°C", *lvl.TempC)
				if *lvl.TempC <= 0.0 {
					tStyle = styleSoundingCyan
				}
			}

			tdStr := "--"
			if imperial && lvl.DewPointF != nil {
				tdStr = fmt.Sprintf("%+.1f°F", *lvl.DewPointF)
			} else if !imperial && lvl.DewPointC != nil {
				tdStr = fmt.Sprintf("%+.1f°C", *lvl.DewPointC)
			}

			wStr := "--"
			if lvl.WindDirDeg != nil {
				dirArrow := windDirToArrow(*lvl.WindDirDeg)
				if imperial && lvl.WindSpeedKT != nil {
					wStr = fmt.Sprintf("%s %03.0f° %.0f kt", dirArrow, *lvl.WindDirDeg, *lvl.WindSpeedKT)
				} else if !imperial && lvl.WindSpeedKPH != nil {
					wStr = fmt.Sprintf("%s %03.0f° %.0f km/h", dirArrow, *lvl.WindDirDeg, *lvl.WindSpeedKPH)
				}
			}

			rhBar := "--"
			if lvl.RHPct != nil {
				rh := *lvl.RHPct
				barLen := int(math.Round(rh / 10.0))
				if barLen > 10 {
					barLen = 10
				}
				filled := strings.Repeat("█", barLen)
				empty := strings.Repeat("░", 10-barLen)

				var rhStyle lipgloss.Style
				if rh >= 80 {
					rhStyle = styleSoundingGreen
				} else if rh >= 50 {
					rhStyle = styleSoundingCyan
				} else {
					rhStyle = styleSoundingDim
				}
				rhBar = fmt.Sprintf("%s%s %3.0f%%", rhStyle.Render(filled), styleSoundingDim.Render(empty), rh)
			}

			fmt.Fprintf(w, "  %-10s %-12s %-12s %-12s %-14s %s\n",
				styleSoundingCyan.Render(lvlStr),
				styleSoundingSub.Render(altStr),
				tStyle.Render(tStr),
				styleSoundingSub.Render(tdStr),
				styleSoundingVal.Render(wStr),
				rhBar,
			)
		}
		fmt.Fprintln(w)
	}

	// 5. Skew-T diagram reference if available
	if snd.SkewTImageURL != "" {
		fmt.Fprintf(w, "  %s %s\n",
			styleSoundingLabel.Render("Skew-T Diagram:"),
			styleSoundingCyan.Render(snd.SkewTImageURL),
		)
		fmt.Fprintln(w)
	}

	return nil
}

func windDirToArrow(deg float64) string {
	arrows := []string{"↓", "↙", "←", "↖", "↑", "↗", "→", "↘"}
	idx := int(math.Floor((deg+22.5)/45.0)) % 8
	if idx < 0 {
		idx += 8
	}
	return arrows[idx]
}
