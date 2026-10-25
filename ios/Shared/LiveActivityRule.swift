import Foundation

/// One nowcast slice the shell already fetched. The widget never computes weather.
struct NowcastIntervalSample: Equatable, Sendable {
    var start: Date
    var end: Date?
    var precipitating: Bool
}

/// Decision input for the Live Activity. `now` is the shell clock.
struct NowcastWindowSample: Equatable, Sendable {
    var now: Date
    var isActivePrecip: Bool
    var nextPrecipTime: Date?
    var intervals: [NowcastIntervalSample]
}

/// Local Live Activity gate. No push, no APNs.
///
/// Start when the nowcast shows precipitation onset within the next 60 minutes.
/// Precipitation already falling counts: it is inside the window. A precipitating
/// nowcast interval that overlaps the window counts too.
/// Update the running activity locally when a newer nowcast still shows
/// precipitation in the window. End it when the nowcast shows none.
enum LiveActivityRule {
    static let windowSeconds: TimeInterval = 60 * 60

    static func showsPrecipitation(in sample: NowcastWindowSample) -> Bool {
        if sample.isActivePrecip {
            return true
        }
        if let next = sample.nextPrecipTime, isOnset(next, now: sample.now) {
            return true
        }
        return sample.intervals.contains { $0.precipitating && overlaps(interval: $0, now: sample.now) }
    }

    /// `now` while precipitation is falling, otherwise `Nm` until onset, otherwise an em dash.
    static func minutesLabel(in sample: NowcastWindowSample) -> String {
        if sample.isActivePrecip || coversNow(sample) {
            return "now"
        }
        guard let onset = nextOnset(sample) else { return "—" }
        let delta = onset.timeIntervalSince(sample.now)
        if delta <= 0 { return "now" }
        let minutes = max(1, Int(ceil(delta / 60)))
        return "\(minutes)m"
    }

    /// Rates and known precip phases count. A probability alone does not.
    static func isPrecipitating(
        phase: String?,
        rateInh: Double?,
        rateMmh: Double?,
        accumIn: Double?,
        accumMm: Double?
    ) -> Bool {
        if (rateInh ?? 0) > 0 || (rateMmh ?? 0) > 0 || (accumIn ?? 0) > 0 || (accumMm ?? 0) > 0 {
            return true
        }
        switch phase?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "rain", "snow", "freezing-rain", "ice-pellets", "mixed":
            return true
        default:
            return false
        }
    }

    private static func isOnset(_ date: Date, now: Date) -> Bool {
        let delta = date.timeIntervalSince(now)
        return delta >= 0 && delta <= windowSeconds
    }

    private static func overlaps(interval: NowcastIntervalSample, now: Date) -> Bool {
        let until = now.addingTimeInterval(windowSeconds)
        let end = interval.end ?? interval.start.addingTimeInterval(15 * 60)
        return interval.start <= until && end > now
    }

    private static func coversNow(_ sample: NowcastWindowSample) -> Bool {
        sample.intervals.contains { interval in
            guard interval.precipitating else { return false }
            let end = interval.end ?? interval.start.addingTimeInterval(15 * 60)
            return interval.start <= sample.now && end > sample.now
        }
    }

    private static func nextOnset(_ sample: NowcastWindowSample) -> Date? {
        var best: Date?
        if let next = sample.nextPrecipTime, isOnset(next, now: sample.now) {
            best = next
        }
        let until = sample.now.addingTimeInterval(windowSeconds)
        for interval in sample.intervals where interval.precipitating {
            guard interval.start >= sample.now, interval.start <= until else { continue }
            if best == nil || interval.start < best! {
                best = interval.start
            }
        }
        return best
    }
}
