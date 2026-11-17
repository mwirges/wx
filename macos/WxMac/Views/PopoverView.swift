import SwiftUI

struct PopoverView: View {
    @EnvironmentObject var store: WeatherStore

    private var hasConditionAlerts: Bool {
        (store.payload?.alerts ?? []).contains { ConditionBand.from(severity: $0.severity) != nil }
    }

    private var periodLimit: Int { hasConditionAlerts ? 3 : 4 }

    private var updatedSubtitle: String? {
        store.lastRefreshed.map { $0.formatted(date: .omitted, time: .shortened) }
    }

    private var currentSize: CGSize {
        WxTheme.popoverSize(for: store.menuBarFormat)
    }

    private var currentRegime: WxAtmosphereRegime {
        WxAtmosphereRegime.resolve(
            conditions: store.payload?.conditions,
            alerts: store.activeAlerts,
            astronomy: store.payload?.conditions?.astronomy ?? store.payload?.astronomy,
            nowcast: store.nowcastPayload?.nowcast ?? store.payload?.conditions?.nowcast ?? store.payload?.nowcast,
            forecast: store.payload?.forecast
        )
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
        .accessibilityIdentifier("wx.hud")
        .background(WarpGlassBackground(regime: currentRegime))
        .clipShape(RoundedRectangle(cornerRadius: 20, style: .continuous))
        .preferredColorScheme(.dark)
    }

    // MARK: - Mode 1: Compact HUD (350 × 320, glanceable, zero scrolling)

    private var compactHUD: some View {
        let c = store.payload?.conditions
        let isMetric = store.units == "metric"
        let tempStr = store.displayTemp
        let feelsStr: String? = {
            if isMetric, let fc = c?.feelsLikeC {
                let r = fc.rounded()
                return String(format: "%.0f°", r == 0 ? 0 : r)
            }
            if !isMetric, let ff = c?.feelsLikeF {
                let r = ff.rounded()
                return String(format: "%.0f°", r == 0 ? 0 : r)
            }
            return nil
        }()
        let sym = store.statusSymbol
        let symTint = ConditionSymbol.tint(for: sym, isNight: store.isNight)
        let desc = store.resolvedConditionDescription
        let compactPeriodCap = store.activeAlerts.isEmpty ? 3 : 2

        return VStack(spacing: 0) {
            // Frosted Top Capsule Bar
            HStack(spacing: 8) {
                HStack(spacing: 5) {
                    Circle()
                        .fill(WxGlassTheme.nominalGreen)
                        .frame(width: 5.5, height: 5.5)
                        .shadow(color: WxGlassTheme.nominalGreen.opacity(0.8), radius: 3)
                    Text(c?.location ?? "wx HUD")
                        .font(.system(size: 11.5, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                        .lineLimit(1)
                }

                Spacer(minLength: 4)

                hudModePicker
            }
            .padding(.horizontal, 12)
            .padding(.top, 9)
            .padding(.bottom, 6)

            horizonDivider

            ScrollView(showsIndicators: false) {
                VStack(spacing: 7) {
                    if let leadAlert = store.activeAlerts.first {
                        let isWarn = leadAlert.isWarning
                        let tint = isWarn ? WxGlassTheme.conditionRed : WxGlassTheme.amber
                        Button {
                            NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                        } label: {
                            HStack(spacing: 6) {
                                Image(systemName: isWarn ? "exclamationmark.triangle.fill" : "bolt.shield.fill")
                                    .font(.system(size: 10, weight: .bold))
                                Text(leadAlert.event)
                                    .font(.system(size: 10.5, weight: .bold, design: .rounded))
                                    .lineLimit(1)
                                Spacer(minLength: 4)
                                Image(systemName: "chevron.right")
                                    .font(.system(size: 8.5, weight: .bold))
                            }
                            .foregroundStyle(isWarn ? Color.white : tint)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 4.5)
                            .background(tint.opacity(isWarn ? 0.82 : 0.16), in: Capsule())
                            .overlay(Capsule().strokeBorder(isWarn ? Color.white.opacity(0.3) : tint.opacity(0.55), lineWidth: 0.8))
                        }
                        .buttonStyle(.plain)
                        .padding(.horizontal, 12)
                    }

                    HStack(spacing: 10) {
                        Image(systemName: sym)
                            .font(.system(size: 32, weight: .medium))
                            .symbolRenderingMode(.hierarchical)
                            .foregroundStyle(symTint)
                            .frame(width: 38)

                        VStack(alignment: .leading, spacing: 1) {
                            HStack(alignment: .firstTextBaseline, spacing: 6) {
                                Text(tempStr)
                                    .font(.system(size: 28, weight: .bold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textPrimary)

                                if let feels = feelsStr {
                                    Text("Feels \(feels)")
                                        .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.textSecondary)
                                }
                            }

                            if let desc, !desc.isEmpty {
                                Text(desc)
                                    .font(.system(size: 11, weight: .medium, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textSecondary)
                                    .lineLimit(1)
                            }
                        }

                        Spacer(minLength: 0)

                        hudRefreshButton
                    }
                    .padding(.horizontal, 12)

                    LazyVGrid(
                        columns: [GridItem(.flexible(), spacing: 5), GridItem(.flexible(), spacing: 5), GridItem(.flexible(), spacing: 5), GridItem(.flexible(), spacing: 5)],
                        spacing: 5
                    ) {
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
                    .padding(.horizontal, 12)

                    HourlyStripView()
                        .padding(.horizontal, 12)
                    PeriodsListView(limit: compactPeriodCap, compactRows: true)
                        .padding(.horizontal, 12)
                }
                .padding(.vertical, 7)
            }

            horizonDivider

            hudFooterBar(showRefresh: false)
        }
    }

    // MARK: - Mode 2: Standard HUD (380 × 460)

    private var standardHUD: some View {
        VStack(spacing: 0) {
            hudTopHeader(badge: "WX.HUD")

            horizonDivider

            ScrollView(showsIndicators: false) {
                VStack(alignment: .leading, spacing: 10) {
                    WxGlassCard(title: "Atmospheric Telemetry", tag: store.payload?.conditions?.station ?? "GRID.OBS") {
                        NowBlockView(compact: true, popoverMetrics: true)
                    }

                    if let nc = store.nowcastPayload?.nowcast ?? store.payload?.conditions?.nowcast ?? store.payload?.nowcast,
                       (nc.isActivePrecip == true || nc.nextPrecipTime != nil || (nc.totalLiquidMm ?? 0) > 0.1) {
                        NowcastCardView(nowcast: nc)
                    }

                    if hasConditionAlerts {
                        WxGlassCard(title: "Active Alerts", tag: "NWS.WARN", accent: WxGlassTheme.conditionRed) {
                            AlertsListView(popoverMode: true)
                        }
                    }

                    WxGlassCard(title: "Synoptic Forecast", tag: "NOAA.NWS") {
                        HourlyStripView()
                        PeriodsListView(limit: periodLimit, compactRows: true)
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 10)
                .frame(maxWidth: .infinity)
            }

            horizonDivider

            hudFooterBar(showRefresh: true)
        }
    }

    // MARK: - Mode 3: Tactical HUD (410 × 540)

    private var tacticalHUD: some View {
        VStack(spacing: 0) {
            hudTopHeader(badge: "WX.TAC")

            horizonDivider

            ScrollView(showsIndicators: false) {
                VStack(alignment: .leading, spacing: 10) {
                    WxGlassCard(title: "Tactical Sensor Control", tag: "SYS.01") {
                        ControlsBar(showOpenWindow: true, compact: true)
                    }

                    if !store.gridCards.isEmpty {
                        WxGlassCard(title: "Pinned Stations", tag: "\(store.gridCards.count)") {
                            PopoverGridPreviewBar()
                        }
                    }

                    if let shift = store.cpcPayload?.patternShift, shift.hasShift {
                        Button {
                            store.selectedDeskTab = .outlooks
                            NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                        } label: {
                            HStack(spacing: 8) {
                                Image(systemName: "exclamationmark.bubble.fill")
                                    .font(.system(size: 12))
                                    .foregroundStyle(WxGlassTheme.amber)
                                Text("Regime Shift: \(shift.summary)")
                                    .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textPrimary)
                                    .lineLimit(1)
                                Spacer()
                                Image(systemName: "chevron.right")
                                    .font(.system(size: 9, weight: .bold))
                                    .foregroundStyle(WxGlassTheme.cyan)
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 8)
                            .wxFrostedCapsule(accent: WxGlassTheme.amber)
                        }
                        .buttonStyle(.plain)
                    }

                    if let chase = store.chasePayload, chase.totalClusters > 0 {
                        Button {
                            store.selectedDeskTab = .chase
                            NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                        } label: {
                            HStack(spacing: 8) {
                                Image(systemName: "bolt.shield.fill")
                                    .font(.system(size: 12))
                                    .foregroundStyle(WxGlassTheme.amber)
                                Text("Storm Chase: \(chase.totalClusters) Active Clusters")
                                    .font(.system(size: 10.5, weight: .bold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textPrimary)
                                    .lineLimit(1)
                                Spacer()
                                Image(systemName: "chevron.right")
                                    .font(.system(size: 9, weight: .bold))
                                    .foregroundStyle(WxGlassTheme.cyan)
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 8)
                            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                        }
                        .buttonStyle(.plain)
                    }

                    WxGlassCard(title: "Atmospheric Telemetry", tag: store.payload?.conditions?.station ?? "GRID.OBS") {
                        NowBlockView(compact: true, popoverMetrics: true)
                    }

                    if let nc = store.nowcastPayload?.nowcast ?? store.payload?.conditions?.nowcast ?? store.payload?.nowcast,
                       (nc.isActivePrecip == true || nc.nextPrecipTime != nil || (nc.totalLiquidMm ?? 0) > 0.1) {
                        NowcastCardView(nowcast: nc)
                    }

                    if hasConditionAlerts {
                        WxGlassCard(title: "Active Alerts", tag: "NWS.WARN", accent: WxGlassTheme.conditionRed) {
                            AlertsListView(popoverMode: true)
                        }
                    }

                    WxGlassCard(title: "Synoptic Forecast", tag: "NOAA.NWS") {
                        HourlyStripView()
                        PeriodsListView(limit: periodLimit, compactRows: true)
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 10)
                .frame(maxWidth: .infinity)
            }

            horizonDivider

            HStack {
                Text("NOAA · NWS · MRMS 1KM")
                    .font(.system(size: 9, weight: .semibold, design: .monospaced))
                    .foregroundStyle(WxGlassTheme.textMuted)
                Spacer()
                if let sub = updatedSubtitle {
                    Text("SYNC \(sub)")
                        .font(.system(size: 9, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxGlassTheme.cyanSoft)
                }
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 7)
            .background(Color.black.opacity(0.22))
        }
    }

    // MARK: - Shared HUD Chrome

    private func hudTopHeader(badge: String) -> some View {
        HStack(spacing: 8) {
            HStack(spacing: 5) {
                Circle()
                    .fill(WxGlassTheme.nominalGreen)
                    .frame(width: 6, height: 6)
                    .shadow(color: WxGlassTheme.nominalGreen.opacity(0.8), radius: 3)
                Text(badge)
                    .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxGlassTheme.gold)
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 3.5)
            .background(WxGlassTheme.gold.opacity(0.14), in: Capsule())

            Text(store.payload?.conditions?.location ?? "Atmospheric HUD")
                .font(.system(size: 12.5, weight: .bold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)
                .lineLimit(1)

            Spacer(minLength: 4)

            if let sub = updatedSubtitle {
                Text(sub)
                    .font(.system(size: 9.5, weight: .semibold, design: .monospaced))
                    .foregroundStyle(WxGlassTheme.textMuted)
            }

            hudModePicker
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 9)
        .background(Color.black.opacity(0.18))
    }

    private func hudFooterBar(showRefresh: Bool) -> some View {
        HStack(spacing: 8) {
            Picker("Units", selection: $store.units) {
                Text("°F").tag("imperial")
                Text("°C").tag("metric")
            }
            .pickerStyle(.segmented)
            .labelsHidden()
            .frame(width: 60)
            .onChange(of: store.units) { _, _ in
                Task { await store.applyLocationAndUnits() }
            }

            if showRefresh {
                hudRefreshButton
            }

            Spacer(minLength: 4)

            Button {
                NotificationCenter.default.post(name: .wxOpenDeskRadar, object: nil)
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: "dot.radiowaves.left.and.right")
                        .font(.system(size: 9))
                    Text("Radar")
                        .font(.system(size: 10, weight: .semibold, design: .rounded))
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 5)
                .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                .foregroundStyle(WxGlassTheme.cyan)
            }
            .buttonStyle(.plain)
            .help("Open Radar (⌘3)")

            Button {
                NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: "macwindow")
                        .font(.system(size: 9))
                    Text("Desk")
                        .font(.system(size: 10, weight: .semibold, design: .rounded))
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 5)
                .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                .foregroundStyle(WxGlassTheme.cyan)
            }
            .buttonStyle(.plain)
            .help("Open Desk Console (⌘1)")
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 7)
        .background(Color.black.opacity(0.22))
    }

    private var hudRefreshButton: some View {
        Button {
            Task { await store.refresh() }
        } label: {
            Group {
                if store.isLoading {
                    ProgressView()
                        .controlSize(.small)
                        .frame(width: 13, height: 13)
                } else {
                    Image(systemName: "arrow.triangle.2.circlepath")
                        .font(.system(size: 10, weight: .bold))
                        .foregroundStyle(WxGlassTheme.cyan)
                }
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 5)
            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
        }
        .buttonStyle(.plain)
        .help("Refresh (⌘R)")
        .disabled(store.isLoading)
    }

    private var horizonDivider: some View {
        LinearGradient(
            colors: [.clear, WxGlassTheme.cyan.opacity(0.42), Color.white.opacity(0.22), WxGlassTheme.cyan.opacity(0.42), .clear],
            startPoint: .leading,
            endPoint: .trailing
        )
        .frame(height: 1)
    }

    private var hudModePicker: some View {
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
            HStack(spacing: 4) {
                Image(systemName: "menubar.arrow.up.rectangle")
                    .font(.system(size: 8.5))
                    .foregroundStyle(WxGlassTheme.cyan)
                Text(store.menuBarFormat.rawValue.capitalized)
                    .font(.system(size: 9.5, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textPrimary)
                Image(systemName: "chevron.down")
                    .font(.system(size: 7, weight: .bold))
                    .foregroundStyle(WxGlassTheme.textMuted)
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
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

                        let sym = ConditionSymbol.resolvedSystemName(for: card.payload)
                        let cardNight = ConditionSymbol.isNight(for: card.payload)
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
                            VStack(alignment: .leading, spacing: 4) {
                                HStack(spacing: 4) {
                                    if isCurrent {
                                        Circle().fill(WxGlassTheme.cyan).frame(width: 5, height: 5)
                                    }
                                    Text(card.displayName)
                                        .font(.system(size: 10.5, weight: isCurrent ? .bold : .semibold, design: .rounded))
                                        .foregroundStyle(isCurrent ? WxGlassTheme.cyan : WxGlassTheme.textPrimary)
                                        .lineLimit(1)
                                    Spacer(minLength: 0)
                                    if hasWarning {
                                        Circle().fill(WxGlassTheme.conditionRed).frame(width: 5.5, height: 5.5)
                                    } else if hasWatch {
                                        Circle().fill(WxGlassTheme.amber).frame(width: 5.5, height: 5.5)
                                    }
                                }

                                HStack(spacing: 5) {
                                    Image(systemName: sym)
                                        .font(.system(size: 12))
                                        .symbolRenderingMode(.hierarchical)
                                        .foregroundStyle(ConditionSymbol.tint(for: sym, isNight: cardNight))
                                    Text(tempStr)
                                        .font(.system(size: 13, weight: .bold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.textPrimary)
                                    if !windStr.isEmpty {
                                        Text(windStr)
                                            .font(.system(size: 9, design: .monospaced))
                                            .foregroundStyle(WxGlassTheme.textMuted)
                                    }
                                }
                            }
                            .padding(.horizontal, 10)
                            .padding(.vertical, 7)
                            .frame(width: 130, alignment: .leading)
                            .background(
                                isCurrent ? WxGlassTheme.cyan.opacity(0.16) : Color.white.opacity(0.05),
                                in: WxGlassTheme.nestedShape()
                            )
                            .overlay(
                                WxGlassTheme.nestedShape()
                                    .strokeBorder(
                                        isCurrent ? WxGlassTheme.cyan.opacity(0.6) : Color.white.opacity(0.12),
                                        lineWidth: isCurrent ? 1 : 0.8
                                    )
                            )
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.vertical, 2)
            }

            HStack {
                Text("\(store.gridCards.count) pinned stations")
                    .font(.system(size: 9, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textMuted)
                Spacer()
                Button {
                    store.selectedDeskTab = .grid
                    NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                } label: {
                    HStack(spacing: 3) {
                        Text("Command Grid")
                            .font(.system(size: 9.5, weight: .semibold, design: .rounded))
                        Image(systemName: "chevron.right")
                            .font(.system(size: 7.5, weight: .bold))
                    }
                    .foregroundStyle(WxGlassTheme.cyan)
                }
                .buttonStyle(.plain)
            }
        }
    }
}
