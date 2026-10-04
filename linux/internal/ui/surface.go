package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/mwirges/wx/linux/internal/present"
	"github.com/mwirges/wx/linux/internal/store"
)

// DailySurface is the observation, nowcast, alerts, hourly strip, and day list.
type DailySurface struct {
	Root   *fyne.Container
	obs    *fyne.Container
	now    *fyne.Container
	alerts *fyne.Container
	hours  *fyne.Container
	days   *fyne.Container
}

// NewDailySurface builds an empty daily surface.
func NewDailySurface() *DailySurface {
	s := &DailySurface{
		obs:    container.NewVBox(),
		now:    container.NewVBox(),
		alerts: container.NewVBox(),
		hours:  container.NewHBox(),
		days:   container.NewVBox(),
	}
	hourBody := container.NewVBox(container.NewHScroll(s.hours))
	s.Root = container.NewVBox(
		sectionCard("wx.observation", "Atmospheric Telemetry", "GRID.OBS", s.obs),
		sectionCard("wx.nowcast", "Precipitation Nowcast & Rain Timeline", "RADAR.QPF", s.now),
		sectionCard("wx.alerts", "Tactical Alerts", "NWS", s.alerts),
		sectionCard("wx.hourly.strip", "Hourly Strip", "NEXT 24H", hourBody),
		sectionCard("wx.days", "Synoptic Forecast Log", "NOAA.NWS", s.days),
	)
	return s
}

// Refresh paints the latest CLI JSON.
func (s *DailySurface) Refresh(st *store.Store) {
	fillObservation(s.obs, st)
	fillNowcast(s.now, st)
	fillAlerts(s.alerts, st)
	fillHourly(s.hours, st)
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
	addLine(body, strings.ToUpper(place), "kicker")
	addLine(body, present.DisplayTemp(st.Payload, st.Units), "temp")
	addLine(body, present.AsString(cond["description"]), "muted")
	feelsKey := "feels_like_f"
	if st.Units == "metric" {
		feelsKey = "feels_like_c"
	}
	if feels, ok := present.AsFloat(cond[feelsKey]); ok {
		addLine(body, fmt.Sprintf("Feels like %.0f°", feels), "muted")
	}
	var metrics []string
	if humidity, ok := present.AsFloat(cond["humidity_pct"]); ok {
		metrics = append(metrics, fmt.Sprintf("Humidity %.0f%%", humidity))
	}
	if wind := present.TacticalWind(st.Payload, st.Units); wind != "" {
		metrics = append(metrics, strings.TrimSpace("Wind "+present.AsString(cond["wind_direction"])+" "+wind))
	}
	if st.Units == "metric" {
		if pressure, ok := present.AsFloat(cond["pressure_hpa"]); ok {
			metrics = append(metrics, fmt.Sprintf("Pressure %.0f hPa", pressure))
		}
		if vis, ok := present.AsFloat(cond["visibility_m"]); ok {
			metrics = append(metrics, fmt.Sprintf("Visibility %.1f km", vis/1000))
		}
	} else {
		if pressure, ok := present.AsFloat(cond["pressure_inhg"]); ok {
			metrics = append(metrics, fmt.Sprintf("Pressure %.2f inHg", pressure))
		}
		if vis, ok := present.AsFloat(cond["visibility_mi"]); ok {
			metrics = append(metrics, fmt.Sprintf("Visibility %.0f mi", vis))
		}
	}
	if len(metrics) > 0 {
		addLine(body, strings.Join(metrics, "   ·   "), "muted")
	}
	var meta []string
	if station := present.AsString(cond["station"]); station != "" {
		meta = append(meta, station)
	}
	if observed := present.AsString(cond["observed_at"]); observed != "" {
		meta = append(meta, observed)
	}
	if len(meta) > 0 {
		addLine(body, strings.Join(meta, "  "), "tag")
	}
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
	addLine(body, present.NowcastBadge(nc), role)
	addLine(body, present.AsString(nc["headline"]), "")
	addLine(body, present.AsString(nc["summary"]), "muted")
	count := 0
	for _, item := range present.Slice(nc["intervals"]) {
		if count >= 8 {
			break
		}
		interval := present.Map(item)
		if interval == nil {
			continue
		}
		extra := ""
		if pop := popText(interval["probability"]); pop != "" {
			extra = "  " + pop
		}
		addLine(body, strings.TrimSpace(present.HourLabel(interval)+"  "+present.AsString(interval["summary"])+extra), "muted")
		count++
	}
}

func fillAlerts(body *fyne.Container, st *store.Store) {
	clearBox(body)
	var items []map[string]any
	for _, item := range present.Alerts(st.Payload) {
		if alert := present.Map(item); alert != nil {
			items = append(items, alert)
		}
	}
	if len(items) == 0 {
		addLine(body, "No active alerts", "ok")
		return
	}
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
		addLine(body, strings.ToUpper(event), role)
		addLine(body, present.AsString(alert["headline"]), "muted")
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
		cell := container.NewVBox(line(present.HourLabel(period), "tag"), line(present.PeriodTemp(period, st.Units), "hour"))
		if pop := popText(period["probability_of_precipitation"]); pop != "" {
			cell.Add(line(pop, "muted"))
		}
		row.Add(cell)
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
		name := line(present.AsString(period["name"]), "kicker")
		name.Wrapping = fyne.TextWrapOff
		temp := line(present.PeriodTemp(period, st.Units), "hour")
		temp.Wrapping = fyne.TextWrapOff
		row := container.NewVBox(container.NewHBox(name, temp))
		short := present.AsString(period["short_description"])
		detail := short
		if pop := popText(period["probability_of_precipitation"]); pop != "" {
			detail = strings.TrimSpace(short + "  " + pop)
		}
		if detail != "" {
			row.Add(line(detail, "muted"))
		}
		body.Add(row)
	}
	body.Refresh()
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
