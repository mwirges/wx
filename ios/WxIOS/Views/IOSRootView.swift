import SwiftUI
import UIKit

struct IOSRootView: View {
    @EnvironmentObject private var store: IOSStore
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    /// Full-screen iPad is regular width. iPhone, and a compact iPad split, keep the phone tabs.
    private var showsSplitDesk: Bool {
        UIDevice.current.userInterfaceIdiom == .pad && horizontalSizeClass == .regular
    }

    private let primaryDockTabs: [WxFloatingTabItem] = [
        WxFloatingTabItem(id: IOSTab.surface.rawValue, symbol: IOSTab.surface.symbol),
        WxFloatingTabItem(id: IOSTab.tactical.rawValue, symbol: IOSTab.tactical.symbol),
        WxFloatingTabItem(id: IOSTab.radar.rawValue, symbol: IOSTab.radar.symbol),
        WxFloatingTabItem(id: IOSTab.severe.rawValue, symbol: IOSTab.severe.symbol),
        WxFloatingTabItem(id: IOSTab.tropics.rawValue, symbol: IOSTab.tropics.symbol),
    ]

    private let configOrbTab = WxFloatingTabItem(
        id: IOSTab.config.rawValue,
        symbol: IOSTab.config.symbol
    )

    private var tabSelectionBinding: Binding<String> {
        Binding(
            get: { store.tab.rawValue },
            set: { raw in
                if let next = IOSTab(rawValue: raw) {
                    store.tab = next
                    syncPane()
                }
            }
        )
    }

    var body: some View {
        ZStack(alignment: .bottom) {
            WxAtmosphereCanvas(regime: store.atmosphereRegime)

            VStack(spacing: 6) {
                floatingTopCapsule
                    .padding(.horizontal, 12)
                    .padding(.top, 4)

                if let note = store.locationNote, !note.isEmpty {
                    HStack(spacing: 6) {
                        Image(systemName: "location.slash.fill")
                            .font(.system(size: 11, weight: .semibold))
                        Text(note)
                            .font(.system(size: 12, weight: .medium))
                    }
                    .foregroundStyle(WxGlassTheme.gold)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 5)
                    .wxFrostedCapsule(accent: WxGlassTheme.gold)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 12)
                    .accessibilityElement(children: .contain)
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

            WxFloatingTabDock(
                primaryTabs: primaryDockTabs,
                orbTab: configOrbTab,
                selection: tabSelectionBinding
            )
            .frame(maxWidth: showsSplitDesk ? 520 : .infinity)
        }
        .preferredColorScheme(.dark)
        .onAppear { syncPane() }
        .onChange(of: horizontalSizeClass) { _, _ in
            syncPane()
        }
        .onChange(of: store.tab) { _, tab in
            syncPane()
            Task { await store.ensure(tab) }
        }
    }

    // MARK: - Floating Location & Sync Bridge Capsule

    private var floatingTopCapsule: some View {
        let resolvedLocation = store.payload?.conditions?.location?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let placeTitle = !resolvedLocation.isEmpty ? resolvedLocation : (store.place.isEmpty ? "No place" : store.place)
        let station = store.payload?.conditions?.station
        let isFav = store.isFavorite(store.place)

        return HStack(spacing: 8) {
            Button {
                store.tab = .config
                syncPane()
            } label: {
                HStack(spacing: 7) {
                    Circle()
                        .fill(store.surfaceError == nil ? WxGlassTheme.nominalGreen : WxGlassTheme.conditionRed)
                        .frame(width: 7, height: 7)
                        .shadow(
                            color: (store.surfaceError == nil ? WxGlassTheme.nominalGreen : WxGlassTheme.conditionRed).opacity(0.85),
                            radius: 3
                        )

                    Text(placeTitle)
                        .font(.system(size: 14, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                        .lineLimit(1)

                    if let station, !station.isEmpty {
                        Text(station.uppercased())
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxGlassTheme.cyanSoft)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(WxGlassTheme.cyan.opacity(0.14), in: Capsule())
                            .fixedSize()
                    }
                }
            }
            .buttonStyle(.plain)

            Spacer(minLength: 4)

            if !store.place.isEmpty || !store.favorites.isEmpty {
                if store.favorites.isEmpty {
                    Button {
                        store.toggleFavorite(store.place)
                    } label: {
                        Image(systemName: isFav ? "star.fill" : "star")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(isFav ? WxGlassTheme.gold : WxGlassTheme.textSecondary)
                            .frame(width: 26, height: 26)
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(isFav ? "Unpin station" : "Pin station")
                } else {
                    Menu {
                        if !store.place.isEmpty {
                            Button {
                                store.toggleFavorite(store.place)
                            } label: {
                                Label(
                                    isFav ? "Unpin \(placeTitle)" : "Pin \(placeTitle)",
                                    systemImage: isFav ? "star.slash" : "star"
                                )
                            }
                        }
                        Section("Pinned Stations") {
                            ForEach(store.favorites) { fav in
                                Button {
                                    store.save(place: fav.value, units: store.units)
                                } label: {
                                    Label(fav.name, systemImage: "star.fill")
                                }
                            }
                        }
                    } label: {
                        Image(systemName: isFav ? "star.fill" : "star")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(isFav ? WxGlassTheme.gold : WxGlassTheme.textSecondary)
                            .frame(width: 26, height: 26)
                    }
                }
            }

            Button {
                Task { await store.refreshSurface() }
            } label: {
                HStack(spacing: 5) {
                    if store.isLoading {
                        ProgressView()
                            .controlSize(.mini)
                            .tint(WxGlassTheme.cyan)
                    } else {
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 10.5, weight: .bold))
                            .foregroundStyle(WxGlassTheme.cyan)
                    }
                    Text(store.isLoading ? "Sync" : store.refreshedLabel)
                        .font(.system(size: 10.5, weight: .semibold, design: .monospaced))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                        .lineLimit(1)
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(Color.white.opacity(0.06), in: Capsule())
                .fixedSize()
            }
            .buttonStyle(.plain)
            .disabled(store.isLoading)
        }
        .padding(.leading, 12)
        .padding(.trailing, 8)
        .padding(.vertical, 6)
        .wxFrostedCapsule(accent: WxGlassTheme.cyan)
        .shadow(color: Color.black.opacity(0.28), radius: 12, y: 5)
    }

    // MARK: - Adaptive Phone & iPad Split Content

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

    /// Regular-width iPad Split Console: Telemetry on the left, live interactive Radar on the right.
    private var splitDesk: some View {
        GeometryReader { geo in
            let leftWidth = min(480, max(340, geo.size.width * 0.41))
            HStack(alignment: .top, spacing: 12) {
                VStack(alignment: .leading, spacing: 6) {
                    paneBadge("TELEMETRY", symbol: "thermometer.sun.fill", id: "desk.telemetry")
                    SurfaceView()
                        .frame(maxHeight: .infinity)
                }
                .frame(width: leftWidth, height: geo.size.height)

                Capsule()
                    .fill(
                        LinearGradient(
                            colors: [.clear, WxGlassTheme.cyan.opacity(0.38), .clear],
                            startPoint: .top,
                            endPoint: .bottom
                        )
                    )
                    .frame(width: 1.5)
                    .padding(.vertical, 12)
                    .accessibilityHidden(true)

                VStack(alignment: .leading, spacing: 6) {
                    paneBadge("DOPPLER RADAR", symbol: "dot.radiowaves.left.and.right", id: "desk.radar")
                    RadarTabView()
                        .frame(maxHeight: .infinity)
                }
                .frame(maxWidth: .infinity)
                .frame(height: geo.size.height)
            }
            .padding(.horizontal, 8)
            .accessibilityElement(children: .contain)
            .accessibilityIdentifier("desk.split")
        }
    }

    private func paneBadge(_ title: String, symbol: String, id: String) -> some View {
        HStack(spacing: 6) {
            Image(systemName: symbol)
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(WxGlassTheme.cyan)
            Text(title)
                .font(.system(size: 11, weight: .bold, design: .rounded))
                .tracking(0.6)
                .foregroundStyle(WxGlassTheme.textSecondary)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 4)
        .wxFrostedCapsule(accent: WxGlassTheme.cyan)
        .padding(.horizontal, 12)
        .padding(.top, 2)
        .accessibilityIdentifier(id)
    }

    private func syncPane() {
        let radarVisible = showsSplitDesk && (store.tab == .surface || store.tab == .radar)
        store.setRadarPaneOnScreen(radarVisible)
    }

    // MARK: - XCUITest Diagnostic Probes (Env-Gated Only)

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
            .foregroundStyle(WxGlassTheme.gold)
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
            .foregroundStyle(WxGlassTheme.gold)
            .buttonStyle(.plain)
        }
        .accessibilityElement(children: .contain)
        .padding(.horizontal, 12)
    }

    private func probe(_ id: String, _ value: String) -> some View {
        Text(value)
            .font(.system(size: 9, weight: .medium, design: .monospaced))
            .foregroundStyle(WxGlassTheme.textSecondary)
            .accessibilityIdentifier(id)
            .accessibilityLabel(value)
    }
}
