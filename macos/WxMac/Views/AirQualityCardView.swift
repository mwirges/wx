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
        WxGlassCard(
            title: "Air Quality & Smoke Plume",
            tag: "EPA.AQI",
            accent: aqiColor
        ) {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .center, spacing: 12) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("US EPA AQI")
                            .font(.system(size: 9, weight: .bold, design: .rounded))
                            .tracking(0.5)
                            .foregroundStyle(WxGlassTheme.textMuted)

                        HStack(alignment: .firstTextBaseline, spacing: 6) {
                            Text("\(aqiValue)")
                                .font(.system(size: 28, weight: .bold, design: .rounded))
                                .foregroundStyle(aqiColor)

                            Text("/ 500")
                                .font(.system(size: 11.5, weight: .semibold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.textSecondary)
                        }
                    }

                    Spacer()

                    VStack(alignment: .trailing, spacing: 4) {
                        let cat = airQuality.category ?? "Good"
                        HStack(spacing: 5) {
                            Circle()
                                .fill(aqiColor)
                                .frame(width: 6, height: 6)
                                .shadow(color: aqiColor.opacity(0.8), radius: 2)

                            Text(cat.uppercased())
                                .font(.system(size: 9.5, weight: .bold, design: .rounded))
                                .foregroundStyle(aqiColor)
                        }
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4)
                        .background(aqiColor.opacity(0.14), in: Capsule())
                        .overlay(Capsule().strokeBorder(aqiColor.opacity(0.38), lineWidth: 0.8))

                        if let uv = airQuality.uvIndex {
                            let uvCat = airQuality.uvCategory ?? "Low"
                            Text("UV \(String(format: "%.1f", uv)) · \(uvCat)")
                                .font(.system(size: 9.5, weight: .semibold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.cyanSoft)
                        }
                    }
                }

                AQISpectrumBar(aqi: aqiValue)

                let pm25Val = airQuality.pm25 ?? 0
                let isSmokeElevated = pm25Val >= 35.5 || !(airQuality.smokeAdvisory ?? "").isEmpty
                if isSmokeElevated {
                    let smokeMsg = (airQuality.smokeAdvisory?.isEmpty == false)
                        ? airQuality.smokeAdvisory!
                        : "Elevated PM2.5 smoke particulates active (\(String(format: "%.1f", pm25Val)) μg/m³). Wildfire haze present."

                    HStack(alignment: .top, spacing: 8) {
                        Image(systemName: "smoke.fill")
                            .font(.system(size: 14))
                            .foregroundStyle(Color.orange)

                        VStack(alignment: .leading, spacing: 2) {
                            Text("Wildfire Smoke / Plume Detected")
                                .font(.system(size: 11, weight: .bold, design: .rounded))
                                .foregroundStyle(Color.orange)

                            Text(smokeMsg)
                                .font(.system(size: 11.5))
                                .foregroundStyle(WxGlassTheme.textPrimary)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                    }
                    .padding(10)
                    .background(Color.orange.opacity(0.12), in: WxGlassTheme.nestedShape())
                    .overlay(WxGlassTheme.nestedShape().strokeBorder(Color.orange.opacity(0.38), lineWidth: 0.8))
                }

                HStack(alignment: .top, spacing: 8) {
                    Image(systemName: "heart.text.square.fill")
                        .font(.system(size: 13))
                        .foregroundStyle(aqiColor)

                    Text(airQuality.resolvedAdvisory)
                        .font(.system(size: 11.5))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
                .padding(10)
                .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
                .overlay(WxGlassTheme.nestedShape().strokeBorder(Color.white.opacity(0.10), lineWidth: 0.7))

                LazyVGrid(columns: [GridItem(.adaptive(minimum: 100), spacing: 7)], spacing: 7) {
                    if let pm25 = airQuality.pm25 {
                        WxSensorCapsule(label: "PM2.5", value: String(format: "%.1f μg", pm25), subtitle: "Fine Smoke", accent: pm25 >= 35.5 ? .orange : WxGlassTheme.cyan, compact: true)
                    }
                    if let pm10 = airQuality.pm10 {
                        WxSensorCapsule(label: "PM10", value: String(format: "%.1f μg", pm10), subtitle: "Coarse Dust", accent: pm10 >= 150 ? .orange : WxGlassTheme.cyan, compact: true)
                    }
                    if let o3 = airQuality.ozone {
                        WxSensorCapsule(label: "Ozone", value: String(format: "%.0f μg", o3), subtitle: "Ground O₃", accent: o3 >= 160 ? .orange : WxGlassTheme.cyan, compact: true)
                    }
                    if let no2 = airQuality.no2 {
                        WxSensorCapsule(label: "NO₂", value: String(format: "%.1f μg", no2), subtitle: "Exhaust", accent: no2 >= 100 ? .orange : WxGlassTheme.cyan, compact: true)
                    }
                    if let co = airQuality.co {
                        WxSensorCapsule(label: "CO", value: String(format: "%.0f μg", co), subtitle: "Combustion", accent: co >= 4000 ? .orange : WxGlassTheme.cyan, compact: true)
                    }
                    if let so2 = airQuality.so2 {
                        WxSensorCapsule(label: "SO₂", value: String(format: "%.1f μg", so2), subtitle: "Industrial", accent: so2 >= 20 ? .orange : WxGlassTheme.cyan, compact: true)
                    }
                }
            }
        }
    }
}

/// Continuous capsule spectrum gauge showing the EPA AQI scale 0 to 500 with glowing orb indicator.
struct AQISpectrumBar: View {
    let aqi: Int

    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width
            let ratio = min(1.0, max(0.0, Double(aqi) / 500.0))
            let markerX = max(5.0, min(w - 5.0, w * ratio))

            ZStack(alignment: .leading) {
                Capsule()
                    .fill(
                        LinearGradient(
                            colors: [
                                WxGlassTheme.nominalGreen,
                                WxGlassTheme.amber,
                                Color.orange,
                                WxGlassTheme.conditionRed,
                                Color.purple,
                                Color(red: 0.5, green: 0.0, blue: 0.15)
                            ],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                    )
                    .frame(height: 6)

                Circle()
                    .fill(Color.white)
                    .frame(width: 11, height: 11)
                    .shadow(color: Color.black.opacity(0.6), radius: 2)
                    .position(x: markerX, y: 5.5)
            }
        }
        .frame(height: 11)
    }
}
