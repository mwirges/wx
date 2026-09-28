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

// HistoryOptions controls rendering of historical weather.
type HistoryOptions struct {
	ForceJSON bool
	Units     string // "imperial" or "metric"
}

type jsonHistoryDay struct {
	Date          string   `json:"date"`
	ConditionCode string   `json:"condition_code,omitempty"`
	Description   string   `json:"description,omitempty"`
	TempMaxC      *float64 `json:"temperature_max_c,omitempty"`
	TempMaxF      *float64 `json:"temperature_max_f,omitempty"`
	TempMinC      *float64 `json:"temperature_min_c,omitempty"`
	TempMinF      *float64 `json:"temperature_min_f,omitempty"`
	ApparentMaxC  *float64 `json:"apparent_max_c,omitempty"`
	ApparentMaxF  *float64 `json:"apparent_max_f,omitempty"`
	ApparentMinC  *float64 `json:"apparent_min_c,omitempty"`
	ApparentMinF  *float64 `json:"apparent_min_f,omitempty"`
	PrecipSumMM   *float64 `json:"precipitation_mm,omitempty"`
	PrecipSumIn   *float64 `json:"precipitation_in,omitempty"`
	WindMaxKPH    *float64 `json:"wind_max_kph,omitempty"`
	WindMaxMPH    *float64 `json:"wind_max_mph,omitempty"`
}

type jsonHistorySummary struct {
	DaysCount     int      `json:"days_count"`
	AvgTempMaxC   *float64 `json:"avg_temperature_max_c,omitempty"`
	AvgTempMaxF   *float64 `json:"avg_temperature_max_f,omitempty"`
	AvgTempMinC   *float64 `json:"avg_temperature_min_c,omitempty"`
	AvgTempMinF   *float64 `json:"avg_temperature_min_f,omitempty"`
	TotalPrecipMM *float64 `json:"total_precipitation_mm,omitempty"`
	TotalPrecipIn *float64 `json:"total_precipitation_in,omitempty"`
	MaxWindKPH    *float64 `json:"max_wind_kph,omitempty"`
	MaxWindMPH    *float64 `json:"max_wind_mph,omitempty"`
}

type jsonHistoricalWeather struct {
	Location  string             `json:"location,omitempty"`
	Latitude  float64            `json:"latitude"`
	Longitude float64            `json:"longitude"`
	Elevation *float64           `json:"elevation_m,omitempty"`
	Days      []jsonHistoryDay   `json:"days"`
	Summary   jsonHistorySummary `json:"summary"`
}

// RenderHistory dispatches to JSON or pretty (TTY) output.
func RenderHistory(history *models.HistoricalWeather, opts HistoryOptions) error {
	return RenderHistoryTo(os.Stdout, history, opts)
}

// RenderHistoryTo renders historical weather to the provided writer.
func RenderHistoryTo(w io.Writer, history *models.HistoricalWeather, opts HistoryOptions) error {
	if history == nil {
		return fmt.Errorf("no historical weather data")
	}

	if opts.ForceJSON || !isTTY {
		return renderHistoryJSON(w, history)
	}

	return renderHistoryPretty(w, history, opts)
}

func renderHistoryJSON(w io.Writer, h *models.HistoricalWeather) error {
	days := make([]jsonHistoryDay, 0, len(h.Days))
	for _, d := range h.Days {
		jd := jsonHistoryDay{
			Date:          d.Date.Format("2006-01-02"),
			ConditionCode: d.ConditionCode,
			Description:   d.Description,
			TempMaxC:      d.TempMaxC,
			TempMinC:      d.TempMinC,
			ApparentMaxC:  d.ApparentMaxC,
			ApparentMinC:  d.ApparentMinC,
			PrecipSumMM:   d.PrecipSumMM,
			WindMaxKPH:    d.WindMaxKPH,
		}

		if d.TempMaxC != nil {
			f := CelsiusToFahrenheit(*d.TempMaxC)
			jd.TempMaxF = &f
		}
		if d.TempMinC != nil {
			f := CelsiusToFahrenheit(*d.TempMinC)
			jd.TempMinF = &f
		}
		if d.ApparentMaxC != nil {
			f := CelsiusToFahrenheit(*d.ApparentMaxC)
			jd.ApparentMaxF = &f
		}
		if d.ApparentMinC != nil {
			f := CelsiusToFahrenheit(*d.ApparentMinC)
			jd.ApparentMinF = &f
		}
		if d.PrecipSumMM != nil {
			in := *d.PrecipSumMM / 25.4
			jd.PrecipSumIn = &in
		}
		if d.WindMaxKPH != nil {
			mph := KphToMPH(*d.WindMaxKPH)
			jd.WindMaxMPH = &mph
		}

		days = append(days, jd)
	}

	summary := jsonHistorySummary{
		DaysCount:     h.Summary.DaysCount,
		AvgTempMaxC:   h.Summary.AvgTempMaxC,
		AvgTempMinC:   h.Summary.AvgTempMinC,
		TotalPrecipMM: h.Summary.TotalPrecipMM,
		MaxWindKPH:    h.Summary.MaxWindKPH,
	}

	if h.Summary.AvgTempMaxC != nil {
		f := CelsiusToFahrenheit(*h.Summary.AvgTempMaxC)
		summary.AvgTempMaxF = &f
	}
	if h.Summary.AvgTempMinC != nil {
		f := CelsiusToFahrenheit(*h.Summary.AvgTempMinC)
		summary.AvgTempMinF = &f
	}
	if h.Summary.TotalPrecipMM != nil {
		in := *h.Summary.TotalPrecipMM / 25.4
		summary.TotalPrecipIn = &in
	}
	if h.Summary.MaxWindKPH != nil {
		mph := KphToMPH(*h.Summary.MaxWindKPH)
		summary.MaxWindMPH = &mph
	}

	payload := jsonHistoricalWeather{
		Location:  h.Location,
		Latitude:  h.Latitude,
		Longitude: h.Longitude,
		Elevation: h.Elevation,
		Days:      days,
		Summary:   summary,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func renderHistoryPretty(w io.Writer, h *models.HistoricalWeather, opts HistoryOptions) error {
	imperial := opts.Units != "metric"

	// Header
	title := "Historical Weather"
	if h.Location != "" {
		title = fmt.Sprintf("%s — Past %d Days", h.Location, len(h.Days))
	} else {
		title = fmt.Sprintf("Past %d Days Observations", len(h.Days))
	}
	fmt.Fprintln(w, styleLocation.Render(title))
	fmt.Fprintln(w)

	// Columns
	colDate := lipgloss.NewStyle().Width(12).Foreground(lipgloss.Color("244")).Bold(true).Render("DATE")
	colHighLow := lipgloss.NewStyle().Width(18).Foreground(lipgloss.Color("244")).Bold(true).Render("HIGH / LOW")
	colFeels := lipgloss.NewStyle().Width(18).Foreground(lipgloss.Color("244")).Bold(true).Render("FEELS LIKE")
	colPrecip := lipgloss.NewStyle().Width(10).Foreground(lipgloss.Color("244")).Bold(true).Render("PRECIP")
	colWind := lipgloss.NewStyle().Width(10).Foreground(lipgloss.Color("244")).Bold(true).Render("MAX WIND")
	colDesc := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Bold(true).Render("CONDITIONS")

	fmt.Fprintf(w, "  %s %s %s %s %s %s\n", colDate, colHighLow, colFeels, colPrecip, colWind, colDesc)
	fmt.Fprintln(w, styleLabel.Render("  "+strings.Repeat("─", 82)))

	for _, d := range h.Days {
		dateStr := d.Date.Format("Mon Jan 02")
		dateStyled := styleForecastName.Width(12).Render(dateStr)

		// High / Low
		highStr := " — "
		if d.TempMaxC != nil {
			highStr = FormatTemp(*d.TempMaxC, imperial)
		}
		lowStr := " — "
		if d.TempMinC != nil {
			lowStr = FormatTemp(*d.TempMinC, imperial)
		}

		var highStyled, lowStyled string
		if d.TempMaxC != nil {
			highStyled = TempStyle(*d.TempMaxC, imperial).Render(highStr)
		} else {
			highStyled = styleLabel.Render(highStr)
		}
		if d.TempMinC != nil {
			lowStyled = TempStyle(*d.TempMinC, imperial).Render(lowStr)
		} else {
			lowStyled = styleLabel.Render(lowStr)
		}
		hlFormatted := fmt.Sprintf("%s / %s", highStyled, lowStyled)
		hlStyled := lipgloss.NewStyle().Width(18).Render(hlFormatted)

		// Feels Like
		fHighStr := " — "
		if d.ApparentMaxC != nil {
			fHighStr = FormatTemp(*d.ApparentMaxC, imperial)
		}
		fLowStr := " — "
		if d.ApparentMinC != nil {
			fLowStr = FormatTemp(*d.ApparentMinC, imperial)
		}
		var fHighStyled, fLowStyled string
		if d.ApparentMaxC != nil {
			fHighStyled = TempStyle(*d.ApparentMaxC, imperial).Render(fHighStr)
		} else {
			fHighStyled = styleLabel.Render(fHighStr)
		}
		if d.ApparentMinC != nil {
			fLowStyled = TempStyle(*d.ApparentMinC, imperial).Render(fLowStr)
		} else {
			fLowStyled = styleLabel.Render(fLowStr)
		}
		feelsFormatted := fmt.Sprintf("%s / %s", fHighStyled, fLowStyled)
		feelsStyled := lipgloss.NewStyle().Width(18).Render(feelsFormatted)

		// Precip
		var precipStr string
		if d.PrecipSumMM != nil && *d.PrecipSumMM > 0.01 {
			if imperial {
				precipStr = fmt.Sprintf("%.2f in", *d.PrecipSumMM/25.4)
			} else {
				precipStr = fmt.Sprintf("%.1f mm", *d.PrecipSumMM)
			}
		} else {
			precipStr = "   —   "
		}
		precipStyled := styleForecastLow.Width(10).Render(precipStr)

		// Max Wind
		var windStr string
		if d.WindMaxKPH != nil && *d.WindMaxKPH > 0 {
			if imperial {
				windStr = fmt.Sprintf("%.0f mph", KphToMPH(*d.WindMaxKPH))
			} else {
				windStr = fmt.Sprintf("%.0f km/h", *d.WindMaxKPH)
			}
		} else {
			windStr = "  —   "
		}
		windStyled := styleLabel.Width(10).Render(windStr)

		// Description
		desc := styleForecastDesc.Render(d.Description)

		fmt.Fprintf(w, "  %s %s %s %s %s %s\n", dateStyled, hlStyled, feelsStyled, precipStyled, windStyled, desc)
	}

	fmt.Fprintln(w, styleLabel.Render("  "+strings.Repeat("─", 82)))

	// Summary Footer
	s := h.Summary
	summaryParts := []string{
		fmt.Sprintf("Days: %d", s.DaysCount),
	}
	if s.AvgTempMaxC != nil {
		summaryParts = append(summaryParts, fmt.Sprintf("Avg High: %s", FormatTemp(*s.AvgTempMaxC, imperial)))
	}
	if s.AvgTempMinC != nil {
		summaryParts = append(summaryParts, fmt.Sprintf("Avg Low: %s", FormatTemp(*s.AvgTempMinC, imperial)))
	}
	if s.TotalPrecipMM != nil {
		if imperial {
			summaryParts = append(summaryParts, fmt.Sprintf("Precip: %.2f in", *s.TotalPrecipMM/25.4))
		} else {
			summaryParts = append(summaryParts, fmt.Sprintf("Precip: %.1f mm", *s.TotalPrecipMM))
		}
	}
	if s.MaxWindKPH != nil {
		if imperial {
			summaryParts = append(summaryParts, fmt.Sprintf("Peak Wind: %.0f mph", KphToMPH(*s.MaxWindKPH)))
		} else {
			summaryParts = append(summaryParts, fmt.Sprintf("Peak Wind: %.0f km/h", *s.MaxWindKPH))
		}
	}

	summaryLine := strings.Join(summaryParts, "   ")
	fmt.Fprintf(w, "  %s  %s\n\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Render("SUMMARY"), styleValue.Render(summaryLine))

	return nil
}
