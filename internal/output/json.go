package output

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mwirges/wx/internal/models"
)

type jsonAstronomy struct {
	Sunrise                *string  `json:"sunrise,omitempty"`
	Sunset                 *string  `json:"sunset,omitempty"`
	SolarNoon              string   `json:"solar_noon,omitempty"`
	DayLengthSec           int64    `json:"day_length_seconds"`
	DayLength              string   `json:"day_length"`
	IsPolarDay             bool     `json:"is_polar_day,omitempty"`
	IsPolarNight           bool     `json:"is_polar_night,omitempty"`
	CivilDawn              *string  `json:"civil_dawn,omitempty"`
	CivilDusk              *string  `json:"civil_dusk,omitempty"`
	NauticalDawn           *string  `json:"nautical_dawn,omitempty"`
	NauticalDusk           *string  `json:"nautical_dusk,omitempty"`
	AstroDawn              *string  `json:"astro_dawn,omitempty"`
	AstroDusk              *string  `json:"astro_dusk,omitempty"`
	GoldenHourMorningStart *string  `json:"golden_hour_morning_start,omitempty"`
	GoldenHourMorningEnd   *string  `json:"golden_hour_morning_end,omitempty"`
	GoldenHourEveningStart *string  `json:"golden_hour_evening_start,omitempty"`
	GoldenHourEveningEnd   *string  `json:"golden_hour_evening_end,omitempty"`
	SolarElevationDeg      *float64 `json:"solar_elevation_deg,omitempty"`
	SolarAzimuthDeg        *float64 `json:"solar_azimuth_deg,omitempty"`
	CurrentPeriod          string   `json:"current_period,omitempty"`
	MoonPhase              string   `json:"moon_phase,omitempty"`
	MoonPhaseIcon          string   `json:"moon_phase_icon,omitempty"`
	MoonIlluminationPct    *float64 `json:"moon_illumination_pct,omitempty"`
	MoonAgeDays            *float64 `json:"moon_age_days,omitempty"`
}

type jsonConditions struct {
	Station       string `json:"station"`
	ObservedAt    string `json:"observed_at"`
	Location      string `json:"location"`
	Description   string `json:"description,omitempty"`
	ConditionCode string `json:"condition_code,omitempty"`

	TempC      *float64 `json:"temperature_c,omitempty"`
	TempF      *float64 `json:"temperature_f,omitempty"`
	FeelsLikeC *float64 `json:"feels_like_c,omitempty"`
	FeelsLikeF *float64 `json:"feels_like_f,omitempty"`

	DewPointC   *float64 `json:"dew_point_c,omitempty"`
	DewPointF   *float64 `json:"dew_point_f,omitempty"`
	HumidityPct *float64 `json:"humidity_pct,omitempty"`

	WindKPH     *float64 `json:"wind_kph,omitempty"`
	WindMPH     *float64 `json:"wind_mph,omitempty"`
	WindGustKPH *float64 `json:"wind_gust_kph,omitempty"`
	WindGustMPH *float64 `json:"wind_gust_mph,omitempty"`
	WindDir     string   `json:"wind_direction,omitempty"`

	PressureHPA  *float64 `json:"pressure_hpa,omitempty"`
	PressureInHg *float64 `json:"pressure_inhg,omitempty"`
	VisibilityM  *float64 `json:"visibility_m,omitempty"`
	VisibilityMi *float64 `json:"visibility_mi,omitempty"`

	Astronomy  *jsonAstronomy  `json:"astronomy,omitempty"`
	AirQuality *jsonAirQuality `json:"air_quality,omitempty"`
	Nowcast    *models.Nowcast `json:"nowcast,omitempty"`
}

type jsonAirQuality struct {
	AQI            *int     `json:"aqi,omitempty"`
	Category       string   `json:"category,omitempty"`
	UVIndex        *float64 `json:"uv_index,omitempty"`
	UVCategory     string   `json:"uv_category,omitempty"`
	PM25           *float64 `json:"pm2_5,omitempty"`
	PM10           *float64 `json:"pm10,omitempty"`
	Ozone          *float64 `json:"ozone,omitempty"`
	NO2            *float64 `json:"no2,omitempty"`
	CO             *float64 `json:"co,omitempty"`
	SO2            *float64 `json:"so2,omitempty"`
	HealthAdvisory string   `json:"health_advisory,omitempty"`
	SmokeAdvisory  string   `json:"smoke_advisory,omitempty"`
}

type jsonPeriod struct {
	Name                       string   `json:"name"`
	StartTime                  string   `json:"start_time"`
	IsDaytime                  bool     `json:"is_daytime"`
	TempF                      float64  `json:"temperature_f,omitempty"`
	TempC                      float64  `json:"temperature_c,omitempty"`
	WindMPH                    float64  `json:"wind_mph,omitempty"`
	WindKPH                    float64  `json:"wind_kph,omitempty"`
	WindDir                    string   `json:"wind_direction,omitempty"`
	ShortDesc                  string   `json:"short_description,omitempty"`
	DetailedDesc               string   `json:"detailed_description,omitempty"`
	ProbabilityOfPrecipitation *float64 `json:"probability_of_precipitation,omitempty"`
	DewPointC                  *float64 `json:"dew_point_c,omitempty"`
	DewPointF                  *float64 `json:"dew_point_f,omitempty"`
	HumidityPct                *float64 `json:"humidity_pct,omitempty"`
}

type jsonForecast struct {
	GeneratedAt string       `json:"generated_at"`
	Periods     []jsonPeriod `json:"periods"`
	Hourly      []jsonPeriod `json:"hourly,omitempty"`
}

type jsonAlert struct {
	Event       string `json:"event"`
	Headline    string `json:"headline,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Urgency     string `json:"urgency,omitempty"`
	Effective   string `json:"effective,omitempty"`
	Expires     string `json:"expires,omitempty"`
	AreaDesc    string `json:"area,omitempty"`
	Description string `json:"description,omitempty"`
	Instruction string `json:"instruction,omitempty"`
}

type jsonFreshness struct {
	ObservedAt string `json:"observed_at,omitempty"`
	FetchedAt  string `json:"fetched_at,omitempty"`
	FromCache  bool   `json:"from_cache,omitempty"`
	AgeSeconds int64  `json:"age_seconds,omitempty"`
	Age        string `json:"age,omitempty"`
}

type jsonOutput struct {
	Conditions *jsonConditions `json:"conditions,omitempty"`
	Forecast   *jsonForecast   `json:"forecast,omitempty"`
	Alerts     []jsonAlert     `json:"alerts,omitempty"`
	Astronomy  *jsonAstronomy  `json:"astronomy,omitempty"`
	AirQuality *jsonAirQuality `json:"air_quality,omitempty"`
	Nowcast    *models.Nowcast `json:"nowcast,omitempty"`
	Freshness  *jsonFreshness  `json:"freshness,omitempty"`
}

func capHourly(periods []models.Period, limit int) []models.Period {
	if len(periods) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = 24
	}
	if len(periods) > limit {
		return periods[:limit]
	}
	return periods
}

func jsonPeriods(periods []models.Period, imperial bool) []jsonPeriod {
	out := make([]jsonPeriod, 0, len(periods))
	for _, p := range periods {
		jp := jsonPeriod{
			Name:                       p.Name,
			StartTime:                  p.StartTime.Format(time.RFC3339),
			IsDaytime:                  p.IsDaytime,
			TempC:                      p.TempC,
			WindKPH:                    p.WindKPH,
			WindDir:                    p.WindDir,
			ShortDesc:                  p.ShortDesc,
			DetailedDesc:               p.DetailedDesc,
			ProbabilityOfPrecipitation: p.ProbabilityOfPrecipitation,
			HumidityPct:                p.HumidityPct,
		}
		if p.DewPointC != nil {
			dpC := *p.DewPointC
			jp.DewPointC = &dpC
			if imperial {
				dpF := CelsiusToFahrenheit(dpC)
				jp.DewPointF = &dpF
			}
		}
		if imperial {
			jp.TempF = CelsiusToFahrenheit(p.TempC)
			jp.WindMPH = KphToMPH(p.WindKPH)
		}
		out = append(out, jp)
	}
	return out
}

func renderJSON(data RenderData, opts RenderOptions) error {
	imperial := opts.Units != "metric"

	out := jsonOutput{}

	freshness := data.Freshness
	if freshness.ObservedAt.IsZero() && data.Conditions != nil {
		freshness.ObservedAt = data.Conditions.ObservedAt
	}
	if !freshness.ObservedAt.IsZero() {
		jf := &jsonFreshness{
			ObservedAt: freshness.ObservedAt.Format(time.RFC3339),
			FromCache:  freshness.FromCache,
			AgeSeconds: int64(freshness.Age().Seconds()),
			Age:        freshness.AgeString(),
		}
		if !freshness.FetchedAt.IsZero() {
			jf.FetchedAt = freshness.FetchedAt.Format(time.RFC3339)
		}
		out.Freshness = jf
	}

	if data.Conditions != nil {
		c := data.Conditions
		jc := &jsonConditions{
			Station:       c.StationID,
			ObservedAt:    c.ObservedAt.Format(time.RFC3339),
			Location:      c.Location,
			Description:   c.Description,
			ConditionCode: c.ConditionCode,
			HumidityPct:   c.HumidityPct,
		}

		// Temperature
		if c.TempC != nil {
			tc := *c.TempC
			jc.TempC = &tc
			if imperial {
				tf := CelsiusToFahrenheit(tc)
				jc.TempF = &tf
			}
		}

		// Feels like (wind chill or heat index)
		fl := c.FeelsLikeC
		if fl == nil {
			fl = FeelsLikeTemp(c.WindChillC, c.HeatIndexC)
		}
		if fl != nil {
			flC := *fl
			jc.FeelsLikeC = &flC
			if imperial {
				flF := CelsiusToFahrenheit(flC)
				jc.FeelsLikeF = &flF
			}
		}

		// Dew point
		if c.DewPointC != nil {
			dp := *c.DewPointC
			jc.DewPointC = &dp
			if imperial {
				dpF := CelsiusToFahrenheit(dp)
				jc.DewPointF = &dpF
			}
		}

		// Wind
		if c.WindKPH != nil {
			kph := *c.WindKPH
			jc.WindKPH = &kph
			if imperial {
				mph := KphToMPH(kph)
				jc.WindMPH = &mph
			}
		}
		if c.WindGustKPH != nil {
			gust := *c.WindGustKPH
			jc.WindGustKPH = &gust
			if imperial {
				mph := KphToMPH(gust)
				jc.WindGustMPH = &mph
			}
		}
		if c.WindDegrees != nil {
			jc.WindDir = DegreesToCompass(*c.WindDegrees)
		}

		// Pressure
		if c.PressureHPA != nil {
			hpa := *c.PressureHPA
			jc.PressureHPA = &hpa
			if imperial {
				inhg := hpa / 33.8639
				jc.PressureInHg = &inhg
			}
		}

		// Visibility
		if c.VisibilityM != nil {
			m := *c.VisibilityM
			jc.VisibilityM = &m
			if imperial {
				mi := m / 1609.344
				jc.VisibilityMi = &mi
			}
		}

		if c.Astronomy != nil {
			ja := &jsonAstronomy{
				SolarNoon:           c.Astronomy.SolarNoon.Format(time.RFC3339),
				DayLengthSec:        int64(c.Astronomy.DayLength.Seconds()),
				DayLength:           fmt.Sprintf("%dh %dm", int(c.Astronomy.DayLength.Hours()), int(c.Astronomy.DayLength.Minutes())%60),
				IsPolarDay:          c.Astronomy.IsPolarDay,
				IsPolarNight:        c.Astronomy.IsPolarNight,
				SolarElevationDeg:   c.Astronomy.SolarElevationDeg,
				SolarAzimuthDeg:     c.Astronomy.SolarAzimuthDeg,
				CurrentPeriod:       c.Astronomy.CurrentPeriod,
				MoonPhase:           c.Astronomy.MoonPhase,
				MoonPhaseIcon:       c.Astronomy.MoonPhaseIcon,
				MoonIlluminationPct: c.Astronomy.MoonIlluminationPct,
				MoonAgeDays:         c.Astronomy.MoonAgeDays,
			}
			if c.Astronomy.Sunrise != nil {
				sr := c.Astronomy.Sunrise.Format(time.RFC3339)
				ja.Sunrise = &sr
			}
			if c.Astronomy.Sunset != nil {
				ss := c.Astronomy.Sunset.Format(time.RFC3339)
				ja.Sunset = &ss
			}
			if c.Astronomy.CivilDawn != nil {
				tStr := c.Astronomy.CivilDawn.Format(time.RFC3339)
				ja.CivilDawn = &tStr
			}
			if c.Astronomy.CivilDusk != nil {
				tStr := c.Astronomy.CivilDusk.Format(time.RFC3339)
				ja.CivilDusk = &tStr
			}
			if c.Astronomy.NauticalDawn != nil {
				tStr := c.Astronomy.NauticalDawn.Format(time.RFC3339)
				ja.NauticalDawn = &tStr
			}
			if c.Astronomy.NauticalDusk != nil {
				tStr := c.Astronomy.NauticalDusk.Format(time.RFC3339)
				ja.NauticalDusk = &tStr
			}
			if c.Astronomy.AstroDawn != nil {
				tStr := c.Astronomy.AstroDawn.Format(time.RFC3339)
				ja.AstroDawn = &tStr
			}
			if c.Astronomy.AstroDusk != nil {
				tStr := c.Astronomy.AstroDusk.Format(time.RFC3339)
				ja.AstroDusk = &tStr
			}
			if c.Astronomy.GoldenHourMorningStart != nil {
				tStr := c.Astronomy.GoldenHourMorningStart.Format(time.RFC3339)
				ja.GoldenHourMorningStart = &tStr
			}
			if c.Astronomy.GoldenHourMorningEnd != nil {
				tStr := c.Astronomy.GoldenHourMorningEnd.Format(time.RFC3339)
				ja.GoldenHourMorningEnd = &tStr
			}
			if c.Astronomy.GoldenHourEveningStart != nil {
				tStr := c.Astronomy.GoldenHourEveningStart.Format(time.RFC3339)
				ja.GoldenHourEveningStart = &tStr
			}
			if c.Astronomy.GoldenHourEveningEnd != nil {
				tStr := c.Astronomy.GoldenHourEveningEnd.Format(time.RFC3339)
				ja.GoldenHourEveningEnd = &tStr
			}
			jc.Astronomy = ja
			out.Astronomy = ja
		}

		if c.AirQuality != nil {
			var healthAdv, smokeAdv string
			if c.AirQuality.AQI != nil {
				healthAdv = models.EPAHealthAdvisory(*c.AirQuality.AQI)
			}
			if c.AirQuality.PM25 != nil {
				smokeAdv = models.SmokeAdvisory(*c.AirQuality.PM25)
			}
			jaq := &jsonAirQuality{
				AQI:            c.AirQuality.AQI,
				Category:       c.AirQuality.Category,
				UVIndex:        c.AirQuality.UVIndex,
				UVCategory:     c.AirQuality.UVCategory,
				PM25:           c.AirQuality.PM25,
				PM10:           c.AirQuality.PM10,
				Ozone:          c.AirQuality.O3,
				NO2:            c.AirQuality.NO2,
				CO:             c.AirQuality.CO,
				SO2:            c.AirQuality.SO2,
				HealthAdvisory: healthAdv,
				SmokeAdvisory:  smokeAdv,
			}
			jc.AirQuality = jaq
			out.AirQuality = jaq
		}

		if c.Nowcast != nil {
			jc.Nowcast = c.Nowcast
			out.Nowcast = c.Nowcast
		}

		out.Conditions = jc
	}

	if data.Forecast != nil {
		jf := &jsonForecast{
			GeneratedAt: data.Forecast.GeneratedAt.Format(time.RFC3339),
			Periods:     jsonPeriods(data.Forecast.Periods, imperial),
		}
		if hourly := jsonPeriods(capHourly(data.Forecast.Hourly, opts.HourlyLimit), imperial); len(hourly) > 0 {
			jf.Hourly = hourly
		}
		out.Forecast = jf
	}

	if len(data.Alerts) > 0 {
		alerts := make([]jsonAlert, 0, len(data.Alerts))
		for _, a := range data.Alerts {
			alerts = append(alerts, jsonAlert{
				Event:       a.Event,
				Headline:    a.Headline,
				Severity:    a.Severity,
				Urgency:     a.Urgency,
				Effective:   a.Effective.Format(time.RFC3339),
				Expires:     a.Expires.Format(time.RFC3339),
				AreaDesc:    a.AreaDesc,
				Description: a.Description,
				Instruction: a.Instruction,
			})
		}
		out.Alerts = alerts
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
