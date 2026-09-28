import SwiftUI

struct EphemerisCardView: View {
    let astronomy: AstronomyDTO

    var body: some View {
        SNWConsoleCard(
            title: "Solar Arc & Ephemeris HUD",
            tag: "ASTRO.ORBIT",
            statusColor: WxTheme.snwGold
        ) {
            VStack(alignment: .leading, spacing: 10) {
                // Top Row: Solar Elevation, Azimuth & Regime
                HStack(alignment: .center, spacing: 12) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("SOLAR ELEVATION")
                            .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver)

                        HStack(alignment: .firstTextBaseline, spacing: 5) {
                            if let elev = astronomy.solarElevationDeg {
                                let sign = elev >= 0 ? "+" : ""
                                Text("\(sign)\(String(format: "%.1f°", elev))")
                                    .font(.system(size: 22, weight: .heavy, design: .monospaced))
                                    .foregroundStyle(elev >= 0 ? WxTheme.snwGold : WxTheme.snwCyan)
                            } else {
                                Text("—")
                                    .font(.system(size: 22, weight: .heavy, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwSilver)
                            }

                            if let az = astronomy.solarAzimuthDeg {
                                Text("\(String(format: "%.0f°", az)) \(astronomy.solarAzimuthCompass)")
                                    .font(.system(size: 10, weight: .semibold, design: .monospaced))
                                    .foregroundStyle(WxTheme.textSecondary)
                            }
                        }
                    }

                    Spacer()

                    // Regime Badge
                    let regime = astronomy.currentPeriod ?? (astronomy.isPolarDay == true ? "Polar Day" : (astronomy.isPolarNight == true ? "Polar Night" : "Daylight"))
                    let regimeColor = regimeColorFor(regime)
                    HStack(spacing: 5) {
                        Circle()
                            .fill(regimeColor)
                            .frame(width: 6, height: 6)
                            .shadow(color: regimeColor.opacity(0.8), radius: 2)

                        Text(regime.uppercased())
                            .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                            .foregroundStyle(regimeColor)
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(regimeColor.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(regimeColor.opacity(0.35), lineWidth: 0.8))
                }

                // Solar Arc Progress Meter
                solarArcMeter

                // Solar Milestones & Twilight Horizons
                VStack(spacing: 4) {
                    HStack {
                        if let sr = astronomy.sunriseFormatted {
                            horizonChip(label: "Sunrise", value: sr, icon: "arrow.up", color: WxTheme.snwGold)
                        }
                        if let sn = astronomy.solarNoonFormatted {
                            horizonChip(label: "Noon", value: sn, icon: "sun.max.fill", color: WxTheme.snwGold)
                        }
                        if let ss = astronomy.sunsetFormatted {
                            horizonChip(label: "Sunset", value: ss, icon: "arrow.down", color: WxTheme.snwAmber)
                        }
                        if let dl = astronomy.dayLength {
                            horizonChip(label: "Daylight", value: dl, icon: "clock.fill", color: WxTheme.snwCyan)
                        }
                    }

                    HStack {
                        if let cd = astronomy.civilDawnFormatted, let cdu = astronomy.civilDuskFormatted {
                            horizonChip(label: "Civil Twilight", value: "\(cd) / \(cdu)", icon: "sun.horizon.fill", color: WxTheme.snwCyan)
                        }
                        if let ghm = astronomy.goldenHourMorningFormatted {
                            horizonChip(label: "AM Golden", value: ghm, icon: "camera.fill", color: Color.orange)
                        }
                    }

                    if let ghe = astronomy.goldenHourEveningFormatted {
                        HStack {
                            horizonChip(label: "PM Golden", value: ghe, icon: "camera.fill", color: Color.orange)
                            if let nd = astronomy.nauticalDawnFormatted, let ndu = astronomy.nauticalDuskFormatted {
                                horizonChip(label: "Nautical", value: "\(nd) / \(ndu)", icon: "water.waves", color: WxTheme.snwSilver)
                            }
                        }
                    }
                }

                Divider().overlay(WxTheme.border.opacity(0.3))

                // Lunar Ephemeris Row
                HStack(alignment: .center, spacing: 10) {
                    let icon = astronomy.moonPhaseIcon ?? "🌖"
                    let phase = astronomy.moonPhase ?? "Moon"
                    Text(icon)
                        .font(.system(size: 20))

                    VStack(alignment: .leading, spacing: 2) {
                        Text(phase.uppercased())
                            .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.text)

                        HStack(spacing: 6) {
                            if let illum = astronomy.moonIlluminationPct {
                                Text("\(String(format: "%.0f%%", illum)) ILLUMINATED")
                                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwGold)
                            }
                            if let age = astronomy.moonAgeDays {
                                Text("AGE: \(String(format: "%.1fd", age))")
                                    .font(.system(size: 8, weight: .medium, design: .monospaced))
                                    .foregroundStyle(WxTheme.textSecondary)
                            }
                        }
                    }

                    Spacer()

                    // Lunar cycle bar
                    lunarCycleTrack
                }
            }
        }
    }

    private func regimeColorFor(_ regime: String) -> Color {
        switch regime.lowercased() {
        case "daylight", "day": return WxTheme.snwGold
        case "golden hour": return Color.orange
        case "civil twilight", "nautical twilight", "astronomical twilight": return WxTheme.snwCyan
        default: return WxTheme.snwSilver
        }
    }

    private func horizonChip(label: String, value: String, icon: String, color: Color) -> some View {
        HStack(spacing: 4) {
            Image(systemName: icon)
                .font(.system(size: 8))
                .foregroundStyle(color)

            VStack(alignment: .leading, spacing: 1) {
                Text(label.uppercased())
                    .font(.system(size: 6.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.textSecondary)
                Text(value)
                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.text)
                    .lineLimit(1)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 6)
        .padding(.vertical, 4)
        .background(WxTheme.snwChassis.opacity(0.85), in: RoundedRectangle(cornerRadius: 4))
        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.border.opacity(0.2), lineWidth: 0.6))
    }

    @ViewBuilder
    private var solarArcMeter: some View {
        GeometryReader { geo in
            let w = geo.size.width
            let elev = astronomy.solarElevationDeg ?? 0
            // Normalize elevation between -90 and +90
            let ratio = min(1.0, max(0.0, (elev + 18.0) / (90.0 + 18.0)))
            let markerX = max(4.0, min(w - 4.0, w * ratio))

            ZStack(alignment: .leading) {
                // Background Track (Night, Twilight, Daytime)
                HStack(spacing: 2) {
                    Rectangle().fill(WxTheme.snwChassis.opacity(0.9)).frame(width: w * 0.16) // Night
                    Rectangle().fill(WxTheme.snwCyan.opacity(0.35)).frame(width: w * 0.16) // Twilight
                    Rectangle().fill(Color.orange.opacity(0.5)).frame(width: w * 0.12) // Golden Hour
                    Rectangle().fill(WxTheme.snwGold.opacity(0.65)).frame(maxWidth: .infinity) // Full Daylight
                }
                .frame(height: 6)
                .clipShape(RoundedRectangle(cornerRadius: 3))

                // Sun Pip Indicator
                Circle()
                    .fill(elev >= 0 ? WxTheme.snwGold : WxTheme.snwCyan)
                    .frame(width: 10, height: 10)
                    .shadow(color: (elev >= 0 ? WxTheme.snwGold : WxTheme.snwCyan).opacity(0.8), radius: 3)
                    .position(x: markerX, y: 3)
            }
        }
        .frame(height: 10)
    }

    @ViewBuilder
    private var lunarCycleTrack: some View {
        let age = astronomy.moonAgeDays ?? 0
        let ratio = min(1.0, max(0.0, age / 29.530588853))

        HStack(spacing: 3) {
            Text("🌑")
                .font(.system(size: 8))

            ZStack(alignment: .leading) {
                RoundedRectangle(cornerRadius: 2)
                    .fill(WxTheme.snwChassis)
                    .frame(width: 48, height: 4)

                Circle()
                    .fill(WxTheme.snwGold)
                    .frame(width: 6, height: 6)
                    .offset(x: max(0, min(42, 48 * ratio)))
            }
            .frame(width: 48, height: 6)

            Text("🌕")
                .font(.system(size: 8))
        }
    }
}
