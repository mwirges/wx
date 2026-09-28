import SwiftUI

struct PopoverView: View {
    @EnvironmentObject var store: WeatherStore

    private var hasConditionAlerts: Bool {
        (store.payload?.alerts ?? []).contains { ConditionBand.from(severity: $0.severity) != nil }
    }

    /// Fit: alerts take extra space -> 4 periods when alerts present; else 5.
    private var periodLimit: Int { hasConditionAlerts ? 4 : 5 }

    private var updatedSubtitle: String? {
        store.lastRefreshed.map { $0.formatted(date: .omitted, time: .shortened) }
    }

    var body: some View {
        VStack(spacing: 0) {
            // Pinned Header
            VStack(spacing: 6) {
                // Bridge Status Bar
                HStack(alignment: .center) {
                    HStack(spacing: 5) {
                        Circle()
                            .fill(WxTheme.snwGreen)
                            .frame(width: 5, height: 5)
                            .shadow(color: WxTheme.snwGreen.opacity(0.9), radius: 3)
                        Text("METEOROLOGICAL SENSOR NETWORK // ACTIVE")
                            .font(.system(size: 8, weight: .bold, design: .monospaced))
                            .tracking(0.8)
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.9))
                    }
                    .padding(.horizontal, 7)
                    .padding(.vertical, 2.5)
                    .background(WxTheme.snwChassis.opacity(0.7), in: Capsule())
                    .overlay(Capsule().strokeBorder(WxTheme.border.opacity(0.3), lineWidth: 0.5))

                    Spacer()

                    if let loc = store.payload?.conditions?.location {
                        Text(loc.uppercased())
                            .font(.system(size: 8.5, weight: .semibold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan.opacity(0.8))
                            .lineLimit(1)
                    }
                }

                // Console Header Pill
                SNWHeaderPill(
                    title: "ATMOSPHERIC TELEMETRY",
                    subtitle: updatedSubtitle.map { "SYNC // \($0)" },
                    badge: "WX.HUD"
                )
            }
            .padding(.horizontal, 12)
            .padding(.top, 10)
            .padding(.bottom, 6)
            .background(
                LinearGradient(
                    colors: [WxTheme.snwChassis.opacity(0.95), WxTheme.bg.opacity(0.85)],
                    startPoint: .top,
                    endPoint: .bottom
                )
            )

            // Cyan Gradient Hairline
            Rectangle()
                .fill(
                    LinearGradient(
                        colors: [WxTheme.snwCyan.opacity(0.55), WxTheme.snwCyan.opacity(0.1), .clear],
                        startPoint: .leading,
                        endPoint: .trailing
                    )
                )
                .frame(height: 1)

            // Scrollable Console Cards Area
            ScrollView(showsIndicators: false) {
                VStack(alignment: .leading, spacing: 8) {
                    SNWConsoleCard(title: "Tactical Sensor Control", tag: "SYS.01") {
                        ControlsBar(showOpenWindow: true, compact: true)
                    }

                    if !store.gridCards.isEmpty {
                        SNWConsoleCard(title: "Command Grid // Pinned Stations", tag: "GRID.PINNED") {
                            PopoverGridPreviewBar()
                        }
                    }

                    if let shift = store.cpcPayload?.patternShift, shift.hasShift {
                        Button {
                            store.selectedDeskTab = .outlooks
                            NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                        } label: {
                            HStack(spacing: 6) {
                                Image(systemName: "exclamationmark.bubble.fill")
                                    .font(.system(size: 11))
                                    .foregroundStyle(WxTheme.snwAmber)
                                Text("// REGIME SHIFT: \(shift.summary)")
                                    .font(.system(size: 8.5, weight: .semibold, design: .monospaced))
                                    .foregroundStyle(WxTheme.text)
                                    .lineLimit(1)
                                Spacer()
                                Image(systemName: "chevron.right")
                                    .font(.system(size: 8))
                                    .foregroundStyle(WxTheme.snwCyan)
                            }
                            .padding(6)
                            .background(WxTheme.snwAmber.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                            .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwAmber.opacity(0.35), lineWidth: 0.8))
                        }
                        .buttonStyle(.plain)
                    }

                    if let chase = store.chasePayload, chase.totalClusters > 0 {
                        Button {
                            store.selectedDeskTab = .chase
                            NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                        } label: {
                            HStack(spacing: 6) {
                                Image(systemName: "bolt.shield.fill")
                                    .font(.system(size: 11))
                                    .foregroundStyle(WxTheme.snwAmber)
                                Text("// STORM CHASE: \(chase.totalClusters) ACTIVE CLUSTERS")
                                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.text)
                                    .lineLimit(1)
                                Spacer()
                                Image(systemName: "chevron.right")
                                    .font(.system(size: 8))
                                    .foregroundStyle(WxTheme.snwCyan)
                            }
                            .padding(6)
                            .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 4))
                            .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.35), lineWidth: 0.8))
                        }
                        .buttonStyle(.plain)
                    }

                    SNWConsoleCard(title: "Atmospheric Telemetry", tag: "GRID.OBS") {
                        NowBlockView(compact: true, popoverMetrics: true)
                    }

                    if let nc = store.nowcastPayload?.nowcast ?? store.payload?.conditions?.nowcast ?? store.payload?.nowcast,
                       (nc.isActivePrecip == true || nc.nextPrecipTime != nil || (nc.totalLiquidMm ?? 0) > 0.1) {
                        NowcastCardView(nowcast: nc)
                    }

                    if hasConditionAlerts {
                        SNWConsoleCard(title: "Tactical Alerts", tag: "NWS.WARN", statusColor: WxTheme.snwRed) {
                            AlertsListView(popoverMode: true)
                        }
                    }

                    SNWConsoleCard(title: "Synoptic Forecast Log", tag: "NOAA.NWS") {
                        PeriodsListView(limit: periodLimit, compactRows: true)
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
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
                    .font(.system(size: 8, weight: .semibold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
                Spacer()
                Text("SEC.04 // HUD")
                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwCyan.opacity(0.65))
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 6)
            .background(WxTheme.snwChassis.opacity(0.5))
        }
        .frame(width: WxTheme.popoverSize.width, height: WxTheme.popoverSize.height)
        .background(WarpGlassBackground())
        .clipShape(RoundedRectangle(cornerRadius: WxTheme.corner, style: .continuous))
        .preferredColorScheme(.dark)
    }
}

struct PopoverGridPreviewBar: View {
    @EnvironmentObject var store: WeatherStore

    private var isMetric: Bool {
        store.units == "metric"
    }

    var body: some View {
        VStack(spacing: 6) {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(store.gridCards) { card in
                        let isCurrent = (store.locationInput.caseInsensitiveCompare(card.locationKey) == .orderedSame) ||
                            (store.payload?.conditions?.location?.caseInsensitiveCompare(card.displayName) == .orderedSame)

                        let tempStr: String = {
                            guard let c = card.payload?.conditions else { return "--" }
                            let t = isMetric ? c.temperatureC : c.temperatureF
                            return t.map { String(format: "%.0f°", $0) } ?? "--"
                        }()

                        let sym = ConditionSymbol.systemName(for: card.payload?.conditions?.conditionCode)
                        let windStr: String = {
                            guard let c = card.payload?.conditions else { return "" }
                            let w = isMetric ? c.windKph.map { String(format: "%.0fkm/h", $0) } : c.windMph.map { String(format: "%.0fmph", $0) }
                            guard let w else { return "" }
                            let arrow = WeatherStore.windArrow(for: c.windDirection)
                            return "\(arrow)\(w)"
                        }()

                        let hasWarning = (card.payload?.alerts ?? []).contains { $0.isWarning }
                        let hasWatch = !hasWarning && (card.payload?.alerts ?? []).contains { $0.isWatch || $0.isAdvisory }

                        Button {
                            store.selectLocation(card.locationKey)
                        } label: {
                            VStack(alignment: .leading, spacing: 3) {
                                HStack(spacing: 4) {
                                    if isCurrent {
                                        Circle().fill(WxTheme.snwCyan).frame(width: 4.5, height: 4.5)
                                    }
                                    Text(card.displayName)
                                        .font(.system(size: 9.5, weight: isCurrent ? .bold : .semibold, design: .default))
                                        .foregroundStyle(isCurrent ? WxTheme.snwCyan : WxTheme.text)
                                        .lineLimit(1)
                                    Spacer(minLength: 0)
                                    if hasWarning {
                                        Circle().fill(WxTheme.snwRed).frame(width: 5, height: 5)
                                    } else if hasWatch {
                                        Circle().fill(WxTheme.snwAmber).frame(width: 5, height: 5)
                                    }
                                }

                                HStack(spacing: 5) {
                                    Image(systemName: sym)
                                        .font(.system(size: 11))
                                        .foregroundStyle(WxTheme.snwCyan)
                                    Text(tempStr)
                                        .font(.system(size: 11.5, weight: .bold, design: .monospaced))
                                        .foregroundStyle(WxTheme.text)
                                    if !windStr.isEmpty {
                                        Text(windStr)
                                            .font(.system(size: 8.5, design: .monospaced))
                                            .foregroundStyle(WxTheme.snwSilver)
                                    }
                                }
                            }
                            .padding(.horizontal, 8)
                            .padding(.vertical, 6)
                            .frame(width: 125, alignment: .leading)
                            .background(
                                isCurrent
                                    ? WxTheme.snwCyan.opacity(0.15)
                                    : WxTheme.snwChassis.opacity(0.85),
                                in: RoundedRectangle(cornerRadius: 5)
                            )
                            .overlay(
                                RoundedRectangle(cornerRadius: 5)
                                    .strokeBorder(
                                        isCurrent ? WxTheme.snwCyan.opacity(0.6) : WxTheme.border.opacity(0.4),
                                        lineWidth: isCurrent ? 1 : 0.6
                                    )
                            )
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.vertical, 2)
            }

            HStack {
                Text("\(store.gridCards.count) PINNED STATIONS ACTIVE")
                    .font(.system(size: 7.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
                Spacer()
                Button {
                    store.selectedDeskTab = .grid
                    NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                } label: {
                    HStack(spacing: 3) {
                        Text("COMMAND GRID")
                            .font(.system(size: 7.5, weight: .bold, design: .monospaced))
                        Image(systemName: "chevron.right")
                            .font(.system(size: 6.5))
                    }
                    .foregroundStyle(WxTheme.snwCyan)
                }
                .buttonStyle(.plain)
            }
        }
    }
}

