"""StatusNotifierItem tray. The label matches the Mac menu bar, including the hazard pip.

A missing StatusNotifier host does not stop the desk window. The same menu
is published on the session bus when a host is present.
"""

from __future__ import annotations

import os

import gi

gi.require_version("Gio", "2.0")
from gi.repository import Gio, GLib

from wxdesk import present
from wxdesk.pixmap import render_status

SNI_XML = """
<node>
  <interface name="org.kde.StatusNotifierItem">
    <method name="Activate">
      <arg name="x" type="i" direction="in"/>
      <arg name="y" type="i" direction="in"/>
    </method>
    <method name="SecondaryActivate">
      <arg name="x" type="i" direction="in"/>
      <arg name="y" type="i" direction="in"/>
    </method>
    <method name="Scroll">
      <arg name="delta" type="i" direction="in"/>
      <arg name="orientation" type="s" direction="in"/>
    </method>
    <method name="ContextMenu">
      <arg name="x" type="i" direction="in"/>
      <arg name="y" type="i" direction="in"/>
    </method>
    <property name="Category" type="s" access="read"/>
    <property name="Id" type="s" access="read"/>
    <property name="Title" type="s" access="read"/>
    <property name="Status" type="s" access="read"/>
    <property name="IconName" type="s" access="read"/>
    <property name="IconPixmap" type="a(iiay)" access="read"/>
    <property name="OverlayIconName" type="s" access="read"/>
    <property name="OverlayIconPixmap" type="a(iiay)" access="read"/>
    <property name="AttentionIconName" type="s" access="read"/>
    <property name="ToolTip" type="(sa(iiay)ss)" access="read"/>
    <property name="Menu" type="o" access="read"/>
    <property name="ItemIsMenu" type="b" access="read"/>
    <property name="XAyatanaLabel" type="s" access="read"/>
    <property name="XAyatanaLabelGuide" type="s" access="read"/>
    <signal name="NewTitle"/>
    <signal name="NewIcon"/>
    <signal name="NewToolTip"/>
    <signal name="NewStatus"/>
  </interface>
</node>
"""

MENU_XML = """
<node>
  <interface name="com.canonical.dbusmenu">
    <method name="GetLayout">
      <arg name="parentId" type="i" direction="in"/>
      <arg name="recursionDepth" type="i" direction="in"/>
      <arg name="propertyNames" type="as" direction="in"/>
      <arg name="revision" type="u" direction="out"/>
      <arg name="layout" type="(ia{sv}av)" direction="out"/>
    </method>
    <method name="GetGroupProperties">
      <arg name="ids" type="ai" direction="in"/>
      <arg name="propertyNames" type="as" direction="in"/>
      <arg name="properties" type="a(ia{sv})" direction="out"/>
    </method>
    <method name="GetProperty">
      <arg name="id" type="i" direction="in"/>
      <arg name="name" type="s" direction="in"/>
      <arg name="value" type="v" direction="out"/>
    </method>
    <method name="Event">
      <arg name="id" type="i" direction="in"/>
      <arg name="eventId" type="s" direction="in"/>
      <arg name="data" type="v" direction="in"/>
      <arg name="timestamp" type="u" direction="in"/>
    </method>
    <method name="AboutToShow">
      <arg name="id" type="i" direction="in"/>
      <arg name="needUpdate" type="b" direction="out"/>
    </method>
    <property name="Version" type="u" access="read"/>
    <property name="Status" type="s" access="read"/>
    <signal name="LayoutUpdated">
      <arg name="revision" type="u"/>
      <arg name="parent" type="i"/>
    </signal>
    <signal name="ItemsPropertiesUpdated">
      <arg name="updatedProps" type="a(ia{sv})"/>
      <arg name="removedProps" type="a(ias)"/>
    </signal>
  </interface>
</node>
"""

ITEM_PATH = "/StatusNotifierItem"
MENU_PATH = "/StatusNotifierItem/Menu"


def number_menu(node: dict, counter: list[int] | None = None) -> dict:
    if counter is None:
        counter = [0]
    numbered = {
        "id": counter[0],
        "props": dict(node.get("props") or {}),
        "children": [],
        "action": node.get("action"),
        "target": node.get("target"),
    }
    counter[0] += 1
    for child in node.get("children") or []:
        numbered["children"].append(number_menu(child, counter))
    return numbered


def flatten_menu(node: dict, found: dict | None = None) -> dict:
    if found is None:
        found = {}
    found[node["id"]] = node
    for child in node.get("children") or []:
        flatten_menu(child, found)
    return found


def layout_variant(node: dict) -> GLib.Variant:
    props = GLib.VariantBuilder.new(GLib.VariantType("a{sv}"))
    for key, value in (node.get("props") or {}).items():
        props.add_value(_dict_entry(key, value))
    kids = GLib.VariantBuilder.new(GLib.VariantType("av"))
    for child in node.get("children") or []:
        kids.add_value(GLib.Variant.new_variant(layout_variant(child)))
    return GLib.Variant.new_tuple(
        GLib.Variant("i", int(node.get("id") or 0)),
        props.end(),
        kids.end(),
    )


def _dict_entry(key: str, value) -> GLib.Variant:
    if isinstance(value, bool):
        packed = GLib.Variant("b", value)
    elif isinstance(value, int) and not isinstance(value, bool):
        packed = GLib.Variant("i", value)
    else:
        packed = GLib.Variant("s", str(value))
    return GLib.Variant.new_dict_entry(GLib.Variant("s", key), GLib.Variant.new_variant(packed))


class Tray:
    def __init__(self, store, on_action):
        self.store = store
        self.on_action = on_action
        self.available = False
        self.error = ""
        self.revision = 1
        self._conn = None
        self._owner = 0
        self._regs = []
        self._menu = number_menu(self._model())
        self._by_id = flatten_menu(self._menu)
        self._pixmap = (0, 0, b"")
        store.subscribe(self._on_store)

    def status_label(self) -> str:
        return present.status_text(
            self.store.payload, self.store.units, self.store.menu_bar_format, self.store.error
        )

    def install(self) -> bool:
        name = f"org.kde.StatusNotifierItem-{os.getpid()}-1"
        self._owner = Gio.bus_own_name(
            Gio.BusType.SESSION,
            name,
            Gio.BusNameOwnerFlags.NONE,
            self._on_bus,
            self._on_name,
            self._on_lost,
        )
        return True

    def shutdown(self) -> None:
        if self._owner:
            Gio.bus_unown_name(self._owner)
            self._owner = 0

    def _on_store(self, _topic: str) -> None:
        self._menu = number_menu(self._model())
        self._by_id = flatten_menu(self._menu)
        self.revision += 1
        self._pixmap = self._render()
        self._emit("org.kde.StatusNotifierItem", "NewIcon", ITEM_PATH)
        self._emit("org.kde.StatusNotifierItem", "NewTitle", ITEM_PATH)
        self._emit("org.kde.StatusNotifierItem", "NewToolTip", ITEM_PATH)
        self._emit(
            "com.canonical.dbusmenu",
            "LayoutUpdated",
            MENU_PATH,
            GLib.Variant("(ui)", (self.revision, 0)),
        )

    def _model(self) -> dict:
        cond = present.conditions(self.store.payload)
        location = str(cond.get("location") or self.store.location or "")
        temp = present.display_temp(self.store.payload, self.store.units)
        cards = []
        for card in self.store.grid_cards or self.store.favorites:
            if card.get("display_name") or card.get("name"):
                name = card.get("display_name") or card.get("name")
                target = card.get("location_key") or card.get("value") or name
                cards.append({"label": str(name), "target": str(target)})
        return present.tray_menu(
            location=location,
            temp=temp,
            favorites=self.store.favorites,
            menu_format=self.store.menu_bar_format,
            cards=cards or None,
        )

    def _render(self) -> tuple[int, int, bytes]:
        text = present.pixmap_text(
            self.store.payload, self.store.units, self.store.menu_bar_format, self.store.error
        )
        return render_status(text, present.pip_kind(self.store.payload))

    def _on_bus(self, connection, _name) -> None:
        self._conn = connection
        self._pixmap = self._render()
        sni = Gio.DBusNodeInfo.new_for_xml(SNI_XML).interfaces[0]
        menu = Gio.DBusNodeInfo.new_for_xml(MENU_XML).interfaces[0]
        self._regs.append(connection.register_object(ITEM_PATH, sni, self._sni_method, self._sni_get, self._sni_set))
        self._regs.append(connection.register_object(MENU_PATH, menu, self._menu_method, self._menu_get, self._sni_set))

    def _on_name(self, connection, name) -> None:
        self.available = False
        try:
            connection.call_sync(
                "org.kde.StatusNotifierWatcher",
                "/StatusNotifierWatcher",
                "org.kde.StatusNotifierWatcher",
                "RegisterStatusNotifierItem",
                GLib.Variant("(s)", (name,)),
                None,
                Gio.DBusCallFlags.NONE,
                1500,
                None,
            )
            self.available = True
            self.error = ""
        except GLib.Error as exc:
            self.available = False
            self.error = str(exc)

    def _on_lost(self, _connection, _name) -> None:
        self.available = False

    def _emit(self, interface: str, signal: str, path: str, params: GLib.Variant | None = None) -> None:
        if self._conn is None:
            return
        try:
            self._conn.emit_signal(None, path, interface, signal, params)
        except GLib.Error:
            pass

    def _sni_method(self, _conn, _sender, _path, _iface, method, _params, invocation) -> None:
        if method in {"Activate", "SecondaryActivate", "ContextMenu"}:
            self.on_action("open-desk", None)
            invocation.return_value(None)
            return
        if method == "Scroll":
            invocation.return_value(None)
            return
        invocation.return_error_literal(Gio.dbus_error_quark(), Gio.DBusError.UNKNOWN_METHOD, method)

    def _sni_get(self, _conn, _sender, _path, _iface, prop):
        label = self.status_label()
        fmt = self.store.menu_bar_format
        icon = present.condition_icon(self.store.payload) if fmt == "standard" else ""
        width, height, blob = self._pixmap if self._pixmap[2] else self._render()
        pixmap = GLib.Variant("a(iiay)", [(width, height, blob)])
        values = {
            "Category": GLib.Variant("s", "ApplicationStatus"),
            "Id": GLib.Variant("s", "wx"),
            "Title": GLib.Variant("s", label),
            "Status": GLib.Variant("s", "Active"),
            "IconName": GLib.Variant("s", icon),
            "IconPixmap": pixmap,
            "OverlayIconName": GLib.Variant("s", ""),
            "OverlayIconPixmap": GLib.Variant("a(iiay)", []),
            "AttentionIconName": GLib.Variant("s", ""),
            "ToolTip": GLib.Variant("(sa(iiay)ss)", (icon or "wx", [], "wx", label)),
            "Menu": GLib.Variant("o", MENU_PATH),
            "ItemIsMenu": GLib.Variant("b", False),
            "XAyatanaLabel": GLib.Variant("s", label),
            "XAyatanaLabelGuide": GLib.Variant("s", "🔴 [WWWW] 000° ↘000mph"),
        }
        return values.get(prop)

    def _sni_set(self, *_args) -> bool:
        return False

    def _menu_get(self, _conn, _sender, _path, _iface, prop):
        if prop == "Version":
            return GLib.Variant("u", 3)
        if prop == "Status":
            return GLib.Variant("s", "normal")
        return None

    def _menu_method(self, _conn, _sender, _path, _iface, method, params, invocation) -> None:
        if method == "GetLayout":
            parent, _depth, _names = params.unpack()
            node = self._by_id.get(int(parent), self._menu)
            invocation.return_value(GLib.Variant.new_tuple(GLib.Variant("u", self.revision), layout_variant(node)))
            return
        if method == "AboutToShow":
            invocation.return_value(GLib.Variant.new_tuple(GLib.Variant("b", False)))
            return
        if method == "Event":
            item_id, event_id, _data, _stamp = params.unpack()
            if event_id == "clicked":
                node = self._by_id.get(int(item_id))
                if node and node.get("action"):
                    self.on_action(node["action"], node.get("target"))
            invocation.return_value(None)
            return
        if method == "GetGroupProperties":
            invocation.return_value(GLib.Variant.new_tuple(GLib.Variant("a(ia{sv})", [])))
            return
        if method == "GetProperty":
            item_id, name = params.unpack()
            node = self._by_id.get(int(item_id))
            value = None if node is None else node.get("props", {}).get(name)
            if value is None:
                invocation.return_value(GLib.Variant.new_tuple(GLib.Variant("v", GLib.Variant("s", ""))))
                return
            if isinstance(value, bool):
                packed = GLib.Variant("b", value)
            elif isinstance(value, int):
                packed = GLib.Variant("i", value)
            else:
                packed = GLib.Variant("s", str(value))
            invocation.return_value(GLib.Variant.new_tuple(GLib.Variant("v", packed)))
            return
        invocation.return_error_literal(Gio.dbus_error_quark(), Gio.DBusError.UNKNOWN_METHOD, method)
