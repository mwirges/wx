# wx Linux shell — architecture

**Core:** Go `wx` CLI JSON. The same contract the Mac shell uses.

**Shell:** one Go binary, `build/wx-desk`, under `linux/`. The UI is Fyne. It decodes JSON for display. It does not call NWS, geocode, or cache weather.

**Boundary:** `cli.WxCLI` runs the `wx` binary (`WX_BINARY`, then `build/wx`, then `PATH`). Views bind `store.Store`.

**Surfaces:** a StatusNotifier tray item and a desk window. Both read the store. The desk has the daily surface, a dual pane (radar beside telemetry, stacked when narrower than 760 px), radar, CPC outlooks, storm chase, climate, tropics, and the favorites grid.

**Radar picture:** the CLI returns PNG bytes. The desk displays those bytes. Save PNG and Save GIF call `wx radar --save` and `wx radar --save-gif`. The shell does not composite frames.

**Desk close:** matches the Mac window (#35, #36, #37). Closing releases the window, stops the radar subprocess and the loop, and skips outlook, chase, and the favorites grid. The tray refresh keeps running. The next Open Desk builds a new window.

**Tray:** a StatusNotifier item on the session bus (`org.kde.StatusNotifierItem` and `com.canonical.dbusmenu`). Compact, standard, and tactical strings match the Mac menu bar. A warning is a steady red pip. A watch or advisory is a steady amber pip. There is no blink. A missing StatusNotifier host does not stop the window.

**Preferences:** `WX_CONFIG` when set, otherwise `~/.config/wx/config.json`. The shell preserves unknown keys. The Go CLI does not read `WX_CONFIG`. Tests set `WX_CONFIG` and do not read the user config.
