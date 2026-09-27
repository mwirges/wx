package openmeteo

import (
	"context"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
)

// Alerts returns active weather alerts. Open-Meteo's standard API does not
// provide CAP/SAME government alerts, so this returns an empty slice.
func (p *Provider) Alerts(ctx context.Context, loc location.Location, c *cache.Cache) ([]models.Alert, error) {
	return []models.Alert{}, nil
}
