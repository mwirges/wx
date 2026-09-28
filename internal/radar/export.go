package radar

import (
	"bytes"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
)

// EncodeGIF encodes a sequence of images into an animated GIF.
// delayMs is the delay between frames in milliseconds (e.g. 500ms).
// The last frame is given an extended dwell time (2x delay) so viewers
// clearly see the latest live radar scan before the loop repeats.
func EncodeGIF(images []image.Image, delayMs int) ([]byte, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("no images to encode into GIF")
	}

	if delayMs <= 0 {
		delayMs = 500
	}
	// GIF delay is measured in units of 10ms (100ths of a second).
	delay100th := delayMs / 10
	if delay100th < 1 {
		delay100th = 1
	}

	out := &gif.GIF{
		Image:     make([]*image.Paletted, len(images)),
		Delay:     make([]int, len(images)),
		LoopCount: 0, // loop indefinitely
	}

	for i, img := range images {
		b := img.Bounds()
		paletted := image.NewPaletted(b, palette.Plan9)
		draw.FloydSteinberg.Draw(paletted, b, img, b.Min)
		out.Image[i] = paletted

		if i == len(images)-1 {
			out.Delay[i] = delay100th * 2 // dwell pause on live frame
		} else {
			out.Delay[i] = delay100th
		}
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, out); err != nil {
		return nil, fmt.Errorf("encode gif: %w", err)
	}
	return buf.Bytes(), nil
}
