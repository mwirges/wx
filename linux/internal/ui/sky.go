package ui

import (
	"image"
	"image/color"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"github.com/mwirges/wx/linux/internal/present"
)

// skyGlyph draws a small condition mark. The Mac desk uses an SF Symbol here.
func skyGlyph(code string, px int) fyne.CanvasObject {
	if px < 8 {
		px = 8
	}
	img := drawSky(code, px)
	mark := canvas.NewImageFromImage(img)
	mark.FillMode = canvas.ImageFillContain
	mark.ScaleMode = canvas.ImageScaleSmooth
	mark.SetMinSize(fyne.NewSize(float32(px), float32(px)))
	return mark
}

// periodGlyph picks a mark from the period text. Night names stay a moon
// unless the description names a cloud or a storm.
func periodGlyph(period map[string]any) string {
	desc := strings.ToLower(present.AsString(period["short_description"]))
	name := strings.ToLower(present.AsString(period["name"]))
	night := !present.AsBool(period["is_daytime"]) || strings.Contains(name, "night") || strings.Contains(name, "tonight")
	switch {
	case strings.Contains(desc, "thunder"), strings.Contains(desc, "storm"):
		return "storm"
	case strings.Contains(desc, "snow"), strings.Contains(desc, "flurr"), strings.Contains(desc, "sleet"):
		return "snow"
	case strings.Contains(desc, "rain"), strings.Contains(desc, "shower"), strings.Contains(desc, "drizzle"):
		return "rain"
	case strings.Contains(desc, "fog"), strings.Contains(desc, "mist"), strings.Contains(desc, "haze"):
		return "fog"
	case strings.Contains(desc, "partly"), strings.Contains(desc, "few"):
		if night {
			return "partly-night"
		}
		return "partly"
	case strings.Contains(desc, "cloud"), strings.Contains(desc, "overcast"):
		return "cloud"
	case night:
		return "clear-night"
	default:
		return "clear-day"
	}
}

func drawSky(code string, px int) *image.NRGBA {
	const scale = 4
	big := image.NewNRGBA(image.Rect(0, 0, px*scale, px*scale))
	paintSky(big, skyKind(code))
	return downsample(big, px, px)
}

func skyKind(code string) string {
	c := strings.ToLower(code)
	switch {
	case strings.Contains(c, "thunder"), strings.Contains(c, "storm"):
		return "storm"
	case strings.Contains(c, "snow"), strings.Contains(c, "sleet"), strings.Contains(c, "flurr"):
		return "snow"
	case strings.Contains(c, "rain"), strings.Contains(c, "drizzle"), strings.Contains(c, "shower"):
		return "rain"
	case strings.Contains(c, "fog"), strings.Contains(c, "mist"), strings.Contains(c, "haze"):
		return "fog"
	case strings.Contains(c, "partly"), strings.Contains(c, "few"):
		if strings.Contains(c, "night") {
			return "partly-night"
		}
		return "partly"
	case strings.Contains(c, "cloud"), strings.Contains(c, "overcast"):
		return "cloud"
	case strings.Contains(c, "night"):
		return "moon"
	default:
		return "sun"
	}
}

func paintSky(img *image.NRGBA, kind string) {
	b := img.Bounds()
	w := float64(b.Dx())
	h := float64(b.Dy())
	switch kind {
	case "moon":
		fillDisc(img, w*0.50, h*0.50, w*0.30, colSilver)
		fillDisc(img, w*0.64, h*0.40, w*0.24, color.NRGBA{})
	case "cloud":
		paintCloud(img, w*0.50, h*0.52, 1, colSilver)
	case "partly":
		fillDisc(img, w*0.38, h*0.38, w*0.18, colGold)
		paintCloud(img, w*0.58, h*0.60, 0.85, colSilver)
	case "partly-night":
		fillDisc(img, w*0.34, h*0.36, w*0.14, colSilver)
		fillDisc(img, w*0.42, h*0.32, w*0.11, color.NRGBA{})
		paintCloud(img, w*0.58, h*0.62, 0.85, colSilver)
	case "rain":
		paintCloud(img, w*0.50, h*0.40, 0.9, colSilver)
		for i, x := range []float64{0.34, 0.50, 0.66} {
			drop(img, w*x, h*(0.62+float64(i%2)*0.06), w*0.035, colCyan)
		}
	case "snow":
		paintCloud(img, w*0.50, h*0.40, 0.9, colSilver)
		for _, p := range []fpt{{0.32, 0.70}, {0.50, 0.78}, {0.68, 0.68}} {
			fillDisc(img, w*p.x, h*p.y, w*0.045, colText)
		}
	case "storm":
		paintCloud(img, w*0.50, h*0.38, 0.9, colSilver)
		fillPoly(img, []fpt{
			{w * 0.54, h * 0.48},
			{w * 0.40, h * 0.72},
			{w * 0.50, h * 0.72},
			{w * 0.44, h * 0.92},
			{w * 0.66, h * 0.62},
			{w * 0.54, h * 0.62},
		}, colGold)
	case "fog":
		for i, y := range []float64{0.28, 0.46, 0.64, 0.82} {
			inset := w * 0.08 * float64(i%2)
			fillRect(img, image.Rect(int(w*0.16+inset), int(h*y), int(w*0.84-inset), int(h*y+h*0.08)), withAlpha(colSilver, 0xCC))
		}
	default:
		fillDisc(img, w*0.50, h*0.50, w*0.20, colGold)
		for i := 0; i < 8; i++ {
			ang := float64(i) * math.Pi / 4
			fillDisc(img, w*0.50+math.Cos(ang)*w*0.36, h*0.50+math.Sin(ang)*h*0.36, w*0.045, colGold)
		}
	}
}

func paintCloud(img *image.NRGBA, cx, cy, scale float64, col color.NRGBA) {
	w := float64(img.Bounds().Dx()) * scale
	fillDisc(img, cx-w*0.16, cy+w*0.02, w*0.16, col)
	fillDisc(img, cx+w*0.12, cy+w*0.04, w*0.14, col)
	fillDisc(img, cx-w*0.02, cy-w*0.08, w*0.18, col)
	fillRect(img, image.Rect(int(cx-w*0.28), int(cy), int(cx+w*0.28), int(cy+w*0.16)), col)
}

func fillDisc(img *image.NRGBA, cx, cy, r float64, col color.NRGBA) {
	if r <= 0 {
		return
	}
	minX := int(math.Floor(cx - r))
	maxX := int(math.Ceil(cx + r))
	minY := int(math.Floor(cy - r))
	maxY := int(math.Ceil(cy + r))
	rr := r * r
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			if dx*dx+dy*dy <= rr {
				if col.A == 0 {
					if image.Pt(x, y).In(img.Bounds()) {
						img.SetNRGBA(x, y, color.NRGBA{})
					}
					continue
				}
				blendOver(img, x, y, col)
			}
		}
	}
}

func drop(img *image.NRGBA, cx, cy, r float64, col color.NRGBA) {
	fillDisc(img, cx, cy, r, col)
	fillDisc(img, cx, cy+r*1.6, r*0.7, col)
}

func downsample(src *image.NRGBA, w, h int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw == 0 || sh == 0 || w == 0 || h == 0 {
		return dst
	}
	for y := 0; y < h; y++ {
		y0 := y * sh / h
		y1 := (y + 1) * sh / h
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < w; x++ {
			x0 := x * sw / w
			x1 := (x + 1) * sw / w
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, b, a, n int
			for yy := y0; yy < y1 && yy < sh; yy++ {
				for xx := x0; xx < x1 && xx < sw; xx++ {
					c := src.NRGBAAt(xx, yy)
					r += int(c.R)
					g += int(c.G)
					b += int(c.B)
					a += int(c.A)
					n++
				}
			}
			if n == 0 {
				continue
			}
			dst.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: uint8(a / n)})
		}
	}
	return dst
}
