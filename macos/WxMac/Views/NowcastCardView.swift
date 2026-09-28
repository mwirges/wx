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
            return WxTheme.snwRed
        } else if hasUpcoming {
            return WxTheme.snwAmber
        } else {
            return WxTheme.snwGreen
        }
    }

    private var statusBadgeText: String {
        if isActive {
            return "ACTIVE PRECIPITATION"
        } else if hasUpcoming {
            return "PRECIPITATION EXPECTED"
        } else {
            return "ALL CLEAR // DRY"
        }
    }

    var body: some View {
        SNWConsoleCard(
            title: "Precipitation Nowcast & Rain Timeline",
            tag: "RADAR.QPF",
            statusColor: statusColor
        ) {
            VStack(alignment: .leading, spacing: 10) {
                // Header Row: Status Badge & Headline
                HStack(alignment: .center, spacing: 8) {
                    HStack(spacing: 5) {
                        Circle()
                            .fill(statusColor)
                            .frame(width: 6, height: 6)
                            .shadow(color: statusColor.opacity(0.8), radius: 2)

                        Text(statusBadgeText)
                            .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                            .foregroundStyle(statusColor)
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(statusColor.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(statusColor.opacity(0.35), lineWidth: 0.8))

                    Spacer()

                    if let phase = nowcast.primaryPhase, !phase.isEmpty, phase != "none" {
                        HStack(spacing: 4) {
                            Image(systemName: phaseIcon(phase))
                                .font(.system(size: 9))
                                .foregroundStyle(phaseColor(phase))
                            Text(phase.uppercased())
                                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(phaseColor(phase))
                        }
                        .padding(.horizontal, 6)
                        .padding(.vertical, 3)
                        .background(WxTheme.snwChassis.opacity(0.8), in: RoundedRectangle(cornerRadius: 4))
                        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.5))
                    }
                }

                // Headline & Narrative Summary
                VStack(alignment: .leading, spacing: 3) {
                    Text(nowcast.headline ?? "No precipitation expected")
                        .font(.system(size: 13, weight: .bold, design: .rounded))
                        .foregroundStyle(WxTheme.text)

                    if let summary = nowcast.summary, !summary.isEmpty {
                        Text(summary)
                            .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.85))
                            .fixedSize(horizontal: false, vertical: true)
                    }
                }

                // Key Telemetry Metric Chips
                HStack(spacing: 6) {
                    let liquidStr = isMetric
                        ? String(format: "%.1f mm", nowcast.totalLiquidMm ?? 0)
                        : String(format: "%.2f in", nowcast.totalLiquidIn ?? 0)
                    MetricChip(label: "Liquid Total", value: liquidStr)

                    if let snowCm = nowcast.totalSnowCm, snowCm > 0.05 {
                        let snowStr = isMetric
                            ? String(format: "%.1f cm", snowCm)
                            : String(format: "%.1f in", nowcast.totalSnowIn ?? (snowCm / 2.54))
                        MetricChip(label: "Snow Total", value: snowStr)
                    }

                    let peakRate = isMetric
                        ? String(format: "%.1f mm/h", nowcast.peakRateMmh ?? 0)
                        : String(format: "%.2f in/h", nowcast.peakRateInh ?? 0)
                    MetricChip(label: "Peak Rate", value: peakRate)

                    if let intervals = nowcast.intervals, !intervals.isEmpty {
                        let spanHours = Double(intervals.count) * 15.0 / 60.0
                        MetricChip(label: "Window", value: String(format: "%.0fh", spanHours))
                    }
                }

                // Timeline Bar Chart (15-min Intervals)
                if let intervals = nowcast.intervals, !intervals.isEmpty {
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text("15-MINUTE INTENSITY TIMELINE")
                                .font(.system(size: 8, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
                            Spacer()
                            Text("HOURLY RATE (IN/H)")
                                .font(.system(size: 7.5, weight: .semibold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwCyan.opacity(0.8))
                        }

                        NowcastTimelineBarsView(intervals: Array(intervals.prefix(24)), isMetric: isMetric)
                    }
                    .padding(8)
                    .background(WxTheme.snwChassis.opacity(0.65), in: RoundedRectangle(cornerRadius: 6))
                    .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.border.opacity(0.35), lineWidth: 0.6))
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
        case "snow": return WxTheme.snwCyan
        case "freezing-rain", "freezing_rain": return Color(red: 0.6, green: 0.8, blue: 1.0)
        case "ice-pellets", "mixed": return WxTheme.snwAmber
        default: return WxTheme.snwCyan
        }
    }
}

// Visual timeline bars for 15-minute precipitation slices
struct NowcastTimelineBarsView: View {
    let intervals: [PrecipIntervalDTO]
    let isMetric: Bool

    // Max rate across intervals to scale bars
    private var maxRate: Double {
        let maxVal = intervals.map { $0.rateInh ?? 0 }.max() ?? 0
        return max(0.20, maxVal) // at least 0.20 in/h baseline
    }

    var body: some View {
        VStack(spacing: 4) {
            GeometryReader { geo in
                let barWidth = max(4.0, (geo.size.width - CGFloat(intervals.count - 1) * 3.0) / CGFloat(intervals.count))
                HStack(alignment: .bottom, spacing: 3) {
                    ForEach(intervals) { iv in
                        let rate = iv.rateInh ?? 0
                        let heightRatio = min(1.0, rate / maxRate)
                        let barHeight = max(3.0, CGFloat(heightRatio) * geo.size.height)

                        VStack(spacing: 1) {
                            Spacer(minLength: 0)
                            RoundedRectangle(cornerRadius: 1.5)
                                .fill(barColor(for: rate, phase: iv.phase))
                                .frame(width: barWidth, height: barHeight)
                                .shadow(color: barColor(for: rate, phase: iv.phase).opacity(rate > 0.05 ? 0.6 : 0), radius: 2)
                        }
                        .help(tooltipText(for: iv))
                    }
                }
            }
            .frame(height: 38)

            // Time Labels Row (spaced intervals)
            HStack {
                if let first = intervals.first {
                    Text(first.formattedTime)
                        .font(.system(size: 7.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
                }
                Spacer()
                if intervals.count > 8 {
                    let midIdx = intervals.count / 2
                    Text(intervals[midIdx].formattedTime)
                        .font(.system(size: 7.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
                }
                Spacer()
                if let last = intervals.last {
                    Text(last.formattedTime)
                        .font(.system(size: 7.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
                }
            }
        }
    }

    private func barColor(for rateInh: Double, phase: String?) -> Color {
        if let phase = phase?.lowercased(), phase == "snow" {
            return Color.white
        }
        if rateInh >= 0.35 {
            return WxTheme.snwRed
        } else if rateInh >= 0.15 {
            return WxTheme.snwAmber
        } else if rateInh >= 0.03 {
            return WxTheme.snwCyan
        } else if rateInh > 0.001 {
            return WxTheme.snwCyan.opacity(0.5)
        } else {
            return WxTheme.border.opacity(0.25)
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
