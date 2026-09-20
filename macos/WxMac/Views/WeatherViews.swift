import SwiftUI

struct NowBlockView: View {
    @EnvironmentObject var store: WeatherStore
    var compact: Bool = false

    var body: some View {
        let c = store.payload?.conditions
        VStack(alignment: .leading, spacing: compact ? 6 : 10) {
            HStack(alignment: .firstTextBaseline) {
                Image(systemName: store.statusSymbol)
                    .font(compact ? .title2 : .largeTitle)
                Text(store.displayTemp)
                    .font(compact ? .title : .system(size: 44, weight: .semibold, design: .rounded))
                Spacer()
                if store.isLoading {
                    ProgressView().controlSize(.small)
                }
            }
            Text(c?.location ?? "—")
                .font(.headline)
            if let desc = c?.description, !desc.isEmpty {
                Text(desc).foregroundStyle(.secondary)
            }
            if let station = c?.station, let observed = c?.observedAt {
                Text("Station \(station) · \(observed)")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
            metricsGrid(c)
        }
    }

    @ViewBuilder
    private func metricsGrid(_ c: Conditions?) -> some View {
        let metric = store.units == "metric"
        FlowMetrics {
            if let h = c?.humidityPct { MetricChip(label: "Humidity", value: String(format: "%.0f%%", h)) }
            if metric {
                if let w = c?.windKph {
                    let dir = c?.windDirection.map { " \($0)" } ?? ""
                    MetricChip(label: "Wind", value: String(format: "%.0f km/h%@", w, dir))
                }
                if let g = c?.windGustKph { MetricChip(label: "Gust", value: String(format: "%.0f km/h", g)) }
                if let f = c?.feelsLikeC { MetricChip(label: "Feels", value: String(format: "%.0f°", f)) }
                if let p = c?.pressureHpa { MetricChip(label: "Pressure", value: String(format: "%.0f hPa", p)) }
                if let v = c?.visibilityM { MetricChip(label: "Vis", value: String(format: "%.1f km", v / 1000)) }
            } else {
                if let w = c?.windMph {
                    let dir = c?.windDirection.map { " \($0)" } ?? ""
                    MetricChip(label: "Wind", value: String(format: "%.0f mph%@", w, dir))
                }
                if let g = c?.windGustMph { MetricChip(label: "Gust", value: String(format: "%.0f mph", g)) }
                if let f = c?.feelsLikeF { MetricChip(label: "Feels", value: String(format: "%.0f°", f)) }
                if let p = c?.pressureInhg { MetricChip(label: "Pressure", value: String(format: "%.2f inHg", p)) }
                if let v = c?.visibilityMi { MetricChip(label: "Vis", value: String(format: "%.1f mi", v)) }
            }
        }
    }
}

struct MetricChip: View {
    let label: String
    let value: String
    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label).font(.caption2).foregroundStyle(.secondary)
            Text(value).font(.caption.weight(.medium))
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(.quaternary.opacity(0.5), in: RoundedRectangle(cornerRadius: 6))
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

    var body: some View {
        let periods = Array((store.payload?.forecast?.periods ?? []).prefix(limit ?? .max))
        if periods.isEmpty {
            EmptyView()
        } else {
            VStack(alignment: .leading, spacing: 4) {
                Text("Forecast").font(.headline)
                ForEach(periods) { p in
                    DisclosureGroup {
                        if let detail = p.detailedDescription {
                            Text(detail).font(.caption).foregroundStyle(.secondary)
                        }
                    } label: {
                        HStack {
                            Text(p.name).frame(maxWidth: .infinity, alignment: .leading)
                            Text(tempText(p)).bold()
                            Text(p.shortDescription ?? "")
                                .foregroundStyle(.secondary)
                                .lineLimit(1)
                        }
                        .font(.callout)
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

    var body: some View {
        let alerts = store.payload?.alerts ?? []
        if alerts.isEmpty {
            EmptyView()
        } else {
            VStack(alignment: .leading, spacing: 6) {
                Text("Alerts").font(.headline)
                ForEach(alerts) { a in
                    DisclosureGroup {
                        VStack(alignment: .leading, spacing: 4) {
                            if let d = a.description { Text(d).font(.caption) }
                            if let i = a.instruction { Text(i).font(.caption).foregroundStyle(.secondary) }
                        }
                    } label: {
                        HStack(alignment: .top) {
                            Circle().fill(severityColor(a.severity)).frame(width: 8, height: 8).padding(.top, 5)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(a.event).font(.callout.weight(.semibold))
                                if let h = a.headline { Text(h).font(.caption).foregroundStyle(.secondary).lineLimit(2) }
                                HStack {
                                    if let exp = a.expires { Text("Expires \(exp)").font(.caption2) }
                                    if let area = a.area { Text(area).font(.caption2).lineLimit(1) }
                                }
                                .foregroundStyle(.tertiary)
                            }
                        }
                    }
                }
            }
        }
    }

    private func severityColor(_ s: String?) -> Color {
        switch (s ?? "").lowercased() {
        case "extreme": return .purple
        case "severe": return .red
        case "moderate": return .orange
        case "minor": return .yellow
        default: return .gray
        }
    }
}

struct ControlsBar: View {
    @EnvironmentObject var store: WeatherStore
    var showOpenWindow: Bool = false

    var body: some View {
        VStack(spacing: 8) {
            HStack {
                TextField("Zip or City, ST", text: $store.locationInput)
                    .textFieldStyle(.roundedBorder)
                    .onSubmit { Task { await store.applyLocationAndUnits() } }
                Picker("Units", selection: $store.units) {
                    Text("°F").tag("imperial")
                    Text("°C").tag("metric")
                }
                .pickerStyle(.segmented)
                .frame(width: 100)
                .onChange(of: store.units) { _, _ in
                    Task { await store.applyLocationAndUnits() }
                }
            }
            HStack {
                Button("Refresh") { Task { await store.refresh() } }
                    .disabled(store.isLoading)
                Button("Apply location") { Task { await store.applyLocationAndUnits() } }
                if showOpenWindow {
                    Button("Open Window") {
                        NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
                    }
                }
                Spacer()
                if let t = store.lastRefreshed {
                    Text("Updated \(t.formatted(date: .omitted, time: .shortened))")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
            if let err = store.errorMessage {
                Text(err)
                    .font(.caption)
                    .foregroundStyle(.red)
                    .frame(maxWidth: .infinity, alignment: .leading)
            } else if let warn = store.payload?.warning, !warn.isEmpty {
                Text(warn)
                    .font(.caption2)
                    .foregroundStyle(.orange)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
    }
}
