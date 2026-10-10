package store

import (
	"sync"
	"testing"
	"time"

	"github.com/mwirges/wx/linux/internal/cli"
	"github.com/mwirges/wx/linux/internal/fixture"
)

func TestCloseStopsRadarFetchAndLoop(t *testing.T) {
	backend := &fixture.Fake{}
	st := New(backend, nil, true)
	st.Favorites = []Favorite{{Name: "Home", Value: "Fort Wayne, IN"}}
	st.OpenDesk()
	st.RefreshRadar()
	if !has(backend.Names(), "radar") {
		t.Fatal("expected radar fetch")
	}
	st.LoopPlaying = true
	st.CloseDesk()
	if st.DeskOpen || st.LoopPlaying {
		t.Fatal("desk still open or looping")
	}
	if backend.Cancels() < 1 {
		t.Fatal("radar was not cancelled")
	}
	before := len(backend.Names())
	st.OnRadarTick()
	if len(backend.Names()) != before {
		t.Fatal("closed desk fetched radar")
	}
	st.OnMenuTick()
	names := backend.Names()
	if !has(names, "weather") {
		t.Fatal("menu tick skipped weather")
	}
	for _, name := range names[before:] {
		if name == "outlook" {
			t.Fatal("closed desk fetched outlook")
		}
	}
	if has(names, "chase") {
		t.Fatal("closed desk fetched chase")
	}
}

func TestOpenDeskRadarTickFetches(t *testing.T) {
	backend := &fixture.Fake{}
	st := New(backend, nil, true)
	st.OpenDesk()
	st.OnRadarTick()
	if !has(backend.Names(), "radar") {
		t.Fatal("open desk skipped radar")
	}
}

func TestClosedDeskSkipsExtras(t *testing.T) {
	backend := &fixture.Fake{}
	st := New(backend, nil, true)
	st.Favorites = []Favorite{{Name: "Home", Value: "Fort Wayne, IN"}}
	st.CloseDesk()
	st.RefreshDeskExtras()
	st.RefreshOutlook()
	st.RefreshChase()
	st.RefreshGrid()
	for _, name := range backend.Names() {
		if name == "outlook" || name == "chase" {
			t.Fatalf("fetched %s", name)
		}
	}
	for _, call := range backend.Calls {
		if call[0] == "weather" && call[3] == false {
			t.Fatal("grid fetch while closed")
		}
	}
}

func TestOpenDeskExtrasAndGridSkipHourly(t *testing.T) {
	backend := &fixture.Fake{}
	st := New(backend, nil, true)
	st.Favorites = []Favorite{{Name: "Home", Value: "Fort Wayne, IN"}}
	st.OpenDesk()
	st.RefreshDeskExtras()
	names := backend.Names()
	if !has(names, "outlook") || !has(names, "chase") {
		t.Fatalf("extras %v", names)
	}
	var grid [][]any
	for _, call := range backend.Calls {
		if call[0] == "weather" && call[3] == false {
			grid = append(grid, call)
		}
		if call[0] == "weather" && call[3] == true {
			t.Fatal("extras requested hourly")
		}
	}
	if len(grid) != 1 || grid[0][1] != "Fort Wayne, IN" {
		t.Fatalf("grid %#v", grid)
	}
}

func TestStaleRadarIsDroppedAfterClose(t *testing.T) {
	st := New(&fixture.Fake{}, nil, true)
	st.OpenDesk()
	st.radarGen = 4
	st.CloseDesk()
	st.RadarFrames = nil
	gen := 4
	st.DeskOpen = false
	st.radarGen = 5
	if gen == st.radarGen && st.DeskOpen {
		st.RadarFrames = []Frame{{PNG: []byte("nope")}}
	}
	if len(st.RadarFrames) != 0 {
		t.Fatal("stale radar applied")
	}
}

func TestMenuRefreshRequestsHourly(t *testing.T) {
	backend := &fixture.Fake{}
	st := New(backend, nil, true)
	st.Refresh()
	call := backend.Calls[0]
	if call[0] != "weather" || call[3] != true {
		t.Fatalf("weather %#v", call)
	}
	cond, _ := st.Payload["conditions"].(map[string]any)
	if cond["location"] != "Fort Wayne, IN" {
		t.Fatalf("location %#v", cond["location"])
	}
}

func TestReopenBuildsAFreshOpenFlag(t *testing.T) {
	st := New(&fixture.Fake{}, nil, true)
	st.OpenDesk()
	st.CloseDesk()
	if st.DeskOpen {
		t.Fatal("still open")
	}
	st.OpenDesk()
	if !st.DeskOpen {
		t.Fatal("did not reopen")
	}
}

func TestLoopAdvanceStopsWhenDeskCloses(t *testing.T) {
	backend := &fixture.Fake{}
	st := New(backend, nil, true)
	st.OpenDesk()
	st.RefreshRadar()
	st.LoopPlaying = true
	st.FrameIndex = 0
	st.AdvanceFrame()
	if st.FrameIndex != 1 {
		t.Fatalf("frame %d", st.FrameIndex)
	}
	st.CloseDesk()
	st.AdvanceFrame()
	if st.LoopPlaying {
		t.Fatal("loop kept playing")
	}
}

func TestCloseDiscardsInflightRadar(t *testing.T) {
	backend := &blockingFake{started: make(chan struct{}), release: make(chan struct{})}
	st := New(backend, nil, false)
	st.OpenDesk()
	st.RefreshRadar()
	select {
	case <-backend.started:
	case <-time.After(2 * time.Second):
		t.Fatal("radar did not start")
	}
	st.CloseDesk()
	st.WaitIdle()
	if st.DeskOpen {
		t.Fatal("desk open")
	}
	if len(st.RadarFrames) != 0 {
		t.Fatal("inflight radar applied")
	}
	if backend.Cancels() < 1 {
		t.Fatal("did not cancel")
	}
}

type blockingFake struct {
	fixture.Fake
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *blockingFake) FetchRadar(location, product string, radius float64, bbox string, frames int) (map[string]any, error) {
	b.Add("radar", location, product, radius, frames)
	b.once.Do(func() { close(b.started) })
	<-b.release
	if b.Cancels() > 0 {
		return nil, cli.Cancelled{}
	}
	return b.Fake.FetchRadar(location, product, radius, bbox, frames)
}

func (b *blockingFake) CancelRadar() {
	b.Fake.CancelRadar()
	select {
	case <-b.release:
	default:
		close(b.release)
	}
}

func has(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}
