package output

import (
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func TestFormatShort_Imperial(t *testing.T) {
	tempC := 18.0 // 64.4°F -> 64°F
	windKPH := 0.0
	deg := 0.0
	hum := 56.0
	sr := time.Date(2026, 9, 26, 7, 32, 0, 0, time.Local)
	ss := time.Date(2026, 9, 26, 19, 32, 0, 0, time.Local)

	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location:    "Fort Wayne, IN",
			TempC:       &tempC,
			Description: "Clear",
			WindKPH:     &windKPH,
			WindDegrees: &deg,
			HumidityPct: &hum,
			Astronomy: &models.Astronomy{
				Sunrise: &sr,
				Sunset:  &ss,
			},
		},
	}

	opts := RenderOptions{Units: "imperial"}
	got := FormatShort(data, opts)

	if !strings.Contains(got, "Fort Wayne, IN: 64°F Clear") {
		t.Errorf("got %q, want prefix 'Fort Wayne, IN: 64°F Clear'", got)
	}
	if !strings.Contains(got, "N 0 mph") {
		t.Errorf("got %q, want 'N 0 mph'", got)
	}
	if !strings.Contains(got, "56% hum") {
		t.Errorf("got %q, want '56%% hum'", got)
	}
	if !strings.Contains(got, "↑7:32 AM ↓7:32 PM") {
		t.Errorf("got %q, want '↑7:32 AM ↓7:32 PM'", got)
	}
}

func TestFormatShort_Metric(t *testing.T) {
	tempC := 18.0
	windKPH := 15.0
	deg := 180.0
	hum := 70.0

	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location:    "Munich, DE",
			TempC:       &tempC,
			Description: "Cloudy",
			WindKPH:     &windKPH,
			WindDegrees: &deg,
			HumidityPct: &hum,
		},
	}

	opts := RenderOptions{Units: "metric"}
	got := FormatShort(data, opts)

	if !strings.Contains(got, "Munich, DE: 18.0°C Cloudy") {
		t.Errorf("got %q, want 'Munich, DE: 18.0°C Cloudy'", got)
	}
	if !strings.Contains(got, "S 15 km/h") {
		t.Errorf("got %q, want 'S 15 km/h'", got)
	}
	if !strings.Contains(got, "70% hum") {
		t.Errorf("got %q, want '70%% hum'", got)
	}
}

func TestFormatShort_FeelsLike(t *testing.T) {
	tempC := 35.0 // 95°F
	heatIndex := 40.0 // 104°F
	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location:   "Austin, TX",
			TempC:      &tempC,
			HeatIndexC: &heatIndex,
		},
	}

	opts := RenderOptions{Units: "imperial"}
	got := FormatShort(data, opts)

	if !strings.Contains(got, "95°F (feels 104°)") {
		t.Errorf("got %q, expected '95°F (feels 104°)'", got)
	}
}

func TestFormatShort_WindGusts(t *testing.T) {
	tempC := 20.0
	windKph := 30.0
	gustKph := 50.0
	deg := 270.0

	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location:    "Chicago, IL",
			TempC:       &tempC,
			WindKPH:     &windKph,
			WindGustKPH: &gustKph,
			WindDegrees: &deg,
		},
	}

	opts := RenderOptions{Units: "imperial"}
	got := FormatShort(data, opts)

	if !strings.Contains(got, "W 19 mph (gusts 31 mph)") {
		t.Errorf("got %q, expected gusts in wind description", got)
	}
}

func TestFormatShort_AstronomyPolar(t *testing.T) {
	tempC := -10.0
	dataDay := RenderData{
		Conditions: &models.CurrentConditions{
			Location:  "Tromso, NO",
			TempC:     &tempC,
			Astronomy: &models.Astronomy{IsPolarDay: true},
		},
	}
	gotDay := FormatShort(dataDay, RenderOptions{Units: "metric"})
	if !strings.Contains(gotDay, "24h sun") {
		t.Errorf("got %q, expected '24h sun'", gotDay)
	}

	dataNight := RenderData{
		Conditions: &models.CurrentConditions{
			Location:  "Tromso, NO",
			TempC:     &tempC,
			Astronomy: &models.Astronomy{IsPolarNight: true},
		},
	}
	gotNight := FormatShort(dataNight, RenderOptions{Units: "metric"})
	if !strings.Contains(gotNight, "polar night") {
		t.Errorf("got %q, expected 'polar night'", gotNight)
	}
}

func TestFormatShort_Alerts(t *testing.T) {
	tempC := 22.0
	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location: "Miami, FL",
			TempC:    &tempC,
		},
		Alerts: []models.Alert{
			{Event: "Flood Warning"},
			{Event: "Wind Advisory"},
		},
	}

	got := FormatShort(data, RenderOptions{Units: "imperial"})
	if !strings.Contains(got, "⚠️ 2 alerts") {
		t.Errorf("got %q, expected '⚠️ 2 alerts'", got)
	}

	// Single alert
	data.Alerts = []models.Alert{{Event: "Tornado Warning"}}
	gotSingle := FormatShort(data, RenderOptions{Units: "imperial"})
	if !strings.Contains(gotSingle, "⚠️ Tornado Warning") {
		t.Errorf("got %q, expected '⚠️ Tornado Warning'", gotSingle)
	}
}

func TestFormatShort_ForecastPrecip(t *testing.T) {
	tempC := 20.0
	pop := 60.0
	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location: "Seattle, WA",
			TempC:    &tempC,
		},
		Forecast: &models.Forecast{
			Periods: []models.Period{
				{ProbabilityOfPrecipitation: &pop},
			},
		},
	}

	got := FormatShort(data, RenderOptions{Units: "imperial"})
	if !strings.Contains(got, "60% precip") {
		t.Errorf("got %q, expected '60%% precip'", got)
	}
}

func TestFormatShort_ForecastOnlyFallback(t *testing.T) {
	pop := 40.0
	data := RenderData{
		Forecast: &models.Forecast{
			Periods: []models.Period{
				{
					Name:                       "Tonight",
					TempC:                      10.0,
					ShortDesc:                  "Showers",
					WindKPH:                    15.0,
					ProbabilityOfPrecipitation: &pop,
				},
			},
		},
	}

	got := FormatShort(data, RenderOptions{Units: "imperial"})
	if !strings.Contains(got, "Tonight: 50°F Showers") {
		t.Errorf("got %q, expected 'Tonight: 50°F Showers'", got)
	}
	if !strings.Contains(got, "40% precip") {
		t.Errorf("got %q, expected '40%% precip'", got)
	}
}

func TestRenderShort_EmptyError(t *testing.T) {
	err := renderShort(RenderData{}, RenderOptions{})
	if err == nil {
		t.Errorf("expected error for empty render data, got nil")
	}
}
