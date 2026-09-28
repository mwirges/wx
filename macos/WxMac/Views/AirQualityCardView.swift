import SwiftUI

struct AirQualityCardView: View {
    let airQuality: AirQualityDTO

    private var aqiValue: Int {
        airQuality.aqi ?? 0
    }

    private var aqiColor: Color {
        WxTheme.aqiColor(airQuality.aqi)
    }

    var body: some View {
        SNWConsoleCard(
            title: "Air Quality & Smoke Plume Console",
            tag: "EPA.AQI",
            statusColor: aqiColor
        ) {
            VStack(alignment: .leading, spacing: 10) {
                // Top Row: Primary AQI Score & Category Badge
                HStack(alignment: .center, spacing: 12) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("US EPA AQI")
                            .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                            .tracking(0.8)
                            .foregroundStyle(WxTheme.snwSilver)

                        HStack(alignment: .firstTextBaseline, spacing: 6) {
                            Text("\(aqiValue)")
                                .font(.system(size: 26, weight: .heavy, design: .monospaced))
                                .foregroundStyle(aqiColor)

                            Text("/ 500")
                                .font(.system(size: 11, weight: .semibold, design: .monospaced))
                                .foregroundStyle(WxTheme.textSecondary)
                        }
                    }

                    Spacer()

                    // Category Pill Badge
                    VStack(alignment: .trailing, spacing: 3) {
                        let cat = airQuality.category ?? "Good"
                        HStack(spacing: 5) {
                            Circle()
                                .fill(aqiColor)
                                .frame(width: 6, height: 6)
                                .shadow(color: aqiColor.opacity(0.8), radius: 2)

                            Text(cat.uppercased())
                                .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(aqiColor)
                        }
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(aqiColor.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(aqiColor.opacity(0.35), lineWidth: 0.8))

                        if let uv = airQuality.uvIndex {
                            let uvCat = airQuality.uvCategory ?? "Low"
                            Text("UV INDEX: \(String(format: "%.1f", uv)) (\(uvCat.uppercased()))")
                                .font(.system(size: 8, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwCyan.opacity(0.85))
                        }
                    }
                }

                // Spectrum Meter Gauge Bar (0-500)
                AQISpectrumBar(aqi: aqiValue)

                // Smoke Plume Advisory Callout (Active when PM2.5 > 35 or smoke detected)
                let pm25Val = airQuality.pm25 ?? 0
                let isSmokeElevated = pm25Val >= 35.5 || !(airQuality.smokeAdvisory ?? "").isEmpty
                if isSmokeElevated {
                    let smokeMsg = (airQuality.smokeAdvisory?.isEmpty == false)
                        ? airQuality.smokeAdvisory!
                        : "Elevated PM2.5 smoke particulates active (\(String(format: "%.1f", pm25Val)) μg/m³). Wildfire haze present."

                    HStack(alignment: .top, spacing: 8) {
                        Image(systemName: "smoke.fill")
                            .font(.system(size: 13))
                            .foregroundStyle(Color.orange)

                        VStack(alignment: .leading, spacing: 2) {
                            Text("WILDFIRE SMOKE / PLUME DETECTED")
                                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(Color.orange)

                            Text(smokeMsg)
                                .font(.system(size: 9, weight: .medium, design: .monospaced))
                                .foregroundStyle(WxTheme.text)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                    }
                    .padding(8)
                    .background(Color.orange.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(Color.orange.opacity(0.35), lineWidth: 0.8))
                }

                // Official EPA Health Advisory Statement
                HStack(alignment: .top, spacing: 7) {
                    Image(systemName: "heart.text.square.fill")
                        .font(.system(size: 12))
                        .foregroundStyle(aqiColor)

                    Text(airQuality.resolvedAdvisory)
                        .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.text.opacity(0.95))
                        .fixedSize(horizontal: false, vertical: true)
                }
                .padding(7)
                .background(WxTheme.snwChassis.opacity(0.7), in: RoundedRectangle(cornerRadius: 4))
                .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.border.opacity(0.25), lineWidth: 0.6))

                // Atmospheric Pollutants Grid
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 95), spacing: 6)], spacing: 6) {
                    if let pm25 = airQuality.pm25 {
                        pollutantTile(label: "PM2.5", sub: "Smoke/Fine", value: String(format: "%.1f μg", pm25), alert: pm25 >= 35.5)
                    }
                    if let pm10 = airQuality.pm10 {
                        pollutantTile(label: "PM10", sub: "Dust/Coarse", value: String(format: "%.1f μg", pm10), alert: pm10 >= 150)
                    }
                    if let o3 = airQuality.ozone {
                        pollutantTile(label: "Ozone", sub: "Ground O₃", value: String(format: "%.0f μg", o3), alert: o3 >= 160)
                    }
                    if let no2 = airQuality.no2 {
                        pollutantTile(label: "NO₂", sub: "Exhaust", value: String(format: "%.1f μg", no2), alert: no2 >= 100)
                    }
                    if let co = airQuality.co {
                        pollutantTile(label: "CO", sub: "Combustion", value: String(format: "%.0f μg", co), alert: co >= 4000)
                    }
                    if let so2 = airQuality.so2 {
                        pollutantTile(label: "SO₂", sub: "Industrial", value: String(format: "%.1f μg", so2), alert: so2 >= 20)
                    }
                }
            }
        }
    }

    private func pollutantTile(label: String, sub: String, value: String, alert: Bool) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            HStack {
                Text(label)
                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(alert ? Color.orange : WxTheme.snwSilver)
                Spacer()
                Text(sub)
                    .font(.system(size: 7, weight: .medium, design: .monospaced))
                    .foregroundStyle(WxTheme.textSecondary.opacity(0.7))
            }

            Text(value)
                .font(.system(size: 11, weight: .bold, design: .monospaced))
                .foregroundStyle(alert ? Color.orange : WxTheme.text)
        }
        .padding(.horizontal, 7)
        .padding(.vertical, 5)
        .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 4))
        .overlay(
            RoundedRectangle(cornerRadius: 4)
                .strokeBorder(alert ? Color.orange.opacity(0.4) : WxTheme.border.opacity(0.2), lineWidth: 0.6)
        )
    }
}

/// Horizontal segmented gauge showing the EPA AQI scale 0 to 500 with indicator
struct AQISpectrumBar: View {
    let aqi: Int

    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width
            let ratio = min(1.0, max(0.0, Double(aqi) / 500.0))
            let markerX = max(4.0, min(w - 4.0, w * ratio))

            ZStack(alignment: .leading) {
                // Background Spectrum
                HStack(spacing: 2) {
                    segment(color: WxTheme.snwGreen)
                    segment(color: WxTheme.snwAmber)
                    segment(color: Color.orange)
                    segment(color: Color.red)
                    segment(color: Color.purple)
                    segment(color: Color(red: 0.5, green: 0.0, blue: 0.15))
                }
                .frame(height: 6)
                .clipShape(RoundedRectangle(cornerRadius: 3))

                // Marker needle
                Rectangle()
                    .fill(Color.white)
                    .frame(width: 2.5, height: 10)
                    .shadow(color: Color.black.opacity(0.7), radius: 2)
                    .position(x: markerX, y: 3)
            }
        }
        .frame(height: 10)
    }

    private func segment(color: Color) -> some View {
        color.opacity(0.85)
            .frame(maxWidth: .infinity)
    }
}
