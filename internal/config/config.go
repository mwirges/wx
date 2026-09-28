package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocationEntry represents a user-defined favorite location alias.
type LocationEntry struct {
	Name  string `json:"name"`
	Value string `json:"value"` // same formats as --location (zip code, "City, ST", etc.)
}

// PerLocationSettings defines custom user settings for a specific location.
type PerLocationSettings struct {
	Units               string  `json:"units,omitempty"`                 // "imperial" | "metric"
	Provider            string  `json:"provider,omitempty"`              // e.g. "nws", "openmeteo"
	DefaultRadarProduct string  `json:"default_radar_product,omitempty"` // e.g. "composite-reflectivity"
	DefaultRadarRadius  float64 `json:"default_radar_radius,omitempty"`  // e.g. 150
	RadarStation        string  `json:"radar_station,omitempty"`         // e.g. "KIWX"
}

// Config holds user-level preferences loaded from ~/.config/wx/config.json.
// All fields are optional; missing fields leave the app using its built-in defaults.
type Config struct {
	// DefaultLocation is used when --location is not passed on the command line.
	// Accepts the same formats as --location: zip code, "City, ST", or empty to
	// fall back to IP-based auto-detection.
	DefaultLocation string `json:"default_location,omitempty"`

	// Units sets the default display units: "imperial" or "metric".
	// Overridden by --units on the command line.
	Units string `json:"units,omitempty"`

	// Provider sets the default weather provider: "nws" or "openmeteo".
	// Overridden by --provider on the command line.
	Provider string `json:"provider,omitempty"`

	// Notifications enables or disables desktop notifications for active alerts.
	Notifications *bool `json:"notifications,omitempty"`

	// Favorites is a list of named locations/aliases (e.g. Home, Work, Cabin).
	Favorites []LocationEntry `json:"favorites,omitempty"`

	// RecentLocations tracks the most recently queried locations (capped at 10).
	RecentLocations []string `json:"recent_locations,omitempty"`

	// PerLocation stores customized settings per location (alias, zip, or city name).
	PerLocation map[string]PerLocationSettings `json:"per_location,omitempty"`

	// MenuBarFormat specifies the macOS menu bar format: "compact", "standard", or "tactical".
	MenuBarFormat string `json:"menu_bar_format,omitempty"`
}

// GetFavorite looks up a favorite location by name (case-insensitive).
func (c *Config) GetFavorite(name string) (string, bool) {
	for _, f := range c.Favorites {
		if strings.EqualFold(f.Name, name) {
			return f.Value, true
		}
	}
	return "", false
}

// SetFavorite adds or updates a favorite location.
func (c *Config) SetFavorite(name, value string) {
	for i, f := range c.Favorites {
		if strings.EqualFold(f.Name, name) {
			c.Favorites[i].Name = name
			c.Favorites[i].Value = value
			return
		}
	}
	c.Favorites = append(c.Favorites, LocationEntry{Name: name, Value: value})
}

// RemoveFavorite removes a favorite location by name (case-insensitive).
func (c *Config) RemoveFavorite(name string) bool {
	for i, f := range c.Favorites {
		if strings.EqualFold(f.Name, name) {
			c.Favorites = append(c.Favorites[:i], c.Favorites[i+1:]...)
			return true
		}
	}
	return false
}

// AddRecent prepends a location to RecentLocations, deduplicating and capping at 10.
// It returns true if RecentLocations was modified, or false if loc was already the most recent or empty.
func (c *Config) AddRecent(loc string) bool {
	if loc == "" {
		return false
	}
	if len(c.RecentLocations) > 0 && strings.EqualFold(c.RecentLocations[0], loc) {
		return false
	}
	var filtered []string
	for _, r := range c.RecentLocations {
		if !strings.EqualFold(r, loc) {
			filtered = append(filtered, r)
		}
	}
	c.RecentLocations = append([]string{loc}, filtered...)
	if len(c.RecentLocations) > 10 {
		c.RecentLocations = c.RecentLocations[:10]
	}
	return true
}

// ClearRecents empties the recent locations list.
func (c *Config) ClearRecents() {
	c.RecentLocations = nil
}

// ResolveLocation resolves a location query through favorites or default location.
// Precedence: explicit alias match -> input -> default_location.
func (c *Config) ResolveLocation(input string) string {
	if input != "" {
		if val, ok := c.GetFavorite(input); ok {
			return val
		}
		return input
	}
	if c.DefaultLocation != "" {
		if val, ok := c.GetFavorite(c.DefaultLocation); ok {
			return val
		}
		return c.DefaultLocation
	}
	return ""
}

// GetLocationSettings retrieves PerLocationSettings for a location (case-insensitive key match).
func (c *Config) GetLocationSettings(loc string) (PerLocationSettings, bool) {
	if c.PerLocation == nil || loc == "" {
		return PerLocationSettings{}, false
	}
	for k, v := range c.PerLocation {
		if strings.EqualFold(k, loc) {
			return v, true
		}
	}
	return PerLocationSettings{}, false
}

// SetLocationSettings stores PerLocationSettings for a location.
func (c *Config) SetLocationSettings(loc string, settings PerLocationSettings) {
	if c.PerLocation == nil {
		c.PerLocation = make(map[string]PerLocationSettings)
	}
	for k := range c.PerLocation {
		if strings.EqualFold(k, loc) {
			delete(c.PerLocation, k)
			break
		}
	}
	c.PerLocation[loc] = settings
}

// DeleteLocationSettings removes PerLocationSettings for a location (case-insensitive).
// Returns true if a setting was deleted.
func (c *Config) DeleteLocationSettings(loc string) bool {
	if c.PerLocation == nil || loc == "" {
		return false
	}
	deleted := false
	for k := range c.PerLocation {
		if strings.EqualFold(k, loc) {
			delete(c.PerLocation, k)
			deleted = true
		}
	}
	return deleted
}

// GetEffectiveUnits returns the units to use for a given location, checking candidate keys
// in priority order, then global config units, defaulting to "imperial".
func (c *Config) GetEffectiveUnits(locs ...string) string {
	for _, l := range locs {
		if l == "" {
			continue
		}
		if s, ok := c.GetLocationSettings(l); ok && s.Units != "" {
			return s.Units
		}
	}
	if c.Units != "" {
		return c.Units
	}
	return "imperial"
}

// GetEffectiveProvider returns the weather provider configured for candidate locations,
// then global config provider, defaulting to empty string (which means auto-selection).
func (c *Config) GetEffectiveProvider(locs ...string) string {
	for _, l := range locs {
		if l == "" {
			continue
		}
		if s, ok := c.GetLocationSettings(l); ok && s.Provider != "" {
			return s.Provider
		}
	}
	if c.Provider != "" {
		return c.Provider
	}
	return ""
}

// GetEffectiveRadarProduct returns the default radar product configured for any candidate location.
func (c *Config) GetEffectiveRadarProduct(locs ...string) string {
	for _, l := range locs {
		if l == "" {
			continue
		}
		if s, ok := c.GetLocationSettings(l); ok && s.DefaultRadarProduct != "" {
			return s.DefaultRadarProduct
		}
	}
	return ""
}

// GetEffectiveRadarRadius returns the default radar radius configured for any candidate location (or 0 if unset).
func (c *Config) GetEffectiveRadarRadius(locs ...string) float64 {
	for _, l := range locs {
		if l == "" {
			continue
		}
		if s, ok := c.GetLocationSettings(l); ok && s.DefaultRadarRadius > 0 {
			return s.DefaultRadarRadius
		}
	}
	return 0
}

// GetEffectiveRadarStation returns the radar station configured for any candidate location.
func (c *Config) GetEffectiveRadarStation(locs ...string) string {
	for _, l := range locs {
		if l == "" {
			continue
		}
		if s, ok := c.GetLocationSettings(l); ok && s.RadarStation != "" {
			return s.RadarStation
		}
	}
	return ""
}

// Path returns the canonical config file path: ~/.config/wx/config.json.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: home dir: %w", err)
	}
	return filepath.Join(home, ".config", "wx", "config.json"), nil
}

// Load reads the config file and returns the parsed Config.
// If the file does not exist, an empty Config and no error are returned.
// If the file exists but is malformed, an error is returned.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return &Config{}, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return &Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return &cfg, nil
}

// Save writes cfg as JSON to path, creating the parent directory if needed.
// The file is written atomically via a temp file + rename.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: mkdir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	data = append(data, '\n')

	// Write to a temp file in the same directory, then rename for atomicity.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json.tmp")
	if err != nil {
		return fmt.Errorf("config: create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // clean up if rename fails

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("config: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("config: rename: %w", err)
	}
	return nil
}

// LoadFrom reads a config file from an explicit path. Used in tests.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return &Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return &cfg, nil
}
