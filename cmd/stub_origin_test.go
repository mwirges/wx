package cmd

import (
	"net/url"
	"strings"
	"testing"
)

func TestRewriteURL(t *testing.T) {
	origin, err := url.Parse("http://127.0.0.1:9")
	if err != nil {
		t.Fatal(err)
	}
	src, err := url.Parse("https://api.weather.gov/points/41.0000,-85.0000?foo=1")
	if err != nil {
		t.Fatal(err)
	}
	got := rewriteURL(origin, src).String()
	want := "http://127.0.0.1:9/api.weather.gov/points/41.0000,-85.0000?foo=1"
	if got != want {
		t.Fatalf("rewriteURL = %s, want %s", got, want)
	}
}

func TestRewriteURL_PreservesQueryEncoding(t *testing.T) {
	origin, err := url.Parse("http://127.0.0.1:9")
	if err != nil {
		t.Fatal(err)
	}
	src, err := url.Parse("https://nominatim.openstreetmap.org/search?q=New+Haven%2C+IN&format=json")
	if err != nil {
		t.Fatal(err)
	}
	got := rewriteURL(origin, src)
	if !strings.Contains(got.Path, "nominatim.openstreetmap.org/search") {
		t.Fatalf("path = %s", got.Path)
	}
	if got.Query().Get("q") != "New Haven, IN" {
		t.Fatalf("q = %q", got.Query().Get("q"))
	}
}
