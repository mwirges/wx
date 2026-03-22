package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds user-level preferences loaded from ~/.config/wx/config.json.
// All fields are optional; missing fields leave the app using its built-in defaults.
type Config struct {
	// DefaultLocation is used when --location is not passed on the command line.
	// Accepts the same formats as --location: zip code, "City, ST", or empty to
	// fall back to IP-based auto-detection.
	DefaultLocation string `json:"default_location"`

	// Units sets the default display units: "imperial" or "metric".
	// Overridden by --units on the command line.
	Units string `json:"units"`
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
