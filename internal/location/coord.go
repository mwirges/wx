package location

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ErrInvalidLocation is returned when an input is a coordinate attempt that
// fails validation. Callers must not geocode these strings.
var ErrInvalidLocation = errors.New("invalid location")

// invalidLocationError is a coordinate-form failure. Unwrap yields
// ErrInvalidLocation so CLI and MCP share one check.
type invalidLocationError struct {
	input  string
	reason string
}

func (e *invalidLocationError) Error() string {
	return fmt.Sprintf("invalid location %q: %s", e.input, e.reason)
}

func (e *invalidLocationError) Unwrap() error { return ErrInvalidLocation }

func invalidLocation(input, reason string) error {
	return &invalidLocationError{input: input, reason: reason}
}

// parseCoordinates classifies input before ZIP and Nominatim.
//
// handled is true when the string is a coordinate form. On success loc is
// set and err is nil. On failure err unwraps to ErrInvalidLocation and the
// caller must not geocode. handled is false for ZIP and place names.
//
// Accepted form: lat,lon with a comma and optional ASCII whitespace around
// the two tokens. Both tokens are finite floats, lat in [-90,90], lon in
// [-180,180]. "lat lon" (whitespace only) is rejected. A compact pair with
// no whitespace and no digits, such as "foo,bar", is an ambiguous coordinate
// attempt, not "City, ST" (which has whitespace). "Chicago, IL" and
// "Fort Wayne, IN" still geocode.
func parseCoordinates(input string) (Location, bool, error) {
	raw := trimASCII(input)
	if strings.Contains(raw, ",") {
		return parseCommaCoords(input, raw)
	}
	if err := rejectSpacedPair(input, raw); err != nil {
		return Location{}, true, err
	}
	return Location{}, false, nil
}

func parseCommaCoords(original, raw string) (Location, bool, error) {
	parts := strings.Split(raw, ",")
	trimmed := make([]string, len(parts))
	allNumeric := true
	for i, p := range parts {
		trimmed[i] = trimASCII(p)
		if !looksNumeric(trimmed[i]) {
			allNumeric = false
		}
	}
	if len(parts) != 2 {
		if allNumeric {
			return Location{}, true, invalidLocation(original, "lat,lon takes exactly two numbers")
		}
		return Location{}, false, nil
	}
	a, b := trimmed[0], trimmed[1]
	if looksNumeric(a) || looksNumeric(b) {
		lat, lon, err := parsePair(a, b)
		if err != nil {
			return Location{}, true, invalidLocation(original, err.Error())
		}
		return Location{
			Lat:         lat,
			Lon:         lon,
			DisplayName: formatCoord(lat, lon),
		}, true, nil
	}
	// "foo,bar" has no whitespace. "City, ST" and "Paris, France" do.
	if !strings.ContainsAny(raw, " \t\n\r\v\f") {
		return Location{}, true, invalidLocation(original, "not a lat,lon pair")
	}
	return Location{}, false, nil
}

func rejectSpacedPair(original, raw string) error {
	fields := strings.FieldsFunc(raw, isASCIISpace)
	if len(fields) != 2 {
		return nil
	}
	if looksNumeric(fields[0]) && looksNumeric(fields[1]) && fields[0] != "" && fields[1] != "" {
		return invalidLocation(original, "use a comma between latitude and longitude")
	}
	return nil
}

func parsePair(a, b string) (float64, float64, error) {
	lat, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("latitude %q is not a number", a)
	}
	lon, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("longitude %q is not a number", b)
	}
	if math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lon) || math.IsInf(lon, 0) {
		return 0, 0, fmt.Errorf("latitude and longitude must be finite")
	}
	if lat < -90 || lat > 90 {
		return 0, 0, fmt.Errorf("latitude %v is outside [-90,90]", lat)
	}
	if lon < -180 || lon > 180 {
		return 0, 0, fmt.Errorf("longitude %v is outside [-180,180]", lon)
	}
	return lat, lon, nil
}

// looksNumeric reports whether s could be a float token, including a broken
// one (empty, ".", "+"). Letters other than the exponent marker are not numeric,
// so "IN" and "Fort Wayne" stay place names.
func looksNumeric(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		switch r {
		case '+', '-', '.', 'e', 'E':
		default:
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func trimASCII(s string) string {
	return strings.Trim(s, " \t\n\r\v\f")
}

func isASCIISpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	default:
		return false
	}
}

func formatCoord(lat, lon float64) string {
	return strconv.FormatFloat(lat, 'f', -1, 64) + "," + strconv.FormatFloat(lon, 'f', -1, 64)
}
