import AppKit
import SwiftUI

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    let store = WeatherStore()
    private var statusController: StatusItemController?
    private var deskWindow: NSWindow?
    private var openObserver: NSObjectProtocol?

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)

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

        store.start()
        // Open desk window on first launch so both surfaces are exercised day one.
        showDeskWindow()
    }

    func applicationWillTerminate(_ notification: Notification) {
        store.stop()
        statusController?.tearDown()
        if let openObserver {
            NotificationCenter.default.removeObserver(openObserver)
        }
    }

    func showDeskWindow() {
        if let deskWindow {
            deskWindow.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
            return
        }
        let hosting = NSHostingController(rootView: DeskWindowView().environmentObject(store))
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 480, height: 640),
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
        NSApp.activate(ignoringOtherApps: true)
    }
}
