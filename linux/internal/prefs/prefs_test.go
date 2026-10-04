package prefs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mwirges/wx/linux/internal/fixture"
)

func TestRoundTripPreservesUnknownKeys(t *testing.T) {
	defer fixture.GuardHome(t)()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("WX_CONFIG", path)
	raw := []byte("{\"provider\":\"nws\",\"per_location\":{\"Home\":{\"radar_station\":\"KIWX\"}}}\n")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(path)
	if err := p.SetMenuBarFormat("tactical"); err != nil {
		t.Fatal(err)
	}
	if err := p.AddFavorite("Home", "Fort Wayne, IN"); err != nil {
		t.Fatal(err)
	}
	if err := p.AddRecent("46808"); err != nil {
		t.Fatal(err)
	}
	data := map[string]any{}
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(buf, &data); err != nil {
		t.Fatal(err)
	}
	if data["provider"] != "nws" {
		t.Fatalf("provider %#v", data["provider"])
	}
	per, _ := data["per_location"].(map[string]any)
	home, _ := per["Home"].(map[string]any)
	if home["radar_station"] != "KIWX" {
		t.Fatalf("per_location %#v", per)
	}
	if data["menu_bar_format"] != "tactical" {
		t.Fatalf("format %#v", data["menu_bar_format"])
	}
	favs, _ := data["favorites"].([]any)
	if len(favs) != 1 {
		t.Fatalf("favorites %#v", data["favorites"])
	}
	fav, _ := favs[0].(map[string]any)
	if fav["name"] != "Home" || fav["value"] != "Fort Wayne, IN" {
		t.Fatalf("favorite %#v", fav)
	}
	recents, _ := data["recent_locations"].([]any)
	if len(recents) != 1 || recents[0] != "46808" {
		t.Fatalf("recents %#v", recents)
	}
}

func TestRecentCapAndFavoriteReplace(t *testing.T) {
	defer fixture.GuardHome(t)()
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("WX_CONFIG", path)
	p := New(path)
	for i := 0; i < 12; i++ {
		if err := p.AddRecent("City " + itoa(i)); err != nil {
			t.Fatal(err)
		}
	}
	data := p.Load()
	recents, _ := data["recent_locations"].([]any)
	if len(recents) != 10 {
		t.Fatalf("len %d %#v", len(recents), recents)
	}
	if recents[0] != "City 11" {
		t.Fatalf("newest %#v", recents[0])
	}
	if err := p.AddFavorite("Home", "A"); err != nil {
		t.Fatal(err)
	}
	if err := p.AddFavorite("Home", "B"); err != nil {
		t.Fatal(err)
	}
	favs, _ := p.Load()["favorites"].([]any)
	if len(favs) != 1 {
		t.Fatalf("favorites %#v", favs)
	}
	fav, _ := favs[0].(map[string]any)
	if fav["name"] != "Home" || fav["value"] != "B" {
		t.Fatalf("replaced %#v", fav)
	}
	if err := p.RemoveFavorite("B"); err != nil {
		t.Fatal(err)
	}
	favs, _ = p.Load()["favorites"].([]any)
	if len(favs) != 0 {
		t.Fatalf("removed %#v", favs)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
