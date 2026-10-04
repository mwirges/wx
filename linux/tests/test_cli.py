import json
import os
import stat
import tempfile
import textwrap
import threading
import unittest
from pathlib import Path

from wxdesk.cli import Cancelled, WxCLI, climate_args, export_gif_args, export_png_args, radar_args, weather_args


class ArgTests(unittest.TestCase):
    def test_weather_args_match_mac_cli(self):
        self.assertEqual(
            weather_args("Fort Wayne, IN", "imperial", hourly=True),
            ["--json", "--forecast", "--alerts", "--hourly", "--hours", "24", "--location", "Fort Wayne, IN", "--units", "imperial"],
        )
        self.assertEqual(
            weather_args("Home", "metric", hourly=False),
            ["--json", "--forecast", "--alerts", "--location", "Home", "--units", "metric"],
        )
        self.assertNotIn("--hourly", weather_args(None, None, hourly=False))

    def test_radar_and_export_args_use_cli_writers(self):
        self.assertEqual(
            radar_args("64101", product="composite-reflectivity", radius_km=200, frames=8),
            [
                "radar",
                "--json",
                "--raw",
                "--loop",
                "--frames",
                "8",
                "--location",
                "64101",
                "--product",
                "composite-reflectivity",
                "--radius",
                "200",
            ],
        )
        self.assertEqual(
            export_png_args("/tmp/wx.png", "64101", "base-reflectivity", 150),
            ["radar", "--save", "/tmp/wx.png", "--location", "64101", "--product", "base-reflectivity", "--radius", "150"],
        )
        gif = export_gif_args("/tmp/wx.gif", "64101", "echo-tops", 200, frames=8)
        self.assertIn("--save-gif", gif)
        self.assertIn("/tmp/wx.gif", gif)
        self.assertNotIn("--json", gif)
        self.assertEqual(climate_args("Home", "imperial")[:2], ["climate", "--json"])


class SubprocessTests(unittest.TestCase):
    def test_decode_weather_json_from_stub(self):
        with tempfile.TemporaryDirectory() as tmp:
            binary = Path(tmp) / "wx"
            binary.write_text(
                textwrap.dedent(
                    """\
                    #!/usr/bin/env python3
                    import json, sys
                    print(json.dumps({"conditions": {"temperature_f": 72, "location": "Stub"}, "alerts": []}))
                    """
                ),
                encoding="utf-8",
            )
            binary.chmod(binary.stat().st_mode | stat.S_IEXEC)
            cli = WxCLI(str(binary))
            payload = cli.fetch_weather("Stub", "imperial", hourly=True)
            self.assertEqual(payload["conditions"]["temperature_f"], 72)

    def test_cancel_kills_radar_process(self):
        with tempfile.TemporaryDirectory() as tmp:
            binary = Path(tmp) / "wx"
            binary.write_text("#!/bin/sh\nexec sleep 30\n", encoding="utf-8")
            binary.chmod(binary.stat().st_mode | stat.S_IEXEC)
            cli = WxCLI(str(binary))
            box = {}

            def run():
                try:
                    cli.fetch_radar("X", "composite-reflectivity", 200)
                    box["err"] = None
                except Exception as exc:
                    box["err"] = exc

            thread = threading.Thread(target=run)
            thread.start()
            threading.Event().wait(0.3)
            cli.cancel_radar()
            thread.join(3)
            self.assertFalse(thread.is_alive())
            self.assertIsInstance(box.get("err"), Cancelled)

    def test_missing_binary(self):
        cli = WxCLI("/tmp/wx-linux-missing-binary")
        with self.assertRaises(Exception):
            cli.fetch_weather(None, None)


class ConfigUntouchedTests(unittest.TestCase):
    def test_suite_does_not_touch_user_config(self):
        home = Path.home() / ".config" / "wx" / "config.json"
        if home.exists():
            before = home.read_bytes()
        else:
            before = None
        # Importing prefs must not create the file.
        os.environ["WX_CONFIG"] = str(Path(tempfile.gettempdir()) / "wx-linux-unused.json")
        from wxdesk import prefs

        prefs.load()
        if before is None:
            self.assertFalse(home.exists())
        else:
            self.assertEqual(home.read_bytes(), before)
