package nws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL   = "https://api.weather.gov"
	userAgent = "wx/1.0 (github.com/mwirges/wx)"
)

type client struct {
	http    *http.Client
	baseURL string // overridable for tests
}

func newClient() *client {
	return &client{
		http:    &http.Client{Timeout: 15 * time.Second},
		baseURL: baseURL,
	}
}

type nwsErrorBody struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
}

// get performs an authenticated GET to the NWS API and decodes the JSON body into v.
func (c *client) get(ctx context.Context, url string, v any) error {
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
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, nwsErr.Detail)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}
