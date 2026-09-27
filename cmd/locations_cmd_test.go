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
