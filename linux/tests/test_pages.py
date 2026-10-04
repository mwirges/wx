"""Pages beyond the daily desk: dual layout, radar PNG, and the other CLI views."""

import base64
import unittest
from pathlib import Path

import gi

gi.require_version("Gtk", "4.0")
gi.require_version("Adw", "1")
from gi.repository import Adw, Gtk

from wxdesk.desk import DeskWindow, section_text
from wxdesk.pages import DualPane, DailySurface, RadarPanel
from wxdesk.store import WeatherStore, _frames
from wxdesk.widgets import find_named

from fixture import PAYLOAD
from test_store import FakeBackend

# 1x1 PNG. The shell displays these bytes; it does not draw the radar.
PNG = base64.b64decode(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
)


def _button(root, text):
    if isinstance(root, Gtk.Button) and root.get_label() == text:
        return root
    child = root.get_first_child()
    while child is not None:
        found = _button(child, text)
        if found is not None:
            return found
        child = child.get_next_sibling()
    return None


class PageTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        Adw.init()
        cls.app = Adw.Application(application_id="com.github.mwirges.wx.desk.pages")
        cls.app.register()

    def test_dual_pane_puts_radar_beside_telemetry_until_760(self):
        store = WeatherStore(FakeBackend(), sync=True)
        pane = DualPane(DailySurface(), RadarPanel(store))
        pane.apply_width(900)
        self.assertEqual(pane.paned.get_orientation(), Gtk.Orientation.HORIZONTAL)
        pane.apply_width(760)
        self.assertEqual(pane.paned.get_orientation(), Gtk.Orientation.HORIZONTAL)
        pane.apply_width(759)
        self.assertEqual(pane.paned.get_orientation(), Gtk.Orientation.VERTICAL)
        self.assertIsNotNone(find_named(pane, "wx.observation"))
        self.assertIsNotNone(find_named(pane, "wx.radar"))

    def test_desk_opens_on_tactical_with_observation_and_radar(self):
        store = WeatherStore(FakeBackend(), sync=True)
        store.payload = PAYLOAD
        win = DeskWindow(self.app, store)
        try:
            self.assertEqual(store.tab, "dual")
            self.assertEqual(win.stack.get_visible_child_name(), "dual")
            self.assertIn("FORT WAYNE, IN", section_text(win, "wx.observation"))
            self.assertIn("NO FRAME", section_text(win, "wx.radar"))
            find_named(win, "wx.mode.weather").emit("clicked")
            self.assertEqual(win.stack.get_visible_child_name(), "weather")
            self.assertIn("72°", section_text(win, "wx.observation"))
        finally:
            win.destroy()

    def test_radar_displays_cli_png_and_export_calls_the_cli(self):
        backend = FakeBackend()
        store = WeatherStore(backend, sync=True)
        store.open_desk()
        store.location = "Fort Wayne, IN"
        encoded = base64.b64encode(PNG).decode()
        frames = _frames({"image_base64": encoded, "valid_time": "2026-10-04T15:00:00Z"})
        self.assertEqual(frames[0]["png"], PNG)
        store.radar_frames = frames
        panel = RadarPanel(store)
        panel.refresh(store)
        self.assertIsNotNone(panel.picture.get_paintable())
        self.assertEqual(panel.frame_label.get_label(), "LIVE")
        self.assertEqual(panel.note.get_label(), "")

        store.radar_frames = [{"png": b"hello", "label": "LIVE"}]
        panel.refresh(store)
        self.assertIn("not a PNG", panel.note.get_label())
        self.assertIsNone(panel.picture.get_paintable())

        panel.save_png("/tmp/wx-linux-radar.png")
        panel.save_gif("/tmp/wx-linux-radar.gif")
        kinds = [call[0] for call in backend.calls]
        self.assertEqual(kinds, ["export-png", "export-gif"])
        self.assertFalse(Path("/tmp/wx-linux-radar.png").exists())
        self.assertFalse(Path("/tmp/wx-linux-radar.gif").exists())

    def test_product_change_refetches_through_the_cli(self):
        backend = FakeBackend()
        store = WeatherStore(backend, sync=True)
        win = DeskWindow(self.app, store)
        try:
            panel = win.radar_panels[0]
            panel.product.set_selected(3)  # echo-tops
            self.assertEqual(store.radar_product, "echo-tops")
            self.assertTrue(any(call[0] == "radar" and call[2] == "echo-tops" for call in backend.calls))
        finally:
            win.destroy()

    def test_outlook_chase_climate_tropics_and_grid_render_cli_json(self):
        backend = FakeBackend()
        store = WeatherStore(backend, sync=True)
        store.payload = PAYLOAD
        store.outlook = {
            "location": "Fort Wayne, IN",
            "pattern_shift": {"has_shift": True, "summary": "Cooling into the 8-14 day"},
            "drought": {"status": "No Drought", "target": "Oct 2026"},
            "outlooks": [
                {
                    "horizon": "6-10 Day",
                    "temp_category": "Above",
                    "temp_probability": 55,
                    "precip_category": "Below",
                    "precip_probability": 40,
                }
            ],
        }
        store.chase = {
            "total_clusters": 1,
            "total_alerts": 4,
            "clusters": [
                {
                    "name": "Northern Indiana",
                    "score": 80,
                    "total_alerts": 4,
                    "primary_hazard": "Severe Thunderstorm Warning",
                    "nearest_radar": "KIWX",
                    "center_lat": 41.1,
                    "center_lon": -85.14,
                }
            ],
            "spc": {"day1": {"category": {"name": "Slight Risk"}}},
        }
        store.climate = {
            "climate": {
                "station_name": "Fort Wayne Intl",
                "normals_period": "1991–2020",
                "today_normals": {"normal_high_f": 68, "normal_low_f": 46, "normal_high_c": 20, "normal_low_c": 8},
                "records": {
                    "record_high": {"value_f": 92, "value_c": 33, "years": [1954]},
                    "record_low": {"value_f": 28, "value_c": -2, "years": [1981]},
                },
                "departure": {"summary": "+5.2°F Above Normal"},
            }
        }
        store.history = {
            "summary": {"days_count": 2},
            "days": [
                {
                    "date": "2026-10-03",
                    "temperature_max_f": 70,
                    "temperature_min_f": 48,
                    "precipitation_in": 0.1,
                    "temperature_max_c": 21,
                    "temperature_min_c": 9,
                    "precipitation_mm": 2.5,
                }
            ],
        }
        store.tropics = {
            "tropics": {
                "total_active": 1,
                "storms": [
                    {
                        "name": "Milton",
                        "category_label": "Category 3",
                        "wind_speed_mph": 120,
                        "wind_speed_kmh": 193,
                        "movement_compass": "NE",
                        "headline": "Moving northeast",
                    }
                ],
                "disturbances": [{"name": "AL91", "chance_48h": 20, "chance_7d": 40}],
            }
        }
        store.favorites = [{"name": "Home", "value": "Fort Wayne, IN"}]
        store.grid_cards = [
            {
                "location_key": "Fort Wayne, IN",
                "display_name": "Home",
                "payload": PAYLOAD,
                "loading": False,
                "error": None,
            }
        ]
        win = DeskWindow(self.app, store)
        try:
            outlook = section_text(win, "wx.outlooks")
            self.assertIn("Cooling into the 8-14 day", outlook)
            self.assertIn("6-10 Day", outlook)
            self.assertIn("Above 55%", outlook)
            self.assertIn("Drought: No Drought (Oct 2026)", outlook)

            chase = section_text(win, "wx.chase")
            self.assertIn("Northern Indiana", chase)
            self.assertIn("SPC Day 1: Slight Risk", chase)
            self.assertIn("KIWX", chase)
            _button(win.chase_view, "Open on radar").emit("clicked")
            self.assertEqual(store.tab, "radar")
            self.assertEqual(store.radar_radius, 250)
            self.assertEqual(store.location, "41.1000,-85.1400")
            self.assertEqual(win.stack.get_visible_child_name(), "radar")
            self.assertTrue(any(call[0] == "radar" and call[3] == 250 for call in backend.calls))

            climate = section_text(win, "wx.climate")
            self.assertIn("FORT WAYNE INTL", climate)
            self.assertIn("Normal high 68°F", climate)
            self.assertIn("Record high 92°F", climate)
            self.assertIn("+5.2°F Above Normal", climate)
            self.assertIn("2026-10-03", climate)
            self.assertIn("H 70°F", climate)

            tropics = section_text(win, "wx.tropics")
            self.assertIn("1 ACTIVE TROPICAL CYCLONE", tropics)
            self.assertIn("Milton", tropics)
            self.assertIn("120 mph", tropics)
            self.assertIn("AL91", tropics)
            self.assertIn("48h 20%", tropics)

            store.units = "metric"
            win.tropics_view.refresh(store)
            win.climate_view.refresh(store)
            tropics_c = section_text(win, "wx.tropics")
            self.assertIn("193 km/h", tropics_c)
            climate_c = section_text(win, "wx.climate")
            self.assertIn("Normal high 20°C", climate_c)
            self.assertIn("2.50 mm", climate_c)

            grid = section_text(win, "wx.grid")
            self.assertIn("Home", grid)
            self.assertIn("72°", grid)
            win.grid_view.name_entry.set_text("Cabin")
            win.grid_view.value_entry.set_text("46803")
            find_named(win, "wx.grid.add").emit("clicked")
            self.assertTrue(any(fav.get("value") == "46803" for fav in store.favorites))
            self.assertTrue(any(call[0] == "weather" and call[1] == "46803" and call[3] is False for call in backend.calls))
        finally:
            win.destroy()

    def test_shell_does_not_composite_radar(self):
        root = Path(__file__).resolve().parents[1] / "wxdesk"
        text = "\n".join(path.read_text(encoding="utf-8") for path in root.glob("*.py"))
        for banned in ("PIL", "ImageMagick", "imageio", "ffmpeg", "wand"):
            self.assertNotIn(banned, text)
        self.assertIn("radar --save", text)
        self.assertIn("radar --save-gif", text)
