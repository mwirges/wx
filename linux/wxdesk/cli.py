"""Run the wx CLI and return its JSON. This module does not interpret weather."""

from __future__ import annotations

import json
import os
import signal
import subprocess
import threading
from pathlib import Path

WEATHER_TIMEOUT = 35
RADAR_TIMEOUT = 45
EXTRA_TIMEOUT = 25


class CLIError(Exception):
    pass


class Cancelled(CLIError):
    pass


def locate_binary(explicit: str | None = None) -> str | None:
    candidates = []
    env = explicit if explicit is not None else os.environ.get("WX_BINARY", "")
    if env.strip():
        candidates.append(env.strip())
    repo = Path(__file__).resolve().parents[2]
    candidates.append(str(repo / "build" / "wx"))
    path_env = os.environ.get("PATH", "")
    for directory in path_env.split(":"):
        if directory:
            candidates.append(str(Path(directory) / "wx"))
    home = Path.home()
    candidates.extend(
        [
            str(home / "bin" / "wx"),
            str(home / "go" / "bin" / "wx"),
            str(home / ".local" / "bin" / "wx"),
            "/usr/local/bin/wx",
            "/usr/bin/wx",
        ]
    )
    seen = set()
    for candidate in candidates:
        if candidate in seen:
            continue
        seen.add(candidate)
        if os.path.isfile(candidate) and os.access(candidate, os.X_OK):
            return candidate
    return None


def weather_args(location: str | None, units: str | None, hourly: bool = False) -> list[str]:
    args = ["--json", "--forecast", "--alerts"]
    if hourly:
        args += ["--hourly", "--hours", "24"]
    if location and location.strip():
        args += ["--location", location.strip()]
    if units:
        args += ["--units", units]
    return args


def radar_args(
    location: str | None,
    product: str | None = None,
    radius_km: float | None = None,
    bbox: str | None = None,
    raw: bool = True,
    loop: bool = True,
    frames: int = 8,
) -> list[str]:
    args = ["radar", "--json"]
    if raw:
        args.append("--raw")
    if loop:
        args += ["--loop", "--frames", str(max(2, frames))]
    if bbox:
        args += ["--bbox", bbox]
    if location and location.strip():
        args += ["--location", location.strip()]
    if product:
        args += ["--product", product]
    if not bbox and radius_km and radius_km > 0:
        args += ["--radius", f"{radius_km:.0f}"]
    return args


def export_png_args(path: str, location: str | None, product: str | None, radius_km: float | None) -> list[str]:
    args = ["radar", "--save", path]
    _append_radar_target(args, location, product, radius_km)
    return args


def export_gif_args(
    path: str,
    location: str | None,
    product: str | None,
    radius_km: float | None,
    frames: int = 8,
    interval_ms: int = 500,
) -> list[str]:
    args = ["radar", "--save-gif", path, "--frames", str(max(2, frames)), "--interval", str(interval_ms)]
    _append_radar_target(args, location, product, radius_km)
    return args


def outlook_args(location: str | None) -> list[str]:
    args = ["outlook", "--json"]
    if location and location.strip():
        args += ["--location", location.strip()]
    return args


def chase_args() -> list[str]:
    return ["chase", "--list", "--json"]


def climate_args(location: str | None, units: str | None) -> list[str]:
    args = ["climate", "--json"]
    if location and location.strip():
        args += ["--location", location.strip()]
    if units:
        args += ["--units", units]
    return args


def history_args(location: str | None, units: str | None, days: int = 14) -> list[str]:
    args = ["history", "--json", "--days", str(days)]
    if location and location.strip():
        args += ["--location", location.strip()]
    if units:
        args += ["--units", units]
    return args


def tropics_args(location: str | None, units: str | None, storm: str | None = None) -> list[str]:
    args = ["tropics", "--json"]
    if location and location.strip():
        args += ["--location", location.strip()]
    if storm and storm.strip():
        args += ["--storm", storm.strip()]
    if units:
        args += ["--units", units]
    return args


def nowcast_args(location: str | None, units: str | None) -> list[str]:
    args = ["nowcast", "--json"]
    if location and location.strip():
        args += ["--location", location.strip()]
    if units:
        args += ["--units", units]
    return args


def _append_radar_target(args: list[str], location: str | None, product: str | None, radius_km: float | None) -> None:
    if location and location.strip():
        args += ["--location", location.strip()]
    if product:
        args += ["--product", product]
    if radius_km and radius_km > 0:
        args += ["--radius", f"{radius_km:.0f}"]


def _kill(proc: subprocess.Popen) -> None:
    if proc.poll() is not None:
        return
    try:
        os.killpg(proc.pid, signal.SIGTERM)
    except ProcessLookupError:
        return
    try:
        proc.wait(timeout=0.6)
    except subprocess.TimeoutExpired:
        try:
            os.killpg(proc.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        try:
            proc.wait(timeout=1)
        except subprocess.TimeoutExpired:
            pass


class WxCLI:
    def __init__(self, binary: str | None = None):
        self.binary = binary or locate_binary()
        self._lock = threading.Lock()
        self._procs: set[subprocess.Popen] = set()
        self._epoch = 0

    @property
    def available(self) -> bool:
        return bool(self.binary)

    def cancel_radar(self) -> None:
        with self._lock:
            self._epoch += 1
            procs = list(self._procs)
        for proc in procs:
            _kill(proc)

    def fetch_weather(self, location: str | None, units: str | None, hourly: bool = True) -> dict:
        out, err, code = self._run(weather_args(location, units, hourly=hourly), WEATHER_TIMEOUT, tracked=False)
        payload = _decode(out, err, code, require_conditions=True)
        return payload

    def fetch_radar(self, location, product, radius_km, bbox=None, frames: int = 8) -> dict:
        self.cancel_radar()
        with self._lock:
            epoch = self._epoch
        out, err, code = self._run(
            radar_args(location, product=product, radius_km=radius_km, bbox=bbox, frames=frames),
            RADAR_TIMEOUT,
            tracked=True,
            epoch=epoch,
        )
        return _decode(out, err, code, require_conditions=False)

    def export_png(self, path: str, location, product, radius_km) -> str:
        _out, err, code = self._run(
            export_png_args(path, location, product, radius_km),
            RADAR_TIMEOUT,
            tracked=True,
        )
        if code != 0:
            raise CLIError(err.strip() or f"wx exited with status {code}.")
        return err.strip()

    def export_gif(self, path: str, location, product, radius_km, frames: int = 8) -> str:
        _out, err, code = self._run(
            export_gif_args(path, location, product, radius_km, frames=frames),
            RADAR_TIMEOUT,
            tracked=True,
        )
        if code != 0:
            raise CLIError(err.strip() or f"wx exited with status {code}.")
        return err.strip()

    def fetch_outlook(self, location: str | None) -> dict:
        return self._extra(outlook_args(location))

    def fetch_chase(self) -> dict:
        return self._extra(chase_args())

    def fetch_climate(self, location: str | None, units: str | None) -> dict:
        return self._extra(climate_args(location, units))

    def fetch_history(self, location: str | None, units: str | None, days: int = 14) -> dict:
        return self._extra(history_args(location, units, days))

    def fetch_tropics(self, location: str | None, units: str | None, storm: str | None = None) -> dict:
        return self._extra(tropics_args(location, units, storm))

    def fetch_nowcast(self, location: str | None, units: str | None) -> dict:
        return self._extra(nowcast_args(location, units))

    def _extra(self, args: list[str]) -> dict:
        out, err, code = self._run(args, EXTRA_TIMEOUT, tracked=False)
        return _decode(out, err, code, require_conditions=False)

    def _run(self, args: list[str], timeout: float, tracked: bool, epoch: int | None = None):
        if not self.binary:
            raise CLIError(
                "wx CLI binary not found. Build the project (`make build`) or set WX_BINARY."
            )
        proc = subprocess.Popen(
            [self.binary, *args],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            start_new_session=True,
        )
        if tracked:
            with self._lock:
                if epoch is not None and self._epoch != epoch:
                    _kill(proc)
                    raise Cancelled("cancelled")
                self._procs.add(proc)
        try:
            try:
                out, err = proc.communicate(timeout=timeout)
            except subprocess.TimeoutExpired:
                _kill(proc)
                raise CLIError(f"wx timed out after {int(timeout)}s.") from None
        finally:
            if tracked:
                with self._lock:
                    self._procs.discard(proc)
        if epoch is not None:
            with self._lock:
                current = self._epoch
            if current != epoch or proc.returncode < 0:
                raise Cancelled("cancelled")
        if tracked and proc.returncode is not None and proc.returncode < 0:
            raise Cancelled("cancelled")
        stdout = out or b""
        stderr = (err or b"").decode("utf-8", "replace")
        return stdout, stderr, proc.returncode if proc.returncode is not None else 1


def _decode(stdout: bytes, stderr: str, code: int, require_conditions: bool) -> dict:
    text = stdout.decode("utf-8", "replace").strip()
    if code != 0 and not text:
        raise CLIError(stderr.strip() or f"wx exited with status {code}.")
    try:
        payload = json.loads(text) if text else {}
    except json.JSONDecodeError as exc:
        raise CLIError(f"Failed to decode wx JSON: {exc}") from exc
    if not isinstance(payload, dict):
        raise CLIError("Failed to decode wx JSON: expected an object")
    if require_conditions and payload.get("conditions") is None and code != 0:
        raise CLIError(stderr.strip() or f"wx exited with status {code}.")
    if require_conditions and not payload.get("warning"):
        warn = stderr.strip()
        if warn:
            payload["warning"] = warn
    return payload
