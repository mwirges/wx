package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mwirges/wx/internal/config"
)

func TestLocationsCommand_AddListRemoveRecents(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	app := NewApp()

	// 1. Initially empty
	out := captureStdout(t, func() {
		err := app.Run([]string{"wx", "locations", "list"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(string(out), "No favorite locations or recent searches") {
		t.Errorf("expected empty message, got: %s", string(out))
	}

	// 2. Add favorite "Home" -> "Fort Wayne, IN"
	out = captureStdout(t, func() {
		err := app.Run([]string{"wx", "locations", "add", "Home", "Fort Wayne, IN"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(string(out), "Saved favorite:") || !strings.Contains(string(out), "Fort Wayne, IN") {
		t.Errorf("expected saved message, got: %s", string(out))
	}

	// 3. Add single-arg favorite "Chicago, IL"
	out = captureStdout(t, func() {
		err := app.Run([]string{"wx", "locations", "add", "Chicago, IL"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(string(out), "Saved favorite:") || !strings.Contains(string(out), "Chicago, IL") {
		t.Errorf("expected saved message, got: %s", string(out))
	}

	// 4. Verify config file was written
	cfgPath := filepath.Join(tempHome, ".config", "wx", "config.json")
	cfg, err := config.LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}
	if len(cfg.Favorites) != 2 {
		t.Fatalf("expected 2 favorites, got %d", len(cfg.Favorites))
	}
	val, ok := cfg.GetFavorite("home")
	if !ok || val != "Fort Wayne, IN" {
		t.Errorf("GetFavorite('home') = %q, %v", val, ok)
	}

	// 5. List favorites
	out = captureStdout(t, func() {
		err := app.Run([]string{"wx", "locations"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	outStr := string(out)
	if !strings.Contains(outStr, "Home") || !strings.Contains(outStr, "Fort Wayne, IN") {
		t.Errorf("expected Home in locations list, got: %s", outStr)
	}
	if !strings.Contains(outStr, "Chicago, IL") {
		t.Errorf("expected Chicago, IL in locations list, got: %s", outStr)
	}

	// 6. Test recents
	cfg.AddRecent("Denver, CO")
	cfg.AddRecent("Austin, TX")
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("failed to save recents: %v", err)
	}

	out = captureStdout(t, func() {
		err := app.Run([]string{"wx", "locations", "recents"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	outStr = string(out)
	if !strings.Contains(outStr, "Austin, TX") || !strings.Contains(outStr, "Denver, CO") {
		t.Errorf("expected Austin and Denver in recents, got: %s", outStr)
	}

	// 7. Clear recents
	out = captureStdout(t, func() {
		err := app.Run([]string{"wx", "locations", "clear-recents"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(string(out), "Cleared recent locations history") {
		t.Errorf("expected cleared message, got: %s", string(out))
	}
	cfg, _ = config.LoadFrom(cfgPath)
	if len(cfg.RecentLocations) != 0 {
		t.Errorf("expected 0 recents after clear, got %d", len(cfg.RecentLocations))
	}

	// 8. Remove favorite
	out = captureStdout(t, func() {
		err := app.Run([]string{"wx", "locations", "rm", "Home"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(string(out), "Removed favorite:") {
		t.Errorf("expected removed message, got: %s", string(out))
	}

	// 9. Remove nonexistent returns error
	err = app.Run([]string{"wx", "locations", "rm", "Nonexistent"})
	if err == nil {
		t.Errorf("expected error when removing nonexistent favorite, got nil")
	}
}

func TestConfigShow_IncludesFavoritesAndRecents(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	cfgPath := filepath.Join(tempHome, ".config", "wx", "config.json")
	cfg := &config.Config{
		DefaultLocation: "Fort Wayne, IN",
		Units:           "imperial",
	}
	cfg.SetFavorite("Cabin", "Traverse City, MI")
	cfg.AddRecent("Seattle, WA")
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	app := NewApp()
	out := captureStdout(t, func() {
		err := app.Run([]string{"wx", "config", "show"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	outStr := string(out)
	if !strings.Contains(outStr, "Favorites:") || !strings.Contains(outStr, "Cabin") || !strings.Contains(outStr, "Traverse City, MI") {
		t.Errorf("expected Favorites section in config show, got: %s", outStr)
	}
	if !strings.Contains(outStr, "Recent Locations:") || !strings.Contains(outStr, "Seattle, WA") {
		t.Errorf("expected Recent Locations section in config show, got: %s", outStr)
	}
}

func TestLocationsCommand_PerLocationConfig(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	app := NewApp()

	// 1. Add favorite with per-location flags
	out := captureStdout(t, func() {
		err := app.Run([]string{
			"wx", "locations", "add",
			"--units", "metric",
			"--radar-product", "base-reflectivity",
			"--radar-radius", "150",
			"--station", "KAMX",
			"Beach", "Miami, FL",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(string(out), "Saved favorite:") {
		t.Errorf("expected saved message, got: %s", string(out))
	}

	// 1b. Add favorite with trailing flags
	captureStdout(t, func() {
		err := app.Run([]string{
			"wx", "locations", "add", "Lake", "Lake Tahoe, CA",
			"--units", "metric",
			"--radar-product", "storm-relative-velocity",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	cfgPath := filepath.Join(tempHome, ".config", "wx", "config.json")
	cfg, err := config.LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	s, ok := cfg.GetLocationSettings("Beach")
	if !ok {
		t.Fatalf("expected location settings for Beach")
	}
	if s.Units != "metric" || s.DefaultRadarProduct != "base-reflectivity" || s.DefaultRadarRadius != 150 || s.RadarStation != "KAMX" {
		t.Errorf("unexpected settings: %+v", s)
	}

	// 2. List locations shows per-location details
	out = captureStdout(t, func() {
		err := app.Run([]string{"wx", "locations", "list"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	outStr := string(out)
	if !strings.Contains(outStr, "Beach") || !strings.Contains(outStr, "units: metric") || !strings.Contains(outStr, "radar: base-reflectivity (150km)") || !strings.Contains(outStr, "station: KAMX") {
		t.Errorf("locations list should display per-location details, got: %s", outStr)
	}

	// 3. Show config via `wx locations config Beach`
	out = captureStdout(t, func() {
		err := app.Run([]string{"wx", "locations", "config", "Beach"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	outStr = string(out)
	if !strings.Contains(outStr, "Settings for") || !strings.Contains(outStr, "metric") {
		t.Errorf("expected settings display, got: %s", outStr)
	}

	// 4. Update settings via `wx locations config Beach --units imperial --radar-radius 200`
	out = captureStdout(t, func() {
		err := app.Run([]string{"wx", "locations", "config", "Beach", "--units", "imperial", "--radar-radius", "200"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(string(out), "Saved:") {
		t.Errorf("expected saved message, got: %s", string(out))
	}

	cfg, _ = config.LoadFrom(cfgPath)
	s, _ = cfg.GetLocationSettings("Beach")
	if s.Units != "imperial" || s.DefaultRadarRadius != 200 || s.DefaultRadarProduct != "base-reflectivity" {
		t.Errorf("expected updated settings, got: %+v", s)
	}

	// 5. Validation errors
	if err := app.Run([]string{"wx", "locations", "config", "Beach", "--units", "kelvin"}); err == nil {
		t.Errorf("expected error for invalid units, got nil")
	}
	if err := app.Run([]string{"wx", "locations", "config", "Beach", "--radar-product", "bogus"}); err == nil {
		t.Errorf("expected error for invalid radar product, got nil")
	}

	// 6. Clear settings
	out = captureStdout(t, func() {
		err := app.Run([]string{"wx", "locations", "config", "Beach", "--clear"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(string(out), "cleared custom settings") {
		t.Errorf("expected cleared message, got: %s", string(out))
	}

	cfg, _ = config.LoadFrom(cfgPath)
	if _, ok := cfg.GetLocationSettings("Beach"); ok {
		t.Errorf("settings for Beach should be cleared")
	}
}
