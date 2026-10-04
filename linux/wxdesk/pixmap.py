"""Tray pixmap. Draws the menu-bar text and a steady hazard dot. No animation."""

from __future__ import annotations

import cairo


def render_status(text: str, pip: str) -> tuple[int, int, bytes]:
    """Return width, height, and ARGB32 bytes in network order."""
    probe = cairo.ImageSurface(cairo.FORMAT_ARGB32, 8, 8)
    ctx = cairo.Context(probe)
    ctx.select_font_face("Sans", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
    ctx.set_font_size(15)
    extents = ctx.text_extents(text or "wx")
    pad = 4
    dot = 10 if pip else 0
    gap = 4 if pip else 0
    width = max(22, int(pad + dot + gap + extents.width + pad + 2))
    height = 22
    surface = cairo.ImageSurface(cairo.FORMAT_ARGB32, width, height)
    ctx = cairo.Context(surface)
    ctx.set_source_rgba(0, 0, 0, 0)
    ctx.paint()
    x = float(pad)
    if pip == "warning":
        ctx.set_source_rgba(1.0, 0x3B / 255, 0x56 / 255, 1)
    elif pip == "watch":
        ctx.set_source_rgba(1.0, 0x9F / 255, 0x0A / 255, 1)
    if pip:
        ctx.arc(x + 4.5, height / 2, 4.0, 0, 6.2832)
        ctx.fill()
        x += dot + gap
    ctx.set_source_rgba(0.941, 0.965, 1.0, 1)
    ctx.select_font_face("Sans", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
    ctx.set_font_size(15)
    ctx.move_to(x, 16)
    ctx.show_text(text or "wx")
    surface.flush()
    return width, height, _argb(surface)


def _argb(surface: cairo.ImageSurface) -> bytes:
    width = surface.get_width()
    height = surface.get_height()
    raw = bytes(surface.get_data())
    out = bytearray(width * height * 4)
    little = True
    for i in range(width * height):
        b0, b1, b2, b3 = raw[i * 4 : i * 4 + 4]
        if little:
            blue, green, red, alpha = b0, b1, b2, b3
        else:
            alpha, red, green, blue = b0, b1, b2, b3
        if alpha and alpha != 255:
            red = min(255, red * 255 // alpha)
            green = min(255, green * 255 // alpha)
            blue = min(255, blue * 255 // alpha)
        out[i * 4 : i * 4 + 4] = bytes((alpha, red, green, blue))
    return bytes(out)


def has_pip_color(blob: bytes, pip: str) -> bool:
    for i in range(0, len(blob), 4):
        _a, red, green, blue = blob[i : i + 4]
        if pip == "warning" and red > 200 and green < 90 and blue < 120:
            return True
        if pip == "watch" and red > 200 and 120 < green < 200 and blue < 40:
            return True
    return False
