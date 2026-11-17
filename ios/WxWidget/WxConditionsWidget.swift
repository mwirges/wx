import SwiftUI
import UIKit
import WidgetKit

enum WidgetColor {
    static let chassis = Color(red: 0x06 / 255, green: 0x0F / 255, blue: 0x20 / 255)
    static let text = Color(red: 0xF3 / 255, green: 0xF8 / 255, blue: 0xFF / 255)
    static let secondary = Color(red: 0xB4 / 255, green: 0xC5 / 255, blue: 0xDF / 255)
    static let muted = Color(red: 0x8C / 255, green: 0xA1 / 255, blue: 0xC1 / 255)
    static let cyan = Color(red: 0x38 / 255, green: 0xE1 / 255, blue: 0xFF / 255)
    static let gold = Color(red: 0xF5 / 255, green: 0xA6 / 255, blue: 0x23 / 255)
    static let periwinkle = Color(red: 0x9A / 255, green: 0xAE / 255, blue: 0xFF / 255)
}

/// Zero-blur analytic sky canvas so WidgetKit stays far below its 30 MB Jetsam ceiling.
struct WidgetSkyCanvas: View {
    var wet: Bool
    var isNight: Bool = false

    var body: some View {
        GeometryReader { geo in
            let maxDim = max(geo.size.width, geo.size.height)
            ZStack {
                LinearGradient(
                    colors: wet
                        ? [Color(red: 0.04, green: 0.11, blue: 0.23), Color(red: 0.05, green: 0.16, blue: 0.31), Color(red: 0.02, green: 0.05, blue: 0.11)]
                        : (isNight
                            ? [Color(red: 0.02, green: 0.04, blue: 0.10), Color(red: 0.04, green: 0.09, blue: 0.19), Color(red: 0.01, green: 0.02, blue: 0.06)]
                            : [Color(red: 0.05, green: 0.15, blue: 0.30), Color(red: 0.07, green: 0.20, blue: 0.37), Color(red: 0.03, green: 0.07, blue: 0.14)]),
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )

                RadialGradient(
                    colors: [(isNight && !wet ? WidgetColor.periwinkle : WidgetColor.cyan).opacity(0.28), .clear],
                    center: UnitPoint(x: 0.18, y: 0.12),
                    startRadius: 0,
                    endRadius: maxDim * 0.72
                )

                RadialGradient(
                    colors: [(wet || isNight ? Color.indigo : WidgetColor.gold).opacity(0.20), .clear],
                    center: UnitPoint(x: 0.88, y: 0.82),
                    startRadius: 0,
                    endRadius: maxDim * 0.65
                )
            }
        }
    }
}

/// Converts sparkline runes (" ▂▃▄▅▆▇█") into crisp rounded capsule intensity bars.
struct WidgetSparkBars: View {
    let sparkline: String
    var barHeight: CGFloat = 14

    private var levels: [CGFloat] {
        sparkline.prefix(16).map { ch in
            switch ch {
            case "▂": return 0.25
            case "▃": return 0.38
            case "▄": return 0.52
            case "▅": return 0.66
            case "▆": return 0.80
            case "▇": return 0.90
            case "█": return 1.00
            default: return 0.14
            }
        }
    }

    var body: some View {
        HStack(alignment: .bottom, spacing: 2.5) {
            ForEach(Array(levels.enumerated()), id: \.offset) { _, lvl in
                Capsule(style: .continuous)
                    .fill(lvl > 0.65 ? WidgetColor.gold : (lvl > 0.18 ? WidgetColor.cyan : Color.white.opacity(0.18)))
                    .frame(width: 4, height: max(3, barHeight * lvl))
            }
        }
        .frame(height: barHeight, alignment: .bottom)
    }
}

struct WxConditionsWidget: Widget {
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: "com.mwirges.wx.ios.conditions", provider: WxConditionsProvider()) { entry in
            WxConditionsView(entry: entry)
        }
        .configurationDisplayName("wx")
        .description("Temperature, precipitation, and a radar thumbnail.")
        .supportedFamilies([
            .systemSmall,
            .systemMedium,
            .accessoryInline,
            .accessoryRectangular,
            .accessoryCircular,
        ])
    }
}

struct WxConditionsView: View {
    @Environment(\.widgetFamily) private var family
    var entry: WxWidgetEntry

    private var weatherSymbol: String {
        if entry.snapshot.cue.shouldRun {
            if let sym = entry.snapshot.symbolName,
               sym.contains("snow") || sym.contains("sleet") || sym.contains("bolt") || sym.contains("rain") {
                return sym
            }
            return "cloud.rain.fill"
        }
        if let sym = entry.snapshot.symbolName, !sym.isEmpty {
            return sym
        }
        return entry.snapshot.isNight == true ? "moon.stars.fill" : "sun.max.fill"
    }

    private var symbolTint: Color {
        let sym = weatherSymbol
        if sym.contains("bolt") { return WidgetColor.gold }
        if sym.contains("rain") || sym.contains("snow") || sym.contains("sleet") { return WidgetColor.cyan }
        if sym.contains("moon") || entry.snapshot.isNight == true { return WidgetColor.periwinkle }
        if sym.contains("sun") { return WidgetColor.gold }
        return WidgetColor.secondary
    }

    /// Dry nowcasts emit 24 space characters; only show intensity bars when precipitation runes exist.
    var hasSparkBars: Bool {
        !entry.snapshot.sparkline.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    var rectangularHeaderText: String {
        let temp = entry.snapshot.temperatureText
        let hi = entry.snapshot.highText
        let lo = entry.snapshot.lowText
        let hasHi = !hi.isEmpty && hi != "—"
        let hasLo = !lo.isEmpty && lo != "—"
        if hasHi && hasLo {
            return "\(temp)  H:\(hi) L:\(lo)"
        }
        if hasLo {
            return "\(temp)  Low \(lo)"
        }
        if hasHi {
            return "\(temp)  High \(hi)"
        }
        return temp
    }

    var body: some View {
        content
            .containerBackground(for: .widget) { background }
    }

    @ViewBuilder
    private var content: some View {
        switch family {
        case .accessoryInline:
            Label {
                Text("\(entry.snapshot.temperatureText) · \(entry.snapshot.precipHeadline)")
                    .lineLimit(1)
            } icon: {
                Image(systemName: weatherSymbol)
            }
        case .accessoryCircular:
            ZStack {
                AccessoryWidgetBackground()
                VStack(spacing: 1) {
                    Image(systemName: weatherSymbol)
                        .font(.system(size: 11, weight: .semibold))
                    Text(entry.snapshot.temperatureText)
                        .font(.system(size: 15, weight: .bold, design: .rounded))
                        .minimumScaleFactor(0.6)
                }
            }
        case .accessoryRectangular:
            rectangular
        case .systemMedium:
            medium
        default:
            small
        }
    }

    private var rectangular: some View {
        VStack(alignment: .leading, spacing: 2) {
            HStack(spacing: 4) {
                Image(systemName: weatherSymbol)
                    .font(.caption.weight(.semibold))
                Text(rectangularHeaderText)
                    .font(.system(.headline, design: .rounded).weight(.bold))
                    .lineLimit(1)
                    .minimumScaleFactor(0.7)
            }
            Text(entry.snapshot.precipHeadline)
                .font(.caption)
                .lineLimit(1)
            if hasSparkBars {
                WidgetSparkBars(sparkline: entry.snapshot.sparkline, barHeight: 9)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var small: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 5) {
                Circle()
                    .fill(WidgetColor.cyan)
                    .frame(width: 5, height: 5)
                Text(entry.snapshot.location)
                    .font(.system(size: 11, weight: .semibold, design: .rounded))
                    .foregroundStyle(WidgetColor.secondary)
                    .lineLimit(1)
            }

            HStack(alignment: .center, spacing: 6) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(entry.snapshot.temperatureText)
                        .font(.system(size: 30, weight: .bold, design: .rounded))
                        .foregroundStyle(WidgetColor.text)
                        .minimumScaleFactor(0.6)
                    HStack(spacing: 4) {
                        Image(systemName: weatherSymbol)
                            .font(.system(size: 10.5, weight: .semibold))
                            .symbolRenderingMode(.hierarchical)
                            .foregroundStyle(symbolTint)
                        Text(entry.snapshot.condition)
                            .font(.system(size: 11, weight: .medium, design: .rounded))
                            .foregroundStyle(WidgetColor.secondary)
                            .lineLimit(1)
                    }
                }
                Spacer(minLength: 0)
                radar(side: 50)
            }

            Spacer(minLength: 0)

            Text(entry.snapshot.precipHeadline)
                .font(.system(size: 11, weight: .semibold, design: .rounded))
                .foregroundStyle(WidgetColor.cyan)
                .lineLimit(2)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private var medium: some View {
        VStack(alignment: .leading, spacing: 5) {
            HStack(alignment: .center, spacing: 8) {
                VStack(alignment: .leading, spacing: 1) {
                    Text(entry.snapshot.location)
                        .font(.system(size: 11, weight: .semibold, design: .rounded))
                        .foregroundStyle(WidgetColor.secondary)
                        .lineLimit(1)
                    HStack(alignment: .firstTextBaseline, spacing: 6) {
                        Image(systemName: weatherSymbol)
                            .font(.system(size: 18, weight: .semibold))
                            .symbolRenderingMode(.hierarchical)
                            .foregroundStyle(symbolTint)
                        Text(entry.snapshot.temperatureText)
                            .font(.system(size: 26, weight: .bold, design: .rounded))
                            .foregroundStyle(WidgetColor.text)
                        Text(entry.snapshot.precipHeadline)
                            .font(.system(size: 11.5, weight: .semibold, design: .rounded))
                            .foregroundStyle(WidgetColor.cyan)
                            .lineLimit(1)
                    }
                }
                Spacer(minLength: 4)
                if hasSparkBars {
                    WidgetSparkBars(sparkline: entry.snapshot.sparkline, barHeight: 16)
                }
                radar(side: 44)
            }

            HStack(alignment: .top, spacing: 12) {
                HStack(spacing: 6) {
                    ForEach(Array(entry.snapshot.hours.prefix(4).enumerated()), id: \.offset) { _, chip in
                        VStack(spacing: 1) {
                            Text(chip.label)
                                .font(.system(size: 9.5, weight: .medium, design: .rounded))
                                .foregroundStyle(WidgetColor.muted)
                                .lineLimit(1)
                            Text(chip.value)
                                .font(.system(size: 10.5, weight: .bold, design: .rounded))
                                .foregroundStyle(WidgetColor.text)
                                .lineLimit(1)
                        }
                        .padding(.horizontal, 6)
                        .padding(.vertical, 4)
                        .background(Color.white.opacity(0.07), in: RoundedRectangle(cornerRadius: 8, style: .continuous))
                    }
                }
                Spacer(minLength: 0)
            }
            Spacer(minLength: 0)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    @ViewBuilder
    private func radar(side: CGFloat) -> some View {
        let squircle = RoundedRectangle(cornerRadius: 12, style: .continuous)
        if let data = entry.radar, let image = UIImage(data: data) {
            Image(uiImage: image)
                .resizable()
                .interpolation(.medium)
                .scaledToFill()
                .frame(width: side, height: side)
                .background(Color.black.opacity(0.35))
                .clipShape(squircle)
                .overlay(squircle.strokeBorder(Color.white.opacity(0.24), lineWidth: 0.9))
        }
    }

    @ViewBuilder
    private var background: some View {
        switch family {
        case .accessoryInline, .accessoryRectangular, .accessoryCircular:
            AccessoryWidgetBackground()
        default:
            WidgetSkyCanvas(wet: entry.snapshot.cue.shouldRun, isNight: entry.snapshot.isNight == true)
        }
    }
}
