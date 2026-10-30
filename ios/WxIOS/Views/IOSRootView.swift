import SwiftUI
import UIKit

struct IOSRootView: View {
    @EnvironmentObject private var store: IOSStore
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    /// Full-screen iPad is regular width. iPhone, and a compact iPad split, keep the phone tabs.
    private var showsSplitDesk: Bool {
        UIDevice.current.userInterfaceIdiom == .pad && horizontalSizeClass == .regular
    }

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

                Group {
                    if showsSplitDesk {
                        padContent
                    } else {
                        phoneTabs
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            tabBar
        }
        .onAppear { syncPane() }
        .onChange(of: horizontalSizeClass) { _, _ in
            syncPane()
        }
        .onChange(of: store.tab) { _, tab in
            syncPane()
            Task { await store.ensure(tab) }
        }
    }

    /// One custom bar. A TabView plus that bar draws a second, faded system
    /// bar on iPhone, and the extra tabs collapse into More.
    private var phoneTabs: some View {
        Group {
            switch store.tab {
            case .surface:
                SurfaceView()
            case .tactical:
                TacticalView()
            case .radar:
                RadarTabView()
            case .severe:
                SevereView()
            case .tropics:
                TropicsTabView()
            case .config:
                ConfigView()
            }
        }
    }

    @ViewBuilder
    private var padContent: some View {
        switch store.tab {
        case .surface, .radar:
            splitDesk
        case .tactical:
            TacticalView()
        case .severe:
            SevereView()
        case .tropics:
            TropicsTabView()
        case .config:
            ConfigView()
        }
    }

    /// Same idea as the Mac dual desk: telemetry on the left, radar filling the right.
    private var splitDesk: some View {
        GeometryReader { geo in
            let leftWidth = min(480, max(340, geo.size.width * 0.40))
            HStack(alignment: .top, spacing: 0) {
                VStack(alignment: .leading, spacing: 6) {
                    paneTitle("TELEMETRY", id: "desk.telemetry")
                    SurfaceView()
                        .frame(maxHeight: .infinity)
                }
                .frame(width: leftWidth, height: geo.size.height)

                Rectangle()
                    .fill(WxTheme.snwCyan.opacity(0.35))
                    .frame(width: 1)
                    .accessibilityHidden(true)

                VStack(alignment: .leading, spacing: 6) {
                    paneTitle("RADAR", id: "desk.radar")
                    RadarTabView()
                        .frame(maxHeight: .infinity)
                }
                .frame(maxWidth: .infinity)
                .frame(height: geo.size.height)
            }
            .accessibilityElement(children: .contain)
            .accessibilityIdentifier("desk.split")
        }
    }

    private func paneTitle(_ title: String, id: String) -> some View {
        Text(title)
            .font(.system(size: 11, weight: .bold, design: .monospaced))
            .foregroundStyle(WxTheme.snwCyan)
            .padding(.horizontal, 12)
            .padding(.top, 4)
            .accessibilityIdentifier(id)
    }

    private func syncPane() {
        let radarVisible = showsSplitDesk && (store.tab == .surface || store.tab == .radar)
        store.setRadarPaneOnScreen(radarVisible)
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
                    syncPane()
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
        .accessibilityElement(children: .contain)
        .accessibilityIdentifier("tabbar.custom")
        .overlay(alignment: .top) {
            Rectangle()
                .fill(WxTheme.snwCyan.opacity(0.25))
                .frame(height: 0.5)
        }
    }
}
