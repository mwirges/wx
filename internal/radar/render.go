package radar

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TermCapability indicates the best image rendering protocol the terminal supports.
type TermCapability int

const (
	TermHalfBlock TermCapability = iota // default: Unicode half-block characters
	TermITerm2                          // iTerm2 inline image protocol
	TermKitty                           // Kitty graphics protocol
)

// RenderOptions controls how a radar frame is drawn in the terminal.
type RenderOptions struct {
	TermWidth  int            // terminal columns available
	TermHeight int            // terminal rows available
	Mode       TermCapability // rendering mode (zero = half-block)
}

const (
	headerLines = 2 // product + location line above the image
	footerLines = 1 // timestamp line below the image
)

// RenderFrame writes a single radar frame to w.
//
// When Mode is TermITerm2 or TermKitty the original high-resolution image is
// sent to the terminal via the corresponding inline-image protocol.
//
// Otherwise the image is downscaled and rendered with half-block Unicode
// characters (▀) using ANSI truecolor, with major city labels overlaid.
func RenderFrame(w io.Writer, frame *Frame, locName string, opts RenderOptions) error {
	imgRows := opts.TermHeight - headerLines - footerLines
	imgW := opts.TermWidth
	imgH := imgRows * 2 // for half-block: 2 pixel rows per character row
	if imgW < 10 || imgH < 10 {
		return fmt.Errorf("terminal too small for radar display (need at least 10×%d)", 10+headerLines+footerLines)
	}

	// ── Header ──────────────────────────────────────────────────────
	writeHeader(w, locName, frame, opts)

	// ── Radar image ─────────────────────────────────────────────────
	if opts.Mode == TermITerm2 || opts.Mode == TermKitty {
		labeled := drawCityLabels(frame.Img, frame.BBox)
		if err := renderInlineImage(w, labeled, imgW, imgRows, opts.Mode); err != nil {
			// Inline rendering failed — fall through to half-block.
			renderHalfBlockWithLabels(w, frame, imgW, imgH)
		}
	} else {
		renderHalfBlockWithLabels(w, frame, imgW, imgH)
	}

	// ── Footer: timestamp ───────────────────────────────────────────
	writeFooter(w, frame, opts)
	return nil
}

// writeHeader outputs the location + product badge + separator line.
func writeHeader(w io.Writer, locName string, frame *Frame, opts RenderOptions) {
	locStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	prodStyle := lipgloss.NewStyle().Bold(true).Foreground(productColor(frame.Product))

	prodBadge := "● " + ProductLabel(frame.Product)
	gap := opts.TermWidth - len(locName) - len(prodBadge)
	if gap < 2 {
		gap = 2
	}
	fmt.Fprintf(w, "%s%s%s\x1b[K\n",
		locStyle.Render(locName),
		strings.Repeat(" ", gap),
		prodStyle.Render(prodBadge))
	fmt.Fprintf(w, "%s\x1b[K\n", subStyle.Render(strings.Repeat("─", opts.TermWidth)))
}

// writeFooter outputs the centered timestamp line.
func writeFooter(w io.Writer, frame *Frame, opts RenderOptions) {
	ts := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).
		Render(frame.ValidTime.Local().Format("Mon Jan 2, 3:04 PM MST"))
	pad := (opts.TermWidth - lipgloss.Width(ts)) / 2
	if pad < 0 {
		pad = 0
	}
	fmt.Fprintf(w, "%s%s\x1b[K\n", strings.Repeat(" ", pad), ts)
}

// renderHalfBlockWithLabels downscales the frame image and renders it using
// half-block characters, overlaying city name labels where the BBox is known.
func renderHalfBlockWithLabels(w io.Writer, frame *Frame, imgW, imgH int) {
	scaled := scaleImage(frame.Img, imgW, imgH)
	termRows := imgH / 2

	// Build label index for O(1) per-cell lookup.
	labels := layoutLabels(frame.BBox, imgW, termRows)
	idx := buildLabelIndex(labels)

	for y := 0; y+1 < imgH; y += 2 {
		termRow := y / 2
		for x := 0; x < imgW; x++ {
			if ch, ok := idx.at(termRow, x); ok {
				// Label character: bright white on a darkened background derived
				// from the underlying radar pixel for visual continuity.
				bot := toRGBA(scaled.At(x, y+1))
				bgR := clampByte(int(bot.R)/4, 20)
				bgG := clampByte(int(bot.G)/4, 20)
				bgB := clampByte(int(bot.B)/4, 20)
				fmt.Fprintf(w, "\x1b[1;38;2;255;255;255m\x1b[48;2;%d;%d;%dm%c",
					bgR, bgG, bgB, ch)
			} else {
				top := toRGBA(scaled.At(x, y))
				bot := toRGBA(scaled.At(x, y+1))
				fmt.Fprintf(w, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
					top.R, top.G, top.B,
					bot.R, bot.G, bot.B)
			}
		}
		fmt.Fprintf(w, "\x1b[0m\x1b[K\n")
	}
}

func clampByte(v, min int) uint8 {
	if v < min {
		return uint8(min)
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// ProductLabel returns a human-readable name for a radar product.
func ProductLabel(p Product) string {
	switch p {
	case ProductCompositeReflectivity:
		return "Composite Reflectivity"
	case ProductBaseReflectivity:
		return "Base Reflectivity"
	case ProductStormRelativeVelocity:
		return "Storm Rel. Velocity"
	case ProductEchoTops:
		return "Echo Tops"
	default:
		return string(p)
	}
}

// scaleImage returns a new *image.RGBA scaled to (w, h).
//
// Radar pixels are area-averaged across the source region for smooth output.
// Geographic boundary lines (white/gray, low color saturation) are detected
// separately: if any source pixel in the region looks like a border (sat < 40,
// luminance > ~30), it is overlaid on top of the averaged radar color. This
// makes state and county lines visible without causing isolated bright radar
// returns to bloat and dominate their source regions.
// Fully-transparent regions are composited onto black.
func scaleImage(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	srcW, srcH := sb.Dx(), sb.Dy()
	for dy := 0; dy < h; dy++ {
		sy0 := sb.Min.Y + dy*srcH/h
		sy1 := sb.Min.Y + (dy+1)*srcH/h
		if sy1 > sb.Max.Y {
			sy1 = sb.Max.Y
		}
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for dx := 0; dx < w; dx++ {
			sx0 := sb.Min.X + dx*srcW/w
			sx1 := sb.Min.X + (dx+1)*srcW/w
			if sx1 > sb.Max.X {
				sx1 = sb.Max.X
			}
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}

			var sumR, sumG, sumB, n uint32
			var lineR, lineG, lineB uint8
			var bestLineLum uint32
			hasLine := false

			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					r32, g32, b32, a32 := src.At(sx, sy).RGBA()
					if a32 == 0 {
						continue
					}
					r, g, b := uint8(r32>>8), uint8(g32>>8), uint8(b32>>8)
					sumR += uint32(r)
					sumG += uint32(g)
					sumB += uint32(b)
					n++

					// Detect geographic line pixels: bright, unsaturated (white/gray).
					// IEM draws state borders in white and county lines in light gray;
					// radar returns are always highly saturated colors (yellow/red/purple).
					maxC := maxUint8(r, g, b)
					minC := minUint8(r, g, b)
					sat := uint32(maxC - minC)
					lum := uint32(r)*299 + uint32(g)*587 + uint32(b)*114
					if sat < 40 && lum > 30000 {
						if !hasLine || lum > bestLineLum {
							lineR, lineG, lineB = r, g, b
							bestLineLum = lum
							hasLine = true
						}
					}
				}
			}

			if hasLine {
				dst.Set(dx, dy, color.RGBA{lineR, lineG, lineB, 255})
			} else if n > 0 {
				dst.Set(dx, dy, color.RGBA{uint8(sumR / n), uint8(sumG / n), uint8(sumB / n), 255})
			} else {
				dst.Set(dx, dy, color.RGBA{0, 0, 0, 255})
			}
		}
	}
	return dst
}

func maxUint8(a, b, c uint8) uint8 {
	if a >= b && a >= c {
		return a
	}
	if b >= c {
		return b
	}
	return c
}

func minUint8(a, b, c uint8) uint8 {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// toRGBA extracts 8-bit color components from any color.Color.
func toRGBA(c color.Color) color.RGBA {
	r, g, b, _ := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}
}

// productColor returns a terminal color for the radar product badge.
func productColor(p Product) lipgloss.Color {
	switch p {
	case ProductBaseReflectivity:
		return lipgloss.Color("51") // cyan — single-tilt / lower-atmosphere focus
	case ProductStormRelativeVelocity:
		return lipgloss.Color("201") // magenta — storm-relative motion
	case ProductEchoTops:
		return lipgloss.Color("208") // orange — cloud top heights
	default:
		return lipgloss.Color("226") // yellow — composite / full-column view
	}
}
