package nws

import (
	"context"
	"fmt"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

type alertsResponse struct {
	Features []struct {
		Properties struct {
			Event       string `json:"event"`
			Headline    string `json:"headline"`
			Description string `json:"description"`
			Instruction string `json:"instruction"`
			Severity    string `json:"severity"`
			Urgency     string `json:"urgency"`
			Effective   string `json:"effective"`
			Expires     string `json:"expires"`
			AreaDesc    string `json:"areaDesc"`
		} `json:"properties"`
	} `json:"features"`
}

// Alerts fetches active NWS weather alerts for the given location.
// Returns an empty slice (not an error) when no alerts are active.
// Cached for 5 minutes.
func (p *Provider) Alerts(ctx context.Context, loc location.Location, c *cache.Cache) ([]models.Alert, error) {
	cacheKey := fmt.Sprintf("nws:alerts:%.4f,%.4f", loc.Lat, loc.Lon)

	var alerts []models.Alert
	if c.Get(cacheKey, &alerts) {
		return alerts, nil
	}

	url := fmt.Sprintf("%s/alerts/active?point=%.4f,%.4f", p.client.baseURL, loc.Lat, loc.Lon)
	var resp alertsResponse
	if err := p.client.get(ctx, url, &resp); err != nil {
		return nil, fmt.Errorf("nws: alerts: %w", err)
	}

	alerts = make([]models.Alert, 0, len(resp.Features))
	for _, f := range resp.Features {
		effective, _ := time.Parse(time.RFC3339, f.Properties.Effective)
		expires, _ := time.Parse(time.RFC3339, f.Properties.Expires)

		alerts = append(alerts, models.Alert{
			Event:       f.Properties.Event,
			Headline:    f.Properties.Headline,
			Description: f.Properties.Description,
			Instruction: f.Properties.Instruction,
			Severity:    f.Properties.Severity,
			Urgency:     f.Properties.Urgency,
			Effective:   effective,
			Expires:     expires,
			AreaDesc:    f.Properties.AreaDesc,
		})
	}

	_ = c.Set(cacheKey, alerts, 5*time.Minute)
	return alerts, nil
}
