package ui

import "testing"

func TestScaleWordUsesTheMacCoverageName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Local 150 km", "LOCAL"},
		{"200 km", "200 KM"},
		{"Metro 250 km", "METRO"},
		{"Regional 500 km", "REGIONAL"},
		{"Synoptic 1000 km", "SYNOPTIC"},
		{"CONUS 2000 km", "CONUS"},
	}
	for _, tc := range cases {
		if got := scaleWord(tc.in); got != tc.want {
			t.Fatalf("scaleWord(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
