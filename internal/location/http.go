package location

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// locationBackoff is the wait before retry n (0 is the first retry).
// Tests shorten it. Production stays under half a second per wait.
var locationBackoff = func(retry int) time.Duration {
	d := 50 * time.Millisecond << retry
	if d > 400*time.Millisecond {
		d = 400 * time.Millisecond
	}
	return d
}

func retryStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// readRetry GETs req up to three times. 429 and transient 5xx are retried.
// Other HTTP statuses and decode-ready 200 bodies return immediately.
// do is http.DefaultClient.Do in production (nominatimDo / ipgeoDo).
func readRetry(ctx context.Context, do func(*http.Request) (*http.Response, error), req *http.Request) ([]byte, error) {
	const attempts = 3
	var last error
	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if i > 0 {
			if err := sleepCtx(ctx, locationBackoff(i-1)); err != nil {
				return nil, err
			}
		}
		resp, err := do(req.Clone(ctx))
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			last = fmt.Errorf("request: %w", err)
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return body, nil
		}
		last = fmt.Errorf("HTTP %d", resp.StatusCode)
		if !retryStatus(resp.StatusCode) {
			return nil, last
		}
	}
	if last == nil {
		last = fmt.Errorf("HTTP request failed")
	}
	return nil, last
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
