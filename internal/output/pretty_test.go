package output

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/mwirges/wx/internal/models"
)

func TestFormatTemp_Imperial(t *testing.T) {
	cases := []struct {
		tempC float64
		want  string
	}{
		{0, "32°F"},
		{100, "212°F"},
		{-40, "-40°F"},
		{20, "68°F"},
	}
	for _, tc := range cases {
		got := FormatTemp(tc.tempC, true)
		if got != tc.want {
			t.Errorf("FormatTemp(%.1f, imperial) = %q, want %q", tc.tempC, got, tc.want)
		}
	}
}

func TestFormatTemp_Metric(t *testing.T) {
	cases := []struct {
		tempC float64
		want  string
	}{
		{0, "0.0°C"},
		{100, "100.0°C"},
		{-40, "-40.0°C"},
		{22.5, "22.5°C"},
	}
	for _, tc := range cases {
		got := FormatTemp(tc.tempC, false)
		if got != tc.want {
			t.Errorf("FormatTemp(%.1f, metric) = %q, want %q", tc.tempC, got, tc.want)
		}
	}
}

func TestDegreesToCompass(t *testing.T) {
	cases := []struct {
		deg  float64
		want string
	}{
		{0, "N"},
		{360, "N"},
		{45, "NE"},
		{90, "E"},
		{135, "SE"},
		{180, "S"},
		{225, "SW"},
		{270, "W"},
		{315, "NW"},
		{337.5, "NNW"},
		{22.5, "NNE"},
	}
	for _, tc := range cases {
		got := DegreesToCompass(tc.deg)
		if got != tc.want {
			t.Errorf("DegreesToCompass(%.1f) = %q, want %q", tc.deg, got, tc.want)
		}
	}
}

func TestTempStyle_Thresholds(t *testing.T) {
	cases := []struct {
		tempC    float64
		imperial bool
		want     lipgloss.Style
	}{
		{-10, true, styleTempCold}, // <32°F
		{10, true, styleTempMild},  // ~50°F
		{22, true, styleTempWarm},  // ~72°F
		{35, true, styleTempHot},   // ~95°F
		{-5, false, styleTempCold}, // <0°C
		{10, false, styleTempMild}, // 0–18°C
		{25, false, styleTempWarm}, // 18–29°C
		{35, false, styleTempHot},  // >29°C
	}

	for _, tc := range cases {
		got := TempStyle(tc.tempC, tc.imperial)
		if got.GetForeground() != tc.want.GetForeground() {
			t.Errorf("TempStyle(%.1f°C, imperial=%v): got foreground %v, want %v",
				tc.tempC, tc.imperial, got.GetForeground(), tc.want.GetForeground())
		}
	}
}

func TestCelsiusToFahrenheit(t *testing.T) {
	cases := []struct{ c, wantF float64 }{
		{0, 32},
		{100, 212},
		{-40, -40},
		{37, 98.6},
	}
	for _, tc := range cases {
		got := CelsiusToFahrenheit(tc.c)
		if got < tc.wantF-0.01 || got > tc.wantF+0.01 {
			t.Errorf("CelsiusToFahrenheit(%.1f) = %.4f, want %.4f", tc.c, got, tc.wantF)
		}
	}
}

func TestKPHToMPH(t *testing.T) {
	cases := []struct{ kph, wantMPH float64 }{
		{0, 0},
		{1.60934, 1},
		{96.5604, 60},
	}
	for _, tc := range cases {
		got := KphToMPH(tc.kph)
		if got < tc.wantMPH-0.01 || got > tc.wantMPH+0.01 {
			t.Errorf("KphToMPH(%.4f) = %.4f, want %.4f", tc.kph, got, tc.wantMPH)
		}
	}
}

func TestFormatWindSpeed(t *testing.T) {
	cases := []struct {
		kph      float64
		imperial bool
		want     string
	}{
		{16.0934, true, "10 mph"},
		{16.0934, false, "16 km/h"},
		{0, false, "0 km/h"},
	}
	for _, tc := range cases {
		got := FormatWindSpeed(tc.kph, tc.imperial)
		if got != tc.want {
			t.Errorf("FormatWindSpeed(%.4f, %v) = %q, want %q", tc.kph, tc.imperial, got, tc.want)
		}
	}
}

func TestFormatPressure(t *testing.T) {
	cases := []struct {
		hpa      float64
		imperial bool
		want     string
	}{
		{1013.25, true, "29.92 inHg"},
		{1013.25, false, "1013 hPa"},
		{1020.0, false, "1020 hPa"},
	}
	for _, tc := range cases {
		got := FormatPressure(tc.hpa, tc.imperial)
		if got != tc.want {
			t.Errorf("FormatPressure(%.2f, %v) = %q, want %q", tc.hpa, tc.imperial, got, tc.want)
		}
	}
}

func TestFormatVisibility(t *testing.T) {
	cases := []struct {
		meters   float64
		imperial bool
		want     string
	}{
		{16093.44, true, "10 mi"},
		{8046.72, true, "5.0 mi"},
		{16000.0, false, "16 km"},
		{5000.0, false, "5.0 km"},
	}
	for _, tc := range cases {
		got := FormatVisibility(tc.meters, tc.imperial)
		if got != tc.want {
			t.Errorf("FormatVisibility(%.2f, %v) = %q, want %q", tc.meters, tc.imperial, got, tc.want)
		}
	}
}

func TestFeelsLikeTemp(t *testing.T) {
	wc := -5.0
	hi := 38.0

	if got := FeelsLikeTemp(&wc, nil); got != &wc {
		t.Error("expected wind chill pointer when heat index is nil")
	}
	if got := FeelsLikeTemp(nil, &hi); got != &hi {
		t.Error("expected heat index pointer when wind chill is nil")
	}
	if got := FeelsLikeTemp(nil, nil); got != nil {
		t.Errorf("expected nil when both nil, got %v", got)
	}
	// wind chill takes precedence
	if got := FeelsLikeTemp(&wc, &hi); got != &wc {
		t.Error("expected wind chill to take precedence over heat index")
	}
}

func TestGetIcon(t *testing.T) {
	// Known code returns a non-blank first line.
	ic := GetIcon("clear-day")
	if ic.Lines[0] == `           ` {
		t.Error("clear-day icon line 0 should not be blank")
	}
	// Unknown code returns blank icon without panic.
	blank := GetIcon("nonexistent-condition")
	for i, l := range blank.Lines {
		if l != `           ` {
			t.Errorf("blank icon line %d = %q, want all spaces", i, l)
		}
	}
}

func TestFormatWind(t *testing.T) {
	deg270 := 270.0

	cases := []struct {
		kph      float64
		degrees  *float64
		imperial bool
		want     string
	}{
		{16.0934, &deg270, true, "W 10 mph"},
		{16.0934, &deg270, false, "W 16 km/h"},
		{16.0934, nil, true, "10 mph"},
		{0, nil, false, "0 km/h"},
	}
	for _, tc := range cases {
		got := FormatWind(tc.kph, tc.degrees, tc.imperial)
		if got != tc.want {
			t.Errorf("FormatWind(%.4f, ..., %v) = %q, want %q", tc.kph, tc.imperial, got, tc.want)
		}
	}
}

func TestRenderPretty_Hourly(t *testing.T) {
	pop := 30.0
	data := RenderData{
		Forecast: &models.Forecast{
			GeneratedAt: time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC),
			Periods: []models.Period{
				{
					Name:                       "Mon 3 PM",
					StartTime:                  time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC),
					TempC:                      18.0,
					WindKPH:                    16.0,
					WindDir:                    "NW",
					ShortDesc:                  "Partly Cloudy",
					ProbabilityOfPrecipitation: &pop,
				},
			},
		},
	}
	opts := RenderOptions{
		ShowForecast: true,
		ShowHourly:   true,
		Units:        "imperial",
	}

	out := captureStdout(t, func() {
		if err := renderPretty(data, opts); err != nil {
			t.Errorf("renderPretty: %v", err)
		}
	})

	outStr := string(out)
	if !strings.Contains(outStr, "Hourly Forecast") {
		t.Errorf("output does not contain 'Hourly Forecast': %s", outStr)
	}
	if !strings.Contains(outStr, "Mon 3 PM") {
		t.Errorf("output does not contain 'Mon 3 PM': %s", outStr)
	}
	if !strings.Contains(outStr, "30%") {
		t.Errorf("output does not contain '30%%': %s", outStr)
	}
}

func TestRenderPretty_Astronomy(t *testing.T) {
	sr := time.Date(2026, 9, 26, 7, 31, 0, 0, time.Local)
	ss := time.Date(2026, 9, 26, 19, 32, 0, 0, time.Local)
	temp := 15.0

	t.Run("Standard Sunrise Sunset", func(t *testing.T) {
		data := RenderData{
			Conditions: &models.CurrentConditions{
				Location:      "Fort Wayne, IN",
				ObservedAt:    time.Date(2026, 9, 26, 12, 0, 0, 0, time.Local),
				TempC:         &temp,
				ConditionCode: "clear-day",
				Astronomy: &models.Astronomy{
					Sunrise:   &sr,
					Sunset:    &ss,
					DayLength: 12*time.Hour + 1*time.Minute,
				},
			},
		}
		out := captureStdout(t, func() {
			renderPretty(data, RenderOptions{Units: "imperial"})
		})
		outStr := string(out)
		if !strings.Contains(outStr, "Sun:") {
			t.Errorf("expected 'Sun:', got %q", outStr)
		}
		if !strings.Contains(outStr, "7:31 AM") {
			t.Errorf("expected '7:31 AM', got %q", outStr)
		}
		if !strings.Contains(outStr, "7:32 PM") {
			t.Errorf("expected '7:32 PM', got %q", outStr)
		}
		if !strings.Contains(outStr, "12h 1m daylight") {
			t.Errorf("expected '12h 1m daylight', got %q", outStr)
		}
	})

	t.Run("Polar Day", func(t *testing.T) {
		data := RenderData{
			Conditions: &models.CurrentConditions{
				Location:      "Barrow, AK",
				ObservedAt:    time.Date(2026, 6, 21, 12, 0, 0, 0, time.Local),
				TempC:         &temp,
				ConditionCode: "clear-day",
				Astronomy: &models.Astronomy{
					IsPolarDay: true,
					DayLength:  24 * time.Hour,
				},
			},
		}
		out := captureStdout(t, func() {
			renderPretty(data, RenderOptions{Units: "imperial"})
		})
		outStr := string(out)
		if !strings.Contains(outStr, "Polar Day") {
			t.Errorf("expected 'Polar Day', got %q", outStr)
		}
	})

	t.Run("Polar Night", func(t *testing.T) {
		data := RenderData{
			Conditions: &models.CurrentConditions{
				Location:      "Barrow, AK",
				ObservedAt:    time.Date(2026, 12, 21, 12, 0, 0, 0, time.Local),
				TempC:         &temp,
				ConditionCode: "clear-night",
				Astronomy: &models.Astronomy{
					IsPolarNight: true,
					DayLength:    0,
				},
			},
		}
		out := captureStdout(t, func() {
			renderPretty(data, RenderOptions{Units: "imperial"})
		})
		outStr := string(out)
		if !strings.Contains(outStr, "Polar Night") {
			t.Errorf("expected 'Polar Night', got %q", outStr)
		}
	})

	t.Run("Moon Phase", func(t *testing.T) {
		illum := 85.0
		age := 12.4
		data := RenderData{
			Conditions: &models.CurrentConditions{
				Location:      "Fort Wayne, IN",
				ObservedAt:    time.Date(2026, 9, 26, 12, 0, 0, 0, time.Local),
				TempC:         &temp,
				ConditionCode: "clear-day",
				Astronomy: &models.Astronomy{
					Sunrise:             &sr,
					Sunset:              &ss,
					DayLength:           12*time.Hour + 1*time.Minute,
					MoonPhase:           "Waxing Gibbous",
					MoonPhaseIcon:       "🌔",
					MoonIlluminationPct: &illum,
					MoonAgeDays:         &age,
				},
			},
		}
		out := captureStdout(t, func() {
			renderPretty(data, RenderOptions{Units: "imperial"})
		})
		outStr := string(out)
		if !strings.Contains(outStr, "Moon:") {
			t.Errorf("expected 'Moon:', got %q", outStr)
		}
		if !strings.Contains(outStr, "🌔 Waxing Gibbous") {
			t.Errorf("expected '🌔 Waxing Gibbous', got %q", outStr)
		}
		if !strings.Contains(outStr, "85% illuminated") {
			t.Errorf("expected '85%% illuminated', got %q", outStr)
		}
		if !strings.Contains(outStr, "12.4d age") {
			t.Errorf("expected '12.4d age', got %q", outStr)
		}
	})
}

func TestRenderPretty_HourlyForecastTable(t *testing.T) {
	pop := 40.0
	humid := 65.0
	periods := []models.Period{
		{
			Name:                       "10 PM",
			TempC:                      15.0,
			ProbabilityOfPrecipitation: &pop,
			HumidityPct:                &humid,
			WindKPH:                    16.0,
			WindDir:                    "NW",
			ShortDesc:                  "Scattered Showers",
		},
		{
			Name:      "11 PM",
			TempC:     14.0,
			WindKPH:   8.0,
			ShortDesc: "Mostly Clear",
		},
		{
			Name:      "12 AM",
			TempC:     13.0,
			WindKPH:   5.0,
			ShortDesc: "Clear",
		},
	}

	data := RenderData{
		Forecast: &models.Forecast{
			Periods: periods,
		},
	}

	// 1. With ShowHourly
	out := captureStdout(t, func() {
		renderPretty(data, RenderOptions{
			Units:       "imperial",
			ShowHourly:  true,
			HourlyLimit: 2,
		})
	})
	outStr := string(out)

	// Check table headers
	if !strings.Contains(outStr, "TIME") || !strings.Contains(outStr, "TEMP") ||
		!strings.Contains(outStr, "PRECIP") || !strings.Contains(outStr, "HUMID") ||
		!strings.Contains(outStr, "WIND") || !strings.Contains(outStr, "FORECAST") {
		t.Errorf("expected hourly table column headers, got:\n%s", outStr)
	}

	// Check data
	if !strings.Contains(outStr, "10 PM") || !strings.Contains(outStr, "40%") || !strings.Contains(outStr, "65%") {
		t.Errorf("expected 10 PM row with pop and humidity, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "11 PM") {
		t.Errorf("expected 11 PM row, got:\n%s", outStr)
	}
	// Limit was 2, so 12 AM should NOT be rendered
	if strings.Contains(outStr, "12 AM") {
		t.Errorf("expected 12 AM to be excluded by HourlyLimit=2, got:\n%s", outStr)
	}
}

func TestRenderPretty_AlertSeverity(t *testing.T) {
	data := RenderData{
		Alerts: []models.Alert{
			{Event: "Tornado Warning", Severity: "Extreme", Headline: "Tornado warning in effect"},
			{Event: "Flood Watch", Severity: "Moderate", Headline: "Flood watch in effect"},
			{Event: "Wind Advisory", Severity: "Minor", Headline: "Wind advisory in effect"},
		},
	}

	out := captureStdout(t, func() {
		renderPretty(data, RenderOptions{Units: "imperial", ShowAlerts: true})
	})
	outStr := string(out)

	if !strings.Contains(outStr, "Tornado warning in effect") {
		t.Errorf("expected Tornado Warning headline, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "Flood watch in effect") {
		t.Errorf("expected Flood Watch headline, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "Wind advisory in effect") {
		t.Errorf("expected Wind Advisory headline, got:\n%s", outStr)
	}
}

func TestRenderPretty_ObservedAtAge(t *testing.T) {
	temp := 20.0
	// 15 minutes ago
	observed := time.Now().Add(-15 * time.Minute)
	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location:      "Fort Wayne, IN",
			ObservedAt:    observed,
			TempC:         &temp,
			ConditionCode: "clear-day",
		},
	}

	out := captureStdout(t, func() {
		renderPretty(data, RenderOptions{Units: "imperial"})
	})
	outStr := string(out)

	if !strings.Contains(outStr, "15m ago") {
		t.Errorf("expected '15m ago' in observed header, got:\n%s", outStr)
	}
}

func TestRenderPretty_AirQuality(t *testing.T) {
	temp := 22.0
	aqi := 45
	uv := 2.5
	data := RenderData{
		Conditions: &models.CurrentConditions{
			Location:      "Fort Wayne, IN",
			ObservedAt:    time.Now(),
			TempC:         &temp,
			ConditionCode: "clear-day",
			AirQuality: &models.AirQuality{
				AQI:        &aqi,
				Category:   models.AQICategory(aqi),
				UVIndex:    &uv,
				UVCategory: models.UVCategory(uv),
			},
		},
	}

	out := captureStdout(t, func() {
		renderPretty(data, RenderOptions{Units: "imperial"})
	})
	outStr := string(out)

	if !strings.Contains(outStr, "Air Quality:") || !strings.Contains(outStr, "45 (Good)") {
		t.Errorf("expected Air Quality in pretty output, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "UV Index:") || !strings.Contains(outStr, "2.5 (Low)") {
		t.Errorf("expected UV Index in pretty output, got:\n%s", outStr)
	}
}

func TestRenderPretty_ForecastThenHourly(t *testing.T) {
	data := RenderData{
		Forecast: &models.Forecast{
			Periods: []models.Period{
				{Name: "Tonight", TempC: 12, ShortDesc: "Clear"},
			},
			Hourly: []models.Period{
				{Name: "3 PM", TempC: 22, ShortDesc: "Sunny"},
				{Name: "4 PM", TempC: 21, ShortDesc: "Sunny"},
				{Name: "5 PM", TempC: 19, ShortDesc: "Sunny"},
			},
		},
	}
	out := captureStdout(t, func() {
		if err := renderPretty(data, RenderOptions{Units: "imperial", ShowHourly: true, HourlyLimit: 2}); err != nil {
			t.Fatalf("render: %v", err)
		}
	})
	text := string(out)
	day := strings.Index(text, "Tonight")
	table := strings.Index(text, "Hourly Forecast")
	hour := strings.Index(text, "3 PM")
	if day < 0 || table < 0 || hour < 0 {
		t.Fatalf("missing day list or hourly table:\n%s", text)
	}
	if !(day < table && table < hour) {
		t.Fatalf("want day list then hourly table, got day=%d table=%d hour=%d", day, table, hour)
	}
	if strings.Contains(text, "5 PM") {
		t.Fatalf("hourly table was not capped:\n%s", text)
	}
}
