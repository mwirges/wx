package nws

import (
	"testing"

	"github.com/mwirges/wx/internal/location"
)

func TestProviderName(t *testing.T) {
	p := New()
	if p.Name() != "nws" {
		t.Errorf("Name() = %q, want %q", p.Name(), "nws")
	}
}

func TestProviderSupports(t *testing.T) {
	p := New()

	cases := []struct {
		loc  location.Location
		want bool
	}{
		{location.Location{CountryCode: "US"}, true},
		{location.Location{CountryCode: "us"}, false}, // must be uppercase
		{location.Location{CountryCode: "CA"}, false},
		{location.Location{CountryCode: "GB"}, false},
		{location.Location{CountryCode: ""}, false},
	}

	for _, tc := range cases {
		got := p.Supports(tc.loc)
		if got != tc.want {
			t.Errorf("Supports(%q) = %v, want %v", tc.loc.CountryCode, got, tc.want)
		}
	}
}
