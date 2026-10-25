import SwiftUI

struct IOSRootView: View {
    @EnvironmentObject private var store: IOSStore

    var body: some View {
        ZStack {
            WxTheme.snwChassis.ignoresSafeArea()
            VStack(spacing: 8) {
                SNWHeaderPill(
                    title: store.place.isEmpty ? "No place" : store.place,
                    subtitle: store.isLoading ? "Sync" : store.refreshedLabel,
                    badge: "WX.IOS"
                )
                .padding(.horizontal, 12)
                .padding(.top, 4)

                if let note = store.locationNote, !note.isEmpty {
                    Text(note)
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(WxTheme.snwGold)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, 16)
                        .accessibilityIdentifier("location.note")
                }

                if store.showsPowerProbe {
                    powerProbe
                }
                if store.showsLiveProbe {
                    liveProbe
                }

                TabView(selection: $store.tab) {
                    SurfaceView()
                        .tag(IOSTab.surface)
                        .tabItem { Label("Surface", systemImage: IOSTab.surface.symbol) }
                    TacticalView()
                        .tag(IOSTab.tactical)
                        .tabItem { Label("Tactical", systemImage: IOSTab.tactical.symbol) }
                    RadarTabView()
                        .tag(IOSTab.radar)
                        .tabItem { Label("Radar", systemImage: IOSTab.radar.symbol) }
                    SevereView()
                        .tag(IOSTab.severe)
                        .tabItem { Label("Severe", systemImage: IOSTab.severe.symbol) }
                    TropicsTabView()
                        .tag(IOSTab.tropics)
                        .tabItem { Label("Tropics", systemImage: IOSTab.tropics.symbol) }
                    ConfigView()
                        .tag(IOSTab.config)
                        .tabItem { Label("Config", systemImage: IOSTab.config.symbol) }
                }
                .tint(WxTheme.snwCyan)
                .toolbar(.hidden, for: .tabBar)
            }
        }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            tabBar
        }
        .onChange(of: store.tab) { _, tab in
            store.syncRadarLoop()
            Task { await store.ensure(tab) }
        }
    }

    private var powerProbe: some View {
        VStack(alignment: .leading, spacing: 2) {
            HStack(spacing: 8) {
                probe("power.loopTicks", "\(store.loopTicks)")
                probe("power.loopState", store.loopPlaying ? "playing" : "paused")
                probe("power.frameCount", "\(store.radarPictures.count)")
                probe("power.fetchCount", "\(store.refreshCount)")
                probe("power.ticksAtPause", "\(store.ticksAtPause)")
                probe("power.ticksWhilePaused", "\(store.ticksWhilePaused)")
                probe("power.pauseCount", "\(store.pauseCount)")
            }
            .accessibilityElement(children: .contain)
            Button("Advance 6 min") {
                store.advanceTestClock()
            }
            .font(.system(size: 11, weight: .semibold, design: .monospaced))
            .foregroundStyle(WxTheme.snwGold)
            .accessibilityIdentifier("power.clock.advance")
        }
        .accessibilityElement(children: .contain)
        .padding(.horizontal, 12)
    }

    private var liveProbe: some View {
        VStack(alignment: .leading, spacing: 2) {
            HStack(spacing: 8) {
                probe("live.status", store.liveStatus)
                probe("live.id", store.liveActivityID)
            }
            HStack(spacing: 12) {
                Button("Start onset") {
                    Task { await store.injectLiveOnset() }
                }
                .accessibilityIdentifier("live.inject.onset")
                Button("End dry") {
                    Task { await store.injectLiveDry() }
                }
                .accessibilityIdentifier("live.inject.dry")
            }
            .font(.system(size: 11, weight: .semibold, design: .monospaced))
            .foregroundStyle(WxTheme.snwGold)
            .buttonStyle(.plain)
        }
        .accessibilityElement(children: .contain)
        .padding(.horizontal, 12)
    }

    private func probe(_ id: String, _ value: String) -> some View {
        Text(value)
            .font(.system(size: 9, weight: .medium, design: .monospaced))
            .foregroundStyle(WxTheme.textSecondary)
            .accessibilityIdentifier(id)
            .accessibilityLabel(value)
    }

    private var tabBar: some View {
        HStack(spacing: 0) {
            ForEach(IOSTab.allCases) { tab in
                Button {
                    store.tab = tab
                    store.syncRadarLoop()
                } label: {
                    VStack(spacing: 2) {
                        Image(systemName: tab.symbol)
                            .font(.system(size: 15, weight: .semibold))
                        Text(tab.rawValue)
                            .font(.system(size: 9, weight: .semibold))
                            .lineLimit(1)
                            .minimumScaleFactor(0.7)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                    .foregroundStyle(store.tab == tab ? WxTheme.snwCyan : WxTheme.textSecondary)
                }
                .buttonStyle(.plain)
                .accessibilityIdentifier("tab.\(tab.rawValue)")
                .accessibilityLabel(tab.rawValue)
                .accessibilityAddTraits(store.tab == tab ? .isSelected : [])
            }
        }
        .background(WxTheme.snwPanel.ignoresSafeArea(edges: .bottom))
        .overlay(alignment: .top) {
            Rectangle()
                .fill(WxTheme.snwCyan.opacity(0.25))
                .frame(height: 0.5)
        }
    }
}
