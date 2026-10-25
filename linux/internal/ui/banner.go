package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mwirges/wx/linux/internal/present"
)

// alertBanner is the Mac AlertBannerView: pinned above every tab, warnings
// then watches. Advisories stay in the scrolling alerts list. The leading
// lamp uses the same 0.6s pulse as the Mac alerts header.
type alertBanner struct {
	section *Section
	bg      *canvas.Rectangle
	event   *widget.RichText
	expiry  *widget.RichText
	extra   *widget.RichText
	pulse   *pulseDot
	kind    present.BannerKind
	alerts  []map[string]any
}

func newAlertBanner() *alertBanner {
	b := &alertBanner{
		bg:     canvas.NewRectangle(colRed),
		event:  bannerRich("", colorBannerOnRed, true),
		expiry: bannerRich("", colorBannerOnRed, false),
		extra:  bannerRich("", colorBannerOnRed, false),
		pulse:  newPulseDot(color.White),
	}
	b.bg.CornerRadius = 2
	// The event is the border center, so it is given the leftover width and
	// ellipsizes there. An HBox would lock it to the one-character minimum
	// of a truncating line, which painted as "…".
	titles := container.NewBorder(nil, nil, nil, b.expiry, b.event)
	row := container.NewBorder(nil, nil, container.NewCenter(b.pulse), b.extra, titles)
	body := container.New(&insetLayout{h: 14, v: 8}, row)
	tap := newTapLayer(func(ev *fyne.PointEvent) { b.open(ev) })
	stack := container.NewStack(b.bg, body, tap)
	gap := container.NewGridWrap(fyne.NewSize(1, 8), canvas.NewRectangle(color.NRGBA{}))
	b.section = newSection("wx.alert.banner", container.NewVBox(stack, gap))
	b.section.Hide()
	return b
}

func (b *alertBanner) paint(payload map[string]any) {
	b.alerts = present.AlertMaps(payload)
	ranked := present.RankedBannerAlerts(payload)
	if len(ranked) == 0 {
		b.kind = ""
		setBannerRich(b.event, "", colorBannerOnRed)
		setBannerRich(b.expiry, "", colorBannerOnRed)
		setBannerRich(b.extra, "", colorBannerOnRed)
		b.pulse.SetLive(false)
		b.section.Hide()
		setBannerMotion(nil)
		return
	}
	lead := ranked[0]
	b.kind = lead.Kind
	ink := colorBannerOnRed
	b.bg.FillColor = colRed
	b.bg.StrokeWidth = 0
	b.pulse.color = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	if lead.Kind == present.BannerWatch {
		ink = colorBannerWatch
		b.bg.FillColor = withAlpha(colAmber, 0x29)
		b.bg.StrokeColor = colAmber
		b.bg.StrokeWidth = 1.5
		b.pulse.color = colAmber
	}
	b.bg.Refresh()
	setBannerRich(b.event, strings.ToUpper(present.AsString(lead.Alert["event"])), ink)
	if exp := present.BannerExpiry(present.AsString(lead.Alert["expires"])); exp != "" {
		setBannerRich(b.expiry, "EXPIRES "+strings.ToUpper(exp), ink)
	} else {
		setBannerRich(b.expiry, "", ink)
	}
	if extra := len(ranked) - 1; extra > 0 {
		setBannerRich(b.extra, fmt.Sprintf("+%d more", extra), ink)
	} else {
		setBannerRich(b.extra, "", ink)
	}
	b.pulse.SetLive(true)
	b.section.Show()
	setBannerMotion([]fyne.CanvasObject{b.pulse})
}

func (b *alertBanner) open(ev *fyne.PointEvent) {
	if len(b.alerts) == 0 || ev == nil || fyne.CurrentApp() == nil {
		return
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(b.section)
	if c == nil {
		return
	}
	box := container.NewVBox(line("ACTIVE ALERTS", "tag"))
	for _, alert := range b.alerts {
		box.Add(alertDetail(alert))
	}
	scroll := container.NewVScroll(box)
	scroll.SetMinSize(fyne.NewSize(440, 420))
	pop := widget.NewPopUp(scroll, c)
	pop.ShowAtPosition(ev.AbsolutePosition)
}

func alertDetail(alert map[string]any) fyne.CanvasObject {
	kind, _ := present.BannerKindOf(alert)
	role := "muted"
	accent := colSilver
	switch kind {
	case present.BannerWarning:
		role = "alert"
		accent = colRed
	case present.BannerWatch:
		role = "watch"
		accent = colAmber
	}
	body := container.NewVBox(line(strings.ToUpper(present.AsString(alert["event"])), role))
	for _, field := range present.AlertDetailFields(alert) {
		body.Add(line(field.Label, "tag"))
		body.Add(line(field.Value, "muted"))
	}
	bg := canvas.NewRectangle(withAlpha(accent, 0x1A))
	bg.CornerRadius = 6
	bg.StrokeColor = withAlpha(accent, 0x66)
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.NewPadded(body))
}

func bannerRich(text string, name fyne.ThemeColorName, truncate bool) *widget.RichText {
	rt := widget.NewRichText(&widget.TextSegment{Text: text, Style: bannerStyle(name)})
	rt.Wrapping = fyne.TextWrapOff
	if truncate {
		rt.Truncation = fyne.TextTruncateEllipsis
	}
	return rt
}

func bannerStyle(name fyne.ThemeColorName) widget.RichTextStyle {
	return widget.RichTextStyle{
		ColorName: name,
		SizeName:  theme.SizeNameCaptionText,
		TextStyle: fyne.TextStyle{Bold: true, Monospace: true},
	}
}

func setBannerRich(rt *widget.RichText, text string, name fyne.ThemeColorName) {
	if rt == nil || len(rt.Segments) == 0 {
		return
	}
	seg, ok := rt.Segments[0].(*widget.TextSegment)
	if !ok {
		return
	}
	if seg.Text == text && seg.Style.ColorName == name && rt.Visible() == (text != "") {
		return
	}
	seg.Text = text
	seg.Style.ColorName = name
	if text == "" {
		rt.Hide()
	} else {
		rt.Show()
	}
	rt.Refresh()
}

type insetLayout struct{ h, v float32 }

func (l insetLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	size := fyne.NewSize(l.h*2, l.v*2)
	if len(objects) == 0 || objects[0] == nil {
		return size
	}
	inner := objects[0].MinSize()
	return fyne.NewSize(inner.Width+l.h*2, inner.Height+l.v*2)
}

func (l insetLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 || objects[0] == nil {
		return
	}
	objects[0].Move(fyne.NewPos(l.h, l.v))
	w := size.Width - l.h*2
	h := size.Height - l.v*2
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	objects[0].Resize(fyne.NewSize(w, h))
}

// tapLayer covers the banner so a click opens the detail popover
// without replacing the text the tests read.
type tapLayer struct {
	widget.BaseWidget
	onTap func(*fyne.PointEvent)
}

func newTapLayer(fn func(*fyne.PointEvent)) *tapLayer {
	t := &tapLayer{onTap: fn}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tapLayer) Tapped(ev *fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap(ev)
	}
}

func (t *tapLayer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.NRGBA{}))
}
