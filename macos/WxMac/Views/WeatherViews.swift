import SwiftUI

struct NowBlockView: View {
    @EnvironmentObject var store: WeatherStore
    var compact: Bool = false
    /// Popover: at most 4 chips (humidity, wind, feels, pressure).
    var popoverMetrics: Bool = false

    var body: some View {
        let c = store.payload?.conditions
        VStack(alignment: .leading, spacing: compact ? 6 : 10) {
            HStack(alignment: .firstTextBaseline) {
                Image(systemName: store.statusSymbol)
                    .font(compact ? .title2 : .system(size: 38))
                    .foregroundStyle(WxTheme.snwCyan)
                    .shadow(color: WxTheme.snwCyan.opacity(0.4), radius: 5)
                Text(store.displayTemp)
                    .font(compact ? .title.weight(.semibold) : .system(size: 46, weight: .bold, design: .rounded))
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
            if !popoverMetrics, let station = c?.station, let observed = c?.observedAt {
                Text("STATION // \(station) · OBSERVED // \(observed)")
                    .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver.opacity(0.75))
            }
            metricsGrid(c)
        }
    }

    @ViewBuilder
    private func metricsGrid(_ c: Conditions?) -> some View {
        let metric = store.units == "metric"
        FlowMetrics {
            if let h = c?.humidityPct {
                MetricChip(label: "Humidity", value: String(format: "%.0f%%", h))
            }
            if metric {
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
                if !popoverMetrics, let v = c?.visibilityM {
                    MetricChip(label: "Vis", value: String(format: "%.1f km", v / 1000))
                }
            } else {
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
                if !popoverMetrics, let v = c?.visibilityMi {
                    MetricChip(label: "Vis", value: String(format: "%.1f mi", v))
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
    @ViewBuilder var content: Content
    var body: some View {
        LazyVGrid(columns: [GridItem(.adaptive(minimum: 90), spacing: 6)], alignment: .leading, spacing: 6) {
            content
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
                        HStack {
                            Text(p.name)
                                .foregroundStyle(WxTheme.text)
                                .frame(maxWidth: .infinity, alignment: .leading)
                            Text(tempText(p))
                                .fontWeight(.semibold)
                                .foregroundStyle(WxTheme.snwCyan)
                                .frame(width: 44, alignment: .trailing)
                            Text(p.shortDescription ?? "")
                                .foregroundStyle(WxTheme.textSecondary)
                                .lineLimit(1)
                                .frame(maxWidth: 160, alignment: .trailing)
                        }
                        .font(.callout)
                        .frame(height: WxTheme.periodRowHeight)
                    } else {
                        DisclosureGroup {
                            if let detail = p.detailedDescription {
                                Text(detail)
                                    .font(.caption)
                                    .foregroundStyle(WxTheme.textSecondary)
                                    .padding(.vertical, 4)
                            }
                        } label: {
                            HStack {
                                Text(p.name)
                                    .font(.system(.body, design: .rounded))
                                    .foregroundStyle(WxTheme.text)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                Text(tempText(p))
                                    .font(.system(.body, design: .monospaced).weight(.bold))
                                    .foregroundStyle(WxTheme.snwCyan)
                                Text(p.shortDescription ?? "")
                                    .font(.caption)
                                    .foregroundStyle(WxTheme.textSecondary)
                                    .lineLimit(1)
                            }
                            .font(.callout)
                        }
                        .tint(WxTheme.snwCyan.opacity(0.8))
                    }
                }
            }
        }
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
                    Circle()
                        .fill(WxTheme.snwRed)
                        .frame(width: 6, height: 6)
                        .shadow(color: WxTheme.snwRed.opacity(0.8), radius: 3)
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
        VStack(spacing: compact ? 6 : 8) {
            HStack(spacing: 8) {
                HStack(spacing: 6) {
                    Image(systemName: "scope")
                        .font(.system(size: 11))
                        .foregroundStyle(WxTheme.snwCyan.opacity(0.8))
                    TextField("Zip or City, ST", text: $store.locationInput)
                        .textFieldStyle(.plain)
                        .font(.system(size: 12, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.text)
                        .onSubmit { Task { await store.applyLocationAndUnits() } }
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 6)
                .background(
                    RoundedRectangle(cornerRadius: 6, style: .continuous)
                        .fill(WxTheme.snwChassis.opacity(0.85))
                        .overlay(
                            RoundedRectangle(cornerRadius: 6, style: .continuous)
                                .strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8)
                        )
                )

                Picker("Units", selection: $store.units) {
                    Text("°F").tag("imperial")
                    Text("°C").tag("metric")
                }
                .pickerStyle(.segmented)
                .labelsHidden()
                .frame(width: 82)
                .onChange(of: store.units) { _, _ in
                    Task { await store.applyLocationAndUnits() }
                }

                if showOpenWindow {
                    Button {
                        NotificationCenter.default.post(name: .wxOpenDeskRadar, object: nil)
                    } label: {
                        Image(systemName: "dot.radiowaves.left.and.right")
                            .foregroundStyle(WxTheme.snwCyan)
                    }
                    .buttonStyle(.plain)
                    .help("Open Radar Array")

                    Button {
                        NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                    } label: {
                        Image(systemName: "macwindow")
                            .foregroundStyle(WxTheme.snwCyan)
                    }
                    .buttonStyle(.plain)
                    .help("Open Desk Console")
                }
            }

            HStack {
                Button {
                    Task { await store.refresh() }
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 9))
                        Text("REFRESH SENSORS")
                            .font(.system(size: 9, weight: .bold, design: .monospaced))
                    }
                    .padding(.horizontal, 7)
                    .padding(.vertical, 3.5)
                    .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.35), lineWidth: 0.8))
                    .foregroundStyle(WxTheme.snwCyan)
                }
                .buttonStyle(.plain)
                .disabled(store.isLoading)

                if !compact {
                    Button {
                        Task { await store.applyLocationAndUnits() }
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "location.fill")
                                .font(.system(size: 9))
                            Text("LOCK LOCATION")
                                .font(.system(size: 9, weight: .bold, design: .monospaced))
                        }
                        .padding(.horizontal, 7)
                        .padding(.vertical, 3.5)
                        .background(WxTheme.snwGold.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwGold.opacity(0.35), lineWidth: 0.8))
                        .foregroundStyle(WxTheme.snwGold)
                    }
                    .buttonStyle(.plain)
                }

                Spacer()

                if let t = store.lastRefreshed {
                    Text("SYNC // \(t.formatted(date: .omitted, time: .shortened))")
                        .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.75))
                }
            }

            if let err = store.errorMessage {
                Text("// ALERT: \(err)")
                    .font(.system(size: 10, weight: .medium, design: .monospaced))
                    .foregroundStyle(WxTheme.snwRed)
                    .frame(maxWidth: .infinity, alignment: .leading)
            } else if let warn = store.payload?.warning, !warn.isEmpty {
                Text("// ADVISORY: \(warn)")
                    .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                    .foregroundStyle(WxTheme.snwAmber)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
    }
}
