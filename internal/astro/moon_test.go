package astro

import (
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func TestCalculateMoon_KnownPhases(t *testing.T) {
	tests := []struct {
		name         string
		utcTime      time.Time
		wantPhase    string
		wantIcon     string
		minIllum     float64
		maxIllum     float64
		minAge       float64
		maxAge       float64
	}{
		{
			name:      "New Moon 2024-01-11",
			utcTime:   time.Date(2024, 1, 11, 11, 57, 0, 0, time.UTC),
			wantPhase: "New Moon",
			wantIcon:  "🌑",
			minIllum:  0.0,
			maxIllum:  3.0,
			minAge:    0.0,
			maxAge:    2.0,
		},
		{
			name:      "New Moon / Eclipse 2024-04-08",
			utcTime:   time.Date(2024, 4, 8, 18, 21, 0, 0, time.UTC),
			wantPhase: "New Moon",
			wantIcon:  "🌑",
			minIllum:  0.0,
			maxIllum:  3.0,
		},
		{
			name:      "First Quarter 2024-01-18",
			utcTime:   time.Date(2024, 1, 18, 3, 53, 0, 0, time.UTC),
			wantPhase: "First Quarter",
			wantIcon:  "🌓",
			minIllum:  45.0,
			maxIllum:  55.0,
		},
		{
			name:      "Full Moon 2024-01-25",
			utcTime:   time.Date(2024, 1, 25, 17, 54, 0, 0, time.UTC),
			wantPhase: "Full Moon",
			wantIcon:  "🌕",
			minIllum:  97.0,
			maxIllum:  100.0,
		},
		{
			name:      "Last Quarter 2024-02-02",
			utcTime:   time.Date(2024, 2, 2, 23, 18, 0, 0, time.UTC),
			wantPhase: "Last Quarter",
			wantIcon:  "🌗",
			minIllum:  45.0,
			maxIllum:  55.0,
		},
		{
			name:      "Waxing Crescent 2024-01-14",
			utcTime:   time.Date(2024, 1, 14, 12, 0, 0, 0, time.UTC),
			wantPhase: "Waxing Crescent",
			wantIcon:  "🌒",
			minIllum:  5.0,
			maxIllum:  30.0,
		},
		{
			name:      "Waxing Gibbous 2024-01-21",
			utcTime:   time.Date(2024, 1, 21, 12, 0, 0, 0, time.UTC),
			wantPhase: "Waxing Gibbous",
			wantIcon:  "🌔",
			minIllum:  70.0,
			maxIllum:  95.0,
		},
		{
			name:      "Waning Gibbous 2024-01-28",
			utcTime:   time.Date(2024, 1, 28, 12, 0, 0, 0, time.UTC),
			wantPhase: "Waning Gibbous",
			wantIcon:  "🌖",
			minIllum:  70.0,
			maxIllum:  98.0,
		},
		{
			name:      "Waning Crescent 2024-02-06",
			utcTime:   time.Date(2024, 2, 6, 12, 0, 0, 0, time.UTC),
			wantPhase: "Waning Crescent",
			wantIcon:  "🌘",
			minIllum:  10.0,
			maxIllum:  35.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := CalculateMoon(tt.utcTime)
			if m.PhaseName != tt.wantPhase {
				t.Errorf("PhaseName = %q, want %q (phaseFrac = %f)", m.PhaseName, tt.wantPhase, m.PhaseFraction)
			}
			if m.PhaseIcon != tt.wantIcon {
				t.Errorf("PhaseIcon = %q, want %q", m.PhaseIcon, tt.wantIcon)
			}
			if m.IlluminationPct < tt.minIllum || m.IlluminationPct > tt.maxIllum {
				t.Errorf("IlluminationPct = %.1f%%, want between %.1f%% and %.1f%%", m.IlluminationPct, tt.minIllum, tt.maxIllum)
			}
		})
	}
}

func TestAstronomyModel_MoonHelpers(t *testing.T) {
	illum := 84.5
	age := 12.4
	a := &models.Astronomy{
		MoonPhase:           "Waxing Gibbous",
		MoonPhaseIcon:       "🌔",
		MoonIlluminationPct: &illum,
		MoonAgeDays:         &age,
	}

	if got := a.MoonIlluminationStr(); got != "85%" {
		t.Errorf("MoonIlluminationStr() = %q, want '85%%'", got)
	}
	if got := a.MoonAgeStr(); got != "12.4d" {
		t.Errorf("MoonAgeStr() = %q, want '12.4d'", got)
	}

	var nilA *models.Astronomy
	if got := nilA.MoonIlluminationStr(); got != "" {
		t.Errorf("expected empty string for nil astronomy, got %q", got)
	}
	if got := nilA.MoonAgeStr(); got != "" {
		t.Errorf("expected empty string for nil astronomy, got %q", got)
	}
}
