import SwiftUI

struct DeskWindowView: View {
    @EnvironmentObject var store: WeatherStore

    private var updatedSubtitle: String? {
        store.lastRefreshed.map { $0.formatted(date: .omitted, time: .shortened) }
    }

    var body: some View {
        VStack(spacing: 0) {
            // Window Traffic Light Safe Space & Console Bridge Header
            VStack(spacing: 8) {
                HStack(alignment: .center) {
                    // Traffic lights clear space
                    Color.clear
                        .frame(width: 68, height: 16)

                    if let loc = store.payload?.conditions?.location {
                        Text(loc.uppercased())
                            .font(.system(size: 8.5, weight: .semibold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan.opacity(0.8))
                            .lineLimit(1)
                    }

                    Spacer()

                    HStack(spacing: 5) {
                        Circle()
                            .fill(WxTheme.snwGreen)
                            .frame(width: 5, height: 5)
                            .shadow(color: WxTheme.snwGreen.opacity(0.9), radius: 3)
                        Text("METEOROLOGICAL SENSOR NETWORK // ACTIVE")
                            .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                            .tracking(0.8)
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.9))
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(WxTheme.snwChassis.opacity(0.7), in: Capsule())
                    .overlay(Capsule().strokeBorder(WxTheme.border.opacity(0.3), lineWidth: 0.5))
                }
                .padding(.top, 12)
                .padding(.horizontal, 16)

                SNWHeaderPill(
                    title: "ATMOSPHERIC TELEMETRY CONSOLE",
                    subtitle: updatedSubtitle.map { "SYNC // \($0)" },
                    badge: "WX.DESK"
                )
                .padding(.horizontal, 16)

                modeSelector
                    .padding(.horizontal, 16)
                    .padding(.top, 2)
            }
            .padding(.bottom, 10)
            .background(
                LinearGradient(
                    colors: [WxTheme.snwChassis.opacity(0.95), WxTheme.bg.opacity(0.85)],
                    startPoint: .top,
                    endPoint: .bottom
                )
            )

            Rectangle()
                .fill(
                    LinearGradient(
                        colors: [WxTheme.snwCyan.opacity(0.55), WxTheme.snwCyan.opacity(0.1), .clear],
                        startPoint: .leading,
                        endPoint: .trailing
                    )
                )
                .frame(height: 1)

            GeometryReader { geo in
                let isWide = geo.size.width >= 760
                if store.selectedDeskTab == .dual && isWide {
                    // Wide Dual-Pane Command Console
                    let leftWidth = max(380, min(440, geo.size.width * 0.44))
                    HStack(alignment: .top, spacing: 14) {
                        // Left Pane: Controls, Surface Conditions (Atmospheric Telemetry), Alerts, Signals, Synoptic Forecast
                        ScrollView {
                            VStack(alignment: .leading, spacing: 12) {
                                SNWConsoleCard(title: "Tactical Sensor Control", tag: "SYS.01") {
                                    ControlsBar(showOpenWindow: false, compact: false)
                                }

                                SNWConsoleCard(title: "Atmospheric Telemetry", tag: "GRID.OBS") {
                                    NowBlockView(compact: false, popoverMetrics: false)
                                }

                                airQualityCard

                                AlertsListView(popoverMode: false)

                                tacticalSignalsCard

                                SNWConsoleCard(title: "Synoptic Forecast Log", tag: "NOAA.NWS") {
                                    PeriodsListView(limit: nil, compactRows: false)
                                }
                            }
                            .padding(.vertical, 12)
                            .padding(.leading, 14)
                            .padding(.trailing, 2)
                        }
                        .frame(width: leftWidth)

                        // Right Pane: Live Doppler Radar Array
                        ScrollView {
                            VStack(alignment: .leading, spacing: 12) {
                                SNWConsoleCard(title: "Doppler Radar Array", tag: "NOAA.MRMS") {
                                    RadarPanelView(fullScreen: false)
                                }
                            }
                            .padding(.vertical, 12)
                            .padding(.leading, 2)
                            .padding(.trailing, 14)
                        }
                        .frame(maxWidth: .infinity)
                    }
                } else if store.selectedDeskTab == .radar {
                    // Full Screen Edge-to-Edge Doppler Radar Array
                    RadarPanelView(fullScreen: true)
                        .frame(width: geo.size.width, height: geo.size.height)
                } else if store.selectedDeskTab == .outlooks {
                    // CPC Climate Prediction Center Long-Range Outlooks
                    CPCOutlookView()
                        .frame(width: geo.size.width, height: geo.size.height)
                } else if store.selectedDeskTab == .chase {
                    // Remote Storm Chasing and Active Alert Clusters
                    StormChaseView()
                        .frame(width: geo.size.width, height: geo.size.height)
                } else if store.selectedDeskTab == .climate {
                    // Climate & Historical Observations Archive
                    ClimateTrendsView()
                        .frame(width: geo.size.width, height: geo.size.height)
                } else if store.selectedDeskTab == .grid {
                    // Multi-Location Command Grid Matrix
                    CommandGridView()
                        .frame(width: geo.size.width, height: geo.size.height)
                } else {
                    // Single Column Responsive Layout (.weather or narrow .dual)
                    ScrollView {
                        VStack(alignment: .leading, spacing: 12) {
                            SNWConsoleCard(title: "Tactical Sensor Control", tag: "SYS.01") {
                                ControlsBar(showOpenWindow: false, compact: false)
                            }

                            SNWConsoleCard(title: "Atmospheric Telemetry", tag: "GRID.OBS") {
                                NowBlockView(compact: false, popoverMetrics: false)
                            }

                            airQualityCard

                            AlertsListView(popoverMode: false)

                            tacticalSignalsCard

                            SNWConsoleCard(title: "Synoptic Forecast Log", tag: "NOAA.NWS") {
                                PeriodsListView(limit: nil, compactRows: false)
                            }

                            if store.selectedDeskTab == .dual {
                                SNWConsoleCard(title: "Doppler Radar Array", tag: "NOAA.MRMS") {
                                    RadarPanelView(fullScreen: false)
                                }
                            }
                        }
                        .padding(14)
                    }
                }
            }

            // Cyan Gradient Hairline
            Rectangle()
                .fill(
                    LinearGradient(
                        colors: [WxTheme.snwCyan.opacity(0.1), WxTheme.snwCyan.opacity(0.45), .clear],
                        startPoint: .leading,
                        endPoint: .trailing
                    )
                )
                .frame(height: 1)

            // Pinned Technical Console Footer
            HStack {
                Text("NOAA.NWS SENSOR ARRAY · GRID 1KM")
                    .font(.system(size: 8.5, weight: .semibold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
                Spacer()
                Text("SEC.01 // TACTICAL")
                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwCyan.opacity(0.65))
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 6)
            .background(WxTheme.snwChassis.opacity(0.5))
        }
        .frame(minWidth: 520, idealWidth: 940, maxWidth: .infinity, minHeight: 600, idealHeight: 760, maxHeight: .infinity)
        .background(WarpGlassBackground())
        .preferredColorScheme(.dark)
        .onAppear {
            if (store.selectedDeskTab == .dual || store.selectedDeskTab == .radar) && store.radarImage == nil && !store.isRadarLoading {
                Task { await store.refreshRadar() }
            }
        }
    }

    @ViewBuilder
    private var tacticalSignalsCard: some View {
        let hasShift = store.cpcPayload?.patternShift.hasShift == true
        let hasChase = (store.chasePayload?.totalClusters ?? 0) > 0

        if hasShift || hasChase {
            SNWConsoleCard(title: "Synoptic Signals & Active Hazards", tag: "NOAA.INTEL", statusColor: WxTheme.snwAmber) {
                VStack(alignment: .leading, spacing: 9) {
                    if let shift = store.cpcPayload?.patternShift, shift.hasShift {
                        HStack(alignment: .top, spacing: 8) {
                            Image(systemName: "exclamationmark.bubble.fill")
                                .font(.system(size: 13))
                                .foregroundStyle(WxTheme.snwAmber)
                            VStack(alignment: .leading, spacing: 3) {
                                HStack {
                                    Text("REGIME SHIFT:")
                                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                        .foregroundStyle(WxTheme.snwAmber)
                                    Spacer()
                                    Button {
                                        store.selectedDeskTab = .outlooks
                                    } label: {
                                        HStack(spacing: 3) {
                                            Text("VIEW OUTLOOKS")
                                            Image(systemName: "arrow.right")
                                        }
                                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                                        .foregroundStyle(WxTheme.snwCyan)
                                    }
                                    .buttonStyle(.plain)
                                }
                                Text(shift.summary)
                                    .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                                    .foregroundStyle(WxTheme.text)
                                    .fixedSize(horizontal: false, vertical: true)
                            }
                        }
                    }

                    if hasShift && hasChase {
                        Divider().overlay(WxTheme.border.opacity(0.3))
                    }

                    if let chase = store.chasePayload, chase.totalClusters > 0 {
                        HStack(alignment: .top, spacing: 8) {
                            Image(systemName: "bolt.shield.fill")
                                .font(.system(size: 13))
                                .foregroundStyle(WxTheme.snwAmber)
                            VStack(alignment: .leading, spacing: 3) {
                                HStack {
                                    Text("STORM CHASE // \(chase.totalClusters) ACTIVE CLUSTERS")
                                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                        .foregroundStyle(WxTheme.snwAmber)
                                    Spacer()
                                    Button {
                                        store.selectedDeskTab = .chase
                                    } label: {
                                        HStack(spacing: 3) {
                                            Text("OPEN CHASE")
                                            Image(systemName: "arrow.right")
                                        }
                                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                                        .foregroundStyle(WxTheme.snwCyan)
                                    }
                                    .buttonStyle(.plain)
                                }
                                if let top = chase.clusters.first {
                                    Text("Top: \(top.name) (\(top.totalAlerts) warnings, score: \(top.score))")
                                        .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                                        .foregroundStyle(WxTheme.snwSilver)
                                        .lineLimit(1)
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var airQualityCard: some View {
        if let aq = store.payload?.conditions?.airQuality ?? store.payload?.airQuality {
            AirQualityCardView(airQuality: aq)
        }
    }

    @ViewBuilder
    private var modeSelector: some View {
        HStack(spacing: 6) {
            modeButton(tab: .dual, label: "TACTICAL", icon: "rectangle.split.2x1.fill")
            modeButton(tab: .weather, label: "SURFACE", icon: "thermometer.sun.fill")
            modeButton(tab: .radar, label: "RADAR", icon: "dot.radiowaves.left.and.right")
            modeButton(tab: .outlooks, label: "CPC OUTLOOKS", icon: "chart.line.uptrend.xyaxis")
            modeButton(tab: .chase, label: "STORM CHASE", icon: "bolt.shield.fill")
            modeButton(tab: .climate, label: "CLIMATE", icon: "calendar.day.timeline.left")
            modeButton(tab: .grid, label: "GRID", icon: "square.grid.2x2.fill")
        }
    }

    private func modeButton(tab: DeskTab, label: String, icon: String) -> some View {
        Button {
            store.selectedDeskTab = tab
            if (tab == .dual || tab == .radar) && store.radarImage == nil && !store.isRadarLoading {
                Task { await store.refreshRadar() }
            }
            if tab == .outlooks && store.cpcPayload == nil && !store.isCPCLoading {
                Task { await store.refreshCPC() }
            }
            if tab == .chase && store.chasePayload == nil && !store.isChaseLoading {
                Task { await store.refreshChase() }
            }
            if tab == .climate && store.historyPayload == nil && !store.isHistoryLoading {
                Task { await store.refreshHistory() }
            }
            if tab == .grid && store.gridCards.isEmpty && !store.isGridLoading {
                Task { await store.refreshGrid() }
            }
        } label: {
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.system(size: 9.5))
                Text(label)
                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 6)
            .background(
                RoundedRectangle(cornerRadius: 6, style: .continuous)
                    .fill(store.selectedDeskTab == tab ? WxTheme.snwCyan.opacity(0.22) : WxTheme.snwChassis.opacity(0.6))
            )
            .overlay(
                RoundedRectangle(cornerRadius: 6, style: .continuous)
                    .strokeBorder(
                        store.selectedDeskTab == tab ? WxTheme.snwCyan : WxTheme.border.opacity(0.25),
                        lineWidth: store.selectedDeskTab == tab ? 1.2 : 0.6
                    )
            )
            .foregroundStyle(store.selectedDeskTab == tab ? WxTheme.text : WxTheme.textSecondary)
        }
        .buttonStyle(.plain)
    }
}
