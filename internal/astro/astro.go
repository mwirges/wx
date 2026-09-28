// Package astro provides pure Go astronomical calculations based on NOAA solar equations.
// It has zero external dependencies or network requests.
package astro

import (
	"errors"
	"math"
	"time"

	"github.com/mwirges/wx/internal/models"
)

var (
	// ErrPolarNight is returned when the sun does not rise above the horizon on the given date.
	ErrPolarNight = errors.New("polar night: sun does not rise")
	// ErrPolarDay is returned when the sun does not set below the horizon on the given date (midnight sun).
	ErrPolarDay = errors.New("polar day: sun does not set")
)

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180.0
}

func radToDeg(rad float64) float64 {
	return rad * 180.0 / math.Pi
}

// Calculate computes solar times (sunrise, sunset, solar noon, and day length)
// for the given latitude and longitude on the local date of t.
// If t is zero, time.Now() is used.
func Calculate(lat, lon float64, t time.Time) (*models.Astronomy, error) {
	if t.IsZero() {
		t = time.Now()
	}

	year, month, day := t.Date()

	m := int(month)
	y := year
	if m <= 2 {
		y -= 1
		m += 12
	}
	A := math.Floor(float64(y) / 100.0)
	B := 2 - A + math.Floor(A/4.0)
	jd := math.Floor(365.25*float64(y+4716)) + math.Floor(30.6001*float64(m+1)) + float64(day) + B - 1524.5

	T := (jd - 2451545.0) / 36525.0

	geomMeanLongSun := math.Mod(280.46646+T*(36000.76983+T*0.0003032), 360.0)
	if geomMeanLongSun < 0 {
		geomMeanLongSun += 360.0
	}

	geomMeanAnomSun := 357.52911 + T*(35999.05029-0.0001537*T)
	eccentEarthOrbit := 0.016708634 - T*(0.000042037+0.0000001267*T)

	sunEqOfCtr := math.Sin(degToRad(geomMeanAnomSun))*(1.914602-T*(0.004817+0.000014*T)) +
		math.Sin(degToRad(2*geomMeanAnomSun))*(0.019993-0.000101*T) +
		math.Sin(degToRad(3*geomMeanAnomSun))*0.000289

	sunTrueLong := geomMeanLongSun + sunEqOfCtr
	sunAppLong := sunTrueLong - 0.00569 - 0.00478*math.Sin(degToRad(125.04-1934.136*T))

	meanObliqEcliptic := 23.0 + (26.0+((21.448-T*(46.815+T*(0.00059-T*0.001813))))/60.0)/60.0
	obliqCorr := meanObliqEcliptic + 0.00256*math.Cos(degToRad(125.04-1934.136*T))

	sinDeclination := math.Sin(degToRad(obliqCorr)) * math.Sin(degToRad(sunAppLong))
	sunDeclin := radToDeg(math.Asin(sinDeclination))

	vary := math.Tan(degToRad(obliqCorr)/2.0) * math.Tan(degToRad(obliqCorr)/2.0)

	eqOfTime := 4.0 * radToDeg(vary*math.Sin(2*degToRad(geomMeanLongSun))-
		2.0*eccentEarthOrbit*math.Sin(degToRad(geomMeanAnomSun))+
		4.0*eccentEarthOrbit*vary*math.Sin(degToRad(geomMeanAnomSun))*math.Cos(2*degToRad(geomMeanLongSun))-
		0.5*vary*vary*math.Sin(4*degToRad(geomMeanLongSun))-
		1.25*eccentEarthOrbit*eccentEarthOrbit*math.Sin(2*degToRad(geomMeanAnomSun)))

	solarNoonUTC := 720.0 - 4.0*lon - eqOfTime

	baseMidnight := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	snTime := baseMidnight.Add(time.Duration(solarNoonUTC * 60 * float64(time.Second))).In(t.Location())

	// Standard atmospheric refraction + solar disc semi-diameter zenith angle: 90° 50' = 90.833333°
	cosHA := (math.Cos(degToRad(90.83333333333333)) - math.Sin(degToRad(lat))*math.Sin(degToRad(sunDeclin))) /
		(math.Cos(degToRad(lat)) * math.Cos(degToRad(sunDeclin)))

	moon := CalculateMoon(t)
	illum := moon.IlluminationPct
	age := moon.AgeDays

	// Twilight & Golden Hour horizons
	civDawn, civDusk := calcHorizon(solarNoonUTC, sunDeclin, lat, 96.0, baseMidnight, t)
	nautDawn, nautDusk := calcHorizon(solarNoonUTC, sunDeclin, lat, 102.0, baseMidnight, t)
	astroDawn, astroDusk := calcHorizon(solarNoonUTC, sunDeclin, lat, 108.0, baseMidnight, t)
	ghStartMorn, ghEndEve := calcHorizon(solarNoonUTC, sunDeclin, lat, 94.0, baseMidnight, t)
	ghEndMorn, ghStartEve := calcHorizon(solarNoonUTC, sunDeclin, lat, 84.0, baseMidnight, t)

	// Instantaneous solar positioning
	hour, min, sec := t.UTC().Clock()
	minsUTC := float64(hour*60+min) + float64(sec)/60.0 + float64(t.UTC().Nanosecond())/1e9/60.0
	haInst := (minsUTC - solarNoonUTC) / 4.0

	sinElev := math.Sin(degToRad(lat))*math.Sin(degToRad(sunDeclin)) +
		math.Cos(degToRad(lat))*math.Cos(degToRad(sunDeclin))*math.Cos(degToRad(haInst))
	if sinElev > 1.0 {
		sinElev = 1.0
	} else if sinElev < -1.0 {
		sinElev = -1.0
	}
	rawElev := radToDeg(math.Asin(sinElev))

	refraction := 0.0
	if rawElev > -0.575 {
		refraction = 1.02 / math.Tan(degToRad(rawElev+10.3/(rawElev+5.11))) / 60.0
	}
	solarElev := rawElev + refraction

	yAz := -math.Sin(degToRad(haInst))
	xAz := math.Tan(degToRad(sunDeclin))*math.Cos(degToRad(lat)) - math.Sin(degToRad(lat))*math.Cos(degToRad(haInst))
	solarAz := math.Mod(radToDeg(math.Atan2(yAz, xAz))+360.0, 360.0)

	var currentPeriod string
	switch {
	case solarElev > 6.0:
		currentPeriod = "Daylight"
	case solarElev >= -4.0 && solarElev <= 6.0:
		currentPeriod = "Golden Hour"
	case solarElev >= -6.0 && solarElev < -4.0:
		currentPeriod = "Civil Twilight"
	case solarElev >= -12.0 && solarElev < -6.0:
		currentPeriod = "Nautical Twilight"
	case solarElev >= -18.0 && solarElev < -12.0:
		currentPeriod = "Astronomical Twilight"
	default:
		currentPeriod = "Night"
	}

	if cosHA > 1.0 {
		return &models.Astronomy{
			SolarNoon:              snTime.Round(time.Minute),
			DayLength:              0,
			IsPolarNight:           true,
			CivilDawn:              civDawn,
			CivilDusk:              civDusk,
			NauticalDawn:           nautDawn,
			NauticalDusk:           nautDusk,
			AstroDawn:              astroDawn,
			AstroDusk:              astroDusk,
			GoldenHourMorningStart: ghStartMorn,
			GoldenHourMorningEnd:   ghEndMorn,
			GoldenHourEveningStart: ghStartEve,
			GoldenHourEveningEnd:   ghEndEve,
			SolarElevationDeg:      &solarElev,
			SolarAzimuthDeg:        &solarAz,
			CurrentPeriod:          currentPeriod,
			MoonPhase:              moon.PhaseName,
			MoonPhaseIcon:          moon.PhaseIcon,
			MoonIlluminationPct:    &illum,
			MoonAgeDays:            &age,
		}, nil
	}
	if cosHA < -1.0 {
		return &models.Astronomy{
			SolarNoon:              snTime.Round(time.Minute),
			DayLength:              24 * time.Hour,
			IsPolarDay:             true,
			CivilDawn:              civDawn,
			CivilDusk:              civDusk,
			NauticalDawn:           nautDawn,
			NauticalDusk:           nautDusk,
			AstroDawn:              astroDawn,
			AstroDusk:              astroDusk,
			GoldenHourMorningStart: ghStartMorn,
			GoldenHourMorningEnd:   ghEndMorn,
			GoldenHourEveningStart: ghStartEve,
			GoldenHourEveningEnd:   ghEndEve,
			SolarElevationDeg:      &solarElev,
			SolarAzimuthDeg:        &solarAz,
			CurrentPeriod:          currentPeriod,
			MoonPhase:              moon.PhaseName,
			MoonPhaseIcon:          moon.PhaseIcon,
			MoonIlluminationPct:    &illum,
			MoonAgeDays:            &age,
		}, nil
	}

	haSunrise := radToDeg(math.Acos(cosHA))

	sunriseUTC := solarNoonUTC - 4.0*haSunrise
	sunsetUTC := solarNoonUTC + 4.0*haSunrise

	srTime := baseMidnight.Add(time.Duration(sunriseUTC * 60 * float64(time.Second))).In(t.Location()).Round(time.Minute)
	ssTime := baseMidnight.Add(time.Duration(sunsetUTC * 60 * float64(time.Second))).In(t.Location()).Round(time.Minute)

	return &models.Astronomy{
		Sunrise:                &srTime,
		Sunset:                 &ssTime,
		SolarNoon:              snTime.Round(time.Minute),
		DayLength:              ssTime.Sub(srTime),
		CivilDawn:              civDawn,
		CivilDusk:              civDusk,
		NauticalDawn:           nautDawn,
		NauticalDusk:           nautDusk,
		AstroDawn:              astroDawn,
		AstroDusk:              astroDusk,
		GoldenHourMorningStart: ghStartMorn,
		GoldenHourMorningEnd:   ghEndMorn,
		GoldenHourEveningStart: ghStartEve,
		GoldenHourEveningEnd:   ghEndEve,
		SolarElevationDeg:      &solarElev,
		SolarAzimuthDeg:        &solarAz,
		CurrentPeriod:          currentPeriod,
		MoonPhase:              moon.PhaseName,
		MoonPhaseIcon:          moon.PhaseIcon,
		MoonIlluminationPct:    &illum,
		MoonAgeDays:            &age,
	}, nil
}

func calcHorizon(solarNoonUTC, sunDeclin, lat, zenithDeg float64, baseMidnight, t time.Time) (rise, set *time.Time) {
	cosHA := (math.Cos(degToRad(zenithDeg)) - math.Sin(degToRad(lat))*math.Sin(degToRad(sunDeclin))) /
		(math.Cos(degToRad(lat)) * math.Cos(degToRad(sunDeclin)))

	if cosHA > 1.0 || cosHA < -1.0 {
		return nil, nil
	}

	ha := radToDeg(math.Acos(cosHA))
	riseUTC := solarNoonUTC - 4.0*ha
	setUTC := solarNoonUTC + 4.0*ha

	rTime := baseMidnight.Add(time.Duration(riseUTC * 60 * float64(time.Second))).In(t.Location()).Round(time.Minute)
	sTime := baseMidnight.Add(time.Duration(setUTC * 60 * float64(time.Second))).In(t.Location()).Round(time.Minute)
	return &rTime, &sTime
}

// SunriseSunset returns sunrise and sunset times for the given coordinates and date.
// Returns ErrPolarNight or ErrPolarDay if the sun does not rise or set.
func SunriseSunset(lat, lon float64, date time.Time) (sunrise, sunset time.Time, err error) {
	a, err := Calculate(lat, lon, date)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if a.IsPolarNight {
		return time.Time{}, time.Time{}, ErrPolarNight
	}
	if a.IsPolarDay {
		return time.Time{}, time.Time{}, ErrPolarDay
	}
	if a.Sunrise == nil || a.Sunset == nil {
		return time.Time{}, time.Time{}, errors.New("astronomy: sunrise/sunset unavailable")
	}
	return *a.Sunrise, *a.Sunset, nil
}

// DayLength computes the duration between sunrise and sunset.
func DayLength(sunrise, sunset time.Time) time.Duration {
	if sunrise.IsZero() || sunset.IsZero() {
		return 0
	}
	return sunset.Sub(sunrise)
}
