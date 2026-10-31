package output

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/mwirges/wx/internal/models"
)

// ConditionsJSON is the conditions object from `wx --json`.
func ConditionsJSON(c *models.CurrentConditions, units string) ([]byte, error) {
	if c == nil {
		return nil, errors.New("nil conditions")
	}
	out := buildJSONOutput(RenderData{Conditions: c}, RenderOptions{Units: units})
	if out.Conditions == nil {
		return nil, errors.New("nil conditions")
	}
	return json.Marshal(out.Conditions)
}

// ForecastJSON is the forecast object from `wx --forecast --json` (daily periods).
func ForecastJSON(f *models.Forecast, units string) ([]byte, error) {
	if f == nil {
		return nil, errors.New("nil forecast")
	}
	payload := jsonForecast{
		GeneratedAt: f.GeneratedAt.Format(time.RFC3339),
		Periods:     jsonPeriods(f.Periods, units != "metric"),
	}
	if payload.Periods == nil {
		payload.Periods = []jsonPeriod{}
	}
	return json.Marshal(payload)
}

// HourlyJSON is the hourly period list in the same shape as forecast.periods
// from `wx hourly --json`, capped to hours (default 24, max 168).
func HourlyJSON(generated time.Time, periods []models.Period, units string, hours int) ([]byte, error) {
	if hours <= 0 {
		hours = 24
	}
	if hours > 168 {
		hours = 168
	}
	if len(periods) > hours {
		periods = periods[:hours]
	}
	payload := jsonForecast{
		GeneratedAt: generated.Format(time.RFC3339),
		Periods:     jsonPeriods(periods, units != "metric"),
	}
	if payload.Periods == nil {
		payload.Periods = []jsonPeriod{}
	}
	return json.Marshal(payload)
}

// AlertsJSON is the alerts array from `wx --alerts --json`.
// An empty or nil slice encodes as [] so callers never see JSON null.
func AlertsJSON(alerts []models.Alert) ([]byte, error) {
	out := make([]jsonAlert, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, jsonAlert{
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
	return json.Marshal(out)
}
