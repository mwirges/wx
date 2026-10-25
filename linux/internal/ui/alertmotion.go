package ui

import (
	"image"
	"image/color"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// conditionBand matches the Mac ConditionPanel. Unknown severity draws no badge.
type conditionBand int

const (
	bandNone conditionBand = iota
	bandRed
	bandYellow
)

func conditionBandOf(severity string) conditionBand {
	switch strings.ToLower(severity) {
	case "extreme", "severe":
		return bandRed
	case "moderate", "minor":
		return bandYellow
	default:
		return bandNone
	}
}

// barOpacity is the Mac 9-phase cascade. Index 0 is the outer bar.
// reduceMotion holds the resting gradient and ignores phase.
func barOpacity(index, phase int, reduceMotion bool) float64 {
	base := [...]float64{0.50, 0.70, 0.84, 0.94, 1.0}[index]
	if reduceMotion {
		return base
	}
	switch phase {
	case 0:
		return base
	case 1:
		if index == 0 {
			return 0
		}
		return base
	case 2:
		if index <= 1 {
			return 0
		}
		return base
	case 3:
		if index <= 2 {
			return 0
		}
		if index == 3 {
			return 0.65
		}
		return 0.90
	case 4:
		if index <= 3 {
			return 0
		}
		return 0.48
	case 5:
		return 0
	case 6:
		if index == 0 || index == 3 {
			return 0.40
		}
		return 0
	case 7:
		if index == 0 || index == 3 {
			return 0.60
		}
		return 0
	case 8:
		if index == 4 {
			return 1
		}
		if index >= 2 {
			return 0.75
		}
		return 0
	default:
		return base
	}
}

// conditionPhase steps a red badge every 50ms and a yellow badge every 90ms,
// the same durations as ConditionBand.stepDuration.
func conditionPhase(elapsed time.Duration, band conditionBand) int {
	step := 50 * time.Millisecond
	if band == bandYellow {
		step = 90 * time.Millisecond
	}
	if elapsed < 0 {
		elapsed = 0
	}
	return int(elapsed/step) % 9
}

// pulseAlpha follows the Mac alerts-header PhaseAnimator: 0.6s ease each way
// between fully lit and 0.35.
func pulseAlpha(elapsed time.Duration) uint8 {
	t := math.Mod(elapsed.Seconds(), 1.2) / 1.2
	wave := 0.5 - 0.5*math.Cos(2*math.Pi*t)
	opacity := 1 - wave*(1-0.35)
	return uint8(opacity*255 + 0.5)
}

type motionState struct {
	sync.Mutex
	start   time.Time
	anim    *fyne.Animation
	running bool
	rows    []fyne.CanvasObject
	banner  []fyne.CanvasObject
}

var alertMotion motionState

func alertElapsed() time.Duration {
	alertMotion.Lock()
	defer alertMotion.Unlock()
	if alertMotion.start.IsZero() {
		return 0
	}
	return time.Since(alertMotion.start)
}

func setBannerMotion(objs []fyne.CanvasObject) { alertMotion.apply("banner", objs) }
func setRowMotion(objs []fyne.CanvasObject)    { alertMotion.apply("rows", objs) }

func (m *motionState) apply(kind string, objs []fyne.CanvasObject) {
	m.Lock()
	copied := append([]fyne.CanvasObject(nil), objs...)
	if kind == "banner" {
		m.banner = copied
	} else {
		m.rows = copied
	}
	want := len(m.banner)+len(m.rows) > 0
	var anim *fyne.Animation
	start := false
	stop := false
	if want {
		if m.start.IsZero() {
			m.start = time.Now()
		}
		if m.anim == nil {
			m.anim = fyne.NewAnimation(50*time.Millisecond, func(float32) { refreshAlertMotion() })
			m.anim.RepeatCount = fyne.AnimationRepeatForever
			m.anim.Curve = fyne.AnimationLinear
		}
		anim = m.anim
		start = !m.running
		m.running = true
	} else if m.running {
		anim = m.anim
		m.running = false
		stop = true
	}
	m.Unlock()
	if stop && anim != nil {
		anim.Stop()
		return
	}
	if start && anim != nil {
		anim.Start()
	}
}

func refreshAlertMotion() {
	alertMotion.Lock()
	items := make([]fyne.CanvasObject, 0, len(alertMotion.banner)+len(alertMotion.rows))
	items = append(items, alertMotion.banner...)
	items = append(items, alertMotion.rows...)
	alertMotion.Unlock()
	for _, item := range items {
		if item != nil {
			item.Refresh()
		}
	}
}

func stopAlertMotion() {
	alertMotion.Lock()
	anim := alertMotion.anim
	alertMotion.running = false
	alertMotion.banner = nil
	alertMotion.rows = nil
	alertMotion.Unlock()
	if anim != nil {
		anim.Stop()
	}
}

func bandPaint(band conditionBand, flash bool) color.NRGBA {
	if band == bandYellow {
		if flash {
			return color.NRGBA{R: 255, G: 220, B: 50, A: 255}
		}
		return colGold
	}
	if flash {
		return color.NRGBA{R: 255, G: 34, B: 34, A: 255}
	}
	return colRed
}

func scaleAlpha(c color.NRGBA, opacity float64) color.NRGBA {
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	c.A = uint8(float64(c.A)*opacity + 0.5)
	return c
}

// conditionBarRects is the five level bars in one wing. Top wing index 0 is
// the outer (first) bar. The bottom wing reverses that, matching ConditionBottomWing.
func conditionBarRects(w, h int, top bool) [5]image.Rectangle {
	var rects [5]image.Rectangle
	if w < 8 || h < 8 {
		return rects
	}
	wingH := int(float64(h) * 0.32)
	if wingH < 6 {
		wingH = 6
	}
	cutoutW := int(float64(w) * 0.4375)
	if cutoutW < 4 {
		cutoutW = 4
	}
	cutoutH := int(float64(wingH) * 0.833)
	if cutoutH < 5 {
		cutoutH = 5
	}
	x := (w - cutoutW) / 2
	y := 1
	if !top {
		y = h - cutoutH - 1
		if y < 0 {
			y = 0
		}
	}
	gap := 1
	barH := (cutoutH - 4*gap) / 5
	if barH < 1 {
		barH = 1
	}
	for i := 0; i < 5; i++ {
		rects[i] = image.Rect(x, y+i*(barH+gap), x+cutoutW, y+i*(barH+gap)+barH)
	}
	return rects
}

func drawConditionBadge(w, h int, band conditionBand, phase int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	if w < 2 || h < 2 {
		return img
	}
	const scale = 3
	big := image.NewNRGBA(image.Rect(0, 0, w*scale, h*scale))
	for i := 3; i < len(big.Pix); i += 4 {
		big.Pix[i] = 0xFF
	}
	flash := phase == 8
	fill := bandPaint(band, flash)
	element := 0.88
	if flash {
		element = 1
	}
	wing := scaleAlpha(fill, element)
	paintConditionWings(big, wing)
	for _, top := range []bool{true, false} {
		rects := conditionBarRects(w, h, top)
		for i, rect := range rects {
			idx := i
			if !top {
				idx = 4 - i
			}
			col := scaleAlpha(fill, barOpacity(idx, phase, false)*element)
			if col.A == 0 {
				continue
			}
			fillRect(big, scaleRect(rect, scale), col)
		}
	}
	small := downsample(big, w, h)
	drawBadgeType(small, band, fill)
	stroke := 0.35
	if flash {
		stroke = 0.9
	}
	drawBorder(small, scaleAlpha(fill, stroke))
	roundCorners(small, max(3, w*3/100))
	return small
}

func scaleRect(r image.Rectangle, scale int) image.Rectangle {
	return image.Rect(r.Min.X*scale, r.Min.Y*scale, r.Max.X*scale, r.Max.Y*scale)
}

type fpt struct{ x, y float64 }

func quadPoints(p0, p1, p2 fpt, n int) []fpt {
	out := make([]fpt, 0, n+1)
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		u := 1 - t
		out = append(out, fpt{
			x: u*u*p0.x + 2*u*t*p1.x + t*t*p2.x,
			y: u*u*p0.y + 2*u*t*p1.y + t*t*p2.y,
		})
	}
	return out
}

func paintConditionWings(img *image.NRGBA, col color.NRGBA) {
	b := img.Bounds()
	W := float64(b.Dx())
	H := float64(b.Dy())
	wingH := H * 0.32
	for _, poly := range [][]fpt{
		topLeftWing(W, wingH),
		topRightWing(W, wingH),
		offsetPoly(topLeftWing(W, wingH), 0, H-wingH, wingH),
		offsetPoly(topRightWing(W, wingH), 0, H-wingH, wingH),
	} {
		fillPoly(img, poly, col)
	}
	bw := W * 0.0775
	bh := H * 0.253
	y := H*0.50 - bh/2
	fillPoly(img, offsetPoly(leadingBracket(bw, bh), W*0.04, y, 0), col)
	fillPoly(img, mirrorBracket(bw, bh, W-W*0.04-bw, y), col)
}

func topLeftWing(W, wingH float64) []fpt {
	cutoutW := W * 0.4375
	cutoutH := wingH * 0.833
	cutoutLeft := (W - cutoutW) / 2
	start := fpt{cutoutLeft, 0}
	poly := quadPoints(start, fpt{W * 0.16, 0}, fpt{W * 0.1025, wingH * 0.50}, 16)
	return append(poly, fpt{W * 0.20, cutoutH}, fpt{cutoutLeft, cutoutH})
}

func topRightWing(W, wingH float64) []fpt {
	cutoutW := W * 0.4375
	cutoutH := wingH * 0.833
	cutoutRight := (W-cutoutW)/2 + cutoutW
	start := fpt{cutoutRight, 0}
	poly := quadPoints(start, fpt{W * 0.84, 0}, fpt{W * 0.8975, wingH * 0.50}, 16)
	return append(poly, fpt{W * 0.80, cutoutH}, fpt{cutoutRight, cutoutH})
}

func leadingBracket(bw, bh float64) []fpt {
	start := fpt{bw, 0}
	neck := fpt{bw * 0.22, 0}
	poly := []fpt{start}
	poly = append(poly, quadPoints(neck, fpt{-bw * 0.40, bh * 0.50}, fpt{bw * 0.22, bh}, 14)...)
	return append(poly, fpt{bw, bh})
}

func mirrorBracket(bw, bh, x, y float64) []fpt {
	local := leadingBracket(bw, bh)
	out := make([]fpt, len(local))
	for i, p := range local {
		out[i] = fpt{x + (bw - p.x), y + p.y}
	}
	return out
}

// offsetPoly moves a wing. flipWing mirrors it inside a box of height wingH
// before the move, which is how the bottom wing is the top wing turned over.
func offsetPoly(poly []fpt, x, y, flipH float64) []fpt {
	out := make([]fpt, len(poly))
	for i, p := range poly {
		py := p.y
		if flipH > 0 {
			py = flipH - p.y
		}
		out[i] = fpt{p.x + x, py + y}
	}
	return out
}

func fillPoly(img *image.NRGBA, poly []fpt, col color.NRGBA) {
	if len(poly) < 3 || col.A == 0 {
		return
	}
	minY, maxY := poly[0].y, poly[0].y
	for _, p := range poly[1:] {
		if p.y < minY {
			minY = p.y
		}
		if p.y > maxY {
			maxY = p.y
		}
	}
	for y := int(math.Floor(minY)); y <= int(math.Ceil(maxY)); y++ {
		var xs []float64
		yy := float64(y) + 0.5
		for i := 0; i < len(poly); i++ {
			a := poly[i]
			b := poly[(i+1)%len(poly)]
			if a.y == b.y || yy < math.Min(a.y, b.y) || yy >= math.Max(a.y, b.y) {
				continue
			}
			t := (yy - a.y) / (b.y - a.y)
			xs = append(xs, a.x+t*(b.x-a.x))
		}
		sort.Float64s(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			x0 := int(math.Ceil(xs[i]))
			x1 := int(math.Floor(xs[i+1]))
			for x := x0; x <= x1; x++ {
				blendOver(img, x, y, col)
			}
		}
	}
}

func blendOver(img *image.NRGBA, x, y int, src color.NRGBA) {
	if !image.Pt(x, y).In(img.Bounds()) || src.A == 0 {
		return
	}
	dst := img.NRGBAAt(x, y)
	sa := float64(src.A) / 255
	img.SetNRGBA(x, y, color.NRGBA{
		R: uint8(float64(src.R)*sa + float64(dst.R)*(1-sa) + 0.5),
		G: uint8(float64(src.G)*sa + float64(dst.G)*(1-sa) + 0.5),
		B: uint8(float64(src.B)*sa + float64(dst.B)*(1-sa) + 0.5),
		A: 0xFF,
	})
}

func drawBadgeType(img *image.NRGBA, band conditionBand, col color.NRGBA) {
	h := img.Bounds().Dy()
	w := img.Bounds().Dx()
	drawCentered(img, "ALERT", float64(h)*0.16, float64(h)*0.50, col, w)
	drawCentered(img, conditionLabel(band), float64(h)*0.078, float64(h)*0.64, col, w)
}

var (
	fontOnce sync.Once
	fontData *opentype.Font
	fontErr  error
	faceMu   sync.Mutex
	faces    map[int]font.Face
)

func deskFace(px float64) (font.Face, bool) {
	fontOnce.Do(func() {
		res := theme.DefaultTheme().Font(fyne.TextStyle{Monospace: true, Bold: true})
		if res == nil {
			fontErr = errNoFont
			return
		}
		fontData, fontErr = opentype.Parse(res.Content())
	})
	if fontErr != nil || px < 5 {
		return nil, false
	}
	size := int(px + 0.5)
	faceMu.Lock()
	defer faceMu.Unlock()
	if faces == nil {
		faces = map[int]font.Face{}
	}
	if face, ok := faces[size]; ok {
		return face, true
	}
	face, err := opentype.NewFace(fontData, &opentype.FaceOptions{
		Size: float64(size), DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, false
	}
	faces[size] = face
	return face, true
}

var errNoFont = errString("desk font")

type errString string

func (e errString) Error() string { return string(e) }

func drawCentered(img *image.NRGBA, text string, size, baseline float64, col color.NRGBA, width int) {
	for size >= 6 {
		face, ok := deskFace(size)
		if !ok {
			return
		}
		drawer := font.Drawer{Face: face}
		adv := drawer.MeasureString(text).Ceil()
		if adv <= width-8 || size <= 6 {
			drawer.Dst = img
			drawer.Src = image.NewUniform(col)
			drawer.Dot = fixed.P((width-adv)/2, int(baseline))
			drawer.DrawString(text)
			return
		}
		size -= 1
	}
}

func roundCorners(img *image.NRGBA, radius int) {
	if radius < 1 {
		return
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cx, cy := 0, 0
			switch {
			case x < radius && y < radius:
				cx, cy = radius, radius
			case x >= w-radius && y < radius:
				cx, cy = w-radius-1, radius
			case x < radius && y >= h-radius:
				cx, cy = radius, h-radius-1
			case x >= w-radius && y >= h-radius:
				cx, cy = w-radius-1, h-radius-1
			default:
				continue
			}
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy > radius*radius {
				img.SetNRGBA(x, y, color.NRGBA{})
			}
		}
	}
}

func fillRect(img *image.NRGBA, rect image.Rectangle, col color.NRGBA) {
	rect = rect.Intersect(img.Bounds())
	if rect.Empty() || col.A == 0 {
		return
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			blendOver(img, x, y, col)
		}
	}
}

func drawBorder(img *image.NRGBA, col color.NRGBA) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	fillRect(img, image.Rect(0, 0, w, 1), col)
	fillRect(img, image.Rect(0, h-1, w, h), col)
	fillRect(img, image.Rect(0, 0, 1, h), col)
	fillRect(img, image.Rect(w-1, 0, w, h), col)
}

// conditionBadge is the Mac ConditionPanel: winged frame, five level bars,
// and the 9-phase strobe. The desk clock calls Refresh.
type conditionBadge struct {
	widget.BaseWidget
	band conditionBand
}

func newConditionBadge(band conditionBand) *conditionBadge {
	b := &conditionBadge{band: band}
	b.ExtendBaseWidget(b)
	return b
}

func (b *conditionBadge) CreateRenderer() fyne.WidgetRenderer {
	badge := b
	raster := canvas.NewRaster(func(w, h int) image.Image {
		return drawConditionBadge(w, h, badge.band, conditionPhase(alertElapsed(), badge.band))
	})
	raster.ScaleMode = canvas.ImageScalePixels
	return &conditionRenderer{badge: b, raster: raster}
}

func conditionLabel(band conditionBand) string {
	if band == bandYellow {
		return "CONDITION: YELLOW"
	}
	return "CONDITION: RED"
}

type conditionRenderer struct {
	badge  *conditionBadge
	raster *canvas.Raster
}

func (r *conditionRenderer) Layout(size fyne.Size) {
	r.raster.Resize(size)
	r.raster.Move(fyne.NewPos(0, 0))
}

func (r *conditionRenderer) MinSize() fyne.Size { return fyne.NewSize(100, 75) }

func (r *conditionRenderer) Refresh() { r.raster.Refresh() }

func (r *conditionRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.raster}
}

func (r *conditionRenderer) Destroy() {}

// pulseDot is the Mac alerts-header lamp. It stays steady until Live is set.
type pulseDot struct {
	widget.BaseWidget
	live  bool
	color color.NRGBA
}

func newPulseDot(c color.Color) *pulseDot {
	p := &pulseDot{color: color.NRGBAModel.Convert(c).(color.NRGBA)}
	p.ExtendBaseWidget(p)
	return p
}

func (p *pulseDot) SetLive(on bool) {
	if p.live == on {
		return
	}
	p.live = on
	p.Refresh()
}

func (p *pulseDot) CreateRenderer() fyne.WidgetRenderer {
	return &pulseRenderer{
		dot:  p,
		glow: canvas.NewCircle(color.NRGBA{}),
		core: canvas.NewCircle(p.color),
	}
}

type pulseRenderer struct {
	dot  *pulseDot
	glow *canvas.Circle
	core *canvas.Circle
}

func (r *pulseRenderer) Layout(size fyne.Size) {
	r.glow.Resize(size)
	r.glow.Move(fyne.NewPos(0, 0))
	core := fyne.NewSize(6, 6)
	r.core.Resize(core)
	r.core.Move(fyne.NewPos((size.Width-core.Width)/2, (size.Height-core.Height)/2))
}

func (r *pulseRenderer) MinSize() fyne.Size { return fyne.NewSize(10, 10) }

func (r *pulseRenderer) Refresh() {
	alpha := uint8(255)
	glow := uint8(0)
	if r.dot.live {
		alpha = pulseAlpha(alertElapsed())
		glow = alpha / 3
	}
	r.core.FillColor = withAlpha(r.dot.color, alpha)
	r.glow.FillColor = withAlpha(r.dot.color, glow)
	r.core.Refresh()
	r.glow.Refresh()
}

func (r *pulseRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.glow, r.core}
}

func (r *pulseRenderer) Destroy() {}
