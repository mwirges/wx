package ui

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"

	"github.com/mwirges/wx/linux/internal/present"
	"github.com/mwirges/wx/linux/internal/store"
	"github.com/mwirges/wx/linux/internal/tray"
)

// Shell is the desk process. The tray keeps refreshing after the window closes.
type Shell struct {
	App   fyne.App
	Store *store.Store
	Tray  *tray.Tray
	Desk  *Desk

	anchor   fyne.Window
	listenID int
	stop     chan struct{}
	stopOnce sync.Once
	loopStop chan struct{}
	mu       sync.Mutex
}

// NewShell builds a desk shell. Fetches run on the Fyne thread when they finish.
func NewShell(app fyne.App, backend store.Backend, prefs store.Preferences) *Shell {
	st := store.New(backend, prefs, false)
	st.Deliver = func(fn func()) { fyne.Do(fn) }
	return &Shell{App: app, Store: st}
}

// Start installs the tray, starts refresh timers, and optionally opens the desk.
// A missing StatusNotifier host does not stop the window.
func (s *Shell) Start(openDesk bool) {
	s.anchor = s.App.NewWindow("wx")
	s.Tray = tray.New(s.snapshot, s.onTray)
	s.listenID = s.Store.Subscribe(s.onStore)
	s.Tray.Install()
	s.stop = make(chan struct{})
	go s.tick(interval(s.Store.RefreshInterval), func() { fyne.Do(func() { s.Store.OnMenuTick() }) })
	go s.tick(interval(s.Store.RadarInterval), func() { fyne.Do(func() { s.Store.OnRadarTick() }) })
	s.Store.Refresh()
	if openDesk {
		s.ShowDesk()
		s.Store.RefreshDeskExtras()
	}
}

// ShowDesk presents the current window, or builds a new one.
func (s *Shell) ShowDesk() {
	if s.Desk != nil && s.Desk.Window != nil {
		s.Store.OpenDesk()
		s.Desk.Window.Show()
	} else {
		s.Desk = NewDesk(s.App, s.Store, s.forget)
	}
	if s.Store.DeskOpen && store.RadarTabs[s.Store.Tab] && len(s.Store.RadarFrames) == 0 && !s.Store.RadarLoading {
		s.Store.RefreshRadar()
	}
}

// Quit closes the desk, stops timers, and leaves the process.
func (s *Shell) Quit() {
	if s.Desk != nil {
		win := s.Desk.Window
		s.Desk.RequestClose()
		if win != nil {
			win.Close()
		}
	}
	s.Store.DeskOpen = false
	s.stopLoop()
	if s.stop != nil {
		s.stopOnce.Do(func() { close(s.stop) })
	}
	if s.Tray != nil {
		s.Tray.Shutdown()
	}
	s.App.Quit()
}

func (s *Shell) forget() {
	s.Desk = nil
	s.stopLoop()
}

func (s *Shell) onStore(topic string) {
	if s.Tray != nil {
		s.Tray.Sync()
	}
	if topic == "loop" || topic == "radar" || topic == "desk-closed" {
		s.syncLoop()
	}
}

func (s *Shell) onTray(action, target string) {
	fyne.Do(func() { s.handleTray(action, target) })
}

func (s *Shell) handleTray(action, target string) {
	switch action {
	case "quit":
		s.Quit()
	case "refresh":
		s.Store.Refresh()
		if s.Store.DeskOpen {
			s.Store.RefreshDeskExtras()
			if store.RadarTabs[s.Store.Tab] {
				s.Store.RefreshRadar()
			}
		}
	case "format":
		if target != "" {
			s.Store.SetMenuBarFormat(target)
		}
	case "location":
		if target != "" {
			s.Store.SelectLocation(target)
			s.ShowDesk()
		}
	case "grid":
		s.Store.SetTab("grid")
		s.ShowDesk()
	case "open-desk":
		s.ShowDesk()
	}
}

func (s *Shell) snapshot() tray.Snapshot {
	cond := present.Conditions(s.Store.Payload)
	location := present.AsString(cond["location"])
	if location == "" {
		location = s.Store.Location
	}
	var favorites []map[string]any
	for _, fav := range s.Store.Favorites {
		favorites = append(favorites, map[string]any{"name": fav.Name, "value": fav.Value})
	}
	var cards []map[string]string
	if len(s.Store.GridCards) > 0 {
		for _, card := range s.Store.GridCards {
			name := card.DisplayName
			if name == "" {
				continue
			}
			target := card.LocationKey
			if target == "" {
				target = name
			}
			cards = append(cards, map[string]string{"label": name, "target": target})
		}
		if len(cards) == 0 {
			cards = nil
		}
	}
	return tray.Snapshot{
		Payload:   s.Store.Payload,
		Units:     s.Store.Units,
		Format:    s.Store.MenuBarFormat,
		Error:     s.Store.Error,
		Location:  location,
		Favorites: favorites,
		Cards:     cards,
	}
}

func (s *Shell) tick(every time.Duration, fn func()) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			fn()
		}
	}
}

func (s *Shell) syncLoop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	want := s.Store.LoopPlaying && s.Store.DeskOpen && len(s.Store.RadarFrames) > 1
	if want && s.loopStop == nil {
		stop := make(chan struct{})
		s.loopStop = stop
		go s.loop(stop)
		return
	}
	if !want && s.loopStop != nil {
		close(s.loopStop)
		s.loopStop = nil
	}
}

func (s *Shell) loop(stop chan struct{}) {
	ticker := time.NewTicker(380 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			fyne.Do(func() {
				if !s.Store.LoopPlaying || !s.Store.DeskOpen {
					s.syncLoop()
					return
				}
				s.Store.AdvanceFrame()
			})
		}
	}
}

func (s *Shell) stopLoop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loopStop != nil {
		close(s.loopStop)
		s.loopStop = nil
	}
}

func interval(seconds float64) time.Duration {
	n := int(seconds)
	if n < 1 {
		n = 1
	}
	return time.Duration(n) * time.Second
}
