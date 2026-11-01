package radar

import (
	"encoding/json"
	"io"
	"math"
	"time"

	"github.com/mwirges/wx/internal/location"
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
	IsComposite  bool        `json:"is_composite,omitempty"`
	Stations     []string    `json:"stations,omitempty"`
	BBox         *JSONBBox   `json:"bbox,omitempty"`
	Center       *JSONCenter `json:"center,omitempty"`
	Frames       []JSONFrame `json:"frames,omitempty"`
}

// DescribeFrame is the non-image portion of `wx radar --json` for one frame.
// ImageBase64 and Frames are left empty. Callers that print the CLI object
// fill those after encoding the PNG. MCP metadata uses this and does not
// fill them.
func DescribeFrame(loc location.Location, opts Options, stationID string, frame *Frame) JSONRadarOutput {
	if frame == nil {
		return JSONRadarOutput{}
	}
	center := &JSONCenter{Lat: loc.Lat, Lon: loc.Lon}
	if opts.BBox != nil {
		center.Lat = (opts.BBox.MinLat + opts.BBox.MaxLat) / 2
		center.Lon = (opts.BBox.MinLon + opts.BBox.MaxLon) / 2
	}
	var bbox *JSONBBox
	var stations []string
	if frame.BBox != nil {
		bbox = &JSONBBox{
			MinLat: frame.BBox.MinLat,
			MinLon: frame.BBox.MinLon,
			MaxLat: frame.BBox.MaxLat,
			MaxLon: frame.BBox.MaxLon,
		}
		for _, st := range StationsInBBox(*frame.BBox) {
			stations = append(stations, st.ID)
		}
	}
	return JSONRadarOutput{
		Product:      string(opts.Product),
		ProductLabel: ProductLabel(opts.Product),
		Location:     loc.DisplayName,
		Station:      stationID,
		ValidTime:    frame.ValidTime.UTC().Format(time.RFC3339),
		RadiusKM:     effectiveRadiusKM(opts, frame.BBox),
		Raw:          opts.Raw,
		IsComposite:  frame.IsComposite,
		Stations:     stations,
		BBox:         bbox,
		Center:       center,
	}
}

// effectiveRadiusKM matches wx radar --json. An explicit bbox replaces
// opts.RadiusKM with the larger half-span of that box, in kilometers.
func effectiveRadiusKM(opts Options, bbox *BBox) float64 {
	if bbox == nil || opts.BBox == nil {
		return opts.RadiusKM
	}
	midLat := (bbox.MinLat + bbox.MaxLat) / 2.0
	dLatKm := (bbox.MaxLat - bbox.MinLat) * 111.0 / 2.0
	dLonKm := (bbox.MaxLon - bbox.MinLon) * 111.0 * math.Cos(midLat*math.Pi/180.0) / 2.0
	return math.Max(dLatKm, dLonKm)
}

// RenderJSON writes out indented JSON to w.
func RenderJSON(w io.Writer, out JSONRadarOutput) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
