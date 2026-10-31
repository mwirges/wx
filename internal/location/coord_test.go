package location

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type nopCache struct{}

func (nopCache) Get(string, any) bool { return false }
func (nopCache) Set(string, any, time.Duration) error {
	return nil
}

func TestResolveLatLonAcceptsCommaForm(t *testing.T) {
	restore := trapDo(t, &nominatimDo)
	defer restore()
	cases := []struct {
		in  string
		lat float64
		lon float64
	}{
		{"41.07,-85.14", 41.07, -85.14},
		{"  41.07 , -85.14  ", 41.07, -85.14},
		{"41.07,\t-85.14", 41.07, -85.14},
		{"-90,-180", -90, -180},
		{"90,180", 90, 180},
		{"+41.07,-85.14", 41.07, -85.14},
	}
	for _, tc := range cases {
		loc, err := Resolve(context.Background(), tc.in, nopCache{})
		if err != nil {
			t.Fatalf("Resolve(%q): %v", tc.in, err)
		}
		if loc.Lat < tc.lat-1e-9 || loc.Lat > tc.lat+1e-9 || loc.Lon < tc.lon-1e-9 || loc.Lon > tc.lon+1e-9 {
			t.Fatalf("Resolve(%q) = %v,%v want %v,%v", tc.in, loc.Lat, loc.Lon, tc.lat, tc.lon)
		}
		if loc.CountryCode != "" {
			t.Fatalf("Resolve(%q) country = %q, want empty (no geocoder)", tc.in, loc.CountryCode)
		}
	}
}

func TestResolveLatLonRejectsWithoutGeocode(t *testing.T) {
	restore := trapDo(t, &nominatimDo)
	defer restore()
	ipRestore := trapDo(t, &ipgeoDo)
	defer ipRestore()

	cases := []string{
		"91,0",
		"foo,bar",
		"41.07,",
		"41.07 -85.14",
		"NaN,1",
		"1,Inf",
		"+Inf,0",
		"90.0001,0",
		"0,180.0001",
		"-90.1,0",
		"0,-180.1",
		"1,2,3",
		",",
		"foo,0",
	}
	for _, in := range cases {
		_, err := Resolve(context.Background(), in, nopCache{})
		if !errors.Is(err, ErrInvalidLocation) {
			t.Errorf("Resolve(%q) err = %v, want ErrInvalidLocation", in, err)
		}
	}
}

func TestResolvePlaceStillGeocodes(t *testing.T) {
	var calls int
	prev := nominatimDo
	nominatimDo = func(r *http.Request) (*http.Response, error) {
		calls++
		if !strings.Contains(r.URL.Host, "nominatim.openstreetmap.org") {
			t.Errorf("host = %s", r.URL.Host)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent")
		}
		body := `[{"lat":"41.08","lon":"-85.14","display_name":"Fort Wayne","address":{"city":"Fort Wayne","state":"Indiana","country_code":"us"}}]`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}
	t.Cleanup(func() { nominatimDo = prev })

	for _, in := range []string{"Fort Wayne, IN", "Chicago, IL", "Paris, France"} {
		calls = 0
		loc, err := Resolve(context.Background(), in, nopCache{})
		if err != nil {
			t.Fatalf("Resolve(%q): %v", in, err)
		}
		if calls != 1 {
			t.Fatalf("Resolve(%q) nominatim calls = %d, want 1", in, calls)
		}
		if loc.Lat == 0 && loc.Lon == 0 {
			t.Fatalf("Resolve(%q) missing coordinates", in)
		}
	}
}

func TestReadRetry429ThenOK(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if ua := r.Header.Get("User-Agent"); ua == "" {
			t.Error("missing User-Agent")
		}
		if n < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"lat":"1","lon":"2"}]`))
	}))
	defer srv.Close()

	prevBack := locationBackoff
	locationBackoff = func(int) time.Duration { return time.Millisecond }
	t.Cleanup(func() { locationBackoff = prevBack })

	prev := nominatimDo
	nominatimDo = func(r *http.Request) (*http.Response, error) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, srv.URL, nil)
		if err != nil {
			return nil, err
		}
		req.Header = r.Header.Clone()
		return http.DefaultClient.Do(req)
	}
	t.Cleanup(func() { nominatimDo = prev })

	results, err := nominatimSearch(context.Background(), url.Values{"q": {"x"}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("attempts = %d, want 2", n)
	}
	if len(results) != 1 || results[0].Lat != "1" {
		t.Fatalf("results = %+v", results)
	}
}

func TestReadRetryPersistent429(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	prevBack := locationBackoff
	locationBackoff = func(int) time.Duration { return time.Millisecond }
	t.Cleanup(func() { locationBackoff = prevBack })
	prev := nominatimDo
	nominatimDo = func(r *http.Request) (*http.Response, error) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, srv.URL, nil)
		if err != nil {
			return nil, err
		}
		return http.DefaultClient.Do(req)
	}
	t.Cleanup(func() { nominatimDo = prev })

	_, err := nominatimSearch(context.Background(), url.Values{"q": {"x"}})
	if err == nil || !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("err = %v, want HTTP 429", err)
	}
	if n != 3 {
		t.Fatalf("attempts = %d, want 3", n)
	}
}

func TestReadRetry404NoRetry(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	prev := nominatimDo
	nominatimDo = func(r *http.Request) (*http.Response, error) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, srv.URL, nil)
		if err != nil {
			return nil, err
		}
		return http.DefaultClient.Do(req)
	}
	t.Cleanup(func() { nominatimDo = prev })
	_, err := nominatimSearch(context.Background(), url.Values{"q": {"x"}})
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("err = %v", err)
	}
	if n != 1 {
		t.Fatalf("attempts = %d, want 1", n)
	}
}

// trapDo fails the test if production HTTP runs.
func trapDo(t *testing.T, slot *func(*http.Request) (*http.Response, error)) func() {
	t.Helper()
	prev := *slot
	*slot = func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected HTTP %s %s", r.Method, r.URL)
		return nil, errors.New("geocoder called")
	}
	return func() { *slot = prev }
}
