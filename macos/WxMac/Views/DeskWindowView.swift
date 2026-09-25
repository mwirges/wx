import SwiftUI

struct DeskWindowView: View {
    @EnvironmentObject var store: WeatherStore

    private var updatedSubtitle: String? {
        store.lastRefreshed.map { $0.formatted(date: .omitted, time: .shortened) }
    }

    var body: some View {
        VStack(spacing: 0) {
            // Window Traffic Light Safe Space & Starfleet Bridge Header
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
                        Text("STARFLEET COMMAND // SENSORS ONLINE")
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
                    title: "METEOROLOGICAL TELEMETRY ARRAY",
                    subtitle: updatedSubtitle.map { "SYNC // \($0)" }
                )
                .padding(.horizontal, 16)
            }
            .padding(.bottom, 8)
            .background(
                LinearGradient(
                    colors: [WxTheme.snwChassis.opacity(0.95), WxTheme.bg.opacity(0.8)],
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

            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    // Tactical Controls & Location
                    SNWConsoleCard(title: "Tactical Sensor Control", tag: "SYS.01") {
                        ControlsBar(showOpenWindow: false, compact: false)
                    }

                    // Futuristic Bridge Mode Selector
                    HStack(spacing: 8) {
                        Button {
                            store.selectedDeskTab = .weather
                        } label: {
                            HStack(spacing: 6) {
                                Image(systemName: "thermometer.sun.fill")
                                    .font(.system(size: 11))
                                Text("SURFACE SENSORS")
                                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 7)
                            .background(
                                RoundedRectangle(cornerRadius: 6, style: .continuous)
                                    .fill(store.selectedDeskTab == .weather ? WxTheme.snwCyan.opacity(0.2) : WxTheme.snwChassis.opacity(0.6))
                            )
                            .overlay(
                                RoundedRectangle(cornerRadius: 6, style: .continuous)
                                    .strokeBorder(
                                        store.selectedDeskTab == .weather ? WxTheme.snwCyan : WxTheme.border.opacity(0.25),
                                        lineWidth: store.selectedDeskTab == .weather ? 1.2 : 0.6
                                    )
                            )
                            .foregroundStyle(store.selectedDeskTab == .weather ? WxTheme.text : WxTheme.textSecondary)
                        }
                        .buttonStyle(.plain)

                        Button {
                            store.selectedDeskTab = .radar
                            if store.radarImage == nil && !store.isRadarLoading {
                                Task { await store.refreshRadar() }
                            }
                        } label: {
                            HStack(spacing: 6) {
                                Image(systemName: "dot.radiowaves.left.and.right")
                                    .font(.system(size: 11))
                                Text("DOPPLER RADAR ARRAY")
                                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 7)
                            .background(
                                RoundedRectangle(cornerRadius: 6, style: .continuous)
                                    .fill(store.selectedDeskTab == .radar ? WxTheme.snwCyan.opacity(0.2) : WxTheme.snwChassis.opacity(0.6))
                            )
                            .overlay(
                                RoundedRectangle(cornerRadius: 6, style: .continuous)
                                    .strokeBorder(
                                        store.selectedDeskTab == .radar ? WxTheme.snwCyan : WxTheme.border.opacity(0.25),
                                        lineWidth: store.selectedDeskTab == .radar ? 1.2 : 0.6
                                    )
                            )
                            .foregroundStyle(store.selectedDeskTab == .radar ? WxTheme.text : WxTheme.textSecondary)
                        }
                        .buttonStyle(.plain)
                    }

                    if store.selectedDeskTab == .weather {
                        SNWConsoleCard(title: "Atmospheric Telemetry", tag: "GRID.OBS") {
                            NowBlockView(compact: false, popoverMetrics: false)
                        }

                        AlertsListView(popoverMode: false)

                        SNWConsoleCard(title: "Synoptic Forecast Log", tag: "NOAA.NWS") {
                            PeriodsListView(limit: nil, compactRows: false)
                        }
                    } else {
                        SNWConsoleCard(title: "Orbital Doppler Array", tag: "MRMS.1KM") {
                            RadarPanelView()
                        }
                    }
                }
                .padding(14)
            }
        }
        .frame(minWidth: 440, minHeight: 680)
        .background(WarpGlassBackground())
        .preferredColorScheme(.dark)
    }
}
