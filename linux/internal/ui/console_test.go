package ui

import (
	"image"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"

	"github.com/mwirges/wx/linux/internal/fixture"
	"github.com/mwirges/wx/linux/internal/store"
)

func TestConsoleCardKeepsBodyTextAndDrawsBrackets(t *testing.T) {
	application := testApp()
	defer application.Quit()
	card := consoleCard("Atmospheric Telemetry", "GRID.OBS", "kicker", colCyan, container.NewVBox(line("FORT WAYNE, IN", "place")))
	text := collectText(card)
	for _, want := range []string{"ATMOSPHERIC TELEMETRY", "GRID.OBS", "FORT WAYNE, IN"} {
		if !strings.Contains(text, want) {
			t.Fatalf("card %q missing %q", text, want)
		}
	}
	canvas := software.NewCanvas()
	canvas.SetPadded(false)
	canvas.SetContent(card)
	canvas.Resize(fyne.NewSize(320, 140))
	img := canvas.Capture().(*image.NRGBA)
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if !bracketIn(img, 0, 0, 14, 14) || !bracketIn(img, w-14, 0, w, 14) || !bracketIn(img, 0, h-14, 14, h) || !bracketIn(img, w-14, h-14, w, h) {
		t.Fatal("corner brackets missing")
	}
}

func TestModePillSwitchesTheVisiblePage(t *testing.T) {
	application := testApp()
	defer application.Quit()
	st := store.New(&fixture.Fake{}, nil, true)
	st.Payload = fixture.Payload
	desk := NewDesk(application, st, nil)
	defer desk.Window.Close()
	canvas := software.NewCanvas()
	canvas.SetPadded(false)
	canvas.SetContent(desk.Window.Content())
	canvas.Resize(deskOpenSize)
	pill := desk.modes["radar"]
	at, ok := originOf(canvas.Content(), pill)
	if !ok {
		t.Fatal("mode pill is not in the window")
	}
	test.TapCanvas(canvas, at.Add(fyne.NewPos(6, 6)))
	if desk.Visible() != "radar" || st.Tab != "radar" {
		t.Fatalf("visible %s tab %s", desk.Visible(), st.Tab)
	}
}

func TestStatusBadgePaintsTheFullLabel(t *testing.T) {
	application := testApp()
	defer application.Quit()
	badge := statusBadge("ALL CLEAR // DRY", "ok")
	if badge.MinSize().Width < 100 {
		t.Fatalf("badge collapsed to %v", badge.MinSize())
	}
	canvas := software.NewCanvas()
	canvas.SetPadded(false)
	canvas.SetContent(badge)
	canvas.Resize(badge.MinSize().Add(fyne.NewSize(4, 4)))
	img := canvas.Capture().(*image.NRGBA)
	lit := 0
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			c := img.NRGBAAt(x, y)
			if c.G > 80 && c.G > c.R {
				lit++
				break
			}
		}
	}
	if lit < 60 {
		t.Fatalf("badge painted %d columns", lit)
	}
}

func bracketIn(img *image.NRGBA, x0, y0, x1, y1 int) bool {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if !image.Pt(x, y).In(img.Bounds()) {
				continue
			}
			c := img.NRGBAAt(x, y)
			if c.G > 120 && c.B > 150 && c.R < 140 {
				return true
			}
		}
	}
	return false
}
