package nws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestClientGet_Retry429ThenOK(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"detail":"slow down"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newClient()
	c.backoff = func(int) time.Duration { return time.Millisecond }
	var v map[string]bool
	if err := c.get(context.Background(), srv.URL, &v); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("attempts = %d, want 3", n)
	}
	if !v["ok"] {
		t.Fatalf("body = %v", v)
	}
}

func TestClientGet_Persistent429(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"detail":"slow down"}`))
	}))
	defer srv.Close()

	c := newClient()
	c.backoff = func(int) time.Duration { return time.Millisecond }
	var v any
	err := c.get(context.Background(), srv.URL, &v)
	if err == nil || !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("err = %v, want HTTP 429", err)
	}
	if n != 3 {
		t.Fatalf("attempts = %d, want 3", n)
	}
}

func TestClientGet_503ThenOK(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := newClient()
	c.backoff = func(int) time.Duration { return time.Millisecond }
	var v map[string]bool
	if err := c.get(context.Background(), srv.URL, &v); err != nil {
		t.Fatal(err)
	}
	if n != 2 || !v["ok"] {
		t.Fatalf("n=%d v=%v", n, v)
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
