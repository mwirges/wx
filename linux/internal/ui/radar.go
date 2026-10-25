package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mwirges/wx/linux/internal/store"
)

var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

type radarProduct struct {
	ID    string
	Label string
}

var radarProducts = []radarProduct{
	{"composite-reflectivity", "Composite"},
	{"base-reflectivity", "Base"},
	{"storm-relative-velocity", "Velocity"},
	{"echo-tops", "Echo Tops"},
	{"precip-type", "Precip Type"},
	{"one-hour-precip", "1-Hr Precip"},
	{"storm-total-precip", "Storm Total"},
}

type radarRadius struct {
	KM    float64
	Label string
}

var radarRadii = []radarRadius{
	{150, "Local 150 km"},
	{200, "200 km"},
	{250, "Metro 250 km"},
	{500, "Regional 500 km"},
	{1000, "Synoptic 1000 km"},
	{2000, "CONUS 2000 km"},
}

// RadarPanel shows the PNG the CLI already produced.
// Save PNG and Save GIF call wx radar --save and wx radar --save-gif.
type RadarPanel struct {
	Root    fyne.CanvasObject
	Product *widget.Select
	Radius  *widget.Select
	Loop    *widget.Button
	Frame   *widget.Label
	Picture *canvas.Image
	Note    *widget.Label
	store   *store.Store
	window  fyne.Window
	guard   bool

	productChip *menuChip
	radiusChip  *menuChip
	loopChip    *actionChip
}

// NewRadarPanel builds the radar controls and picture.
func NewRadarPanel(st *store.Store, win fyne.Window) *RadarPanel {
	p := &RadarPanel{store: st, window: win}
	p.Product = widget.NewSelect(productLabels(), nil)
	p.Product.SetSelectedIndex(productIndex(st.RadarProduct))
	p.Product.OnChanged = p.onProduct
	p.Radius = widget.NewSelect(radiusLabels(), nil)
	p.Radius.SetSelectedIndex(radiusIndex(st.RadarRadius))
	p.Radius.OnChanged = p.onRadius
	p.Loop = widget.NewButton("Loop", func() { st.ToggleLoop() })
	p.productChip = newMenuChip("PRODUCT //", p.Product, strings.ToUpper)
	p.radiusChip = newMenuChip("SCALE //", p.Radius, scaleWord)
	p.loopChip = newActionChip("LOOP", func() { st.ToggleLoop() })
	prev := newActionChip("<", func() { st.StepFrame(-1) })
	next := newActionChip(">", func() { st.StepFrame(1) })
	pngBtn := newActionChip("PNG", func() { p.choose("wx-radar.png", p.SavePNG) })
	gifBtn := newActionChip("GIF", func() { p.choose("wx-radar.gif", p.SaveGIF) })
	p.Frame = widget.NewLabel("RADAR")
	p.Frame.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	p.Frame.SizeName = theme.SizeNameCaptionText
	p.Picture = canvas.NewImageFromImage(image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	p.Picture.FillMode = canvas.ImageFillContain
	p.Picture.ScaleMode = canvas.ImageScaleSmooth
	p.Picture.SetMinSize(fyne.NewSize(160, 120))
	p.Note = widget.NewLabel("")
	p.Note.Wrapping = fyne.TextWrapWord
	// The selects stay off the row so tests can still drive SetSelectedIndex.
	// The picture is the center and fills the pane between the two chip rows.
	menus := container.NewHBox(p.productChip, p.radiusChip)
	transport := container.NewHBox(prev, p.loopChip, next, pngBtn, gifBtn, p.Frame)
	body := container.NewBorder(menus, container.NewVBox(transport, p.Note), nil, nil, p.Picture)
	p.Root = newSection("wx.radar", consoleCard("Doppler Radar Array", "NOAA.MRMS", "kicker", colCyan, body))
	return p
}

// Refresh paints the current CLI frame.
func (p *RadarPanel) Refresh(st *store.Store) {
	p.guard = true
	defer func() { p.guard = false }()
	if idx := productIndex(st.RadarProduct); p.Product.SelectedIndex() != idx {
		p.Product.SetSelectedIndex(idx)
	}
	if st.LoopPlaying {
		p.Loop.SetText("Pause")
		p.loopChip.SetText("PAUSE")
	} else {
		p.Loop.SetText("Loop")
		p.loopChip.SetText("LOOP")
	}
	if idx := radiusIndex(st.RadarRadius); p.Radius.SelectedIndex() != idx {
		p.Radius.SetSelectedIndex(idx)
	}
	p.productChip.Refresh()
	p.radiusChip.Refresh()
	switch {
	case st.RadarError != "":
		p.Note.SetText(st.RadarError)
	case st.RadarLoading:
		p.Note.SetText("Fetching radar…")
	default:
		p.Note.SetText("")
	}
	frame := st.CurrentFrame()
	if frame == nil {
		p.Frame.SetText("NO FRAME")
		return
	}
	label := frame.Label
	if label == "" {
		label = "LIVE"
	}
	p.Frame.SetText(label)
	p.showPNG(frame.PNG)
}

func (p *RadarPanel) showPNG(blob []byte) {
	if len(blob) == 0 {
		return
	}
	if !bytes.HasPrefix(blob, pngMagic) {
		p.Picture.Image = nil
		p.Picture.Refresh()
		p.Note.SetText("Radar image from wx was not a PNG.")
		return
	}
	img, err := png.Decode(bytes.NewReader(blob))
	if err != nil {
		p.Picture.Image = nil
		p.Picture.Refresh()
		p.Note.SetText("Radar image from wx was not a PNG (" + err.Error() + ").")
		return
	}
	p.Picture.File = ""
	p.Picture.Resource = nil
	p.Picture.Image = matteRadar(img)
	p.Picture.Refresh()
}

// matteRadar paints the CLI PNG onto the desk background. Raw frames are
// transparent, and a transparent texture is invisible in this pane.
func matteRadar(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(radarField), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

var radarField = color.NRGBA{R: 0x06, G: 0x0A, B: 0x14, A: 0xFF}

// SavePNG asks the CLI to write a PNG. The desk does not encode the image.
func (p *RadarPanel) SavePNG(path string) {
	note, err := p.store.ExportPNG(path)
	if err != nil {
		p.Note.SetText(err.Error())
		return
	}
	if note == "" {
		note = "wx radar --save " + path
	}
	p.Note.SetText(note)
}

// SaveGIF asks the CLI to write a GIF. The desk does not composite frames.
func (p *RadarPanel) SaveGIF(path string) {
	note, err := p.store.ExportGIF(path)
	if err != nil {
		p.Note.SetText(err.Error())
		return
	}
	if note == "" {
		note = "wx radar --save-gif " + path
	}
	p.Note.SetText(note)
}

func (p *RadarPanel) choose(name string, save func(string)) {
	if p.window == nil {
		return
	}
	dlg := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		path := uc.URI().Path()
		_ = uc.Close()
		if path != "" {
			save(path)
		}
	}, p.window)
	dlg.SetFileName(name)
	dlg.Show()
}

func (p *RadarPanel) onProduct(label string) {
	if p.guard {
		return
	}
	idx := indexOf(productLabels(), label)
	if idx < 0 {
		return
	}
	product := radarProducts[idx].ID
	if product == p.store.RadarProduct {
		return
	}
	p.store.RadarProduct = product
	if p.store.DeskOpen {
		p.store.RefreshRadar()
	}
}

func (p *RadarPanel) onRadius(label string) {
	if p.guard {
		return
	}
	idx := indexOf(radiusLabels(), label)
	if idx < 0 {
		return
	}
	radius := radarRadii[idx].KM
	if radius == p.store.RadarRadius {
		return
	}
	p.store.RadarRadius = radius
	if p.store.DeskOpen {
		p.store.RefreshRadar()
	}
}

// scaleWord is the Mac coverage name. The select keeps the longer option.
func scaleWord(label string) string {
	switch {
	case strings.HasPrefix(label, "Local"):
		return "LOCAL"
	case strings.HasPrefix(label, "Metro"):
		return "METRO"
	case strings.HasPrefix(label, "Regional"):
		return "REGIONAL"
	case strings.HasPrefix(label, "Synoptic"):
		return "SYNOPTIC"
	case strings.HasPrefix(label, "CONUS"):
		return "CONUS"
	default:
		return strings.ToUpper(label)
	}
}

func productLabels() []string {
	out := make([]string, len(radarProducts))
	for i, product := range radarProducts {
		out[i] = product.Label
	}
	return out
}

func radiusLabels() []string {
	out := make([]string, len(radarRadii))
	for i, radius := range radarRadii {
		out[i] = radius.Label
	}
	return out
}

func productIndex(id string) int {
	for i, product := range radarProducts {
		if product.ID == id {
			return i
		}
	}
	return 0
}

func radiusIndex(km float64) int {
	best := 0
	diff := math.Abs(radarRadii[0].KM - km)
	for i, radius := range radarRadii {
		next := math.Abs(radius.KM - km)
		if next < diff {
			best = i
			diff = next
		}
	}
	return best
}

func indexOf(options []string, label string) int {
	for i, option := range options {
		if option == label {
			return i
		}
	}
	return -1
}
