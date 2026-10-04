package ui

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/mwirges/wx/linux/internal/fixture"
	"github.com/mwirges/wx/linux/internal/store"
)

func testApp() fyne.App {
	application := test.NewApp()
	application.Settings().SetTheme(NewTheme())
	return application
}

func TestDailySurfaceShowsCLIJSON(t *testing.T) {
	application := testApp()
	defer application.Quit()
	st := store.New(&fixture.Fake{}, nil, true)
	st.Payload = fixture.Payload
	desk := NewDesk(application, st, nil)
	defer desk.Window.Close()
	if !st.DeskOpen {
		t.Fatal("desk did not open")
	}
	obs := desk.SectionText("wx.observation")
	for _, want := range []string{"FORT WAYNE, IN", "72°", "Partly Cloudy"} {
		if !contains(obs, want) {
			t.Fatalf("observation %q missing %q", obs, want)
		}
	}
	now := desk.SectionText("wx.nowcast")
	for _, want := range []string{"ALL CLEAR // DRY", "Dry next 6h"} {
		if !contains(now, want) {
			t.Fatalf("nowcast %q missing %q", now, want)
		}
	}
	if alerts := desk.SectionText("wx.alerts"); !contains(alerts, "SEVERE THUNDERSTORM WARNING") {
		t.Fatalf("alerts %q", alerts)
	}
	if hours := desk.SectionText("wx.hourly.strip"); !contains(hours, "72°") {
		t.Fatalf("hours %q", hours)
	}
	days := desk.SectionText("wx.days")
	for _, want := range []string{"This Afternoon", "Partly Sunny", "74°"} {
		if !contains(days, want) {
			t.Fatalf("days %q missing %q", days, want)
		}
	}
}

func TestClosingWindowStopsRadarAndReleasesIt(t *testing.T) {
	application := testApp()
	defer application.Quit()
	backend := &fixture.Fake{}
	st := store.New(backend, nil, true)
	st.Payload = fixture.Payload
	closed := 0
	desk := NewDesk(application, st, func() { closed++ })
	st.RefreshRadar()
	st.LoopPlaying = true
	if len(st.RadarFrames) == 0 {
		t.Fatal("no radar frames")
	}
	desk.Window.Close()
	if closed != 1 {
		t.Fatalf("closed %d", closed)
	}
	if st.DeskOpen || st.LoopPlaying {
		t.Fatal("desk still open or looping")
	}
	if backend.Cancels() < 1 {
		t.Fatal("radar was not cancelled")
	}
	before := len(backend.Names())
	st.OnRadarTick()
	if len(backend.Names()) != before {
		t.Fatal("radar tick after close")
	}
}

func TestFormatDropdownChangesStatusLine(t *testing.T) {
	application := testApp()
	defer application.Quit()
	st := store.New(&fixture.Fake{}, nil, true)
	st.Payload = fixture.Payload
	desk := NewDesk(application, st, nil)
	defer desk.Window.Close()
	desk.Format.SetSelectedIndex(2)
	text := desk.SectionText("wx.status")
	if !contains(text, "🔴 [FWA] 72° ↘12mph") {
		t.Fatalf("tactical %q", text)
	}
	desk.Format.SetSelectedIndex(0)
	text = desk.SectionText("wx.status")
	if !contains(text, "🔴 72°") || contains(text, "[FWA]") {
		t.Fatalf("compact %q", text)
	}
}

func TestDualPanePutsRadarBesideTelemetryUntil760(t *testing.T) {
	application := testApp()
	defer application.Quit()
	st := store.New(&fixture.Fake{}, nil, true)
	pane := NewDualPane(NewDailySurface().Root, NewRadarPanel(st, nil).Root)
	pane.ApplyWidth(900)
	if !pane.Horizontal() {
		t.Fatal("900 was vertical")
	}
	pane.ApplyWidth(760)
	if !pane.Horizontal() {
		t.Fatal("760 was vertical")
	}
	pane.ApplyWidth(759)
	if pane.Horizontal() {
		t.Fatal("759 was horizontal")
	}
	if findID(pane, "wx.observation") == nil || findID(pane, "wx.radar") == nil {
		t.Fatal("dual pane lost a section")
	}
}

func TestDeskOpensOnTacticalWithObservationAndRadar(t *testing.T) {
	application := testApp()
	defer application.Quit()
	st := store.New(&fixture.Fake{}, nil, true)
	st.Payload = fixture.Payload
	desk := NewDesk(application, st, nil)
	defer desk.Window.Close()
	if st.Tab != "dual" || desk.Visible() != "dual" {
		t.Fatalf("tab %s visible %s", st.Tab, desk.Visible())
	}
	if obs := desk.SectionText("wx.observation"); !contains(obs, "FORT WAYNE, IN") {
		t.Fatalf("obs %q", obs)
	}
	if radar := desk.SectionText("wx.radar"); !contains(radar, "NO FRAME") {
		t.Fatalf("radar %q", radar)
	}
	desk.TapMode("weather")
	if desk.Visible() != "weather" {
		t.Fatalf("visible %s", desk.Visible())
	}
	if obs := desk.SectionText("wx.observation"); !contains(obs, "72°") {
		t.Fatalf("surface %q", obs)
	}
}

func TestRadarDisplaysCLIPNGAndExportCallsTheCLI(t *testing.T) {
	application := testApp()
	defer application.Quit()
	backend := &fixture.Fake{}
	st := store.New(backend, nil, true)
	st.OpenDesk()
	st.Location = "Fort Wayne, IN"
	raw, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
	if err != nil {
		t.Fatal(err)
	}
	frames := store.DecodeFrames(map[string]any{"image_base64": base64.StdEncoding.EncodeToString(raw), "valid_time": "2026-10-04T15:00:00Z"})
	if len(frames) != 1 || string(frames[0].PNG) != string(raw) {
		t.Fatalf("frames %#v", frames)
	}
	st.RadarFrames = frames
	panel := NewRadarPanel(st, nil)
	panel.Refresh(st)
	if panel.Picture.Image == nil {
		t.Fatal("png was not shown")
	}
	if panel.Frame.Text != "LIVE" || panel.Note.Text != "" {
		t.Fatalf("frame %q note %q", panel.Frame.Text, panel.Note.Text)
	}
	matted, ok := panel.Picture.Image.(*image.NRGBA)
	if !ok || matted.Bounds().Dx() != 1 || matted.NRGBAAt(0, 0).A != 255 {
		t.Fatalf("picture %#v", panel.Picture.Image)
	}
	clear := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	clear.SetNRGBA(1, 0, color.NRGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, clear); err != nil {
		t.Fatal(err)
	}
	st.RadarFrames = []store.Frame{{PNG: buf.Bytes(), Label: "LIVE"}}
	panel.Refresh(st)
	shown, ok := panel.Picture.Image.(*image.NRGBA)
	if !ok || shown.NRGBAAt(0, 0).A != 255 || shown.NRGBAAt(1, 0).R != 255 {
		t.Fatalf("transparent frame stayed blank: %#v", panel.Picture.Image)
	}
	st.RadarFrames = []store.Frame{{PNG: []byte("hello"), Label: "LIVE"}}
	panel.Refresh(st)
	if !contains(panel.Note.Text, "not a PNG") || panel.Picture.Image != nil {
		t.Fatalf("bad png note %q image %#v", panel.Note.Text, panel.Picture.Image)
	}
	panel.SavePNG("/tmp/wx-linux-radar.png")
	panel.SaveGIF("/tmp/wx-linux-radar.gif")
	if names := backend.Names(); len(names) != 2 || names[0] != "export-png" || names[1] != "export-gif" {
		t.Fatalf("calls %v", names)
	}
	if _, err := os.Stat("/tmp/wx-linux-radar.png"); err == nil {
		t.Fatal("desk wrote the png")
	}
	if _, err := os.Stat("/tmp/wx-linux-radar.gif"); err == nil {
		t.Fatal("desk wrote the gif")
	}
}

func TestProductChangeRefetchesThroughTheCLI(t *testing.T) {
	application := testApp()
	defer application.Quit()
	backend := &fixture.Fake{}
	st := store.New(backend, nil, true)
	desk := NewDesk(application, st, nil)
	defer desk.Window.Close()
	desk.RadarPanels[0].Product.SetSelectedIndex(3)
	if st.RadarProduct != "echo-tops" {
		t.Fatalf("product %s", st.RadarProduct)
	}
	found := false
	for _, call := range backend.Calls {
		if call[0] == "radar" && call[2] == "echo-tops" {
			found = true
		}
	}
	if !found {
		t.Fatalf("calls %#v", backend.Calls)
	}
}

func TestOutlookChaseClimateTropicsAndGridRenderCLIJSON(t *testing.T) {
	application := testApp()
	defer application.Quit()
	backend := &fixture.Fake{}
	st := store.New(backend, nil, true)
	st.Payload = fixture.Payload
	st.Outlook = map[string]any{
		"location": "Fort Wayne, IN",
		"pattern_shift": map[string]any{
			"has_shift": true,
			"summary":   "Cooling into the 8-14 day",
		},
		"drought": map[string]any{"status": "No Drought", "target": "Oct 2026"},
		"outlooks": []any{
			map[string]any{
				"horizon":            "6-10 Day",
				"temp_category":      "Above",
				"temp_probability":   55.0,
				"precip_category":    "Below",
				"precip_probability": 40.0,
			},
		},
	}
	st.Chase = map[string]any{
		"total_clusters": 1.0,
		"total_alerts":   4.0,
		"clusters": []any{
			map[string]any{
				"name":           "Northern Indiana",
				"score":          80.0,
				"total_alerts":   4.0,
				"primary_hazard": "Severe Thunderstorm Warning",
				"nearest_radar":  "KIWX",
				"center_lat":     41.1,
				"center_lon":     -85.14,
			},
		},
		"spc": map[string]any{"day1": map[string]any{"category": map[string]any{"name": "Slight Risk"}}},
	}
	st.Climate = map[string]any{
		"climate": map[string]any{
			"station_name":   "Fort Wayne Intl",
			"normals_period": "1991–2020",
			"today_normals": map[string]any{
				"normal_high_f": 68.0, "normal_low_f": 46.0, "normal_high_c": 20.0, "normal_low_c": 8.0,
			},
			"records": map[string]any{
				"record_high": map[string]any{"value_f": 92.0, "value_c": 33.0, "years": []any{1954.0}},
				"record_low":  map[string]any{"value_f": 28.0, "value_c": -2.0, "years": []any{1981.0}},
			},
			"departure": map[string]any{"summary": "+5.2°F Above Normal"},
		},
	}
	st.History = map[string]any{
		"summary": map[string]any{"days_count": 2.0},
		"days": []any{
			map[string]any{
				"date": "2026-10-03", "temperature_max_f": 70.0, "temperature_min_f": 48.0, "precipitation_in": 0.1,
				"temperature_max_c": 21.0, "temperature_min_c": 9.0, "precipitation_mm": 2.5,
			},
		},
	}
	st.Tropics = map[string]any{
		"tropics": map[string]any{
			"total_active": 1.0,
			"storms": []any{
				map[string]any{
					"name": "Milton", "category_label": "Category 3", "wind_speed_mph": 120.0,
					"wind_speed_kmh": 193.0, "movement_compass": "NE", "headline": "Moving northeast",
				},
			},
			"disturbances": []any{map[string]any{"name": "AL91", "chance_48h": 20.0, "chance_7d": 40.0}},
		},
	}
	st.Favorites = []store.Favorite{{Name: "Home", Value: "Fort Wayne, IN"}}
	st.GridCards = []store.GridCard{{
		LocationKey: "Fort Wayne, IN", DisplayName: "Home", Payload: fixture.Payload,
	}}
	desk := NewDesk(application, st, nil)
	defer desk.Window.Close()

	outlook := desk.SectionText("wx.outlooks")
	for _, want := range []string{"Cooling into the 8-14 day", "6-10 Day", "Above 55%", "Drought: No Drought (Oct 2026)"} {
		if !contains(outlook, want) {
			t.Fatalf("outlook %q missing %q", outlook, want)
		}
	}
	chase := desk.SectionText("wx.chase")
	for _, want := range []string{"Northern Indiana", "SPC Day 1: Slight Risk", "KIWX"} {
		if !contains(chase, want) {
			t.Fatalf("chase %q missing %q", chase, want)
		}
	}
	button := findButton(desk.Chase.Root, "Open on radar")
	if button == nil {
		t.Fatal("missing chase button")
	}
	button.OnTapped()
	if st.Tab != "radar" || st.RadarRadius != 250 || st.Location != "41.1000,-85.1400" {
		t.Fatalf("tab %s radius %v location %s", st.Tab, st.RadarRadius, st.Location)
	}
	if desk.Visible() != "radar" {
		t.Fatalf("visible %s", desk.Visible())
	}
	found := false
	for _, call := range backend.Calls {
		if call[0] == "radar" && call[3] == 250.0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("calls %#v", backend.Calls)
	}

	climate := desk.SectionText("wx.climate")
	for _, want := range []string{"FORT WAYNE INTL", "Normal high 68°F", "Record high 92°F", "+5.2°F Above Normal", "2026-10-03", "H 70°F"} {
		if !contains(climate, want) {
			t.Fatalf("climate %q missing %q", climate, want)
		}
	}
	tropics := desk.SectionText("wx.tropics")
	for _, want := range []string{"1 ACTIVE TROPICAL CYCLONE", "Milton", "120 mph", "AL91", "48h 20%"} {
		if !contains(tropics, want) {
			t.Fatalf("tropics %q missing %q", tropics, want)
		}
	}
	st.Units = "metric"
	desk.Tropics.Refresh(st)
	desk.Climate.Refresh(st)
	if tropics = desk.SectionText("wx.tropics"); !contains(tropics, "193 km/h") {
		t.Fatalf("metric tropics %q", tropics)
	}
	climate = desk.SectionText("wx.climate")
	if !contains(climate, "Normal high 20°C") || !contains(climate, "2.50 mm") {
		t.Fatalf("metric climate %q", climate)
	}
	grid := desk.SectionText("wx.grid")
	if !contains(grid, "Home") || !contains(grid, "72°") {
		t.Fatalf("grid %q", grid)
	}
	desk.Grid.Name.SetText("Cabin")
	desk.Grid.Value.SetText("46803")
	desk.Grid.AddBtn.OnTapped()
	pinned := false
	for _, fav := range st.Favorites {
		if fav.Value == "46803" {
			pinned = true
		}
	}
	if !pinned {
		t.Fatalf("favorites %#v", st.Favorites)
	}
	hourly := false
	for _, call := range backend.Calls {
		if call[0] == "weather" && call[1] == "46803" && call[3] == false {
			hourly = true
		}
	}
	if !hourly {
		t.Fatalf("grid calls %#v", backend.Calls)
	}
}

func TestBootOpensDeskAndCloseStopsRadar(t *testing.T) {
	defer fixture.GuardHome(t)()
	application := testApp()
	dir := t.TempDir()
	binary := filepath.Join(dir, "wx")
	script := `#!/usr/bin/env python3
import json, sys
cmd = sys.argv[1] if len(sys.argv) > 1 and not sys.argv[1].startswith("-") else ""
if cmd == "radar":
    json.dump({"product":"composite-reflectivity","product_label":"Composite","location":"Stub","valid_time":"2026-10-04T15:00:00Z","image_base64":"","frames":[]}, sys.stdout)
elif cmd == "outlook":
    json.dump({"location":"Stub","fetched_at":"2026-10-04T15:00:00Z","outlooks":[],"pattern_shift":{"has_shift":False,"summary":""}}, sys.stdout)
elif cmd == "chase":
    json.dump({"generated_at":"2026-10-04T15:00:00Z","total_alerts":0,"total_clusters":0,"clusters":[]}, sys.stdout)
else:
    json.dump({"conditions":{"station":"KFWA","location":"Fort Wayne, IN","temperature_f":72,"description":"Fair","condition_code":"clear-day","nowcast":{"headline":"Dry next 6h","is_active_precip":False,"summary":"Dry","primary_phase":"none","intervals":[]}},"forecast":{"periods":[{"name":"Tonight","temperature_f":60,"short_description":"Clear"}],"hourly":[{"name":"4 PM","start_time":"2026-10-04T20:00:00Z","temperature_f":70}]},"alerts":[{"event":"Wind Advisory","severity":"Minor","headline":"Breezy"}]}, sys.stdout)
sys.exit(0)
`
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WX_CONFIG", filepath.Join(dir, "config.json"))
	t.Setenv("WX_BINARY", binary)
	t.Setenv("WX_REFRESH_SECONDS", "3600")
	t.Setenv("WX_RADAR_REFRESH_SECONDS", "3600")
	shell := NewShell(application, newBootCLI(binary), newBootPrefs(filepath.Join(dir, "config.json")))
	shell.Store.Sync = true
	defer shell.Quit()
	shell.Start(true)
	if shell.Desk == nil {
		t.Fatal("desk did not open")
	}
	text := shell.Desk.SectionText("wx.status")
	if !contains(text, "🟠") {
		t.Fatalf("status %q", text)
	}
	if !shell.Store.DeskOpen {
		t.Fatal("store desk closed")
	}
	shell.Desk.Window.Close()
	if shell.Desk != nil {
		t.Fatal("window was kept")
	}
	if shell.Store.DeskOpen || shell.Store.LoopPlaying {
		t.Fatal("close did not stop the desk")
	}
}

func TestCityFieldStaysBelowTheTitleAndRadarFillsItsPane(t *testing.T) {
	application := testApp()
	defer application.Quit()
	st := store.New(&fixture.Fake{}, nil, true)
	st.Payload = fixture.Payload
	st.RadarFrames = []store.Frame{{PNG: solidPNG(t, 40, 30, color.NRGBA{R: 255, G: 32, B: 32, A: 255}), Label: "LIVE", Live: true}}
	desk := NewDesk(application, st, nil)
	defer desk.Window.Close()

	canvas := software.NewCanvas()
	canvas.SetPadded(false)
	canvas.SetContent(desk.Window.Content())
	canvas.Resize(fyne.NewSize(980, 760))

	content := canvas.Content()
	title := findRich(content, "ATMOSPHERIC TELEMETRY CONSOLE")
	if title == nil {
		t.Fatal("missing title")
	}
	titleAt, ok := originOf(content, title)
	locAt, locOK := originOf(content, desk.Location)
	if !ok || !locOK {
		t.Fatal("missing title or city field")
	}
	if locAt.Y < titleAt.Y+title.Size().Height-1 {
		t.Fatalf("city field overlaps the title: city %v title %v h %.1f", locAt, titleAt, title.Size().Height)
	}
	picture := desk.RadarPanels[0].Picture
	if picture.Size().Height < 360 {
		t.Fatalf("radar picture is a short strip: %v", picture.Size())
	}
	picAt, ok := originOf(content, picture)
	if !ok {
		t.Fatal("missing picture")
	}
	img := canvas.Capture().(*image.NRGBA)
	red := 0
	bleed := 0
	for y := int(picAt.Y); y < int(picAt.Y+picture.Size().Height); y++ {
		for x := int(picAt.X); x < int(picAt.X+picture.Size().Width); x++ {
			if !image.Pt(x, y).In(img.Bounds()) {
				continue
			}
			c := img.NRGBAAt(x, y)
			if c.R > 200 && c.G < 80 && c.B < 80 {
				red++
			}
			if x < int(picAt.X)+12 && c.G > 180 && c.B > 180 && c.R < 120 {
				bleed++
			}
		}
	}
	if red < 80000 {
		t.Fatalf("radar pane stayed blank: %d red pixels, picture %v at %v", red, picture.Size(), picAt)
	}
	if bleed > 20 {
		t.Fatalf("left column cuts into the radar: %d pixels", bleed)
	}
}

func solidPNG(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	drawFill(img, c)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func drawFill(img *image.NRGBA, c color.NRGBA) {
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

func findRich(root fyne.CanvasObject, text string) *widget.RichText {
	var found *widget.RichText
	walk(root, func(c fyne.CanvasObject) {
		if found != nil {
			return
		}
		if rt, ok := c.(*widget.RichText); ok && rt.String() == text {
			found = rt
		}
	})
	return found
}

func originOf(root, target fyne.CanvasObject) (fyne.Position, bool) {
	var found fyne.Position
	var ok bool
	var visit func(fyne.CanvasObject, fyne.Position)
	visit = func(o fyne.CanvasObject, at fyne.Position) {
		if o == nil || ok {
			return
		}
		at = at.Add(o.Position())
		if o == target {
			found = at
			ok = true
			return
		}
		switch w := o.(type) {
		case *fyne.Container:
			for _, child := range w.Objects {
				visit(child, at)
			}
		case *Section:
			visit(w.Obj, at)
		case *container.Scroll:
			visit(w.Content, at)
		case *container.Clip:
			visit(w.Content, at)
		case *container.Split:
			visit(w.Leading, at)
			visit(w.Trailing, at)
		case *DualPane:
			visit(w.split, at)
		}
	}
	visit(root, fyne.NewPos(0, 0))
	return found, ok
}

func contains(text, needle string) bool {
	return strings.Contains(text, needle)
}
