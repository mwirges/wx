import AppKit
import SwiftUI

@MainActor
final class StatusItemController: NSObject {
    private var statusItem: NSStatusItem?
    private var popover: NSPopover?
    private let store: WeatherStore
    private var updateObserver: NSObjectProtocol?

    init(store: WeatherStore) {
        self.store = store
        super.init()
    }

    func install() {
        let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        statusItem = item
        if let button = item.button {
            button.image = NSImage(systemSymbolName: store.statusSymbol, accessibilityDescription: "wx")
            button.imagePosition = .imageLeading
            button.title = " \(store.displayTemp)"
            button.action = #selector(togglePopover(_:))
            button.target = self
        }

        let pop = NSPopover()
        pop.behavior = .transient
        pop.contentSize = NSSize(width: 420, height: 620)
        pop.contentViewController = NSHostingController(rootView: PopoverView().environmentObject(store))
        popover = pop

        updateObserver = NotificationCenter.default.addObserver(
            forName: .wxWeatherDidUpdate,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor in self?.refreshButton() }
        }
        refreshButton()
    }

    func tearDown() {
        if let updateObserver {
            NotificationCenter.default.removeObserver(updateObserver)
        }
        popover?.performClose(nil)
        if let statusItem {
            NSStatusBar.system.removeStatusItem(statusItem)
        }
        statusItem = nil
        popover = nil
    }

    private func refreshButton() {
        guard let button = statusItem?.button else { return }
        button.image = NSImage(systemSymbolName: store.statusSymbol, accessibilityDescription: "wx")
        if store.errorMessage != nil && store.payload?.conditions == nil {
            button.title = " wx?"
        } else {
            button.title = " \(store.displayTemp)"
        }
    }

    @objc private func togglePopover(_ sender: Any?) {
        guard let button = statusItem?.button, let popover else { return }
        if popover.isShown {
            popover.performClose(sender)
        } else {
            popover.show(relativeTo: button.bounds, of: button, preferredEdge: .minY)
            NSApp.activate()
        }
    }
}
