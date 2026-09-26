import AppKit
import SwiftUI

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    let store = WeatherStore()
    private var statusController: StatusItemController?
    private var deskWindow: NSWindow?
    private var popoverQAWindow: NSWindow?
    private var openObserver: NSObjectProtocol?
    private var openRadarObserver: NSObjectProtocol?

    func applicationDidFinishLaunching(_ notification: Notification) {
        // Full Mac app activation policy: displays system menu bar and Dock icon.
        NSApp.setActivationPolicy(.regular)
        DispatchQueue.main.async { [weak self] in
            self?.installMainMenu()
        }

        let status = StatusItemController(store: store)
        status.install()
        statusController = status

        openObserver = NotificationCenter.default.addObserver(
            forName: .wxOpenDeskWindow,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor in self?.showDeskWindow() }
        }

        openRadarObserver = NotificationCenter.default.addObserver(
            forName: .wxOpenDeskRadar,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor in
                self?.store.selectedDeskTab = .radar
                self?.showDeskWindow()
                if self?.store.radarImage == nil {
                    await self?.store.refreshRadar()
                }
            }
        }

        store.start()
        if ProcessInfo.processInfo.environment["WX_DESK_TAB"] == "radar" {
            store.selectedDeskTab = .radar
            Task { await store.refreshRadar() }
        }
        // Open desk window on launch as a full Mac app.
        showDeskWindow()
        if ProcessInfo.processInfo.environment["WX_QA_POPOVER"] == "1" {
            showPopoverQAWindow()
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        store.stop()
        statusController?.tearDown()
        if let openObserver {
            NotificationCenter.default.removeObserver(openObserver)
        }
        if let openRadarObserver {
            NotificationCenter.default.removeObserver(openRadarObserver)
        }
    }

    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        if !flag {
            showDeskWindow()
        }
        return true
    }

    // ── Full Standard macOS Main Menu Bar ──────────────────────────────────────────

    private func installMainMenu() {
        let mainMenu = NSMenu()

        // 1. wx Application Menu
        let appMenuItem = NSMenuItem()
        let appMenu = NSMenu(title: "wx")
        let aboutItem = NSMenuItem(title: "About wx", action: #selector(showAbout(_:)), keyEquivalent: "")
        aboutItem.target = self
        appMenu.addItem(aboutItem)
        appMenu.addItem(.separator())

        let settingsItem = NSMenuItem(title: "Settings…", action: #selector(openSettings(_:)), keyEquivalent: ",")
        settingsItem.target = self
        appMenu.addItem(settingsItem)
        appMenu.addItem(.separator())

        let servicesItem = NSMenuItem(title: "Services", action: nil, keyEquivalent: "")
        let servicesMenu = NSMenu(title: "Services")
        servicesItem.submenu = servicesMenu
        NSApp.servicesMenu = servicesMenu
        appMenu.addItem(servicesItem)
        appMenu.addItem(.separator())

        let hideItem = NSMenuItem(title: "Hide wx", action: #selector(NSApplication.hide(_:)), keyEquivalent: "h")
        appMenu.addItem(hideItem)

        let hideOthersItem = NSMenuItem(title: "Hide Others", action: #selector(NSApplication.hideOtherApplications(_:)), keyEquivalent: "h")
        hideOthersItem.keyEquivalentModifierMask = [.command, .option]
        appMenu.addItem(hideOthersItem)

        let showAllItem = NSMenuItem(title: "Show All", action: #selector(NSApplication.unhideAllApplications(_:)), keyEquivalent: "")
        appMenu.addItem(showAllItem)
        appMenu.addItem(.separator())

        let quitItem = NSMenuItem(title: "Quit wx", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        appMenu.addItem(quitItem)
        appMenuItem.submenu = appMenu
        mainMenu.addItem(appMenuItem)

        // 2. File Menu
        let fileMenuItem = NSMenuItem()
        let fileMenu = NSMenu(title: "File")
        let openDeskItem = NSMenuItem(title: "Open Desk Window", action: #selector(showDeskWindowAction(_:)), keyEquivalent: "n")
        openDeskItem.target = self
        fileMenu.addItem(openDeskItem)

        let refreshItem = NSMenuItem(title: "Refresh Telemetry", action: #selector(refreshAction(_:)), keyEquivalent: "r")
        refreshItem.target = self
        fileMenu.addItem(refreshItem)
        fileMenu.addItem(.separator())

        let closeItem = NSMenuItem(title: "Close Window", action: #selector(NSWindow.performClose(_:)), keyEquivalent: "w")
        fileMenu.addItem(closeItem)
        fileMenuItem.submenu = fileMenu
        mainMenu.addItem(fileMenuItem)

        // 3. Edit Menu (Standard Responder shortcuts for text fields)
        let editMenuItem = NSMenuItem()
        let editMenu = NSMenu(title: "Edit")
        editMenu.addItem(withTitle: "Undo", action: Selector(("undo:")), keyEquivalent: "z")
        let redoItem = NSMenuItem(title: "Redo", action: Selector(("redo:")), keyEquivalent: "Z")
        editMenu.addItem(redoItem)
        editMenu.addItem(.separator())
        editMenu.addItem(withTitle: "Cut", action: #selector(NSText.cut(_:)), keyEquivalent: "x")
        editMenu.addItem(withTitle: "Copy", action: #selector(NSText.copy(_:)), keyEquivalent: "c")
        editMenu.addItem(withTitle: "Paste", action: #selector(NSText.paste(_:)), keyEquivalent: "v")
        editMenu.addItem(withTitle: "Select All", action: #selector(NSText.selectAll(_:)), keyEquivalent: "a")
        editMenuItem.submenu = editMenu
        mainMenu.addItem(editMenuItem)

        // 4. View Menu
        let viewMenuItem = NSMenuItem()
        let viewMenu = NSMenu(title: "View")
        let dualModeItem = NSMenuItem(title: "Command Console (Dual)", action: #selector(selectDualTab(_:)), keyEquivalent: "1")
        dualModeItem.target = self
        viewMenu.addItem(dualModeItem)

        let weatherModeItem = NSMenuItem(title: "Surface Telemetry", action: #selector(selectWeatherTab(_:)), keyEquivalent: "2")
        weatherModeItem.target = self
        viewMenu.addItem(weatherModeItem)

        let radarModeItem = NSMenuItem(title: "Radar Tactical Display", action: #selector(selectRadarTab(_:)), keyEquivalent: "3")
        radarModeItem.target = self
        viewMenu.addItem(radarModeItem)
        viewMenu.addItem(.separator())

        let imperialItem = NSMenuItem(title: "Units: Imperial (°F)", action: #selector(setUnitsImperial(_:)), keyEquivalent: "")
        imperialItem.target = self
        viewMenu.addItem(imperialItem)

        let metricItem = NSMenuItem(title: "Units: Metric (°C)", action: #selector(setUnitsMetric(_:)), keyEquivalent: "")
        metricItem.target = self
        viewMenu.addItem(metricItem)
        viewMenu.addItem(.separator())

        let recenterItem = NSMenuItem(title: "Recenter Radar Array", action: #selector(recenterRadarAction(_:)), keyEquivalent: "l")
        recenterItem.target = self
        viewMenu.addItem(recenterItem)
        viewMenuItem.submenu = viewMenu
        mainMenu.addItem(viewMenuItem)

        // 5. Window Menu
        let windowMenuItem = NSMenuItem()
        let windowMenu = NSMenu(title: "Window")
        let minItem = NSMenuItem(title: "Minimize", action: #selector(NSWindow.performMiniaturize(_:)), keyEquivalent: "m")
        windowMenu.addItem(minItem)
        let zoomItem = NSMenuItem(title: "Zoom", action: #selector(NSWindow.performZoom(_:)), keyEquivalent: "")
        windowMenu.addItem(zoomItem)
        windowMenu.addItem(.separator())
        let frontItem = NSMenuItem(title: "Bring All to Front", action: #selector(NSApplication.arrangeInFront(_:)), keyEquivalent: "")
        windowMenu.addItem(frontItem)
        windowMenuItem.submenu = windowMenu
        mainMenu.addItem(windowMenuItem)
        NSApp.windowsMenu = windowMenu

        // 6. Help Menu
        let helpMenuItem = NSMenuItem()
        let helpMenu = NSMenu(title: "Help")
        let helpDocs = NSMenuItem(title: "NWS Meteorological Documentation", action: #selector(openNWSHelp(_:)), keyEquivalent: "")
        helpDocs.target = self
        helpMenu.addItem(helpDocs)
        helpMenuItem.submenu = helpMenu
        mainMenu.addItem(helpMenuItem)
        NSApp.helpMenu = helpMenu

        NSApp.mainMenu = mainMenu
    }

    // ── Menu Action Handlers ──────────────────────────────────────────────────────

    @objc private func showAbout(_ sender: Any?) {
        let alert = NSAlert()
        alert.messageText = "wx — Advanced Meteorological Console"
        alert.informativeText = "High-Resolution MRMS Doppler Radar & Surface Telemetry\nNational Weather Service (NWS) & NOAA MRMS Array"
        alert.alertStyle = .informational
        alert.addButton(withTitle: "Acknowledge")
        alert.runModal()
    }

    @objc private func openSettings(_ sender: Any?) {
        showDeskWindow()
    }

    @objc private func showDeskWindowAction(_ sender: Any?) {
        showDeskWindow()
    }

    @objc private func refreshAction(_ sender: Any?) {
        Task {
            await store.refresh()
            await store.refreshRadar()
        }
    }

    @objc private func selectDualTab(_ sender: Any?) {
        store.selectedDeskTab = .dual
        showDeskWindow()
        if store.radarImage == nil && !store.isRadarLoading {
            Task { await store.refreshRadar() }
        }
    }

    @objc private func selectWeatherTab(_ sender: Any?) {
        store.selectedDeskTab = .weather
        showDeskWindow()
    }

    @objc private func selectRadarTab(_ sender: Any?) {
        store.selectedDeskTab = .radar
        showDeskWindow()
        if store.radarImage == nil && !store.isRadarLoading {
            Task { await store.refreshRadar() }
        }
    }

    @objc private func setUnitsImperial(_ sender: Any?) {
        store.units = "imperial"
        Task { await store.applyLocationAndUnits() }
    }

    @objc private func setUnitsMetric(_ sender: Any?) {
        store.units = "metric"
        Task { await store.applyLocationAndUnits() }
    }

    @objc private func recenterRadarAction(_ sender: Any?) {
        store.selectedDeskTab = .radar
        showDeskWindow()
        NotificationCenter.default.post(name: .wxOpenDeskRadar, object: nil)
    }

    @objc private func openNWSHelp(_ sender: Any?) {
        if let url = URL(string: "https://www.weather.gov/documentation/services-web-api") {
            NSWorkspace.shared.open(url)
        }
    }

    // ── Window Management ─────────────────────────────────────────────────────────

    func showDeskWindow() {
        NSApp.setActivationPolicy(.regular)
        if let deskWindow {
            deskWindow.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
            return
        }
        let hosting = NSHostingController(rootView: DeskWindowView().environmentObject(store))
        hosting.preferredContentSize = NSSize(width: 940, height: 760)
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 940, height: 760),
            styleMask: [.titled, .closable, .miniaturizable, .resizable, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )
        window.minSize = NSSize(width: 520, height: 600)
        window.title = "wx"
        window.titleVisibility = .hidden
        window.titlebarAppearsTransparent = true
        window.isMovableByWindowBackground = true
        window.backgroundColor = .clear
        window.contentViewController = hosting
        window.setContentSize(NSSize(width: 940, height: 760))
        window.center()
        window.isReleasedWhenClosed = false
        window.makeKeyAndOrderFront(nil)
        deskWindow = window
        NSApp.activate(ignoringOtherApps: true)

        if store.selectedDeskTab == .dual || store.selectedDeskTab == .radar {
            if store.radarImage == nil && !store.isRadarLoading {
                Task { await store.refreshRadar() }
            }
        }
    }

    /// Env-gated (`WX_QA_POPOVER=1`) fixed 420×620 surface matching menu popover — for QA screenshots only.
    func showPopoverQAWindow() {
        if let popoverQAWindow {
            popoverQAWindow.makeKeyAndOrderFront(nil)
            return
        }
        let hosting = NSHostingController(rootView: PopoverView().environmentObject(store))
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 420, height: 620),
            styleMask: [.titled, .closable],
            backing: .buffered,
            defer: false
        )
        window.title = "wx Popover QA"
        window.contentViewController = hosting
        window.setContentSize(NSSize(width: 420, height: 620))
        window.center()
        window.isReleasedWhenClosed = false
        window.makeKeyAndOrderFront(nil)
        popoverQAWindow = window
    }
}
