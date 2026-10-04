"""Desk pages beyond the daily surface. Each page renders CLI JSON."""

from __future__ import annotations

import gi

gi.require_version("Gtk", "4.0")
from gi.repository import Gdk, GLib, Gtk

from wxdesk import present
from wxdesk.widgets import card, clear, label

RADAR_PRODUCTS = [
    ("composite-reflectivity", "Composite"),
    ("base-reflectivity", "Base"),
    ("storm-relative-velocity", "Velocity"),
    ("echo-tops", "Echo Tops"),
    ("precip-type", "Precip Type"),
    ("one-hour-precip", "1-Hr Precip"),
    ("storm-total-precip", "Storm Total"),
]

RADAR_RADII = [
    (150, "Local 150 km"),
    (200, "200 km"),
    (250, "Metro 250 km"),
    (500, "Regional 500 km"),
    (1000, "Synoptic 1000 km"),
    (2000, "CONUS 2000 km"),
]


class DailySurface(Gtk.Box):
    """Observation, nowcast, alerts, hourly strip, and the day list."""

    def __init__(self):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=12)
        self.set_margin_top(12)
        self.set_margin_bottom(16)
        self.set_margin_start(14)
        self.set_margin_end(14)
        self.set_name("wx.surface")
        obs, self.obs_body = card("Atmospheric Telemetry", "GRID.OBS")
        obs.set_name("wx.observation")
        now, self.now_body = card("Precipitation Nowcast & Rain Timeline", "RADAR.QPF")
        now.set_name("wx.nowcast")
        alerts, self.alert_body = card("Tactical Alerts", "NWS")
        alerts.set_name("wx.alerts")
        hours, hour_body = card("Hourly Strip", "NEXT 24H")
        hours.set_name("wx.hourly.strip")
        self.hour_row = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=8)
        hour_scroll = Gtk.ScrolledWindow()
        hour_scroll.set_policy(Gtk.PolicyType.AUTOMATIC, Gtk.PolicyType.NEVER)
        hour_scroll.set_child(self.hour_row)
        hour_scroll.set_min_content_height(72)
        hour_body.append(hour_scroll)
        days, self.day_body = card("Synoptic Forecast Log", "NOAA.NWS")
        days.set_name("wx.days")
        for widget in (obs, now, alerts, hours, days):
            self.append(widget)

    def refresh(self, store) -> None:
        _fill_observation(self.obs_body, store)
        _fill_nowcast(self.now_body, store)
        _fill_alerts(self.alert_body, store)
        _fill_hourly(self.hour_row, store)
        _fill_days(self.day_body, store)


class RadarPanel(Gtk.Box):
    """Shows the PNG the CLI already composited. Export calls wx radar --save / --save-gif."""

    def __init__(self, store):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=8)
        self.store = store
        self.set_name("wx.radar")
        self.set_margin_top(8)
        self.set_margin_bottom(8)
        self.set_margin_start(8)
        self.set_margin_end(8)
        controls = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=6)
        self.product = Gtk.DropDown.new_from_strings([label for _id, label in RADAR_PRODUCTS])
        self.product.set_name("wx.radar.product")
        current = next((i for i, (pid, _) in enumerate(RADAR_PRODUCTS) if pid == store.radar_product), 0)
        self.product.set_selected(current)
        self.product.connect("notify::selected", self._product_changed)
        self.radius = Gtk.DropDown.new_from_strings([label for _r, label in RADAR_RADII])
        self.radius.set_name("wx.radar.radius")
        ridx = min(range(len(RADAR_RADII)), key=lambda i: abs(RADAR_RADII[i][0] - store.radar_radius))
        self.radius.set_selected(ridx)
        self.radius.connect("notify::selected", self._radius_changed)
        self.loop_btn = Gtk.Button(label="Loop")
        self.loop_btn.connect("clicked", lambda *_: store.toggle_loop())
        prev_btn = Gtk.Button(label="◀")
        prev_btn.connect("clicked", lambda *_: store.step_frame(-1))
        next_btn = Gtk.Button(label="▶")
        next_btn.connect("clicked", lambda *_: store.step_frame(1))
        png_btn = Gtk.Button(label="Save PNG")
        png_btn.set_name("wx.radar.save-png")
        png_btn.connect("clicked", self._choose_png)
        gif_btn = Gtk.Button(label="Save GIF")
        gif_btn.set_name("wx.radar.save-gif")
        gif_btn.connect("clicked", self._choose_gif)
        for widget in (self.product, self.radius, prev_btn, self.loop_btn, next_btn, png_btn, gif_btn):
            controls.append(widget)
        self.append(controls)
        self.frame_label = label("RADAR", "wx-tag", name="wx.radar.frame")
        self.append(self.frame_label)
        self.picture = Gtk.Picture()
        self.picture.set_name("wx.radar.picture")
        self.picture.set_can_shrink(True)
        self.picture.set_content_fit(Gtk.ContentFit.CONTAIN)
        self.picture.set_vexpand(True)
        self.picture.set_hexpand(True)
        self.append(self.picture)
        self.note = label("", "wx-muted", wrap=True, name="wx.radar.note")
        self.append(self.note)
        self._guard = False

    def refresh(self, store) -> None:
        self._guard = True
        try:
            idx = next((i for i, (pid, _) in enumerate(RADAR_PRODUCTS) if pid == store.radar_product), 0)
            if self.product.get_selected() != idx:
                self.product.set_selected(idx)
            self.loop_btn.set_label("Pause" if store.loop_playing else "Loop")
            ridx = min(range(len(RADAR_RADII)), key=lambda i: abs(RADAR_RADII[i][0] - float(store.radar_radius)))
            if self.radius.get_selected() != ridx:
                self.radius.set_selected(ridx)
            frame = store.current_frame()
            if store.radar_error:
                self.note.set_label(store.radar_error)
            elif store.radar_loading:
                self.note.set_label("Fetching radar…")
            else:
                self.note.set_label("")
            if frame is None:
                self.frame_label.set_label("NO FRAME")
                return
            self.frame_label.set_label(str(frame.get("label") or "LIVE"))
            self._show_png(frame.get("png") or b"")
        finally:
            self._guard = False

    def _show_png(self, blob: bytes) -> None:
        if not blob:
            return
        # The CLI already composited this PNG. Reject anything else before GDK.
        if not blob.startswith(b"\x89PNG\r\n\x1a\n"):
            self.picture.set_paintable(None)
            self.note.set_label("Radar image from wx was not a PNG.")
            return
        try:
            texture = Gdk.Texture.new_from_bytes(GLib.Bytes.new(blob))
        except Exception as exc:
            self.picture.set_paintable(None)
            self.note.set_label(f"Radar image from wx was not a PNG ({exc}).")
            return
        self.picture.set_paintable(texture)

    def save_png(self, path: str) -> None:
        try:
            note = self.store.export_png(path)
            self.note.set_label(note or f"wx radar --save {path}")
        except Exception as exc:
            self.note.set_label(str(exc))

    def save_gif(self, path: str) -> None:
        try:
            note = self.store.export_gif(path)
            self.note.set_label(note or f"wx radar --save-gif {path}")
        except Exception as exc:
            self.note.set_label(str(exc))

    def _choose_png(self, *_args) -> None:
        dialog = Gtk.FileDialog()
        dialog.set_title("Save Radar PNG")
        dialog.set_initial_name("wx-radar.png")
        dialog.save(self.get_root(), None, self._png_done)

    def _choose_gif(self, *_args) -> None:
        dialog = Gtk.FileDialog()
        dialog.set_title("Export Radar GIF")
        dialog.set_initial_name("wx-radar.gif")
        dialog.save(self.get_root(), None, self._gif_done)

    def _png_done(self, dialog, result) -> None:
        try:
            chosen = dialog.save_finish(result)
        except GLib.Error:
            return
        if chosen is not None and chosen.get_path():
            self.save_png(chosen.get_path())

    def _gif_done(self, dialog, result) -> None:
        try:
            chosen = dialog.save_finish(result)
        except GLib.Error:
            return
        if chosen is not None and chosen.get_path():
            self.save_gif(chosen.get_path())

    def _product_changed(self, dropdown, _pspec) -> None:
        if self._guard:
            return
        idx = dropdown.get_selected()
        if idx is None or idx < 0 or idx >= len(RADAR_PRODUCTS):
            return
        product = RADAR_PRODUCTS[idx][0]
        if product == self.store.radar_product:
            return
        self.store.radar_product = product
        if self.store.desk_open:
            self.store.refresh_radar()

    def _radius_changed(self, dropdown, _pspec) -> None:
        if self._guard:
            return
        idx = dropdown.get_selected()
        if idx is None or idx < 0 or idx >= len(RADAR_RADII):
            return
        radius = float(RADAR_RADII[idx][0])
        if radius == self.store.radar_radius:
            return
        self.store.radar_radius = radius
        if self.store.desk_open:
            self.store.refresh_radar()


class OutlookView(Gtk.Box):
    def __init__(self):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=8)
        self.set_name("wx.outlooks")
        self.set_margin_top(12)
        self.set_margin_start(14)
        self.set_margin_end(14)
        self.set_margin_bottom(16)
        _title(self, "NOAA CLIMATE PREDICTION CENTER (CPC)")
        self.body = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=6)
        self.append(self.body)

    def refresh(self, store) -> None:
        clear(self.body)
        if store.outlook_error:
            self.body.append(label(store.outlook_error, "wx-alert", wrap=True))
        payload = store.outlook or {}
        if not payload:
            self.body.append(label("Outlooks load while the desk is open.", "wx-muted"))
            return
        if payload.get("location"):
            self.body.append(label(str(payload["location"]).upper(), "wx-kicker"))
        shift = payload.get("pattern_shift") or {}
        if shift.get("summary"):
            css = "wx-watch" if shift.get("has_shift") else "wx-muted"
            self.body.append(label(str(shift["summary"]), css, wrap=True))
        drought = payload.get("drought") or {}
        if drought.get("status"):
            target = f" ({drought['target']})" if drought.get("target") else ""
            self.body.append(label(f"Drought: {drought['status']}{target}", "wx-muted"))
        for item in payload.get("outlooks") or []:
            if not isinstance(item, dict):
                continue
            horizon = item.get("horizon") or "Outlook"
            temp = f"{item.get('temp_category') or '—'} { _pct(item.get('temp_probability')) }"
            precip = f"{item.get('precip_category') or '—'} { _pct(item.get('precip_probability')) }"
            self.body.append(label(f"{horizon}   Temp {temp.strip()}   Precip {precip.strip()}", wrap=True))


class ChaseView(Gtk.Box):
    def __init__(self, store):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=8)
        self.store = store
        self.set_name("wx.chase")
        self.set_margin_top(12)
        self.set_margin_start(14)
        self.set_margin_end(14)
        self.set_margin_bottom(16)
        _title(self, "REMOTE STORM CHASING // NATIONAL SEVERE CLUSTERS")
        self.body = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=8)
        self.append(self.body)

    def refresh(self, store) -> None:
        clear(self.body)
        if store.chase_error:
            self.body.append(label(store.chase_error, "wx-alert", wrap=True))
        payload = store.chase or {}
        if not payload:
            self.body.append(label("Scanning CONUS alerts…", "wx-muted"))
            return
        clusters = payload.get("clusters") or []
        total = payload.get("total_clusters")
        alerts = payload.get("total_alerts")
        css = "wx-watch" if clusters else "wx-ok"
        self.body.append(label(f"{total if total is not None else len(clusters)} ACTIVE CLUSTERS · {alerts or 0} SEVERE CELLS", css))
        spc = payload.get("spc") or {}
        day1 = (spc.get("day1") or {}).get("category") or {}
        if day1.get("name"):
            self.body.append(label(f"SPC Day 1: {day1.get('name')}", "wx-kicker"))
        if not clusters:
            self.body.append(label("No active storm clusters.", "wx-ok"))
            return
        for cluster in clusters:
            if not isinstance(cluster, dict):
                continue
            block = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=2)
            block.add_css_class("wx-card")
            name = cluster.get("name") or "Cluster"
            block.append(label(str(name), "wx-kicker"))
            block.append(
                label(
                    f"Score {cluster.get('score', '—')} · {cluster.get('total_alerts', 0)} alerts · {cluster.get('primary_hazard') or ''}",
                    "wx-muted",
                    wrap=True,
                )
            )
            if cluster.get("nearest_radar"):
                block.append(label(f"Radar {cluster['nearest_radar']}", "wx-tag"))
            button = Gtk.Button(label="Open on radar")
            lat = cluster.get("center_lat")
            lon = cluster.get("center_lon")
            if lat is not None and lon is not None:
                button.connect("clicked", lambda _b, la=lat, lo=lon: store.chase_to_radar(float(la), float(lo)))
            block.append(button)
            self.body.append(block)


class ClimateView(Gtk.Box):
    def __init__(self):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=8)
        self.set_name("wx.climate")
        self.set_margin_top(12)
        self.set_margin_start(14)
        self.set_margin_end(14)
        self.set_margin_bottom(16)
        _title(self, "CLIMATOLOGICAL OBSERVATION ARCHIVE")
        self.body = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=6)
        self.append(self.body)

    def refresh(self, store) -> None:
        clear(self.body)
        if store.climate_error:
            self.body.append(label(store.climate_error, "wx-alert", wrap=True))
        if store.history_error:
            self.body.append(label(store.history_error, "wx-alert", wrap=True))
        report = (store.climate or {}).get("climate") or {}
        if report:
            place = report.get("station_name") or report.get("location") or (store.climate or {}).get("location") or ""
            self.body.append(label(str(place).upper(), "wx-kicker"))
            if report.get("normals_period"):
                self.body.append(label(f"Normals {report['normals_period']}", "wx-tag"))
            normals = report.get("today_normals") or {}
            high, low, unit = _pair(normals, "normal_high", store.units)
            if high is not None:
                self.body.append(label(f"Normal high {high:.0f}{unit}   low {low:.0f}{unit}" if low is not None else f"Normal high {high:.0f}{unit}"))
            records = report.get("records") or {}
            rec_high = (records.get("record_high") or {})
            rec_low = (records.get("record_low") or {})
            if store.units == "metric":
                if rec_high.get("value_c") is not None:
                    self.body.append(label(f"Record high {float(rec_high['value_c']):.0f}°C {_years(rec_high)}"))
                if rec_low.get("value_c") is not None:
                    self.body.append(label(f"Record low {float(rec_low['value_c']):.0f}°C {_years(rec_low)}"))
            else:
                if rec_high.get("value_f") is not None:
                    self.body.append(label(f"Record high {float(rec_high['value_f']):.0f}°F {_years(rec_high)}"))
                if rec_low.get("value_f") is not None:
                    self.body.append(label(f"Record low {float(rec_low['value_f']):.0f}°F {_years(rec_low)}"))
            departure = report.get("departure") or {}
            if departure.get("summary"):
                self.body.append(label(str(departure["summary"]), "wx-watch", wrap=True))
        history = store.history or {}
        days = history.get("days") or []
        summary = history.get("summary") or {}
        if summary.get("days_count"):
            self.body.append(label(f"History {summary['days_count']} days", "wx-kicker"))
        if not report and not days:
            self.body.append(label("Climate loads when this page is open.", "wx-muted"))
        for day in days[:14]:
            if not isinstance(day, dict):
                continue
            if store.units == "metric":
                hi, lo = day.get("temperature_max_c"), day.get("temperature_min_c")
                precip = day.get("precipitation_mm")
                unit, punit = "°C", "mm"
            else:
                hi, lo = day.get("temperature_max_f"), day.get("temperature_min_f")
                precip = day.get("precipitation_in")
                unit, punit = "°F", "in"
            temps = []
            if hi is not None:
                temps.append(f"H {float(hi):.0f}{unit}")
            if lo is not None:
                temps.append(f"L {float(lo):.0f}{unit}")
            if precip is not None:
                temps.append(f"{float(precip):.2f} {punit}")
            self.body.append(label(f"{day.get('date') or ''}  {'  '.join(temps)}".strip(), "wx-muted"))


class TropicsView(Gtk.Box):
    def __init__(self):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=8)
        self.set_name("wx.tropics")
        self.set_margin_top(12)
        self.set_margin_start(14)
        self.set_margin_end(14)
        self.set_margin_bottom(16)
        _title(self, "NOAA NATIONAL HURRICANE CENTER // TROPICAL TRACKER")
        self.body = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=6)
        self.append(self.body)

    def refresh(self, store) -> None:
        clear(self.body)
        if store.tropics_error:
            self.body.append(label(store.tropics_error, "wx-alert", wrap=True))
        report = (store.tropics or {}).get("tropics") or {}
        if not report:
            self.body.append(label("Scanning Atlantic and Pacific basins…", "wx-muted"))
            return
        count = report.get("total_active") or 0
        css = "wx-watch" if count else "wx-ok"
        text = f"{count} ACTIVE TROPICAL CYCLONE" + ("" if count == 1 else "S")
        if not count:
            text = "ALL BASINS QUIET // NO ACTIVE CYCLONES"
        self.body.append(label(text, css))
        for storm in report.get("storms") or []:
            if not isinstance(storm, dict):
                continue
            wind = storm.get("wind_speed_kmh") if store.units == "metric" else storm.get("wind_speed_mph")
            unit = "km/h" if store.units == "metric" else "mph"
            wind_text = f" {float(wind):.0f} {unit}" if wind is not None else ""
            move = storm.get("movement_compass") or ""
            self.body.append(label(f"{storm.get('name') or 'Storm'}  {storm.get('category_label') or ''}{wind_text} {move}".strip(), "wx-kicker", wrap=True))
            if storm.get("headline"):
                self.body.append(label(str(storm["headline"]), "wx-muted", wrap=True))
        for dist in report.get("disturbances") or []:
            if not isinstance(dist, dict):
                continue
            self.body.append(
                label(
                    f"{dist.get('name') or 'Disturbance'}  48h {dist.get('chance_48h', '—')}%  7d {dist.get('chance_7d', '—')}%",
                    "wx-muted",
                    wrap=True,
                )
            )


class GridView(Gtk.Box):
    def __init__(self, store):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=8)
        self.store = store
        self.set_name("wx.grid")
        self.set_margin_top(12)
        self.set_margin_start(14)
        self.set_margin_end(14)
        self.set_margin_bottom(16)
        _title(self, "PINNED LOCATIONS // COMMAND GRID")
        form = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=6)
        self.name_entry = Gtk.Entry()
        self.name_entry.set_placeholder_text("Name")
        self.value_entry = Gtk.Entry()
        self.value_entry.set_placeholder_text("City, ST or zip")
        self.value_entry.set_hexpand(True)
        add = Gtk.Button(label="Add")
        add.set_name("wx.grid.add")
        add.connect("clicked", self._add)
        form.append(self.name_entry)
        form.append(self.value_entry)
        form.append(add)
        self.append(form)
        self.flow = Gtk.FlowBox()
        self.flow.set_selection_mode(Gtk.SelectionMode.NONE)
        self.flow.set_max_children_per_line(3)
        self.flow.set_column_spacing(8)
        self.flow.set_row_spacing(8)
        self.append(self.flow)

    def refresh(self, store) -> None:
        while (child := self.flow.get_child_at_index(0)) is not None:
            self.flow.remove(child)
        cards = store.grid_cards or [
            {
                "location_key": fav.get("value") or fav.get("name"),
                "display_name": fav.get("name") or fav.get("value"),
                "payload": None,
                "loading": False,
                "error": None,
            }
            for fav in store.favorites
        ]
        if not cards:
            self.flow.append(label("No pinned locations yet.", "wx-muted"))
            return
        for card_data in cards:
            self.flow.append(self._card(store, card_data))

    def _card(self, store, card_data: dict) -> Gtk.Widget:
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=2)
        box.add_css_class("wx-card")
        name = str(card_data.get("display_name") or card_data.get("location_key") or "")
        box.append(label(name, "wx-kicker"))
        payload = card_data.get("payload")
        if card_data.get("loading"):
            box.append(label("Loading…", "wx-muted"))
        elif card_data.get("error"):
            box.append(label(str(card_data["error"]), "wx-alert", wrap=True))
        elif payload:
            box.append(label(present.pip_glyph(payload) + present.display_temp(payload, store.units), "wx-temp"))
            desc = present.conditions(payload).get("description") or ""
            if desc:
                box.append(label(str(desc), "wx-muted"))
        key = str(card_data.get("location_key") or name)
        open_btn = Gtk.Button(label="Open")
        open_btn.connect("clicked", lambda *_: store.select_location(key))
        remove = Gtk.Button(label="Remove")
        remove.connect("clicked", lambda *_: store.remove_favorite(key))
        box.append(open_btn)
        box.append(remove)
        return box

    def _add(self, *_args) -> None:
        name = self.name_entry.get_text().strip()
        value = self.value_entry.get_text().strip()
        if not value:
            return
        self.store.add_favorite(name or value, value)
        self.name_entry.set_text("")
        self.value_entry.set_text("")
        if self.store.desk_open:
            self.store.refresh_grid()


class DualPane(Gtk.Box):
    """Radar beside telemetry when the desk is at least 760 px wide."""

    def __init__(self, telemetry: Gtk.Widget, radar: Gtk.Widget):
        super().__init__(orientation=Gtk.Orientation.VERTICAL)
        self.set_name("wx.dual")
        self.set_vexpand(True)
        self.paned = Gtk.Paned.new(Gtk.Orientation.HORIZONTAL)
        self.paned.set_name("wx.dual.paned")
        scroll = Gtk.ScrolledWindow()
        scroll.set_policy(Gtk.PolicyType.NEVER, Gtk.PolicyType.AUTOMATIC)
        scroll.set_child(telemetry)
        scroll.set_size_request(380, -1)
        self.paned.set_start_child(scroll)
        self.paned.set_resize_start_child(False)
        self.paned.set_shrink_start_child(False)
        self.paned.set_end_child(radar)
        self.paned.set_resize_end_child(True)
        self.paned.set_vexpand(True)
        self.paned.set_position(420)
        self.append(self.paned)
        self._wide = True

    def apply_width(self, width: int) -> None:
        wide = width >= 760
        if wide == self._wide:
            return
        self._wide = wide
        self.paned.set_orientation(Gtk.Orientation.HORIZONTAL if wide else Gtk.Orientation.VERTICAL)


def _title(parent: Gtk.Box, text: str) -> None:
    parent.append(label(text, "wx-kicker"))


def _pct(value) -> str:
    if value is None:
        return ""
    return f"{float(value):.0f}%"


def _years(record: dict) -> str:
    years = record.get("years") or []
    if not years:
        return ""
    return "(" + ", ".join(str(y) for y in years) + ")"


def _pair(normals: dict, prefix: str, units: str):
    if units == "metric":
        return normals.get(prefix + "_c"), normals.get("normal_low_c"), "°C"
    return normals.get(prefix + "_f"), normals.get("normal_low_f"), "°F"


def _fill_observation(body, store) -> None:
    clear(body)
    if store.error and not present.conditions(store.payload):
        body.append(label(store.error, "wx-alert", wrap=True))
        return
    cond = present.conditions(store.payload)
    if not cond:
        body.append(label("Waiting for wx…", "wx-muted"))
        return
    body.append(label(str(cond.get("location") or store.location or "").upper(), "wx-kicker"))
    body.append(label(present.display_temp(store.payload, store.units), "wx-temp"))
    if cond.get("description"):
        body.append(label(str(cond["description"]), "wx-muted"))
    feels = cond.get("feels_like_c") if store.units == "metric" else cond.get("feels_like_f")
    if feels is not None:
        body.append(label(f"Feels like {float(feels):.0f}°", "wx-muted"))
    metrics = []
    if cond.get("humidity_pct") is not None:
        metrics.append(f"Humidity {float(cond['humidity_pct']):.0f}%")
    wind = present.tactical_wind(store.payload, store.units)
    if wind:
        metrics.append(f"Wind {cond.get('wind_direction') or ''} {wind}".strip())
    if store.units == "metric" and cond.get("pressure_hpa") is not None:
        metrics.append(f"Pressure {float(cond['pressure_hpa']):.0f} hPa")
    elif cond.get("pressure_inhg") is not None:
        metrics.append(f"Pressure {float(cond['pressure_inhg']):.2f} inHg")
    if store.units == "metric" and cond.get("visibility_m") is not None:
        metrics.append(f"Visibility {float(cond['visibility_m']) / 1000:.1f} km")
    elif cond.get("visibility_mi") is not None:
        metrics.append(f"Visibility {float(cond['visibility_mi']):.0f} mi")
    if metrics:
        body.append(label("   ·   ".join(metrics), "wx-muted", wrap=True))
    meta = "  ".join(str(p) for p in (cond.get("station"), cond.get("observed_at")) if p)
    if meta:
        body.append(label(meta, "wx-tag"))


def _fill_nowcast(body, store) -> None:
    clear(body)
    nc = store.nowcast()
    if not nc:
        body.append(label("Nowcast unavailable", "wx-muted"))
        return
    badge = present.nowcast_badge(nc)
    css = "wx-alert" if nc.get("is_active_precip") else "wx-watch" if nc.get("next_precip_time") else "wx-ok"
    body.append(label(badge, css))
    if nc.get("headline"):
        body.append(label(str(nc["headline"]), wrap=True))
    if nc.get("summary"):
        body.append(label(str(nc["summary"]), "wx-muted", wrap=True))
    for interval in (nc.get("intervals") or [])[:8]:
        if not isinstance(interval, dict):
            continue
        prob = interval.get("probability")
        extra = f"  {float(prob):.0f}%" if prob else ""
        body.append(label(f"{present.hour_label(interval)}  {interval.get('summary') or ''}{extra}", "wx-muted"))


def _fill_alerts(body, store) -> None:
    clear(body)
    items = [a for a in present.alerts(store.payload) if isinstance(a, dict)]
    if not items:
        body.append(label("No active alerts", "wx-ok"))
        return
    for alert in items:
        if present.is_warning(alert):
            css = "wx-alert"
        elif present.is_watch(alert) or present.is_advisory(alert):
            css = "wx-watch"
        else:
            css = "wx-muted"
        body.append(label(str(alert.get("event") or "Alert").upper(), css, wrap=True))
        if alert.get("headline"):
            body.append(label(str(alert["headline"]), "wx-muted", wrap=True))


def _fill_hourly(row, store) -> None:
    clear(row)
    hours = ((store.payload or {}).get("forecast") or {}).get("hourly") or []
    if not hours:
        row.append(label("Hourly forecast unavailable", "wx-muted"))
        return
    for period in hours:
        if not isinstance(period, dict):
            continue
        cell = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=1)
        cell.add_css_class("wx-hour")
        cell.append(label(present.hour_label(period), "wx-tag"))
        cell.append(label(present.period_temp(period, store.units), "wx-hour-temp"))
        pop = period.get("probability_of_precipitation")
        if pop:
            cell.append(label(f"{float(pop):.0f}%", "wx-muted"))
        row.append(cell)


def _fill_days(body, store) -> None:
    clear(body)
    periods = ((store.payload or {}).get("forecast") or {}).get("periods") or []
    if not periods:
        body.append(label("Forecast unavailable", "wx-muted"))
        return
    for period in periods:
        if not isinstance(period, dict):
            continue
        line = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=8)
        line.append(label(str(period.get("name") or ""), "wx-kicker"))
        line.append(label(present.period_temp(period, store.units), "wx-hour-temp"))
        short = period.get("short_description") or ""
        pop = period.get("probability_of_precipitation")
        detail = f"{short}  {float(pop):.0f}%".strip() if pop else short
        if detail:
            text = label(detail, "wx-muted", wrap=True)
            text.set_hexpand(True)
            line.append(text)
        body.append(line)
