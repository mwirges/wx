package location

import (
	"testing"
)

func TestIsZip(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"64101", true},
		{"90210", true},
		{"00501", true},
		{"64101-1234", true},
		{"Kansas City, MO", false},
		{"Chicago", false},
		{"641", false},
		{"641010", false},
		{"6410a", false},
		{"", false},
	}

	for _, tc := range cases {
		got := isZip(tc.input)
		if got != tc.want {
			t.Errorf("isZip(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
