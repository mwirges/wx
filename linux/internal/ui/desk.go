package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mwirges/wx/linux/internal/present"
	"github.com/mwirges/wx/linux/internal/store"
)

var tabOrder = []struct{ id, title string }{
	{"dual", "TACTICAL"},
	{"weather", "SURFACE"},
	{"radar", "RADAR"},
	{"outlooks", "OUTLOOKS"},
	{"chase", "CHASE"},
	{"climate", "CLIMATE"},
	{"tropics", "TROPICS"},
	{"grid", "GRID"},
}

// Desk is the desk window. Closing it stops radar fetch and the loop.
type Desk struct {
	Window      fyne.Window
	Format      *widget.Select
	Units       *widget.Select
	Status      *widget.Label
	Location    *widget.Entry
	Surfaces    []*DailySurface
	RadarPanels []*RadarPanel
	Dual        *DualPane
	Outlook     *OutlookView
	Chase       *ChaseView
	Climate     *ClimateView
	Tropics     *TropicsView
	Grid        *GridView

	store    *store.Store
	pages    map[string]fyne.CanvasObject
	pageSlot *fyne.Container
	root     *fyne.Container
	visible  string
	building bool
	done     bool
	listenID int
	onClosed func()
}

// NewDesk opens a desk window and paints the current store. It does not fetch.
func NewDesk(app fyne.App, st *store.Store, onClosed func()) *Desk {
	d := &Desk{store: st, onClosed: onClosed, pages: map[string]fyne.CanvasObject{}}
	st.OpenDesk()
	d.Window = app.NewWindow("wx")
	d.Window.Resize(fyne.NewSize(980, 760))
	d.build()
	d.Window.SetContent(d.root)
	d.Window.SetCloseIntercept(func() { d.Window.Close() })
	d.Window.SetOnClosed(func() { d.RequestClose() })
	d.listenID = st.Subscribe(d.onStore)
	d.Refresh()
	d.Window.Show()
	return d
}

// Visible is the page name on screen.
func (d *Desk) Visible() string { return d.visible }

// Find returns a named section from any page, including pages that are not visible.
func (d *Desk) Find(id string) fyne.CanvasObject {
	for _, tab := range tabOrder {
		if found := findID(d.pages[tab.id], id); found != nil {
			return found
		}
	}
	return findID(d.root, id)
}

// SectionText is the text inside a named section.
func (d *Desk) SectionText(id string) string {
	return collectText(d.Find(id))
}

// TapMode switches pages the way the mode button does.
func (d *Desk) TapMode(tab string) {
	d.store.SetTab(tab)
}

// RequestClose stops radar and the loop, then tells the shell the window is gone.
// A second call does nothing. The tray refresh is left running.
func (d *Desk) RequestClose() {
	if d.done {
		return
	}
	d.done = true
	d.store.Unsubscribe(d.listenID)
	d.store.CloseDesk()
	if d.onClosed != nil {
		cb := d.onClosed
		d.onClosed = nil
		cb()
	}
}

// Refresh paints every page from the store.
func (d *Desk) Refresh() {
	if d.Surfaces == nil {
		return
	}
	d.building = true
	defer func() { d.building = false }()
	for _, surface := range d.Surfaces {
		surface.Refresh(d.store)
	}
	for _, panel := range d.RadarPanels {
		panel.Refresh(d.store)
	}
	d.Outlook.Refresh(d.store)
	d.Chase.Refresh(d.store)
	d.Climate.Refresh(d.store)
	d.Tropics.Refresh(d.store)
	d.Grid.Refresh(d.store)
	text := present.StatusText(d.store.Payload, d.store.Units, d.store.MenuBarFormat, d.store.Error)
	place := present.AsString(present.Conditions(d.store.Payload)["location"])
	if place == "" {
		place = d.store.Location
	}
	if place == "" {
		place = "AUTO"
	}
	d.Status.SetText(place + "   " + text)
	if label := present.FormatLabels[d.store.MenuBarFormat]; d.Format.Selected != label {
		d.Format.SetSelected(label)
	}
	unitsLabel := "Imperial"
	if d.store.Units == "metric" {
		unitsLabel = "Metric"
	}
	if d.Units.Selected != unitsLabel {
		d.Units.SetSelected(unitsLabel)
	}
	d.showTab(d.store.Tab)
	if d.Window != nil && d.Window.Canvas().Size().Width > 1 {
		d.Dual.ApplyWidth(d.Window.Canvas().Size().Width)
	}
}

func (d *Desk) onStore(topic string) {
	if d.done {
		return
	}
	if topic == "tab" {
		d.showTab(d.store.Tab)
	}
	d.Refresh()
}

func (d *Desk) build() {
	d.building = true
	defer func() { d.building = false }()
	d.Location = widget.NewEntry()
	d.Location.SetPlaceHolder("City, ST  or  zip")
	d.Location.SetText(d.store.Location)
	d.Location.OnSubmitted = func(text string) { d.store.SelectLocation(text) }

	d.Format = widget.NewSelect(formatLabels(), nil)
	d.Format.SetSelected(present.FormatLabels[d.store.MenuBarFormat])
	d.Format.OnChanged = d.onFormat

	d.Units = widget.NewSelect([]string{"Imperial", "Metric"}, nil)
	if d.store.Units == "metric" {
		d.Units.SetSelected("Metric")
	} else {
		d.Units.SetSelected("Imperial")
	}
	d.Units.OnChanged = d.onUnits

	refresh := widget.NewButton("Refresh", func() { d.store.Refresh() })
	controls := container.NewHBox(d.Format, d.Units, refresh)
	title := line("ATMOSPHERIC TELEMETRY CONSOLE", "kicker")
	title.Wrapping = fyne.TextWrapOff
	title.Truncation = fyne.TextTruncateEllipsis
	header := container.NewVBox(title, container.NewBorder(nil, nil, nil, controls, d.Location))

	d.Status = widget.NewLabel("")
	d.Status.Wrapping = fyne.TextWrapWord

	d.Surfaces = []*DailySurface{NewDailySurface(), NewDailySurface()}
	d.RadarPanels = []*RadarPanel{NewRadarPanel(d.store, d.Window), NewRadarPanel(d.store, d.Window)}
	d.Dual = NewDualPane(d.Surfaces[0].Root, d.RadarPanels[0].Root)
	d.Outlook = NewOutlookView()
	d.Chase = NewChaseView(d.store)
	d.Climate = NewClimateView()
	d.Tropics = NewTropicsView()
	d.Grid = NewGridView(d.store)
	d.pages["dual"] = d.Dual
	d.pages["weather"] = d.Surfaces[1].Root
	d.pages["radar"] = d.RadarPanels[1].Root
	d.pages["outlooks"] = d.Outlook.Root
	d.pages["chase"] = d.Chase.Root
	d.pages["climate"] = d.Climate.Root
	d.pages["tropics"] = d.Tropics.Root
	d.pages["grid"] = d.Grid.Root

	modes := container.NewHBox()
	for _, tab := range tabOrder {
		name := tab.id
		btn := widget.NewButton(tab.title, func() { d.store.SetTab(name) })
		modes.Add(newSection("wx.mode."+name, btn))
	}
	d.pageSlot = container.NewStack()
	d.showTab(d.store.Tab)
	d.root = container.NewBorder(container.NewVBox(header, newSection("wx.status", d.Status), modes), nil, nil, nil, d.pageSlot)
}

func (d *Desk) showTab(tab string) {
	page, ok := d.pages[tab]
	if !ok {
		tab = "dual"
		page = d.pages[tab]
	}
	d.visible = tab
	shown := page
	if tab != "dual" {
		shown = container.NewScroll(page)
	}
	d.pageSlot.Objects = []fyne.CanvasObject{shown}
	d.pageSlot.Refresh()
}

func (d *Desk) onFormat(label string) {
	if d.building {
		return
	}
	for _, key := range present.Formats {
		if present.FormatLabels[key] == label {
			d.store.SetMenuBarFormat(key)
			return
		}
	}
}

func (d *Desk) onUnits(label string) {
	if d.building {
		return
	}
	units := "imperial"
	if label == "Metric" {
		units = "metric"
	}
	if units != d.store.Units {
		d.store.SetUnits(units)
	}
}

func formatLabels() []string {
	out := make([]string, len(present.Formats))
	for i, key := range present.Formats {
		out[i] = present.FormatLabels[key]
	}
	return out
}
