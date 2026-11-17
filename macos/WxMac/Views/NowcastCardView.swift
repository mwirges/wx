import SwiftUI

struct NowcastCardView: View {
    @EnvironmentObject var store: WeatherStore
    let nowcast: NowcastDTO

    private var isMetric: Bool {
        store.units == "metric"
    }

    private var isActive: Bool {
        nowcast.isActivePrecip == true
    }

    private var hasUpcoming: Bool {
        nowcast.nextPrecipTime != nil
    }

    private var statusColor: Color {
        if isActive {
            return WxGlassTheme.conditionRed
        } else if hasUpcoming {
            return WxGlassTheme.amber
        } else {
            return WxGlassTheme.nominalGreen
        }
    }

    private var statusBadgeText: String {
        if isActive {
            return "ACTIVE PRECIPITATION"
        } else if hasUpcoming {
            return "PRECIPITATION EXPECTED"
        } else {
            return "ALL CLEAR · DRY"
        }
    }

    var body: some View {
        WxGlassCard(
            title: "Precipitation Nowcast",
            tag: "RADAR.QPF",
            accent: statusColor
        ) {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .center, spacing: 8) {
                    HStack(spacing: 5) {
                        Circle()
                            .fill(statusColor)
                            .frame(width: 6, height: 6)
                            .shadow(color: statusColor.opacity(0.8), radius: 2)

                        Text(statusBadgeText)
                            .font(.system(size: 9, weight: .bold, design: .rounded))
                            .foregroundStyle(statusColor)
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 4)
                    .background(statusColor.opacity(0.14), in: Capsule())
                    .overlay(Capsule().strokeBorder(statusColor.opacity(0.38), lineWidth: 0.8))

                    Spacer()

                    if let phase = nowcast.primaryPhase, !phase.isEmpty, phase.caseInsensitiveCompare("none") != .orderedSame {
                        HStack(spacing: 4) {
                            Image(systemName: phaseIcon(phase))
                                .font(.system(size: 9.5))
                                .foregroundStyle(phaseColor(phase))
                            Text(phase.uppercased())
                                .font(.system(size: 9, weight: .bold, design: .rounded))
                                .foregroundStyle(phaseColor(phase))
                        }
                        .padding(.horizontal, 9)
                        .padding(.vertical, 3.5)
                        .background(Color.white.opacity(0.06), in: Capsule())
                        .overlay(Capsule().strokeBorder(Color.white.opacity(0.14), lineWidth: 0.7))
                    }
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text(nowcast.headline ?? "No precipitation expected")
                        .font(.system(size: 14, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)

                    if let summary = nowcast.summary, !summary.isEmpty {
                        Text(summary)
                            .font(.system(size: 11.5))
                            .foregroundStyle(WxGlassTheme.textSecondary)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                }

                HStack(spacing: 7) {
                    let liquidStr = isMetric
                        ? String(format: "%.1f mm", nowcast.totalLiquidMm ?? 0)
                        : String(format: "%.2f in", nowcast.totalLiquidIn ?? 0)
                    MetricChip(label: "Liquid Total", value: liquidStr, symbol: "drop.fill")

                    if let snowCm = nowcast.totalSnowCm, snowCm > 0.05 {
                        let snowStr = isMetric
                            ? String(format: "%.1f cm", snowCm)
                            : String(format: "%.1f in", nowcast.totalSnowIn ?? (snowCm / 2.54))
                        MetricChip(label: "Snow Total", value: snowStr, symbol: "snowflake")
                    }

                    let peakRate = isMetric
                        ? String(format: "%.1f mm/h", nowcast.peakRateMmh ?? 0)
                        : String(format: "%.2f in/h", nowcast.peakRateInh ?? 0)
                    MetricChip(label: "Peak Rate", value: peakRate, symbol: "cloud.heavyrain.fill")

                    if let intervals = nowcast.intervals, !intervals.isEmpty {
                        let spanHours = Double(intervals.count) * 15.0 / 60.0
                        MetricChip(label: "Window", value: String(format: "%.0fh", spanHours), symbol: "clock.fill")
                    }
                }

                if let intervals = nowcast.intervals, !intervals.isEmpty {
                    VStack(alignment: .leading, spacing: 5) {
                        HStack {
                            Text("15-MINUTE INTENSITY TIMELINE")
                                .font(.system(size: 8.5, weight: .bold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.textMuted)
                            Spacer()
                            Text(isMetric ? "RATE (MM/H)" : "RATE (IN/H)")
                                .font(.system(size: 8, weight: .semibold, design: .monospaced))
                                .foregroundStyle(WxGlassTheme.cyanSoft)
                        }

                        NowcastTimelineBarsView(intervals: Array(intervals.prefix(24)), isMetric: isMetric)
                    }
                    .padding(10)
                    .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
                    .overlay(WxGlassTheme.nestedShape().strokeBorder(Color.white.opacity(0.10), lineWidth: 0.7))
                }
            }
        }
    }

    private func phaseIcon(_ phase: String) -> String {
        switch phase.lowercased() {
        case "snow": return "snowflake"
        case "freezing-rain", "freezing_rain": return "thermometer.snowflake"
        case "ice-pellets", "mixed": return "cloud.hail.fill"
        default: return "cloud.rain.fill"
        }
    }

    private func phaseColor(_ phase: String) -> Color {
        switch phase.lowercased() {
        case "snow": return WxGlassTheme.cyan
        case "freezing-rain", "freezing_rain": return WxGlassTheme.periwinkle
        case "ice-pellets", "mixed": return WxGlassTheme.amber
        default: return WxGlassTheme.cyan
        }
    }
}

struct NowcastTimelineBarsView: View {
    let intervals: [PrecipIntervalDTO]
    let isMetric: Bool

    private var maxRate: Double {
        let maxVal = intervals.map { $0.rateInh ?? 0 }.max() ?? 0
        return max(0.20, maxVal)
    }

    var body: some View {
        VStack(spacing: 4) {
            GeometryReader { geo in
                let barWidth = max(4.0, (geo.size.width - CGFloat(intervals.count - 1) * 3.0) / CGFloat(intervals.count))
                HStack(alignment: .bottom, spacing: 3) {
                    ForEach(intervals) { iv in
                        let rate = iv.rateInh ?? 0
                        let heightRatio = min(1.0, rate / maxRate)
                        let barHeight = max(3.5, CGFloat(heightRatio) * geo.size.height)

                        VStack(spacing: 0) {
                            Spacer(minLength: 0)
                            Capsule(style: .continuous)
                                .fill(barColor(for: rate, phase: iv.phase))
                                .frame(width: barWidth, height: barHeight)
                                .shadow(color: barColor(for: rate, phase: iv.phase).opacity(rate > 0.05 ? 0.55 : 0), radius: 3)
                        }
                        .help(tooltipText(for: iv))
                    }
                }
            }
            .frame(height: 38)

            HStack {
                if let first = intervals.first {
                    Text(first.formattedTime)
                }
                Spacer()
                if intervals.count > 8 {
                    Text(intervals[intervals.count / 2].formattedTime)
                }
                Spacer()
                if let last = intervals.last {
                    Text(last.formattedTime)
                }
            }
            .font(.system(size: 8.5, weight: .medium, design: .monospaced))
            .foregroundStyle(WxGlassTheme.textMuted)
        }
    }

    private func barColor(for rateInh: Double, phase: String?) -> Color {
        if let phase = phase?.lowercased(), phase == "snow" {
            return Color.white
        }
        if rateInh >= 0.35 {
            return WxGlassTheme.conditionRed
        } else if rateInh >= 0.15 {
            return WxGlassTheme.amber
        } else if rateInh >= 0.03 {
            return WxGlassTheme.cyan
        } else if rateInh > 0.001 {
            return WxGlassTheme.cyan.opacity(0.55)
        } else {
            return Color.white.opacity(0.14)
        }
    }

    private func tooltipText(for iv: PrecipIntervalDTO) -> String {
        let time = iv.formattedTime
        let summary = iv.summary ?? "Clear"
        let prob = String(format: "%.0f%%", iv.probability ?? 0)
        let rateStr = isMetric
            ? String(format: "%.1f mm/h", iv.rateMmh ?? 0)
            : String(format: "%.2f in/h", iv.rateInh ?? 0)
        return "\(time) · \(summary) (\(prob))\nRate: \(rateStr)"
    }
}
