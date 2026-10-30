import SwiftUI

@main
struct WxIOSApp: App {
    @StateObject private var store = IOSStore()
    @Environment(\.scenePhase) private var scenePhase

    var body: some Scene {
        WindowGroup {
            IOSRootView()
                .environmentObject(store)
                .preferredColorScheme(.dark)
                .task {
                    WxBridge.prepareCacheDir()
                    await store.boot()
                }
                .onChange(of: scenePhase) { _, phase in
                    store.noteForeground(phase == .active)
                }
        }
    }
}
