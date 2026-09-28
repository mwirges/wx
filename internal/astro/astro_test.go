package astro

import (
	"errors"
	"testing"
	"time"
)

func TestCalculate_USCities(t *testing.T) {
	locNY, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	locLA, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	locChi, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	locHon, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		lat, lon    float64
		date        time.Time
		wantSRHour  int
		wantSRMin   int
		wantSSHour  int
		wantSSMin   int
		minDayHours float64
		maxDayHours float64
	}{
		{
			name:        "New York (Sept 26 Equinox Window)",
			lat:         40.7128,
			lon:         -74.0060,
			date:        time.Date(2026, 9, 26, 12, 0, 0, 0, locNY),
			wantSRHour:  6,
			wantSRMin:   47,
			wantSSHour:  18,
			wantSSMin:   48,
			minDayHours: 11.9,
			maxDayHours: 12.2,
		},
		{
			name:        "Los Angeles (Sept 26 Equinox Window)",
			lat:         34.0522,
			lon:         -118.2437,
			date:        time.Date(2026, 9, 26, 12, 0, 0, 0, locLA),
			wantSRHour:  6,
			wantSRMin:   44,
			wantSSHour:  18,
			wantSSMin:   45,
			minDayHours: 11.9,
			maxDayHours: 12.2,
		},
		{
			name:        "Chicago (Sept 26 Equinox Window)",
			lat:         41.8781,
			lon:         -87.6298,
			date:        time.Date(2026, 9, 26, 12, 0, 0, 0, locChi),
			wantSRHour:  6,
			wantSRMin:   42,
			wantSSHour:  18,
			wantSSMin:   42,
			minDayHours: 11.9,
			maxDayHours: 12.2,
		},
		{
			name:        "Honolulu (Sept 26)",
			lat:         21.3069,
			lon:         -157.8583,
			date:        time.Date(2026, 9, 26, 12, 0, 0, 0, locHon),
			wantSRHour:  6,
			wantSRMin:   21,
			wantSSHour:  18,
			wantSSMin:   25,
			minDayHours: 11.9,
			maxDayHours: 12.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Calculate(tt.lat, tt.lon, tt.date)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Sunrise == nil || res.Sunset == nil {
				t.Fatalf("expected non-nil sunrise and sunset")
			}

			// Tolerance of ±2 minutes against solar tables
			srH, srM, _ := res.Sunrise.Clock()
			diffSR := (srH-tt.wantSRHour)*60 + (srM - tt.wantSRMin)
			if diffSR < -2 || diffSR > 2 {
				t.Errorf("Sunrise got %02d:%02d, want %02d:%02d (diff %d min)", srH, srM, tt.wantSRHour, tt.wantSRMin, diffSR)
			}

			ssH, ssM, _ := res.Sunset.Clock()
			diffSS := (ssH-tt.wantSSHour)*60 + (ssM - tt.wantSSMin)
			if diffSS < -2 || diffSS > 2 {
				t.Errorf("Sunset got %02d:%02d, want %02d:%02d (diff %d min)", ssH, ssM, tt.wantSSHour, tt.wantSSMin, diffSS)
			}

			dayHours := res.DayLength.Hours()
			if dayHours < tt.minDayHours || dayHours > tt.maxDayHours {
				t.Errorf("DayLength got %.2f hours, want between %.2f and %.2f", dayHours, tt.minDayHours, tt.maxDayHours)
			}
		})
	}
}

func TestCalculate_PolarEdgeCases(t *testing.T) {
	locAK, err := time.LoadLocation("America/Anchorage")
	if err != nil {
		t.Fatal(err)
	}

	// Utqiagvik (Barrow), AK: Lat 71.2906, Lon -156.7886
	lat, lon := 71.2906, -156.7886

	t.Run("Polar Night (Winter Solstice)", func(t *testing.T) {
		date := time.Date(2026, 12, 21, 12, 0, 0, 0, locAK)
		res, err := Calculate(lat, lon, date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.IsPolarNight {
			t.Errorf("expected IsPolarNight = true")
		}
		if res.Sunrise != nil || res.Sunset != nil {
			t.Errorf("expected nil sunrise and sunset for polar night")
		}
		if res.DayLength != 0 {
			t.Errorf("expected DayLength = 0, got %v", res.DayLength)
		}

		_, _, errSS := SunriseSunset(lat, lon, date)
		if !errors.Is(errSS, ErrPolarNight) {
			t.Errorf("expected ErrPolarNight, got %v", errSS)
		}
	})

	t.Run("Midnight Sun (Summer Solstice)", func(t *testing.T) {
		date := time.Date(2026, 6, 21, 12, 0, 0, 0, locAK)
		res, err := Calculate(lat, lon, date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.IsPolarDay {
			t.Errorf("expected IsPolarDay = true")
		}
		if res.Sunrise != nil || res.Sunset != nil {
			t.Errorf("expected nil sunrise and sunset for polar day")
		}
		if res.DayLength != 24*time.Hour {
			t.Errorf("expected DayLength = 24h, got %v", res.DayLength)
		}

		_, _, errSS := SunriseSunset(lat, lon, date)
		if !errors.Is(errSS, ErrPolarDay) {
			t.Errorf("expected ErrPolarDay, got %v", errSS)
		}
	})
}

func TestSunriseSunset_And_DayLength(t *testing.T) {
	loc, _ := time.LoadLocation("America/Indiana/Indianapolis")
	date := time.Date(2026, 9, 26, 12, 0, 0, 0, loc)
	lat, lon := 41.0793, -85.1394 // Fort Wayne, IN

	sr, ss, err := SunriseSunset(lat, lon, date)
	if err != nil {
		t.Fatalf("SunriseSunset error: %v", err)
	}

	dl := DayLength(sr, ss)
	if dl.Hours() < 11.9 || dl.Hours() > 12.2 {
		t.Errorf("expected DayLength ~12h around equinox, got %v", dl)
	}

	if DayLength(time.Time{}, ss) != 0 {
		t.Errorf("expected 0 for zero sunrise")
	}
}

func TestCalculate_ZeroTime(t *testing.T) {
	res, err := Calculate(40.7128, -74.0060, time.Time{})
	if err != nil {
		t.Fatalf("Calculate with zero time returned error: %v", err)
	}
	if res.SolarNoon.IsZero() {
		t.Errorf("expected non-zero SolarNoon")
	}
}

func TestCalculate_TwilightsAndSolarCoordinates(t *testing.T) {
	loc, _ := time.LoadLocation("America/Indiana/Indianapolis")
	noon := time.Date(2026, 9, 26, 13, 30, 0, 0, loc) // close to solar noon
	lat, lon := 41.0793, -85.1394                     // Fort Wayne, IN

	res, err := Calculate(lat, lon, noon)
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}

	// Verify twilights exist and are chronologically ordered:
	// AstroDawn < NauticalDawn < CivilDawn < Sunrise
	if res.AstroDawn == nil || res.NauticalDawn == nil || res.CivilDawn == nil || res.Sunrise == nil {
		t.Fatalf("expected morning twilights to be populated")
	}
	if !res.AstroDawn.Before(*res.NauticalDawn) {
		t.Errorf("AstroDawn (%v) should be before NauticalDawn (%v)", res.AstroDawn, res.NauticalDawn)
	}
	if !res.NauticalDawn.Before(*res.CivilDawn) {
		t.Errorf("NauticalDawn (%v) should be before CivilDawn (%v)", res.NauticalDawn, res.CivilDawn)
	}
	if !res.CivilDawn.Before(*res.Sunrise) {
		t.Errorf("CivilDawn (%v) should be before Sunrise (%v)", res.CivilDawn, res.Sunrise)
	}

	// Sunset < CivilDusk < NauticalDusk < AstroDusk
	if res.Sunset == nil || res.CivilDusk == nil || res.NauticalDusk == nil || res.AstroDusk == nil {
		t.Fatalf("expected evening twilights to be populated")
	}
	if !res.Sunset.Before(*res.CivilDusk) {
		t.Errorf("Sunset (%v) should be before CivilDusk (%v)", res.Sunset, res.CivilDusk)
	}
	if !res.CivilDusk.Before(*res.NauticalDusk) {
		t.Errorf("CivilDusk (%v) should be before NauticalDusk (%v)", res.CivilDusk, res.NauticalDusk)
	}
	if !res.NauticalDusk.Before(*res.AstroDusk) {
		t.Errorf("NauticalDusk (%v) should be before AstroDusk (%v)", res.NauticalDusk, res.AstroDusk)
	}

	// Golden hour
	if res.GoldenHourMorningStart == nil || res.GoldenHourMorningEnd == nil {
		t.Errorf("expected morning golden hour to be populated")
	}
	if res.GoldenHourEveningStart == nil || res.GoldenHourEveningEnd == nil {
		t.Errorf("expected evening golden hour to be populated")
	}

	// Solar elevation and azimuth at midday
	if res.SolarElevationDeg == nil || *res.SolarElevationDeg < 30.0 {
		t.Errorf("expected midday solar elevation > 30 deg, got %v", res.SolarElevationDeg)
	}
	if res.SolarAzimuthDeg == nil || *res.SolarAzimuthDeg < 150.0 || *res.SolarAzimuthDeg > 220.0 {
		t.Errorf("expected midday solar azimuth ~180 deg (South), got %v", res.SolarAzimuthDeg)
	}
	if res.CurrentPeriod != "Daylight" {
		t.Errorf("expected current period 'Daylight', got %q", res.CurrentPeriod)
	}
}

