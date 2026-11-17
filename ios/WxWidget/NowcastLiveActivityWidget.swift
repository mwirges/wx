import ActivityKit
import SwiftUI
import WidgetKit

/// Lock screen and Dynamic Island presentation. The app pushes local state.
struct NowcastLiveActivityWidget: Widget {
    private func precipSymbol(for phase: String) -> String {
        let lower = phase.lowercased()
        if lower.contains("snow") {
            return "cloud.snow.fill"
        }
        if lower.contains("sleet") || lower.contains("ice") || lower.contains("freez") {
            return "cloud.sleet.fill"
        }
        return "cloud.rain.fill"
    }

    private func hasSparkBars(_ sparkline: String) -> Bool {
        !sparkline.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    var body: some WidgetConfiguration {
        ActivityConfiguration(for: NowcastAttributes.self) { context in
            let symbol = precipSymbol(for: context.state.phase)
            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 6) {
                    Image(systemName: symbol)
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(WidgetColor.cyan)
                    Text(context.attributes.location)
                        .font(.system(size: 12, weight: .semibold, design: .rounded))
                        .foregroundStyle(WidgetColor.secondary)
                        .lineLimit(1)
                    Spacer(minLength: 4)
                    Text(context.state.minutesLabel.uppercased())
                        .font(.system(size: 11, weight: .bold, design: .monospaced))
                        .foregroundStyle(WidgetColor.cyan)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 2.5)
                        .background(WidgetColor.cyan.opacity(0.16), in: Capsule())
                }

                HStack(alignment: .center, spacing: 10) {
                    Text(context.state.temperatureText)
                        .font(.system(size: 26, weight: .bold, design: .rounded))
                        .foregroundStyle(WidgetColor.text)

                    VStack(alignment: .leading, spacing: 2) {
                        Text(context.state.headline)
                            .font(.system(size: 14, weight: .semibold, design: .rounded))
                            .foregroundStyle(WidgetColor.text)
                            .lineLimit(2)
                        if !context.state.cessationText.isEmpty {
                            Text("until \(context.state.cessationText)")
                                .font(.system(size: 11.5, weight: .medium, design: .rounded))
                                .foregroundStyle(WidgetColor.secondary)
                                .lineLimit(1)
                        }
                    }

                    Spacer(minLength: 0)

                    if hasSparkBars(context.state.sparkline) {
                        WidgetSparkBars(sparkline: context.state.sparkline, barHeight: 20)
                    }
                }
            }
            .padding(.vertical, 4)
            .frame(maxWidth: .infinity, alignment: .leading)
            .activityBackgroundTint(WidgetColor.chassis.opacity(0.88))
            .activitySystemActionForegroundColor(WidgetColor.cyan)
        } dynamicIsland: { context in
            let symbol = precipSymbol(for: context.state.phase)
            return DynamicIsland {
                DynamicIslandExpandedRegion(.leading) {
                    HStack(spacing: 4) {
                        Image(systemName: symbol)
                            .foregroundStyle(WidgetColor.cyan)
                        Text(context.state.temperatureText)
                            .font(.system(.headline, design: .rounded).weight(.bold))
                    }
                }
                DynamicIslandExpandedRegion(.trailing) {
                    Text(context.state.minutesLabel)
                        .font(.system(.subheadline, design: .monospaced).weight(.bold))
                        .foregroundStyle(WidgetColor.cyan)
                }
                DynamicIslandExpandedRegion(.center) {
                    Text(context.attributes.location)
                        .font(.system(.caption2, design: .rounded).weight(.semibold))
                        .foregroundStyle(WidgetColor.secondary)
                        .lineLimit(1)
                }
                DynamicIslandExpandedRegion(.bottom) {
                    HStack(spacing: 8) {
                        Text(cessationOrHeadline(context.state))
                            .font(.system(.caption, design: .rounded).weight(.medium))
                            .lineLimit(1)
                        Spacer(minLength: 4)
                        if hasSparkBars(context.state.sparkline) {
                            WidgetSparkBars(sparkline: context.state.sparkline, barHeight: 14)
                        }
                    }
                }
            } compactLeading: {
                Image(systemName: symbol)
                    .foregroundStyle(WidgetColor.cyan)
            } compactTrailing: {
                Text(context.state.minutesLabel)
                    .font(.system(.caption, design: .monospaced).weight(.bold))
                    .foregroundStyle(WidgetColor.cyan)
            } minimal: {
                Image(systemName: symbol)
                    .foregroundStyle(WidgetColor.cyan)
            }
        }
    }

    private func cessationOrHeadline(_ state: NowcastAttributes.ContentState) -> String {
        if state.cessationText.isEmpty {
            return state.headline
        }
        return "until \(state.cessationText)"
    }
}
