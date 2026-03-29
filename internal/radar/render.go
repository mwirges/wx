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

// scaleImage returns a new *image.RGBA scaled to (w, h) via nearest-neighbour.
// Transparent source pixels are composited onto black.
func scaleImage(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	srcW, srcH := sb.Dx(), sb.Dy()
	for dy := 0; dy < h; dy++ {
		sy := sb.Min.Y + dy*srcH/h
		for dx := 0; dx < w; dx++ {
			sx := sb.Min.X + dx*srcW/w
			_, _, _, a := src.At(sx, sy).RGBA()
			if a == 0 {
				dst.Set(dx, dy, color.RGBA{0, 0, 0, 255})
			} else {
				r, g, b, _ := src.At(sx, sy).RGBA()
				dst.Set(dx, dy, color.RGBA{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
					A: 255,
				})
			}
		}
	}
	return dst
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
