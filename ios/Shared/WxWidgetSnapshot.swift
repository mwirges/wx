import Foundation

enum WxAppGroup {
    static let id = "group.com.mwirges.wx"
    static let snapshotFile = "widget-snapshot.json"
    static let radarFile = "radar-thumb.png"
    static let fallbackFolder = "wx-widget-cache"
}

enum WxWidgetLimits {
    /// Longest edge of the single radar frame stored for the widget.
    static let radarMaxPixel = 72
    /// Refuse anything larger before the widget decodes it. Far under the 30 MB cap.
    static let radarMaxBytes = 32 * 1024
    static let timelineMinutes = 15
    static let textLimit = 120
}

enum WxWidgetText {
    static func clip(_ raw: String, limit: Int = WxWidgetLimits.textLimit) -> String {
        if raw.count <= limit { return raw }
        return String(raw.prefix(limit))
    }
}

enum WxTemperature {
    static func text(f: Double?, c: Double?, units: String) -> String {
        if units == "metric", let c {
            return String(format: "%.0f°", c)
        }
        if let f {
            return String(format: "%.0f°", f)
        }
        if let c {
            return String(format: "%.0f°", c)
        }
        return "—"
    }
}

struct LiveActivityCue: Codable, Equatable, Sendable {
    var shouldRun: Bool
    var location: String
    var headline: String
    var phase: String
    var minutesLabel: String
    var sparkline: String
    var temperatureText: String
    var detail: String
    var cessationText: String

    static let idle = LiveActivityCue(
        shouldRun: false,
        location: "wx",
        headline: "—",
        phase: "none",
        minutesLabel: "—",
        sparkline: "",
        temperatureText: "—",
        detail: "",
        cessationText: ""
    )

    static func make(
        sample: NowcastWindowSample,
        location: String,
        headline: String,
        phase: String,
        sparkline: String,
        temperatureText: String,
        detail: String,
        cessationText: String
    ) -> LiveActivityCue {
        LiveActivityCue(
            shouldRun: LiveActivityRule.showsPrecipitation(in: sample),
            location: WxWidgetText.clip(location),
            headline: WxWidgetText.clip(headline),
            phase: WxWidgetText.clip(phase, limit: 32),
            minutesLabel: LiveActivityRule.minutesLabel(in: sample),
            sparkline: WxWidgetText.clip(sparkline, limit: 32),
            temperatureText: WxWidgetText.clip(temperatureText, limit: 16),
            detail: WxWidgetText.clip(detail),
            cessationText: WxWidgetText.clip(cessationText, limit: 32)
        )
    }
}

/// What the widget extension is allowed to read. One nowcast, no radar loop.
struct WxWidgetSnapshot: Codable, Equatable, Sendable {
    struct Chip: Codable, Equatable, Sendable {
        var label: String
        var value: String
    }

    var updatedAt: Date
    var location: String
    var units: String
    var temperatureText: String
    var condition: String
    var highText: String
    var lowText: String
    var precipHeadline: String
    var precipSummary: String
    var sparkline: String
    var hours: [Chip]
    var days: [Chip]
    var cue: LiveActivityCue

    static let placeholder = WxWidgetSnapshot(
        updatedAt: Date(timeIntervalSince1970: 0),
        location: "wx",
        units: "imperial",
        temperatureText: "—",
        condition: "No cache",
        highText: "—",
        lowText: "—",
        precipHeadline: "—",
        precipSummary: "",
        sparkline: "",
        hours: [],
        days: [],
        cue: .idle
    )
}
