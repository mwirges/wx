package mcp

import (
	"context"
	"io"
	"time"
)

// watchParent cancels when pid exits. pid <= 1 cancels immediately and does
// not probe with kill. The probe itself uses signal 0, which does not
// terminate the process. The Mac app passes its own pid via --parent-pid
// so an app crash (including kill -9, which skips applicationWillTerminate)
// still ends the child.
func watchParent(ctx context.Context, pid int, cancel context.CancelFunc) {
	if pid <= 1 {
		cancel()
		return
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		if !processAlive(pid) {
			cancel()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// watchStdin cancels when r returns an error, including EOF. The Mac
// supervisor keeps the write end open for the life of the child. It does
// not write the token or anything else. A headless `wx mcp serve` does not
// enable this watch, so a terminal or /dev/null stdin stays up.
func watchStdin(ctx context.Context, r io.Reader, cancel context.CancelFunc) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 256)
		for {
			if _, err := r.Read(buf); err != nil {
				return
			}
		}
	}()
	select {
	case <-ctx.Done():
	case <-done:
		cancel()
	}
}

// RestartDelay is the Mac supervisor's backoff, tested here so the sequence
// stays pinned. attempt is 1-based. Attempts after maxRestarts do not run.
// Delays: 500ms, 1s, 2s, 4s, 8s.
func RestartDelay(attempt int) (time.Duration, bool) {
	const maxRestarts = 5
	if attempt < 1 || attempt > maxRestarts {
		return 0, false
	}
	delay := 500 * time.Millisecond
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay > 8*time.Second {
			delay = 8 * time.Second
		}
	}
	return delay, true
}
