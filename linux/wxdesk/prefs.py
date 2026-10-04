"""Shell preferences. Same file and WX_CONFIG override as the Mac shell.

The wx CLI keeps its own loader and does not read WX_CONFIG. This module
only stores desk preferences (location, units, menu-bar format, favorites).
Unknown keys are preserved. Nothing here is weather data.
"""

from __future__ import annotations

import json
import os
import tempfile

DEFAULT_NAME = os.path.join(".config", "wx", "config.json")


def config_path() -> str:
    override = os.environ.get("WX_CONFIG", "").strip()
    if override:
        return override
    return os.path.join(os.path.expanduser("~"), DEFAULT_NAME)


def load(path: str | None = None) -> dict:
    path = path or config_path()
    try:
        with open(path, encoding="utf-8") as fh:
            data = json.load(fh)
    except (OSError, json.JSONDecodeError):
        return {}
    return data if isinstance(data, dict) else {}


def save(data: dict, path: str | None = None) -> None:
    path = path or config_path()
    os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
    payload = json.dumps(data, indent=2, ensure_ascii=False) + "\n"
    directory = os.path.dirname(path) or "."
    fd, tmp = tempfile.mkstemp(prefix=".config-", suffix=".json.tmp", dir=directory)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as fh:
            fh.write(payload)
            fh.flush()
            os.fsync(fh.fileno())
        os.replace(tmp, path)
    except Exception:
        try:
            os.unlink(tmp)
        except OSError:
            pass
        raise


def update(mutator, path: str | None = None) -> dict:
    path = path or config_path()
    data = load(path)
    mutator(data)
    save(data, path)
    return data


class Prefs:
    def __init__(self, path: str | None = None):
        self.path = path or config_path()

    def load(self) -> dict:
        return load(self.path)

    def set_menu_bar_format(self, fmt: str) -> None:
        update(lambda cfg: cfg.__setitem__("menu_bar_format", fmt), self.path)

    def set_location_and_units(self, location: str | None, units: str | None) -> None:
        def mutate(cfg: dict) -> None:
            if location:
                cfg["default_location"] = location
            if units:
                cfg["units"] = units

        update(mutate, self.path)

    def add_recent(self, location: str) -> None:
        clean = location.strip()
        if not clean:
            return

        def mutate(cfg: dict) -> None:
            recents = [r for r in (cfg.get("recent_locations") or []) if str(r).lower() != clean.lower()]
            recents.insert(0, clean)
            cfg["recent_locations"] = recents[:10]

        update(mutate, self.path)

    def add_favorite(self, name: str, value: str) -> None:
        clean_name = name.strip()
        clean_value = value.strip()
        if not clean_name or not clean_value:
            return

        def mutate(cfg: dict) -> None:
            favs = list(cfg.get("favorites") or [])
            updated = False
            for entry in favs:
                if str(entry.get("name", "")).lower() == clean_name.lower():
                    entry["name"] = clean_name
                    entry["value"] = clean_value
                    updated = True
                    break
            if not updated:
                favs.append({"name": clean_name, "value": clean_value})
            cfg["favorites"] = favs

        update(mutate, self.path)

    def remove_favorite(self, name_or_value: str) -> None:
        key = name_or_value.strip().lower()
        if not key:
            return

        def mutate(cfg: dict) -> None:
            favs = []
            for entry in cfg.get("favorites") or []:
                if str(entry.get("name", "")).lower() == key or str(entry.get("value", "")).lower() == key:
                    continue
                favs.append(entry)
            cfg["favorites"] = favs

        update(mutate, self.path)
