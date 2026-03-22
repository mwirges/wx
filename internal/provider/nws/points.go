package nws

import (
	"context"
	"fmt"
	"time"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
)

// gridInfo is stored in the cache to avoid re-fetching on every run.
type gridInfo struct {
	GridID                 string `json:"gridId"`
	GridX                  int    `json:"gridX"`
	GridY                  int    `json:"gridY"`
	ForecastURL            string `json:"forecastUrl"`
	ForecastHourlyURL      string `json:"forecastHourlyUrl"`
	ObservationStationsURL string `json:"observationStationsUrl"`
	StationID              string `json:"stationId"`
	CityState              string `json:"cityState"` // from relativeLocation
}

type pointsResponse struct {
	Properties struct {
		GridID              string `json:"gridId"`
		GridX               int    `json:"gridX"`
		GridY               int    `json:"gridY"`
		Forecast            string `json:"forecast"`
		ForecastHourly      string `json:"forecastHourly"`
		ObservationStations string `json:"observationStations"`
		RelativeLocation    struct {
			Properties struct {
				City  string `json:"city"`
				State string `json:"state"`
			} `json:"properties"`
		} `json:"relativeLocation"`
	} `json:"properties"`
}

type stationsResponse struct {
	Features []struct {
		Properties struct {
			StationIdentifier string `json:"stationIdentifier"`
			Name              string `json:"name"`
		} `json:"properties"`
	} `json:"features"`
}

// getGridInfo fetches (or retrieves from cache) NWS grid and station info for a location.
// Cached for 24 hours since grid points and nearest stations virtually never change.
func (p *Provider) getGridInfo(ctx context.Context, loc location.Location, c *cache.Cache) (*gridInfo, error) {
	cacheKey := fmt.Sprintf("nws:points:%.4f,%.4f", loc.Lat, loc.Lon)

	var info gridInfo
	if c.Get(cacheKey, &info) {
		return &info, nil
	}

	// Step 1: /points/{lat},{lon}
	url := fmt.Sprintf("%s/points/%.4f,%.4f", p.client.baseURL, loc.Lat, loc.Lon)
	var pts pointsResponse
	if err := p.client.get(ctx, url, &pts); err != nil {
		return nil, fmt.Errorf("nws: points lookup: %w", err)
	}
	if pts.Properties.GridID == "" {
		return nil, fmt.Errorf("nws: points response missing gridId — location may be outside NWS coverage")
	}

	// Step 2: fetch nearest observation station
	var stations stationsResponse
	if err := p.client.get(ctx, pts.Properties.ObservationStations, &stations); err != nil {
		return nil, fmt.Errorf("nws: stations lookup: %w", err)
	}
	if len(stations.Features) == 0 {
		return nil, fmt.Errorf("nws: no observation stations found near location")
	}

	cityState := pts.Properties.RelativeLocation.Properties.City
	if state := pts.Properties.RelativeLocation.Properties.State; state != "" {
		if cityState != "" {
			cityState += ", " + state
		} else {
			cityState = state
		}
	}

	info = gridInfo{
		GridID:                 pts.Properties.GridID,
		GridX:                  pts.Properties.GridX,
		GridY:                  pts.Properties.GridY,
		ForecastURL:            pts.Properties.Forecast,
		ForecastHourlyURL:      pts.Properties.ForecastHourly,
		ObservationStationsURL: pts.Properties.ObservationStations,
		StationID:              stations.Features[0].Properties.StationIdentifier,
		CityState:              cityState,
	}

	_ = c.Set(cacheKey, info, 24*time.Hour)
	return &info, nil
}
