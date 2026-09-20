import AppKit
import SwiftUI

@MainActor
final class StatusItemController: NSObject {
    private var statusItem: NSStatusItem?
    private var popover: NSPopover?
    private var statusMenu: NSMenu?
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
            button.target = self
            button.action = #selector(statusItemClick(_:))
            button.sendAction(on: [.leftMouseUp, .rightMouseUp])
        }

        statusMenu = buildMenu()

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
        statusMenu = nil
    }

    private func buildMenu() -> NSMenu {
        let menu = NSMenu()
        let openDesk = NSMenuItem(
            title: "Open Desk",
            action: #selector(openDesk(_:)),
            keyEquivalent: ""
        )
        openDesk.target = self
        menu.addItem(openDesk)
        menu.addItem(.separator())
        let quit = NSMenuItem(
            title: "Quit wx",
            action: #selector(quitWx(_:)),
            keyEquivalent: "q"
        )
        quit.keyEquivalentModifierMask = [.command]
        quit.target = self
        menu.addItem(quit)
        return menu
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

    @objc private func statusItemClick(_ sender: Any?) {
        guard let event = NSApp.currentEvent else {
            togglePopover(sender)
            return
        }
        switch event.type {
        case .rightMouseUp:
            showStatusMenu()
        default:
            togglePopover(sender)
        }
    }

    private func showStatusMenu() {
        guard let button = statusItem?.button, let menu = statusMenu else { return }
        popover?.performClose(nil)
        // Pop up under the status item without assigning statusItem.menu (that would steal left-click).
        let point = NSPoint(x: button.bounds.midX, y: button.bounds.maxY + 2)
        menu.popUp(positioning: nil, at: point, in: button)
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

    @objc private func openDesk(_ sender: Any?) {
        popover?.performClose(nil)
        NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
    }

    @objc private func quitWx(_ sender: Any?) {
        NSApp.terminate(nil)
    }
}
