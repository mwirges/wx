package output

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type jsonAstronomy struct {
	Sunrise             *string  `json:"sunrise,omitempty"`
	Sunset              *string  `json:"sunset,omitempty"`
	SolarNoon           string   `json:"solar_noon,omitempty"`
	DayLengthSec        int64    `json:"day_length_seconds"`
	DayLength           string   `json:"day_length"`
	IsPolarDay          bool     `json:"is_polar_day,omitempty"`
	IsPolarNight        bool     `json:"is_polar_night,omitempty"`
	MoonPhase           string   `json:"moon_phase,omitempty"`
	MoonPhaseIcon       string   `json:"moon_phase_icon,omitempty"`
	MoonIlluminationPct *float64 `json:"moon_illumination_pct,omitempty"`
	MoonAgeDays         *float64 `json:"moon_age_days,omitempty"`
}

type jsonConditions struct {
	Station     string   `json:"station"`
	ObservedAt  string   `json:"observed_at"`
	Location    string   `json:"location"`
	Description string   `json:"description,omitempty"`
	ConditionCode string `json:"condition_code,omitempty"`

	TempC       *float64 `json:"temperature_c,omitempty"`
	TempF       *float64 `json:"temperature_f,omitempty"`
	FeelsLikeC  *float64 `json:"feels_like_c,omitempty"`
	FeelsLikeF  *float64 `json:"feels_like_f,omitempty"`

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

	Astronomy *jsonAstronomy `json:"astronomy,omitempty"`
	AirQuality *jsonAirQuality `json:"air_quality,omitempty"`
}

type jsonAirQuality struct {
	AQI        *int     `json:"aqi,omitempty"`
	Category   string   `json:"category,omitempty"`
	UVIndex    *float64 `json:"uv_index,omitempty"`
	UVCategory string   `json:"uv_category,omitempty"`
	PM25       *float64 `json:"pm2_5,omitempty"`
	PM10       *float64 `json:"pm10,omitempty"`
	Ozone      *float64 `json:"ozone,omitempty"`
	NO2        *float64 `json:"no2,omitempty"`
	CO         *float64 `json:"co,omitempty"`
	SO2        *float64 `json:"so2,omitempty"`
}

type jsonPeriod struct {
	Name         string  `json:"name"`
	StartTime    string  `json:"start_time"`
	IsDaytime    bool    `json:"is_daytime"`
	TempF        float64 `json:"temperature_f,omitempty"`
	TempC        float64 `json:"temperature_c,omitempty"`
	WindMPH      float64 `json:"wind_mph,omitempty"`
	WindKPH      float64 `json:"wind_kph,omitempty"`
	WindDir      string  `json:"wind_direction,omitempty"`
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
	Freshness  *jsonFreshness  `json:"freshness,omitempty"`
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
			jc.Astronomy = ja
			out.Astronomy = ja
		}

		if c.AirQuality != nil {
			jaq := &jsonAirQuality{
				AQI:        c.AirQuality.AQI,
				Category:   c.AirQuality.Category,
				UVIndex:    c.AirQuality.UVIndex,
				UVCategory: c.AirQuality.UVCategory,
				PM25:       c.AirQuality.PM25,
				PM10:       c.AirQuality.PM10,
				Ozone:      c.AirQuality.O3,
				NO2:        c.AirQuality.NO2,
				CO:         c.AirQuality.CO,
				SO2:        c.AirQuality.SO2,
			}
			jc.AirQuality = jaq
			out.AirQuality = jaq
		}

		out.Conditions = jc
	}

	if data.Forecast != nil {
		periods := make([]jsonPeriod, 0, len(data.Forecast.Periods))
		for _, p := range data.Forecast.Periods {
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
			periods = append(periods, jp)
		}
		out.Forecast = &jsonForecast{
			GeneratedAt: data.Forecast.GeneratedAt.Format(time.RFC3339),
			Periods:     periods,
		}
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
