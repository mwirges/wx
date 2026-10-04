"""Daily desk window. Renders CLI JSON. Closing it stops radar fetch and the loop."""

from __future__ import annotations

from pathlib import Path

import gi

gi.require_version("Gtk", "4.0")
gi.require_version("Adw", "1")
from gi.repository import Adw, Gdk, Gtk

from wxdesk import present
from wxdesk.widgets import card, clear, collect_text, find_named, label

CSS_PATH = Path(__file__).with_name("style.css")
_CSS_LOADED = False


def load_css() -> None:
    global _CSS_LOADED
    if _CSS_LOADED or not CSS_PATH.is_file():
        return
    display = Gdk.Display.get_default()
    if display is None:
        return
    provider = Gtk.CssProvider()
    provider.load_from_path(str(CSS_PATH))
    Gtk.StyleContext.add_provider_for_display(display, provider, Gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
    _CSS_LOADED = True


class DeskWindow(Adw.ApplicationWindow):
    def __init__(self, application, store, on_closed=None):
        super().__init__(application=application)
        load_css()
        self.store = store
        self._on_closed = on_closed
        self._building = False
        self._alive = True
        self.store.open_desk()
        self.set_title("wx")
        self.set_name("wx.desk")
        self.add_css_class("wx-desk")
        self.set_default_size(860, 760)
        self.connect("close-request", self._close_request)
        self.set_content(self._build())
        self.store.subscribe(self._on_store)
        self.refresh()

    def _close_request(self, *_args) -> bool:
        # Release the window. The next Open Desk builds a new one (#37).
        self._alive = False
        try:
            self.store._listeners.remove(self._on_store)
        except ValueError:
            pass
        self.store.close_desk()
        if self._on_closed:
            self._on_closed()
        return False

    def _on_store(self, topic: str) -> None:
        if not self._alive:
            return
        if topic in {"weather", "format", "desk-opened"}:
            self.refresh()

    def _build(self) -> Gtk.Widget:
        self._building = True
        root = Gtk.Box(orientation=Gtk.Orientation.VERTICAL)
        header = Adw.HeaderBar()
        header.set_title_widget(label("ATMOSPHERIC TELEMETRY CONSOLE", "wx-kicker"))
        self.location_entry = Gtk.Entry()
        self.location_entry.set_placeholder_text("City, ST  or  zip")
        self.location_entry.set_text(self.store.location)
        self.location_entry.set_width_chars(22)
        self.location_entry.connect("activate", self._submit_location)
        header.pack_start(self.location_entry)

        self.format_dropdown = Gtk.DropDown.new_from_strings([present.FORMAT_LABELS[k] for k in present.FORMATS])
        self.format_dropdown.set_name("wx.format")
        self.format_dropdown.set_selected(present.FORMATS.index(self.store.menu_bar_format))
        self.format_dropdown.connect("notify::selected", self._format_changed)
        header.pack_end(self.format_dropdown)

        units = Gtk.DropDown.new_from_strings(["Imperial", "Metric"])
        units.set_selected(0 if self.store.units != "metric" else 1)
        units.connect("notify::selected", self._units_changed)
        self.units_dropdown = units
        header.pack_end(units)

        refresh = Gtk.Button(label="Refresh")
        refresh.connect("clicked", lambda *_: self.store.refresh())
        header.pack_end(refresh)
        root.append(header)

        self.status_line = label("", "wx-muted", name="wx.status")
        self.status_line.set_margin_start(12)
        self.status_line.set_margin_top(6)
        root.append(self.status_line)

        scroll = Gtk.ScrolledWindow()
        scroll.set_policy(Gtk.PolicyType.NEVER, Gtk.PolicyType.AUTOMATIC)
        scroll.set_vexpand(True)
        self.page_host = Gtk.Box(orientation=Gtk.Orientation.VERTICAL)
        scroll.set_child(self.page_host)
        root.append(scroll)
        self._mount_pages()
        self._building = False
        return root

    def _mount_pages(self) -> None:
        """Daily surface. Later pages are added beside this one."""
        self.page_host.append(self._daily_page())

    def _daily_page(self) -> Gtk.Widget:
        page = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=12)
        page.set_margin_top(12)
        page.set_margin_bottom(16)
        page.set_margin_start(14)
        page.set_margin_end(14)
        page.set_name("wx.surface")

        obs, self.obs_body = card("Atmospheric Telemetry", "GRID.OBS")
        obs.set_name("wx.observation")
        page.append(obs)

        now, self.now_body = card("Precipitation Nowcast & Rain Timeline", "RADAR.QPF")
        now.set_name("wx.nowcast")
        page.append(now)

        alerts, self.alert_body = card("Tactical Alerts", "NWS")
        alerts.set_name("wx.alerts")
        page.append(alerts)

        hours, self.hour_body = card("Hourly Strip", "NEXT 24H")
        hours.set_name("wx.hourly.strip")
        self.hour_row = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=8)
        hour_scroll = Gtk.ScrolledWindow()
        hour_scroll.set_policy(Gtk.PolicyType.AUTOMATIC, Gtk.PolicyType.NEVER)
        hour_scroll.set_child(self.hour_row)
        hour_scroll.set_min_content_height(72)
        self.hour_body.append(hour_scroll)
        page.append(hours)

        days, self.day_body = card("Synoptic Forecast Log", "NOAA.NWS")
        days.set_name("wx.days")
        page.append(days)
        return page

    def refresh(self) -> None:
        if not hasattr(self, "obs_body"):
            return
        self._building = True
        try:
            self._fill_observation()
            self._fill_nowcast()
            self._fill_alerts()
            self._fill_hourly()
            self._fill_days()
            text = present.status_text(
                self.store.payload, self.store.units, self.store.menu_bar_format, self.store.error
            )
            place = present.conditions(self.store.payload).get("location") or self.store.location or "AUTO"
            self.status_line.set_label(f"{place}   {text}")
            idx = present.FORMATS.index(self.store.menu_bar_format)
            if self.format_dropdown.get_selected() != idx:
                self.format_dropdown.set_selected(idx)
        finally:
            self._building = False

    def _fill_observation(self) -> None:
        clear(self.obs_body)
        if self.store.error and not present.conditions(self.store.payload):
            self.obs_body.append(label(self.store.error, "wx-alert", wrap=True))
            return
        cond = present.conditions(self.store.payload)
        if not cond:
            self.obs_body.append(label("Waiting for wx…", "wx-muted"))
            return
        loc = str(cond.get("location") or self.store.location or "")
        self.obs_body.append(label(loc.upper(), "wx-kicker"))
        self.obs_body.append(label(present.display_temp(self.store.payload, self.store.units), "wx-temp"))
        desc = str(cond.get("description") or "")
        if desc:
            self.obs_body.append(label(desc, "wx-muted"))
        feels = cond.get("feels_like_c") if self.store.units == "metric" else cond.get("feels_like_f")
        if feels is not None:
            self.obs_body.append(label(f"Feels like {float(feels):.0f}°", "wx-muted"))
        metrics = []
        if cond.get("humidity_pct") is not None:
            metrics.append(f"Humidity {float(cond['humidity_pct']):.0f}%")
        wind = present.tactical_wind(self.store.payload, self.store.units)
        direction = cond.get("wind_direction") or ""
        if wind:
            metrics.append(f"Wind {direction} {wind}".strip())
        if self.store.units == "metric" and cond.get("pressure_hpa") is not None:
            metrics.append(f"Pressure {float(cond['pressure_hpa']):.0f} hPa")
        elif cond.get("pressure_inhg") is not None:
            metrics.append(f"Pressure {float(cond['pressure_inhg']):.2f} inHg")
        if self.store.units == "metric" and cond.get("visibility_m") is not None:
            metrics.append(f"Visibility {float(cond['visibility_m']) / 1000:.1f} km")
        elif cond.get("visibility_mi") is not None:
            metrics.append(f"Visibility {float(cond['visibility_mi']):.0f} mi")
        if metrics:
            self.obs_body.append(label("   ·   ".join(metrics), "wx-muted", wrap=True))
        station = cond.get("station")
        observed = cond.get("observed_at")
        meta = "  ".join(str(p) for p in (station, observed) if p)
        if meta:
            self.obs_body.append(label(meta, "wx-tag"))

    def _fill_nowcast(self) -> None:
        clear(self.now_body)
        nc = self.store.nowcast()
        if not nc:
            self.now_body.append(label("Nowcast unavailable", "wx-muted"))
            return
        badge = present.nowcast_badge(nc)
        css = "wx-alert" if nc.get("is_active_precip") else "wx-watch" if nc.get("next_precip_time") else "wx-ok"
        self.now_body.append(label(badge, css))
        if nc.get("headline"):
            self.now_body.append(label(str(nc["headline"]), wrap=True))
        if nc.get("summary"):
            self.now_body.append(label(str(nc["summary"]), "wx-muted", wrap=True))
        phase = nc.get("primary_phase")
        if phase and phase != "none":
            self.now_body.append(label(str(phase).upper(), "wx-tag"))
        for interval in (nc.get("intervals") or [])[:8]:
            if not isinstance(interval, dict):
                continue
            hour = present.hour_label(interval)
            summary = interval.get("summary") or interval.get("phase") or ""
            prob = interval.get("probability")
            extra = f"  {float(prob):.0f}%" if prob else ""
            self.now_body.append(label(f"{hour}  {summary}{extra}", "wx-muted"))

    def _fill_alerts(self) -> None:
        clear(self.alert_body)
        items = [a for a in present.alerts(self.store.payload) if isinstance(a, dict)]
        if not items:
            self.alert_body.append(label("No active alerts", "wx-ok"))
            return
        for alert in items:
            css = "wx-alert" if present.is_warning(alert) else "wx-watch" if present.is_watch(alert) or present.is_advisory(alert) else "wx-muted"
            self.alert_body.append(label(str(alert.get("event") or "Alert").upper(), css, wrap=True))
            if alert.get("headline"):
                self.alert_body.append(label(str(alert["headline"]), "wx-muted", wrap=True))
            if alert.get("severity"):
                self.alert_body.append(label(str(alert["severity"]), "wx-tag"))

    def _fill_hourly(self) -> None:
        clear(self.hour_row)
        hours = ((self.store.payload or {}).get("forecast") or {}).get("hourly") or []
        if not hours:
            self.hour_row.append(label("Hourly forecast unavailable", "wx-muted"))
            return
        for period in hours:
            if not isinstance(period, dict):
                continue
            cell = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=1)
            cell.add_css_class("wx-hour")
            cell.append(label(present.hour_label(period), "wx-tag"))
            cell.append(label(present.period_temp(period, self.store.units), "wx-hour-temp"))
            pop = period.get("probability_of_precipitation")
            if pop:
                cell.append(label(f"{float(pop):.0f}%", "wx-muted"))
            self.hour_row.append(cell)

    def _fill_days(self) -> None:
        clear(self.day_body)
        periods = ((self.store.payload or {}).get("forecast") or {}).get("periods") or []
        if not periods:
            self.day_body.append(label("Forecast unavailable", "wx-muted"))
            return
        for period in periods:
            if not isinstance(period, dict):
                continue
            row = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=8)
            row.append(label(str(period.get("name") or ""), "wx-kicker"))
            row.append(label(present.period_temp(period, self.store.units), "wx-hour-temp"))
            short = period.get("short_description") or ""
            pop = period.get("probability_of_precipitation")
            detail = short
            if pop:
                detail = f"{short}  {float(pop):.0f}%".strip()
            if detail:
                text = label(detail, "wx-muted", wrap=True)
                text.set_hexpand(True)
                row.append(text)
            self.day_body.append(row)

    def _submit_location(self, entry) -> None:
        self.store.select_location(entry.get_text())

    def _format_changed(self, dropdown, _pspec) -> None:
        if self._building:
            return
        idx = dropdown.get_selected()
        if idx is None or idx < 0 or idx >= len(present.FORMATS):
            return
        self.store.set_menu_bar_format(present.FORMATS[idx])

    def _units_changed(self, dropdown, _pspec) -> None:
        if self._building:
            return
        units = "metric" if dropdown.get_selected() == 1 else "imperial"
        if units != self.store.units:
            self.store.set_units(units)


def section_text(window: DeskWindow, name: str) -> str:
    found = find_named(window, name)
    if found is None:
        return ""
    return collect_text(found)
