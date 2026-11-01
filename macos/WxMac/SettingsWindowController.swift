import AppKit
import SwiftUI

/// Dedicated Settings window. SwiftUI `Settings` + `showSettingsWindow:` does not
/// present `MCPSettingsView` on this AppDelegate shell (issue #81).
@MainActor
final class SettingsWindowController: NSWindowController {
    convenience init() {
        let hosting = NSHostingController(rootView: MCPSettingsView())
        hosting.sizingOptions = []
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 580, height: 640),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Settings"
        window.identifier = NSUserInterfaceItemIdentifier("wx-settings-window")
        window.contentViewController = hosting
        window.minSize = NSSize(width: 540, height: 480)
        window.setContentSize(NSSize(width: 580, height: 640))
        window.setFrameAutosaveName("wxSettingsWindow")
        window.center()
        self.init(window: window)
        // Closed window stays owned here so the next explicit open can order it front.
        window.isReleasedWhenClosed = false
    }

    /// Orders Settings in front. Callers must be an explicit user action
    /// (status-menu Settings…, the wx Settings menu item, or ⌘,).
    /// Do not call this from refresh, alerts, MCP restart, or location paths.
    func presentForUserAction() {
        guard let window else { return }
        NSApp.activate(ignoringOtherApps: true)
        window.makeKeyAndOrderFront(nil)
    }
}
