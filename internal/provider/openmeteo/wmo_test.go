package openmeteo

import "testing"

func TestWmoToConditionCode(t *testing.T) {
	tests := []struct {
		code  int
		isDay bool
		want  string
	}{
		{0, true, "clear-day"},
		{0, false, "clear-night"},
		{1, true, "clear-day"},
		{1, false, "clear-night"},
		{2, true, "partly-cloudy-day"},
		{2, false, "partly-cloudy-night"},
		{3, true, "cloudy"},
		{45, true, "fog"},
		{48, false, "fog"},
		{51, true, "rain"},
		{61, true, "rain"},
		{65, true, "heavy-rain"},
		{82, true, "heavy-rain"},
		{56, true, "sleet"},
		{67, true, "sleet"},
		{71, true, "snow"},
		{86, true, "snow"},
		{95, true, "thunder"},
		{99, false, "thunder"},
		{999, true, "partly-cloudy-day"},
	}

	for _, tc := range tests {
		got := wmoToConditionCode(tc.code, tc.isDay)
		if got != tc.want {
			t.Errorf("wmoToConditionCode(%d, %v) = %q, want %q", tc.code, tc.isDay, got, tc.want)
		}
	}
}

func TestWmoToDescription(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "Clear"},
		{1, "Mainly Clear"},
		{2, "Partly Cloudy"},
		{3, "Overcast"},
		{45, "Fog"},
		{51, "Light Drizzle"},
		{61, "Slight Rain"},
		{65, "Heavy Rain"},
		{71, "Slight Snow Fall"},
		{95, "Thunderstorm"},
		{999, "Fair"},
	}

	for _, tc := range tests {
		got := wmoToDescription(tc.code)
		if got != tc.want {
			t.Errorf("wmoToDescription(%d) = %q, want %q", tc.code, got, tc.want)
		}
	}
}

func TestDegreesToCompass(t *testing.T) {
	tests := []struct {
		deg  float64
		want string
	}{
		{0, "N"},
		{360, "N"},
		{90, "E"},
		{180, "S"},
		{270, "W"},
		{45, "NE"},
		{135, "SE"},
		{225, "SW"},
		{315, "NW"},
		{-90, "W"},
	}

	for _, tc := range tests {
		got := degreesToCompass(tc.deg)
		if got != tc.want {
			t.Errorf("degreesToCompass(%.1f) = %q, want %q", tc.deg, got, tc.want)
		}
	}
}
