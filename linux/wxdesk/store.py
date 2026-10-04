"""Presentation store. Fetches only through the wx CLI. Views stay dumb."""

from __future__ import annotations

import threading
import time

from wxdesk.present import nowcast_of

RADAR_TABS = ("dual", "radar")


def _seconds(name: str, fallback: float) -> float:
    import os

    raw = os.environ.get(name, "").strip()
    try:
        value = float(raw)
    except ValueError:
        return fallback
    return value if value > 0 else fallback


class WeatherStore:
    def __init__(self, backend, prefs=None, sync: bool = False):
        self.backend = backend
        self.prefs = prefs
        self.sync = sync
        self.payload = None
        self.nowcast_payload = None
        self.outlook = None
        self.chase = None
        self.climate = None
        self.history = None
        self.tropics = None
        self.radar = None
        self.radar_frames: list[dict] = []
        self.frame_index = 0
        self.loop_playing = False
        self.radar_loading = False
        self.radar_error = None
        self.loading = False
        self.error = None
        self.outlook_error = None
        self.chase_error = None
        self.climate_error = None
        self.history_error = None
        self.tropics_error = None
        self.location = ""
        self.units = "imperial"
        self.menu_bar_format = "standard"
        self.favorites: list[dict] = []
        self.recents: list[str] = []
        self.grid_cards: list[dict] = []
        self.tab = "dual"
        self.desk_open = False
        self.radar_product = "composite-reflectivity"
        self.radar_radius = 200.0
        self.history_days = 14
        self.last_refreshed = None
        self.wx_gen = 0
        self.radar_gen = 0
        self._listeners = []
        self.refresh_interval = _seconds("WX_REFRESH_SECONDS", 5 * 60)
        self.radar_interval = _seconds("WX_RADAR_REFRESH_SECONDS", 2 * 60)
        if prefs is not None:
            self.load_prefs()

    def subscribe(self, fn) -> None:
        self._listeners.append(fn)

    def _emit(self, topic: str) -> None:
        for fn in list(self._listeners):
            fn(topic)

    def load_prefs(self) -> None:
        cfg = self.prefs.load()
        self.location = str(cfg.get("default_location") or "")
        units = str(cfg.get("units") or "").strip()
        self.units = "metric" if units == "metric" else "imperial"
        fmt = str(cfg.get("menu_bar_format") or "standard").lower()
        self.menu_bar_format = fmt if fmt in ("compact", "standard", "tactical") else "standard"
        favs = cfg.get("favorites") or []
        self.favorites = [f for f in favs if isinstance(f, dict)]
        recents = cfg.get("recent_locations") or []
        self.recents = [str(r) for r in recents]

    def set_menu_bar_format(self, fmt: str) -> None:
        if fmt not in ("compact", "standard", "tactical"):
            return
        if fmt == self.menu_bar_format:
            return
        self.menu_bar_format = fmt
        if self.prefs is not None:
            self.prefs.set_menu_bar_format(fmt)
        self._emit("format")

    def set_units(self, units: str) -> None:
        self.units = "metric" if units == "metric" else "imperial"
        if self.prefs is not None:
            self.prefs.set_location_and_units(self.location.strip() or None, self.units)
        self.refresh()

    def select_location(self, location: str) -> None:
        clean = location.strip()
        if not clean:
            return
        self.location = clean
        self.wx_gen += 1
        if self.prefs is not None:
            self.prefs.add_recent(clean)
            self.prefs.set_location_and_units(clean, self.units)
            self.recents = self.prefs.load().get("recent_locations") or self.recents
        self.refresh()
        if self.desk_open:
            self.refresh_desk_extras()
            if self.tab in RADAR_TABS:
                self.refresh_radar()

    def add_favorite(self, name: str, value: str) -> None:
        if self.prefs is not None:
            self.prefs.add_favorite(name, value)
            self.load_prefs()
        else:
            self.favorites.append({"name": name.strip(), "value": value.strip()})
        self._emit("favorites")

    def remove_favorite(self, name_or_value: str) -> None:
        if self.prefs is not None:
            self.prefs.remove_favorite(name_or_value)
            self.load_prefs()
        else:
            key = name_or_value.strip().lower()
            self.favorites = [
                f
                for f in self.favorites
                if f.get("name", "").lower() != key and f.get("value", "").lower() != key
            ]
        self._emit("favorites")

    def open_desk(self) -> None:
        self.desk_open = True
        self._emit("desk-opened")

    def close_desk(self) -> None:
        """Match Mac windowWillClose (#35, #36, #37).

        The menu-bar refresh keeps running. Radar fetch, the loop, and the
        desk-only outlook / chase / grid work stop.
        """
        self.desk_open = False
        self.loop_playing = False
        self.radar_loading = False
        self.radar_gen += 1
        cancel = getattr(self.backend, "cancel_radar", None)
        if cancel:
            cancel()
        self._emit("desk-closed")

    def set_tab(self, tab: str) -> None:
        self.tab = tab
        self._emit("tab")
        if not self.desk_open:
            return
        if tab in RADAR_TABS and not self.radar_frames and not self.radar_loading:
            self.refresh_radar()
        if tab == "outlooks" and self.outlook is None:
            self.refresh_outlook()
        if tab == "chase" and self.chase is None:
            self.refresh_chase()
        if tab == "climate" and (self.climate is None or self.history is None):
            self.refresh_climate()
            self.refresh_history()
        if tab == "tropics" and self.tropics is None:
            self.refresh_tropics()
        if tab == "grid" and not self.grid_cards:
            self.refresh_grid()

    def on_menu_tick(self) -> None:
        self.refresh()
        self.refresh_desk_extras()

    def on_radar_tick(self) -> None:
        # Mac radar timer: fetch only while the desk window is open.
        if self.desk_open:
            self.refresh_radar()

    def refresh_desk_extras(self) -> None:
        if not self.desk_open:
            return
        self.refresh_outlook()
        self.refresh_chase()
        if self.favorites:
            self.refresh_grid()

    def refresh(self) -> None:
        self.loading = True
        gen = self.wx_gen
        loc = self.location.strip()
        units = self.units

        def work() -> None:
            try:
                payload = self.backend.fetch_weather(loc or None, units, hourly=True)
                err = None
            except Exception as exc:
                payload, err = None, str(exc)

            def apply() -> None:
                if gen != self.wx_gen:
                    return
                self.loading = False
                if err:
                    self.error = err
                else:
                    self.error = None
                    self.payload = payload
                    self.last_refreshed = time.time()
                self._emit("weather")

            self._deliver(apply)

        self._spawn(work)

    def refresh_radar(self) -> None:
        if not self.desk_open:
            return
        self.radar_loading = True
        self.radar_error = None
        self.radar_gen += 1
        gen = self.radar_gen
        loc = self.location.strip()
        product = self.radar_product
        radius = self.radar_radius

        def work() -> None:
            try:
                payload = self.backend.fetch_radar(loc or None, product, radius)
                err = None
            except Exception as exc:
                name = type(exc).__name__
                if name == "Cancelled" or str(exc) == "cancelled":
                    payload, err = None, None
                else:
                    payload, err = None, str(exc)

            def apply() -> None:
                if gen != self.radar_gen or not self.desk_open:
                    self.radar_loading = False
                    return
                self.radar_loading = False
                if err:
                    self.radar_error = err
                    self._emit("radar")
                    return
                if payload is None:
                    return
                self.radar = payload
                self.radar_frames = _frames(payload)
                if self.radar_frames:
                    self.frame_index = len(self.radar_frames) - 1
                    self.radar_error = None
                else:
                    self.radar_error = "Failed to decode radar frame telemetry"
                self._emit("radar")

            self._deliver(apply)

        self._spawn(work)

    def refresh_outlook(self) -> None:
        self._fetch_extra("outlook", lambda b, loc, units: b.fetch_outlook(loc))

    def refresh_chase(self) -> None:
        self._fetch_extra("chase", lambda b, loc, units: b.fetch_chase())

    def refresh_climate(self) -> None:
        self._fetch_extra("climate", lambda b, loc, units: b.fetch_climate(loc, units))

    def refresh_history(self) -> None:
        days = self.history_days
        self._fetch_extra("history", lambda b, loc, units: b.fetch_history(loc, units, days))

    def refresh_tropics(self, storm: str | None = None) -> None:
        self._fetch_extra("tropics", lambda b, loc, units: b.fetch_tropics(loc, units, storm))

    def refresh_grid(self) -> None:
        if not self.desk_open:
            return
        favs = list(self.favorites)
        if not favs:
            self.grid_cards = []
            self._emit("grid")
            return
        units = self.units
        cards = []
        for fav in favs:
            key = fav.get("value") or fav.get("name") or ""
            name = fav.get("name") or key
            cards.append({"location_key": key, "display_name": name, "payload": None, "loading": True, "error": None})
        self.grid_cards = cards
        self._emit("grid")

        def work() -> None:
            updated = []
            for card in cards:
                key = card["location_key"]
                try:
                    payload = self.backend.fetch_weather(key, units, hourly=False)
                    updated.append({**card, "payload": payload, "loading": False, "error": None})
                except Exception as exc:
                    updated.append({**card, "loading": False, "error": str(exc)})

            def apply() -> None:
                if not self.desk_open:
                    return
                self.grid_cards = updated
                self._emit("grid")

            self._deliver(apply)

        self._spawn(work)

    def toggle_loop(self) -> None:
        if not self.desk_open:
            self.loop_playing = False
            self._emit("loop")
            return
        self.loop_playing = not self.loop_playing
        self._emit("loop")

    def stop_loop(self) -> None:
        self.loop_playing = False
        self._emit("loop")

    def step_frame(self, delta: int) -> None:
        self.loop_playing = False
        if not self.radar_frames:
            return
        self.frame_index = (self.frame_index + delta) % len(self.radar_frames)
        self._emit("radar")

    def advance_frame(self) -> None:
        if not self.loop_playing or not self.desk_open or len(self.radar_frames) < 2:
            self.loop_playing = False
            return
        self.frame_index = (self.frame_index + 1) % len(self.radar_frames)
        self._emit("radar")

    def current_frame(self) -> dict | None:
        if not self.radar_frames:
            return None
        idx = max(0, min(self.frame_index, len(self.radar_frames) - 1))
        return self.radar_frames[idx]

    def export_png(self, path: str) -> str:
        loc = self.location.strip() or None
        return self.backend.export_png(path, loc, self.radar_product, self.radar_radius)

    def export_gif(self, path: str) -> str:
        loc = self.location.strip() or None
        frames = max(2, len(self.radar_frames) or 8)
        return self.backend.export_gif(path, loc, self.radar_product, self.radar_radius, frames)

    def chase_to_radar(self, lat: float, lon: float) -> None:
        self.radar_radius = 250
        self.tab = "radar"
        self._emit("tab")
        self.select_location(f"{lat:.4f},{lon:.4f}")

    def nowcast(self) -> dict | None:
        return nowcast_of(self.payload, self.nowcast_payload)

    def _fetch_extra(self, attr: str, call) -> None:
        if not self.desk_open:
            return
        loc = self.location.strip() or None
        units = self.units

        def work() -> None:
            try:
                payload = call(self.backend, loc, units)
                err = None
            except Exception as exc:
                payload, err = None, str(exc)

            def apply() -> None:
                if not self.desk_open:
                    return
                if err:
                    setattr(self, f"{attr}_error", err)
                else:
                    setattr(self, f"{attr}_error", None)
                    setattr(self, attr, payload)
                self._emit(attr)

            self._deliver(apply)

        self._spawn(work)

    def _spawn(self, fn) -> None:
        if self.sync:
            fn()
        else:
            threading.Thread(target=fn, daemon=True).start()

    def _deliver(self, fn) -> None:
        if self.sync:
            fn()
        else:
            from gi.repository import GLib

            GLib.idle_add(lambda: fn() or False)


def _frames(payload: dict) -> list[dict]:
    frames = []
    raw_frames = payload.get("frames") if isinstance(payload, dict) else None
    if isinstance(raw_frames, list) and raw_frames:
        for idx, frame in enumerate(raw_frames):
            if not isinstance(frame, dict):
                continue
            blob = _b64(frame.get("image_base64"))
            if not blob:
                continue
            frames.append(
                {
                    "valid_time": frame.get("valid_time") or payload.get("valid_time") or "",
                    "png": blob,
                    "label": "LIVE" if idx == len(raw_frames) - 1 else f"F{idx + 1}",
                    "live": idx == len(raw_frames) - 1,
                }
            )
        return frames
    blob = _b64(payload.get("image_base64") if isinstance(payload, dict) else None)
    if blob:
        frames.append(
            {
                "valid_time": payload.get("valid_time") or "",
                "png": blob,
                "label": "LIVE",
                "live": True,
            }
        )
    return frames


def _b64(text) -> bytes | None:
    if not text or not isinstance(text, str):
        return None
    import base64

    try:
        return base64.b64decode(text)
    except Exception:
        return None
