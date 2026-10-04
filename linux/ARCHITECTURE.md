# wx Linux shell — architecture

**Core:** Go `wx` CLI JSON. The same contract the Mac shell uses.

**Shell:** GTK 4 and libadwaita, under `linux/wxdesk`. It decodes JSON for display. It does not call NWS, geocode, or cache weather.

**Boundary:** `WxCLI` runs the `wx` binary (`WX_BINARY`, then `build/wx`, then `PATH`). Views bind `WeatherStore`.

**Surfaces:** a StatusNotifier tray item and a desk window. Both read the store.

**Desk close:** matches the Mac window (#35, #36, #37). Closing releases the window, stops the radar subprocess and the loop, and skips outlook, chase, and the favorites grid. The menu-bar refresh keeps running. The next Open Desk builds a new window.

**Tray:** compact, standard, and tactical strings match the Mac menu bar. A warning is a steady red pip. A watch or advisory is a steady amber pip. There is no blink.

**Preferences:** `WX_CONFIG` when set, otherwise `~/.config/wx/config.json`. The shell preserves unknown keys. The Go CLI does not read `WX_CONFIG`.
