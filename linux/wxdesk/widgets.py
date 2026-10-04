"""Small GTK helpers for the desk shell."""

from __future__ import annotations

import gi

gi.require_version("Gtk", "4.0")
from gi.repository import Gtk


def label(text: str, *classes: str, wrap: bool = False, name: str | None = None) -> Gtk.Label:
    widget = Gtk.Label(label=text, xalign=0)
    widget.add_css_class("wx-label")
    for css in classes:
        widget.add_css_class(css)
    if wrap:
        widget.set_wrap(True)
        widget.set_wrap_mode(Gtk.WrapMode.WORD_CHAR)
    if name:
        widget.set_name(name)
    return widget


def card(title: str, tag: str) -> tuple[Gtk.Box, Gtk.Box]:
    outer = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=6)
    outer.add_css_class("wx-card")
    header = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=8)
    header.append(label(title.upper(), "wx-kicker"))
    spacer = Gtk.Box()
    spacer.set_hexpand(True)
    header.append(spacer)
    header.append(label(tag, "wx-tag"))
    outer.append(header)
    body = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=4)
    outer.append(body)
    return outer, body


def clear(box: Gtk.Widget) -> None:
    child = box.get_first_child()
    while child is not None:
        nxt = child.get_next_sibling()
        box.remove(child)
        child = nxt


def find_named(root: Gtk.Widget, name: str) -> Gtk.Widget | None:
    if root.get_name() == name:
        return root
    child = root.get_first_child()
    while child is not None:
        found = find_named(child, name)
        if found is not None:
            return found
        child = child.get_next_sibling()
    return None


def collect_text(root: Gtk.Widget) -> str:
    parts = []
    if isinstance(root, Gtk.Label):
        parts.append(root.get_label() or "")
    child = root.get_first_child()
    while child is not None:
        parts.append(collect_text(child))
        child = child.get_next_sibling()
    return "\n".join(p for p in parts if p)
