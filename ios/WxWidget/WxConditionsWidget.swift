import SwiftUI
import UIKit
import WidgetKit

enum WidgetColor {
    static let chassis = Color(red: 0x0A / 255, green: 0x12 / 255, blue: 0x22 / 255)
    static let text = Color(red: 0xF0 / 255, green: 0xF6 / 255, blue: 0xFF / 255)
    static let secondary = Color(red: 0x94 / 255, green: 0xA7 / 255, blue: 0xC5 / 255)
    static let cyan = Color(red: 0x38 / 255, green: 0xE1 / 255, blue: 0xFF / 255)
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

    var body: some View {
        content
            .containerBackground(for: .widget) { background }
    }

    @ViewBuilder
    private var content: some View {
        switch family {
        case .accessoryInline:
            Text(inlineLine)
                .lineLimit(1)
        case .accessoryCircular:
            Text(entry.snapshot.temperatureText)
                .font(.headline.weight(.semibold))
                .minimumScaleFactor(0.6)
        case .accessoryRectangular:
            rectangular
        case .systemMedium:
            medium
        default:
            small
        }
    }

    private var inlineLine: String {
        let mark = entry.snapshot.cue.shouldRun ? "💧" : "☀️"
        return "\(mark) \(entry.snapshot.temperatureText) · \(entry.snapshot.precipHeadline)"
    }

    private var rectangular: some View {
        VStack(alignment: .leading, spacing: 1) {
            Text("\(entry.snapshot.temperatureText)  H \(entry.snapshot.highText)  L \(entry.snapshot.lowText)")
                .font(.headline)
                .lineLimit(1)
                .minimumScaleFactor(0.7)
            Text(entry.snapshot.precipHeadline)
                .font(.caption)
                .lineLimit(1)
            if !entry.snapshot.sparkline.isEmpty {
                Text(entry.snapshot.sparkline)
                    .font(.system(size: 12, design: .monospaced))
                    .lineLimit(1)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var small: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(entry.snapshot.location)
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(WidgetColor.secondary)
                .lineLimit(1)
            HStack(alignment: .center, spacing: 6) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(entry.snapshot.temperatureText)
                        .font(.system(size: 28, weight: .semibold, design: .rounded))
                        .foregroundStyle(WidgetColor.text)
                        .minimumScaleFactor(0.6)
                    Text(entry.snapshot.condition)
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(WidgetColor.secondary)
                        .lineLimit(1)
                    Text(entry.snapshot.precipHeadline)
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(WidgetColor.cyan)
                        .lineLimit(2)
                }
                Spacer(minLength: 0)
                radar(side: 48)
            }
            Spacer(minLength: 0)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private var medium: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(alignment: .center, spacing: 8) {
                Text(entry.snapshot.temperatureText)
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(WidgetColor.text)
                Text(entry.snapshot.precipHeadline)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(WidgetColor.cyan)
                    .lineLimit(2)
                Spacer(minLength: 4)
                radar(side: 40)
            }
            if !entry.snapshot.sparkline.isEmpty {
                Text(entry.snapshot.sparkline)
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundStyle(WidgetColor.text)
                    .lineLimit(1)
            }
            HStack(alignment: .top, spacing: 12) {
                VStack(alignment: .leading, spacing: 1) {
                    ForEach(Array(entry.snapshot.hours.prefix(6).enumerated()), id: \.offset) { _, chip in
                        Text("\(chip.label)  \(chip.value)")
                            .font(.system(size: 10, weight: .medium, design: .monospaced))
                            .foregroundStyle(WidgetColor.secondary)
                            .lineLimit(1)
                    }
                }
                Spacer(minLength: 4)
                VStack(alignment: .leading, spacing: 1) {
                    ForEach(Array(entry.snapshot.days.prefix(3).enumerated()), id: \.offset) { _, chip in
                        Text("\(chip.label)  \(chip.value)")
                            .font(.system(size: 10, weight: .medium))
                            .foregroundStyle(WidgetColor.text)
                            .lineLimit(1)
                    }
                }
            }
            Spacer(minLength: 0)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    @ViewBuilder
    private func radar(side: CGFloat) -> some View {
        if let data = entry.radar, let image = UIImage(data: data) {
            Image(uiImage: image)
                .resizable()
                .interpolation(.medium)
                .scaledToFit()
                .frame(width: side, height: side)
                .clipShape(RoundedRectangle(cornerRadius: 4))
        }
    }

    @ViewBuilder
    private var background: some View {
        switch family {
        case .accessoryInline, .accessoryRectangular, .accessoryCircular:
            AccessoryWidgetBackground()
        default:
            WidgetColor.chassis
        }
    }
}
