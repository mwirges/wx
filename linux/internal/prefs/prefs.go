// Package prefs stores desk preferences in the wx config file.
// WX_CONFIG overrides the path. Unknown keys are preserved.
// The wx CLI does not read WX_CONFIG. This package does not store weather.
package prefs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigPath is WX_CONFIG when set, otherwise ~/.config/wx/config.json.
func ConfigPath() string {
	if override := strings.TrimSpace(os.Getenv("WX_CONFIG")); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".config", "wx", "config.json")
	}
	return filepath.Join(home, ".config", "wx", "config.json")
}

// Load reads a config object. A missing or invalid file is an empty object.
func Load(path string) map[string]any {
	if strings.TrimSpace(path) == "" {
		path = ConfigPath()
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{}
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil || data == nil {
		return map[string]any{}
	}
	return data
}

// Save writes the object atomically and keeps every key.
func Save(data map[string]any, path string) error {
	if strings.TrimSpace(path) == "" {
		path = ConfigPath()
	}
	if data == nil {
		data = map[string]any{}
	}
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".config-*.json.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(buf); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// Update loads, mutates, and saves.
func Update(path string, mutate func(map[string]any)) (map[string]any, error) {
	if strings.TrimSpace(path) == "" {
		path = ConfigPath()
	}
	data := Load(path)
	mutate(data)
	if err := Save(data, path); err != nil {
		return nil, err
	}
	return data, nil
}

// Prefs is the desk's view of the config file.
type Prefs struct {
	Path string
}

// New uses path, or ConfigPath when path is empty.
func New(path string) *Prefs {
	if strings.TrimSpace(path) == "" {
		path = ConfigPath()
	}
	return &Prefs{Path: path}
}

// Load reads this file.
func (p *Prefs) Load() map[string]any { return Load(p.Path) }

// SetMenuBarFormat stores compact, standard, or tactical.
func (p *Prefs) SetMenuBarFormat(format string) error {
	_, err := Update(p.Path, func(cfg map[string]any) {
		cfg["menu_bar_format"] = format
	})
	return err
}

// SetLocationAndUnits writes only the fields that are non-empty.
func (p *Prefs) SetLocationAndUnits(location, units string) error {
	_, err := Update(p.Path, func(cfg map[string]any) {
		if location != "" {
			cfg["default_location"] = location
		}
		if units != "" {
			cfg["units"] = units
		}
	})
	return err
}

// AddRecent puts location at the front and keeps ten.
func (p *Prefs) AddRecent(location string) error {
	clean := strings.TrimSpace(location)
	if clean == "" {
		return nil
	}
	_, err := Update(p.Path, func(cfg map[string]any) {
		recents := stringList(cfg["recent_locations"])
		kept := make([]string, 0, len(recents)+1)
		for _, recent := range recents {
			if !strings.EqualFold(recent, clean) {
				kept = append(kept, recent)
			}
		}
		kept = append([]string{clean}, kept...)
		if len(kept) > 10 {
			kept = kept[:10]
		}
		cfg["recent_locations"] = kept
	})
	return err
}

// AddFavorite inserts or replaces a favorite by name.
func (p *Prefs) AddFavorite(name, value string) error {
	cleanName := strings.TrimSpace(name)
	cleanValue := strings.TrimSpace(value)
	if cleanName == "" || cleanValue == "" {
		return nil
	}
	_, err := Update(p.Path, func(cfg map[string]any) {
		favs := favoriteMaps(cfg["favorites"])
		updated := false
		for _, entry := range favs {
			if strings.EqualFold(asString(entry["name"]), cleanName) {
				entry["name"] = cleanName
				entry["value"] = cleanValue
				updated = true
				break
			}
		}
		if !updated {
			favs = append(favs, map[string]any{"name": cleanName, "value": cleanValue})
		}
		cfg["favorites"] = favs
	})
	return err
}

// RemoveFavorite drops a favorite matched by name or value.
func (p *Prefs) RemoveFavorite(nameOrValue string) error {
	key := strings.ToLower(strings.TrimSpace(nameOrValue))
	if key == "" {
		return nil
	}
	_, err := Update(p.Path, func(cfg map[string]any) {
		var favs []map[string]any
		for _, entry := range favoriteMaps(cfg["favorites"]) {
			name := strings.ToLower(asString(entry["name"]))
			value := strings.ToLower(asString(entry["value"]))
			if name == key || value == key {
				continue
			}
			favs = append(favs, entry)
		}
		if favs == nil {
			favs = []map[string]any{}
		}
		cfg["favorites"] = favs
	})
	return err
}

func stringList(v any) []string {
	switch raw := v.(type) {
	case []string:
		return append([]string{}, raw...)
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			out = append(out, asString(item))
		}
		return out
	default:
		return nil
	}
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

func favoriteMaps(v any) []map[string]any {
	switch raw := v.(type) {
	case []map[string]any:
		out := make([]map[string]any, len(raw))
		copy(out, raw)
		return out
	case []any:
		out := make([]map[string]any, 0, len(raw))
		for _, item := range raw {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, entry)
		}
		return out
	default:
		return nil
	}
}
