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
