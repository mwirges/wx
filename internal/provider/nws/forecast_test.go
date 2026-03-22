package nws

import (
	"math"
	"testing"
)

func TestParseWindKPH(t *testing.T) {
	cases := []struct {
		input   string
		wantMPH float64 // we verify the km/h value equals mphToKPH(wantMPH)
	}{
		{"10 mph", 10},
		{"5 to 15 mph", 15},   // upper bound
		{"0 mph", 0},
		{"Calm", 0},
		{"calm", 0},
		{"", 0},
		{"25 mph", 25},
		{"10 to 20 mph", 20},
	}

	for _, tc := range cases {
		got := parseWindKPH(tc.input)
		want := mphToKPH(tc.wantMPH)
		if math.Abs(got-want) > 0.01 {
			t.Errorf("parseWindKPH(%q) = %.4f, want %.4f", tc.input, got, want)
		}
	}
}

func TestFahrenheitToCelsius(t *testing.T) {
	cases := []struct {
		f, wantC float64
	}{
		{32, 0},
		{212, 100},
		{98.6, 37},
		{-40, -40},
	}
	for _, tc := range cases {
		got := fahrenheitToCelsius(tc.f)
		if math.Abs(got-tc.wantC) > 0.01 {
			t.Errorf("fahrenheitToCelsius(%.1f) = %.4f, want %.4f", tc.f, got, tc.wantC)
		}
	}
}

func TestMPHToKPH(t *testing.T) {
	cases := []struct {
		mph, wantKPH float64
	}{
		{0, 0},
		{1, 1.60934},
		{60, 96.5604},
		{100, 160.934},
	}
	for _, tc := range cases {
		got := mphToKPH(tc.mph)
		if math.Abs(got-tc.wantKPH) > 0.01 {
			t.Errorf("mphToKPH(%.1f) = %.4f, want %.4f", tc.mph, got, tc.wantKPH)
		}
	}
}
