package radar

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInteractiveModel_Init(t *testing.T) {
	m := NewInteractiveModel(InteractiveConfig{
		Product:   ProductCompositeReflectivity,
		RadiusKM:  200,
		NumFrames: 6,
	})
	if m.product != ProductCompositeReflectivity {
		t.Errorf("product = %q, want composite-reflectivity", m.product)
	}
	if m.radius != 200 {
		t.Errorf("radius = %v, want 200", m.radius)
	}
}

func TestInteractiveModel_ProductCycle(t *testing.T) {
	m := NewInteractiveModel(InteractiveConfig{
		Product:  ProductCompositeReflectivity,
		RadiusKM: 200,
	})
	m.width, m.height = 80, 24

	// Simulate pressing 'p' — should cycle to base reflectivity.
	// The Update returns a fetch command; we just check the model state.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m2 := updated.(InteractiveModel)
	if m2.product != ProductBaseReflectivity {
		t.Errorf("after 'p': product = %q, want base-reflectivity", m2.product)
	}
}

func TestInteractiveModel_RadiusZoom(t *testing.T) {
	m := NewInteractiveModel(InteractiveConfig{
		Product:  ProductCompositeReflectivity,
		RadiusKM: 200,
	})
	m.width, m.height = 80, 24

	// '+' should zoom in (smaller radius).
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	m2 := updated.(InteractiveModel)
	if m2.radius >= 200 {
		t.Errorf("after '+': radius = %v, want < 200 (zoom in)", m2.radius)
	}

	// '-' should zoom out (larger radius).
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	m3 := updated.(InteractiveModel)
	if m3.radius <= m2.radius {
		t.Errorf("after '-': radius = %v, want > %v (zoom out)", m3.radius, m2.radius)
	}
}

func TestInteractiveModel_LoopToggle(t *testing.T) {
	m := NewInteractiveModel(InteractiveConfig{
		Product:  ProductCompositeReflectivity,
		RadiusKM: 200,
	})
	m.width, m.height = 80, 24

	if m.loopMode {
		t.Error("loop should be off initially")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m2 := updated.(InteractiveModel)
	if !m2.loopMode {
		t.Error("after 'l': loop should be on")
	}

	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m3 := updated.(InteractiveModel)
	if m3.loopMode {
		t.Error("after second 'l': loop should be off")
	}
}

func TestInteractiveModel_Quit(t *testing.T) {
	m := NewInteractiveModel(InteractiveConfig{
		Product:  ProductCompositeReflectivity,
		RadiusKM: 200,
	})
	m.width, m.height = 80, 24

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m2 := updated.(InteractiveModel)
	if !m2.quitting {
		t.Error("should be quitting after 'q'")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command")
	}
}

func TestInteractiveModel_PauseResume(t *testing.T) {
	m := NewInteractiveModel(InteractiveConfig{
		Product:  ProductCompositeReflectivity,
		RadiusKM: 200,
	})
	m.width, m.height = 80, 24

	// Enable loop mode first.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m2 := updated.(InteractiveModel)
	if !m2.loopMode {
		t.Fatal("expected loop mode on")
	}
	if m2.paused {
		t.Error("should not be paused initially")
	}

	// Space → pause.
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m3 := updated.(InteractiveModel)
	if !m3.paused {
		t.Error("after space: should be paused")
	}

	// Space again → resume.
	updated, _ = m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m4 := updated.(InteractiveModel)
	if m4.paused {
		t.Error("after second space: should be resumed")
	}
}

func TestInteractiveModel_SpaceNoOpOutsideLoop(t *testing.T) {
	m := NewInteractiveModel(InteractiveConfig{
		Product:  ProductCompositeReflectivity,
		RadiusKM: 200,
	})
	m.width, m.height = 80, 24

	// Space outside loop mode should not toggle pause.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m2 := updated.(InteractiveModel)
	if m2.paused {
		t.Error("space outside loop mode should not set paused")
	}
}

func TestNextRadius(t *testing.T) {
	// Zoom in from 200 → 150
	if got := nextRadius(200, -1); got != 150 {
		t.Errorf("nextRadius(200, -1) = %v, want 150", got)
	}
	// Zoom out from 200 → 300
	if got := nextRadius(200, 1); got != 300 {
		t.Errorf("nextRadius(200, 1) = %v, want 300", got)
	}
	// At minimum, stay
	if got := nextRadius(50, -1); got != 50 {
		t.Errorf("nextRadius(50, -1) = %v, want 50", got)
	}
	// At maximum, stay
	if got := nextRadius(500, 1); got != 500 {
		t.Errorf("nextRadius(500, 1) = %v, want 500", got)
	}
}

func TestNextProduct(t *testing.T) {
	got := nextProduct(ProductCompositeReflectivity)
	if got != ProductBaseReflectivity {
		t.Errorf("nextProduct(CR) = %q, want BR", got)
	}
	got = nextProduct(ProductBaseReflectivity)
	if got != ProductBaseVelocity {
		t.Errorf("nextProduct(BR) = %q, want BV", got)
	}
	got = nextProduct(ProductBaseVelocity)
	if got != ProductStormRelativeVelocity {
		t.Errorf("nextProduct(BV) = %q, want SRV", got)
	}
	got = nextProduct(ProductStormRelativeVelocity)
	if got != ProductEchoTops {
		t.Errorf("nextProduct(SRV) = %q, want ET", got)
	}
	// Echo tops wraps back to composite reflectivity.
	got = nextProduct(ProductEchoTops)
	if got != ProductCompositeReflectivity {
		t.Errorf("nextProduct(ET) = %q, want CR", got)
	}
}
