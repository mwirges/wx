package output

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
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
		got := formatTemp(tc.tempC, true)
		if got != tc.want {
			t.Errorf("formatTemp(%.1f, imperial) = %q, want %q", tc.tempC, got, tc.want)
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
		got := formatTemp(tc.tempC, false)
		if got != tc.want {
			t.Errorf("formatTemp(%.1f, metric) = %q, want %q", tc.tempC, got, tc.want)
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
		got := degreesToCompass(tc.deg)
		if got != tc.want {
			t.Errorf("degreesToCompass(%.1f) = %q, want %q", tc.deg, got, tc.want)
		}
	}
}

func TestTempStyle_Thresholds(t *testing.T) {
	// Verify each temperature band maps to the correct package-level style
	// by comparing the returned style to the expected style variable.
	cases := []struct {
		tempC    float64
		imperial bool
		want     lipgloss.Style
	}{
		{-10, true, styleTempCold},  // <32°F
		{10, true, styleTempMild},   // ~50°F
		{22, true, styleTempWarm},   // ~72°F
		{35, true, styleTempHot},    // ~95°F
		{-5, false, styleTempCold},  // <0°C
		{10, false, styleTempMild},  // 0–18°C
		{25, false, styleTempWarm},  // 18–29°C
		{35, false, styleTempHot},   // >29°C
	}

	for _, tc := range cases {
		got := tempStyle(tc.tempC, tc.imperial)
		if got.GetForeground() != tc.want.GetForeground() {
			t.Errorf("tempStyle(%.1f°C, imperial=%v): got foreground %v, want %v",
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
		got := celsiusToFahrenheit(tc.c)
		if got < tc.wantF-0.01 || got > tc.wantF+0.01 {
			t.Errorf("celsiusToFahrenheit(%.1f) = %.4f, want %.4f", tc.c, got, tc.wantF)
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
		got := kphToMPH(tc.kph)
		if got < tc.wantMPH-0.01 || got > tc.wantMPH+0.01 {
			t.Errorf("kphToMPH(%.4f) = %.4f, want %.4f", tc.kph, got, tc.wantMPH)
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
		got := formatWindSpeed(tc.kph, tc.imperial)
		if got != tc.want {
			t.Errorf("formatWindSpeed(%.4f, %v) = %q, want %q", tc.kph, tc.imperial, got, tc.want)
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
		got := formatPressure(tc.hpa, tc.imperial)
		if got != tc.want {
			t.Errorf("formatPressure(%.2f, %v) = %q, want %q", tc.hpa, tc.imperial, got, tc.want)
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
		got := formatVisibility(tc.meters, tc.imperial)
		if got != tc.want {
			t.Errorf("formatVisibility(%.2f, %v) = %q, want %q", tc.meters, tc.imperial, got, tc.want)
		}
	}
}

func TestFeelsLikeTemp(t *testing.T) {
	wc := -5.0
	hi := 38.0

	if got := feelsLikeTemp(&wc, nil); got != &wc {
		t.Error("expected wind chill pointer when heat index is nil")
	}
	if got := feelsLikeTemp(nil, &hi); got != &hi {
		t.Error("expected heat index pointer when wind chill is nil")
	}
	if got := feelsLikeTemp(nil, nil); got != nil {
		t.Errorf("expected nil when both nil, got %v", got)
	}
	// wind chill takes precedence
	if got := feelsLikeTemp(&wc, &hi); got != &wc {
		t.Error("expected wind chill to take precedence over heat index")
	}
}

func TestGetIcon(t *testing.T) {
	// Known code returns a non-blank first line.
	ic := getIcon("clear-day")
	if ic.lines[0] == `           ` {
		t.Error("clear-day icon line 0 should not be blank")
	}
	// Unknown code returns blank icon without panic.
	blank := getIcon("nonexistent-condition")
	for i, l := range blank.lines {
		if l != `           ` {
			t.Errorf("blank icon line %d = %q, want all spaces", i, l)
		}
	}
}

func TestFormatWind(t *testing.T) {
	deg270 := 270.0

	cases := []struct {
		kph     float64
		degrees *float64
		imperial bool
		want    string
	}{
		{16.0934, &deg270, true, "W 10 mph"},
		{16.0934, &deg270, false, "W 16 km/h"},
		{16.0934, nil, true, "10 mph"},
		{0, nil, false, "0 km/h"},
	}
	for _, tc := range cases {
		got := formatWind(tc.kph, tc.degrees, tc.imperial)
		if got != tc.want {
			t.Errorf("formatWind(%.4f, ..., %v) = %q, want %q", tc.kph, tc.imperial, got, tc.want)
		}
	}
}
