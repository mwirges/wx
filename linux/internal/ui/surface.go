package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mwirges/wx/linux/internal/present"
	"github.com/mwirges/wx/linux/internal/store"
)

// DailySurface is the observation, nowcast, alerts, hourly strip, and day list.
type DailySurface struct {
	Root        *fyne.Container
	obs         *fyne.Container
	now         *fyne.Container
	alerts      *fyne.Container
	alertPulse  *pulseDot
	hours       *fyne.Container
	hoursScroll *container.Scroll
	days        *fyne.Container
}

// NewDailySurface builds an empty daily surface.
func NewDailySurface() *DailySurface {
	s := &DailySurface{
		obs:        container.NewVBox(),
		now:        container.NewVBox(),
		alerts:     container.NewVBox(),
		alertPulse: newPulseDot(colRed),
		hours:      container.NewHBox(),
		days:       container.NewVBox(),
	}
	s.hoursScroll = container.NewHScroll(s.hours)
	s.Root = container.NewVBox(
		sectionCard("wx.observation", "Atmospheric Telemetry", "GRID.OBS", s.obs),
		sectionCard("wx.nowcast", "Precipitation Nowcast & Rain Timeline", "RADAR.QPF", s.now),
		newSection("wx.alerts", consoleCardMark("Tactical Alerts // NWS Watches & Warnings", "NWS.WARN", "alertkicker", colRed, s.alertPulse, s.alerts)),
		newSection("wx.hourly.strip", s.hoursScroll),
		sectionCard("wx.days", "Synoptic Forecast Log", "NOAA.NWS", s.days),
	)
	return s
}

// Refresh paints the latest CLI JSON.
func (s *DailySurface) Refresh(st *store.Store) {
	fillObservation(s.obs, st)
	fillNowcast(s.now, st)
	fillAlerts(s.alerts, s.alertPulse, st)
	fillHourly(s.hours, st)
	// The bar sits inside the scroll. Leave room or it strikes through the last line.
	s.hoursScroll.SetMinSize(fyne.NewSize(48, s.hours.MinSize().Height+14))
	fillDays(s.days, st)
}

func fillObservation(body *fyne.Container, st *store.Store) {
	clearBox(body)
	if st.Error != "" && len(present.Conditions(st.Payload)) == 0 {
		addLine(body, st.Error, "alert")
		return
	}
	cond := present.Conditions(st.Payload)
	if len(cond) == 0 {
		addLine(body, "Waiting for wx…", "muted")
		return
	}
	place := present.AsString(cond["location"])
	if place == "" {
		place = st.Location
	}
	body.Add(container.NewHBox(skyMark(present.AsString(cond["condition_code"])), line(present.DisplayTemp(st.Payload, st.Units), "temp")))
	addLine(body, strings.ToUpper(place), "place")
	addLine(body, present.AsString(cond["description"]), "muted")
	station := present.AsString(cond["station"])
	observed := present.LocalStamp(present.AsString(cond["observed_at"]))
	if station != "" || observed != "" {
		addLine(body, strings.TrimSpace(fmt.Sprintf("STATION // %s   ·   OBSERVED // %s", station, observed)), "tag")
	}
	var tiles []fyne.CanvasObject
	if humidity, ok := present.AsFloat(cond["humidity_pct"]); ok {
		tiles = append(tiles, metricTile("Humidity", fmt.Sprintf("%.0f%%", humidity)))
	}
	if _, value := windReadout(st); value != "" {
		tiles = append(tiles, metricTile("Wind", value))
	}
	feelsKey := "feels_like_f"
	if st.Units == "metric" {
		feelsKey = "feels_like_c"
	}
	if feels, ok := present.AsFloat(cond[feelsKey]); ok {
		tiles = append(tiles, metricTile("Feels", fmt.Sprintf("%.0f°", feels)))
	}
	if st.Units == "metric" {
		if pressure, ok := present.AsFloat(cond["pressure_hpa"]); ok {
			tiles = append(tiles, metricTile("Pressure", fmt.Sprintf("%.0f hPa", pressure)))
		}
		if vis, ok := present.AsFloat(cond["visibility_m"]); ok {
			tiles = append(tiles, metricTile("Vis", fmt.Sprintf("%.1f km", vis/1000)))
		}
	} else {
		if pressure, ok := present.AsFloat(cond["pressure_inhg"]); ok {
			tiles = append(tiles, metricTile("Pressure", fmt.Sprintf("%.2f inHg", pressure)))
		}
		if vis, ok := present.AsFloat(cond["visibility_mi"]); ok {
			tiles = append(tiles, metricTile("Vis", fmt.Sprintf("%.1f mi", vis)))
		}
	}
	if len(tiles) > 0 {
		body.Add(container.NewGridWithColumns(3, tiles...))
	}
}

// windReadout is the Mac tile, "12 mph NW". The sentence form stays in WindLine.
func windReadout(st *store.Store) (string, string) {
	cond := present.Conditions(st.Payload)
	var speed string
	if st.Units == "metric" {
		if value, ok := present.AsFloat(cond["wind_kph"]); ok {
			speed = fmt.Sprintf("%.0f km/h", value)
		}
	} else if value, ok := present.AsFloat(cond["wind_mph"]); ok {
		speed = fmt.Sprintf("%.0f mph", value)
	}
	if speed == "" {
		return "", ""
	}
	return "Wind", strings.TrimSpace(speed + " " + present.AsString(cond["wind_direction"]))
}

func fillNowcast(body *fyne.Container, st *store.Store) {
	clearBox(body)
	nc := st.Nowcast()
	if len(nc) == 0 {
		addLine(body, "Nowcast unavailable", "muted")
		return
	}
	role := "ok"
	if present.AsBool(nc["is_active_precip"]) {
		role = "alert"
	} else if truthy(nc["next_precip_time"]) {
		role = "watch"
	}
	badge := statusBadge(present.NowcastBadge(nc), role)
	if phase := strings.TrimSpace(present.AsString(nc["primary_phase"])); phase != "" && !strings.EqualFold(phase, "none") {
		badge = container.NewHBox(badge, tight(strings.ToUpper(phase), "kicker"))
	}
	body.Add(badge)
	addLine(body, present.AsString(nc["headline"]), "")
	addLine(body, present.AsString(nc["summary"]), "muted")

	var intervals []map[string]any
	var rates []float64
	var phases []string
	for _, item := range present.Slice(nc["intervals"]) {
		if len(intervals) >= 24 {
			break
		}
		interval := present.Map(item)
		if interval == nil {
			continue
		}
		intervals = append(intervals, interval)
		rates = append(rates, intervalRate(interval))
		phases = append(phases, present.AsString(interval["phase"]))
	}
	tiles := []fyne.CanvasObject{
		metricTile("Liquid Total", liquidReadout(nc, st.Units)),
		metricTile("Peak Rate", peakReadout(nc, st.Units)),
	}
	if snow := snowReadout(nc, st.Units); snow != "" {
		tiles = append(tiles, metricTile("Snow Total", snow))
	}
	if len(intervals) > 0 {
		hours := float64(len(intervals)) * 15 / 60
		tiles = append(tiles, metricTile("Window", fmt.Sprintf("%.0fh", hours)))
	}
	body.Add(container.NewGridWithColumns(3, tiles...))
	if len(intervals) == 0 {
		return
	}
	addLine(body, "15-MINUTE INTENSITY TIMELINE", "tag")
	body.Add(newRateBar(rates, phases))
	if labels := timelineLabels(intervals); labels != nil {
		body.Add(labels)
	}
}

func liquidReadout(nc map[string]any, units string) string {
	if units == "metric" {
		v, _ := present.AsFloat(nc["total_liquid_mm"])
		return fmt.Sprintf("%.1f mm", v)
	}
	v, _ := present.AsFloat(nc["total_liquid_in"])
	return fmt.Sprintf("%.2f in", v)
}

func peakReadout(nc map[string]any, units string) string {
	if units == "metric" {
		v, _ := present.AsFloat(nc["peak_rate_mmh"])
		return fmt.Sprintf("%.1f mm/h", v)
	}
	v, _ := present.AsFloat(nc["peak_rate_inh"])
	return fmt.Sprintf("%.2f in/h", v)
}

func snowReadout(nc map[string]any, units string) string {
	cm, ok := present.AsFloat(nc["total_snow_cm"])
	if !ok || cm <= 0.05 {
		return ""
	}
	if units == "metric" {
		return fmt.Sprintf("%.1f cm", cm)
	}
	if v, ok := present.AsFloat(nc["total_snow_in"]); ok {
		return fmt.Sprintf("%.1f in", v)
	}
	return fmt.Sprintf("%.1f in", cm/2.54)
}

func intervalRate(interval map[string]any) float64 {
	if v, ok := present.AsFloat(interval["rate_inh"]); ok {
		return v
	}
	if v, ok := present.AsFloat(interval["rate_mmh"]); ok {
		return v / 25.4
	}
	return 0
}

func timelineLabels(intervals []map[string]any) fyne.CanvasObject {
	if len(intervals) == 0 {
		return nil
	}
	// These sit in the border's edge slots. A truncating line's minimum is
	// one character, so the clocks were painting as "…".
	first := clockLine(present.ClockLabel(intervals[0]))
	if len(intervals) == 1 {
		return first
	}
	last := clockLine(present.ClockLabel(intervals[len(intervals)-1]))
	if len(intervals) < 8 {
		return container.NewBorder(nil, nil, first, last, layoutSpacer())
	}
	mid := clockLine(present.ClockLabel(intervals[len(intervals)/2]))
	return container.NewBorder(nil, nil, first, last, container.NewCenter(mid))
}

// rateBar is the Mac 15-minute intensity chart. Dry slots stay a short stub.
type rateBar struct {
	widget.BaseWidget
	rates  []float64
	phases []string
}

func newRateBar(rates []float64, phases []string) *rateBar {
	b := &rateBar{rates: rates, phases: phases}
	b.ExtendBaseWidget(b)
	return b
}

func (b *rateBar) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(withAlpha(colChassis, 0xA6))
	bg.CornerRadius = 4
	bg.StrokeColor = withAlpha(colCyan, 0x40)
	bg.StrokeWidth = 1
	rects := make([]*canvas.Rectangle, len(b.rates))
	objs := []fyne.CanvasObject{bg}
	for i := range rects {
		rects[i] = canvas.NewRectangle(rateColor(b.phases, b.rates, i))
		rects[i].CornerRadius = 1
		objs = append(objs, rects[i])
	}
	return &rateRenderer{bar: b, bg: bg, rects: rects, objs: objs}
}

type rateRenderer struct {
	bar   *rateBar
	bg    *canvas.Rectangle
	rects []*canvas.Rectangle
	objs  []fyne.CanvasObject
}

func (r *rateRenderer) Layout(size fyne.Size) {
	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(size)
	n := len(r.rects)
	if n == 0 {
		return
	}
	pad := float32(6)
	innerW := size.Width - pad*2
	innerH := size.Height - pad*2
	if innerW < 1 || innerH < 1 {
		return
	}
	gap := float32(2)
	w := (innerW - gap*float32(n-1)) / float32(n)
	if w < 2 {
		w = 2
	}
	maxRate := 0.20
	for _, v := range r.bar.rates {
		if v > maxRate {
			maxRate = v
		}
	}
	for i, rect := range r.rects {
		rate := 0.0
		if i < len(r.bar.rates) {
			rate = r.bar.rates[i]
		}
		ratio := rate / maxRate
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		h := float32(ratio) * innerH
		if h < 7 {
			h = 7
		}
		if h > innerH {
			h = innerH
		}
		rect.Move(fyne.NewPos(pad+float32(i)*(w+gap), pad+(innerH-h)))
		rect.Resize(fyne.NewSize(w, h))
	}
}

func (r *rateRenderer) MinSize() fyne.Size { return fyne.NewSize(48, 42) }

func (r *rateRenderer) Refresh() {
	for i, rect := range r.rects {
		rect.FillColor = rateColor(r.bar.phases, r.bar.rates, i)
		rect.Refresh()
	}
	r.bg.Refresh()
}

func (r *rateRenderer) Objects() []fyne.CanvasObject { return r.objs }

func (r *rateRenderer) Destroy() {}

func rateColor(phases []string, rates []float64, i int) color.NRGBA {
	rate := 0.0
	if i < len(rates) {
		rate = rates[i]
	}
	if rate <= 0.01 {
		return withAlpha(colSilver, 0xCC)
	}
	phase := ""
	if i < len(phases) {
		phase = strings.ToLower(phases[i])
	}
	switch {
	case strings.Contains(phase, "snow"):
		return colText
	case strings.Contains(phase, "ice"), strings.Contains(phase, "mixed"), strings.Contains(phase, "freezing"):
		return colAmber
	default:
		return colCyan
	}
}

func fillAlerts(body *fyne.Container, pulse *pulseDot, st *store.Store) {
	clearBox(body)
	items := present.AlertMaps(st.Payload)
	if len(items) == 0 {
		if pulse != nil {
			pulse.SetLive(false)
		}
		addLine(body, "No active alerts", "ok")
		return
	}
	banded := false
	for _, alert := range items {
		role := "muted"
		if present.IsWarning(alert) {
			role = "alert"
		} else if present.IsWatch(alert) || present.IsAdvisory(alert) {
			role = "watch"
		}
		event := present.AsString(alert["event"])
		if event == "" {
			event = "Alert"
		}
		eventLine := line(strings.ToUpper(event), role)
		eventLine.Wrapping = fyne.TextWrapOff
		eventLine.Truncation = fyne.TextTruncateEllipsis
		text := container.NewVBox(eventLine)
		if headline := brief(present.AsString(alert["headline"]), 88); headline != "" {
			text.Add(line(headline, "muted"))
		}
		var row fyne.CanvasObject = text
		if band := conditionBandOf(present.AsString(alert["severity"])); band != bandNone {
			banded = true
			row = alertPlate(band, container.NewBorder(nil, nil,
				container.NewCenter(newConditionBadge(band)), nil, text))
		}
		body.Add(row)
	}
	if pulse != nil {
		pulse.SetLive(banded)
	}
}

func fillHourly(row *fyne.Container, st *store.Store) {
	clearBox(row)
	hours := present.Slice(present.Map(present.Map(st.Payload)["forecast"])["hourly"])
	if len(hours) == 0 {
		row.Add(line("Hourly forecast unavailable", "muted"))
		row.Refresh()
		return
	}
	for _, item := range hours {
		period := present.Map(item)
		if period == nil {
			continue
		}
		parts := []fyne.CanvasObject{
			tight(present.HourLabel(period), "tag"),
			skyGlyph(periodGlyph(period), 16),
			tight(present.PeriodTemp(period, st.Units), "hour"),
		}
		if pop := popText(period["probability_of_precipitation"]); pop != "" {
			parts = append(parts, tight(pop, "muted"))
		}
		row.Add(chip(48, parts...))
	}
	row.Refresh()
}

func fillDays(body *fyne.Container, st *store.Store) {
	clearBox(body)
	periods := present.Slice(present.Map(present.Map(st.Payload)["forecast"])["periods"])
	if len(periods) == 0 {
		addLine(body, "Forecast unavailable", "muted")
		return
	}
	for _, item := range periods {
		period := present.Map(item)
		if period == nil {
			continue
		}
		body.Add(periodRow(period, st.Units))
	}
	body.Refresh()
}

// alertPlate is the Mac compact alert row: a tinted plate behind the badge.
func alertPlate(band conditionBand, content fyne.CanvasObject) fyne.CanvasObject {
	ink := colRed
	if band == bandYellow {
		ink = colGold
	}
	bg := canvas.NewRectangle(withAlpha(ink, 0x1A))
	bg.CornerRadius = 6
	bg.StrokeColor = withAlpha(ink, 0x66)
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.NewPadded(content))
}

func periodRow(period map[string]any, units string) fyne.CanvasObject {
	name := line(present.AsString(period["name"]), "")
	name.Wrapping = fyne.TextWrapOff
	temp := line(present.PeriodTemp(period, units), "hour")
	temp.Wrapping = fyne.TextWrapOff
	short := line(present.AsString(period["short_description"]), "muted")
	short.Wrapping = fyne.TextWrapOff
	short.Truncation = fyne.TextTruncateEllipsis
	right := []fyne.CanvasObject{temp}
	if pop := popText(period["probability_of_precipitation"]); pop != "" {
		right = append([]fyne.CanvasObject{tight(pop, "hour")}, right...)
	}
	left := container.NewHBox(skyGlyph(periodGlyph(period), 16), name)
	row := container.NewBorder(nil, nil, left, container.NewHBox(right...), short)
	bg := canvas.NewRectangle(withAlpha(colChassis, 0xA6))
	bg.CornerRadius = 4
	return container.NewStack(bg, container.NewPadded(row))
}

func clockLine(text string) *widget.RichText {
	rt := line(text, "tag")
	rt.Wrapping = fyne.TextWrapOff
	return rt
}

func brief(text string, n int) string {
	text = strings.TrimSpace(text)
	if n < 8 || len(text) <= n {
		return text
	}
	cut := strings.LastIndex(text[:n], " ")
	if cut < n/2 {
		cut = n
	}
	return strings.TrimSpace(text[:cut]) + "…"
}

func popText(v any) string {
	f, ok := present.AsFloat(v)
	if !ok || f == 0 {
		return ""
	}
	return fmt.Sprintf("%.0f%%", f)
}

func truthy(v any) bool {
	if v == nil {
		return false
	}
	if text, ok := v.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	if f, ok := present.AsFloat(v); ok {
		return f != 0
	}
	return true
}
