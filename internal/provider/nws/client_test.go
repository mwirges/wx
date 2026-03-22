package nws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientGet_UserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != userAgent {
			t.Errorf("User-Agent = %q, want %q", ua, userAgent)
		}
		accept := r.Header.Get("Accept")
		if accept != "application/geo+json" {
			t.Errorf("Accept = %q, want %q", accept, "application/geo+json")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newClient()
	var v map[string]bool
	if err := c.get(context.Background(), srv.URL, &v); err != nil {
		t.Fatalf("get: %v", err)
	}
	if !v["ok"] {
		t.Error("response body not decoded correctly")
	}
}

func TestClientGet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"title":"Service Unavailable","detail":"NWS is down","status":503}`))
	}))
	defer srv.Close()

	c := newClient()
	var v any
	err := c.get(context.Background(), srv.URL, &v)
	if err == nil {
		t.Fatal("expected error for HTTP 503, got nil")
	}
}

func TestClientGet_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newClient()
	var v any
	err := c.get(context.Background(), srv.URL, &v)
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}
