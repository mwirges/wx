package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

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
	case "tag":
		style.ColorName = theme.ColorNameDisabled
		style.TextStyle = fyne.TextStyle{Monospace: true}
		style.SizeName = theme.SizeNameCaptionText
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
	// The title takes the leftover width and truncates. A wrapping label in the
	// left slot reports a tiny minimum width and then paints across the radar.
	titleLine := line(strings.ToUpper(title), "kicker")
	titleLine.Wrapping = fyne.TextWrapOff
	titleLine.Truncation = fyne.TextTruncateEllipsis
	tagLine := line(tag, "tag")
	tagLine.Wrapping = fyne.TextWrapOff
	header := container.NewBorder(nil, nil, nil, tagLine, titleLine)
	inner := container.NewVBox(header, body)
	bg := canvas.NewRectangle(color.NRGBA{R: 0x0E, G: 0x1A, B: 0x2D, A: 0xF0})
	return container.NewStack(bg, container.NewPadded(inner))
}

func sectionCard(id, title, tag string, body *fyne.Container) *Section {
	return newSection(id, card(title, tag, body))
}
