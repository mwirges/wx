import SwiftUI
import UIKit

struct TacticalView: View {
    @EnvironmentObject private var store: IOSStore

    var body: some View {
        let conditions = store.payload?.conditions
        let air = conditions?.airQuality ?? store.payload?.airQuality
        let astro = conditions?.astronomy ?? store.payload?.astronomy
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                Text("Tactical sensors")
                    .font(.system(size: 13, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwCyan)
                    .accessibilityIdentifier("tactical.root")
                if let error = store.surfaceError, store.payload == nil {
                    Text(error)
                        .foregroundStyle(WxTheme.snwRed)
                        .accessibilityIdentifier("wx.error")
                }
                SNWConsoleCard(title: "Sensors", tag: conditions?.station) {
                    LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 8) {
                        tile("Humidity", conditions?.humidityPct.map { String(format: "%.0f%%", $0) } ?? "—")
                        tile("Wind", wind(conditions))
                        tile("Pressure", pressure(conditions))
                        tile("Dew point", dew(conditions))
                        tile("AQI", air.map { aqi($0) } ?? "—")
                        tile("UV", air?.uvIndex.map { String(format: "%.1f", $0) } ?? "—")
                        tile("Moon", astro?.moonPhase ?? "—")
                        tile("Vis", visibility(conditions))
                    }
                }
            }
            .padding(12)
            .accessibilityElement(children: .contain)
        }
    }

    private func tile(_ label: String, _ value: String) -> some View {
        SNWMetricTile(label: label, value: value)
    }

    private func wind(_ conditions: Conditions?) -> String {
        let direction = conditions?.windDirection ?? ""
        if store.units == "metric", let speed = conditions?.windKph {
            return String(format: "%.0f km/h", speed) + (direction.isEmpty ? "" : " \(direction)")
        }
        if let speed = conditions?.windMph {
            return String(format: "%.0f mph", speed) + (direction.isEmpty ? "" : " \(direction)")
        }
        return "—"
    }

    private func pressure(_ conditions: Conditions?) -> String {
        if store.units == "metric", let hpa = conditions?.pressureHpa {
            return String(format: "%.0f hPa", hpa)
        }
        if let inches = conditions?.pressureInhg {
            return String(format: "%.2f inHg", inches)
        }
        return "—"
    }

    private func dew(_ conditions: Conditions?) -> String {
        IOSFormat.temperature(f: conditions?.dewPointF, c: conditions?.dewPointC, units: store.units)
    }

    private func visibility(_ conditions: Conditions?) -> String {
        if store.units == "metric", let meters = conditions?.visibilityM {
            return String(format: "%.1f km", meters / 1000)
        }
        if let miles = conditions?.visibilityMi {
            return String(format: "%.1f mi", miles)
        }
        return "—"
    }

    private func aqi(_ air: AirQualityDTO) -> String {
        guard let value = air.aqi else { return "—" }
        if let category = air.category, !category.isEmpty {
            return "\(value) \(category)"
        }
        return "\(value)"
    }
}

struct RadarTabView: View {
    @EnvironmentObject private var store: IOSStore

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                if store.radarLoading {
                    ProgressView().tint(WxTheme.snwCyan)
                }
                if let error = store.radarError, !error.isEmpty {
                    Text(error)
                        .foregroundStyle(WxTheme.snwRed)
                        .accessibilityIdentifier("wx.error")
                }
                if let radar = store.radar {
                    SNWConsoleCard(title: "Radar", tag: radar.station) {
                        Text(radar.productLabel)
                            .font(.system(size: 16, weight: .semibold))
                            .foregroundStyle(WxTheme.text)
                            .accessibilityIdentifier("radar.product")
                        Text(radar.location)
                            .font(.system(size: 13))
                            .foregroundStyle(WxTheme.textSecondary)
                        radarImage(radar.imageBase64)
                        if !radar.frames.isEmpty {
                            Text("\(radar.frames.count) frames")
                                .font(.system(size: 12, design: .monospaced))
                                .foregroundStyle(WxTheme.textSecondary)
                        }
                    }
                } else if !store.radarLoading && store.radarError == nil {
                    Text("Radar idle")
                        .foregroundStyle(WxTheme.textSecondary)
                }
            }
            .padding(12)
            .accessibilityElement(children: .contain)
        }
        .accessibilityIdentifier("radar.root")
    }

    @ViewBuilder
    private func radarImage(_ b64: String) -> some View {
        if let data = Data(base64Encoded: b64, options: .ignoreUnknownCharacters),
           let image = UIImage(data: data) {
            Image(uiImage: image)
                .resizable()
                .interpolation(.none)
                .scaledToFit()
                .frame(maxWidth: .infinity, maxHeight: 240)
                .accessibilityIdentifier("radar.image")
        }
    }
}

struct SevereView: View {
    @EnvironmentObject private var store: IOSStore

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                if store.severeLoading {
                    ProgressView().tint(WxTheme.snwCyan)
                }
                if let error = store.severeError, !error.isEmpty {
                    Text(error)
                        .foregroundStyle(WxTheme.snwRed)
                        .accessibilityIdentifier("wx.error")
                }
                if let summary = store.spc?.convectiveSummary, !summary.isEmpty {
                    Text(summary)
                        .font(.system(size: 18, weight: .semibold))
                        .foregroundStyle(WxTheme.text)
                        .accessibilityIdentifier("severe.summary")
                }
                if let day = store.spc?.day1 {
                    SNWConsoleCard(title: "Day 1", tag: day.category.code, statusColor: WxTheme.snwGold) {
                        Text(day.category.name)
                            .font(.headline)
                            .foregroundStyle(WxTheme.text)
                        Text(day.category.description)
                            .font(.system(size: 13))
                            .foregroundStyle(WxTheme.textSecondary)
                    }
                }
                let alerts = store.payload?.alerts ?? []
                if !alerts.isEmpty {
                    SNWConsoleCard(title: "Alerts", tag: "\(alerts.count)", statusColor: WxTheme.snwRed) {
                        ForEach(alerts) { alert in
                            Text(alert.event)
                                .foregroundStyle(WxTheme.text)
                        }
                    }
                }
            }
            .padding(12)
            .accessibilityElement(children: .contain)
        }
        .accessibilityIdentifier("severe.root")
    }
}

struct TropicsTabView: View {
    @EnvironmentObject private var store: IOSStore

    var body: some View {
        let storms = store.tropics?.tropics?.storms ?? []
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                if store.tropicsLoading {
                    ProgressView().tint(WxTheme.snwCyan)
                }
                if let error = store.tropicsError, !error.isEmpty {
                    Text(error)
                        .foregroundStyle(WxTheme.snwRed)
                        .accessibilityIdentifier("wx.error")
                }
                if storms.isEmpty && !store.tropicsLoading && store.tropicsError == nil && store.tropics != nil {
                    Text("No active storms")
                        .foregroundStyle(WxTheme.textSecondary)
                }
                ForEach(storms) { storm in
                    SNWConsoleCard(title: storm.classification, tag: storm.categoryLabel, statusColor: WxTheme.snwAmber) {
                        Text(storm.name)
                            .font(.system(size: 18, weight: .semibold))
                            .foregroundStyle(WxTheme.text)
                            .accessibilityIdentifier("tropics.storm")
                        Text("\(storm.windSpeedMph) mph · \(storm.pressureMb) mb")
                            .font(.system(size: 13, design: .monospaced))
                            .foregroundStyle(WxTheme.textSecondary)
                        if let headline = storm.headline, !headline.isEmpty {
                            Text(headline)
                                .font(.system(size: 13))
                                .foregroundStyle(WxTheme.textSecondary)
                        }
                    }
                }
            }
            .padding(12)
            .accessibilityElement(children: .contain)
        }
        .accessibilityIdentifier("tropics.root")
    }
}

struct ConfigView: View {
    @EnvironmentObject private var store: IOSStore
    @State private var draft: String = ""
    @State private var units: String = "imperial"

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                Text("Saved place")
                    .font(.system(size: 13, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwCyan)
                    .accessibilityIdentifier("config.saved")
                TextField("City, ST", text: $draft)
                    .textFieldStyle(.roundedBorder)
                    .textInputAutocapitalization(.words)
                    .autocorrectionDisabled()
                    .accessibilityIdentifier("config.place")
                Picker("Units", selection: $units) {
                    Text("Imperial").tag("imperial")
                    Text("Metric").tag("metric")
                }
                .pickerStyle(.segmented)
                .accessibilityIdentifier("config.units")
                Button("Save place") {
                    store.save(place: draft, units: units)
                }
                .buttonStyle(.borderedProminent)
                .tint(WxTheme.snwCyan)
                .accessibilityIdentifier("config.save")
                Button("Use current location") {
                    Task {
                        await store.useCurrentLocation()
                        draft = store.place
                    }
                }
                .buttonStyle(.bordered)
                .accessibilityIdentifier("config.locate")
                Text("Location is one shot, only while the app is in use. Denying it keeps this saved place.")
                    .font(.system(size: 12))
                    .foregroundStyle(WxTheme.textSecondary)
            }
            .padding(12)
            .accessibilityElement(children: .contain)
        }
        .accessibilityIdentifier("config.root")
        .onAppear {
            draft = store.place
            units = store.units
        }
        .onChange(of: store.place) { _, place in
            draft = place
        }
    }
}
