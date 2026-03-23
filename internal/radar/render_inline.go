package radar

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
)

// renderInlineImage sends img to the terminal using the appropriate inline
// image protocol. cols and rows specify the display area in character cells.
func renderInlineImage(w io.Writer, img image.Image, cols, rows int, mode TermCapability) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("encode inline image: %w", err)
	}
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	switch mode {
	case TermITerm2:
		return renderITerm2(w, b64, cols, rows)
	case TermKitty:
		return renderKitty(w, b64, cols, rows)
	default:
		return fmt.Errorf("unsupported inline image mode")
	}
}

// renderITerm2 emits an iTerm2 inline image escape sequence.
// Supported by: iTerm2, WezTerm, Konsole, mintty.
func renderITerm2(w io.Writer, b64 string, cols, rows int) error {
	// \x1b]1337;File=...\a  — BEL terminator is more portable than ST.
	_, err := fmt.Fprintf(w,
		"\x1b]1337;File=inline=1;width=%d;height=%d;preserveAspectRatio=0:%s\a\n",
		cols, rows, b64)
	return err
}

// renderKitty emits a Kitty graphics protocol transmission.
// Supported by: Kitty, Ghostty, WezTerm.
func renderKitty(w io.Writer, b64 string, cols, rows int) error {
	const chunkSize = 4096

	for i := 0; i < len(b64); i += chunkSize {
		end := i + chunkSize
		if end > len(b64) {
			end = len(b64)
		}
		chunk := b64[i:end]
		more := 1
		if end >= len(b64) {
			more = 0
		}

		if i == 0 {
			// First chunk carries the full control payload.
			_, err := fmt.Fprintf(w, "\x1b_Ga=T,f=100,c=%d,r=%d,m=%d;%s\x1b\\",
				cols, rows, more, chunk)
			if err != nil {
				return err
			}
		} else {
			_, err := fmt.Fprintf(w, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
			if err != nil {
				return err
			}
		}
	}
	fmt.Fprintln(w)
	return nil
}
