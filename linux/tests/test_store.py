import threading
import unittest

from wxdesk.cli import Cancelled
from wxdesk.store import WeatherStore

from fixture import PAYLOAD


class FakeBackend:
    def __init__(self):
        self.calls = []
        self.cancel_count = 0
        self.weather = PAYLOAD
        self.fail = None

    def fetch_weather(self, location, units, hourly=True):
        self.calls.append(("weather", location, units, hourly))
        if self.fail:
            raise RuntimeError(self.fail)
        return self.weather

    def fetch_radar(self, location, product, radius, bbox=None, frames=8):
        self.calls.append(("radar", location, product, radius, frames))
        return {
            "product": product,
            "product_label": "Composite Reflectivity",
            "location": location or "",
            "valid_time": "2026-10-04T15:00:00Z",
            "image_base64": "aGVsbG8=",
            "frames": [
                {"valid_time": "2026-10-04T14:50:00Z", "image_base64": "aGVsbG8="},
                {"valid_time": "2026-10-04T15:00:00Z", "image_base64": "d29ybGQ="},
            ],
        }

    def cancel_radar(self):
        self.cancel_count += 1

    def fetch_outlook(self, location):
        self.calls.append(("outlook", location))
        return {"location": location or "", "outlooks": []}

    def fetch_chase(self):
        self.calls.append(("chase",))
        return {"total_clusters": 0, "clusters": []}

    def fetch_climate(self, location, units):
        self.calls.append(("climate", location, units))
        return {"location": location}

    def fetch_history(self, location, units, days):
        self.calls.append(("history", location, units, days))
        return {"days": []}

    def fetch_tropics(self, location, units, storm=None):
        self.calls.append(("tropics", location))
        return {"tropics": {"storms": [], "total_active": 0}}

    def export_png(self, path, location, product, radius):
        self.calls.append(("export-png", path))
        return "saved"

    def export_gif(self, path, location, product, radius, frames):
        self.calls.append(("export-gif", path, frames))
        return "saved"


def names(backend):
    return [call[0] for call in backend.calls]


class StoreLifecycleTests(unittest.TestCase):
    def setUp(self):
        self.backend = FakeBackend()
        self.store = WeatherStore(self.backend, sync=True)
        self.store.favorites = [{"name": "Home", "value": "Fort Wayne, IN"}]

    def test_close_stops_radar_fetch_and_loop(self):
        self.store.open_desk()
        self.store.refresh_radar()
        self.assertIn("radar", names(self.backend))
        self.store.loop_playing = True
        self.store.close_desk()
        self.assertFalse(self.store.desk_open)
        self.assertFalse(self.store.loop_playing)
        self.assertGreaterEqual(self.backend.cancel_count, 1)
        before = len(self.backend.calls)
        self.store.on_radar_tick()
        self.assertEqual(len(self.backend.calls), before)
        self.store.on_menu_tick()
        self.assertIn("weather", names(self.backend))
        self.assertNotIn("outlook", names(self.backend)[before:])
        self.assertNotIn("chase", names(self.backend))

    def test_open_desk_radar_tick_fetches(self):
        self.store.open_desk()
        self.store.on_radar_tick()
        self.assertIn("radar", names(self.backend))

    def test_closed_desk_skips_extras(self):
        self.store.close_desk()
        self.store.refresh_desk_extras()
        self.store.refresh_outlook()
        self.store.refresh_chase()
        self.store.refresh_grid()
        self.assertNotIn("outlook", names(self.backend))
        self.assertNotIn("chase", names(self.backend))
        self.assertFalse(any(call[0] == "weather" and call[3] is False for call in self.backend.calls))

    def test_open_desk_extras_and_grid_skip_hourly(self):
        self.store.open_desk()
        self.store.refresh_desk_extras()
        self.assertIn("outlook", names(self.backend))
        self.assertIn("chase", names(self.backend))
        grid = [call for call in self.backend.calls if call[0] == "weather" and call[3] is False]
        self.assertEqual(len(grid), 1)
        self.assertEqual(grid[0][1], "Fort Wayne, IN")
        hourly = [call for call in self.backend.calls if call[0] == "weather" and call[3] is True]
        self.assertEqual(hourly, [])

    def test_stale_radar_is_dropped_after_close(self):
        self.store.open_desk()
        self.store.radar_gen = 4
        self.backend.fetch_radar(None, "composite-reflectivity", 200)
        self.store.close_desk()
        self.store.radar_frames = []
        gen = 4
        self.store.desk_open = False
        self.store.radar_gen = 5

        def apply_like_worker():
            if gen != self.store.radar_gen or not self.store.desk_open:
                return
            self.store.radar_frames = [{"png": b"nope"}]

        apply_like_worker()
        self.assertEqual(self.store.radar_frames, [])

    def test_menu_refresh_requests_hourly(self):
        self.store.refresh()
        weather = self.backend.calls[0]
        self.assertEqual(weather[0], "weather")
        self.assertTrue(weather[3])
        self.assertEqual(self.store.payload["conditions"]["location"], "Fort Wayne, IN")

    def test_reopen_builds_a_fresh_open_flag(self):
        self.store.open_desk()
        self.store.close_desk()
        self.assertIsNone(self.store.window if hasattr(self.store, "window") else None)
        self.store.open_desk()
        self.assertTrue(self.store.desk_open)

    def test_loop_advance_stops_when_desk_closes(self):
        self.store.open_desk()
        self.store.refresh_radar()
        self.store.loop_playing = True
        self.store.frame_index = 0
        self.store.advance_frame()
        self.assertEqual(self.store.frame_index, 1)
        self.store.close_desk()
        self.store.advance_frame()
        self.assertFalse(self.store.loop_playing)


class InFlightTests(unittest.TestCase):
    def test_close_discards_inflight_radar(self):
        started = threading.Event()
        release = threading.Event()

        class Blocking(FakeBackend):
            def fetch_radar(self, location, product, radius, bbox=None, frames=8):
                self.calls.append(("radar", location, product, radius, frames))
                started.set()
                release.wait(2)
                if self.cancel_count:
                    raise Cancelled("cancelled")
                return super().fetch_radar(location, product, radius, bbox, frames)

            def cancel_radar(self):
                super().cancel_radar()
                release.set()

        backend = Blocking()
        store = WeatherStore(backend, sync=False)
        store.open_desk()
        store.refresh_radar()
        self.assertTrue(started.wait(2))
        store.close_desk()
        # Let the worker deliver. Give the thread a moment.
        release.wait(1)
        threading.Event().wait(0.2)
        self.assertFalse(store.desk_open)
        self.assertEqual(store.radar_frames, [])
        self.assertGreaterEqual(backend.cancel_count, 1)
