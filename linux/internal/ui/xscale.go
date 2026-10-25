package ui

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CalmXWaylandScale keeps the desk at its logical size on GNOME Wayland.
//
// Fyne falls back to X11 there, then sizes the window from the XWayland
// mode and the panel's millimetres. That mode is the framebuffer (here
// 10240×5760 on a 2560×1440 desktop), so the DPI is several times Xft.dpi
// and the frame opens maximized with a minimum almost as large. Window
// coordinates on this session are already logical pixels. Leaving DPI
// detection on multiplies those pixels again.
//
// FYNE_SCALE and FYNE_DISABLE_DPI_DETECTION win when the user set them.
func CalmXWaylandScale() {
	if os.Getenv("FYNE_SCALE") != "" || os.Getenv("FYNE_DISABLE_DPI_DETECTION") != "" {
		return
	}
	if os.Getenv("WAYLAND_DISPLAY") == "" || os.Getenv("DISPLAY") == "" {
		return
	}
	px, mm, ok := primaryOutput(commandOutput("xrandr", "--query"))
	if !ok || !framebufferDPIExceeds(px, mm, parseXftDPI(commandOutput("xrdb", "-query"))) {
		return
	}
	os.Setenv("FYNE_DISABLE_DPI_DETECTION", "1")
}

// framebufferDPIExceeds reports that an X mode's DPI is well above the
// desktop DPI, which is the XWayland framebuffer sitting on a logical layout.
func framebufferDPIExceeds(widthPx, widthMm, xftDPI int) bool {
	if widthPx <= 0 || widthMm <= 0 {
		return false
	}
	dpi := float64(widthPx) / (float64(widthMm) / 25.4)
	base := float64(xftDPI)
	if base < 48 {
		base = 96
	}
	return dpi > base*1.75
}

var outputSize = regexp.MustCompile(`(?m)^(\S+) connected (primary )?(\d+)x\d+\+\d+\+\d+.* (\d+)mm x`)

// primaryOutput returns the primary connected mode, or the first connected
// output when none is marked primary.
func primaryOutput(xrandr string) (widthPx, widthMm int, ok bool) {
	var fallbackPx, fallbackMm int
	found := false
	for _, match := range outputSize.FindAllStringSubmatch(xrandr, -1) {
		px, pxErr := strconv.Atoi(match[3])
		mm, mmErr := strconv.Atoi(match[4])
		if pxErr != nil || mmErr != nil || px <= 0 || mm <= 0 {
			continue
		}
		if match[2] != "" {
			return px, mm, true
		}
		if !found {
			fallbackPx, fallbackMm = px, mm
			found = true
		}
	}
	return fallbackPx, fallbackMm, found
}

func parseXftDPI(xrdb string) int {
	for _, line := range strings.Split(xrdb, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != "Xft.dpi" {
			continue
		}
		dpi, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || dpi < 1 {
			return 0
		}
		return int(dpi + 0.5)
	}
	return 0
}

func commandOutput(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}
