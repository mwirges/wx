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
        NSApp.setActivationPolicy(.accessory)
        installMainMenu()

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
        // Open desk window on first launch so both surfaces are exercised day one.
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


    /// Minimal main menu so ⌘Q works for an accessory / LSUIElement app.
    private func installMainMenu() {
        let mainMenu = NSMenu()
        let appMenuItem = NSMenuItem()
        mainMenu.addItem(appMenuItem)
        let appMenu = NSMenu(title: "wx")
        let quit = NSMenuItem(
            title: "Quit wx",
            action: #selector(NSApplication.terminate(_:)),
            keyEquivalent: "q"
        )
        quit.target = NSApp
        appMenu.addItem(quit)
        appMenuItem.submenu = appMenu
        NSApp.mainMenu = mainMenu
    }

    func showDeskWindow() {
        if let deskWindow {
            deskWindow.makeKeyAndOrderFront(nil)
            NSApp.activate()
            return
        }
        let hosting = NSHostingController(rootView: DeskWindowView().environmentObject(store))
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 440, height: 690),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "wx"
        window.contentViewController = hosting
        window.center()
        window.isReleasedWhenClosed = false
        window.makeKeyAndOrderFront(nil)
        deskWindow = window
        NSApp.activate()
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

