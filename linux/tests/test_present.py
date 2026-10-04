import os
import unittest
from pathlib import Path

from wxdesk import present
from wxdesk.pixmap import has_pip_color, render_status
from wxdesk.tray import flatten_menu, layout_variant, number_menu

from fixture import PAYLOAD, QUIET, WATCH


class PresentTests(unittest.TestCase):
    def test_menu_bar_formats_match_mac(self):
        self.assertEqual(present.status_text(PAYLOAD, "imperial", "compact"), "🔴 72°")
        self.assertEqual(present.status_text(PAYLOAD, "imperial", "standard"), " 🔴 72°")
        self.assertEqual(present.status_text(PAYLOAD, "imperial", "tactical"), "🔴 [FWA] 72° ↘12mph")
        self.assertEqual(present.status_text(QUIET, "imperial", "compact"), "68°")
        self.assertEqual(present.status_text(QUIET, "imperial", "standard"), " 68°")
        self.assertEqual(present.status_text(WATCH, "imperial", "compact"), "🟠 80°")
        self.assertEqual(present.status_text(None, "imperial", "compact", error="missing"), " wx?")

    def test_warning_pip_beats_watch_and_does_not_blink(self):
        both = {
            "conditions": {"temperature_f": 70, "station": "KFWA"},
            "alerts": [
                {"event": "Wind Advisory", "severity": "Minor"},
                {"event": "Tornado Warning", "severity": "Extreme"},
            ],
        }
        first = present.status_text(both, "imperial", "compact")
        second = present.status_text(both, "imperial", "compact")
        self.assertEqual(first, second)
        self.assertTrue(first.startswith("🔴 "))
        self.assertNotIn("⭕", first)

    def test_station_tag_and_openmeteo_fallback(self):
        self.assertEqual(present.tactical_station_tag(PAYLOAD), "FWA")
        openmeteo = {"conditions": {"station": "OPENMETEO", "location": "Fort Wayne, IN", "temperature_f": 70}}
        self.assertEqual(present.tactical_station_tag(openmeteo), "FOR")
        self.assertEqual(present.tactical_status(openmeteo, "imperial"), "[FOR] 70°")

    def test_metric_temp_and_wind(self):
        self.assertEqual(present.display_temp(PAYLOAD, "metric"), "22°")
        self.assertEqual(present.tactical_wind(PAYLOAD, "metric"), "↘19km/h")

    def test_advisory_statement_is_amber(self):
        payload = {"conditions": {"temperature_f": 60}, "alerts": [{"event": "Special Weather Statement"}]}
        self.assertEqual(present.pip_kind(payload), "watch")
        self.assertEqual(present.pip_glyph(payload), "🟠 ")

    def test_pixmap_pip_is_steady_color(self):
        _w, _h, warn = render_status("72°", "warning")
        _w, _h, watch = render_status("[FWA] 72° ↘12mph", "watch")
        _w, _h, quiet = render_status("72°", "")
        self.assertTrue(has_pip_color(warn, "warning"))
        self.assertTrue(has_pip_color(watch, "watch"))
        self.assertFalse(has_pip_color(quiet, "warning"))
        self.assertGreater(len(warn), 16)

    def test_tray_menu_has_mac_entries(self):
        menu = present.tray_menu(
            location="Fort Wayne, IN",
            temp="72°",
            favorites=[{"name": "Home", "value": "Fort Wayne, IN"}],
            menu_format="tactical",
        )
        labels = present.menu_labels(menu)
        self.assertIn("ACTIVE // FORT WAYNE, IN (72°)", labels)
        self.assertIn("PINNED LOCATIONS // COMMAND GRID", labels)
        self.assertIn("Home", labels)
        self.assertIn("Menu Bar Format", labels)
        self.assertIn(present.FORMAT_LABELS["compact"], labels)
        self.assertIn(present.FORMAT_LABELS["standard"], labels)
        self.assertIn(present.FORMAT_LABELS["tactical"], labels)
        self.assertIn("Open Desk Window", labels)
        self.assertIn("Refresh Telemetry", labels)
        self.assertIn("Quit wx", labels)
        numbered = number_menu(menu)
        flat = flatten_menu(numbered)
        tactical = next(node for node in flat.values() if node["props"].get("label") == present.FORMAT_LABELS["tactical"])
        self.assertEqual(tactical["props"]["toggle-state"], 1)
        variant = layout_variant(numbered)
        self.assertEqual(variant.get_type_string(), "(ia{sv}av)")
        unpacked = variant.unpack()
        self.assertEqual(unpacked[0], 0)

    def test_shell_does_not_call_weather_providers(self):
        root = Path(__file__).resolve().parents[1] / "wxdesk"
        banned = ("weather.gov", "api.weather.gov", "open-meteo.com", "mesonet.agron")
        for path in root.rglob("*.py"):
            text = path.read_text(encoding="utf-8")
            for token in banned:
                self.assertNotIn(token, text, f"{path.name} contains {token}")

    def test_user_config_path_is_not_the_default_under_override(self):
        os.environ["WX_CONFIG"] = "/tmp/wx-linux-not-home.json"
        from wxdesk import prefs

        self.assertEqual(prefs.config_path(), "/tmp/wx-linux-not-home.json")
        self.assertFalse(prefs.config_path().endswith("/.config/wx/config.json") and "/tmp/" not in prefs.config_path())
