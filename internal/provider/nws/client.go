package nws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL   = "https://api.weather.gov"
	userAgent = "wx/1.0 (github.com/mwirges/wx)"
)

type client struct {
	http     *http.Client
	baseURL  string // overridable for tests
	attempts int
	backoff  func(retry int) time.Duration
}

func newClient() *client {
	return &client{
		http:     &http.Client{Timeout: 15 * time.Second},
		baseURL:  baseURL,
		attempts: 3,
		backoff:  defaultBackoff,
	}
}

func defaultBackoff(retry int) time.Duration {
	d := 50 * time.Millisecond << retry
	if d > 400*time.Millisecond {
		d = 400 * time.Millisecond
	}
	return d
}

// httpStatusError is an NWS HTTP failure. 429 and transient 5xx are retried.
type httpStatusError struct {
	code   int
	detail string
}

func (e *httpStatusError) Error() string {
	if e.detail != "" {
		return fmt.Sprintf("HTTP %d: %s", e.code, e.detail)
	}
	return fmt.Sprintf("HTTP %d", e.code)
}

type nwsErrorBody struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
}

// get performs an authenticated GET to the NWS API and decodes the JSON body into v.
// 429 and transient 5xx are retried with a short bounded backoff. The last
// error is returned when the budget is spent, so callers can map HTTP 429
// without hanging.
func (c *client) get(ctx context.Context, url string, v any) error {
	attempts := c.attempts
	if attempts < 1 {
		attempts = 3
	}
	back := c.backoff
	if back == nil {
		back = defaultBackoff
	}
	var last error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			if err := sleepCtx(ctx, back(i-1)); err != nil {
				return err
			}
		}
		err := c.getOnce(ctx, url, v)
		if err == nil {
			return nil
		}
		last = err
		if !retryable(err) {
			return err
		}
	}
	if last == nil {
		last = errors.New("nws: request failed")
	}
	return last
}

func (c *client) getOnce(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var nwsErr nwsErrorBody
		if json.Unmarshal(body, &nwsErr) == nil && nwsErr.Detail != "" {
			return &httpStatusError{code: resp.StatusCode, detail: nwsErr.Detail}
		}
		return &httpStatusError{code: resp.StatusCode}
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}

func retryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var hs *httpStatusError
	if errors.As(err, &hs) {
		switch hs.code {
		case http.StatusTooManyRequests, http.StatusInternalServerError,
			http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "decode:") || strings.HasPrefix(msg, "build request:") {
		return false
	}
	return true
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
