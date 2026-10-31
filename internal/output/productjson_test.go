package output

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func TestProductJSONMatchesRender(t *testing.T) {
	when := time.Date(2026, 10, 31, 15, 4, 5, 0, time.UTC)
	pop := 40.0
	data := RenderData{
		Conditions: &models.CurrentConditions{
			StationID:   "KFWA",
			ObservedAt:  when,
			Location:    "Fort Wayne, IN",
			Description: "Cloudy",
			TempC:       floatPtr(10),
			HumidityPct: floatPtr(55),
		},
		Forecast: &models.Forecast{
			GeneratedAt: when,
			Periods: []models.Period{
				{Name: "Tonight", StartTime: when, TempC: 8, ShortDesc: "Cloudy", ProbabilityOfPrecipitation: &pop},
				{Name: "Saturday", StartTime: when.Add(12 * time.Hour), IsDaytime: true, TempC: 14, ShortDesc: "Sun"},
			},
		},
		Alerts: []models.Alert{{
			Event: "Frost Advisory", Severity: "Minor", Effective: when, Expires: when.Add(time.Hour),
		}},
	}
	rendered := captureStdout(t, func() {
		if err := renderJSON(data, RenderOptions{Units: "imperial"}); err != nil {
			t.Fatal(err)
		}
	})
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(rendered, &envelope); err != nil {
		t.Fatal(err)
	}
	cond, err := ConditionsJSON(data.Conditions, "imperial")
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, cond, envelope["conditions"]) {
		t.Fatalf("conditions\n got %s\nwant %s", cond, envelope["conditions"])
	}
	fc, err := ForecastJSON(data.Forecast, "imperial")
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, fc, envelope["forecast"]) {
		t.Fatalf("forecast\n got %s\nwant %s", fc, envelope["forecast"])
	}
	alerts, err := AlertsJSON(data.Alerts)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, alerts, envelope["alerts"]) {
		t.Fatalf("alerts\n got %s\nwant %s", alerts, envelope["alerts"])
	}
	empty, err := AlertsJSON(nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(empty) != "[]" {
		t.Fatalf("empty alerts = %s", empty)
	}
}

func TestHourlyJSONCaps(t *testing.T) {
	when := time.Date(2026, 10, 31, 15, 0, 0, 0, time.UTC)
	periods := []models.Period{
		{Name: "3 PM", StartTime: when, TempC: 20, ShortDesc: "A"},
		{Name: "4 PM", StartTime: when.Add(time.Hour), TempC: 19, ShortDesc: "B"},
		{Name: "5 PM", StartTime: when.Add(2 * time.Hour), TempC: 18, ShortDesc: "C"},
	}
	raw, err := HourlyJSON(when, periods, "imperial", 2)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Periods []struct {
			Name string `json:"name"`
		} `json:"periods"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Periods) != 2 || got.Periods[0].Name != "3 PM" || got.Periods[1].Name != "4 PM" {
		t.Fatalf("periods = %+v", got.Periods)
	}
}

func floatPtr(v float64) *float64 { return &v }

func jsonEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var ja, jb any
	if err := json.Unmarshal(a, &ja); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &jb); err != nil {
		t.Fatal(err)
	}
	ab, _ := json.Marshal(ja)
	bb, _ := json.Marshal(jb)
	return string(ab) == string(bb)
}
