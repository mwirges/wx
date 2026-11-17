import SwiftUI
import UIKit

// MARK: - Tactical Sensors & Solar Ephemeris Tab

struct TacticalView: View {
    @EnvironmentObject private var store: IOSStore

    var body: some View {
        let conditions = store.payload?.conditions
        let air = conditions?.airQuality ?? store.payload?.airQuality
        let astro = conditions?.astronomy ?? store.payload?.astronomy

        ScrollView(showsIndicators: false) {
            VStack(alignment: .leading, spacing: 14) {
                HStack(spacing: 8) {
                    Image(systemName: "safari.fill")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(WxGlassTheme.cyan)
                    Text("Tactical sensors")
                        .font(.system(size: 16, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                        .accessibilityIdentifier("tactical.root")
                }
                .padding(.horizontal, 2)

                if let error = store.surfaceError, store.payload == nil {
                    Text(error)
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(WxGlassTheme.conditionRed)
                        .accessibilityIdentifier("wx.error")
                }

                // 1. Solar Orbit & Lunar Ephemeris Card
                if let astro {
                    solarEphemerisCard(astro)
                }

                // 2. Concentric Atmospheric Sensor Grid
                WxGlassCard(title: "Surface Sensor Array", tag: conditions?.station, accent: WxGlassTheme.cyan) {
                    LazyVGrid(columns: [GridItem(.flexible(), spacing: 10), GridItem(.flexible(), spacing: 10)], spacing: 10) {
                        WxSensorCapsule(
                            label: "Humidity",
                            value: conditions?.humidityPct.map { String(format: "%.0f%%", $0) } ?? "—",
                            subtitle: dewSubtitle(conditions),
                            symbol: "drop.fill",
                            accent: WxGlassTheme.cyan
                        )
                        WxSensorCapsule(
                            label: "Wind",
                            value: wind(conditions),
                            subtitle: gustSubtitle(conditions),
                            symbol: "wind",
                            accent: WxGlassTheme.cyan
                        )
                        WxSensorCapsule(
                            label: "Pressure",
                            value: pressure(conditions),
                            subtitle: "Barometric",
                            symbol: "gauge.with.dots.needle.bottom.50percent",
                            accent: WxGlassTheme.periwinkle
                        )
                        WxSensorCapsule(
                            label: "Dew point",
                            value: dew(conditions),
                            subtitle: "Thermal comfort",
                            symbol: "thermometer.medium",
                            accent: WxGlassTheme.cyanSoft
                        )
                        WxSensorCapsule(
                            label: "AQI",
                            value: air.map { aqi($0) } ?? "—",
                            subtitle: air?.category,
                            symbol: "aqi.medium",
                            accent: WxTheme.aqiColor(air?.aqi)
                        )
                        WxSensorCapsule(
                            label: "UV",
                            value: air?.uvIndex.map { String(format: "%.1f", $0) } ?? "—",
                            subtitle: air?.uvCategory ?? "Solar UV",
                            symbol: "sun.max.trianglebadge.exclamationmark",
                            accent: WxGlassTheme.gold
                        )
                        WxSensorCapsule(
                            label: "Moon",
                            value: astro?.moonPhase ?? "—",
                            subtitle: astro?.moonIlluminationPct.map { String(format: "%.0f%% illum", $0) },
                            symbol: ConditionSymbol.moonSymbol(for: astro?.moonPhase),
                            accent: WxGlassTheme.gold
                        )
                        WxSensorCapsule(
                            label: "Vis",
                            value: visibility(conditions),
                            subtitle: "Horizontal range",
                            symbol: "eye.fill",
                            accent: WxGlassTheme.cyan
                        )
                    }
                }

                // 3. Air Quality Spectrum & Health Guidance Card
                if let air, let aqiVal = air.aqi {
                    airQualityCard(air: air, aqiVal: aqiVal)
                }
            }
            .padding(.horizontal, 14)
            .padding(.top, 6)
            .padding(.bottom, 8)
            .accessibilityElement(children: .contain)
        }
        .wxDockScrollClearance()
    }

    private func solarEphemerisCard(_ astro: AstronomyDTO) -> some View {
        let regime = astro.currentPeriod ?? (astro.isPolarDay == true ? "Polar Day" : (astro.isPolarNight == true ? "Polar Night" : "Solar Orbit"))
        return WxGlassCard(title: "Solar & Lunar Ephemeris", tag: regime, accent: WxGlassTheme.gold) {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .firstTextBaseline) {
                    if let elev = astro.solarElevationDeg {
                        let sign = elev >= 0 ? "+" : ""
                        Text("\(sign)\(String(format: "%.1f°", elev))")
                            .font(.system(size: 24, weight: .bold, design: .rounded))
                            .foregroundStyle(elev >= 0 ? WxGlassTheme.gold : WxGlassTheme.cyan)
                        if let az = astro.solarAzimuthDeg {
                            Text("\(String(format: "%.0f°", az)) \(astro.solarAzimuthCompass)")
                                .font(.system(size: 12, weight: .semibold, design: .monospaced))
                                .foregroundStyle(WxGlassTheme.textSecondary)
                        }
                    } else if let dl = astro.dayLength {
                        Text("\(dl) daylight")
                            .font(.system(size: 16, weight: .semibold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textPrimary)
                    }
                    Spacer()
                    HStack(spacing: 5) {
                        Image(systemName: ConditionSymbol.moonSymbol(for: astro.moonPhase))
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(WxGlassTheme.gold)
                        Text(astro.moonPhase ?? "Lunar")
                            .font(.system(size: 12, weight: .semibold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textSecondary)
                            .lineLimit(1)
                    }
                }

                WxSolarOrbitGraph(
                    solarElevationDeg: astro.solarElevationDeg,
                    solarAzimuthDeg: astro.solarAzimuthDeg
                )

                HStack(spacing: 8) {
                    if let sr = astro.sunriseFormatted {
                        miniAstroPill(label: "Rise", value: sr, symbol: "sunrise.fill", color: WxGlassTheme.gold)
                    }
                    if let sn = astro.solarNoonFormatted {
                        miniAstroPill(label: "Noon", value: sn, symbol: "sun.max.fill", color: WxGlassTheme.gold)
                    }
                    if let ss = astro.sunsetFormatted {
                        miniAstroPill(label: "Set", value: ss, symbol: "sunset.fill", color: WxGlassTheme.amber)
                    }
                    if let dl = astro.dayLength {
                        miniAstroPill(label: "Span", value: dl, symbol: "clock.fill", color: WxGlassTheme.cyan)
                    }
                }
            }
        }
    }

    private func miniAstroPill(label: String, value: String, symbol: String, color: Color) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            HStack(spacing: 3) {
                Image(systemName: symbol)
                    .font(.system(size: 9.5, weight: .semibold))
                    .foregroundStyle(color)
                Text(label.uppercased())
                    .font(.system(size: 9, weight: .bold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textMuted)
            }
            Text(value)
                .font(.system(size: 12, weight: .bold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)
                .lineLimit(1)
                .minimumScaleFactor(0.75)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 8)
        .padding(.vertical, 6)
        .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
        .overlay(WxGlassTheme.nestedShape().strokeBorder(Color.white.opacity(0.10), lineWidth: 0.7))
    }

    private func airQualityCard(air: AirQualityDTO, aqiVal: Int) -> some View {
        let color = WxTheme.aqiColor(aqiVal)
        return WxGlassCard(title: "Air Quality Index", tag: air.category ?? "EPA.AQI", accent: color) {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    Text("\(aqiVal)")
                        .font(.system(size: 28, weight: .bold, design: .rounded))
                        .foregroundStyle(color)
                    Text("/ 500 US EPA")
                        .font(.system(size: 12, weight: .medium, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textMuted)
                    Spacer()
                }

                GeometryReader { geo in
                    let w = geo.size.width
                    let ratio = min(1.0, max(0.0, Double(aqiVal) / 500.0))
                    let x = max(5.0, min(w - 5.0, w * ratio))
                    ZStack(alignment: .leading) {
                        Capsule()
                            .fill(
                                LinearGradient(
                                    colors: [WxGlassTheme.nominalGreen, WxGlassTheme.amber, .orange, WxGlassTheme.conditionRed, .purple],
                                    startPoint: .leading,
                                    endPoint: .trailing
                                )
                            )
                            .frame(height: 6)
                        Circle()
                            .fill(Color.white)
                            .frame(width: 11, height: 11)
                            .shadow(color: .black.opacity(0.5), radius: 2)
                            .position(x: x, y: 5.5)
                    }
                }
                .frame(height: 11)

                Text(air.resolvedAdvisory)
                    .font(.system(size: 12.5))
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
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

    private func gustSubtitle(_ conditions: Conditions?) -> String? {
        if store.units == "metric", let gust = conditions?.windGustKph {
            return String(format: "Gusts %.0f km/h", gust)
        }
        if let gust = conditions?.windGustMph {
            return String(format: "Gusts %.0f mph", gust)
        }
        return nil
    }

    private func dewSubtitle(_ conditions: Conditions?) -> String? {
        let d = dew(conditions)
        return d == "—" ? nil : "Dew pt \(d)"
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

// MARK: - Full-Viewport Interactive Radar Tab

private struct IOSRadarProductItem: Identifiable {
    let id: String
    let short: String
    let full: String
}

private let iosRadarProducts: [IOSRadarProductItem] = [
    IOSRadarProductItem(id: "composite-reflectivity", short: "Composite", full: "Composite Reflectivity"),
    IOSRadarProductItem(id: "base-reflectivity", short: "Base", full: "Base Reflectivity"),
    IOSRadarProductItem(id: "storm-relative-velocity", short: "Velocity", full: "Storm-Relative Velocity"),
    IOSRadarProductItem(id: "echo-tops", short: "Echo Tops", full: "Echo Tops"),
    IOSRadarProductItem(id: "precip-type", short: "Precip Type", full: "Precipitation Type"),
    IOSRadarProductItem(id: "one-hour-precip", short: "1-Hr Precip", full: "1-Hour Precipitation"),
    IOSRadarProductItem(id: "storm-total-precip", short: "Storm Total", full: "Storm Total Precip"),
]

private let iosRadarRadii: [(Double, String)] = [
    (150, "150 km Local"),
    (200, "200 km Metro"),
    (500, "500 km Regional"),
    (1000, "1000 km Synoptic"),
]

struct RadarTabView: View {
    @EnvironmentObject private var store: IOSStore
    @State private var recenterID = 0

    var body: some View {
        ZStack {
            // 1. Full-Viewport MapKit / Scope Radar Stage (extends behind floating glass dock)
            radarStage
                .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 24, style: .continuous)
                        .strokeBorder(WxGlassTheme.specularRim(accent: WxGlassTheme.cyan), lineWidth: 1)
                )
                .shadow(color: Color.black.opacity(0.38), radius: 18, y: 8)
                .padding(.horizontal, 10)
                .padding(.top, 4)
                .padding(.bottom, 8)

            // 2. Floating Top & Bottom Liquid Glass Radar Chrome
            VStack(spacing: 8) {
                floatingRadarHeader
                    .padding(.horizontal, 18)
                    .padding(.top, 12)

                if let error = store.radarError, !error.isEmpty {
                    Text(error)
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(WxGlassTheme.conditionRed)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 6)
                        .wxFrostedCapsule(accent: WxGlassTheme.conditionRed)
                        .accessibilityIdentifier("wx.error")
                }

                Spacer(minLength: 8)

                if store.radar != nil {
                    floatingRadarTransport
                        .padding(.horizontal, 18)
                        .padding(.bottom, 74)
                }
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .accessibilityElement(children: .contain)
        .accessibilityIdentifier("radar.root")
    }

    @ViewBuilder
    private var radarStage: some View {
        if let radar = store.radar,
           radar.bbox != nil,
           radar.center != nil,
           let cgImage = store.currentRadarImage {
            IOSRadarMapView(
                payload: radar,
                cgImage: cgImage,
                recenterID: recenterID,
                viewportRevision: store.radarViewportRevision,
                targetRadiusKm: store.selectedRadarRadius,
                onBBoxNeedsUpdate: { bbox in
                    Task { await store.loadRadar(bbox: bbox) }
                }
            )
            .accessibilityIdentifier("radar.image")
        } else if let cgImage = store.currentRadarImage {
            ZStack {
                tacticalScopeBackdrop
                Image(uiImage: UIImage(cgImage: cgImage))
                    .resizable()
                    .interpolation(.none)
                    .scaledToFit()
                    .padding(24)
                    .accessibilityIdentifier("radar.image")
            }
        } else if store.radarLoading {
            ZStack {
                tacticalScopeBackdrop
                VStack(spacing: 10) {
                    ProgressView()
                        .tint(WxGlassTheme.cyan)
                    Text("Acquiring Doppler Radar…")
                        .font(.system(size: 13, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.cyanSoft)
                }
            }
        } else {
            ZStack {
                tacticalScopeBackdrop
                VStack(spacing: 8) {
                    Image(systemName: "dot.radiowaves.left.and.right")
                        .font(.system(size: 28, weight: .medium))
                        .foregroundStyle(WxGlassTheme.cyan.opacity(0.7))
                    Text("Radar idle")
                        .font(.system(size: 14, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                }
            }
        }
    }

    private var tacticalScopeBackdrop: some View {
        Canvas { ctx, size in
            let rect = CGRect(origin: .zero, size: size)
            ctx.fill(
                Path(rect),
                with: .linearGradient(
                    Gradient(colors: [
                        Color(red: 0.03, green: 0.07, blue: 0.14),
                        Color(red: 0.05, green: 0.11, blue: 0.21),
                    ]),
                    startPoint: .zero,
                    endPoint: CGPoint(x: 0, y: size.height)
                )
            )
            let center = CGPoint(x: size.width * 0.5, y: size.height * 0.5)
            let maxR = min(size.width, size.height) * 0.42
            for frac in [0.33, 0.66, 1.0] {
                let r = maxR * frac
                let ring = CGRect(x: center.x - r, y: center.y - r, width: r * 2, height: r * 2)
                ctx.stroke(Path(ellipseIn: ring), with: .color(WxGlassTheme.cyan.opacity(0.14)), lineWidth: 0.8)
            }
            var cross = Path()
            cross.move(to: CGPoint(x: center.x - maxR, y: center.y))
            cross.addLine(to: CGPoint(x: center.x + maxR, y: center.y))
            cross.move(to: CGPoint(x: center.x, y: center.y - maxR))
            cross.addLine(to: CGPoint(x: center.x, y: center.y + maxR))
            ctx.stroke(cross, with: .color(WxGlassTheme.cyan.opacity(0.10)), style: StrokeStyle(lineWidth: 0.8, dash: [4, 4]))
        }
        .allowsHitTesting(false)
    }

    @ViewBuilder
    private var floatingRadarHeader: some View {
        VStack(alignment: .leading, spacing: 6) {
            if let radar = store.radar {
                HStack(spacing: 8) {
                    Circle()
                        .fill(WxGlassTheme.nominalGreen)
                        .frame(width: 6, height: 6)
                        .shadow(color: WxGlassTheme.nominalGreen.opacity(0.8), radius: 3)

                    VStack(alignment: .leading, spacing: 1) {
                        Text(radar.productLabel)
                            .font(.system(size: 14, weight: .bold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textPrimary)
                            .lineLimit(1)
                            .accessibilityIdentifier("radar.product")
                        Text(radar.location)
                            .font(.system(size: 11.5, weight: .medium, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textSecondary)
                            .lineLimit(1)
                    }

                    Spacer(minLength: 4)

                    if let station = radar.station, !station.isEmpty {
                        Text(station.uppercased())
                            .font(.system(size: 10.5, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxGlassTheme.cyanSoft)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(WxGlassTheme.cyan.opacity(0.16), in: Capsule())
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .wxFrostedCapsule(accent: WxGlassTheme.cyan)
            }

            // Product, Scale, Recenter & Refresh Pills
            HStack(spacing: 6) {
                Menu {
                    ForEach(iosRadarProducts) { item in
                        Button {
                            store.setRadarProduct(item.id)
                        } label: {
                            HStack {
                                Text(item.full)
                                if store.selectedRadarProduct == item.id {
                                    Image(systemName: "checkmark")
                                }
                            }
                        }
                    }
                } label: {
                    let activeName = iosRadarProducts.first(where: { $0.id == store.selectedRadarProduct })?.short ?? "Composite"
                    HStack(spacing: 4) {
                        Image(systemName: "antenna.radiowaves.left.and.right")
                            .font(.system(size: 10, weight: .semibold))
                            .foregroundStyle(WxGlassTheme.cyan)
                        Text(activeName)
                            .font(.system(size: 11, weight: .semibold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textPrimary)
                        Image(systemName: "chevron.down")
                            .font(.system(size: 8.5, weight: .bold))
                            .foregroundStyle(WxGlassTheme.textSecondary)
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                }

                Menu {
                    ForEach(iosRadarRadii, id: \.0) { radius, label in
                        Button {
                            store.setRadarRadius(radius)
                        } label: {
                            HStack {
                                Text(label)
                                if abs(store.selectedRadarRadius - radius) < 1 {
                                    Image(systemName: "checkmark")
                                }
                            }
                        }
                    }
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "circle.dashed")
                            .font(.system(size: 10, weight: .semibold))
                            .foregroundStyle(WxGlassTheme.cyanSoft)
                        Text("\(Int(store.selectedRadarRadius)) km")
                            .font(.system(size: 11, weight: .semibold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textPrimary)
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                }

                Spacer(minLength: 2)

                Button {
                    recenterID += 1
                    if store.radarHasCustomBBox {
                        Task { await store.loadRadar() }
                    }
                } label: {
                    Image(systemName: "scope")
                        .font(.system(size: 12, weight: .bold))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                        .frame(width: 32, height: 30)
                        .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Recenter radar")

                Button {
                    Task { await store.loadRadar() }
                } label: {
                    Group {
                        if store.radarLoading {
                            ProgressView().controlSize(.mini).tint(WxGlassTheme.cyan)
                        } else {
                            Image(systemName: "arrow.clockwise")
                                .font(.system(size: 12, weight: .bold))
                                .foregroundStyle(WxGlassTheme.cyan)
                        }
                    }
                    .frame(width: 32, height: 30)
                    .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                }
                .buttonStyle(.plain)
                .disabled(store.radarLoading)
                .accessibilityLabel("Refresh radar")
            }
        }
    }

    private var floatingRadarTransport: some View {
        let frameCount = store.radarPictures.count
        let shape = RoundedRectangle(cornerRadius: 18, style: .continuous)

        return VStack(spacing: 8) {
            HStack(spacing: 8) {
                // Play / Pause Capsule Button
                Button {
                    store.toggleRadarPlayback()
                } label: {
                    HStack(spacing: 5) {
                        Image(systemName: store.loopPlaying ? "pause.fill" : "play.fill")
                            .font(.system(size: 11, weight: .bold))
                        Text(store.loopPlaying ? "Loop" : "Held")
                            .font(.system(size: 11.5, weight: .bold, design: .rounded))
                            .accessibilityIdentifier("radar.transport")
                    }
                    .foregroundStyle(store.loopPlaying ? Color(red: 0.03, green: 0.07, blue: 0.14) : WxGlassTheme.cyan)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .background(
                        store.loopPlaying ? WxGlassTheme.cyan : Color.white.opacity(0.08),
                        in: Capsule()
                    )
                }
                .buttonStyle(.plain)
                .disabled(frameCount <= 1)

                // Step Backward / Forward
                if frameCount > 1 {
                    HStack(spacing: 4) {
                        Button {
                            store.stepRadarFrame(by: -1)
                        } label: {
                            Image(systemName: "backward.frame.fill")
                                .font(.system(size: 10.5, weight: .semibold))
                                .foregroundStyle(WxGlassTheme.textSecondary)
                                .frame(width: 26, height: 26)
                                .background(Color.white.opacity(0.06), in: Circle())
                        }
                        .buttonStyle(.plain)

                        Button {
                            store.stepRadarFrame(by: 1)
                        } label: {
                            Image(systemName: "forward.frame.fill")
                                .font(.system(size: 10.5, weight: .semibold))
                                .foregroundStyle(WxGlassTheme.textSecondary)
                                .frame(width: 26, height: 26)
                                .background(Color.white.opacity(0.06), in: Circle())
                        }
                        .buttonStyle(.plain)
                    }
                }

                // Frame Scrubber Ticks
                if frameCount > 1 {
                    HStack(spacing: 4) {
                        ForEach(0..<frameCount, id: \.self) { idx in
                            let active = (idx == store.loopFrame)
                            Button {
                                store.seekRadarFrame(to: idx)
                            } label: {
                                Capsule()
                                    .fill(active ? WxGlassTheme.cyan : Color.white.opacity(0.24))
                                    .frame(maxWidth: .infinity)
                                    .frame(height: active ? 7 : 4)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                } else {
                    Spacer(minLength: 4)
                }

                if frameCount > 0 {
                    Text("\(store.loopFrame + 1)/\(frameCount)")
                        .font(.system(size: 11.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                        .accessibilityIdentifier("radar.frame")
                }
            }

            // dBZ Intensity Legend Bar + Timestamp
            HStack(spacing: 8) {
                Text("dBZ")
                    .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxGlassTheme.textMuted)

                LinearGradient(
                    colors: [
                        Color(red: 0.1, green: 0.8, blue: 0.2),
                        Color(red: 0.9, green: 0.9, blue: 0.1),
                        Color(red: 1.0, green: 0.5, blue: 0.0),
                        Color(red: 0.9, green: 0.1, blue: 0.1),
                        Color(red: 0.8, green: 0.1, blue: 0.8),
                    ],
                    startPoint: .leading,
                    endPoint: .trailing
                )
                .frame(height: 5)
                .clipShape(Capsule())

                if let stamp = formattedValidTime {
                    Text(stamp)
                        .font(.system(size: 10, weight: .semibold, design: .monospaced))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                }
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background {
            ZStack {
                shape.fill(.ultraThinMaterial)
                shape.fill(WxGlassTheme.pillTintOverlay())
            }
        }
        .overlay(shape.strokeBorder(Color.white.opacity(0.18), lineWidth: 0.9))
        .shadow(color: Color.black.opacity(0.35), radius: 14, y: 6)
    }

    private var formattedValidTime: String? {
        guard let iso = store.currentRadarValidTime, !iso.isEmpty else { return nil }
        let parser = ISO8601DateFormatter()
        parser.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = parser.date(from: iso) {
            return d.formatted(date: .omitted, time: .shortened)
        }
        parser.formatOptions = [.withInternetDateTime]
        if let d = parser.date(from: iso) {
            return d.formatted(date: .omitted, time: .shortened)
        }
        return iso
    }
}

// MARK: - Severe Weather & ConditionPanel Tab

struct SevereView: View {
    @EnvironmentObject private var store: IOSStore

    var body: some View {
        let alerts = store.payload?.alerts ?? []

        ScrollView(showsIndicators: false) {
            VStack(alignment: .leading, spacing: 14) {
                if store.severeLoading {
                    HStack(spacing: 8) {
                        ProgressView().tint(WxGlassTheme.cyan)
                        Text("Syncing SPC convective intelligence…")
                            .font(.system(size: 13, weight: .medium, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textSecondary)
                    }
                }

                if let error = store.severeError, !error.isEmpty {
                    Text(error)
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(WxGlassTheme.conditionRed)
                        .accessibilityIdentifier("wx.error")
                }

                if let summary = store.spc?.convectiveSummary, !summary.isEmpty {
                    HStack(spacing: 8) {
                        Image(systemName: "bolt.shield.fill")
                            .font(.system(size: 16, weight: .semibold))
                            .foregroundStyle(WxGlassTheme.gold)
                        Text(summary)
                            .font(.system(size: 18, weight: .bold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textPrimary)
                            .accessibilityIdentifier("severe.summary")
                    }
                    .padding(.horizontal, 2)
                }

                // 1. Active NWS Alerts with Starfleet ConditionPanel
                if !alerts.isEmpty {
                    WxGlassCard(title: "Active NWS Alerts", tag: "\(alerts.count) ACTIVE", accent: WxGlassTheme.conditionRed) {
                        VStack(alignment: .leading, spacing: 12) {
                            ForEach(alerts) { alert in
                                severeAlertRow(alert)
                            }
                        }
                    }
                }

                // 2. SPC Days 1–3 Convective Outlooks
                if let day1 = store.spc?.day1 {
                    spcDayCard(title: "Day 1 Convective Outlook", item: day1)
                }
                if let day2 = store.spc?.day2 {
                    spcDayCard(title: "Day 2 Convective Outlook", item: day2)
                }
                if let day3 = store.spc?.day3 {
                    spcDayCard(title: "Day 3 Convective Outlook", item: day3)
                }

                // 3. Quiet All-Clear Card when no hazards or outlooks exist
                if alerts.isEmpty && store.spc?.day1 == nil && !store.severeLoading && store.severeError == nil {
                    WxGlassCard(title: "Hazard Status", tag: "NOMINAL", accent: WxGlassTheme.nominalGreen) {
                        HStack(spacing: 12) {
                            Image(systemName: "checkmark.shield.fill")
                                .font(.system(size: 28))
                                .foregroundStyle(WxGlassTheme.nominalGreen)
                            VStack(alignment: .leading, spacing: 3) {
                                Text("No Active Watches or Warnings")
                                    .font(.system(size: 15, weight: .bold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textPrimary)
                                Text("NWS hazard feeds and SPC convective outlooks are currently quiet for this location.")
                                    .font(.system(size: 12.5))
                                    .foregroundStyle(WxGlassTheme.textSecondary)
                            }
                        }
                    }
                }
            }
            .padding(.horizontal, 14)
            .padding(.top, 6)
            .padding(.bottom, 8)
            .accessibilityElement(children: .contain)
        }
        .wxDockScrollClearance()
        .accessibilityIdentifier("severe.root")
    }

    private func severeAlertRow(_ alert: Alert) -> some View {
        let band = ConditionBand.from(severity: alert.severity) ?? (alert.isWarning ? .red : .yellow)
        let accent = band == .red ? WxGlassTheme.conditionRed : WxGlassTheme.gold

        return VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .center, spacing: 12) {
                ConditionPanel(band: band, size: 90)

                VStack(alignment: .leading, spacing: 3) {
                    Text(alert.event)
                        .font(.system(size: 15, weight: .bold, design: .rounded))
                        .foregroundStyle(accent)

                    if let headline = alert.headline, !headline.isEmpty {
                        Text(headline)
                            .font(.system(size: 12.5))
                            .foregroundStyle(WxGlassTheme.textPrimary)
                            .lineLimit(3)
                    }

                    if let expires = alert.expires, !expires.isEmpty {
                        Text("Expires \(expires)")
                            .font(.system(size: 10.5, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxGlassTheme.textMuted)
                            .lineLimit(1)
                    }
                }
                Spacer(minLength: 0)
            }

            if let instruction = alert.instruction, !instruction.isEmpty {
                Text(instruction)
                    .font(.system(size: 12))
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .padding(10)
        .background(accent.opacity(0.10), in: WxGlassTheme.nestedShape())
        .overlay(WxGlassTheme.nestedShape().strokeBorder(accent.opacity(0.38), lineWidth: 0.9))
    }

    private func spcDayCard(title: String, item: SPCOutlookItemDTO) -> some View {
        let accent = spcAccent(for: item.category.code)
        let hasProbs = (item.tornadoProb != nil) || (item.windProb != nil) || (item.hailProb != nil)

        return WxGlassCard(title: title, tag: item.category.code, accent: accent) {
            VStack(alignment: .leading, spacing: 10) {
                HStack(spacing: 8) {
                    Text(item.category.name)
                        .font(.system(size: 17, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)

                    Spacer()

                    Text("CAT \(item.category.dn)")
                        .font(.system(size: 10.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(accent)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(accent.opacity(0.16), in: Capsule())
                }

                Text(item.category.description)
                    .font(.system(size: 13))
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)

                if hasProbs {
                    HStack(spacing: 8) {
                        if let tor = item.tornadoProb {
                            WxSensorCapsule(label: "Tornado", value: tor, symbol: "tornado", accent: WxGlassTheme.conditionRed, compact: true)
                        }
                        if let wind = item.windProb {
                            WxSensorCapsule(label: "Wind", value: wind, symbol: "wind", accent: WxGlassTheme.gold, compact: true)
                        }
                        if let hail = item.hailProb {
                            WxSensorCapsule(label: "Hail", value: hail, symbol: "cloud.hail.fill", accent: WxGlassTheme.cyan, compact: true)
                        }
                    }
                }
            }
        }
    }

    private func spcAccent(for code: String) -> Color {
        switch code.uppercased() {
        case "HIGH", "MDT": return WxGlassTheme.conditionRed
        case "ENH": return WxGlassTheme.amber
        case "SLGT": return WxGlassTheme.gold
        case "MRGL", "TSTM": return WxGlassTheme.nominalGreen
        default: return WxGlassTheme.cyan
        }
    }
}

// MARK: - Tropical Cyclones & NHC Invests Tab

struct TropicsTabView: View {
    @EnvironmentObject private var store: IOSStore

    var body: some View {
        let storms = store.tropics?.tropics?.storms ?? []
        let disturbances = store.tropics?.tropics?.disturbances ?? []

        ScrollView(showsIndicators: false) {
            VStack(alignment: .leading, spacing: 14) {
                if store.tropicsLoading {
                    HStack(spacing: 8) {
                        ProgressView().tint(WxGlassTheme.cyan)
                        Text("Scanning NHC Atlantic & Pacific basins…")
                            .font(.system(size: 13, weight: .medium, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textSecondary)
                    }
                }

                if let error = store.tropicsError, !error.isEmpty {
                    Text(error)
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(WxGlassTheme.conditionRed)
                        .accessibilityIdentifier("wx.error")
                }

                if storms.isEmpty && !store.tropicsLoading && store.tropicsError == nil && store.tropics != nil {
                    WxGlassCard(title: "Tropical Basin Monitor", tag: "NHC.QUIET", accent: WxGlassTheme.nominalGreen) {
                        HStack(spacing: 12) {
                            Image(systemName: "hurricane")
                                .font(.system(size: 26))
                                .foregroundStyle(WxGlassTheme.nominalGreen)
                            VStack(alignment: .leading, spacing: 3) {
                                Text("No active storms")
                                    .font(.system(size: 15, weight: .bold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textPrimary)
                                Text("Atlantic and East Pacific basins have no active tropical cyclones.")
                                    .font(.system(size: 12.5))
                                    .foregroundStyle(WxGlassTheme.textSecondary)
                            }
                        }
                    }
                }

                ForEach(storms) { storm in
                    stormCard(storm)
                }

                if !disturbances.isEmpty {
                    WxGlassCard(title: "Tropical Invest Disturbances", tag: "\(disturbances.count) INVESTS", accent: WxGlassTheme.amber) {
                        VStack(alignment: .leading, spacing: 10) {
                            ForEach(disturbances) { invest in
                                VStack(alignment: .leading, spacing: 6) {
                                    HStack {
                                        Text("\(invest.name) · \(invest.basin)")
                                            .font(.system(size: 14, weight: .bold, design: .rounded))
                                            .foregroundStyle(WxGlassTheme.textPrimary)
                                        Spacer()
                                        Text("48h \(invest.chance48h)% · 7d \(invest.chance7d)%")
                                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                                            .foregroundStyle(WxGlassTheme.gold)
                                            .padding(.horizontal, 8)
                                            .padding(.vertical, 3)
                                            .background(WxGlassTheme.gold.opacity(0.14), in: Capsule())
                                    }
                                    if let summary = invest.summary, !summary.isEmpty {
                                        Text(summary)
                                            .font(.system(size: 12.5))
                                            .foregroundStyle(WxGlassTheme.textSecondary)
                                            .lineLimit(3)
                                    }
                                }
                                .padding(10)
                                .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
                            }
                        }
                    }
                }
            }
            .padding(.horizontal, 14)
            .padding(.top, 6)
            .padding(.bottom, 8)
            .accessibilityElement(children: .contain)
        }
        .wxDockScrollClearance()
        .accessibilityIdentifier("tropics.root")
    }

    private func stormCard(_ storm: TropicalStormDTO) -> some View {
        let accent: Color = {
            if storm.category >= 3 { return WxGlassTheme.conditionRed }
            if storm.category >= 1 { return WxGlassTheme.amber }
            return WxGlassTheme.gold
        }()
        let isMetric = (store.units == "metric")
        let windVal = isMetric ? "\(storm.windSpeedKmh) km/h" : "\(storm.windSpeedMph) mph"
        let moveVal = isMetric
            ? "\(storm.movementCompass) \(storm.movementSpeedKmh) km/h"
            : "\(storm.movementCompass) \(storm.movementSpeedMph) mph"

        return WxGlassCard(title: storm.classification, tag: storm.categoryLabel ?? storm.classification, accent: accent) {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .firstTextBaseline) {
                    Text(storm.name)
                        .font(.system(size: 20, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                        .accessibilityIdentifier("tropics.storm")
                    Spacer()
                    Text(storm.locationText)
                        .font(.system(size: 11.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxGlassTheme.textMuted)
                }

                LazyVGrid(columns: [GridItem(.flexible(), spacing: 8), GridItem(.flexible(), spacing: 8)], spacing: 8) {
                    WxSensorCapsule(
                        label: "Max Winds",
                        value: windVal,
                        subtitle: "\(storm.intensityKt) kt sustained",
                        symbol: "wind",
                        accent: accent,
                        compact: true
                    )
                    WxSensorCapsule(
                        label: "Min Pressure",
                        value: storm.pressureMb > 0 ? "\(storm.pressureMb) mb" : "—",
                        subtitle: storm.pressureInHg > 0 ? String(format: "%.2f inHg", storm.pressureInHg) : nil,
                        symbol: "gauge.with.dots.needle.bottom.50percent",
                        accent: WxGlassTheme.cyan,
                        compact: true
                    )
                    WxSensorCapsule(
                        label: "Movement",
                        value: moveVal,
                        subtitle: "Heading \(storm.movementDir)°",
                        symbol: "location.north.line.fill",
                        accent: WxGlassTheme.gold,
                        compact: true
                    )
                    WxSensorCapsule(
                        label: "Proximity",
                        value: storm.distanceMiles.map { "\(Int($0)) mi" } ?? "Basin",
                        subtitle: storm.proximityText,
                        symbol: "hurricane",
                        accent: accent,
                        compact: true
                    )
                }

                if let headline = storm.headline, !headline.isEmpty {
                    Text(headline)
                        .font(.system(size: 13))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }
        }
    }
}

// MARK: - Dark Liquid Glass Config & Stations Tab

struct ConfigView: View {
    @EnvironmentObject private var store: IOSStore
    @State private var draft: String = ""
    @State private var units: String = "imperial"

    var body: some View {
        let cleanDraft = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        let isFav = store.isFavorite(cleanDraft.isEmpty ? store.place : cleanDraft)

        ScrollView(showsIndicators: false) {
            VStack(alignment: .leading, spacing: 14) {
                HStack(spacing: 8) {
                    Image(systemName: "slider.horizontal.3")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(WxGlassTheme.cyan)
                    Text("Saved place")
                        .font(.system(size: 16, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                        .accessibilityIdentifier("config.saved")
                }
                .padding(.horizontal, 2)

                // 1. Primary Dark-Glass Location & Units Card
                WxGlassCard(title: "Station & Telemetry Units", tag: units.uppercased(), accent: WxGlassTheme.cyan) {
                    VStack(alignment: .leading, spacing: 12) {
                        // Dark-Glass Concentric Search Input
                        HStack(spacing: 8) {
                            Image(systemName: "magnifyingglass")
                                .font(.system(size: 14, weight: .semibold))
                                .foregroundStyle(WxGlassTheme.cyan)

                            TextField("City, ST or ZIP", text: $draft)
                                .textFieldStyle(.plain)
                                .font(.system(size: 15, weight: .medium, design: .rounded))
                                .foregroundStyle(WxGlassTheme.textPrimary)
                                .textInputAutocapitalization(.words)
                                .autocorrectionDisabled()
                                .submitLabel(.go)
                                .onSubmit {
                                    store.save(place: draft, units: units)
                                }
                                .accessibilityIdentifier("config.place")

                            if !draft.isEmpty {
                                Button {
                                    draft = ""
                                } label: {
                                    Image(systemName: "xmark.circle.fill")
                                        .font(.system(size: 14))
                                        .foregroundStyle(WxGlassTheme.textMuted)
                                }
                                .buttonStyle(.plain)
                            }

                            Button {
                                let target = cleanDraft.isEmpty ? store.place : cleanDraft
                                store.toggleFavorite(target)
                            } label: {
                                Image(systemName: isFav ? "star.fill" : "star")
                                    .font(.system(size: 14, weight: .semibold))
                                    .foregroundStyle(isFav ? WxGlassTheme.gold : WxGlassTheme.textSecondary)
                            }
                            .buttonStyle(.plain)
                            .disabled((cleanDraft.isEmpty ? store.place : cleanDraft).isEmpty)
                            .accessibilityLabel(isFav ? "Unpin station" : "Pin station")
                        }
                        .padding(.horizontal, 12)
                        .padding(.vertical, 11)
                        .background(Color.white.opacity(0.07), in: WxGlassTheme.nestedShape())
                        .overlay(
                            WxGlassTheme.nestedShape()
                                .strokeBorder(WxGlassTheme.cyan.opacity(0.36), lineWidth: 1)
                        )

                        // Sculpted Segmented Unit Selector
                        Picker("Units", selection: $units) {
                            Text("Imperial (°F)").tag("imperial")
                            Text("Metric (°C)").tag("metric")
                        }
                        .pickerStyle(.segmented)
                        .accessibilityIdentifier("config.units")
                        .onChange(of: units) { _, nextUnits in
                            guard nextUnits != store.units else { return }
                            let target = cleanDraft.isEmpty ? store.place : cleanDraft
                            store.save(place: target, units: nextUnits)
                        }

                        // Primary Action Capsules
                        HStack(spacing: 10) {
                            Button {
                                store.save(place: draft, units: units)
                            } label: {
                                HStack(spacing: 6) {
                                    Image(systemName: "checkmark.circle.fill")
                                        .font(.system(size: 13, weight: .bold))
                                    Text("Save place")
                                        .font(.system(size: 14, weight: .bold, design: .rounded))
                                }
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 11)
                                .foregroundStyle(Color(red: 0.03, green: 0.07, blue: 0.14))
                                .background(
                                    LinearGradient(
                                        colors: [WxGlassTheme.cyanSoft, WxGlassTheme.cyan],
                                        startPoint: .topLeading,
                                        endPoint: .bottomTrailing
                                    ),
                                    in: Capsule()
                                )
                                .shadow(color: WxGlassTheme.cyan.opacity(0.35), radius: 8, y: 3)
                            }
                            .buttonStyle(.plain)
                            .accessibilityIdentifier("config.save")

                            Button {
                                Task {
                                    await store.useCurrentLocation()
                                    draft = store.place
                                }
                            } label: {
                                HStack(spacing: 6) {
                                    Image(systemName: "location.fill")
                                        .font(.system(size: 12.5, weight: .semibold))
                                        .foregroundStyle(WxGlassTheme.cyan)
                                    Text("Use current location")
                                        .font(.system(size: 13.5, weight: .semibold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.textPrimary)
                                        .lineLimit(1)
                                        .minimumScaleFactor(0.8)
                                }
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 11)
                                .padding(.horizontal, 8)
                                .background(Color.white.opacity(0.08), in: Capsule())
                                .overlay(Capsule().strokeBorder(Color.white.opacity(0.22), lineWidth: 1))
                            }
                            .buttonStyle(.plain)
                            .accessibilityIdentifier("config.locate")
                        }

                        Text("Location is one shot, only while the app is in use. Denying it keeps this saved place.")
                            .font(.system(size: 12))
                            .foregroundStyle(WxGlassTheme.textMuted)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                }

                // 2. Pinned Favorites & Recent Stations Card
                WxGlassCard(
                    title: "Pinned & Recent Stations",
                    tag: store.favorites.isEmpty ? nil : "\(store.favorites.count) PINNED",
                    accent: WxGlassTheme.gold
                ) {
                    VStack(alignment: .leading, spacing: 10) {
                        if !store.favorites.isEmpty {
                            VStack(spacing: 8) {
                                ForEach(store.favorites) { fav in
                                    HStack(spacing: 10) {
                                        Button {
                                            draft = fav.value
                                            store.save(place: fav.value, units: units)
                                        } label: {
                                            HStack(spacing: 8) {
                                                Image(systemName: "star.fill")
                                                    .font(.system(size: 12))
                                                    .foregroundStyle(WxGlassTheme.gold)
                                                Text(fav.name)
                                                    .font(.system(size: 14, weight: .semibold, design: .rounded))
                                                    .foregroundStyle(WxGlassTheme.textPrimary)
                                                Spacer()
                                                if store.place.caseInsensitiveCompare(fav.value) == .orderedSame {
                                                    Text("ACTIVE")
                                                        .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                                                        .foregroundStyle(WxGlassTheme.cyan)
                                                        .padding(.horizontal, 7)
                                                        .padding(.vertical, 2)
                                                        .background(WxGlassTheme.cyan.opacity(0.14), in: Capsule())
                                                }
                                            }
                                        }
                                        .buttonStyle(.plain)

                                        Button {
                                            store.removeFavorite(fav.name)
                                        } label: {
                                            Image(systemName: "minus.circle.fill")
                                                .font(.system(size: 14))
                                                .foregroundStyle(WxGlassTheme.textMuted)
                                        }
                                        .buttonStyle(.plain)
                                        .accessibilityLabel("Remove \(fav.name)")
                                    }
                                    .padding(.horizontal, 12)
                                    .padding(.vertical, 9)
                                    .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
                                }
                            }
                        } else {
                            Text("Tap the star inside the search bar to pin favorite cities for instant one-tap switching.")
                                .font(.system(size: 12.5))
                                .foregroundStyle(WxGlassTheme.textSecondary)
                        }

                        // Quick Regional Presets / Recents Strip
                        let suggestions = store.recentLocations.isEmpty
                            ? ["Fort Wayne, IN", "New Haven, IN", "Milwaukee, WI", "Chicago, IL"]
                            : Array(store.recentLocations.prefix(6))

                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 8) {
                                ForEach(suggestions, id: \.self) { item in
                                    Button {
                                        draft = item
                                        store.save(place: item, units: units)
                                    } label: {
                                        HStack(spacing: 4) {
                                            Image(systemName: store.recentLocations.isEmpty ? "sparkles" : "clock.arrow.circlepath")
                                                .font(.system(size: 10, weight: .semibold))
                                                .foregroundStyle(WxGlassTheme.cyan)
                                            Text(item)
                                                .font(.system(size: 12, weight: .semibold, design: .rounded))
                                                .foregroundStyle(WxGlassTheme.textPrimary)
                                        }
                                        .padding(.horizontal, 10)
                                        .padding(.vertical, 6)
                                        .background(Color.white.opacity(0.06), in: Capsule())
                                        .overlay(Capsule().strokeBorder(Color.white.opacity(0.14), lineWidth: 0.8))
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                        }
                    }
                }
            }
            .padding(.horizontal, 14)
            .padding(.top, 6)
            .padding(.bottom, 8)
            .accessibilityElement(children: .contain)
        }
        .wxDockScrollClearance()
        .accessibilityIdentifier("config.root")
        .onAppear {
            draft = store.place
            units = store.units
            store.reloadSavedStations()
        }
        .onChange(of: store.place) { _, place in
            draft = place
        }
        .onChange(of: store.units) { _, nextUnits in
            units = nextUnits
        }
    }
}
