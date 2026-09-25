package radar

import (
	"encoding/json"
	"io"
)

// JSONFrame holds a single radar image frame encoded as base64 PNG with timestamp.
type JSONFrame struct {
	ValidTime   string `json:"valid_time"`
	ImageBase64 string `json:"image_base64"`
}

// JSONRadarOutput is the structured JSON output for wx radar --json.
type JSONRadarOutput struct {
	Product      string      `json:"product"`
	ProductLabel string      `json:"product_label"`
	Location     string      `json:"location"`
	Station      string      `json:"station,omitempty"`
	ValidTime    string      `json:"valid_time"`
	ImageBase64  string      `json:"image_base64"`
	RadiusKM     float64     `json:"radius_km"`
	Frames       []JSONFrame `json:"frames,omitempty"`
}

// RenderJSON writes out indented JSON to w.
func RenderJSON(w io.Writer, out JSONRadarOutput) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
