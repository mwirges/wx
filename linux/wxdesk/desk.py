"""Daily desk window. Renders CLI JSON. Closing it stops radar fetch and the loop."""

from __future__ import annotations

from pathlib import Path

import gi

gi.require_version("Gtk", "4.0")
gi.require_version("Adw", "1")
from gi.repository import Adw, Gdk, Gtk

from wxdesk import present
from wxdesk.pages import (
    ChaseView,
    ClimateView,
    DailySurface,
    DualPane,
    GridView,
    OutlookView,
    RadarPanel,
    TropicsView,
)
from wxdesk.widgets import collect_text, find_named, label

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
        self.set_default_size(980, 760)
        self.connect("close-request", self._close_request)
        self.set_content(self._build())
        self.store.subscribe(self._on_store)
        self.refresh()

    def do_size_allocate(self, width, height, baseline):
        Adw.ApplicationWindow.do_size_allocate(self, width, height, baseline)
        dual = getattr(self, "dual", None)
        if dual is not None and width > 1:
            dual.apply_width(width)

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
        if topic == "tab":
            self._show_tab(self.store.tab)
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

        self.page_host = Gtk.Box(orientation=Gtk.Orientation.VERTICAL)
        self.page_host.set_vexpand(True)
        root.append(self.page_host)
        self._mount_pages()
        self._building = False
        return root

    def _mount_pages(self) -> None:
        modes = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=4)
        modes.set_margin_start(12)
        modes.set_margin_end(12)
        modes.set_margin_top(6)
        self.mode_buttons = {}
        for tab, title in (
            ("dual", "TACTICAL"),
            ("weather", "SURFACE"),
            ("radar", "RADAR"),
            ("outlooks", "OUTLOOKS"),
            ("chase", "CHASE"),
            ("climate", "CLIMATE"),
            ("tropics", "TROPICS"),
            ("grid", "GRID"),
        ):
            button = Gtk.Button(label=title)
            button.set_name(f"wx.mode.{tab}")
            button.connect("clicked", lambda _b, name=tab: self.store.set_tab(name))
            modes.append(button)
            self.mode_buttons[tab] = button
        self.page_host.append(modes)

        self.stack = Gtk.Stack()
        self.stack.set_vexpand(True)
        self.surfaces = [DailySurface(), DailySurface()]
        self.radar_panels = [RadarPanel(self.store), RadarPanel(self.store)]
        self.dual = DualPane(self.surfaces[0], self.radar_panels[0])
        weather_scroll = _scroller(self.surfaces[1])
        radar_scroll = _scroller(self.radar_panels[1])
        self.outlook_view = OutlookView()
        self.chase_view = ChaseView(self.store)
        self.climate_view = ClimateView()
        self.tropics_view = TropicsView()
        self.grid_view = GridView(self.store)
        self.stack.add_named(self.dual, "dual")
        self.stack.add_named(weather_scroll, "weather")
        self.stack.add_named(radar_scroll, "radar")
        self.stack.add_named(_scroller(self.outlook_view), "outlooks")
        self.stack.add_named(_scroller(self.chase_view), "chase")
        self.stack.add_named(_scroller(self.climate_view), "climate")
        self.stack.add_named(_scroller(self.tropics_view), "tropics")
        self.stack.add_named(_scroller(self.grid_view), "grid")
        self.page_host.append(self.stack)
        self._show_tab(self.store.tab)

    def _show_tab(self, tab: str) -> None:
        if tab not in self.mode_buttons:
            tab = "dual"
        self.stack.set_visible_child_name(tab)
        for name, button in self.mode_buttons.items():
            if name == tab:
                button.add_css_class("wx-mode-on")
            else:
                button.remove_css_class("wx-mode-on")

    def refresh(self) -> None:
        if not hasattr(self, "surfaces"):
            return
        self._building = True
        try:
            for surface in self.surfaces:
                surface.refresh(self.store)
            for panel in self.radar_panels:
                panel.refresh(self.store)
            self.outlook_view.refresh(self.store)
            self.chase_view.refresh(self.store)
            self.climate_view.refresh(self.store)
            self.tropics_view.refresh(self.store)
            self.grid_view.refresh(self.store)
            text = present.status_text(
                self.store.payload, self.store.units, self.store.menu_bar_format, self.store.error
            )
            place = present.conditions(self.store.payload).get("location") or self.store.location or "AUTO"
            self.status_line.set_label(f"{place}   {text}")
            idx = present.FORMATS.index(self.store.menu_bar_format)
            if self.format_dropdown.get_selected() != idx:
                self.format_dropdown.set_selected(idx)
            self._show_tab(self.store.tab)
            width = self.get_width()
            if width > 1:
                self.dual.apply_width(width)
        finally:
            self._building = False

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


def _scroller(child: Gtk.Widget) -> Gtk.ScrolledWindow:
    scroll = Gtk.ScrolledWindow()
    scroll.set_policy(Gtk.PolicyType.NEVER, Gtk.PolicyType.AUTOMATIC)
    scroll.set_vexpand(True)
    scroll.set_child(child)
    return scroll


def section_text(window: DeskWindow, name: str) -> str:
    found = find_named(window, name)
    if found is None:
        return ""
    return collect_text(found)
