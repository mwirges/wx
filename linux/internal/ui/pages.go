package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mwirges/wx/linux/internal/present"
	"github.com/mwirges/wx/linux/internal/store"
)

// OutlookView renders CPC JSON from the CLI.
type OutlookView struct {
	Root fyne.CanvasObject
	body *fyne.Container
}

// NewOutlookView builds the outlook page.
func NewOutlookView() *OutlookView {
	v := &OutlookView{body: container.NewVBox()}
	v.Root = newSection("wx.outlooks", container.NewVBox(line("NOAA CLIMATE PREDICTION CENTER (CPC)", "kicker"), v.body))
	return v
}

// Refresh paints the outlook payload.
func (v *OutlookView) Refresh(st *store.Store) {
	clearBox(v.body)
	if st.OutlookError != "" {
		addLine(v.body, st.OutlookError, "alert")
	}
	payload := st.Outlook
	if len(payload) == 0 {
		addLine(v.body, "Outlooks load while the desk is open.", "muted")
		return
	}
	if place := present.AsString(payload["location"]); place != "" {
		addLine(v.body, strings.ToUpper(place), "kicker")
	}
	shift := present.Map(payload["pattern_shift"])
	if summary := present.AsString(shift["summary"]); summary != "" {
		role := "muted"
		if present.AsBool(shift["has_shift"]) {
			role = "watch"
		}
		addLine(v.body, summary, role)
	}
	drought := present.Map(payload["drought"])
	if status := present.AsString(drought["status"]); status != "" {
		extra := ""
		if target := present.AsString(drought["target"]); target != "" {
			extra = " (" + target + ")"
		}
		addLine(v.body, "Drought: "+status+extra, "muted")
	}
	for _, item := range present.Slice(payload["outlooks"]) {
		row := present.Map(item)
		if row == nil {
			continue
		}
		horizon := present.AsString(row["horizon"])
		if horizon == "" {
			horizon = "Outlook"
		}
		temp := strings.TrimSpace(orDash(present.AsString(row["temp_category"])) + " " + pct(row["temp_probability"]))
		precip := strings.TrimSpace(orDash(present.AsString(row["precip_category"])) + " " + pct(row["precip_probability"]))
		addLine(v.body, fmt.Sprintf("%s   Temp %s   Precip %s", horizon, temp, precip), "")
	}
}

// ChaseView renders national severe clusters.
type ChaseView struct {
	Root  fyne.CanvasObject
	body  *fyne.Container
	store *store.Store
}

// NewChaseView builds the chase page.
func NewChaseView(st *store.Store) *ChaseView {
	v := &ChaseView{body: container.NewVBox(), store: st}
	v.Root = newSection("wx.chase", container.NewVBox(line("REMOTE STORM CHASING // NATIONAL SEVERE CLUSTERS", "kicker"), v.body))
	return v
}

// Refresh paints chase JSON.
func (v *ChaseView) Refresh(st *store.Store) {
	clearBox(v.body)
	if st.ChaseError != "" {
		addLine(v.body, st.ChaseError, "alert")
	}
	payload := st.Chase
	if len(payload) == 0 {
		addLine(v.body, "Scanning CONUS alerts…", "muted")
		return
	}
	var clusters []map[string]any
	for _, item := range present.Slice(payload["clusters"]) {
		if cluster := present.Map(item); cluster != nil {
			clusters = append(clusters, cluster)
		}
	}
	total := len(clusters)
	if n, ok := present.AsFloat(payload["total_clusters"]); ok {
		total = int(n)
	}
	alerts := 0
	if n, ok := present.AsFloat(payload["total_alerts"]); ok {
		alerts = int(n)
	}
	role := "ok"
	if len(clusters) > 0 {
		role = "watch"
	}
	addLine(v.body, fmt.Sprintf("%d ACTIVE CLUSTERS · %d SEVERE CELLS", total, alerts), role)
	day1 := present.Map(present.Map(present.Map(payload["spc"])["day1"])["category"])
	if name := present.AsString(day1["name"]); name != "" {
		addLine(v.body, "SPC Day 1: "+name, "kicker")
	}
	if len(clusters) == 0 {
		addLine(v.body, "No active storm clusters.", "ok")
		return
	}
	for _, cluster := range clusters {
		block := container.NewVBox()
		addLine(block, present.AsString(cluster["name"]), "kicker")
		if present.AsString(cluster["name"]) == "" {
			clearBox(block)
			addLine(block, "Cluster", "kicker")
		}
		score := "—"
		if _, ok := cluster["score"]; ok && cluster["score"] != nil {
			score = whole(cluster["score"])
		}
		addLine(block, fmt.Sprintf("Score %s · %s alerts · %s", score, whole(cluster["total_alerts"]), present.AsString(cluster["primary_hazard"])), "muted")
		if radar := present.AsString(cluster["nearest_radar"]); radar != "" {
			addLine(block, "Radar "+radar, "tag")
		}
		lat, latOK := present.AsFloat(cluster["center_lat"])
		lon, lonOK := present.AsFloat(cluster["center_lon"])
		btn := widget.NewButton("Open on radar", nil)
		if latOK && lonOK {
			la, lo := lat, lon
			btn.OnTapped = func() { v.store.ChaseToRadar(la, lo) }
		}
		block.Add(btn)
		v.body.Add(block)
	}
	v.body.Refresh()
}

// ClimateView renders the climate report and recent history.
type ClimateView struct {
	Root fyne.CanvasObject
	body *fyne.Container
}

// NewClimateView builds the climate page.
func NewClimateView() *ClimateView {
	v := &ClimateView{body: container.NewVBox()}
	v.Root = newSection("wx.climate", container.NewVBox(line("CLIMATOLOGICAL OBSERVATION ARCHIVE", "kicker"), v.body))
	return v
}

// Refresh paints climate and history JSON.
func (v *ClimateView) Refresh(st *store.Store) {
	clearBox(v.body)
	if st.ClimateError != "" {
		addLine(v.body, st.ClimateError, "alert")
	}
	if st.HistoryError != "" {
		addLine(v.body, st.HistoryError, "alert")
	}
	report := present.Map(present.Map(st.Climate)["climate"])
	if len(report) > 0 {
		place := present.AsString(report["station_name"])
		if place == "" {
			place = present.AsString(report["location"])
		}
		if place == "" {
			place = present.AsString(present.Map(st.Climate)["location"])
		}
		addLine(v.body, strings.ToUpper(place), "kicker")
		if period := present.AsString(report["normals_period"]); period != "" {
			addLine(v.body, "Normals "+period, "tag")
		}
		normals := present.Map(report["today_normals"])
		high, highOK, low, lowOK, unit := normalPair(normals, st.Units)
		if highOK && lowOK {
			addLine(v.body, fmt.Sprintf("Normal high %.0f%s   low %.0f%s", high, unit, low, unit), "")
		} else if highOK {
			addLine(v.body, fmt.Sprintf("Normal high %.0f%s", high, unit), "")
		}
		records := present.Map(report["records"])
		recHigh := present.Map(records["record_high"])
		recLow := present.Map(records["record_low"])
		if st.Units == "metric" {
			addRecord(v.body, "Record high", recHigh["value_c"], "°C", recHigh)
			addRecord(v.body, "Record low", recLow["value_c"], "°C", recLow)
		} else {
			addRecord(v.body, "Record high", recHigh["value_f"], "°F", recHigh)
			addRecord(v.body, "Record low", recLow["value_f"], "°F", recLow)
		}
		if summary := present.AsString(present.Map(report["departure"])["summary"]); summary != "" {
			addLine(v.body, summary, "watch")
		}
	}
	history := st.History
	days := present.Slice(present.Map(history)["days"])
	if n, ok := present.AsFloat(present.Map(present.Map(history)["summary"])["days_count"]); ok && n != 0 {
		addLine(v.body, fmt.Sprintf("History %.0f days", n), "kicker")
	}
	if len(report) == 0 && len(days) == 0 {
		addLine(v.body, "Climate loads when this page is open.", "muted")
	}
	limit := len(days)
	if limit > 14 {
		limit = 14
	}
	for _, item := range days[:limit] {
		day := present.Map(item)
		if day == nil {
			continue
		}
		var temps []string
		unit, punit := "°F", "in"
		hiKey, loKey, pKey := "temperature_max_f", "temperature_min_f", "precipitation_in"
		if st.Units == "metric" {
			unit, punit = "°C", "mm"
			hiKey, loKey, pKey = "temperature_max_c", "temperature_min_c", "precipitation_mm"
		}
		if hi, ok := present.AsFloat(day[hiKey]); ok {
			temps = append(temps, fmt.Sprintf("H %.0f%s", hi, unit))
		}
		if lo, ok := present.AsFloat(day[loKey]); ok {
			temps = append(temps, fmt.Sprintf("L %.0f%s", lo, unit))
		}
		if precip, ok := present.AsFloat(day[pKey]); ok {
			temps = append(temps, fmt.Sprintf("%.2f %s", precip, punit))
		}
		addLine(v.body, strings.TrimSpace(present.AsString(day["date"])+"  "+strings.Join(temps, "  ")), "muted")
	}
}

// TropicsView renders the tropical tracker.
type TropicsView struct {
	Root fyne.CanvasObject
	body *fyne.Container
}

// NewTropicsView builds the tropics page.
func NewTropicsView() *TropicsView {
	v := &TropicsView{body: container.NewVBox()}
	v.Root = newSection("wx.tropics", container.NewVBox(line("NOAA NATIONAL HURRICANE CENTER // TROPICAL TRACKER", "kicker"), v.body))
	return v
}

// Refresh paints tropics JSON.
func (v *TropicsView) Refresh(st *store.Store) {
	clearBox(v.body)
	if st.TropicsError != "" {
		addLine(v.body, st.TropicsError, "alert")
	}
	report := present.Map(present.Map(st.Tropics)["tropics"])
	if len(report) == 0 {
		addLine(v.body, "Scanning Atlantic and Pacific basins…", "muted")
		return
	}
	count := 0
	if n, ok := present.AsFloat(report["total_active"]); ok {
		count = int(n)
	}
	text := "ALL BASINS QUIET // NO ACTIVE CYCLONES"
	role := "ok"
	if count > 0 {
		text = fmt.Sprintf("%d ACTIVE TROPICAL CYCLONE", count)
		if count != 1 {
			text += "S"
		}
		role = "watch"
	}
	addLine(v.body, text, role)
	for _, item := range present.Slice(report["storms"]) {
		storm := present.Map(item)
		if storm == nil {
			continue
		}
		windKey, unit := "wind_speed_mph", "mph"
		if st.Units == "metric" {
			windKey, unit = "wind_speed_kmh", "km/h"
		}
		windText := ""
		if wind, ok := present.AsFloat(storm[windKey]); ok {
			windText = fmt.Sprintf(" %.0f %s", wind, unit)
		}
		name := present.AsString(storm["name"])
		if name == "" {
			name = "Storm"
		}
		addLine(v.body, strings.TrimSpace(fmt.Sprintf("%s  %s%s %s", name, present.AsString(storm["category_label"]), windText, present.AsString(storm["movement_compass"]))), "kicker")
		addLine(v.body, present.AsString(storm["headline"]), "muted")
	}
	for _, item := range present.Slice(report["disturbances"]) {
		dist := present.Map(item)
		if dist == nil {
			continue
		}
		name := present.AsString(dist["name"])
		if name == "" {
			name = "Disturbance"
		}
		addLine(v.body, fmt.Sprintf("%s  48h %s%%  7d %s%%", name, chance(dist["chance_48h"]), chance(dist["chance_7d"])), "muted")
	}
}

// GridView is the favorites grid.
type GridView struct {
	Root   fyne.CanvasObject
	Name   *widget.Entry
	Value  *widget.Entry
	AddBtn *widget.Button
	body   *fyne.Container
	store  *store.Store
}

// NewGridView builds the pinned-location grid.
func NewGridView(st *store.Store) *GridView {
	v := &GridView{store: st, body: container.NewVBox()}
	v.Name = widget.NewEntry()
	v.Name.SetPlaceHolder("Name")
	v.Value = widget.NewEntry()
	v.Value.SetPlaceHolder("City, ST or zip")
	v.AddBtn = widget.NewButton("Add", v.Add)
	form := container.NewBorder(nil, nil, v.Name, newSection("wx.grid.add", v.AddBtn), v.Value)
	inner := container.NewVBox(line("PINNED LOCATIONS // COMMAND GRID", "kicker"), form, v.body)
	v.Root = newSection("wx.grid", inner)
	return v
}

// Add pins the entry text and reloads the grid while the desk is open.
func (v *GridView) Add() {
	name := strings.TrimSpace(v.Name.Text)
	value := strings.TrimSpace(v.Value.Text)
	if value == "" {
		return
	}
	if name == "" {
		name = value
	}
	v.store.AddFavorite(name, value)
	v.Name.SetText("")
	v.Value.SetText("")
	if v.store.DeskOpen {
		v.store.RefreshGrid()
	}
}

// Refresh paints favorite cards from CLI JSON.
func (v *GridView) Refresh(st *store.Store) {
	cards := append([]store.GridCard(nil), st.GridCards...)
	if len(cards) == 0 {
		for _, fav := range st.Favorites {
			key := fav.Value
			if key == "" {
				key = fav.Name
			}
			name := fav.Name
			if name == "" {
				name = key
			}
			cards = append(cards, store.GridCard{LocationKey: key, DisplayName: name})
		}
	}
	v.body.Objects = nil
	if len(cards) == 0 {
		v.body.Add(line("No pinned locations yet.", "muted"))
		v.body.Refresh()
		return
	}
	objs := make([]fyne.CanvasObject, 0, len(cards))
	for _, cardData := range cards {
		objs = append(objs, v.card(st, cardData))
	}
	cols := 3
	if len(objs) < cols {
		cols = len(objs)
	}
	v.body.Add(container.NewGridWithColumns(cols, objs...))
	v.body.Refresh()
}

func (v *GridView) card(st *store.Store, cardData store.GridCard) fyne.CanvasObject {
	box := container.NewVBox()
	name := cardData.DisplayName
	if name == "" {
		name = cardData.LocationKey
	}
	addLine(box, name, "kicker")
	switch {
	case cardData.Loading:
		addLine(box, "Loading…", "muted")
	case cardData.Error != "":
		addLine(box, cardData.Error, "alert")
	case cardData.Payload != nil:
		addLine(box, present.PipGlyph(cardData.Payload)+present.DisplayTemp(cardData.Payload, st.Units), "temp")
		addLine(box, present.AsString(present.Conditions(cardData.Payload)["description"]), "muted")
	}
	key := cardData.LocationKey
	if key == "" {
		key = name
	}
	box.Add(widget.NewButton("Open", func() { v.store.SelectLocation(key) }))
	box.Add(widget.NewButton("Remove", func() { v.store.RemoveFavorite(key) }))
	return box
}

func orDash(text string) string {
	if strings.TrimSpace(text) == "" {
		return "—"
	}
	return text
}

func pct(v any) string {
	f, ok := present.AsFloat(v)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%.0f%%", f)
}

func whole(v any) string {
	if v == nil {
		return "0"
	}
	if f, ok := present.AsFloat(v); ok {
		return fmt.Sprintf("%.0f", f)
	}
	return present.AsString(v)
}

func chance(v any) string {
	if v == nil {
		return "—"
	}
	if f, ok := present.AsFloat(v); ok {
		return fmt.Sprintf("%.0f", f)
	}
	text := present.AsString(v)
	if text == "" {
		return "—"
	}
	return text
}

func normalPair(normals map[string]any, units string) (high float64, highOK bool, low float64, lowOK bool, unit string) {
	if units == "metric" {
		high, highOK = present.AsFloat(normals["normal_high_c"])
		low, lowOK = present.AsFloat(normals["normal_low_c"])
		return high, highOK, low, lowOK, "°C"
	}
	high, highOK = present.AsFloat(normals["normal_high_f"])
	low, lowOK = present.AsFloat(normals["normal_low_f"])
	return high, highOK, low, lowOK, "°F"
}

func addRecord(body *fyne.Container, label string, value any, unit string, record map[string]any) {
	f, ok := present.AsFloat(value)
	if !ok {
		return
	}
	years := yearList(record)
	text := fmt.Sprintf("%s %.0f%s", label, f, unit)
	if years != "" {
		text += " " + years
	}
	addLine(body, text, "")
}

func yearList(record map[string]any) string {
	if record == nil {
		return ""
	}
	var parts []string
	switch years := record["years"].(type) {
	case []any:
		for _, year := range years {
			parts = append(parts, whole(year))
		}
	case []float64:
		for _, year := range years {
			parts = append(parts, fmt.Sprintf("%.0f", year))
		}
	case []int:
		for _, year := range years {
			parts = append(parts, fmt.Sprintf("%d", year))
		}
	case []string:
		parts = append(parts, years...)
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
