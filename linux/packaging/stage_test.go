package packaging

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStageInstallsDeskAndIcon(t *testing.T) {
	root := t.TempDir()
	icon := filepath.Join(root, "icon.png")
	writePNG(t, icon, 64)
	bins := filepath.Join(root, "bins")
	if err := os.Mkdir(bins, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"wx", "wx-desk"} {
		if err := os.WriteFile(filepath.Join(bins, name), []byte(name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	dest := filepath.Join(root, "stage")
	err := Stage(Options{
		Icon:      icon,
		Templates: templatesDir(t),
		Dest:      dest,
		Bins:      bins,
		Version:   "1.2.3",
		Arch:      "amd64",
		Date:      "2026-10-11",
	})
	if err != nil {
		t.Fatal(err)
	}
	desk := string(mustRead(t, filepath.Join(dest, "usr", "share", "applications", "wx-desk.desktop")))
	for _, want := range []string{"Name=wx", "Exec=wx-desk", "Icon=wx", "Type=Application", "StartupWMClass=wx-desk"} {
		if !strings.Contains(desk, want) {
			t.Fatalf("desktop missing %s\n%s", want, desk)
		}
	}
	info := string(mustRead(t, filepath.Join(dest, "usr", "share", "metainfo", "com.github.mwirges.wx.desk.metainfo.xml")))
	if !strings.Contains(info, `version="1.2.3"`) || !strings.Contains(info, `date="2026-10-11"`) {
		t.Fatalf("metainfo %s", info)
	}
	control := string(mustRead(t, filepath.Join(dest, "DEBIAN", "control")))
	for _, want := range []string{"Package: wx", "Version: 1.2.3", "Architecture: amd64", "libgl1"} {
		if !strings.Contains(control, want) {
			t.Fatalf("control missing %s\n%s", want, control)
		}
	}
	pngPath := filepath.Join(dest, "usr", "share", "icons", "hicolor", "48x48", "apps", "wx.png")
	f, err := os.Open(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 48 || img.Bounds().Dy() != 48 {
		t.Fatalf("icon %v", img.Bounds())
	}
	infoBin, err := os.Stat(filepath.Join(dest, "usr", "bin", "wx-desk"))
	if err != nil {
		t.Fatal(err)
	}
	if infoBin.Mode().Perm()&0o111 == 0 {
		t.Fatalf("wx-desk mode %s", infoBin.Mode())
	}
}

func templatesDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

func writePNG(t *testing.T, path string, size int) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
