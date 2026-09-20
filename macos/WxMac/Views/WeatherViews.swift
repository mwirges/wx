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
                    .font(compact ? .title2 : .largeTitle)
                    .foregroundStyle(WxTheme.accent)
                Text(store.displayTemp)
                    .font(compact ? .title.weight(.semibold) : .system(size: 44, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxTheme.text)
                Spacer()
                if store.isLoading {
                    ProgressView().controlSize(.small).tint(WxTheme.accent)
                }
            }
            Text(c?.location ?? "—")
                .font(.headline)
                .foregroundStyle(WxTheme.text)
            if let desc = c?.description, !desc.isEmpty {
                Text(desc).foregroundStyle(WxTheme.textSecondary)
            }
            if !popoverMetrics, let station = c?.station, let observed = c?.observedAt {
                Text("Station \(station) · \(observed)")
                    .font(.caption)
                    .foregroundStyle(WxTheme.textSecondary.opacity(0.8))
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
        VStack(alignment: .leading, spacing: 2) {
            Text(label).font(.caption2).foregroundStyle(WxTheme.textSecondary)
            Text(value).font(.caption.weight(.medium)).foregroundStyle(WxTheme.text)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(
            Capsule(style: .continuous)
                .fill(WxTheme.accent.opacity(0.10))
                .overlay(Capsule(style: .continuous).strokeBorder(WxTheme.border, lineWidth: 1))
        )
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
                Text("Forecast")
                    .font(.headline)
                    .foregroundStyle(WxTheme.text)
                ForEach(periods) { p in
                    if compactRows {
                        HStack {
                            Text(p.name)
                                .foregroundStyle(WxTheme.text)
                                .frame(maxWidth: .infinity, alignment: .leading)
                            Text(tempText(p))
                                .fontWeight(.semibold)
                                .foregroundStyle(WxTheme.accent)
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
                                Text(detail).font(.caption).foregroundStyle(WxTheme.textSecondary)
                            }
                        } label: {
                            HStack {
                                Text(p.name).frame(maxWidth: .infinity, alignment: .leading)
                                    .foregroundStyle(WxTheme.text)
                                Text(tempText(p)).bold().foregroundStyle(WxTheme.accent)
                                Text(p.shortDescription ?? "")
                                    .foregroundStyle(WxTheme.textSecondary)
                                    .lineLimit(1)
                            }
                            .font(.callout)
                        }
                        .tint(WxTheme.accentSecondary)
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
    /// Popover: ≤2 collapsed rows; overflow → Open Desk.
    var popoverMode: Bool = false

    var body: some View {
        let alerts = store.payload?.alerts ?? []
        if alerts.isEmpty {
            EmptyView()
        } else if popoverMode {
            let shown = Array(alerts.prefix(2))
            let extra = alerts.count - shown.count
            VStack(alignment: .leading, spacing: 6) {
                Text("Alerts").font(.headline).foregroundStyle(WxTheme.text)
                ForEach(shown) { a in
                    HStack(alignment: .top, spacing: 8) {
                        AlertSeverityGlyph(severity: a.severity, size: 18)
                            .padding(.top, 2)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(a.event)
                                .font(.callout.weight(.semibold))
                                .foregroundStyle(WxTheme.text)
                                .lineLimit(1)
                            if let h = a.headline {
                                Text(h).font(.caption).foregroundStyle(WxTheme.textSecondary).lineLimit(1)
                            }
                        }
                    }
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
            VStack(alignment: .leading, spacing: 6) {
                Text("Alerts").font(.headline).foregroundStyle(WxTheme.text)
                ForEach(alerts) { a in
                    DisclosureGroup {
                        VStack(alignment: .leading, spacing: 4) {
                            if let d = a.description { Text(d).font(.caption).foregroundStyle(WxTheme.text) }
                            if let i = a.instruction { Text(i).font(.caption).foregroundStyle(WxTheme.textSecondary) }
                        }
                    } label: {
                        HStack(alignment: .top) {
                            AlertSeverityGlyph(severity: a.severity, size: 20)
                                .padding(.top, 1)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(a.event).font(.callout.weight(.semibold)).foregroundStyle(WxTheme.text)
                                if let h = a.headline {
                                    Text(h).font(.caption).foregroundStyle(WxTheme.textSecondary).lineLimit(2)
                                }
                                HStack {
                                    if let exp = a.expires { Text("Expires \(exp)").font(.caption2) }
                                    if let area = a.area { Text(area).font(.caption2).lineLimit(1) }
                                }
                                .foregroundStyle(WxTheme.textSecondary.opacity(0.8))
                            }
                        }
                    }
                    .tint(WxTheme.accentSecondary)
                }
            }
        }
    }
}

struct ControlsBar: View {
    @EnvironmentObject var store: WeatherStore
    var showOpenWindow: Bool = false
    var compact: Bool = false

    var body: some View {
        VStack(spacing: compact ? 6 : 8) {
            HStack(spacing: 8) {
                TextField("Zip or City, ST", text: $store.locationInput)
                    .textFieldStyle(.plain)
                    .padding(6)
                    .background(
                        RoundedRectangle(cornerRadius: 8, style: .continuous)
                            .fill(WxTheme.accent.opacity(0.08))
                            .overlay(
                                RoundedRectangle(cornerRadius: 8, style: .continuous)
                                    .strokeBorder(WxTheme.border, lineWidth: 1)
                            )
                    )
                    .foregroundStyle(WxTheme.text)
                    .onSubmit { Task { await store.applyLocationAndUnits() } }
                Picker("Units", selection: $store.units) {
                    Text("°F").tag("imperial")
                    Text("°C").tag("metric")
                }
                .pickerStyle(.segmented)
                .labelsHidden()
                .frame(width: 90)
                .onChange(of: store.units) { _, _ in
                    Task { await store.applyLocationAndUnits() }
                }
                if showOpenWindow {
                    Button {
                        NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                    } label: {
                        Image(systemName: "macwindow")
                            .foregroundStyle(WxTheme.accent)
                    }
                    .buttonStyle(.plain)
                    .help("Open Desk")
                }
            }
            HStack {
                Button("Refresh") { Task { await store.refresh() } }
                    .disabled(store.isLoading)
                    .foregroundStyle(WxTheme.accent)
                if !compact {
                    Button("Apply location") { Task { await store.applyLocationAndUnits() } }
                        .foregroundStyle(WxTheme.accentSecondary)
                }
                Spacer()
                if let t = store.lastRefreshed {
                    Text("Updated \(t.formatted(date: .omitted, time: .shortened))")
                        .font(.caption2)
                        .foregroundStyle(WxTheme.textSecondary)
                }
            }
            if let err = store.errorMessage {
                Text(err)
                    .font(.caption)
                    .foregroundStyle(WxTheme.alert)
                    .frame(maxWidth: .infinity, alignment: .leading)
            } else if let warn = store.payload?.warning, !warn.isEmpty {
                Text(warn)
                    .font(.caption2)
                    .foregroundStyle(WxTheme.warn)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
    }
}
