package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Mac desk palette from WxTheme. The Fyne theme uses the same bytes.
var (
	colChassis = color.NRGBA{R: 0x0A, G: 0x12, B: 0x22, A: 0xFF}
	colPanel   = color.NRGBA{R: 0x0E, G: 0x1A, B: 0x2D, A: 0xF0}
	colCyan    = color.NRGBA{R: 0x38, G: 0xE1, B: 0xFF, A: 0xFF}
	colGold    = color.NRGBA{R: 0xF5, G: 0xA6, B: 0x23, A: 0xFF}
	colAmber   = color.NRGBA{R: 0xFF, G: 0x9F, B: 0x0A, A: 0xFF}
	colRed     = color.NRGBA{R: 0xFF, G: 0x3B, B: 0x56, A: 0xFF}
	colGreen   = color.NRGBA{R: 0x30, G: 0xD1, B: 0x58, A: 0xFF}
	colSilver  = color.NRGBA{R: 0x94, G: 0xA7, B: 0xC5, A: 0xFF}
	colText    = color.NRGBA{R: 0xF0, G: 0xF6, B: 0xFF, A: 0xFF}
)

func withAlpha(c color.Color, a uint8) color.NRGBA {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	n.A = a
	return n
}

// bracketFrame paints a panel, a hairline stroke, and eight corner ticks
// in the margin. The body is last and inset, so it receives clicks and
// walk() still sees a container.
type bracketFrame struct {
	inset  float32
	length float32
}

func (b bracketFrame) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	pad := b.inset * 2
	return objects[len(objects)-1].MinSize().Add(fyne.NewSize(pad, pad))
}

func (b bracketFrame) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 11 {
		return
	}
	for _, obj := range objects[:2] {
		obj.Move(fyne.NewPos(0, 0))
		obj.Resize(size)
	}
	placeBrackets(objects[2:10], size, b.length)
	pad := b.inset
	inner := fyne.NewSize(size.Width-pad*2, size.Height-pad*2)
	if inner.Width < 0 {
		inner.Width = 0
	}
	if inner.Height < 0 {
		inner.Height = 0
	}
	objects[10].Move(fyne.NewPos(pad, pad))
	objects[10].Resize(inner)
}

func placeBrackets(lines []fyne.CanvasObject, size fyne.Size, length float32) {
	w, h := size.Width, size.Height
	if length > w/2 {
		length = w / 2
	}
	if length > h/2 {
		length = h / 2
	}
	// Inset by 2 so the stroke sits inside the panel instead of on the
	// clipped outer pixel.
	const o float32 = 2
	setLine(lines[0], o, o, length, o)
	setLine(lines[1], o, o, o, length)
	setLine(lines[2], w-length, o, w-o, o)
	setLine(lines[3], w-o, o, w-o, length)
	setLine(lines[4], o, h-o, length, h-o)
	setLine(lines[5], o, h-length, o, h-o)
	setLine(lines[6], w-length, h-o, w-o, h-o)
	setLine(lines[7], w-o, h-length, w-o, h-o)
}

func setLine(obj fyne.CanvasObject, x1, y1, x2, y2 float32) {
	line := obj.(*canvas.Line)
	line.Position1 = fyne.NewPos(x1, y1)
	line.Position2 = fyne.NewPos(x2, y2)
}

func framed(accent color.Color, content fyne.CanvasObject, inset, length float32) *fyne.Container {
	bg := canvas.NewRectangle(colPanel)
	bg.CornerRadius = 8
	stroke := canvas.NewRectangle(color.NRGBA{})
	stroke.StrokeColor = withAlpha(accent, 0x59)
	stroke.StrokeWidth = 1
	stroke.CornerRadius = 8
	objs := []fyne.CanvasObject{bg, stroke}
	tick := withAlpha(accent, 0xE0)
	for i := 0; i < 8; i++ {
		line := canvas.NewLine(tick)
		line.StrokeWidth = 2
		objs = append(objs, line)
	}
	objs = append(objs, content)
	return container.New(&bracketFrame{inset: inset, length: length}, objs...)
}

// consoleCard is the Mac SNWConsoleCard: status dot, mono title, silver tag,
// hairline, panel, corner brackets.
func consoleCard(title, tag, role string, accent color.Color, body fyne.CanvasObject) *fyne.Container {
	return consoleCardMark(title, tag, role, accent, nil, body)
}

// consoleCardMark is consoleCard with a caller-owned status lamp. A nil mark
// uses the steady dot. The alerts card passes a pulse so the header can breathe.
func consoleCardMark(title, tag, role string, accent color.Color, mark, body fyne.CanvasObject) *fyne.Container {
	// Border, not a stack of rows. The body has to take leftover height or
	// the radar picture stays at its minimum.
	content := body
	if strings.TrimSpace(title) != "" || strings.TrimSpace(tag) != "" {
		content = container.NewBorder(cardHeader(title, tag, role, accent, mark), nil, nil, nil, body)
	}
	return framed(accent, container.NewPadded(content), 7, 8)
}

func cardHeader(title, tag, role string, accent color.Color, mark fyne.CanvasObject) fyne.CanvasObject {
	titleLine := tight(strings.ToUpper(title), role)
	// The tag sits in the right slot, which is only as wide as its minimum.
	// A truncating line's minimum is one character, so the tag became "…".
	tagLine := line(strings.ToUpper(tag), "tag")
	tagLine.Wrapping = fyne.TextWrapOff
	if mark == nil {
		mark = statusDot(accent, 6)
	}
	row := container.NewBorder(nil, nil, mark, tagLine, titleLine)
	return container.NewVBox(row, hairline(withAlpha(accent, 0x66)))
}

func statusDot(c color.Color, size float32) fyne.CanvasObject {
	return container.NewGridWrap(fyne.NewSize(size, size), canvas.NewCircle(c))
}

type hairLayout struct{}

func (hairLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(1, 1) }

func (hairLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(size.Width, 1))
}

func hairline(c color.Color) fyne.CanvasObject {
	return container.New(&hairLayout{}, canvas.NewRectangle(c))
}

// metricTile is the Mac SNWMetricTile: silver label, value, small brackets.
func metricTile(label, value string) fyne.CanvasObject {
	inner := container.NewVBox(tight(strings.ToUpper(label), "tag"), tight(value, "tile"))
	return framed(colCyan, container.NewPadded(inner), 3, 5)
}

// headerPill is the Mac SNWHeaderPill: gold WX.DESK badge, title, sync time.
func headerPill(title, subtitle fyne.CanvasObject) fyne.CanvasObject {
	badgeText := canvas.NewText("WX.DESK", colGold)
	badgeText.TextSize = 10
	badgeText.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	badgeBG := canvas.NewRectangle(withAlpha(colGold, 0x1F))
	badgeBG.CornerRadius = 4
	badgeBG.StrokeColor = withAlpha(colGold, 0x59)
	badgeBG.StrokeWidth = 1
	badge := container.NewStack(badgeBG, container.NewPadded(container.NewHBox(statusDot(colGreen, 6), badgeText)))
	row := container.NewBorder(nil, nil, badge, subtitle, title)
	return framed(colCyan, container.NewPadded(row), 4, 6)
}

// modePill is one Mac mode button: equal cell, cyan wash when selected.
type modePill struct {
	widget.BaseWidget
	Text   string
	Active bool
	OnTap  func()
}

func newModePill(text string, onTap func()) *modePill {
	m := &modePill{Text: text, OnTap: onTap}
	m.ExtendBaseWidget(m)
	return m
}

func (m *modePill) Tapped(*fyne.PointEvent) {
	if m.OnTap != nil {
		m.OnTap()
	}
}

func (m *modePill) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(withAlpha(colChassis, 0x99))
	bg.CornerRadius = 6
	bg.StrokeWidth = 1
	bg.StrokeColor = withAlpha(colCyan, 0x40)
	label := canvas.NewText(m.Text, colSilver)
	label.TextSize = 9
	label.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	label.Alignment = fyne.TextAlignCenter
	return &modePillRenderer{pill: m, bg: bg, label: label}
}

type modePillRenderer struct {
	pill  *modePill
	bg    *canvas.Rectangle
	label *canvas.Text
}

func (r *modePillRenderer) Layout(size fyne.Size) {
	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(size)
	text := r.label.MinSize()
	r.label.Resize(text)
	r.label.Move(fyne.NewPos((size.Width-text.Width)/2, (size.Height-text.Height)/2))
}

func (r *modePillRenderer) MinSize() fyne.Size {
	return r.label.MinSize().Add(fyne.NewSize(12, 8))
}

func (r *modePillRenderer) Refresh() {
	r.label.Text = r.pill.Text
	if r.pill.Active {
		r.bg.FillColor = withAlpha(colCyan, 0x38)
		r.bg.StrokeColor = colCyan
		r.bg.StrokeWidth = 1.2
		r.label.Color = colText
	} else {
		r.bg.FillColor = withAlpha(colChassis, 0x99)
		r.bg.StrokeColor = withAlpha(colCyan, 0x40)
		r.bg.StrokeWidth = 1
		r.label.Color = colSilver
	}
	r.bg.Refresh()
	r.label.Refresh()
}

func (r *modePillRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.label}
}

func (r *modePillRenderer) Destroy() {}

// menuChip is a Mac menu label ("PRODUCT // COMPOSITE"). The select stays
// off the layout so tests can still call SetSelectedIndex.
type menuChip struct {
	widget.BaseWidget
	Prefix string
	Select *widget.Select
	Short  func(string) string
}

func newMenuChip(prefix string, sel *widget.Select, short func(string) string) *menuChip {
	m := &menuChip{Prefix: prefix, Select: sel, Short: short}
	m.ExtendBaseWidget(m)
	return m
}

func (m *menuChip) shown() string {
	val := ""
	if m.Select != nil {
		val = m.Select.Selected
	}
	if m.Short != nil {
		val = m.Short(val)
	} else {
		val = strings.ToUpper(val)
	}
	if m.Prefix == "" {
		return val
	}
	return m.Prefix + " " + val
}

func (m *menuChip) Tapped(ev *fyne.PointEvent) {
	if m.Select == nil || len(m.Select.Options) == 0 || ev == nil {
		return
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(m)
	if c == nil {
		return
	}
	box := container.NewVBox()
	var pop *widget.PopUp
	for i, opt := range m.Select.Options {
		label := opt
		if m.Short != nil {
			label = m.Short(opt)
		}
		idx := i
		btn := widget.NewButton(label, func() {
			m.Select.SetSelectedIndex(idx)
			m.Refresh()
			if pop != nil {
				pop.Hide()
			}
		})
		if i == m.Select.SelectedIndex() {
			btn.Importance = widget.HighImportance
		} else {
			btn.Importance = widget.LowImportance
		}
		box.Add(btn)
	}
	pop = widget.NewPopUp(box, c)
	pop.ShowAtPosition(ev.AbsolutePosition)
}

func (m *menuChip) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(withAlpha(colChassis, 0xD9))
	bg.CornerRadius = 5
	bg.StrokeWidth = 1
	bg.StrokeColor = withAlpha(colCyan, 0x66)
	label := canvas.NewText(m.shown(), colText)
	label.TextSize = 10
	label.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	return &menuChipRenderer{chip: m, bg: bg, label: label}
}

type menuChipRenderer struct {
	chip  *menuChip
	bg    *canvas.Rectangle
	label *canvas.Text
}

func (r *menuChipRenderer) Layout(size fyne.Size) {
	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(size)
	text := r.label.MinSize()
	r.label.Resize(text)
	r.label.Move(fyne.NewPos(8, (size.Height-text.Height)/2))
}

func (r *menuChipRenderer) MinSize() fyne.Size {
	return r.label.MinSize().Add(fyne.NewSize(16, 10))
}

func (r *menuChipRenderer) Refresh() {
	r.label.Text = r.chip.shown()
	r.label.Refresh()
	r.bg.Refresh()
}

func (r *menuChipRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.label}
}

func (r *menuChipRenderer) Destroy() {}

// unitSegment is the Mac °F / °C control.
type unitSegment struct {
	widget.BaseWidget
	Select *widget.Select
}

func newUnitSegment(sel *widget.Select) *unitSegment {
	u := &unitSegment{Select: sel}
	u.ExtendBaseWidget(u)
	return u
}

func (u *unitSegment) metric() bool {
	return u.Select != nil && u.Select.Selected == "°C"
}

func (u *unitSegment) Tapped(ev *fyne.PointEvent) {
	if u.Select == nil || ev == nil {
		return
	}
	if ev.Position.X < u.Size().Width/2 {
		u.Select.SetSelected("°F")
	} else {
		u.Select.SetSelected("°C")
	}
	u.Refresh()
}

func (u *unitSegment) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(withAlpha(colChassis, 0xD9))
	bg.CornerRadius = 5
	bg.StrokeWidth = 1
	bg.StrokeColor = withAlpha(colCyan, 0x66)
	f := canvas.NewText("°F", colText)
	f.TextSize = 11
	f.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	c := canvas.NewText("°C", colSilver)
	c.TextSize = 11
	c.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	mark := canvas.NewRectangle(withAlpha(colCyan, 0x33))
	mark.CornerRadius = 4
	return &unitRenderer{seg: u, bg: bg, mark: mark, f: f, c: c}
}

type unitRenderer struct {
	seg      *unitSegment
	bg, mark *canvas.Rectangle
	f, c     *canvas.Text
}

func (r *unitRenderer) Layout(size fyne.Size) {
	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(size)
	half := size.Width / 2
	r.mark.Resize(fyne.NewSize(half-4, size.Height-4))
	if r.seg.metric() {
		r.mark.Move(fyne.NewPos(half+2, 2))
	} else {
		r.mark.Move(fyne.NewPos(2, 2))
	}
	fs, cs := r.f.MinSize(), r.c.MinSize()
	r.f.Resize(fs)
	r.c.Resize(cs)
	r.f.Move(fyne.NewPos((half-fs.Width)/2, (size.Height-fs.Height)/2))
	r.c.Move(fyne.NewPos(half+(half-cs.Width)/2, (size.Height-cs.Height)/2))
}

func (r *unitRenderer) MinSize() fyne.Size { return fyne.NewSize(72, 26) }

func (r *unitRenderer) Refresh() {
	if r.seg.metric() {
		r.f.Color = colSilver
		r.c.Color = colText
	} else {
		r.f.Color = colText
		r.c.Color = colSilver
	}
	r.f.Refresh()
	r.c.Refresh()
	r.Layout(r.seg.Size())
	r.bg.Refresh()
	r.mark.Refresh()
}

func (r *unitRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.mark, r.f, r.c}
}

func (r *unitRenderer) Destroy() {}

// actionChip is a small console button: SYNC, LOOP, PNG.
type actionChip struct {
	widget.BaseWidget
	Text  string
	OnTap func()
}

func newActionChip(text string, onTap func()) *actionChip {
	a := &actionChip{Text: text, OnTap: onTap}
	a.ExtendBaseWidget(a)
	return a
}

func (a *actionChip) SetText(text string) {
	a.Text = text
	a.Refresh()
}

func (a *actionChip) Tapped(*fyne.PointEvent) {
	if a.OnTap != nil {
		a.OnTap()
	}
}

func (a *actionChip) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(withAlpha(colChassis, 0xD9))
	bg.CornerRadius = 5
	bg.StrokeWidth = 1
	bg.StrokeColor = withAlpha(colCyan, 0x66)
	label := canvas.NewText(a.Text, colCyan)
	label.TextSize = 10
	label.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	return &actionRenderer{chip: a, bg: bg, label: label}
}

type actionRenderer struct {
	chip  *actionChip
	bg    *canvas.Rectangle
	label *canvas.Text
}

func (r *actionRenderer) Layout(size fyne.Size) {
	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(size)
	text := r.label.MinSize()
	r.label.Resize(text)
	r.label.Move(fyne.NewPos((size.Width-text.Width)/2, (size.Height-text.Height)/2))
}

func (r *actionRenderer) MinSize() fyne.Size {
	return r.label.MinSize().Add(fyne.NewSize(14, 10))
}

func (r *actionRenderer) Refresh() {
	r.label.Text = r.chip.Text
	r.label.Color = colCyan
	r.label.Refresh()
	r.bg.Refresh()
}

func (r *actionRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.label}
}

func (r *actionRenderer) Destroy() {}

// statusBadge is the Mac nowcast pill. The words stay in a rich text node.
// The line must not truncate: a truncating segment's minimum is one character,
// and the pill then paints as "…".
func statusBadge(text, role string) fyne.CanvasObject {
	wash, edge, ink := withAlpha(colGreen, 0x28), withAlpha(colGreen, 0x73), colGreen
	switch role {
	case "alert":
		wash, edge, ink = withAlpha(colRed, 0x28), withAlpha(colRed, 0x73), colRed
	case "watch":
		wash, edge, ink = withAlpha(colAmber, 0x28), withAlpha(colAmber, 0x73), colAmber
	}
	bg := canvas.NewRectangle(wash)
	bg.CornerRadius = 4
	bg.StrokeColor = edge
	bg.StrokeWidth = 1
	words := line(text, role)
	words.Wrapping = fyne.TextWrapOff
	row := container.NewHBox(statusDot(ink, 6), words)
	return container.NewStack(bg, container.NewPadded(row))
}

func layoutSpacer() fyne.CanvasObject {
	return canvas.NewRectangle(color.NRGBA{})
}

// skyMark is the observation symbol beside the temperature.
func skyMark(code string) fyne.CanvasObject {
	return skyGlyph(code, 36)
}
