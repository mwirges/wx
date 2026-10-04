"""Display strings copied from the Mac shell. No forecasts are computed here."""

from __future__ import annotations

from datetime import datetime

FORMATS = ("compact", "standard", "tactical")

FORMAT_LABELS = {
    "compact": "Compact (72°)",
    "standard": "Standard (☀️ 72°)",
    "tactical": "Tactical ([FWA] 72° ↘12mph)",
}

# Freedesktop icon names. The Mac shell maps the same condition_code to SF Symbols.
CONDITION_ICONS = {
    "clear-day": "weather-clear",
    "clear-night": "weather-clear-night",
    "partly-cloudy-day": "weather-few-clouds",
    "partly-cloudy-night": "weather-few-clouds-night",
    "cloudy": "weather-overcast",
    "rain": "weather-showers",
    "heavy-rain": "weather-showers",
    "snow": "weather-snow",
    "sleet": "weather-snow-rain",
    "thunder": "weather-storm",
    "fog": "weather-fog",
    "wind": "weather-windy",
}


def conditions(payload: dict | None) -> dict:
    if not payload:
        return {}
    cond = payload.get("conditions")
    return cond if isinstance(cond, dict) else {}


def alerts(payload: dict | None) -> list:
    if not payload:
        return []
    raw = payload.get("alerts")
    return raw if isinstance(raw, list) else []


def is_warning(alert: dict) -> bool:
    event = str(alert.get("event") or "")
    severity = str(alert.get("severity") or "")
    return "warning" in event.lower() or severity.lower() in ("extreme", "severe")


def is_watch(alert: dict) -> bool:
    return "watch" in str(alert.get("event") or "").lower()


def is_advisory(alert: dict) -> bool:
    event = str(alert.get("event") or "").lower()
    return "advisory" in event or "statement" in event


def pip_kind(payload: dict | None) -> str:
    """Steady hazard pip. Warning wins over watch. Empty when the sky is quiet."""
    items = alerts(payload)
    if any(is_warning(a) for a in items if isinstance(a, dict)):
        return "warning"
    if any(is_watch(a) or is_advisory(a) for a in items if isinstance(a, dict)):
        return "watch"
    return ""


def pip_glyph(payload: dict | None) -> str:
    kind = pip_kind(payload)
    if kind == "warning":
        return "🔴 "
    if kind == "watch":
        return "🟠 "
    return ""


def display_temp(payload: dict | None, units: str) -> str:
    cond = conditions(payload)
    if units == "metric":
        value = cond.get("temperature_c")
    else:
        value = cond.get("temperature_f")
    if value is None:
        return "--"
    return f"{float(value):.0f}°"


def condition_icon(payload: dict | None) -> str:
    code = str(conditions(payload).get("condition_code") or "").lower()
    return CONDITION_ICONS.get(code, "weather-few-clouds")


def wind_arrow(direction: str | None) -> str:
    # Same table as WeatherStore.windArrow. The arrow shows where the wind blows.
    if not direction:
        return ""
    key = direction.upper().strip()
    return {
        "N": "↓",
        "NNE": "↙",
        "NE": "↙",
        "ENE": "←",
        "E": "←",
        "ESE": "↖",
        "SE": "↖",
        "SSE": "↑",
        "S": "↑",
        "SSW": "↗",
        "SW": "↗",
        "WSW": "→",
        "W": "→",
        "WNW": "↘",
        "NW": "↘",
        "NNW": "↓",
    }.get(key, "")


def tactical_station_tag(payload: dict | None) -> str:
    cond = conditions(payload)
    station = str(cond.get("station") or "")
    if station:
        upper = station.upper()
        if len(upper) == 4 and upper.startswith("K"):
            return upper[1:]
        if len(upper) <= 4 and upper != "OPENMETEO":
            return upper
    location = str(cond.get("location") or "")
    if location:
        first = location.split()[0]
        letters = "".join(ch for ch in first if ch.isalpha())[:3].upper()
        if letters:
            return letters
    return "WX"


def tactical_wind(payload: dict | None, units: str) -> str:
    cond = conditions(payload)
    if units == "metric":
        speed = cond.get("wind_kph")
        text = f"{float(speed):.0f}km/h" if speed is not None else ""
    else:
        speed = cond.get("wind_mph")
        text = f"{float(speed):.0f}mph" if speed is not None else ""
    if not text:
        return ""
    return f"{wind_arrow(cond.get('wind_direction'))}{text}"


def tactical_status(payload: dict | None, units: str) -> str:
    tag = tactical_station_tag(payload)
    temp = display_temp(payload, units)
    wind = tactical_wind(payload, units)
    if not wind:
        return f"[{tag}] {temp}"
    return f"[{tag}] {temp} {wind}"


def status_text(payload: dict | None, units: str, fmt: str, error: str | None = None) -> str:
    """Menu-bar title. Matches StatusItemController.refreshButton."""
    if error and not conditions(payload):
        return " wx?"
    pip = pip_glyph(payload)
    if fmt == "compact":
        return f"{pip}{display_temp(payload, units)}"
    if fmt == "tactical":
        return f"{pip}{tactical_status(payload, units)}"
    # standard: the symbol is a separate icon, so the title keeps a leading space
    if pip:
        return f" {pip}{display_temp(payload, units)}"
    return f" {display_temp(payload, units)}"


def pixmap_text(payload: dict | None, units: str, fmt: str, error: str | None = None) -> str:
    """Status text without the emoji pip. The pixmap draws a steady dot instead."""
    if error and not conditions(payload):
        return "wx?"
    if fmt == "compact":
        return display_temp(payload, units)
    if fmt == "tactical":
        return tactical_status(payload, units)
    return display_temp(payload, units)


def parse_iso(raw: str | None) -> datetime | None:
    if not raw or not isinstance(raw, str):
        return None
    text = raw.strip()
    if text.endswith("Z"):
        text = text[:-1] + "+00:00"
    try:
        return datetime.fromisoformat(text)
    except ValueError:
        return None


def hour_label(period: dict) -> str:
    stamp = parse_iso(period.get("start_time"))
    if stamp is None:
        return str(period.get("name") or "")
    local = stamp.astimezone()
    hour = local.strftime("%I").lstrip("0") or "12"
    return f"{hour} {local.strftime('%p')}"


def period_temp(period: dict, units: str) -> str:
    if units == "metric" and period.get("temperature_c") is not None:
        return f"{float(period['temperature_c']):.0f}°"
    if period.get("temperature_f") is not None:
        return f"{float(period['temperature_f']):.0f}°"
    return "—"


def nowcast_of(payload: dict | None, separate: dict | None = None) -> dict | None:
    if isinstance(separate, dict) and isinstance(separate.get("nowcast"), dict):
        return separate["nowcast"]
    cond = conditions(payload)
    if isinstance(cond.get("nowcast"), dict):
        return cond["nowcast"]
    if payload and isinstance(payload.get("nowcast"), dict):
        return payload["nowcast"]
    return None


def nowcast_badge(nowcast: dict) -> str:
    if nowcast.get("is_active_precip"):
        return "ACTIVE PRECIPITATION"
    if nowcast.get("next_precip_time"):
        return "PRECIPITATION EXPECTED"
    return "ALL CLEAR // DRY"


def tray_menu(*, location: str, temp: str, favorites: list, menu_format: str, cards: list | None = None) -> dict:
    """Nested menu matching the Mac status-item menu."""
    children = []
    if location:
        children.append(_item(f"ACTIVE // {location.upper()} ({temp})", enabled=False))
        children.append(_sep())

    children.append(_item("PINNED LOCATIONS // COMMAND GRID", enabled=False))
    shown = cards if cards is not None else favorites
    if not shown:
        children.append(_item("  No pinned locations (add in Command Grid)", enabled=False))
    else:
        for card in shown:
            if isinstance(card, dict) and "label" in card:
                children.append(_item(card["label"], action="location", target=card.get("target") or card["label"]))
            elif isinstance(card, dict):
                name = card.get("name") or card.get("value") or ""
                value = card.get("value") or name
                children.append(_item(str(name), action="location", target=str(value)))
            else:
                children.append(_item(str(card), action="location", target=str(card)))
    children.append(_item("Manage Command Grid...", action="grid"))
    children.append(_sep())

    formats = []
    for key in FORMATS:
        formats.append(
            _item(
                FORMAT_LABELS[key],
                action="format",
                target=key,
                toggle=key == menu_format,
            )
        )
    children.append({"props": {"label": "Menu Bar Format", "children-display": "submenu"}, "children": formats})
    children.append(_sep())
    children.append(_item("Open Desk Window", action="open-desk"))
    children.append(_item("Refresh Telemetry", action="refresh"))
    children.append(_sep())
    children.append(_item("Quit wx", action="quit"))
    return {"props": {"children-display": "submenu"}, "children": children}


def _item(label: str, enabled: bool = True, action: str | None = None, target: str | None = None, toggle: bool | None = None) -> dict:
    props: dict = {"label": label, "enabled": enabled}
    if toggle is not None:
        props["toggle-type"] = "radio"
        props["toggle-state"] = 1 if toggle else 0
    node = {"props": props, "children": []}
    if action:
        node["action"] = action
    if target is not None:
        node["target"] = target
    return node


def _sep() -> dict:
    return {"props": {"type": "separator"}, "children": []}


def menu_labels(node: dict) -> list[str]:
    found = []
    label = node.get("props", {}).get("label")
    if label:
        found.append(label)
    for child in node.get("children") or []:
        found.extend(menu_labels(child))
    return found
