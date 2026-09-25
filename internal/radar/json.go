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

// JSONBBox holds bounding box coordinates for radar positioning.
type JSONBBox struct {
	MinLat float64 `json:"min_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLat float64 `json:"max_lat"`
	MaxLon float64 `json:"max_lon"`
}

// JSONCenter holds the geographic center coordinates.
type JSONCenter struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
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
	Raw          bool        `json:"raw,omitempty"`
	BBox         *JSONBBox   `json:"bbox,omitempty"`
	Center       *JSONCenter `json:"center,omitempty"`
	Frames       []JSONFrame `json:"frames,omitempty"`
}

// RenderJSON writes out indented JSON to w.
func RenderJSON(w io.Writer, out JSONRadarOutput) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
