package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// tight is a single line that keeps its own width. A wrapping label inside a
// horizontal strip reports a tiny minimum, then paints across the next cell.
func tight(text, role string) *widget.RichText {
	rt := line(text, role)
	rt.Wrapping = fyne.TextWrapOff
	rt.Truncation = fyne.TextTruncateEllipsis
	return rt
}

// chip reserves a column width and takes its height from the text once it
// has a renderer. A height captured at construction is one line short, and
// the next line paints through the card below.
func chip(width float32, parts ...fyne.CanvasObject) fyne.CanvasObject {
	s := &chipSlot{width: width, body: container.NewVBox(parts...)}
	s.ExtendBaseWidget(s)
	return s
}

type chipSlot struct {
	widget.BaseWidget
	width float32
	body  *fyne.Container
}

func (s *chipSlot) CreateRenderer() fyne.WidgetRenderer {
	return &chipRenderer{slot: s}
}

type chipRenderer struct{ slot *chipSlot }

func (r *chipRenderer) Layout(size fyne.Size) {
	r.slot.body.Move(fyne.NewPos(0, 0))
	r.slot.body.Resize(size)
}

func (r *chipRenderer) MinSize() fyne.Size {
	height := r.slot.body.MinSize().Height
	if height < 28 {
		height = 28
	}
	return fyne.NewSize(r.slot.width, height)
}

func (r *chipRenderer) Refresh() { r.slot.body.Refresh() }

func (r *chipRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.slot.body} }

func (r *chipRenderer) Destroy() {}

func line(text, role string) *widget.RichText {
	style := widget.RichTextStyle{
		ColorName: theme.ColorNameForeground,
		SizeName:  theme.SizeNameText,
		TextStyle: fyne.TextStyle{},
	}
	switch role {
	case "kicker":
		style.ColorName = theme.ColorNamePrimary
		style.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
		style.SizeName = theme.SizeNameCaptionText
	case "alertkicker":
		style.ColorName = theme.ColorNameError
		style.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
		style.SizeName = theme.SizeNameCaptionText
	case "console":
		style.ColorName = theme.ColorNameForeground
		style.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	case "sync":
		style.ColorName = theme.ColorNamePrimary
		style.TextStyle = fyne.TextStyle{Monospace: true}
		style.SizeName = theme.SizeNameCaptionText
	case "place":
		style.ColorName = theme.ColorNameForeground
		style.TextStyle = fyne.TextStyle{Bold: true}
	case "tag":
		style.ColorName = theme.ColorNameDisabled
		style.TextStyle = fyne.TextStyle{Monospace: true}
		style.SizeName = theme.SizeNameCaptionText
	case "tile":
		style.ColorName = theme.ColorNameForeground
		style.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	case "muted":
		style.ColorName = theme.ColorNameDisabled
		style.SizeName = theme.SizeNameText
	case "temp":
		style.ColorName = theme.ColorNameForeground
		style.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
		style.SizeName = theme.SizeNameHeadingText
	case "hour":
		style.ColorName = theme.ColorNamePrimary
		style.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	case "alert":
		style.ColorName = theme.ColorNameError
		style.TextStyle = fyne.TextStyle{Bold: true}
	case "watch":
		style.ColorName = theme.ColorNameWarning
		style.TextStyle = fyne.TextStyle{Bold: true}
	case "ok":
		style.ColorName = theme.ColorNameSuccess
		style.TextStyle = fyne.TextStyle{Bold: true}
	}
	rt := widget.NewRichText(&widget.TextSegment{Text: text, Style: style})
	rt.Wrapping = fyne.TextWrapWord
	return rt
}

func addLine(box *fyne.Container, text, role string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	box.Add(line(text, role))
}

func clearBox(box *fyne.Container) {
	box.Objects = nil
	box.Refresh()
}

// Section is a named block the tests and the desk can find.
type Section struct {
	widget.BaseWidget
	ID  string
	Obj fyne.CanvasObject
}

func newSection(id string, obj fyne.CanvasObject) *Section {
	s := &Section{ID: id, Obj: obj}
	s.ExtendBaseWidget(s)
	return s
}

// CreateRenderer draws the section body.
func (s *Section) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.Obj)
}

func card(title, tag string, body *fyne.Container) *fyne.Container {
	return consoleCard(title, tag, "kicker", colCyan, body)
}

func sectionCardTone(id, title, tag, role string, accent color.Color, body *fyne.Container) *Section {
	return newSection(id, consoleCard(title, tag, role, accent, body))
}

func setRich(rt *widget.RichText, text string) {
	if rt == nil || len(rt.Segments) == 0 {
		return
	}
	seg, ok := rt.Segments[0].(*widget.TextSegment)
	if !ok || seg.Text == text {
		return
	}
	seg.Text = text
	rt.Refresh()
}

func sectionCard(id, title, tag string, body *fyne.Container) *Section {
	return newSection(id, card(title, tag, body))
}
