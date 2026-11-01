package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/radar"
)

// RadarImageMaxBytes is the hard cap for one MCP radar image.
// A decoded PNG or its base64 form over this size is payload_too_large.
// 256 KiB. CLI multi-frame loops are not returned on this path.
const RadarImageMaxBytes = 256 << 10

// radarFunc replaces the live one-frame fetch in tests.
// It must not stand in for a six-frame loop. The handler still drops
// image bytes unless include_image is set, and still enforces the cap.
type radarFunc func(ctx context.Context, loc location.Location, opts radar.Options, station string) (radarShot, error)

// radarShot is one current frame. PNG is the encoded image, empty when
// the caller did not ask for pixels. Out must not be trusted to omit
// images: radarToolJSON strips them unless include_image is set.
type radarShot struct {
	Out radar.JSONRadarOutput
	PNG []byte
}

type radarArgs struct {
	Location     string  `json:"location,omitempty" jsonschema:"Optional place. Omit, empty, or default uses the configured default. Otherwise a favorite, ZIP, City ST, or lat,lon."`
	Product      string  `json:"product,omitempty" jsonschema:"Radar product. Omit to use the configured product, or composite-reflectivity."`
	RadiusKM     float64 `json:"radius_km,omitempty" jsonschema:"Kilometer radius around the location. Omit to use the configured radius, or 200."`
	Station      string  `json:"station,omitempty" jsonschema:"NEXRAD station id such as KIWX. Omit to use the configured station."`
	Raw          bool    `json:"raw,omitempty" jsonschema:"Raw radar imagery without the background map or city labels."`
	IncludeImage bool    `json:"include_image,omitempty" jsonschema:"When true, include one frame. Omit or false for metadata only. The frame is refused with payload_too_large when the PNG or its base64 exceeds 256 KiB."`
}

func (s *Server) addRadarTool() {
	const polite = " Be polite to NWS and IEM: these calls share a small in-process cap."
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_radar",
		Description: "Radar for one location. The default result is metadata only: product, product_label, stations, bbox, center, radius_km, valid_time, location, is_composite, and raw when set. It has no image_base64 and no frames array. include_image returns at most one frame, never a multi-frame loop. A PNG or its base64 over 256 KiB returns payload_too_large." + polite,
	}, s.wxRadar)
}

func (s *Server) wxRadar(ctx context.Context, _ *sdkmcp.CallToolRequest, in radarArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.finish(s.fetchRadar(ctx, in))
}

func (s *Server) fetchRadar(ctx context.Context, in radarArgs) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()

	cfg, err := config.Load()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}
	ch, err := cache.New()
	if err != nil {
		ch = cache.NewNoOp()
	}
	if err := s.acquire(); err != nil {
		return nil, err
	}
	defer s.release()

	loc, query, err := resolveToolLocation(ctx, cfg, in.Location, ch)
	if err != nil {
		return nil, err
	}

	var shot radarShot
	if s.radar != nil {
		opts := radar.Options{Product: radar.ProductCompositeReflectivity, RadiusKM: 200, Raw: in.Raw}
		if strings.TrimSpace(in.Product) != "" {
			opts.Product = radar.NormalizeProduct(in.Product)
		}
		if in.RadiusKM > 0 {
			opts.RadiusKM = in.RadiusKM
		}
		shot, err = s.radar(ctx, loc, opts, strings.TrimSpace(in.Station))
	} else {
		shot, err = liveRadar(ctx, cfg, loc, query, in, ch)
	}
	if err != nil {
		return nil, err
	}
	return radarToolJSON(shot, in.IncludeImage)
}

// liveRadar fetches the current frame only. It does not call RecentFrames,
// so a CLI default of six loop frames cannot leak into the tool result.
func liveRadar(ctx context.Context, cfg *config.Config, loc location.Location, query string, in radarArgs, ch *cache.Cache) (radarShot, error) {
	prov, err := radar.ForLocation(loc)
	if err != nil {
		return radarShot{}, err
	}
	rawArg := strings.TrimSpace(in.Location)
	stationID := strings.TrimSpace(in.Station)
	if stationID == "" && cfg != nil {
		stationID = cfg.GetEffectiveRadarStation(query, rawArg, loc.DisplayName)
	}
	if stationID != "" {
		sl, ok := prov.(radar.StationLookup)
		if !ok {
			return radarShot{}, fmt.Errorf("radar provider %q does not support station lookup", prov.Name())
		}
		st, lookupErr := sl.LookupStation(ctx, stationID, ch)
		if lookupErr != nil {
			return radarShot{}, fmt.Errorf("station lookup: %w", lookupErr)
		}
		loc.Lat = st.Lat
		loc.Lon = st.Lon
		loc.DisplayName = fmt.Sprintf("%s — %s", st.ID, st.Name)
	}

	product := radar.ProductCompositeReflectivity
	if strings.TrimSpace(in.Product) != "" {
		product = radar.NormalizeProduct(in.Product)
	} else if cfg != nil {
		if p := cfg.GetEffectiveRadarProduct(query, rawArg, loc.DisplayName); p != "" {
			product = radar.NormalizeProduct(p)
		}
	}
	radius := in.RadiusKM
	if radius <= 0 {
		radius = 200
		if cfg != nil {
			if r := cfg.GetEffectiveRadarRadius(query, rawArg, loc.DisplayName); r > 0 {
				radius = r
			}
		}
	}
	opts := radar.Options{Product: product, RadiusKM: radius, Raw: in.Raw}
	frame, err := prov.CurrentFrame(ctx, loc, opts, ch)
	if err != nil {
		return radarShot{}, err
	}
	if frame == nil {
		return radarShot{}, errors.New("radar: empty frame")
	}
	shot := radarShot{Out: radar.DescribeFrame(loc, opts, stationID, frame)}
	if !in.IncludeImage {
		return shot, nil
	}
	if frame.Img == nil {
		return radarShot{}, errors.New("radar: empty image")
	}
	img := frame.Img
	if !opts.Raw {
		img = radar.DrawCityLabels(img, frame.BBox)
	}
	png, err := radar.EncodePNG(img)
	if err != nil {
		return radarShot{}, err
	}
	shot.PNG = png
	return shot, nil
}

type radarMetaJSON struct {
	Product      string            `json:"product"`
	ProductLabel string            `json:"product_label"`
	Stations     []string          `json:"stations,omitempty"`
	BBox         *radar.JSONBBox   `json:"bbox,omitempty"`
	Center       *radar.JSONCenter `json:"center,omitempty"`
	RadiusKM     float64           `json:"radius_km"`
	ValidTime    string            `json:"valid_time"`
	Location     string            `json:"location"`
	Station      string            `json:"station,omitempty"`
	IsComposite  bool              `json:"is_composite,omitempty"`
	Raw          bool              `json:"raw,omitempty"`
}

type radarImageJSON struct {
	radarMetaJSON
	ImageBase64 string            `json:"image_base64"`
	Frames      []radar.JSONFrame `json:"frames"`
}

func radarMetaFrom(out radar.JSONRadarOutput) radarMetaJSON {
	return radarMetaJSON{
		Product:      out.Product,
		ProductLabel: out.ProductLabel,
		Stations:     out.Stations,
		BBox:         out.BBox,
		Center:       out.Center,
		RadiusKM:     out.RadiusKM,
		ValidTime:    out.ValidTime,
		Location:     out.Location,
		Station:      out.Station,
		IsComposite:  out.IsComposite,
		Raw:          out.Raw,
	}
}

// radarToolJSON returns metadata, or metadata plus one frame.
// ImageBase64 and Frames on shot.Out are ignored. A PNG or its base64
// encoding over RadarImageMaxBytes is payload_too_large and the bytes
// are not returned.
func radarToolJSON(shot radarShot, includeImage bool) (json.RawMessage, error) {
	meta := radarMetaFrom(shot.Out)
	if !includeImage {
		return mustJSON(meta)
	}
	if len(shot.PNG) == 0 {
		return nil, errors.New("radar: empty image")
	}
	b64Len := base64.StdEncoding.EncodedLen(len(shot.PNG))
	if len(shot.PNG) > RadarImageMaxBytes || b64Len > RadarImageMaxBytes {
		return nil, &toolError{
			Code: "payload_too_large",
			Message: fmt.Sprintf(
				"radar image exceeds %d byte cap (png %d bytes, base64 %d bytes)",
				RadarImageMaxBytes, len(shot.PNG), b64Len,
			),
		}
	}
	b64 := base64.StdEncoding.EncodeToString(shot.PNG)
	frame := radar.JSONFrame{ValidTime: meta.ValidTime, ImageBase64: b64}
	return mustJSON(radarImageJSON{
		radarMetaJSON: meta,
		ImageBase64:   b64,
		Frames:        []radar.JSONFrame{frame},
	})
}
