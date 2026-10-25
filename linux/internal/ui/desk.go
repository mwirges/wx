package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mwirges/wx/linux/internal/present"
	"github.com/mwirges/wx/linux/internal/store"
)

var tabOrder = []struct{ id, title string }{
	{"dual", "TACTICAL"},
	{"weather", "SURFACE"},
	{"radar", "RADAR"},
	{"outlooks", "CPC OUTLOOKS"},
	{"chase", "STORM CHASE"},
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

	store      *store.Store
	pages      map[string]fyne.CanvasObject
	modes      map[string]*modePill
	controls   fyne.CanvasObject
	unitsChip  *unitSegment
	formatChip *menuChip
	placeLine  *widget.RichText
	syncLine   *widget.RichText
	banner     *alertBanner
	pageSlot   *fyne.Container
	root       fyne.CanvasObject
	visible    string
	building   bool
	done       bool
	listenID   int
	onClosed   func()
}

// NewDesk opens a desk window and paints the current store. It does not fetch.
func NewDesk(app fyne.App, st *store.Store, onClosed func()) *Desk {
	d := &Desk{store: st, onClosed: onClosed, pages: map[string]fyne.CanvasObject{}, modes: map[string]*modePill{}}
	st.OpenDesk()
	d.Window = app.NewWindow("wx")
	d.build()
	d.Window.SetContent(d.root)
	// Resize after the content exists. Fyne will not shrink a window below
	// that content's minimum, so the minimum has to be the scroll, not the
	// tab row.
	d.Window.Resize(deskOpenSize)
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
	stopAlertMotion()
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
	if d.banner != nil {
		d.banner.paint(d.store.Payload)
	}
	d.bindAlertMotion()
	text := present.StatusText(d.store.Payload, d.store.Units, d.store.MenuBarFormat, d.store.Error)
	place := present.AsString(present.Conditions(d.store.Payload)["location"])
	if place == "" {
		place = d.store.Location
	}
	if place == "" {
		place = "AUTO"
	}
	d.Status.SetText(strings.TrimSpace(text))
	setRich(d.placeLine, strings.ToUpper(place))
	setRich(d.syncLine, syncLabel(d.store))
	if strings.TrimSpace(d.Location.Text) == "" && place != "" && place != "AUTO" {
		d.Location.SetText(place)
	}
	if label := deskFormatLabel(d.store.MenuBarFormat); d.Format.Selected != label {
		d.Format.SetSelected(label)
	}
	unitsLabel := "°F"
	if d.store.Units == "metric" {
		unitsLabel = "°C"
	}
	if d.Units.Selected != unitsLabel {
		d.Units.SetSelected(unitsLabel)
	}
	if d.unitsChip != nil {
		d.unitsChip.Refresh()
	}
	if d.formatChip != nil {
		d.formatChip.Refresh()
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
	d.Format.SetSelected(deskFormatLabel(d.store.MenuBarFormat))
	d.Format.OnChanged = d.onFormat

	d.Units = widget.NewSelect([]string{"°F", "°C"}, nil)
	if d.store.Units == "metric" {
		d.Units.SetSelected("°C")
	} else {
		d.Units.SetSelected("°F")
	}
	d.Units.OnChanged = d.onUnits

	// The selects stay off the row. Fyne's closed menu is wider than this
	// column, and it was painting through the city field. The chips call
	// the same selects, which the tests still drive.
	d.unitsChip = newUnitSegment(d.Units)
	d.formatChip = newMenuChip("MENU BAR:", d.Format, strings.ToUpper)
	sync := newActionChip("SYNC", func() { d.store.Refresh() })
	tools := container.NewHBox(d.unitsChip, d.formatChip, sync)
	controls := container.NewVBox(d.Location, tools)
	d.Status = widget.NewLabel("")
	d.Status.Wrapping = fyne.TextWrapOff
	d.Status.Truncation = fyne.TextTruncateEllipsis
	d.Status.Importance = widget.LowImportance
	d.Status.TextStyle = fyne.TextStyle{Monospace: true}
	d.Status.SizeName = theme.SizeNameCaptionText
	d.controls = consoleCard("Tactical Sensor Control", "SYS.01", "kicker", colCyan, container.NewVBox(controls, newSection("wx.status", d.Status)))

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

	var modes []fyne.CanvasObject
	for _, tab := range tabOrder {
		name := tab.id
		pill := newModePill(tab.title, func() { d.store.SetTab(name) })
		d.modes[name] = pill
		modes = append(modes, pill)
	}
	modeBar := container.NewHScroll(container.NewGridWithColumns(8, modes...))
	modeBar.SetMinSize(fyne.NewSize(200, 32))

	d.placeLine = line("AUTO", "kicker")
	d.placeLine.Wrapping = fyne.TextWrapOff
	d.syncLine = line("SYNC // —", "sync")
	d.syncLine.Wrapping = fyne.TextWrapOff
	network := canvas.NewText("METEOROLOGICAL SENSOR NETWORK // ACTIVE", colSilver)
	network.TextSize = 9
	network.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	capBG := canvas.NewRectangle(withAlpha(colChassis, 0xB3))
	capBG.CornerRadius = 10
	capBG.StrokeColor = withAlpha(colCyan, 0x40)
	capBG.StrokeWidth = 1
	capsule := container.NewStack(capBG, container.NewPadded(container.NewHBox(statusDot(colGreen, 6), network)))
	top := container.NewBorder(nil, nil, d.placeLine, capsule, layout.NewSpacer())
	title := tight("ATMOSPHERIC TELEMETRY CONSOLE", "console")
	bandBG := canvas.NewRectangle(withAlpha(colChassis, 0xF2))
	band := container.NewStack(bandBG, container.NewPadded(container.NewVBox(top, headerPill(title, d.syncLine), modeBar)))

	mark := canvas.NewText("NOAA.NWS SENSOR ARRAY · GRID 1KM", colSilver)
	mark.TextSize = 10
	mark.TextStyle = fyne.TextStyle{Monospace: true}
	sec := canvas.NewText("SEC.01 // TACTICAL", colCyan)
	sec.TextSize = 10
	sec.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	footBG := canvas.NewRectangle(withAlpha(colChassis, 0xCC))
	footer := container.NewStack(footBG, container.NewPadded(container.NewBorder(nil, nil, mark, sec, layout.NewSpacer())))

	d.pageSlot = container.NewStack()
	d.banner = newAlertBanner()
	d.showTab(d.store.Tab)
	// The banner sits under the header hairline and above every page, the
	// same place as AlertBannerView on the Mac desk. It is not inside the
	// scrolling alerts card.
	body := container.NewBorder(
		container.NewVBox(band, hairline(withAlpha(colCyan, 0x8C)), d.banner.section),
		container.NewVBox(hairline(withAlpha(colCyan, 0x73)), footer),
		nil, nil, d.pageSlot,
	)
	// The scroll reports a small minimum, so the frame can shrink. The body
	// still stretches when the window is larger, and scrolls when it is not.
	scroll := container.NewScroll(body)
	scroll.SetMinSize(deskMinSize)
	d.root = scroll
}

// deskOpenSize matches the Mac desk's ideal frame: wide enough for radar
// beside the readout, short of filling a 1440p panel.
var deskOpenSize = fyne.NewSize(940, 760)

// deskMinSize is the smallest frame. Content below this scrolls.
var deskMinSize = fyne.NewSize(480, 360)

// bindAlertMotion keeps the condition-badge strobe and the header lamp
// running for both columns. The banner lamp is registered from paint.
func (d *Desk) bindAlertMotion() {
	var objs []fyne.CanvasObject
	for _, surface := range d.Surfaces {
		if surface == nil {
			continue
		}
		if surface.alertPulse != nil && surface.alertPulse.live {
			objs = append(objs, surface.alertPulse)
		}
		walk(surface.alerts, func(c fyne.CanvasObject) {
			if badge, ok := c.(*conditionBadge); ok {
				objs = append(objs, badge)
			}
		})
	}
	setRowMotion(objs)
}

func (d *Desk) showTab(tab string) {
	page, ok := d.pages[tab]
	if !ok {
		tab = "dual"
		page = d.pages[tab]
	}
	d.visible = tab
	d.markTab(tab)
	d.placeControls(tab)
	shown := page
	if tab != "dual" {
		shown = container.NewScroll(page)
	}
	d.pageSlot.Objects = []fyne.CanvasObject{shown}
	d.pageSlot.Refresh()
}

func (d *Desk) markTab(tab string) {
	for id, pill := range d.modes {
		pill.Active = id == tab
		pill.Refresh()
	}
}

// placeControls keeps the sensor card on the tactical or surface column.
// Other pages leave it mounted on the tactical column so the status line
// stays in the tree.
func (d *Desk) placeControls(tab string) {
	if d.controls == nil || len(d.Surfaces) < 2 {
		return
	}
	host := d.Surfaces[0].Root
	if tab == "weather" {
		host = d.Surfaces[1].Root
	}
	if len(host.Objects) > 0 && host.Objects[0] == d.controls {
		return
	}
	for _, surface := range d.Surfaces {
		kept := make([]fyne.CanvasObject, 0, len(surface.Root.Objects))
		for _, obj := range surface.Root.Objects {
			if obj != d.controls {
				kept = append(kept, obj)
			}
		}
		surface.Root.Objects = kept
	}
	host.Objects = append([]fyne.CanvasObject{d.controls}, host.Objects...)
	host.Refresh()
}

func (d *Desk) onFormat(label string) {
	if d.building {
		return
	}
	for _, item := range deskFormats {
		if item.label == label {
			d.store.SetMenuBarFormat(item.key)
			return
		}
	}
}

func (d *Desk) onUnits(label string) {
	if d.building {
		return
	}
	units := "imperial"
	if label == "°C" {
		units = "metric"
	}
	if units != d.store.Units {
		d.store.SetUnits(units)
	}
}

// Desk select titles stay short. The tray keeps the long examples in FormatLabels,
// which ellipsize to "Standard (..." in this control.
var deskFormats = []struct{ key, label string }{
	{"compact", "Compact"},
	{"standard", "Standard"},
	{"tactical", "Tactical"},
}

func deskFormatLabel(key string) string {
	for _, item := range deskFormats {
		if item.key == key {
			return item.label
		}
	}
	return "Standard"
}

func syncLabel(st *store.Store) string {
	if st != nil && !st.LastRefreshed.IsZero() {
		return "SYNC // " + st.LastRefreshed.Format("3:04 PM")
	}
	if st != nil {
		if raw := present.AsString(present.Conditions(st.Payload)["observed_at"]); raw != "" {
			if stamp := present.LocalStamp(raw); stamp != "" {
				return "SYNC // " + stamp
			}
		}
	}
	return "SYNC // —"
}

func formatLabels() []string {
	out := make([]string, len(deskFormats))
	for i, item := range deskFormats {
		out[i] = item.label
	}
	return out
}
