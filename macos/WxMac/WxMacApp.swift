import SwiftUI

@main
struct WxMacApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    var body: some Scene {
        // Desk and Settings are AppDelegate NSWindows. This Settings scene does not
        // present on the status-item shell (issue #81). ⌘, is the wx menu item.
        Settings {
            MCPSettingsView()
        }
    }
}
