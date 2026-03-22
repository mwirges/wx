package cache

import (
	"testing"
	"time"
)

type payload struct {
	Value string `json:"value"`
}

func newTestCache(t *testing.T) *Cache {
	t.Helper()
	c, err := NewWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("NewWithDir: %v", err)
	}
	return c
}

func TestCacheRoundTrip(t *testing.T) {
	c := newTestCache(t)

	want := payload{Value: "hello"}
	if err := c.Set("key1", want, time.Hour); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var got payload
	if !c.Get("key1", &got) {
		t.Fatal("Get returned false; expected cache hit")
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCacheMiss(t *testing.T) {
	c := newTestCache(t)
	var v payload
	if c.Get("nonexistent", &v) {
		t.Error("Get returned true for missing key")
	}
}

func TestCacheExpiry(t *testing.T) {
	c := newTestCache(t)

	if err := c.Set("expiring", payload{Value: "bye"}, -1*time.Second); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var v payload
	if c.Get("expiring", &v) {
		t.Error("Get returned true for expired entry")
	}
}

func TestCacheOverwrite(t *testing.T) {
	c := newTestCache(t)

	_ = c.Set("k", payload{Value: "first"}, time.Hour)
	_ = c.Set("k", payload{Value: "second"}, time.Hour)

	var got payload
	if !c.Get("k", &got) {
		t.Fatal("Get returned false")
	}
	if got.Value != "second" {
		t.Errorf("got %q, want %q", got.Value, "second")
	}
}

func TestCacheNoOp(t *testing.T) {
	c := NewNoOp()

	if err := c.Set("k", payload{Value: "x"}, time.Hour); err != nil {
		t.Errorf("NoOp Set returned error: %v", err)
	}

	var v payload
	if c.Get("k", &v) {
		t.Error("NoOp Get returned true; should always miss")
	}
}

func TestCacheDifferentKeys(t *testing.T) {
	c := newTestCache(t)

	_ = c.Set("a", payload{Value: "alpha"}, time.Hour)
	_ = c.Set("b", payload{Value: "beta"}, time.Hour)

	var a, b payload
	c.Get("a", &a)
	c.Get("b", &b)

	if a.Value != "alpha" {
		t.Errorf("key a: got %q, want %q", a.Value, "alpha")
	}
	if b.Value != "beta" {
		t.Errorf("key b: got %q, want %q", b.Value, "beta")
	}
}
