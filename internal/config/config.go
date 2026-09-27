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

	// Notifications enables or disables desktop notifications for active alerts.
	Notifications *bool `json:"notifications,omitempty"`

	// Favorites is a list of named locations/aliases (e.g. Home, Work, Cabin).
	Favorites []LocationEntry `json:"favorites,omitempty"`

	// RecentLocations tracks the most recently queried locations (capped at 10).
	RecentLocations []string `json:"recent_locations,omitempty"`
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
