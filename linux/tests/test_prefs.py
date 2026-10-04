import json
import os
import tempfile
import unittest
from pathlib import Path

from wxdesk.prefs import Prefs


class PrefsTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.path = str(Path(self.tmp.name) / "config.json")
        os.environ["WX_CONFIG"] = self.path
        self.prefs = Prefs(self.path)
        self.home = Path.home() / ".config" / "wx" / "config.json"
        self.home_bytes = self.home.read_bytes() if self.home.exists() else None

    def tearDown(self):
        if self.home_bytes is None:
            self.assertFalse(self.home.exists())
        else:
            self.assertEqual(self.home.read_bytes(), self.home_bytes)
        self.tmp.cleanup()

    def test_round_trip_preserves_unknown_keys(self):
        Path(self.path).write_text(json.dumps({"provider": "nws", "per_location": {"Home": {"radar_station": "KIWX"}}}) + "\n", encoding="utf-8")
        self.prefs.set_menu_bar_format("tactical")
        self.prefs.add_favorite("Home", "Fort Wayne, IN")
        self.prefs.add_recent("46808")
        data = json.loads(Path(self.path).read_text(encoding="utf-8"))
        self.assertEqual(data["provider"], "nws")
        self.assertEqual(data["per_location"]["Home"]["radar_station"], "KIWX")
        self.assertEqual(data["menu_bar_format"], "tactical")
        self.assertEqual(data["favorites"], [{"name": "Home", "value": "Fort Wayne, IN"}])
        self.assertEqual(data["recent_locations"], ["46808"])

    def test_recent_cap_and_favorite_replace(self):
        for i in range(12):
            self.prefs.add_recent(f"City {i}")
        data = self.prefs.load()
        self.assertEqual(len(data["recent_locations"]), 10)
        self.assertEqual(data["recent_locations"][0], "City 11")
        self.prefs.add_favorite("Home", "A")
        self.prefs.add_favorite("Home", "B")
        self.assertEqual(self.prefs.load()["favorites"], [{"name": "Home", "value": "B"}])
        self.prefs.remove_favorite("B")
        self.assertEqual(self.prefs.load()["favorites"], [])
