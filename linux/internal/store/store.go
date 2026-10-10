// Package store fetches through the wx CLI and keeps the desk model.
// Views stay dumb. Closing the desk stops radar, the loop, and desk-only fetches.
package store

import (
	"encoding/base64"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mwirges/wx/linux/internal/cli"
	"github.com/mwirges/wx/linux/internal/present"
)

// RadarTabs are the surfaces that show the radar picture.
var RadarTabs = map[string]bool{"dual": true, "radar": true}

// Backend is the wx CLI. The store does not fetch weather itself.
type Backend interface {
	FetchWeather(location, units string, hourly bool) (map[string]any, error)
	FetchRadar(location, product string, radius float64, bbox string, frames int) (map[string]any, error)
	CancelRadar()
	ExportPNG(path, location, product string, radius float64) (string, error)
	ExportGIF(path, location, product string, radius float64, frames int) (string, error)
	FetchOutlook(location string) (map[string]any, error)
	FetchChase() (map[string]any, error)
	FetchClimate(location, units string) (map[string]any, error)
	FetchHistory(location, units string, days int) (map[string]any, error)
	FetchTropics(location, units, storm string) (map[string]any, error)
}

// Preferences is the desk config file.
type Preferences interface {
	Load() map[string]any
	SetMenuBarFormat(string) error
	SetLocationAndUnits(location, units string) error
	AddRecent(string) error
	AddFavorite(name, value string) error
	RemoveFavorite(string) error
}

// Frame is one PNG the CLI already composited.
type Frame struct {
	ValidTime string
	PNG       []byte
	Label     string
	Live      bool
}

// GridCard is one pinned location.
type GridCard struct {
	LocationKey string
	DisplayName string
	Payload     map[string]any
	Loading     bool
	Error       string
}

// Favorite is a pinned name and place.
type Favorite struct {
	Name  string
	Value string
}

// Store is the desk model.
type Store struct {
	backend Backend
	prefs   Preferences
	Sync    bool
	Deliver func(func())

	Payload         map[string]any
	NowcastPayload  map[string]any
	Outlook         map[string]any
	Chase           map[string]any
	Climate         map[string]any
	History         map[string]any
	Tropics         map[string]any
	Radar           map[string]any
	RadarFrames     []Frame
	FrameIndex      int
	LoopPlaying     bool
	RadarLoading    bool
	RadarError      string
	Loading         bool
	Error           string
	OutlookError    string
	ChaseError      string
	ClimateError    string
	HistoryError    string
	TropicsError    string
	Location        string
	Units           string
	MenuBarFormat   string
	Favorites       []Favorite
	Recents         []string
	GridCards       []GridCard
	Tab             string
	DeskOpen        bool
	RadarProduct    string
	RadarRadius     float64
	HistoryDays     int
	LastRefreshed   time.Time
	RefreshInterval float64
	RadarInterval   float64

	wxGen     int
	radarGen  int
	listeners []listener
	nextID    int
	wg        sync.WaitGroup
	mu        sync.Mutex
}

type listener struct {
	id int
	fn func(string)
}

// New builds a store. prefs may be nil. sync runs fetches on the caller.
func New(backend Backend, prefs Preferences, sync bool) *Store {
	s := &Store{
		backend:         backend,
		prefs:           prefs,
		Sync:            sync,
		Units:           "imperial",
		MenuBarFormat:   "standard",
		Tab:             "dual",
		RadarProduct:    "composite-reflectivity",
		RadarRadius:     200,
		HistoryDays:     14,
		RefreshInterval: envSeconds("WX_REFRESH_SECONDS", 5*60),
		RadarInterval:   envSeconds("WX_RADAR_REFRESH_SECONDS", 2*60),
	}
	if prefs != nil {
		s.LoadPrefs()
	}
	return s
}

// Subscribe registers fn and returns an id for Unsubscribe.
func (s *Store) Subscribe(fn func(string)) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	id := s.nextID
	s.listeners = append(s.listeners, listener{id: id, fn: fn})
	return id
}

// Unsubscribe removes a listener.
func (s *Store) Unsubscribe(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.listeners[:0]
	for _, ln := range s.listeners {
		if ln.id != id {
			kept = append(kept, ln)
		}
	}
	s.listeners = kept
}

func (s *Store) emit(topic string) {
	s.mu.Lock()
	ls := append([]listener(nil), s.listeners...)
	s.mu.Unlock()
	for _, ln := range ls {
		ln.fn(topic)
	}
}

// LoadPrefs copies desk fields out of the config file.
func (s *Store) LoadPrefs() {
	if s.prefs == nil {
		return
	}
	cfg := s.prefs.Load()
	s.Location = present.AsString(cfg["default_location"])
	units := strings.TrimSpace(present.AsString(cfg["units"]))
	if units == "metric" {
		s.Units = "metric"
	} else {
		s.Units = "imperial"
	}
	format := strings.ToLower(strings.TrimSpace(present.AsString(cfg["menu_bar_format"])))
	if format != "compact" && format != "standard" && format != "tactical" {
		format = "standard"
	}
	s.MenuBarFormat = format
	s.Favorites = favoritesFrom(cfg["favorites"])
	s.Recents = recentsFrom(cfg["recent_locations"])
}

// SetMenuBarFormat stores compact, standard, or tactical.
func (s *Store) SetMenuBarFormat(format string) {
	if format != "compact" && format != "standard" && format != "tactical" {
		return
	}
	if format == s.MenuBarFormat {
		return
	}
	s.MenuBarFormat = format
	if s.prefs != nil {
		_ = s.prefs.SetMenuBarFormat(format)
	}
	s.emit("format")
}

// SetUnits saves the unit system and refetches.
func (s *Store) SetUnits(units string) {
	if units == "metric" {
		s.Units = "metric"
	} else {
		s.Units = "imperial"
	}
	if s.prefs != nil {
		_ = s.prefs.SetLocationAndUnits(strings.TrimSpace(s.Location), s.Units)
	}
	s.Refresh()
}

// SelectLocation switches the place, refetches, and continues desk work while the window is open.
func (s *Store) SelectLocation(location string) {
	clean := strings.TrimSpace(location)
	if clean == "" {
		return
	}
	s.Location = clean
	s.wxGen++
	if s.prefs != nil {
		_ = s.prefs.AddRecent(clean)
		_ = s.prefs.SetLocationAndUnits(clean, s.Units)
		if cfg := s.prefs.Load(); cfg != nil {
			if recents := recentsFrom(cfg["recent_locations"]); recents != nil {
				s.Recents = recents
			}
		}
	}
	s.Refresh()
	if s.DeskOpen {
		s.RefreshDeskExtras()
		if RadarTabs[s.Tab] {
			s.RefreshRadar()
		}
	}
}

// AddFavorite pins a place.
func (s *Store) AddFavorite(name, value string) {
	if s.prefs != nil {
		_ = s.prefs.AddFavorite(name, value)
		s.LoadPrefs()
	} else {
		s.Favorites = append(s.Favorites, Favorite{Name: strings.TrimSpace(name), Value: strings.TrimSpace(value)})
	}
	s.emit("favorites")
}

// RemoveFavorite drops a pin by name or value.
func (s *Store) RemoveFavorite(nameOrValue string) {
	if s.prefs != nil {
		_ = s.prefs.RemoveFavorite(nameOrValue)
		s.LoadPrefs()
	} else {
		key := strings.ToLower(strings.TrimSpace(nameOrValue))
		kept := s.Favorites[:0]
		for _, fav := range s.Favorites {
			if strings.ToLower(fav.Name) == key || strings.ToLower(fav.Value) == key {
				continue
			}
			kept = append(kept, fav)
		}
		s.Favorites = kept
	}
	s.emit("favorites")
}

// OpenDesk marks the window open.
func (s *Store) OpenDesk() {
	s.DeskOpen = true
	s.emit("desk-opened")
}

// CloseDesk matches the Mac window close. The menu-bar refresh keeps running.
// Radar fetch, the loop, and desk-only outlook, chase, and grid work stop.
func (s *Store) CloseDesk() {
	s.mu.Lock()
	s.DeskOpen = false
	s.LoopPlaying = false
	s.RadarLoading = false
	s.radarGen++
	s.mu.Unlock()
	if s.backend != nil {
		s.backend.CancelRadar()
	}
	s.emit("desk-closed")
}

// SetTab switches pages and fetches that page's JSON while the desk is open.
func (s *Store) SetTab(tab string) {
	s.Tab = tab
	s.emit("tab")
	if !s.DeskOpen {
		return
	}
	if RadarTabs[tab] && len(s.RadarFrames) == 0 && !s.RadarLoading {
		s.RefreshRadar()
	}
	if tab == "outlooks" && s.Outlook == nil {
		s.RefreshOutlook()
	}
	if tab == "chase" && s.Chase == nil {
		s.RefreshChase()
	}
	if tab == "climate" && (s.Climate == nil || s.History == nil) {
		s.RefreshClimate()
		s.RefreshHistory()
	}
	if tab == "tropics" && s.Tropics == nil {
		s.RefreshTropics("")
	}
	if tab == "grid" && len(s.GridCards) == 0 {
		s.RefreshGrid()
	}
}

// OnMenuTick is the tray timer. It keeps running while the desk is closed.
func (s *Store) OnMenuTick() {
	s.Refresh()
	s.RefreshDeskExtras()
}

// OnRadarTick fetches only while the desk window is open.
func (s *Store) OnRadarTick() {
	if s.DeskOpen {
		s.RefreshRadar()
	}
}

// RefreshDeskExtras loads outlook, chase, and the favorites grid.
func (s *Store) RefreshDeskExtras() {
	if !s.DeskOpen {
		return
	}
	s.RefreshOutlook()
	s.RefreshChase()
	if len(s.Favorites) > 0 {
		s.RefreshGrid()
	}
}

// Refresh loads the observation, forecast, alerts, and hourly strip.
func (s *Store) Refresh() {
	s.Loading = true
	gen := s.wxGen
	loc := strings.TrimSpace(s.Location)
	units := s.Units
	s.spawn(func() {
		payload, err := s.backend.FetchWeather(loc, units, true)
		s.deliver(func() {
			if gen != s.wxGen {
				return
			}
			s.Loading = false
			if err != nil {
				s.Error = err.Error()
			} else {
				s.Error = ""
				s.Payload = payload
				s.LastRefreshed = time.Now()
			}
			s.emit("weather")
		})
	})
}

// RefreshRadar loads PNG frames from wx radar --json.
func (s *Store) RefreshRadar() {
	if !s.DeskOpen {
		return
	}
	s.mu.Lock()
	s.RadarLoading = true
	s.RadarError = ""
	s.radarGen++
	gen := s.radarGen
	s.mu.Unlock()
	loc := strings.TrimSpace(s.Location)
	product := s.RadarProduct
	radius := s.RadarRadius
	s.spawn(func() {
		payload, err := s.backend.FetchRadar(loc, product, radius, "", 8)
		if cli.IsCancel(err) {
			payload, err = nil, nil
		}
		s.deliver(func() {
			s.mu.Lock()
			stale := gen != s.radarGen || !s.DeskOpen
			if stale {
				s.RadarLoading = false
				s.mu.Unlock()
				return
			}
			s.RadarLoading = false
			if err != nil {
				s.RadarError = err.Error()
				s.mu.Unlock()
				s.emit("radar")
				return
			}
			if payload == nil {
				s.mu.Unlock()
				return
			}
			s.Radar = payload
			s.RadarFrames = DecodeFrames(payload)
			if len(s.RadarFrames) > 0 {
				s.FrameIndex = len(s.RadarFrames) - 1
				s.RadarError = ""
			} else {
				s.RadarError = "Failed to decode radar frame telemetry"
			}
			s.mu.Unlock()
			s.emit("radar")
		})
	})
}

// RefreshOutlook loads CPC JSON.
func (s *Store) RefreshOutlook() {
	s.fetchExtra("outlook", func(b Backend, loc, units string) (map[string]any, error) {
		return b.FetchOutlook(loc)
	})
}

// RefreshChase loads storm-chase clusters.
func (s *Store) RefreshChase() {
	s.fetchExtra("chase", func(b Backend, loc, units string) (map[string]any, error) {
		return b.FetchChase()
	})
}

// RefreshClimate loads the climate report.
func (s *Store) RefreshClimate() {
	s.fetchExtra("climate", func(b Backend, loc, units string) (map[string]any, error) {
		return b.FetchClimate(loc, units)
	})
}

// RefreshHistory loads the recent daily history.
func (s *Store) RefreshHistory() {
	days := s.HistoryDays
	s.fetchExtra("history", func(b Backend, loc, units string) (map[string]any, error) {
		return b.FetchHistory(loc, units, days)
	})
}

// RefreshTropics loads the tropical tracker. storm may be empty.
func (s *Store) RefreshTropics(storm string) {
	s.fetchExtra("tropics", func(b Backend, loc, units string) (map[string]any, error) {
		return b.FetchTropics(loc, units, storm)
	})
}

// RefreshGrid loads each favorite without the hourly strip.
func (s *Store) RefreshGrid() {
	if !s.DeskOpen {
		return
	}
	favs := append([]Favorite(nil), s.Favorites...)
	if len(favs) == 0 {
		s.GridCards = nil
		s.emit("grid")
		return
	}
	units := s.Units
	cards := make([]GridCard, 0, len(favs))
	for _, fav := range favs {
		key := fav.Value
		if key == "" {
			key = fav.Name
		}
		name := fav.Name
		if name == "" {
			name = key
		}
		cards = append(cards, GridCard{LocationKey: key, DisplayName: name, Loading: true})
	}
	s.GridCards = cards
	s.emit("grid")
	s.spawn(func() {
		updated := make([]GridCard, len(cards))
		copy(updated, cards)
		for i, card := range cards {
			payload, err := s.backend.FetchWeather(card.LocationKey, units, false)
			updated[i].Loading = false
			if err != nil {
				updated[i].Error = err.Error()
			} else {
				updated[i].Payload = payload
			}
		}
		s.deliver(func() {
			if !s.DeskOpen {
				return
			}
			s.GridCards = updated
			s.emit("grid")
		})
	})
}

// ToggleLoop starts or stops the frame cycle. A closed desk stays stopped.
func (s *Store) ToggleLoop() {
	if !s.DeskOpen {
		s.LoopPlaying = false
		s.emit("loop")
		return
	}
	s.LoopPlaying = !s.LoopPlaying
	s.emit("loop")
}

// StopLoop halts the animation.
func (s *Store) StopLoop() {
	s.LoopPlaying = false
	s.emit("loop")
}

// StepFrame moves one frame and stops the loop.
func (s *Store) StepFrame(delta int) {
	s.LoopPlaying = false
	if len(s.RadarFrames) == 0 {
		return
	}
	n := len(s.RadarFrames)
	s.FrameIndex = (s.FrameIndex + delta) % n
	if s.FrameIndex < 0 {
		s.FrameIndex += n
	}
	s.emit("radar")
}

// AdvanceFrame steps the loop forward. A closed desk stops it.
func (s *Store) AdvanceFrame() {
	if !s.LoopPlaying || !s.DeskOpen || len(s.RadarFrames) < 2 {
		s.LoopPlaying = false
		return
	}
	s.FrameIndex = (s.FrameIndex + 1) % len(s.RadarFrames)
	s.emit("radar")
}

// CurrentFrame is the PNG on screen.
func (s *Store) CurrentFrame() *Frame {
	if len(s.RadarFrames) == 0 {
		return nil
	}
	idx := s.FrameIndex
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s.RadarFrames) {
		idx = len(s.RadarFrames) - 1
	}
	frame := s.RadarFrames[idx]
	return &frame
}

// ExportPNG asks the CLI to write a PNG.
func (s *Store) ExportPNG(path string) (string, error) {
	return s.backend.ExportPNG(path, strings.TrimSpace(s.Location), s.RadarProduct, s.RadarRadius)
}

// ExportGIF asks the CLI to write a GIF. The desk does not composite frames.
func (s *Store) ExportGIF(path string) (string, error) {
	frames := len(s.RadarFrames)
	if frames < 2 {
		frames = 8
	}
	return s.backend.ExportGIF(path, strings.TrimSpace(s.Location), s.RadarProduct, s.RadarRadius, frames)
}

// ChaseToRadar centers the radar on a cluster.
func (s *Store) ChaseToRadar(lat, lon float64) {
	s.RadarRadius = 250
	s.Tab = "radar"
	s.emit("tab")
	s.SelectLocation(strconv.FormatFloat(lat, 'f', 4, 64) + "," + strconv.FormatFloat(lon, 'f', 4, 64))
}

// Nowcast returns the nowcast object from the CLI JSON.
func (s *Store) Nowcast() map[string]any {
	return present.NowcastOf(s.Payload, s.NowcastPayload)
}

// WaitIdle waits for asynchronous fetches started by this store.
func (s *Store) WaitIdle() { s.wg.Wait() }

func (s *Store) fetchExtra(attr string, call func(Backend, string, string) (map[string]any, error)) {
	if !s.DeskOpen {
		return
	}
	loc := strings.TrimSpace(s.Location)
	units := s.Units
	s.spawn(func() {
		payload, err := call(s.backend, loc, units)
		s.deliver(func() {
			if !s.DeskOpen {
				return
			}
			if err != nil {
				s.setExtraError(attr, err.Error())
			} else {
				s.setExtraError(attr, "")
				s.setExtra(attr, payload)
			}
			s.emit(attr)
		})
	})
}

func (s *Store) setExtra(attr string, payload map[string]any) {
	switch attr {
	case "outlook":
		s.Outlook = payload
	case "chase":
		s.Chase = payload
	case "climate":
		s.Climate = payload
	case "history":
		s.History = payload
	case "tropics":
		s.Tropics = payload
	}
}

func (s *Store) setExtraError(attr, msg string) {
	switch attr {
	case "outlook":
		s.OutlookError = msg
	case "chase":
		s.ChaseError = msg
	case "climate":
		s.ClimateError = msg
	case "history":
		s.HistoryError = msg
	case "tropics":
		s.TropicsError = msg
	}
}

func (s *Store) spawn(fn func()) {
	if s.Sync {
		fn()
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		fn()
	}()
}

func (s *Store) deliver(fn func()) {
	if s.Sync || s.Deliver == nil {
		fn()
		return
	}
	s.Deliver(fn)
}

// DecodeFrames reads PNG bytes from radar JSON. It does not draw the radar.
func DecodeFrames(payload map[string]any) []Frame {
	if payload == nil {
		return nil
	}
	if raw, ok := payload["frames"].([]any); ok && len(raw) > 0 {
		frames := make([]Frame, 0, len(raw))
		for idx, item := range raw {
			frame, ok := item.(map[string]any)
			if !ok {
				continue
			}
			blob := decodeB64(present.AsString(frame["image_base64"]))
			if len(blob) == 0 {
				continue
			}
			valid := present.AsString(frame["valid_time"])
			if valid == "" {
				valid = present.AsString(payload["valid_time"])
			}
			live := idx == len(raw)-1
			label := "F" + strconv.Itoa(idx+1)
			if live {
				label = "LIVE"
			}
			frames = append(frames, Frame{ValidTime: valid, PNG: blob, Label: label, Live: live})
		}
		return frames
	}
	blob := decodeB64(present.AsString(payload["image_base64"]))
	if len(blob) == 0 {
		return nil
	}
	return []Frame{{
		ValidTime: present.AsString(payload["valid_time"]),
		PNG:       blob,
		Label:     "LIVE",
		Live:      true,
	}}
}

func decodeB64(text string) []byte {
	if text == "" {
		return nil
	}
	blob, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil
	}
	return blob
}

func favoritesFrom(v any) []Favorite {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]Favorite, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, Favorite{Name: present.AsString(entry["name"]), Value: present.AsString(entry["value"])})
	}
	return out
}

func recentsFrom(v any) []string {
	switch raw := v.(type) {
	case []string:
		return append([]string{}, raw...)
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			out = append(out, present.AsString(item))
		}
		return out
	default:
		return nil
	}
}

func envSeconds(name string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
