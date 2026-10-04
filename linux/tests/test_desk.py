import os
import subprocess
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path

import gi

gi.require_version("Gtk", "4.0")
gi.require_version("Adw", "1")
from gi.repository import Adw

from wxdesk.desk import DeskWindow, section_text
from wxdesk.store import WeatherStore

from fixture import PAYLOAD
from test_store import FakeBackend


class DeskTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        Adw.init()
        cls.app = Adw.Application(application_id="com.github.mwirges.wx.desk.test")
        cls.app.register()

    def test_daily_surface_shows_cli_json(self):
        store = WeatherStore(FakeBackend(), sync=True)
        store.payload = PAYLOAD
        win = DeskWindow(self.app, store)
        try:
            self.assertTrue(store.desk_open)
            obs = section_text(win, "wx.observation")
            self.assertIn("FORT WAYNE, IN", obs)
            self.assertIn("72°", obs)
            self.assertIn("Partly Cloudy", obs)
            now = section_text(win, "wx.nowcast")
            self.assertIn("ALL CLEAR // DRY", now)
            self.assertIn("Dry next 6h", now)
            alerts = section_text(win, "wx.alerts")
            self.assertIn("SEVERE THUNDERSTORM WARNING", alerts)
            hours = section_text(win, "wx.hourly.strip")
            self.assertIn("72°", hours)
            days = section_text(win, "wx.days")
            self.assertIn("This Afternoon", days)
            self.assertIn("Partly Sunny", days)
            self.assertIn("74°", days)
        finally:
            win.destroy()

    def test_closing_window_stops_radar_and_releases_it(self):
        backend = FakeBackend()
        store = WeatherStore(backend, sync=True)
        store.payload = PAYLOAD
        closed = []
        win = DeskWindow(self.app, store, on_closed=lambda: closed.append(True))
        store.refresh_radar()
        store.loop_playing = True
        self.assertTrue(store.radar_frames)
        win.emit("close-request")
        self.assertEqual(closed, [True])
        self.assertFalse(store.desk_open)
        self.assertFalse(store.loop_playing)
        self.assertGreaterEqual(backend.cancel_count, 1)
        before = len(backend.calls)
        store.on_radar_tick()
        self.assertEqual(len(backend.calls), before)
        win.destroy()

    def test_format_dropdown_changes_status_line(self):
        store = WeatherStore(FakeBackend(), sync=True)
        store.payload = PAYLOAD
        win = DeskWindow(self.app, store)
        try:
            win.format_dropdown.set_selected(2)  # tactical
            text = section_text(win, "wx.status")
            self.assertIn("🔴 [FWA] 72° ↘12mph", text)
            win.format_dropdown.set_selected(0)  # compact
            text = section_text(win, "wx.status")
            self.assertIn("🔴 72°", text)
            self.assertNotIn("[FWA]", text)
        finally:
            win.destroy()


class AppBootTests(unittest.TestCase):
    def test_process_opens_desk_and_close_stops_radar(self):
        root = Path(__file__).resolve().parents[2]
        with tempfile.TemporaryDirectory() as tmp:
            binary = Path(tmp) / "wx"
            binary.write_text(
                textwrap.dedent(
                    """\
                    #!/usr/bin/env python3
                    import json, sys
                    cmd = sys.argv[1] if len(sys.argv) > 1 and not sys.argv[1].startswith("-") else ""
                    if cmd == "radar":
                        json.dump({"product":"composite-reflectivity","product_label":"Composite","location":"Stub","valid_time":"2026-10-04T15:00:00Z","image_base64":"","frames":[]}, sys.stdout)
                    elif cmd == "outlook":
                        json.dump({"location":"Stub","fetched_at":"2026-10-04T15:00:00Z","outlooks":[],"pattern_shift":{"has_shift":False,"summary":""}}, sys.stdout)
                    elif cmd == "chase":
                        json.dump({"generated_at":"2026-10-04T15:00:00Z","total_alerts":0,"total_clusters":0,"clusters":[]}, sys.stdout)
                    else:
                        json.dump({"conditions":{"station":"KFWA","location":"Fort Wayne, IN","temperature_f":72,"description":"Fair","condition_code":"clear-day","nowcast":{"headline":"Dry next 6h","is_active_precip":False,"summary":"Dry","primary_phase":"none","intervals":[]}},"forecast":{"periods":[{"name":"Tonight","temperature_f":60,"short_description":"Clear"}],"hourly":[{"name":"4 PM","start_time":"2026-10-04T20:00:00Z","temperature_f":70}]},"alerts":[{"event":"Wind Advisory","severity":"Minor","headline":"Breezy"}]}, sys.stdout)
                    sys.exit(0)
                    """
                ),
                encoding="utf-8",
            )
            binary.chmod(0o755)
            cfg = Path(tmp) / "config.json"
            script = Path(tmp) / "boot.py"
            script.write_text(
                textwrap.dedent(
                    """\
                    import gi
                    gi.require_version("Adw", "1")
                    from gi.repository import GLib
                    from wxdesk.app import WxApp
                    from wxdesk.prefs import Prefs

                    app = WxApp(prefs=Prefs())
                    def stop():
                        win = app.window
                        assert win is not None, "desk did not open"
                        text = win.status_line.get_label()
                        assert "🟠" in text, text
                        assert app.store.desk_open
                        win.emit("close-request")
                        assert app.window is None
                        assert app.store.desk_open is False
                        assert app.store.loop_playing is False
                        app.quit_app()
                        return False
                    GLib.timeout_add(700, stop)
                    raise SystemExit(app.run(None))
                    """
                ),
                encoding="utf-8",
            )
            env = os.environ.copy()
            env["PYTHONPATH"] = str(root / "linux")
            env["WX_CONFIG"] = str(cfg)
            env["WX_BINARY"] = str(binary)
            env["WX_REFRESH_SECONDS"] = "3600"
            env["WX_RADAR_REFRESH_SECONDS"] = "3600"
            proc = subprocess.run(
                [sys.executable, str(script)],
                env=env,
                cwd=str(root),
                capture_output=True,
                text=True,
                timeout=20,
            )
            self.assertEqual(proc.returncode, 0, proc.stdout + "\n" + proc.stderr)
            self.assertNotIn("Gtk-CRITICAL", proc.stderr)
            self.assertNotIn("Traceback", proc.stderr)

