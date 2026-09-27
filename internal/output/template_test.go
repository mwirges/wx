package output

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func TestExecuteTemplate_Basic(t *testing.T) {
	tempC := 18.0
	hum := 55.0
	windKPH := 10.0
	deg := 270.0
	sr := time.Date(2026, 9, 26, 7, 30, 0, 0, time.Local)
	ss := time.Date(2026, 9, 26, 19, 30, 0, 0, time.Local)

	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location:    "Fort Wayne, IN",
			TempC:       &tempC,
			Description: "Partly Cloudy",
			HumidityPct: &hum,
			WindKPH:     &windKPH,
			WindDegrees: &deg,
			Astronomy: &models.Astronomy{
				Sunrise:   &sr,
				Sunset:    &ss,
				DayLength: 12 * time.Hour,
			},
		},
	}

	opts := RenderOptions{Units: "imperial"}
	var buf bytes.Buffer
	tmpl := `{{.Conditions.Location}} | {{.Conditions.TempStr}} | {{.Conditions.Description}} | {{.Conditions.Astronomy.Sunrise}}`
	err := ExecuteTemplate(tmpl, data, opts, &buf)
	if err != nil {
		t.Fatalf("ExecuteTemplate failed: %v", err)
	}

	got := buf.String()
	want := "Fort Wayne, IN | 64°F | Partly Cloudy | 7:30 AM"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExecuteTemplate_Helpers(t *testing.T) {
	tempC := 20.0
	windKph := 16.0934
	deg := 180.0
	obs := time.Now().Add(-15 * time.Minute)

	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location:    "Chicago, IL",
			TempC:       &tempC,
			Description: "breezy",
			WindKPH:     &windKph,
			WindDegrees: &deg,
			ObservedAt:  obs,
		},
	}

	tmpl := `{{upper .Conditions.Description}} - {{cToF .Conditions.TempC | round 0}}°F - {{compass .Conditions.WindDegrees}} - {{age .Conditions.ObservedAt}}`
	var buf bytes.Buffer
	err := ExecuteTemplate(tmpl, data, RenderOptions{Units: "imperial"}, &buf)
	if err != nil {
		t.Fatalf("ExecuteTemplate failed: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "BREEZY") {
		t.Errorf("expected 'BREEZY', got %q", got)
	}
	if !strings.Contains(got, "68°F") {
		t.Errorf("expected '68°F', got %q", got)
	}
	if !strings.Contains(got, "S") {
		t.Errorf("expected compass 'S', got %q", got)
	}
	if !strings.Contains(got, "15m ago") {
		t.Errorf("expected '15m ago', got %q", got)
	}
}

func TestExecuteTemplate_ForecastAndAlerts(t *testing.T) {
	pop := 50.0
	data := RenderData{
		Forecast: &models.Forecast{
			Periods: []models.Period{
				{
					Name:                       "Monday",
					TempC:                      25.0,
					ShortDesc:                  "Showers",
					ProbabilityOfPrecipitation: &pop,
				},
				{
					Name:      "Monday Night",
					TempC:     15.0,
					ShortDesc: "Clear",
				},
			},
		},
		Alerts: []models.Alert{
			{Event: "Wind Advisory", Severity: "Minor"},
		},
	}

	tmpl := `{{range .Forecast.Periods}}{{.Name}}: {{.TempStr}} ({{.PoPStr}}) {{end}}| {{range .Alerts}}{{.Event}}: {{.Severity}}{{end}}`
	var buf bytes.Buffer
	err := ExecuteTemplate(tmpl, data, RenderOptions{Units: "imperial"}, &buf)
	if err != nil {
		t.Fatalf("ExecuteTemplate failed: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "Monday: 77°F (50%)") {
		t.Errorf("expected forecast period with PoP, got %q", got)
	}
	if !strings.Contains(got, "Monday Night: 59°F") {
		t.Errorf("expected Monday Night, got %q", got)
	}
	if !strings.Contains(got, "Wind Advisory: Minor") {
		t.Errorf("expected alert in output, got %q", got)
	}
}

func TestExecuteTemplate_FilePrefix(t *testing.T) {
	dir := t.TempDir()
	tmplPath := filepath.Join(dir, "prompt.tmpl")
	content := `{{.Conditions.Location}} -> {{.Conditions.TempStr}}`
	if err := os.WriteFile(tmplPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write tmpl file: %v", err)
	}

	tempC := 10.0
	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location: "Boston, MA",
			TempC:    &tempC,
		},
	}

	var buf bytes.Buffer
	err := ExecuteTemplate("@"+tmplPath, data, RenderOptions{Units: "imperial"}, &buf)
	if err != nil {
		t.Fatalf("ExecuteTemplate with @file failed: %v", err)
	}

	got := buf.String()
	want := "Boston, MA -> 50°F"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExecuteTemplate_Errors(t *testing.T) {
	t.Run("Invalid Syntax", func(t *testing.T) {
		var buf bytes.Buffer
		err := ExecuteTemplate("{{.Conditions.Missing | }}", RenderData{}, RenderOptions{}, &buf)
		if err == nil {
			t.Errorf("expected template parse error, got nil")
		}
	})

	t.Run("Nonexistent File", func(t *testing.T) {
		var buf bytes.Buffer
		err := ExecuteTemplate("@/nonexistent/template.tmpl", RenderData{}, RenderOptions{}, &buf)
		if err == nil {
			t.Errorf("expected file read error, got nil")
		}
	})
}

func TestExecuteTemplate_FreshnessAndAstronomy(t *testing.T) {
	tempC := 22.0
	sr := time.Date(2026, 9, 26, 7, 0, 0, 0, time.Local)
	ss := time.Date(2026, 9, 26, 19, 0, 0, 0, time.Local)
	obs := time.Now().Add(-10 * time.Minute)

	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location:   "Chicago, IL",
			TempC:      &tempC,
			ObservedAt: obs,
			Astronomy: &models.Astronomy{
				Sunrise:   &sr,
				Sunset:    &ss,
				DayLength: 12 * time.Hour,
			},
		},
	}

	tmpl := `{{.Conditions.Location}}: {{.Conditions.TempStr}} | {{.Astronomy.Sunrise}} | {{.Freshness.AgeString}}`
	var buf bytes.Buffer
	err := ExecuteTemplate(tmpl, data, RenderOptions{Units: "imperial"}, &buf)
	if err != nil {
		t.Fatalf("ExecuteTemplate failed: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "Chicago, IL: 72°F") {
		t.Errorf("expected 'Chicago, IL: 72°F', got %q", got)
	}
	if !strings.Contains(got, "7:00 AM") {
		t.Errorf("expected '7:00 AM', got %q", got)
	}
	if !strings.Contains(got, "10m ago") {
		t.Errorf("expected '10m ago', got %q", got)
	}
}
