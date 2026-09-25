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
                        // Left Pane: Controls, Surface Conditions, Alerts, Synoptic Forecast
                        ScrollView {
                            VStack(alignment: .leading, spacing: 12) {
                                SNWConsoleCard(title: "Tactical Sensor Control", tag: "SYS.01") {
                                    ControlsBar(showOpenWindow: false, compact: false)
                                }

                                SNWConsoleCard(title: "Atmospheric Telemetry", tag: "GRID.OBS") {
                                    NowBlockView(compact: false, popoverMetrics: false)
                                }

                                AlertsListView(popoverMode: false)

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
                                    RadarPanelView()
                                }
                            }
                            .padding(.vertical, 12)
                            .padding(.leading, 2)
                            .padding(.trailing, 14)
                        }
                        .frame(maxWidth: .infinity)
                    }
                } else {
                    // Single Column Responsive Layout
                    ScrollView {
                        VStack(alignment: .leading, spacing: 12) {
                            SNWConsoleCard(title: "Tactical Sensor Control", tag: "SYS.01") {
                                ControlsBar(showOpenWindow: false, compact: false)
                            }

                            if store.selectedDeskTab == .radar {
                                SNWConsoleCard(title: "Doppler Radar Array", tag: "NOAA.MRMS") {
                                    RadarPanelView()
                                }
                            } else {
                                SNWConsoleCard(title: "Atmospheric Telemetry", tag: "GRID.OBS") {
                                    NowBlockView(compact: false, popoverMetrics: false)
                                }

                                AlertsListView(popoverMode: false)

                                SNWConsoleCard(title: "Synoptic Forecast Log", tag: "NOAA.NWS") {
                                    PeriodsListView(limit: nil, compactRows: false)
                                }

                                if store.selectedDeskTab == .dual {
                                    SNWConsoleCard(title: "Doppler Radar Array", tag: "NOAA.MRMS") {
                                        RadarPanelView()
                                    }
                                }
                            }
                        }
                        .padding(14)
                    }
                }
            }
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
    private var modeSelector: some View {
        HStack(spacing: 8) {
            modeButton(tab: .dual, label: "DUAL CONSOLE", icon: "rectangle.split.2x1.fill")
            modeButton(tab: .weather, label: "SURFACE SENSORS", icon: "thermometer.sun.fill")
            modeButton(tab: .radar, label: "DOPPLER RADAR", icon: "dot.radiowaves.left.and.right")
        }
    }

    private func modeButton(tab: DeskTab, label: String, icon: String) -> some View {
        Button {
            store.selectedDeskTab = tab
            if (tab == .dual || tab == .radar) && store.radarImage == nil && !store.isRadarLoading {
                Task { await store.refreshRadar() }
            }
        } label: {
            HStack(spacing: 5) {
                Image(systemName: icon)
                    .font(.system(size: 10.5))
                Text(label)
                    .font(.system(size: 9.5, weight: .bold, design: .monospaced))
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
