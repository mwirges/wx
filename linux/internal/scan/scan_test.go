package scan

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestShellDoesNotCallWeatherProviders(t *testing.T) {
	root := moduleRoot(t)
	banned := []string{"weather.gov", "api.weather.gov", "open-meteo.com", "mesonet.agron", "github.com/mwirges/wx/internal"}
	walkGo(t, root, func(path, text string) {
		for _, token := range banned {
			if strings.Contains(text, token) {
				t.Errorf("%s contains %s", path, token)
			}
		}
	})
}

func TestShellDoesNotCompositeRadarOrLinkGTK(t *testing.T) {
	root := moduleRoot(t)
	var joined strings.Builder
	walkGo(t, root, func(path, text string) {
		joined.WriteString(text)
		joined.WriteByte('\n')
	})
	text := joined.String()
	for _, banned := range []string{"PIL", "ImageMagick", "imageio", "ffmpeg", "wand", "libadwaita", "cairo", "gotk3", "libgtk"} {
		if strings.Contains(text, banned) {
			t.Errorf("shell contains %s", banned)
		}
	}
	if !strings.Contains(text, "radar --save") || !strings.Contains(text, "radar --save-gif") {
		t.Fatal("shell does not call wx radar --save / --save-gif")
	}
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"gtk", "libadwaita", "cairo", "gotk3"} {
		if strings.Contains(string(mod), banned) {
			t.Errorf("go.mod contains %s", banned)
		}
	}
}

func walkGo(t *testing.T, root string, fn func(path, text string)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			base := entry.Name()
			if base == "testdata" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		buf, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fn(path, string(buf))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}
