package monitor

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// ── padToWidth ────────────────────────────────────────────────────────────────

func TestPadToWidth_Plain(t *testing.T) {
	s := padToWidth("hello", 10)
	vis := lipgloss.Width(s)
	if vis != 10 {
		t.Errorf("padToWidth visual width = %d, want 10", vis)
	}
	if !strings.HasPrefix(s, "hello") {
		t.Errorf("padToWidth should preserve original content, got %q", s)
	}
}

func TestPadToWidth_AlreadyWide(t *testing.T) {
	s := padToWidth("hello world", 5)
	// Should not truncate — returns as-is when already >= w
	if s != "hello world" {
		t.Errorf("padToWidth should not truncate, got %q", s)
	}
}

func TestPadToWidth_AnsiString(t *testing.T) {
	// lipgloss-styled string: visual width != byte length
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("hi")
	padded := padToWidth(styled, 10)
	vis := lipgloss.Width(padded)
	if vis != 10 {
		t.Errorf("ANSI padToWidth visual width = %d, want 10", vis)
	}
}

// ── normalizeLines ────────────────────────────────────────────────────────────

func TestNormalizeLines_Pad(t *testing.T) {
	lines := []string{"a", "b"}
	result := normalizeLines(lines, 5, 10)
	if len(result) != 5 {
		t.Errorf("len = %d, want 5", len(result))
	}
}

func TestNormalizeLines_Truncate(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	result := normalizeLines(lines, 3, 10)
	if len(result) != 3 {
		t.Errorf("len = %d, want 3", len(result))
	}
}

func TestNormalizeLines_Exact(t *testing.T) {
	lines := []string{"a", "b", "c"}
	result := normalizeLines(lines, 3, 10)
	if len(result) != 3 {
		t.Errorf("len = %d, want 3", len(result))
	}
}

// ── truncateStr ───────────────────────────────────────────────────────────────

func TestTruncateStr_ShortString(t *testing.T) {
	s := truncateStr("hello", 20)
	if s != "hello" {
		t.Errorf("truncateStr should not modify short string, got %q", s)
	}
}

func TestTruncateStr_LongString(t *testing.T) {
	long := strings.Repeat("x", 100)
	result := truncateStr(long, 10)
	if lipgloss.Width(result) > 10 {
		t.Errorf("truncateStr result width = %d, want <= 10", lipgloss.Width(result))
	}
}

// ── renderScrollHeader ────────────────────────────────────────────────────────

func TestRenderScrollHeader_NoScroll(t *testing.T) {
	// All periods fit: no arrows
	s := renderScrollHeader(0, 3, 5, 40)
	if strings.Contains(s, "▲") || strings.Contains(s, "▼") {
		t.Errorf("no-scroll header should have no arrows, got %q", s)
	}
}

func TestRenderScrollHeader_CanScrollDown(t *testing.T) {
	// More periods than visible, at top
	s := renderScrollHeader(0, 10, 5, 60)
	if !strings.Contains(s, "▼") {
		t.Errorf("expected ▼ arrow when can scroll down, got %q", s)
	}
	if strings.Contains(s, "▲") {
		t.Errorf("should not have ▲ when at top, got %q", s)
	}
}

func TestRenderScrollHeader_CanScrollUp(t *testing.T) {
	// More periods than visible, scrolled down
	s := renderScrollHeader(5, 10, 5, 60)
	if !strings.Contains(s, "▲") {
		t.Errorf("expected ▲ arrow when can scroll up, got %q", s)
	}
}

func TestRenderScrollHeader_AtBottom(t *testing.T) {
	// Scrolled to the bottom
	s := renderScrollHeader(5, 10, 5, 60)
	// offset(5) + visible(5) = 10 = total → no down arrow
	if strings.Contains(s, "▼") {
		t.Errorf("should not have ▼ when at bottom, got %q", s)
	}
}

// ── View ─────────────────────────────────────────────────────────────────────

func TestView_ZeroSizeReturnsLoading(t *testing.T) {
	m := New(MonitorConfig{Imperial: true}, testLoc())
	// width/height = 0 → loading message
	v := m.View()
	if v == "" {
		t.Error("View should return loading message when size is 0")
	}
}

func TestView_WithSize(t *testing.T) {
	m := New(MonitorConfig{Imperial: true}, testLoc())
	m.width = 100
	m.height = 30

	v := m.View()
	if v == "" {
		t.Error("View should return non-empty string with valid size")
	}
	// Should contain the location name
	if !strings.Contains(v, "Kansas City, MO") {
		t.Errorf("View should contain location name, got:\n%s", v)
	}
}

func TestView_HelpBar_Normal(t *testing.T) {
	m := New(MonitorConfig{Imperial: true}, testLoc())
	m.width = 100
	m.height = 30

	v := m.renderHelpBar()
	if !strings.Contains(v, "refresh") {
		t.Errorf("help bar should contain 'refresh', got %q", v)
	}
	if !strings.Contains(v, "quit") {
		t.Errorf("help bar should contain 'quit', got %q", v)
	}
}

func TestView_HelpBar_InputMode(t *testing.T) {
	m := New(MonitorConfig{Imperial: true}, testLoc())
	m.inputMode = true

	v := m.renderHelpBar()
	if !strings.Contains(v, "confirm") {
		t.Errorf("input mode help bar should contain 'confirm', got %q", v)
	}
	if !strings.Contains(v, "cancel") {
		t.Errorf("input mode help bar should contain 'cancel', got %q", v)
	}
}

// ── max/min ───────────────────────────────────────────────────────────────────

func TestMax(t *testing.T) {
	if max(3, 5) != 5 {
		t.Error("max(3,5) should be 5")
	}
	if max(7, 2) != 7 {
		t.Error("max(7,2) should be 7")
	}
}

func TestMin(t *testing.T) {
	if min(3, 5) != 3 {
		t.Error("min(3,5) should be 3")
	}
	if min(7, 2) != 2 {
		t.Error("min(7,2) should be 2")
	}
}
