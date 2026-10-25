import ActivityKit
import Foundation

struct LiveActivitySyncResult: Equatable {
    var status: String
    var id: String
}

/// Starts, updates, and ends one local nowcast activity. `pushType` stays nil.
@MainActor
enum NowcastLiveActivitySync {
    static func sync(cue: LiveActivityCue) async -> LiveActivitySyncResult {
        guard ActivityAuthorizationInfo().areActivitiesEnabled else {
            return LiveActivitySyncResult(status: "unavailable", id: "none")
        }
        let state = NowcastAttributes.ContentState(
            headline: cue.headline,
            phase: cue.phase,
            minutesLabel: cue.minutesLabel,
            sparkline: cue.sparkline,
            temperatureText: cue.temperatureText,
            detail: cue.detail,
            cessationText: cue.cessationText
        )
        let content = ActivityContent(state: state, staleDate: nil)
        let running = Activity<NowcastAttributes>.activities
        if !cue.shouldRun {
            guard !running.isEmpty else {
                return LiveActivitySyncResult(status: "inactive", id: "none")
            }
            for activity in running {
                await activity.end(nil, dismissalPolicy: .immediate)
            }
            let leftover = Activity<NowcastAttributes>.activities
            if leftover.isEmpty {
                return LiveActivitySyncResult(status: "ended", id: "none")
            }
            return LiveActivitySyncResult(status: "failed: activity still running", id: leftover[0].id)
        }
        if let current = running.first {
            await current.update(content)
            for extra in running.dropFirst() {
                await extra.end(nil, dismissalPolicy: .immediate)
            }
            return LiveActivitySyncResult(status: "active", id: current.id)
        }
        do {
            let activity = try Activity.request(
                attributes: NowcastAttributes(location: cue.location),
                content: content,
                pushType: nil
            )
            return LiveActivitySyncResult(status: "active", id: activity.id)
        } catch {
            return LiveActivitySyncResult(status: classify(error), id: "none")
        }
    }

    private static func classify(_ error: Error) -> String {
        let ns = error as NSError
        let text = ns.localizedDescription.lowercased()
        if text.contains("disabled") || text.contains("unavailable") || text.contains("not available") || text.contains("unsupported") {
            return "unavailable"
        }
        if ns.domain.contains("ActivityKit") {
            return "unavailable"
        }
        return "failed: \(ns.domain) \(ns.code) \(ns.localizedDescription)"
    }
}
