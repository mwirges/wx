package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// DualPane places radar beside telemetry at 760 px and wider, and stacks them below that.
type DualPane struct {
	widget.BaseWidget
	Telemetry  fyne.CanvasObject
	Radar      fyne.CanvasObject
	split      *container.Split
	horizontal bool
}

// NewDualPane builds a horizontal split. ApplyWidth flips it under 760 px.
func NewDualPane(telemetry, radar fyne.CanvasObject) *DualPane {
	d := &DualPane{Telemetry: telemetry, Radar: radar, horizontal: true}
	// Clip each side. Fyne does not clip a split child, so a wide label paints
	// across the divider into the other pane.
	d.split = container.NewHSplit(
		container.NewClip(container.NewVScroll(telemetry)),
		container.NewClip(radar),
	)
	d.split.Offset = 0.42
	d.ExtendBaseWidget(d)
	return d
}

// Horizontal reports the current split direction.
func (d *DualPane) Horizontal() bool { return d.horizontal }

// ApplyWidth keeps the radar beside telemetry until the desk is narrower than 760 px.
func (d *DualPane) ApplyWidth(width float32) {
	wide := width >= 760
	if wide == d.horizontal {
		return
	}
	d.horizontal = wide
	d.split.Horizontal = wide
	d.split.Refresh()
}

// CreateRenderer lays the split out and updates its direction from the width.
func (d *DualPane) CreateRenderer() fyne.WidgetRenderer {
	return &dualRenderer{pane: d}
}

type dualRenderer struct {
	pane *DualPane
}

func (r *dualRenderer) Layout(size fyne.Size) {
	r.pane.ApplyWidth(size.Width)
	r.pane.split.Resize(size)
	r.pane.split.Move(fyne.NewPos(0, 0))
}

func (r *dualRenderer) MinSize() fyne.Size { return fyne.NewSize(320, 240) }

func (r *dualRenderer) Refresh() { r.pane.split.Refresh() }

func (r *dualRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.pane.split} }

func (r *dualRenderer) Destroy() {}
