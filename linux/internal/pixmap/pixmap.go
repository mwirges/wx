// Package pixmap draws the tray icon: menu-bar text and a steady hazard dot.
// The dot does not blink. Drawing uses the Go image package and a bundled font.
package pixmap

import (
	"image"
	"image/color"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const height = 22

var (
	faceOnce sync.Once
	face     font.Face
)

func trayFace() font.Face {
	faceOnce.Do(func() {
		parsed, err := opentype.Parse(gobold.TTF)
		if err != nil {
			return
		}
		drawn, err := opentype.NewFace(parsed, &opentype.FaceOptions{
			Size: 15, DPI: 72, Hinting: font.HintingFull,
		})
		if err != nil {
			return
		}
		face = drawn
	})
	return face
}

// RenderStatus returns width, height, and ARGB32 bytes in network order.
func RenderStatus(text, pip string) (int, int, []byte) {
	if text == "" {
		text = "wx"
	}
	pad := 4
	dot := 0
	gap := 0
	if pip != "" {
		dot = 10
		gap = 4
	}
	textWidth := 8 * len([]rune(text))
	if f := trayFace(); f != nil {
		textWidth = font.MeasureString(f, text).Ceil()
	}
	width := pad + dot + gap + textWidth + pad + 2
	if width < 22 {
		width = 22
	}
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	x := pad
	if pip != "" {
		fillCircle(img, float64(x)+4.5, float64(height)/2, 4, pipColor(pip))
		x += dot + gap
	}
	if f := trayFace(); f != nil {
		d := font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(color.NRGBA{R: 240, G: 246, B: 255, A: 255}),
			Face: f,
			Dot:  fixed.P(x, 16),
		}
		d.DrawString(text)
	}
	return width, height, argb(img)
}

func pipColor(pip string) color.NRGBA {
	switch pip {
	case "warning":
		return color.NRGBA{R: 255, G: 0x3B, B: 0x56, A: 255}
	case "watch":
		return color.NRGBA{R: 255, G: 0x9F, B: 0x0A, A: 255}
	default:
		return color.NRGBA{}
	}
}

func fillCircle(img *image.NRGBA, cx, cy, r float64, c color.NRGBA) {
	rad := int(r + 1)
	ix, iy := int(cx), int(cy)
	rr := r * r
	for y := -rad; y <= rad; y++ {
		for x := -rad; x <= rad; x++ {
			if float64(x*x+y*y) <= rr {
				px, py := ix+x, iy+y
				if image.Pt(px, py).In(img.Bounds()) {
					img.SetNRGBA(px, py, c)
				}
			}
		}
	}
}

func argb(img *image.NRGBA) []byte {
	b := img.Bounds()
	out := make([]byte, b.Dx()*b.Dy()*4)
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			out[i] = c.A
			out[i+1] = c.R
			out[i+2] = c.G
			out[i+3] = c.B
			i += 4
		}
	}
	return out
}

// HasPipColor reports whether the pixmap contains the steady hazard color.
func HasPipColor(blob []byte, pip string) bool {
	for i := 0; i+3 < len(blob); i += 4 {
		red, green, blue := blob[i+1], blob[i+2], blob[i+3]
		if pip == "warning" && red > 200 && green < 90 && blue < 120 {
			return true
		}
		if pip == "watch" && red > 200 && green > 120 && green < 200 && blue < 40 {
			return true
		}
	}
	return false
}
