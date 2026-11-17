import SwiftUI

struct NowBlockView: View {
    @EnvironmentObject var store: WeatherStore
    var compact: Bool = false
    /// Popover: concise telemetry grid.
    var popoverMetrics: Bool = false

    var body: some View {
        let c = store.payload?.conditions
        let sym = store.statusSymbol
        let symTint = ConditionSymbol.tint(for: sym, isNight: store.isNight)
        let desc = store.resolvedConditionDescription
        VStack(alignment: .leading, spacing: compact ? 8 : 12) {
            if compact {
                HStack(alignment: .center, spacing: 10) {
                    Image(systemName: sym)
                        .font(.system(size: 32, weight: .medium))
                        .symbolRenderingMode(.hierarchical)
                        .foregroundStyle(symTint)
                        .shadow(color: symTint.opacity(0.4), radius: 5)

                    VStack(alignment: .leading, spacing: 2) {
                        HStack(alignment: .firstTextBaseline, spacing: 8) {
                            Text(store.displayTemp)
                                .font(.system(size: 28, weight: .bold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.textPrimary)
                            if let desc, !desc.isEmpty {
                                Text(desc)
                                    .font(.system(size: 12.5, weight: .medium, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textSecondary)
                                    .lineLimit(1)
                            }
                        }
                        Text(c?.location ?? "—")
                            .font(.system(size: 11.5, weight: .semibold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textMuted)
                            .lineLimit(1)
                    }

                    Spacer()

                    if store.isLoading {
                        ProgressView().controlSize(.small).tint(WxGlassTheme.cyan)
                    }
                }
            } else {
                HStack(alignment: .center, spacing: 12) {
                    Image(systemName: sym)
                        .font(.system(size: 40, weight: .medium))
                        .symbolRenderingMode(.hierarchical)
                        .foregroundStyle(symTint)
                        .shadow(color: symTint.opacity(0.45), radius: 8)

                    Text(store.displayTemp)
                        .font(.system(size: 50, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                        .shadow(color: Color.black.opacity(0.25), radius: 6, y: 2)

                    Spacer()

                    if store.isLoading {
                        ProgressView().controlSize(.small).tint(WxGlassTheme.cyan)
                    }
                }

                Text(c?.location ?? "—")
                    .font(.system(size: 17, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textPrimary)

                if let desc, !desc.isEmpty {
                    Text(desc)
                        .font(.system(size: 13.5, weight: .medium, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                }

                if let station = c?.station, let observed = c?.observedAt {
                    Text("STATION \(station.uppercased()) · \(observed)")
                        .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxGlassTheme.textMuted)
                }
            }

            metricsGrid(c)
        }
    }

    @ViewBuilder
    private func metricsGrid(_ c: Conditions?) -> some View {
        let metric = store.units == "metric"
        FlowMetrics(isPopover: popoverMetrics) {
            if let h = c?.humidityPct {
                MetricChip(label: "Humidity", value: String(format: "%.0f%%", h), symbol: "drop.fill")
            }
            if metric {
                if let dp = c?.dewPointC {
                    MetricChip(label: "Dew Point", value: String(format: "%.0f°", dp), symbol: "thermometer.medium")
                }
                if let w = c?.windKph {
                    let dir = c?.windDirection.map { " \($0)" } ?? ""
                    MetricChip(label: "Wind", value: String(format: "%.0f km/h%@", w, dir), symbol: "wind")
                }
                if !popoverMetrics, let g = c?.windGustKph {
                    MetricChip(label: "Gust", value: String(format: "%.0f km/h", g), symbol: "wind")
                }
                if let f = c?.feelsLikeC {
                    MetricChip(label: "Feels", value: String(format: "%.0f°", f), symbol: "thermometer.sun.fill")
                }
                if let p = c?.pressureHpa {
                    MetricChip(label: "Pressure", value: String(format: "%.0f hPa", p), symbol: "gauge.with.dots.needle.bottom.50percent")
                }
                if let v = c?.visibilityM {
                    MetricChip(label: "Vis", value: String(format: "%.1f km", v / 1000), symbol: "eye.fill")
                }
            } else {
                if let dp = c?.dewPointF {
                    MetricChip(label: "Dew Point", value: String(format: "%.0f°", dp), symbol: "thermometer.medium")
                }
                if let w = c?.windMph {
                    let dir = c?.windDirection.map { " \($0)" } ?? ""
                    MetricChip(label: "Wind", value: String(format: "%.0f mph%@", w, dir), symbol: "wind")
                }
                if !popoverMetrics, let g = c?.windGustMph {
                    MetricChip(label: "Gust", value: String(format: "%.0f mph", g), symbol: "wind")
                }
                if let f = c?.feelsLikeF {
                    MetricChip(label: "Feels", value: String(format: "%.0f°", f), symbol: "thermometer.sun.fill")
                }
                if let p = c?.pressureInhg {
                    MetricChip(label: "Pressure", value: String(format: "%.2f inHg", p), symbol: "gauge.with.dots.needle.bottom.50percent")
                }
                if let v = c?.visibilityMi {
                    MetricChip(label: "Vis", value: String(format: "%.1f mi", v), symbol: "eye.fill")
                }
            }

            // Solar & Lunar Telemetry (SF Symbols — Zero OS Color Emojis)
            if let astro = c?.astronomy ?? store.payload?.astronomy {
                if astro.isPolarDay == true {
                    MetricChip(label: "Sun", value: "Polar Day", symbol: "sun.max.fill")
                } else if astro.isPolarNight == true {
                    MetricChip(label: "Sun", value: "Polar Night", symbol: "moon.stars.fill")
                } else if let sr = astro.sunriseFormatted, let ss = astro.sunsetFormatted {
                    if popoverMetrics {
                        MetricChip(label: "Sun", value: "↑\(sr) ↓\(ss)", symbol: "sun.horizon.fill")
                    } else {
                        MetricChip(label: "Sunrise", value: sr, symbol: "sunrise.fill")
                        MetricChip(label: "Sunset", value: ss, symbol: "sunset.fill")
                        if let dl = astro.dayLength {
                            MetricChip(label: "Daylight", value: dl, symbol: "clock.fill")
                        }
                    }
                }

                if let phase = astro.moonPhase {
                    let moonSym = ConditionSymbol.moonSymbol(for: phase)
                    if popoverMetrics {
                        let illum = astro.moonIlluminationPct.map { String(format: " (%.0f%%)", $0) } ?? ""
                        MetricChip(label: "Moon", value: "\(phase)\(illum)", symbol: moonSym)
                    } else {
                        MetricChip(label: "Moon", value: phase, symbol: moonSym)
                        if let illum = astro.moonIlluminationPct {
                            MetricChip(label: "Moon Illum", value: String(format: "%.0f%%", illum), symbol: moonSym)
                        }
                        if let age = astro.moonAgeDays {
                            MetricChip(label: "Moon Age", value: String(format: "%.1fd", age), symbol: "moon.stars.fill")
                        }
                    }
                }
            }

            if let aq = c?.airQuality ?? store.payload?.airQuality {
                if let aqi = aq.aqi {
                    let cat = aq.category.map { " (\($0))" } ?? ""
                    MetricChip(label: "AQI", value: "\(aqi)\(cat)", symbol: "aqi.medium")
                }
                if let uv = aq.uvIndex {
                    let cat = aq.uvCategory.map { " (\($0))" } ?? ""
                    MetricChip(label: "UV Index", value: String(format: "%.1f%@", uv, cat), symbol: "sun.max.trianglebadge.exclamationmark")
                }
            }
        }
    }
}

struct MetricChip: View {
    let label: String
    let value: String
    var symbol: String? = nil

    var body: some View {
        WxSensorCapsule(label: label, value: value, symbol: symbol, compact: true)
    }
}

struct FlowMetrics<Content: View>: View {
    var isPopover: Bool = false
    @ViewBuilder var content: Content

    var body: some View {
        if isPopover {
            LazyVGrid(
                columns: [GridItem(.flexible(), spacing: 6), GridItem(.flexible(), spacing: 6), GridItem(.flexible(), spacing: 6)],
                alignment: .leading,
                spacing: 6
            ) {
                content
            }
        } else {
            LazyVGrid(columns: [GridItem(.adaptive(minimum: 102), spacing: 7)], alignment: .leading, spacing: 7) {
                content
            }
        }
    }
}

struct PeriodsListView: View {
    @EnvironmentObject var store: WeatherStore
    var limit: Int?
    /// Popover: fixed rows, no disclosure.
    var compactRows: Bool = false

    var body: some View {
        let cap = limit ?? Int.max
        let allRows = store.payload?.forecast?.synopticRows(units: store.units) ?? []
        let rows = Array(allRows.prefix(cap))
        let allTemps = allRows.flatMap { [$0.lowValue, $0.highValue].compactMap { $0 } }
        let weekMin = allTemps.min() ?? 40
        let weekMax = max(weekMin + 1, allTemps.max() ?? 80)

        if rows.isEmpty {
            EmptyView()
        } else {
            VStack(alignment: .leading, spacing: compactRows ? 4 : 6) {
                ForEach(rows) { row in
                    if compactRows {
                        rowSummary(row, weekMin: weekMin, weekMax: weekMax, compact: true)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 5)
                            .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
                    } else {
                        DisclosureGroup {
                            VStack(alignment: .leading, spacing: 6) {
                                periodDetailBlock(row.primaryPeriod, heading: row.secondaryPeriod != nil ? row.primaryPeriod.name : nil)
                                if let night = row.secondaryPeriod {
                                    Divider().overlay(Color.white.opacity(0.08))
                                    periodDetailBlock(night, heading: night.name)
                                }
                            }
                            .padding(.vertical, 4)
                        } label: {
                            rowSummary(row, weekMin: weekMin, weekMax: weekMax, compact: false)
                        }
                        .tint(WxGlassTheme.cyan.opacity(0.85))
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(Color.white.opacity(0.04), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func rowSummary(_ row: SynopticForecastRow, weekMin: Double, weekMax: Double, compact: Bool) -> some View {
        let p = row.primaryPeriod
        HStack(spacing: compact ? 6 : 8) {
            Image(systemName: periodSymbol(p))
                .font(.system(size: compact ? 11.5 : 13, weight: .medium))
                .symbolRenderingMode(.hierarchical)
                .foregroundStyle(periodTint(p))
                .frame(width: compact ? 16 : 18)

            Text(row.name)
                .font(.system(size: compact ? 12 : 13.5, weight: .semibold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)
                .lineLimit(1)
                .minimumScaleFactor(0.82)
                .frame(maxWidth: .infinity, alignment: .leading)

            if let pop = row.precipPct, pop > 0 {
                HStack(spacing: 2) {
                    Image(systemName: "drop.fill")
                        .font(.system(size: compact ? 7.5 : 8))
                    Text(String(format: "%.0f%%", pop))
                        .font(.system(size: compact ? 9 : 9.5, weight: .bold, design: .rounded))
                }
                .foregroundStyle(WxGlassTheme.cyan)
                .padding(.horizontal, compact ? 5 : 6)
                .padding(.vertical, 2)
                .background(WxGlassTheme.cyan.opacity(0.14), in: Capsule())
            }

            switch row.kind {
            case .paired:
                Text(row.lowText ?? "—")
                    .font(.system(size: compact ? 11.5 : 12.5, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textMuted)
                    .frame(width: compact ? 28 : 30, alignment: .trailing)

                if let lo = row.lowValue, let hi = row.highValue {
                    WxThermalRangeBar(
                        dayLow: min(lo, hi),
                        dayHigh: max(lo, hi),
                        weekMin: weekMin,
                        weekMax: weekMax
                    )
                    .frame(width: compact ? 46 : 56)
                }

                Text(row.highText ?? "—")
                    .font(.system(size: compact ? 12.5 : 14, weight: .bold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textPrimary)
                    .frame(width: compact ? 30 : 32, alignment: .trailing)

            case .nightOnly:
                HStack(spacing: 4) {
                    Text("Low")
                        .font(.system(size: compact ? 10 : 11, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.periwinkle)
                    Text(row.lowText ?? "—")
                        .font(.system(size: compact ? 12 : 13.5, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                }
                .padding(.horizontal, compact ? 7 : 9)
                .padding(.vertical, 2)
                .background(WxGlassTheme.periwinkle.opacity(0.12), in: Capsule())
                .overlay(Capsule().strokeBorder(WxGlassTheme.periwinkle.opacity(0.32), lineWidth: 0.8))

            case .dayOnly:
                HStack(spacing: 4) {
                    Text("High")
                        .font(.system(size: compact ? 10 : 11, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.gold)
                    Text(row.highText ?? "—")
                        .font(.system(size: compact ? 12 : 13.5, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                }
                .padding(.horizontal, compact ? 7 : 9)
                .padding(.vertical, 2)
                .background(WxGlassTheme.gold.opacity(0.12), in: Capsule())
                .overlay(Capsule().strokeBorder(WxGlassTheme.gold.opacity(0.32), lineWidth: 0.8))
            }

            if !compact, let blurb = row.shortDescription, !blurb.isEmpty {
                Text(blurb)
                    .font(.system(size: 11))
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .lineLimit(1)
                    .frame(maxWidth: 110, alignment: .trailing)
            }
        }
    }

    @ViewBuilder
    private func periodDetailBlock(_ p: Period, heading: String?) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            if let heading {
                Text(heading.uppercased())
                    .font(.system(size: 9.5, weight: .bold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textMuted)
            }
            if let detail = p.detailedDescription {
                Text(detail)
                    .font(.system(size: 12))
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .padding(.vertical, 1)
            }
            HStack(spacing: 12) {
                if let pop = p.probabilityOfPrecipitation {
                    Label(String(format: "Precip %.0f%%", pop), systemImage: "drop.fill")
                        .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.cyan)
                }
                if let wind = windSummary(p) {
                    Label(wind, systemImage: "wind")
                        .font(.system(size: 10.5, weight: .medium, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                }
                if let dp = p.dewPointF, store.units != "metric" {
                    Text("Dew Point \(Int(dp))°")
                        .font(.system(size: 10.5, weight: .medium, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textMuted)
                } else if let dp = p.dewPointC, store.units == "metric" {
                    Text("Dew Point \(Int(dp))°")
                        .font(.system(size: 10.5, weight: .medium, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textMuted)
                }
            }
        }
    }

    private func periodSymbol(_ p: Period) -> String {
        ConditionSymbol.periodSymbol(for: p)
    }

    private func periodTint(_ p: Period) -> Color {
        ConditionSymbol.tint(for: periodSymbol(p), isNight: !p.resolvedIsDaytime)
    }

    private func windSummary(_ p: Period) -> String? {
        let dir = p.windDirection ?? ""
        if store.units == "metric", let w = p.windKph {
            return String(format: "%.0f km/h %@", w, dir).trimmingCharacters(in: .whitespaces)
        } else if let w = p.windMph {
            return String(format: "%.0f mph %@", w, dir).trimmingCharacters(in: .whitespaces)
        }
        return nil
    }
}

struct AlertsListView: View {
    @EnvironmentObject var store: WeatherStore
    var popoverMode: Bool = false

    var body: some View {
        let alerts = store.payload?.alerts ?? []
        let withBand: [(Alert, ConditionBand)] = alerts.compactMap { a in
            guard let b = ConditionBand.from(severity: a.severity) else { return nil }
            return (a, b)
        }
        if withBand.isEmpty {
            EmptyView()
        } else if popoverMode {
            let shown = Array(withBand.prefix(1))
            let extra = withBand.count - shown.count
            VStack(alignment: .leading, spacing: 8) {
                ForEach(shown, id: \.0.id) { item in
                    compactRow(alert: item.0, band: item.1, badgeSize: 88)
                }
                if extra > 0 {
                    Button {
                        NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                    } label: {
                        Text("+\(extra) more in Desk")
                            .font(.system(size: 11.5, weight: .semibold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.cyan)
                    }
                    .buttonStyle(.plain)
                }
            }
        } else {
            WxGlassCard(title: "Active NWS Watches & Warnings", tag: "\(withBand.count)", accent: WxGlassTheme.conditionRed) {
                VStack(alignment: .leading, spacing: 10) {
                    ForEach(withBand, id: \.0.id) { item in
                        DisclosureGroup {
                            VStack(alignment: .leading, spacing: 6) {
                                if let d = item.0.description {
                                    Text(d)
                                        .font(.system(size: 11.5))
                                        .foregroundStyle(WxGlassTheme.textPrimary)
                                }
                                if let i = item.0.instruction {
                                    Text(i)
                                        .font(.system(size: 11.5))
                                        .foregroundStyle(WxGlassTheme.textSecondary)
                                }
                            }
                            .padding(.top, 4)
                        } label: {
                            compactRow(alert: item.0, band: item.1, badgeSize: 96)
                        }
                        .tint(WxGlassTheme.periwinkle)
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func compactRow(alert: Alert, band: ConditionBand, badgeSize: CGFloat) -> some View {
        let alertColor = (band == .red ? WxGlassTheme.conditionRed : WxGlassTheme.gold)
        let shape = RoundedRectangle(cornerRadius: 12, style: .continuous)

        HStack(alignment: .center, spacing: 10) {
            ConditionPanel(band: band, size: badgeSize)
            VStack(alignment: .leading, spacing: 3) {
                Text(alert.event.uppercased())
                    .font(.system(size: 12, weight: .bold, design: .rounded))
                    .foregroundStyle(alertColor)
                    .lineLimit(1)
                if let h = alert.headline {
                    Text(h)
                        .font(.system(size: 11.5))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                        .lineLimit(popoverMode ? 1 : 2)
                } else if let exp = alert.expires {
                    Text("EXPIRES \(exp)")
                        .font(.system(size: 9.5, design: .monospaced))
                        .foregroundStyle(WxGlassTheme.textMuted)
                        .lineLimit(1)
                }
            }
            Spacer(minLength: 0)
        }
        .padding(9)
        .background(alertColor.opacity(0.10), in: shape)
        .overlay(shape.strokeBorder(alertColor.opacity(0.42), lineWidth: 0.9))
    }
}

struct ControlsBar: View {
    @EnvironmentObject var store: WeatherStore
    var showOpenWindow: Bool = false
    var compact: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: 7) {
            if compact {
                LocationBarView(isHUD: true)

                HStack(spacing: 6) {
                    Picker("Units", selection: $store.units) {
                        Text("°F").tag("imperial")
                        Text("°C").tag("metric")
                    }
                    .pickerStyle(.segmented)
                    .labelsHidden()
                    .frame(width: 64)
                    .onChange(of: store.units) { _, _ in
                        Task { await store.applyLocationAndUnits() }
                    }

                    menuBarFormatMenu(compact: true)

                    refreshButton

                    Spacer(minLength: 4)

                    if showOpenWindow {
                        openRadarButton
                        openDeskButton
                    }
                }
            } else {
                HStack(spacing: 6) {
                    LocationBarView()

                    Picker("Units", selection: $store.units) {
                        Text("°F").tag("imperial")
                        Text("°C").tag("metric")
                    }
                    .pickerStyle(.segmented)
                    .labelsHidden()
                    .frame(width: 72)
                    .onChange(of: store.units) { _, _ in
                        Task { await store.applyLocationAndUnits() }
                    }

                    menuBarFormatMenu(compact: false)

                    refreshButton

                    if showOpenWindow {
                        openRadarButton
                        openDeskButton
                    }
                }
            }

            if !store.favorites.isEmpty {
                FavoritesQuickBarView()
            }

            if let err = store.errorMessage {
                if err == LocationManagerError.accessOffMessage {
                    locationAccessOffNotice
                } else {
                    Text("// ALERT: \(err)")
                        .font(.system(size: 10, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxGlassTheme.conditionRed)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .accessibilityElement(children: .ignore)
                        .accessibilityIdentifier("wx.location-alert")
                        .accessibilityLabel("// ALERT: \(err)")
                }
            } else if let warn = store.payload?.warning, !warn.isEmpty {
                Text(warn)
                    .font(.system(size: 10, weight: .medium, design: .rounded))
                    .foregroundStyle(WxGlassTheme.amber)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
    }

    private var locationAccessOffNotice: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(LocationManagerError.accessOffMessage)
                .font(.system(size: 10.5, weight: .medium, design: .rounded))
                .foregroundStyle(WxGlassTheme.conditionRed)
                .frame(maxWidth: .infinity, alignment: .leading)
                .accessibilityElement(children: .ignore)
                .accessibilityIdentifier("wx.location-denied")
                .accessibilityLabel(LocationManagerError.accessOffMessage)
            Button {
                store.openLocationSettings()
            } label: {
                Text("Open Location Services Settings")
                    .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                    .underline()
                    .foregroundStyle(WxGlassTheme.cyan)
            }
            .buttonStyle(.plain)
            .accessibilityIdentifier("wx.open-location-settings")
            .accessibilityLabel("Open Location Services Settings")
        }
    }

    @ViewBuilder
    private func menuBarFormatMenu(compact: Bool) -> some View {
        Menu {
            Text("MENU BAR & HUD MODE")
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
                    .font(.system(size: 9))
                    .foregroundStyle(WxGlassTheme.cyan)
                Text(store.menuBarFormat.rawValue.capitalized)
                    .font(.system(size: 10, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textPrimary)
                Image(systemName: "chevron.down")
                    .font(.system(size: 7.5, weight: .bold))
                    .foregroundStyle(WxGlassTheme.textMuted)
            }
            .padding(.horizontal, 9)
            .padding(.vertical, 5)
            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
        }
        .menuStyle(.borderlessButton)
        .fixedSize()
        .help("Configure macOS Menu Bar & HUD Popover Mode (Compact, Standard, Tactical)")
    }

    private var refreshButton: some View {
        Button {
            Task {
                await store.refresh()
                await store.refreshCPC()
                await store.refreshChase()
            }
        } label: {
            Group {
                if store.isLoading {
                    ProgressView()
                        .controlSize(.small)
                        .frame(width: 14, height: 14)
                } else {
                    Image(systemName: "arrow.triangle.2.circlepath")
                        .font(.system(size: 10.5, weight: .bold))
                        .foregroundStyle(WxGlassTheme.cyan)
                }
            }
            .padding(.horizontal, 9)
            .padding(.vertical, 5)
            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
        }
        .buttonStyle(.plain)
        .help("Refresh Sensor Telemetry (⌘R)")
        .disabled(store.isLoading)
    }

    private var openRadarButton: some View {
        Button {
            NotificationCenter.default.post(name: .wxOpenDeskRadar, object: nil)
        } label: {
            HStack(spacing: 4) {
                Image(systemName: "dot.radiowaves.left.and.right")
                    .font(.system(size: 9))
                Text("Radar")
                    .font(.system(size: 10, weight: .semibold, design: .rounded))
            }
            .padding(.horizontal, 9)
            .padding(.vertical, 5)
            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
            .foregroundStyle(WxGlassTheme.cyan)
        }
        .buttonStyle(.plain)
        .help("Open Radar Tactical Array (⌘3)")
    }

    private var openDeskButton: some View {
        Button {
            NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
        } label: {
            HStack(spacing: 4) {
                Image(systemName: "macwindow")
                    .font(.system(size: 9))
                Text("Desk")
                    .font(.system(size: 10, weight: .semibold, design: .rounded))
            }
            .padding(.horizontal, 9)
            .padding(.vertical, 5)
            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
            .foregroundStyle(WxGlassTheme.cyan)
        }
        .buttonStyle(.plain)
        .help("Open Tactical Console (⌘1)")
    }
}

func wxPeriodSymbol(_ p: Period) -> String {
    ConditionSymbol.periodSymbol(for: p)
}

/// One horizontal hour row. Draws nothing when hourly data is missing.
struct HourlyStripView: View {
    @EnvironmentObject var store: WeatherStore

    var body: some View {
        let hours = store.payload?.forecast?.activeHourlyPeriods() ?? []
        if hours.isEmpty {
            EmptyView()
        } else {
            Group {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 7) {
                        ForEach(hours) { p in
                            hourCell(p)
                        }
                    }
                    .padding(.horizontal, 2)
                }
                .frame(height: 68)
            }
            .accessibilityElement(children: .contain)
            .accessibilityIdentifier("wx.hourly.strip")
            .accessibilityLabel(hours.map { "\(hourLabel($0)) \(tempText($0))" }.joined(separator: ", "))
        }
    }

    private func hourCell(_ p: Period) -> some View {
        let hour = hourLabel(p)
        let temp = tempText(p)
        let sym = wxPeriodSymbol(p)
        let iconTint = ConditionSymbol.tint(for: sym, isNight: !p.resolvedIsDaytime)

        return VStack(spacing: 3) {
            Text(hour)
                .font(.system(size: 9.5, weight: .semibold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textSecondary)
            Image(systemName: sym)
                .font(.system(size: 13, weight: .medium))
                .symbolRenderingMode(.hierarchical)
                .foregroundStyle(iconTint)
            Text(temp)
                .font(.system(size: 12, weight: .bold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)
            if let pop = p.probabilityOfPrecipitation, pop > 0 {
                Text(String(format: "%.0f%%", pop))
                    .font(.system(size: 8.5, weight: .bold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.cyan)
            }
        }
        .frame(minWidth: 48)
        .padding(.vertical, 6)
        .padding(.horizontal, 5)
        .background(Color.white.opacity(0.05), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 10, style: .continuous).strokeBorder(Color.white.opacity(0.10), lineWidth: 0.7))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(hour) \(temp)")
    }

    private func hourLabel(_ p: Period) -> String {
        p.formattedHourLabel
    }

    private func tempText(_ p: Period) -> String {
        p.formattedTemperature(units: store.units)
    }
}
