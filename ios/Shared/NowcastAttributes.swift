import ActivityKit
import Foundation

/// Shared by the app (which starts the activity) and the widget (which draws it).
/// Same source file in both targets. Local updates only.
struct NowcastAttributes: ActivityAttributes {
    struct ContentState: Codable, Hashable {
        var headline: String
        var phase: String
        var minutesLabel: String
        var sparkline: String
        var temperatureText: String
        var detail: String
        var cessationText: String
    }

    var location: String
}
