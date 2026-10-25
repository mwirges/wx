package ui

import "testing"

func TestFramebufferDPIExceedsLogicalDesktop(t *testing.T) {
	// 10240 px across 600 mm is the XWayland framebuffer on a 27" 1440p panel.
	// Xft.dpi 96 is the logical desktop. Fyne must not scale from the framebuffer.
	if !framebufferDPIExceeds(10240, 600, 96) {
		t.Fatal("framebuffer DPI was treated as the desktop")
	}
	// The mirrored output is 5120 px on the same panel. Still a lie.
	if !framebufferDPIExceeds(5120, 600, 96) {
		t.Fatal("half framebuffer was treated as the desktop")
	}
	// A 1920×1080 panel about 530 mm wide matches Xft.dpi. Leave Fyne alone.
	if framebufferDPIExceeds(1920, 530, 96) {
		t.Fatal("a normal panel was treated as a framebuffer")
	}
	// A sharp laptop whose Xft.dpi already says so is not the XWayland lie.
	if framebufferDPIExceeds(3840, 340, 192) {
		t.Fatal("a hi-dpi panel with a matching Xft.dpi was overridden")
	}
	if framebufferDPIExceeds(0, 600, 96) || framebufferDPIExceeds(1920, 0, 96) {
		t.Fatal("missing geometry was treated as a framebuffer")
	}
}

func TestPrimaryOutputPrefersThePrimaryMode(t *testing.T) {
	text := "" +
		"Screen 0: minimum 16 x 16, current 10240 x 5760\n" +
		"HDMI-1 connected primary 10240x5760+0+0 (normal left inverted right x axis y axis) 600mm x 340mm\n" +
		"DP-4 connected 5120x2880+0+0 (normal left inverted right x axis y axis) 600mm x 340mm\n"
	px, mm, ok := primaryOutput(text)
	if !ok || px != 10240 || mm != 600 {
		t.Fatalf("primary = %d px %d mm ok=%v", px, mm, ok)
	}
}

func TestPrimaryOutputFallsBackToTheFirstConnected(t *testing.T) {
	text := "DP-4 connected 5120x2880+0+0 (normal) 600mm x 340mm\n"
	px, mm, ok := primaryOutput(text)
	if !ok || px != 5120 || mm != 600 {
		t.Fatalf("fallback = %d px %d mm ok=%v", px, mm, ok)
	}
	if _, _, ok := primaryOutput("Screen 0: current 10240 x 5760\n"); ok {
		t.Fatal("a screen line was parsed as an output")
	}
}

func TestParseXftDPI(t *testing.T) {
	if got := parseXftDPI("Xft.dpi:\t96\nXcursor.size:\t24\n"); got != 96 {
		t.Fatalf("dpi %d", got)
	}
	if got := parseXftDPI("Xft.dpi: 192.0\n"); got != 192 {
		t.Fatalf("dpi %d", got)
	}
	if got := parseXftDPI(""); got != 0 {
		t.Fatalf("dpi %d", got)
	}
}
