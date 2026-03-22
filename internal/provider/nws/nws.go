package nws

import (
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/provider"
)

// Provider implements provider.WeatherProvider using the NWS (weather.gov) API.
type Provider struct {
	client *client
}

// New returns a new NWS Provider.
func New() *Provider {
	return &Provider{client: newClient()}
}

// init registers the NWS provider automatically when this package is imported.
func init() {
	provider.Register(New())
}

func (p *Provider) Name() string { return "nws" }

// Supports returns true for US locations only.
func (p *Provider) Supports(loc location.Location) bool {
	return loc.CountryCode == "US"
}
