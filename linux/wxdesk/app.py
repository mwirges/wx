"""GTK application. The window and the tray are shells over the wx CLI."""

from __future__ import annotations

import gi

gi.require_version("Gtk", "4.0")
gi.require_version("Adw", "1")
from gi.repository import Adw, GLib, Gio

from wxdesk.cli import WxCLI
from wxdesk.desk import DeskWindow
from wxdesk.prefs import Prefs
from wxdesk.store import RADAR_TABS, WeatherStore
from wxdesk.tray import Tray


def _interval(store_value: float) -> int:
    return max(1, int(store_value))


class WxApp(Adw.Application):
    def __init__(self, backend=None, prefs=None, open_on_start: bool = True):
        super().__init__(application_id="com.github.mwirges.wx.desk", flags=Gio.ApplicationFlags.NON_UNIQUE)
        self.backend = backend or WxCLI()
        self.prefs = prefs if prefs is not None else Prefs()
        self.store = WeatherStore(self.backend, prefs=self.prefs, sync=False)
        self.tray = Tray(self.store, self._on_tray)
        self.window = None
        self.open_on_start = open_on_start
        self._booted = False
        self._menu_timer = 0
        self._radar_timer = 0
        self._loop_timer = 0

    def do_activate(self, *_args) -> None:
        if self._booted:
            self.show_desk()
            return
        self._booted = True
        self.hold()
        self.store.subscribe(self._on_store)
        self.tray.install()
        self._menu_timer = GLib.timeout_add_seconds(_interval(self.store.refresh_interval), self._menu_tick)
        self._radar_timer = GLib.timeout_add_seconds(_interval(self.store.radar_interval), self._radar_tick)
        self.store.refresh()
        if self.open_on_start:
            self.show_desk()
            self.store.refresh_desk_extras()

    def show_desk(self) -> DeskWindow:
        if self.window is not None:
            self.store.open_desk()
            self.window.present()
        else:
            self.window = DeskWindow(self, self.store, on_closed=self._forget_window)
            self.window.present()
        if (
            self.store.desk_open
            and self.store.tab in RADAR_TABS
            and not self.store.radar_frames
            and not self.store.radar_loading
        ):
            self.store.refresh_radar()
        return self.window

    def _forget_window(self) -> None:
        self.window = None
        self._stop_loop_timer()

    def _on_store(self, topic: str) -> None:
        if topic in {"loop", "desk-closed", "radar"}:
            self.sync_loop_timer()

    def _menu_tick(self) -> bool:
        self.store.on_menu_tick()
        return True

    def _radar_tick(self) -> bool:
        self.store.on_radar_tick()
        return True

    def _on_tray(self, action: str, target) -> None:
        if action == "quit":
            self.quit_app()
            return
        if action == "refresh":
            self.store.refresh()
            if self.store.desk_open:
                self.store.refresh_desk_extras()
                if self.store.tab in RADAR_TABS:
                    self.store.refresh_radar()
            return
        if action == "format" and target:
            self.store.set_menu_bar_format(str(target))
            return
        if action == "location" and target:
            self.store.select_location(str(target))
            self.show_desk()
            return
        if action == "grid":
            self.store.set_tab("grid")
            self.show_desk()
            return
        if action == "open-desk":
            self.show_desk()

    def sync_loop_timer(self) -> None:
        if self.store.loop_playing and self.store.desk_open and len(self.store.radar_frames) > 1:
            if not self._loop_timer:
                self._loop_timer = GLib.timeout_add(380, self._loop_tick)
        else:
            self._stop_loop_timer()

    def _loop_tick(self) -> bool:
        if not self.store.loop_playing or not self.store.desk_open:
            self._loop_timer = 0
            return False
        self.store.advance_frame()
        return True

    def _stop_loop_timer(self) -> None:
        if self._loop_timer:
            GLib.source_remove(self._loop_timer)
            self._loop_timer = 0

    def quit_app(self) -> None:
        self.store.close_desk()
        self.store.desk_open = False
        self._stop_loop_timer()
        if self._menu_timer:
            GLib.source_remove(self._menu_timer)
            self._menu_timer = 0
        if self._radar_timer:
            GLib.source_remove(self._radar_timer)
            self._radar_timer = 0
        self.tray.shutdown()
        self.release()
        self.quit()


def main() -> None:
    app = WxApp()
    raise SystemExit(app.run(None))
