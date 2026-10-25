import Foundation

protocol IOSClock {
    func now() -> Date
}

struct WallClock: IOSClock {
    func now() -> Date { Date() }
}

/// Frozen or stepped clock for UI tests. Production uses WallClock.
final class AdjustableClock: IOSClock {
    private var instant: Date

    init(_ instant: Date) {
        self.instant = instant
    }

    func now() -> Date { instant }

    func advance(by interval: TimeInterval) {
        instant = instant.addingTimeInterval(interval)
    }
}

enum IOSClocks {
    /// WX_UI_TEST, WX_CLOCK=frozen, WX_NOW, or -WXClockFrozen selects the
    /// adjustable clock so a test can freeze time and step it. WX_NOW is
    /// ISO-8601. Without it the clock starts at 2026-10-25T18:00:00Z.
    static func make(
        environment: [String: String]? = nil,
        arguments: [String]? = nil
    ) -> any IOSClock {
        let env = environment ?? ProcessInfo.processInfo.environment
        let args = arguments ?? ProcessInfo.processInfo.arguments
        let frozen = env["WX_UI_TEST"] == "1"
            || env["WX_CLOCK"] == "frozen"
            || env["WX_NOW"] != nil
            || args.contains("-WXClockFrozen")
        guard frozen else { return WallClock() }
        let raw = env["WX_NOW"]?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let instant = parse(raw) ?? parse("2026-10-25T18:00:00Z") ?? Date(timeIntervalSince1970: 0)
        return AdjustableClock(instant)
    }

    static func parse(_ raw: String) -> Date? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }
        let iso = ISO8601DateFormatter()
        iso.formatOptions = [.withInternetDateTime]
        if let date = iso.date(from: trimmed) {
            return date
        }
        iso.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return iso.date(from: trimmed)
    }
}

enum IOSLoopTiming {
    /// 10 fps. The power budget is 10–15 fps, not a 60 fps draw.
    static let defaultMilliseconds = 100

    static func intervalMilliseconds(environment: [String: String]? = nil) -> Int {
        let env = environment ?? ProcessInfo.processInfo.environment
        if let raw = env["WX_LOOP_MS"], let ms = Int(raw), (16...5_000).contains(ms) {
            return ms
        }
        return defaultMilliseconds
    }
}
