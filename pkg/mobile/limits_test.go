package main

import (
	"runtime/debug"
	"testing"
)

func TestMobileMemoryLimits(t *testing.T) {
	const wantLimit int64 = 40 * 1024 * 1024
	const wantGC = 50

	// init applies the phone budget before any test runs. Setting the same
	// values again returns whatever is already in force.
	if got := debug.SetMemoryLimit(wantLimit); got != wantLimit {
		t.Fatalf("memory limit = %d, want %d", got, wantLimit)
	}
	if got := debug.SetGCPercent(wantGC); got != wantGC {
		t.Fatalf("GC percent = %d, want %d", got, wantGC)
	}
}
