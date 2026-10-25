package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

// WX_STUB_ORIGIN, when set to an absolute URL such as http://127.0.0.1:9,
// sends every default-transport HTTP request to that origin. The original
// host is kept as the first path segment so a fixture server can route it.
// The mobile bridge test sets this. Unset, wx talks to the public endpoints.
func init() {
	raw := strings.TrimSpace(os.Getenv("WX_STUB_ORIGIN"))
	if raw == "" {
		return
	}
	if err := installStubOrigin(raw); err != nil {
		fmt.Fprintf(os.Stderr, "wx: WX_STUB_ORIGIN: %v\n", err)
	}
}

var stubOnce sync.Once

func installStubOrigin(raw string) error {
	origin, err := url.Parse(raw)
	if err != nil || origin.Scheme == "" || origin.Host == "" {
		return fmt.Errorf("invalid origin %q", raw)
	}
	stubOnce.Do(func() {
		base := http.DefaultTransport
		http.DefaultTransport = &stubTransport{origin: origin, base: base}
	})
	return nil
}

type stubTransport struct {
	origin *url.URL
	base   http.RoundTripper
}

func (t *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	out := req.Clone(req.Context())
	out.URL = rewriteURL(t.origin, req.URL)
	out.Host = out.URL.Host
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(out)
}

// rewriteURL maps https://api.weather.gov/points/1,2?q=1 onto
// http://127.0.0.1:9/api.weather.gov/points/1,2?q=1.
func rewriteURL(origin, src *url.URL) *url.URL {
	path := "/"
	host := "unknown"
	rawQuery := ""
	if src != nil {
		if src.Path != "" {
			path = src.Path
		}
		if src.Host != "" {
			host = src.Host
		}
		rawQuery = src.RawQuery
	}
	u := *origin
	prefix := strings.TrimRight(origin.Path, "/")
	u.Path = prefix + "/" + host + path
	u.RawPath = ""
	u.RawQuery = rawQuery
	u.Fragment = ""
	return &u
}
