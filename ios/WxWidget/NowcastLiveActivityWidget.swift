import ActivityKit
import SwiftUI
import WidgetKit

/// Lock screen and Dynamic Island presentation. The app pushes local state.
struct NowcastLiveActivityWidget: Widget {
    var body: some WidgetConfiguration {
        ActivityConfiguration(for: NowcastAttributes.self) { context in
            VStack(alignment: .leading, spacing: 4) {
                Text(context.attributes.location)
                    .font(.caption)
                    .foregroundStyle(WidgetColor.secondary)
                    .lineLimit(1)
                HStack(alignment: .firstTextBaseline) {
                    Text(context.state.temperatureText)
                        .font(.title2.weight(.semibold))
                        .foregroundStyle(WidgetColor.text)
                    Text(context.state.minutesLabel)
                        .font(.headline.weight(.semibold))
                        .foregroundStyle(WidgetColor.cyan)
                    Spacer(minLength: 0)
                }
                Text(context.state.headline)
                    .font(.subheadline)
                    .foregroundStyle(WidgetColor.text)
                    .lineLimit(2)
                if !context.state.sparkline.isEmpty {
                    Text(context.state.sparkline)
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(WidgetColor.text)
                        .lineLimit(1)
                }
                if !context.state.cessationText.isEmpty {
                    Text("until \(context.state.cessationText)")
                        .font(.caption)
                        .foregroundStyle(WidgetColor.secondary)
                        .lineLimit(1)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .activityBackgroundTint(WidgetColor.chassis)
            .activitySystemActionForegroundColor(WidgetColor.cyan)
        } dynamicIsland: { context in
            DynamicIsland {
                DynamicIslandExpandedRegion(.leading) {
                    Text(context.state.temperatureText)
                        .font(.headline.weight(.semibold))
                }
                DynamicIslandExpandedRegion(.trailing) {
                    Text(context.state.minutesLabel)
                        .font(.headline.weight(.semibold))
                }
                DynamicIslandExpandedRegion(.center) {
                    Text(context.attributes.location)
                        .font(.caption2)
                        .lineLimit(1)
                }
                DynamicIslandExpandedRegion(.bottom) {
                    Text(context.state.sparkline.isEmpty ? context.state.headline : context.state.sparkline)
                        .font(.system(.caption, design: .monospaced))
                        .lineLimit(1)
                    Text(cessationOrHeadline(context.state))
                        .font(.caption2)
                        .lineLimit(1)
                }
            } compactLeading: {
                Text("💧")
            } compactTrailing: {
                Text(context.state.minutesLabel)
                    .font(.caption.weight(.semibold))
            } minimal: {
                Text("💧")
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
