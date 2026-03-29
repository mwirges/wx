package output

import (
	"encoding/json"
	"os"
	"time"
)

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
	ShortDesc    string  `json:"short_description,omitempty"`
	DetailedDesc string  `json:"detailed_description,omitempty"`
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

type jsonOutput struct {
	Conditions *jsonConditions `json:"conditions,omitempty"`
	Forecast   *jsonForecast   `json:"forecast,omitempty"`
	Alerts     []jsonAlert     `json:"alerts,omitempty"`
}

func renderJSON(data RenderData, opts RenderOptions) error {
	imperial := opts.Units != "metric"

	out := jsonOutput{}

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
		if fl := FeelsLikeTemp(c.WindChillC, c.HeatIndexC); fl != nil {
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

		out.Conditions = jc
	}

	if data.Forecast != nil {
		periods := make([]jsonPeriod, 0, len(data.Forecast.Periods))
		for _, p := range data.Forecast.Periods {
			jp := jsonPeriod{
				Name:         p.Name,
				StartTime:    p.StartTime.Format(time.RFC3339),
				IsDaytime:    p.IsDaytime,
				TempC:        p.TempC,
				WindKPH:      p.WindKPH,
				WindDir:      p.WindDir,
				ShortDesc:    p.ShortDesc,
				DetailedDesc: p.DetailedDesc,
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
