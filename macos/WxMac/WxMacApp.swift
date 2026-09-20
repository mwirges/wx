import SwiftUI

@main
struct WxMacApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    var body: some Scene {
        // Desk window is owned by AppDelegate (NSWindow) so LSUIElement + status item stay primary.
        Settings {
            EmptyView()
        }
    }
}
