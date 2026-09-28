import SwiftUI

struct PopoverView: View {
    @EnvironmentObject var store: WeatherStore

    private var hasConditionAlerts: Bool {
        (store.payload?.alerts ?? []).contains { ConditionBand.from(severity: $0.severity) != nil }
    }

    /// Fit: alerts take extra space -> 3 periods when alerts present; else 4.
    private var periodLimit: Int { hasConditionAlerts ? 3 : 4 }

    private var updatedSubtitle: String? {
        store.lastRefreshed.map { $0.formatted(date: .omitted, time: .shortened) }
    }

    private var currentSize: CGSize {
        WxTheme.popoverSize(for: store.menuBarFormat)
    }

    var body: some View {
        VStack(spacing: 0) {
            switch store.menuBarFormat {
            case .compact:
                compactHUD
            case .standard:
                standardHUD
            case .tactical:
                tacticalHUD
            }
        }
        .frame(width: currentSize.width, height: currentSize.height)
        .background(WarpGlassBackground())
        .clipShape(RoundedRectangle(cornerRadius: WxTheme.corner, style: .continuous))
        .preferredColorScheme(.dark)
    }

    // MARK: - Mode 1: Compact HUD (350 x 320, glanceable, zero scrolling)

    private var compactHUD: some View {
        let c = store.payload?.conditions
        let isMetric = store.units == "metric"
        let tempStr: String = {
            let t = isMetric ? c?.temperatureC : c?.temperatureF
            return t.map { String(format: "%.0f°", $0) } ?? "--"
        }()
        let feelsStr: String? = {
            let f = isMetric ? c?.feelsLikeC : c?.feelsLikeF
            return f.map { String(format: "%.0f°", $0) }
        }()
        let sym = ConditionSymbol.systemName(for: c?.conditionCode)

        return VStack(spacing: 0) {
            // Header bar
            HStack(spacing: 6) {
                HStack(spacing: 4) {
                    Circle()
                        .fill(WxTheme.snwGreen)
                        .frame(width: 5, height: 5)
                        .shadow(color: WxTheme.snwGreen.opacity(0.8), radius: 3)
                    Text("WX.HUD")
                        .font(.system(size: 8.5, weight: .black, design: .monospaced))
                        .foregroundStyle(WxTheme.snwGold)
                }
                .padding(.horizontal, 6)
                .padding(.vertical, 2.5)
                .background(WxTheme.snwChassis.opacity(0.85), in: Capsule())
                .overlay(Capsule().strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.6))

                if let loc = c?.location {
                    Text(loc.uppercased())
                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.text)
                        .lineLimit(1)
                }

                Spacer(minLength: 4)

                hudModePicker(compact: true)
            }
            .padding(.horizontal, 10)
            .padding(.top, 8)
            .padding(.bottom, 6)
            .background(WxTheme.snwChassis.opacity(0.9))

            cyanHairline

            // Compact Observation Body
            VStack(spacing: 8) {
                // Condition Hero Row
                HStack(spacing: 10) {
                    Image(systemName: sym)
                        .font(.system(size: 30))
                        .foregroundStyle(WxTheme.snwCyan)
                        .frame(width: 36)

                    VStack(alignment: .leading, spacing: 1) {
                        HStack(alignment: .firstTextBaseline, spacing: 6) {
                            Text(tempStr)
                                .font(.system(size: 26, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.text)

                            if let feels = feelsStr {
                                Text("FEELS \(feels)")
                                    .font(.system(size: 9, weight: .semibold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwSilver.opacity(0.8))
                            }
                        }

                        if let desc = c?.description, !desc.isEmpty {
                            Text(desc)
                                .font(.system(size: 10.5, weight: .medium))
                                .foregroundStyle(WxTheme.textSecondary)
                                .lineLimit(1)
                        }
                    }

                    Spacer(minLength: 0)

                    // Quick Refresh Button
                    Button {
                        Task { await store.refresh() }
                    } label: {
                        if store.isLoading {
                            ProgressView()
                                .controlSize(.small)
                                .frame(width: 13, height: 13)
                        } else {
                            Image(systemName: "arrow.triangle.2.circlepath")
                                .font(.system(size: 9.5, weight: .bold))
                                .foregroundStyle(WxTheme.snwCyan)
                        }
                    }
                    .buttonStyle(.plain)
                    .padding(5)
                    .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))
                    .help("Refresh (⌘R)")
                    .disabled(store.isLoading)
                }
                .padding(.horizontal, 10)
                .padding(.top, 4)

                // 4 Essential Metric Chips in a 4-col flexible grid
                LazyVGrid(columns: [GridItem(.flexible(), spacing: 5), GridItem(.flexible(), spacing: 5), GridItem(.flexible(), spacing: 5), GridItem(.flexible(), spacing: 5)], spacing: 5) {
                    if let h = c?.humidityPct {
                        SNWMetricTile(label: "Humidity", value: String(format: "%.0f%%", h))
                    }
                    if isMetric {
                        if let dp = c?.dewPointC {
                            SNWMetricTile(label: "Dew Pt", value: String(format: "%.0f°", dp))
                        }
                        if let w = c?.windKph {
                            let dir = c?.windDirection.map { " \($0)" } ?? ""
                            SNWMetricTile(label: "Wind", value: String(format: "%.0fkm/h%@", w, dir))
                        }
                    } else {
                        if let dp = c?.dewPointF {
                            SNWMetricTile(label: "Dew Pt", value: String(format: "%.0f°", dp))
                        }
                        if let w = c?.windMph {
                            let dir = c?.windDirection.map { " \($0)" } ?? ""
                            SNWMetricTile(label: "Wind", value: String(format: "%.0fmph%@", w, dir))
                        }
                    }
                    if let astro = c?.astronomy ?? store.payload?.astronomy,
                       let sr = astro.sunriseFormatted {
                        SNWMetricTile(label: "Sunrise", value: sr)
                    } else if let aq = c?.airQuality ?? store.payload?.airQuality, let aqi = aq.aqi {
                        SNWMetricTile(label: "AQI", value: "\(aqi)")
                    }
                }
                .padding(.horizontal, 10)

                // 3 Compact Forecast Rows
                PeriodsListView(limit: 3, compactRows: true)
                    .padding(.horizontal, 10)
            }

            Spacer(minLength: 0)

            cyanHairline

            // Compact Action Bar
            HStack(spacing: 8) {
                Picker("Units", selection: $store.units) {
                    Text("°F").tag("imperial")
                    Text("°C").tag("metric")
                }
                .pickerStyle(.segmented)
                .labelsHidden()
                .frame(width: 58)
                .onChange(of: store.units) { _, _ in
                    Task { await store.applyLocationAndUnits() }
                }

                Spacer(minLength: 4)

                Button {
                    NotificationCenter.default.post(name: .wxOpenDeskRadar, object: nil)
                } label: {
                    HStack(spacing: 3) {
                        Image(systemName: "dot.radiowaves.left.and.right")
                            .font(.system(size: 8))
                        Text("RADAR")
                            .font(.system(size: 8, weight: .bold, design: .monospaced))
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 4)
                    .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                    .foregroundStyle(WxTheme.snwCyan)
                }
                .buttonStyle(.plain)
                .help("Open Radar (⌘3)")

                Button {
                    NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                } label: {
                    HStack(spacing: 3) {
                        Image(systemName: "macwindow")
                            .font(.system(size: 8))
                        Text("DESK")
                            .font(.system(size: 8, weight: .bold, design: .monospaced))
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 4)
                    .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                    .foregroundStyle(WxTheme.snwCyan)
                }
                .buttonStyle(.plain)
                .help("Open Desk Console (⌘1)")
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 5.5)
            .background(WxTheme.snwChassis.opacity(0.6))
        }
    }

    // MARK: - Mode 2: Standard HUD (380 x 460, balanced telemetry & forecast)

    private var standardHUD: some View {
        VStack(spacing: 0) {
            // Pinned Header
            VStack(spacing: 5) {
                HStack(alignment: .center) {
                    HStack(spacing: 5) {
                        Circle()
                            .fill(WxTheme.snwGreen)
                            .frame(width: 5, height: 5)
                            .shadow(color: WxTheme.snwGreen.opacity(0.9), radius: 3)
                        Text("SENSOR NETWORK // ACTIVE")
                            .font(.system(size: 7.5, weight: .bold, design: .monospaced))
                            .tracking(0.7)
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.9))
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2.5)
                    .background(WxTheme.snwChassis.opacity(0.7), in: Capsule())
                    .overlay(Capsule().strokeBorder(WxTheme.border.opacity(0.3), lineWidth: 0.5))

                    Spacer()

                    hudModePicker(compact: true)
                }

                SNWHeaderPill(
                    title: "ATMOSPHERIC TELEMETRY",
                    subtitle: updatedSubtitle.map { "SYNC // \($0)" },
                    badge: "WX.HUD"
                )
            }
            .padding(.horizontal, 10)
            .padding(.top, 8)
            .padding(.bottom, 6)
            .background(
                LinearGradient(
                    colors: [WxTheme.snwChassis.opacity(0.95), WxTheme.bg.opacity(0.85)],
                    startPoint: .top,
                    endPoint: .bottom
                )
            )

            cyanHairline

            // Scrollable Console Cards Area (constrained horizontally)
            ScrollView(showsIndicators: false) {
                VStack(alignment: .leading, spacing: 8) {
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
                .padding(.horizontal, 10)
                .padding(.vertical, 8)
                .frame(maxWidth: .infinity)
            }

            cyanHairline

            // Pinned Footer with controls
            HStack(spacing: 8) {
                Picker("Units", selection: $store.units) {
                    Text("°F").tag("imperial")
                    Text("°C").tag("metric")
                }
                .pickerStyle(.segmented)
                .labelsHidden()
                .frame(width: 58)
                .onChange(of: store.units) { _, _ in
                    Task { await store.applyLocationAndUnits() }
                }

                Button {
                    Task { await store.refresh() }
                } label: {
                    if store.isLoading {
                        ProgressView()
                            .controlSize(.small)
                            .frame(width: 13, height: 13)
                    } else {
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 9.5, weight: .bold))
                            .foregroundStyle(WxTheme.snwCyan)
                    }
                }
                .buttonStyle(.plain)
                .padding(.horizontal, 6)
                .padding(.vertical, 4)
                .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 4))
                .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))
                .help("Refresh (⌘R)")
                .disabled(store.isLoading)

                Spacer()

                Button {
                    NotificationCenter.default.post(name: .wxOpenDeskRadar, object: nil)
                } label: {
                    HStack(spacing: 3) {
                        Image(systemName: "dot.radiowaves.left.and.right")
                            .font(.system(size: 8))
                        Text("RADAR")
                            .font(.system(size: 8, weight: .bold, design: .monospaced))
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 4)
                    .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                    .foregroundStyle(WxTheme.snwCyan)
                }
                .buttonStyle(.plain)
                .help("Open Radar (⌘3)")

                Button {
                    NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                } label: {
                    HStack(spacing: 3) {
                        Image(systemName: "macwindow")
                            .font(.system(size: 8))
                        Text("DESK")
                            .font(.system(size: 8, weight: .bold, design: .monospaced))
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 4)
                    .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                    .foregroundStyle(WxTheme.snwCyan)
                }
                .buttonStyle(.plain)
                .help("Open Desk Console (⌘1)")
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(WxTheme.snwChassis.opacity(0.6))
        }
    }

    // MARK: - Mode 3: Tactical HUD (410 x 540, full tactical command console)

    private var tacticalHUD: some View {
        VStack(spacing: 0) {
            // Pinned Header
            VStack(spacing: 6) {
                HStack(alignment: .center) {
                    HStack(spacing: 5) {
                        Circle()
                            .fill(WxTheme.snwGreen)
                            .frame(width: 5, height: 5)
                            .shadow(color: WxTheme.snwGreen.opacity(0.9), radius: 3)
                        Text("TACTICAL SENSOR NETWORK // ACTIVE")
                            .font(.system(size: 8, weight: .bold, design: .monospaced))
                            .tracking(0.8)
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.9))
                    }
                    .padding(.horizontal, 7)
                    .padding(.vertical, 2.5)
                    .background(WxTheme.snwChassis.opacity(0.7), in: Capsule())
                    .overlay(Capsule().strokeBorder(WxTheme.border.opacity(0.3), lineWidth: 0.5))

                    Spacer()

                    hudModePicker(compact: true)
                }

                SNWHeaderPill(
                    title: "ATMOSPHERIC TELEMETRY",
                    subtitle: updatedSubtitle.map { "SYNC // \($0)" },
                    badge: "WX.TAC"
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

            cyanHairline

            // Scrollable Console Cards Area (constrained horizontally)
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
                .padding(.horizontal, 10)
                .padding(.vertical, 8)
                .frame(maxWidth: .infinity)
            }

            cyanHairline

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
    }

    // MARK: - Shared Helpers

    private var cyanHairline: some View {
        Rectangle()
            .fill(
                LinearGradient(
                    colors: [WxTheme.snwCyan.opacity(0.45), WxTheme.snwCyan.opacity(0.1), .clear],
                    startPoint: .leading,
                    endPoint: .trailing
                )
            )
            .frame(height: 1)
    }

    private func hudModePicker(compact: Bool) -> some View {
        Menu {
            Text("HUD / MENU BAR MODE")
                .font(.system(size: 9, weight: .bold, design: .monospaced))
            Divider()
            ForEach(MenuBarFormat.allCases) { fmt in
                Button {
                    store.setMenuBarFormat(fmt)
                } label: {
                    HStack {
                        Text(fmt.displayName)
                        if store.menuBarFormat == fmt {
                            Image(systemName: "checkmark")
                        }
                    }
                }
            }
        } label: {
            HStack(spacing: 3) {
                Image(systemName: "menubar.arrow.up.rectangle")
                    .font(.system(size: 8))
                    .foregroundStyle(WxTheme.snwCyan)
                Text(compact ? "HUD:" : "MODE:")
                    .font(.system(size: 7.5, weight: .semibold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver.opacity(0.8))
                Text(store.menuBarFormat.rawValue.uppercased())
                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwCyan)
                Image(systemName: "chevron.down")
                    .font(.system(size: 6.5, weight: .bold))
                    .foregroundStyle(WxTheme.snwSilver.opacity(0.6))
            }
            .padding(.horizontal, 5)
            .padding(.vertical, 3)
            .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 4))
            .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))
            .foregroundStyle(WxTheme.snwSilver)
        }
        .menuStyle(.borderlessButton)
        .fixedSize()
        .help("Configure HUD & Menu Bar Mode (Compact, Standard, Tactical)")
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

