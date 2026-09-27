// Package astro provides pure Go astronomical calculations.
package astro

import (
	"math"
	"time"
)

// MoonInfo holds calculated lunar phase and illumination details.
type MoonInfo struct {
	PhaseName       string  // e.g. "Waxing Gibbous", "Full Moon"
	PhaseIcon       string  // Unicode emoji: 🌑, 🌒, 🌓, 🌔, 🌕, 🌖, 🌗, 🌘
	IlluminationPct float64 // 0.0 to 100.0 percent of visible disk illuminated
	AgeDays         float64 // 0.0 to 29.53 days since last new moon
	PhaseFraction   float64 // 0.0 to 1.0 progress through lunar cycle
}

// SynodicMonth is the average duration in days between identical lunar phases (≈ 29.53059 days).
const SynodicMonth = 29.530588853

// CalculateMoon computes the lunar phase, illuminated fraction, phase name, and age
// for the given time t. If t is zero, time.Now() is used.
func CalculateMoon(t time.Time) MoonInfo {
	if t.IsZero() {
		t = time.Now()
	}

	utc := t.UTC()
	year, month, day := utc.Date()
	hour, min, sec := utc.Clock()
	nsec := utc.Nanosecond()

	m := int(month)
	y := year
	if m <= 2 {
		y -= 1
		m += 12
	}
	A := math.Floor(float64(y) / 100.0)
	B := 2.0 - A + math.Floor(A/4.0)
	dayFrac := (float64(hour) + float64(min)/60.0 + (float64(sec)+float64(nsec)/1e9)/3600.0) / 24.0
	jd := math.Floor(365.25*float64(y+4716)) + math.Floor(30.6001*float64(m+1)) + float64(day) + dayFrac + B - 1524.5

	// Julian centuries since J2000.0
	T := (jd - 2451545.0) / 36525.0

	// Meeus Chapter 48: Illuminated Fraction of the Moon
	// Mean elongation of the Moon
	D := 297.8501921 + 445267.1114034*T - 0.0018819*T*T + (T*T*T)/545868.0
	// Sun's mean anomaly
	M := 357.5291092 + 35999.0502909*T - 0.0001536*T*T + (T*T*T)/24490000.0
	// Moon's mean anomaly
	Mprime := 134.9633964 + 477198.8675055*T + 0.0087414*T*T + (T*T*T)/69699.0

	D = math.Mod(D, 360.0)
	if D < 0 {
		D += 360.0
	}
	M = math.Mod(M, 360.0)
	if M < 0 {
		M += 360.0
	}
	Mprime = math.Mod(Mprime, 360.0)
	if Mprime < 0 {
		Mprime += 360.0
	}

	dr := degToRad

	// Phase angle i (angle Sun-Moon-Earth) in degrees
	iDeg := 180.0 - D - 6.289*math.Sin(dr(Mprime)) +
		2.100*math.Sin(dr(M)) -
		1.274*math.Sin(dr(2*D-Mprime)) -
		0.658*math.Sin(dr(2*D)) -
		0.214*math.Sin(dr(2*Mprime)) -
		0.110*math.Sin(dr(D))

	// Illuminated fraction k (0.0 to 1.0)
	k := (1.0 + math.Cos(dr(iDeg))) / 2.0
	if k < 0.0 {
		k = 0.0
	} else if k > 1.0 {
		k = 1.0
	}

	// True elongation angle taking into account primary orbital perturbations
	elongation := D + 6.289*math.Sin(dr(Mprime)) -
		2.100*math.Sin(dr(M)) +
		1.274*math.Sin(dr(2*D-Mprime)) +
		0.658*math.Sin(dr(2*D))

	elongation = math.Mod(elongation, 360.0)
	if elongation < 0 {
		elongation += 360.0
	}

	phaseFrac := elongation / 360.0
	ageDays := phaseFrac * SynodicMonth

	name, icon := classifyMoonPhase(phaseFrac)

	return MoonInfo{
		PhaseName:       name,
		PhaseIcon:       icon,
		IlluminationPct: math.Round(k*1000.0) / 10.0, // 1 decimal place, e.g. 84.5
		AgeDays:         math.Round(ageDays*10.0) / 10.0,
		PhaseFraction:   phaseFrac,
	}
}

func classifyMoonPhase(phaseFrac float64) (string, string) {
	switch {
	case phaseFrac < 0.025 || phaseFrac >= 0.975:
		return "New Moon", "🌑"
	case phaseFrac < 0.225:
		return "Waxing Crescent", "🌒"
	case phaseFrac < 0.275:
		return "First Quarter", "🌓"
	case phaseFrac < 0.475:
		return "Waxing Gibbous", "🌔"
	case phaseFrac < 0.525:
		return "Full Moon", "🌕"
	case phaseFrac < 0.725:
		return "Waning Gibbous", "🌖"
	case phaseFrac < 0.775:
		return "Last Quarter", "🌗"
	default:
		return "Waning Crescent", "🌘"
	}
}
