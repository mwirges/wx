// Package packaging stages a Debian root for the wx desk and CLI.
package packaging

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// Icon sizes installed under /usr/share/icons/hicolor.
var iconSizes = []int{16, 24, 32, 48, 64, 128, 256, 512}

// Options selects the icon, the template directory, and the package root.
type Options struct {
	Icon      string
	Templates string
	Dest      string
	Bins      string
	Version   string
	Arch      string
	Date      string
}

// Stage writes Dest as a dpkg-deb root: binaries, icon, desktop entry, and control files.
func Stage(opts Options) error {
	if opts.Version == "" || opts.Arch == "" || opts.Dest == "" {
		return fmt.Errorf("version, arch, and dest are required")
	}
	if err := os.RemoveAll(opts.Dest); err != nil {
		return err
	}
	if err := os.MkdirAll(opts.Dest, 0o755); err != nil {
		return err
	}
	if err := installBins(opts); err != nil {
		return err
	}
	if err := installIcons(opts); err != nil {
		return err
	}
	if err := installMeta(opts); err != nil {
		return err
	}
	return writeControl(opts)
}

func installBins(opts Options) error {
	dir := filepath.Join(opts.Dest, "usr", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"wx", "wx-desk"} {
		src := filepath.Join(opts.Bins, name)
		dst := filepath.Join(dir, name)
		if err := copyFile(src, dst, 0o755); err != nil {
			return err
		}
		if strip, err := exec.LookPath("strip"); err == nil {
			_ = exec.Command(strip, "--strip-unneeded", dst).Run()
		}
	}
	return nil
}

func installIcons(opts Options) error {
	f, err := os.Open(opts.Icon)
	if err != nil {
		return err
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return err
	}
	for _, size := range iconSizes {
		dir := filepath.Join(opts.Dest, "usr", "share", "icons", "hicolor", fmt.Sprintf("%dx%d", size, size), "apps")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		out, err := os.Create(filepath.Join(dir, "wx.png"))
		if err != nil {
			return err
		}
		err = png.Encode(out, scale(src, size))
		closeErr := out.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if err := os.Chmod(out.Name(), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func scale(src image.Image, size int) image.Image {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func installMeta(opts Options) error {
	apps := filepath.Join(opts.Dest, "usr", "share", "applications")
	if err := os.MkdirAll(apps, 0o755); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(opts.Templates, "wx-desk.desktop"), filepath.Join(apps, "wx-desk.desktop"), 0o644); err != nil {
		return err
	}
	meta := filepath.Join(opts.Dest, "usr", "share", "metainfo")
	if err := os.MkdirAll(meta, 0o755); err != nil {
		return err
	}
	body, err := os.ReadFile(filepath.Join(opts.Templates, "wx-desk.metainfo.xml"))
	if err != nil {
		return err
	}
	text := strings.ReplaceAll(string(body), "@VERSION@", opts.Version)
	text = strings.ReplaceAll(text, "@DATE@", opts.Date)
	metaPath := filepath.Join(meta, "com.github.mwirges.wx.desk.metainfo.xml")
	if err := os.WriteFile(metaPath, []byte(text), 0o644); err != nil {
		return err
	}
	deb := filepath.Join(opts.Dest, "DEBIAN")
	if err := os.MkdirAll(deb, 0o755); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(opts.Templates, "postinst"), filepath.Join(deb, "postinst"), 0o755); err != nil {
		return err
	}
	return copyFile(filepath.Join(opts.Templates, "postrm"), filepath.Join(deb, "postrm"), 0o755)
}

func writeControl(opts Options) error {
	size := installedKiB(filepath.Join(opts.Dest, "usr"))
	control := fmt.Sprintf(`Package: wx
Version: %s
Section: utils
Priority: optional
Architecture: %s
Maintainer: Matt Wirges <mwirges@gmail.com>
Depends: libc6, libffi8, libgl1, libwayland-client0
Homepage: https://github.com/mwirges/wx
Installed-Size: %d
Description: weather desk and command line
 wx shows current conditions, forecasts, and radar.
 The Linux desk is a window and tray over the wx command.
 Weather data stays in that command.
`, opts.Version, opts.Arch, size)
	return os.WriteFile(filepath.Join(opts.Dest, "DEBIAN", "control"), []byte(control), 0o644)
}

func installedKiB(root string) int64 {
	var n int64
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && info.Mode().IsRegular() {
			n += info.Size()
		}
		return nil
	})
	return (n + 1023) / 1024
}

func copyFile(src, dst string, mode os.FileMode) error {
	body, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, body, mode)
}
