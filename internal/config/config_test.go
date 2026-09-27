package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFrom_Missing(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.DefaultLocation != "" || cfg.Units != "" {
		t.Errorf("expected empty config, got %+v", cfg)
	}
}

func TestLoadFrom_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{
		"default_location": "Kansas City, MO",
		"units": "imperial"
	}`), 0o644)

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.DefaultLocation != "Kansas City, MO" {
		t.Errorf("DefaultLocation = %q, want %q", cfg.DefaultLocation, "Kansas City, MO")
	}
	if cfg.Units != "imperial" {
		t.Errorf("Units = %q, want %q", cfg.Units, "imperial")
	}
}

func TestLoadFrom_ZipCode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{"default_location": "64101"}`), 0o644)

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.DefaultLocation != "64101" {
		t.Errorf("DefaultLocation = %q, want %q", cfg.DefaultLocation, "64101")
	}
}

func TestLoadFrom_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`not json`), 0o644)

	_, err := LoadFrom(path)
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestLoadFrom_PartialConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Only units set, no default_location
	os.WriteFile(path, []byte(`{"units": "metric"}`), 0o644)

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Units != "metric" {
		t.Errorf("Units = %q, want %q", cfg.Units, "metric")
	}
	if cfg.DefaultLocation != "" {
		t.Errorf("DefaultLocation = %q, want empty", cfg.DefaultLocation)
	}
}

func TestLoadFrom_EmptyObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{}`), 0o644)

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.DefaultLocation != "" || cfg.Units != "" {
		t.Errorf("expected empty config, got %+v", cfg)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := &Config{DefaultLocation: "Kansas City, MO", Units: "imperial"}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom after Save: %v", err)
	}
	if got.DefaultLocation != cfg.DefaultLocation {
		t.Errorf("DefaultLocation = %q, want %q", got.DefaultLocation, cfg.DefaultLocation)
	}
	if got.Units != cfg.Units {
		t.Errorf("Units = %q, want %q", got.Units, cfg.Units)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	// Path with a subdirectory that doesn't exist yet
	path := filepath.Join(t.TempDir(), "subdir", "wx", "config.json")
	if err := Save(path, &Config{Units: "metric"}); err != nil {
		t.Fatalf("Save (new dir): %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestSaveOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(`{"default_location":"old","units":"metric"}`), 0o644)

	if err := Save(path, &Config{DefaultLocation: "new", Units: "imperial"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, _ := LoadFrom(path)
	if got.DefaultLocation != "new" {
		t.Errorf("DefaultLocation = %q, want %q", got.DefaultLocation, "new")
	}
}

func TestSaveClearsField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(`{"default_location":"Kansas City, MO","units":"metric"}`), 0o644)

	// Save with empty DefaultLocation should persist the empty value
	if err := Save(path, &Config{Units: "imperial"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, _ := LoadFrom(path)
	if got.DefaultLocation != "" {
		t.Errorf("DefaultLocation = %q, want empty", got.DefaultLocation)
	}
}

func TestFavorites(t *testing.T) {
	cfg := &Config{}

	// Add new
	cfg.SetFavorite("Home", "Fort Wayne, IN")
	cfg.SetFavorite("Work", "46802")
	if len(cfg.Favorites) != 2 {
		t.Fatalf("expected 2 favorites, got %d", len(cfg.Favorites))
	}

	// Lookup case-insensitive
	val, ok := cfg.GetFavorite("home")
	if !ok || val != "Fort Wayne, IN" {
		t.Errorf("GetFavorite('home') = %q, %v; want 'Fort Wayne, IN', true", val, ok)
	}
	val, ok = cfg.GetFavorite("WORK")
	if !ok || val != "46802" {
		t.Errorf("GetFavorite('WORK') = %q, %v; want '46802', true", val, ok)
	}
	_, ok = cfg.GetFavorite("Nonexistent")
	if ok {
		t.Errorf("expected false for nonexistent favorite")
	}

	// Overwrite existing
	cfg.SetFavorite("home", "Indianapolis, IN")
	if len(cfg.Favorites) != 2 {
		t.Fatalf("expected 2 favorites after update, got %d", len(cfg.Favorites))
	}
	val, _ = cfg.GetFavorite("Home")
	if val != "Indianapolis, IN" {
		t.Errorf("expected updated value 'Indianapolis, IN', got %q", val)
	}

	// Remove
	removed := cfg.RemoveFavorite("work")
	if !removed {
		t.Errorf("expected true for removed favorite")
	}
	if len(cfg.Favorites) != 1 {
		t.Errorf("expected 1 favorite left, got %d", len(cfg.Favorites))
	}
	_, ok = cfg.GetFavorite("Work")
	if ok {
		t.Errorf("expected Work to be gone")
	}
}

func TestRecentLocations(t *testing.T) {
	cfg := &Config{}

	// Add recents
	if !cfg.AddRecent("Chicago, IL") {
		t.Errorf("expected AddRecent('Chicago, IL') to return true")
	}
	cfg.AddRecent("Denver, CO")
	cfg.AddRecent("Austin, TX")

	if cfg.AddRecent("Austin, TX") {
		t.Errorf("expected AddRecent('Austin, TX') to return false when already top recent")
	}
	if cfg.AddRecent("") {
		t.Errorf("expected AddRecent('') to return false")
	}

	if len(cfg.RecentLocations) != 3 {
		t.Fatalf("expected 3 recents, got %d", len(cfg.RecentLocations))
	}
	if cfg.RecentLocations[0] != "Austin, TX" {
		t.Errorf("expected most recent to be Austin, TX, got %q", cfg.RecentLocations[0])
	}

	// Deduplication moves to front
	if !cfg.AddRecent("Chicago, IL") {
		t.Errorf("expected AddRecent('Chicago, IL') to return true when moving to front")
	}
	if len(cfg.RecentLocations) != 3 {
		t.Fatalf("expected 3 recents after deduplication, got %d", len(cfg.RecentLocations))
	}
	if cfg.RecentLocations[0] != "Chicago, IL" {
		t.Errorf("expected Chicago, IL to move to front, got %q", cfg.RecentLocations[0])
	}

	// Cap at 10
	for i := 1; i <= 15; i++ {
		cfg.AddRecent(filepath.Join("City", string(rune('A'+i))))
	}
	if len(cfg.RecentLocations) != 10 {
		t.Errorf("expected capped at 10, got %d", len(cfg.RecentLocations))
	}

	// Clear
	cfg.ClearRecents()
	if len(cfg.RecentLocations) != 0 {
		t.Errorf("expected empty recents after clear, got %v", cfg.RecentLocations)
	}
}

func TestResolveLocation(t *testing.T) {
	cfg := &Config{
		DefaultLocation: "Default City, ST",
	}
	cfg.SetFavorite("Home", "Fort Wayne, IN")
	cfg.SetFavorite("Default City, ST", "Resolved Default, ST")

	// Match favorite alias
	if got := cfg.ResolveLocation("home"); got != "Fort Wayne, IN" {
		t.Errorf("ResolveLocation('home') = %q, want 'Fort Wayne, IN'", got)
	}

	// Literal input when not in favorites
	if got := cfg.ResolveLocation("64101"); got != "64101" {
		t.Errorf("ResolveLocation('64101') = %q, want '64101'", got)
	}

	// Fallback to default location (which itself resolves alias)
	if got := cfg.ResolveLocation(""); got != "Resolved Default, ST" {
		t.Errorf("ResolveLocation('') = %q, want 'Resolved Default, ST'", got)
	}
}

func TestPerLocationSettings(t *testing.T) {
	cfg := &Config{
		Units: "imperial",
	}

	// Unset returns false and defaults
	if _, ok := cfg.GetLocationSettings("Chicago"); ok {
		t.Errorf("expected false for unset location")
	}
	if u := cfg.GetEffectiveUnits("Chicago"); u != "imperial" {
		t.Errorf("expected global default imperial, got %q", u)
	}

	// Set per-location settings
	cfg.SetLocationSettings("chicago", PerLocationSettings{
		Units:               "metric",
		DefaultRadarProduct: "base-reflectivity",
		DefaultRadarRadius:  150,
		RadarStation:        "KLOT",
	})

	// Case-insensitive lookup
	s, ok := cfg.GetLocationSettings("Chicago")
	if !ok {
		t.Fatalf("expected to find Chicago settings")
	}
	if s.Units != "metric" || s.RadarStation != "KLOT" || s.DefaultRadarRadius != 150 || s.DefaultRadarProduct != "base-reflectivity" {
		t.Errorf("unexpected settings: %+v", s)
	}

	// Effective helpers
	if u := cfg.GetEffectiveUnits("Denver", "Chicago"); u != "metric" {
		t.Errorf("expected Denver fallback to Chicago metric, got %q", u)
	}
	if prod := cfg.GetEffectiveRadarProduct("Chicago"); prod != "base-reflectivity" {
		t.Errorf("expected base-reflectivity, got %q", prod)
	}
	if rad := cfg.GetEffectiveRadarRadius("Chicago"); rad != 150 {
		t.Errorf("expected radius 150, got %f", rad)
	}
	if stn := cfg.GetEffectiveRadarStation("Chicago"); stn != "KLOT" {
		t.Errorf("expected station KLOT, got %q", stn)
	}

	// Update existing
	cfg.SetLocationSettings("CHICAGO", PerLocationSettings{
		Units: "imperial",
	})
	s, _ = cfg.GetLocationSettings("chicago")
	if s.Units != "imperial" || s.RadarStation != "" {
		t.Errorf("expected updated settings, got %+v", s)
	}
}

func TestConfig_Provider(t *testing.T) {
	cfg := &Config{
		Provider: "nws",
	}

	// Default provider
	if p := cfg.GetEffectiveProvider(); p != "nws" {
		t.Errorf("GetEffectiveProvider() = %q, want %q", p, "nws")
	}

	// Per-location provider override
	cfg.SetLocationSettings("Toronto", PerLocationSettings{
		Provider: "openmeteo",
	})

	if p := cfg.GetEffectiveProvider("Toronto"); p != "openmeteo" {
		t.Errorf("GetEffectiveProvider(Toronto) = %q, want %q", p, "openmeteo")
	}

	// Unset location falls back to global
	if p := cfg.GetEffectiveProvider("Chicago"); p != "nws" {
		t.Errorf("GetEffectiveProvider(Chicago) = %q, want %q", p, "nws")
	}
}

