import SwiftUI

struct EphemerisCardView: View {
    let astronomy: AstronomyDTO

    var body: some View {
        WxGlassCard(
            title: "Solar Arc & Ephemeris",
            tag: "ASTRO.ORBIT",
            accent: WxGlassTheme.gold
        ) {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .center, spacing: 12) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("SOLAR ELEVATION")
                            .font(.system(size: 9, weight: .bold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textMuted)

                        HStack(alignment: .firstTextBaseline, spacing: 6) {
                            if let elev = astronomy.solarElevationDeg {
                                let sign = elev >= 0 ? "+" : ""
                                Text("\(sign)\(String(format: "%.1f°", elev))")
                                    .font(.system(size: 24, weight: .bold, design: .rounded))
                                    .foregroundStyle(elev >= 0 ? WxGlassTheme.gold : WxGlassTheme.cyan)
                            } else {
                                Text("—")
                                    .font(.system(size: 24, weight: .bold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textSecondary)
                            }

                            if let az = astronomy.solarAzimuthDeg {
                                Text("\(String(format: "%.0f°", az)) \(astronomy.solarAzimuthCompass)")
                                    .font(.system(size: 11, weight: .semibold, design: .monospaced))
                                    .foregroundStyle(WxGlassTheme.textSecondary)
                            }
                        }
                    }

                    Spacer()

                    let regime = astronomy.currentPeriod ?? (astronomy.isPolarDay == true ? "Polar Day" : (astronomy.isPolarNight == true ? "Polar Night" : "Daylight"))
                    let regimeColor = regimeColorFor(regime)
                    HStack(spacing: 5) {
                        Circle()
                            .fill(regimeColor)
                            .frame(width: 6, height: 6)
                            .shadow(color: regimeColor.opacity(0.8), radius: 2)

                        Text(regime.uppercased())
                            .font(.system(size: 9.5, weight: .bold, design: .rounded))
                            .foregroundStyle(regimeColor)
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 4)
                    .background(regimeColor.opacity(0.14), in: Capsule())
                    .overlay(Capsule().strokeBorder(regimeColor.opacity(0.38), lineWidth: 0.8))
                }

                // Smooth Bézier Solar Horizon Curve
                WxSolarOrbitGraph(
                    solarElevationDeg: astronomy.solarElevationDeg,
                    solarAzimuthDeg: astronomy.solarAzimuthDeg
                )

                // Solar Milestones & Twilight Horizons
                VStack(spacing: 6) {
                    HStack(spacing: 6) {
                        if let sr = astronomy.sunriseFormatted {
                            horizonChip(label: "Sunrise", value: sr, icon: "sunrise.fill", color: WxGlassTheme.gold)
                        }
                        if let sn = astronomy.solarNoonFormatted {
                            horizonChip(label: "Noon", value: sn, icon: "sun.max.fill", color: WxGlassTheme.gold)
                        }
                        if let ss = astronomy.sunsetFormatted {
                            horizonChip(label: "Sunset", value: ss, icon: "sunset.fill", color: WxGlassTheme.amber)
                        }
                        if let dl = astronomy.dayLength {
                            horizonChip(label: "Daylight", value: dl, icon: "clock.fill", color: WxGlassTheme.cyan)
                        }
                    }

                    HStack(spacing: 6) {
                        if let cd = astronomy.civilDawnFormatted, let cdu = astronomy.civilDuskFormatted {
                            horizonChip(label: "Civil Twilight", value: "\(cd) – \(cdu)", icon: "sun.horizon.fill", color: WxGlassTheme.cyan)
                        }
                        if let ghm = astronomy.goldenHourMorningFormatted {
                            horizonChip(label: "AM Golden", value: ghm, icon: "camera.fill", color: Color.orange)
                        }
                    }

                    if let ghe = astronomy.goldenHourEveningFormatted {
                        HStack(spacing: 6) {
                            horizonChip(label: "PM Golden", value: ghe, icon: "camera.fill", color: Color.orange)
                            if let nd = astronomy.nauticalDawnFormatted, let ndu = astronomy.nauticalDuskFormatted {
                                horizonChip(label: "Nautical", value: "\(nd) – \(ndu)", icon: "water.waves", color: WxGlassTheme.periwinkle)
                            }
                        }
                    }
                }

                Divider().overlay(Color.white.opacity(0.10))

                // Vector Lunar Ephemeris Row (SF Symbols — Zero OS Color Emojis)
                HStack(alignment: .center, spacing: 10) {
                    let phase = astronomy.moonPhase ?? "Lunar Orbit"
                    Image(systemName: ConditionSymbol.moonSymbol(for: phase))
                        .font(.system(size: 22, weight: .medium))
                        .symbolRenderingMode(.hierarchical)
                        .foregroundStyle(WxGlassTheme.gold)
                        .frame(width: 26)

                    VStack(alignment: .leading, spacing: 2) {
                        Text(phase)
                            .font(.system(size: 12, weight: .bold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textPrimary)

                        HStack(spacing: 8) {
                            if let illum = astronomy.moonIlluminationPct {
                                Text("\(String(format: "%.0f%%", illum)) illuminated")
                                    .font(.system(size: 10, weight: .semibold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.gold)
                            }
                            if let age = astronomy.moonAgeDays {
                                Text("Age \(String(format: "%.1fd", age))")
                                    .font(.system(size: 10, weight: .medium, design: .monospaced))
                                    .foregroundStyle(WxGlassTheme.textMuted)
                            }
                        }
                    }

                    Spacer()

                    lunarCycleTrack
                }
            }
        }
    }

    private func regimeColorFor(_ regime: String) -> Color {
        switch regime.lowercased() {
        case "daylight", "day": return WxGlassTheme.gold
        case "golden hour": return Color.orange
        case "civil twilight", "nautical twilight", "astronomical twilight": return WxGlassTheme.cyan
        default: return WxGlassTheme.periwinkle
        }
    }

    private func horizonChip(label: String, value: String, icon: String, color: Color) -> some View {
        HStack(spacing: 5) {
            Image(systemName: icon)
                .font(.system(size: 9.5, weight: .semibold))
                .foregroundStyle(color)

            VStack(alignment: .leading, spacing: 1) {
                Text(label.uppercased())
                    .font(.system(size: 8, weight: .bold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textMuted)
                Text(value)
                    .font(.system(size: 10.5, weight: .bold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textPrimary)
                    .lineLimit(1)
                    .minimumScaleFactor(0.8)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 8)
        .padding(.vertical, 6)
        .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
        .overlay(WxGlassTheme.nestedShape().strokeBorder(Color.white.opacity(0.10), lineWidth: 0.7))
    }

    @ViewBuilder
    private var lunarCycleTrack: some View {
        let age = astronomy.moonAgeDays ?? 0
        let ratio = min(1.0, max(0.0, age / 29.530588853))

        HStack(spacing: 5) {
            Image(systemName: "moonphase.new.moon")
                .font(.system(size: 9.5))
                .foregroundStyle(WxGlassTheme.textMuted)

            ZStack(alignment: .leading) {
                Capsule()
                    .fill(Color.white.opacity(0.12))
                    .frame(width: 50, height: 5)

                Circle()
                    .fill(WxGlassTheme.gold)
                    .frame(width: 7, height: 7)
                    .shadow(color: WxGlassTheme.gold.opacity(0.7), radius: 2)
                    .offset(x: max(0, min(43, 50 * ratio)))
            }
            .frame(width: 50, height: 7)

            Image(systemName: "moonphase.full.moon")
                .font(.system(size: 9.5))
                .foregroundStyle(WxGlassTheme.gold)
        }
    }
}
