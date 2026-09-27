package provider_test

import (
	"context"
	"testing"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/provider"
)

type mockProvider struct {
	name        string
	supportFunc func(loc location.Location) bool
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Supports(loc location.Location) bool {
	if m.supportFunc != nil {
		return m.supportFunc(loc)
	}
	return false
}
func (m *mockProvider) CurrentConditions(ctx context.Context, loc location.Location, c *cache.Cache) (*models.CurrentConditions, error) {
	return &models.CurrentConditions{StationID: m.name}, nil
}
func (m *mockProvider) Forecast(ctx context.Context, loc location.Location, hourly bool, c *cache.Cache) (*models.Forecast, error) {
	return &models.Forecast{}, nil
}
func (m *mockProvider) Alerts(ctx context.Context, loc location.Location, c *cache.Cache) ([]models.Alert, error) {
	return []models.Alert{}, nil
}

func TestProviderRegistry_PriorityAndSelection(t *testing.T) {
	initial := provider.All()
	provider.ClearRegistry()
	defer func() {
		provider.ClearRegistry()
		for _, p := range initial {
			provider.Register(p)
		}
	}()

	fallback := &mockProvider{
		name:        "global-fallback",
		supportFunc: func(loc location.Location) bool { return true },
	}
	specialized := &mockProvider{
		name: "us-specialized",
		supportFunc: func(loc location.Location) bool {
			return loc.CountryCode == "US"
		},
	}

	// Register out of order
	provider.RegisterWithPriority(fallback, provider.PriorityFallback)
	provider.RegisterWithPriority(specialized, provider.PrioritySpecialized)

	// US location: specialized should win due to higher priority
	usLoc := location.Location{Lat: 41.0, Lon: -85.0, CountryCode: "US"}
	selected, err := provider.ForLocation(usLoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.Name() != "us-specialized" {
		t.Errorf("got provider %q, want %q", selected.Name(), "us-specialized")
	}

	// Non-US location: fallback should win because specialized does not support it
	caLoc := location.Location{Lat: 43.7, Lon: -79.4, CountryCode: "CA"}
	selectedCA, err := provider.ForLocation(caLoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selectedCA.Name() != "global-fallback" {
		t.Errorf("got provider %q, want %q", selectedCA.Name(), "global-fallback")
	}

	// Unsupported location
	unsupported := &mockProvider{
		name:        "restricted",
		supportFunc: func(loc location.Location) bool { return false },
	}
	provider.ClearRegistry()
	provider.Register(unsupported)
	_, err = provider.ForLocation(usLoc)
	if err == nil {
		t.Error("expected error for unsupported location, got nil")
	}
}

func TestProviderRegistry_PreferenceAndFallback(t *testing.T) {
	initial := provider.All()
	provider.ClearRegistry()
	defer func() {
		provider.ClearRegistry()
		for _, p := range initial {
			provider.Register(p)
		}
	}()

	p1 := &mockProvider{
		name:        "nws",
		supportFunc: func(loc location.Location) bool { return loc.CountryCode == "US" },
	}
	p2 := &mockProvider{
		name:        "openmeteo",
		supportFunc: func(loc location.Location) bool { return true },
	}

	provider.RegisterWithPriority(p1, 100)
	provider.RegisterWithPriority(p2, 10)

	usLoc := location.Location{Lat: 41.0, Lon: -85.0, CountryCode: "US"}
	caLoc := location.Location{Lat: 43.7, Lon: -79.4, CountryCode: "CA"}

	// Preference override for US
	pref, err := provider.ForLocationWithPreference(usLoc, "openmeteo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pref.Name() != "openmeteo" {
		t.Errorf("got %q, want %q", pref.Name(), "openmeteo")
	}

	// Preference for unsupported provider should fail
	_, err = provider.ForLocationWithPreference(caLoc, "nws")
	if err == nil {
		t.Error("expected error when preferred provider does not support location")
	}

	// Unknown provider should fail
	_, err = provider.ForLocationWithPreference(usLoc, "nonexistent")
	if err == nil {
		t.Error("expected error for unknown provider")
	}

	// Empty preference should delegate to ForLocation
	auto, err := provider.ForLocationWithPreference(usLoc, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auto.Name() != "nws" {
		t.Errorf("got %q, want %q", auto.Name(), "nws")
	}

	// Fallback for location when primary fails
	fb, err := provider.FallbackForLocation(usLoc, "nws")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fb.Name() != "openmeteo" {
		t.Errorf("got fallback %q, want %q", fb.Name(), "openmeteo")
	}

	// Fallback when no other provider exists
	_, err = provider.FallbackForLocation(caLoc, "openmeteo")
	if err == nil {
		t.Error("expected error when no other provider supports location")
	}
}

func TestProviderRegistry_GetAndAll(t *testing.T) {
	initial := provider.All()
	provider.ClearRegistry()
	defer func() {
		provider.ClearRegistry()
		for _, p := range initial {
			provider.Register(p)
		}
	}()

	p1 := &mockProvider{name: "alpha"}
	p2 := &mockProvider{name: "beta"}

	provider.RegisterWithPriority(p1, 10)
	provider.RegisterWithPriority(p2, 90)

	all := provider.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d providers, want 2", len(all))
	}
	if all[0].Name() != "beta" || all[1].Name() != "alpha" {
		t.Errorf("All() not in descending priority: [%s, %s]", all[0].Name(), all[1].Name())
	}

	p, ok := provider.Get("ALPHA")
	if !ok || p.Name() != "alpha" {
		t.Errorf("Get(ALPHA) failed: ok=%v, p=%v", ok, p)
	}

	_, ok = provider.Get("missing")
	if ok {
		t.Error("Get(missing) expected false, got true")
	}
}
