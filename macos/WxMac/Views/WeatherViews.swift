import SwiftUI

struct NowBlockView: View {
    @EnvironmentObject var store: WeatherStore
    var compact: Bool = false
    /// Popover: at most 4 chips (humidity, wind, feels, pressure).
    var popoverMetrics: Bool = false

    var body: some View {
        let c = store.payload?.conditions
        VStack(alignment: .leading, spacing: compact ? 8 : 10) {
            if compact {
                HStack(alignment: .center, spacing: 10) {
                    Image(systemName: store.statusSymbol)
                        .font(.system(size: 32))
                        .foregroundStyle(WxTheme.snwCyan)
                        .shadow(color: WxTheme.snwCyan.opacity(0.4), radius: 4)

                    VStack(alignment: .leading, spacing: 2) {
                        HStack(alignment: .firstTextBaseline, spacing: 8) {
                            Text(store.displayTemp)
                                .font(.system(size: 26, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.text)
                            if let desc = c?.description, !desc.isEmpty {
                                Text(desc)
                                    .font(.system(size: 12, weight: .medium))
                                    .foregroundStyle(WxTheme.textSecondary)
                                    .lineLimit(1)
                            }
                        }
                        Text(c?.location ?? "—")
                            .font(.system(size: 11, weight: .semibold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver)
                            .lineLimit(1)
                    }

                    Spacer()

                    if store.isLoading {
                        ProgressView().controlSize(.small).tint(WxTheme.snwCyan)
                    }
                }
            } else {
                HStack(alignment: .firstTextBaseline) {
                    Image(systemName: store.statusSymbol)
                        .font(.system(size: 38))
                        .foregroundStyle(WxTheme.snwCyan)
                        .shadow(color: WxTheme.snwCyan.opacity(0.4), radius: 5)
                    Text(store.displayTemp)
                        .font(.system(size: 46, weight: .bold, design: .rounded))
                        .foregroundStyle(WxTheme.text)
                        .shadow(color: WxTheme.snwCyan.opacity(0.18), radius: 6)
                    Spacer()
                    if store.isLoading {
                        ProgressView().controlSize(.small).tint(WxTheme.snwCyan)
                    }
                }
                Text(c?.location ?? "—")
                    .font(.headline)
                    .foregroundStyle(WxTheme.text)
                if let desc = c?.description, !desc.isEmpty {
                    Text(desc)
                        .font(.subheadline)
                        .foregroundStyle(WxTheme.textSecondary)
                }
                if let station = c?.station, let observed = c?.observedAt {
                    Text("STATION // \(station) · OBSERVED // \(observed)")
                        .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.75))
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
                MetricChip(label: "Humidity", value: String(format: "%.0f%%", h))
            }
            if metric {
                if let dp = c?.dewPointC {
                    MetricChip(label: "Dew Point", value: String(format: "%.0f°", dp))
                }
                if let w = c?.windKph {
                    let dir = c?.windDirection.map { " \($0)" } ?? ""
                    MetricChip(label: "Wind", value: String(format: "%.0f km/h%@", w, dir))
                }
                if !popoverMetrics, let g = c?.windGustKph {
                    MetricChip(label: "Gust", value: String(format: "%.0f km/h", g))
                }
                if let f = c?.feelsLikeC {
                    MetricChip(label: "Feels", value: String(format: "%.0f°", f))
                }
                if let p = c?.pressureHpa {
                    MetricChip(label: "Pressure", value: String(format: "%.0f hPa", p))
                }
                if let v = c?.visibilityM {
                    MetricChip(label: "Vis", value: String(format: "%.1f km", v / 1000))
                }
            } else {
                if let dp = c?.dewPointF {
                    MetricChip(label: "Dew Point", value: String(format: "%.0f°", dp))
                }
                if let w = c?.windMph {
                    let dir = c?.windDirection.map { " \($0)" } ?? ""
                    MetricChip(label: "Wind", value: String(format: "%.0f mph%@", w, dir))
                }
                if !popoverMetrics, let g = c?.windGustMph {
                    MetricChip(label: "Gust", value: String(format: "%.0f mph", g))
                }
                if let f = c?.feelsLikeF {
                    MetricChip(label: "Feels", value: String(format: "%.0f°", f))
                }
                if let p = c?.pressureInhg {
                    MetricChip(label: "Pressure", value: String(format: "%.2f inHg", p))
                }
                if let v = c?.visibilityMi {
                    MetricChip(label: "Vis", value: String(format: "%.1f mi", v))
                }
            }

            // Solar Telemetry
            if let astro = c?.astronomy ?? store.payload?.astronomy {
                if astro.isPolarDay == true {
                    MetricChip(label: "Sun", value: "Polar Day")
                } else if astro.isPolarNight == true {
                    MetricChip(label: "Sun", value: "Polar Night")
                } else if let sr = astro.sunriseFormatted, let ss = astro.sunsetFormatted {
                    if popoverMetrics {
                        MetricChip(label: "Sun", value: "↑\(sr) ↓\(ss)")
                    } else {
                        MetricChip(label: "Sunrise", value: sr)
                        MetricChip(label: "Sunset", value: ss)
                        if let dl = astro.dayLength {
                            MetricChip(label: "Daylight", value: dl)
                        }
                    }
                }

                // Lunar Telemetry
                if let phase = astro.moonPhase {
                    let icon = astro.moonPhaseIcon.map { "\($0) " } ?? ""
                    if popoverMetrics {
                        let illum = astro.moonIlluminationPct.map { String(format: " (%.0f%%)", $0) } ?? ""
                        MetricChip(label: "Moon", value: "\(icon)\(phase)\(illum)")
                    } else {
                        MetricChip(label: "Moon", value: "\(icon)\(phase)")
                        if let illum = astro.moonIlluminationPct {
                            MetricChip(label: "Moon Illum", value: String(format: "%.0f%%", illum))
                        }
                        if let age = astro.moonAgeDays {
                            MetricChip(label: "Moon Age", value: String(format: "%.1fd", age))
                        }
                    }
                }
            }

            // Air Quality & UV Telemetry
            if let aq = c?.airQuality ?? store.payload?.airQuality {
                if let aqi = aq.aqi {
                    let cat = aq.category.map { " (\($0))" } ?? ""
                    MetricChip(label: "AQI", value: "\(aqi)\(cat)")
                }
                if let uv = aq.uvIndex {
                    let cat = aq.uvCategory.map { " (\($0))" } ?? ""
                    MetricChip(label: "UV Index", value: String(format: "%.1f%@", uv, cat))
                }
            }
        }
    }
}

struct MetricChip: View {
    let label: String
    let value: String
    var body: some View {
        SNWMetricTile(label: label, value: value)
    }
}

struct FlowMetrics<Content: View>: View {
    var isPopover: Bool = false
    @ViewBuilder var content: Content
    var body: some View {
        if isPopover {
            LazyVGrid(columns: [GridItem(.flexible(), spacing: 6), GridItem(.flexible(), spacing: 6), GridItem(.flexible(), spacing: 6)], alignment: .leading, spacing: 6) {
                content
            }
        } else {
            LazyVGrid(columns: [GridItem(.adaptive(minimum: 90), spacing: 6)], alignment: .leading, spacing: 6) {
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
        let periods = Array((store.payload?.forecast?.periods ?? []).prefix(cap))
        if periods.isEmpty {
            EmptyView()
        } else {
            VStack(alignment: .leading, spacing: compactRows ? 0 : 4) {
                ForEach(periods) { p in
                    if compactRows {
                        HStack(spacing: 8) {
                            Image(systemName: periodSymbol(p))
                                .font(.system(size: 9.5))
                                .foregroundStyle(p.isDaytime == false ? WxTheme.snwSilver : WxTheme.snwGold)
                                .frame(width: 14)

                            Text(p.name)
                                .font(.system(size: 11.5, weight: .medium, design: .monospaced))
                                .foregroundStyle(WxTheme.text)
                                .frame(maxWidth: .infinity, alignment: .leading)

                            if let pop = p.probabilityOfPrecipitation, pop > 0 {
                                HStack(spacing: 2) {
                                    Image(systemName: "drop.fill")
                                        .font(.system(size: 7))
                                    Text(String(format: "%.0f%%", pop))
                                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                }
                                .foregroundStyle(WxTheme.snwCyan)
                                .padding(.horizontal, 4)
                                .padding(.vertical, 1.5)
                                .background(WxTheme.snwCyan.opacity(0.12), in: Capsule())
                            }

                            Text(tempText(p))
                                .font(.system(size: 11.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwCyan)
                                .frame(width: 34, alignment: .trailing)

                            Text(p.shortDescription ?? "")
                                .font(.system(size: 10.5))
                                .foregroundStyle(WxTheme.textSecondary)
                                .lineLimit(1)
                                .frame(maxWidth: 120, alignment: .trailing)
                        }
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3.5)
                        .background(
                            RoundedRectangle(cornerRadius: 4, style: .continuous)
                                .fill(WxTheme.snwChassis.opacity(0.45))
                        )
                    } else {
                        DisclosureGroup {
                            VStack(alignment: .leading, spacing: 6) {
                                if let detail = p.detailedDescription {
                                    Text(detail)
                                        .font(.caption)
                                        .foregroundStyle(WxTheme.text)
                                        .padding(.vertical, 2)
                                }
                                HStack(spacing: 12) {
                                    if let pop = p.probabilityOfPrecipitation {
                                        Label(String(format: "Precip: %.0f%%", pop), systemImage: "drop.fill")
                                            .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                                            .foregroundStyle(WxTheme.snwCyan)
                                    }
                                    if let wind = windSummary(p) {
                                        Label(wind, systemImage: "wind")
                                            .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                                            .foregroundStyle(WxTheme.snwSilver)
                                    }
                                    if let dp = p.dewPointF, store.units != "metric" {
                                        Text("Dew Point: \(Int(dp))°")
                                            .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                                            .foregroundStyle(WxTheme.snwSilver.opacity(0.8))
                                    } else if let dp = p.dewPointC, store.units == "metric" {
                                        Text("Dew Point: \(Int(dp))°")
                                            .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                                            .foregroundStyle(WxTheme.snwSilver.opacity(0.8))
                                    }
                                }
                                .padding(.top, 2)
                            }
                            .padding(.vertical, 4)
                        } label: {
                            HStack(spacing: 8) {
                                Image(systemName: periodSymbol(p))
                                    .font(.system(size: 10))
                                    .foregroundStyle(p.isDaytime == false ? WxTheme.snwSilver : WxTheme.snwGold)
                                    .frame(width: 14)

                                Text(p.name)
                                    .font(.system(.body, design: .rounded))
                                    .foregroundStyle(WxTheme.text)
                                    .frame(maxWidth: .infinity, alignment: .leading)

                                if let pop = p.probabilityOfPrecipitation, pop > 0 {
                                    HStack(spacing: 2) {
                                        Image(systemName: "drop.fill")
                                            .font(.system(size: 8))
                                        Text(String(format: "%.0f%%", pop))
                                            .font(.system(size: 9, weight: .bold, design: .monospaced))
                                    }
                                    .foregroundStyle(WxTheme.snwCyan)
                                    .padding(.horizontal, 5)
                                    .padding(.vertical, 2)
                                    .background(WxTheme.snwCyan.opacity(0.12), in: Capsule())
                                }

                                Text(tempText(p))
                                    .font(.system(.body, design: .monospaced).weight(.bold))
                                    .foregroundStyle(WxTheme.snwCyan)

                                Text(p.shortDescription ?? "")
                                    .font(.caption)
                                    .foregroundStyle(WxTheme.textSecondary)
                                    .lineLimit(1)
                                    .frame(maxWidth: 130, alignment: .trailing)
                            }
                            .font(.callout)
                        }
                        .tint(WxTheme.snwCyan.opacity(0.8))
                    }
                }
            }
        }
    }

    private func periodSymbol(_ p: Period) -> String {
        if let dt = p.isDaytime {
            return dt ? "sun.max.fill" : "moon.stars.fill"
        }
        let lower = p.name.lowercased()
        if lower.contains("night") || lower.contains("tonight") {
            return "moon.stars.fill"
        }
        return "sun.max.fill"
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

    private func tempText(_ p: Period) -> String {
        if store.units == "metric", let t = p.temperatureC { return String(format: "%.0f°", t) }
        if let t = p.temperatureF { return String(format: "%.0f°", t) }
        return "—"
    }
}

struct AlertsListView: View {
    @EnvironmentObject var store: WeatherStore
    /// Popover: Option A — one compact badge row + "+N more"; desk shows all with badges.
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
                            .font(.caption.weight(.medium))
                            .foregroundStyle(WxTheme.accent)
                    }
                    .buttonStyle(.plain)
                }
            }
        } else {
            VStack(alignment: .leading, spacing: 10) {
                HStack(spacing: 6) {
                    PhaseAnimator([false, true]) { lit in
                        Circle()
                            .fill(WxTheme.snwRed)
                            .frame(width: 6, height: 6)
                            .opacity(lit ? 1.0 : 0.35)
                            .shadow(color: WxTheme.snwRed.opacity(lit ? 0.9 : 0.1), radius: lit ? 4 : 1)
                    } animation: { _ in
                        .easeInOut(duration: 0.6)
                    }
                    Text("TACTICAL ALERTS // NWS WATCHES & WARNINGS")
                        .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwRed)
                }
                ForEach(withBand, id: \.0.id) { item in
                    DisclosureGroup {
                        VStack(alignment: .leading, spacing: 4) {
                            if let d = item.0.description {
                                Text(d).font(.caption).foregroundStyle(WxTheme.text)
                            }
                            if let i = item.0.instruction {
                                Text(i).font(.caption).foregroundStyle(WxTheme.textSecondary)
                            }
                        }
                    } label: {
                        compactRow(alert: item.0, band: item.1, badgeSize: 100)
                    }
                    .tint(WxTheme.accentSecondary)
                }
            }
        }
    }

    @ViewBuilder
    private func compactRow(alert: Alert, band: ConditionBand, badgeSize: CGFloat) -> some View {
        let alertColor = (band == .red ? WxTheme.snwRed : WxTheme.snwGold)
        HStack(alignment: .center, spacing: 10) {
            ConditionPanel(band: band, size: badgeSize)
            VStack(alignment: .leading, spacing: 2) {
                Text(alert.event.uppercased())
                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                    .foregroundStyle(alertColor)
                    .lineLimit(1)
                if let h = alert.headline {
                    Text(h)
                        .font(.caption)
                        .foregroundStyle(WxTheme.textSecondary)
                        .lineLimit(popoverMode ? 1 : 2)
                } else if let exp = alert.expires {
                    Text("EXPIRES // \(exp)")
                        .font(.system(size: 8.5, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.8))
                        .lineLimit(1)
                }
            }
            Spacer(minLength: 0)
        }
        .padding(8)
        .background(
            RoundedRectangle(cornerRadius: 6, style: .continuous)
                .fill(alertColor.opacity(0.10))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 6, style: .continuous)
                .strokeBorder(alertColor.opacity(0.4), lineWidth: 0.8)
        )
        .overlay(SNWCornerBrackets(color: alertColor, length: 6, thickness: 1))
    }
}

struct ControlsBar: View {
    @EnvironmentObject var store: WeatherStore
    var showOpenWindow: Bool = false
    var compact: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            if compact {
                // Popover / Compact 2-row layout to prevent horizontal overflow
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
                // Wide Desk window single-row layout
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
                Text("// ALERT: \(err)")
                    .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                    .foregroundStyle(WxTheme.snwRed)
                    .frame(maxWidth: .infinity, alignment: .leading)
            } else if let warn = store.payload?.warning, !warn.isEmpty {
                Text("// ADVISORY: \(warn)")
                    .font(.system(size: 9, weight: .medium, design: .monospaced))
                    .foregroundStyle(WxTheme.snwAmber)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
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
            HStack(spacing: 3.5) {
                Image(systemName: "menubar.arrow.up.rectangle")
                    .font(.system(size: 8.5))
                    .foregroundStyle(WxTheme.snwCyan)
                Text(compact ? "BAR:" : "MENU BAR:")
                    .font(.system(size: 7.5, weight: .semibold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver.opacity(0.85))
                Text(store.menuBarFormat.rawValue.uppercased())
                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwCyan)
                Image(systemName: "chevron.down")
                    .font(.system(size: 6.5, weight: .bold))
                    .foregroundStyle(WxTheme.snwSilver.opacity(0.6))
            }
            .padding(.horizontal, 6)
            .padding(.vertical, 5)
            .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 5))
            .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))
            .foregroundStyle(WxTheme.snwSilver)
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
            if store.isLoading {
                ProgressView()
                    .controlSize(.small)
                    .frame(width: 14, height: 14)
            } else {
                Image(systemName: "arrow.triangle.2.circlepath")
                    .font(.system(size: 10, weight: .bold))
                    .foregroundStyle(WxTheme.snwCyan)
            }
        }
        .buttonStyle(.plain)
        .padding(.horizontal, 7)
        .padding(.vertical, 5)
        .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 5))
        .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))
        .help("Refresh Sensor Telemetry (⌘R)")
        .disabled(store.isLoading)
    }

    private var openRadarButton: some View {
        Button {
            NotificationCenter.default.post(name: .wxOpenDeskRadar, object: nil)
        } label: {
            HStack(spacing: 3) {
                Image(systemName: "dot.radiowaves.left.and.right")
                    .font(.system(size: 8.5))
                Text("RADAR")
                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
            }
            .padding(.horizontal, 6)
            .padding(.vertical, 5)
            .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 5))
            .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
            .foregroundStyle(WxTheme.snwCyan)
        }
        .buttonStyle(.plain)
        .help("Open Radar Tactical Array (⌘3)")
    }

    private var openDeskButton: some View {
        Button {
            NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
        } label: {
            HStack(spacing: 3) {
                Image(systemName: "macwindow")
                    .font(.system(size: 8.5))
                Text("DESK")
                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
            }
            .padding(.horizontal, 6)
            .padding(.vertical, 5)
            .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 5))
            .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
            .foregroundStyle(WxTheme.snwCyan)
        }
        .buttonStyle(.plain)
        .help("Open Tactical Console (⌘1)")
    }
}
