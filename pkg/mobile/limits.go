package main

import "runtime/debug"

// Phone heap budget. 40 MiB is the soft limit (GOMEMLIMIT). GC at 50 keeps
// the heap from doubling the way GOGC=100 does, which is what gets an
// in-process Go runtime killed by iOS jetsam.
const (
	mobileMemoryLimit int64 = 40 * 1024 * 1024
	mobileGCPercent         = 50
)

func applyMobileLimits() {
	debug.SetMemoryLimit(mobileMemoryLimit)
	debug.SetGCPercent(mobileGCPercent)
}
