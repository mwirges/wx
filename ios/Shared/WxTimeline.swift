import Foundation
import WidgetKit

struct WxWidgetEntry: TimelineEntry, Equatable {
    var date: Date
    var snapshot: WxWidgetSnapshot
    var radar: Data?
}

struct WxTimelinePlan: Equatable {
    var date: Date
    var reloadAfter: Date
    var snapshot: WxWidgetSnapshot
    var showsRadar: Bool
    var radarByteCount: Int
}

enum WxTimelinePlanBuilder {
    static func plan(reading directory: URL, now: Date) -> WxTimelinePlan {
        let (snapshot, radar) = WxWidgetCache.read(from: directory)
        return WxTimelinePlan(
            date: now,
            reloadAfter: now.addingTimeInterval(TimeInterval(WxWidgetLimits.timelineMinutes * 60)),
            snapshot: snapshot,
            showsRadar: radar != nil,
            radarByteCount: radar?.count ?? 0
        )
    }
}

/// Reads the shared cache only. It does not call WxCore or the network.
struct WxConditionsProvider: TimelineProvider {
    var directory: URL?

    func placeholder(in context: Context) -> WxWidgetEntry {
        WxWidgetEntry(date: Date(), snapshot: .placeholder, radar: nil)
    }

    func getSnapshot(in context: Context, completion: @escaping (WxWidgetEntry) -> Void) {
        completion(entry(now: Date()))
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<WxWidgetEntry>) -> Void) {
        completion(makeTimeline(now: Date()))
    }

    func makeTimeline(now: Date) -> Timeline<WxWidgetEntry> {
        let entry = entry(now: now)
        let reload = now.addingTimeInterval(TimeInterval(WxWidgetLimits.timelineMinutes * 60))
        return Timeline(entries: [entry], policy: .after(reload))
    }

    func entry(now: Date) -> WxWidgetEntry {
        let (snapshot, radar) = WxWidgetCache.read(from: directory ?? WxWidgetCache.liveDirectory())
        return WxWidgetEntry(date: now, snapshot: snapshot, radar: radar)
    }
}
