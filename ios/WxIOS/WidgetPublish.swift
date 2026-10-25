import Foundation
import WidgetKit

/// Maps shell DTOs into the small cache the widget reads. Weather stays in Go.
enum WidgetPublish {
    static func snapshot(
        place: String,
        units: String,
        payload: WxPayload?,
        nowcast: NowcastPayloadDTO?,
        now: Date
    ) -> WxWidgetSnapshot {
        let resolvedUnits = units == "metric" ? "metric" : "imperial"
        let cast = nowcast?.nowcast ?? payload?.conditions?.nowcast ?? payload?.nowcast
        let conditions = payload?.conditions
        let temperature = WxTemperature.text(
            f: conditions?.temperatureF,
            c: conditions?.temperatureC,
            units: resolvedUnits
        )
        let headline = headlineText(cast)
        let spark = (cast?.intervals ?? []).prefix(24).map { String($0.sparkRune) }.joined()
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: cast?.isActivePrecip ?? false,
            nextPrecipTime: WxPublishISO.date(cast?.nextPrecipTime),
            intervals: intervals(cast)
        )
        let high = daytime(payload)
        let low = nighttime(payload)
        let cue = LiveActivityCue.make(
            sample: sample,
            location: place.isEmpty ? "wx" : place,
            headline: headline,
            phase: cast?.primaryPhase ?? "none",
            sparkline: spark,
            temperatureText: temperature,
            detail: cast?.summary ?? "",
            cessationText: clockText(WxPublishISO.date(cast?.precipEndTime))
        )
        let rawCondition = conditions?.description?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return WxWidgetSnapshot(
            updatedAt: now,
            location: WxWidgetText.clip(place.isEmpty ? "wx" : place),
            units: resolvedUnits,
            temperatureText: temperature,
            condition: WxWidgetText.clip(rawCondition.isEmpty ? "—" : rawCondition),
            highText: WxTemperature.text(f: high?.temperatureF, c: high?.temperatureC, units: resolvedUnits),
            lowText: WxTemperature.text(f: low?.temperatureF, c: low?.temperatureC, units: resolvedUnits),
            precipHeadline: WxWidgetText.clip(headline),
            precipSummary: WxWidgetText.clip(cast?.summary ?? ""),
            sparkline: WxWidgetText.clip(spark, limit: 32),
            hours: hours(payload, units: resolvedUnits),
            days: days(payload, units: resolvedUnits),
            cue: cue
        )
    }

    static func store(_ snapshot: WxWidgetSnapshot, radarBase64: String?, to directory: URL? = nil) throws {
        let png = radarBase64.flatMap { RadarThumb.png(base64: $0) }
        if let directory {
            try WxWidgetCache.write(snapshot, radarPNG: png, to: directory)
            return
        }
        let primary = WxWidgetCache.liveDirectory()
        do {
            try WxWidgetCache.write(snapshot, radarPNG: png, to: primary)
        } catch {
            let caches = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first
                ?? URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true)
            let fallback = WxCacheDirectories.resolve(groupURL: nil, cachesURL: caches)
            if fallback.path == primary.path {
                throw error
            }
            try WxWidgetCache.write(snapshot, radarPNG: png, to: fallback)
        }
        WidgetCenter.shared.reloadAllTimelines()
    }

    private static func headlineText(_ cast: NowcastDTO?) -> String {
        if let headline = cast?.headline?.trimmingCharacters(in: .whitespacesAndNewlines), !headline.isEmpty {
            return headline
        }
        return "—"
    }

    private static func intervals(_ cast: NowcastDTO?) -> [NowcastIntervalSample] {
        guard let rows = cast?.intervals else { return [] }
        return rows.compactMap { row in
            guard let start = WxPublishISO.date(row.startTime) else { return nil }
            return NowcastIntervalSample(
                start: start,
                end: WxPublishISO.date(row.endTime),
                precipitating: LiveActivityRule.isPrecipitating(
                    phase: row.phase,
                    rateInh: row.rateInh,
                    rateMmh: row.rateMmh,
                    accumIn: row.accumIn,
                    accumMm: row.accumMm
                )
            )
        }
    }

    private static func daytime(_ payload: WxPayload?) -> Period? {
        let periods = payload?.forecast?.periods ?? []
        return periods.first { $0.isDaytime == true } ?? periods.first
    }

    private static func nighttime(_ payload: WxPayload?) -> Period? {
        (payload?.forecast?.periods ?? []).first { $0.isDaytime == false }
    }

    private static func hours(_ payload: WxPayload?, units: String) -> [WxWidgetSnapshot.Chip] {
        (payload?.forecast?.hourly ?? []).prefix(6).map { period in
            let temp = IOSFormat.period(period, units: units)
            let pop = percent(period.probabilityOfPrecipitation)
            let value = pop == "—" ? temp : "\(temp) \(pop)"
            return WxWidgetSnapshot.Chip(label: IOSFormat.hourLabel(period), value: value)
        }
    }

    private static func days(_ payload: WxPayload?, units: String) -> [WxWidgetSnapshot.Chip] {
        (payload?.forecast?.periods ?? []).prefix(3).map { period in
            let temp = IOSFormat.period(period, units: units)
            let pop = percent(period.probabilityOfPrecipitation)
            let value = pop == "—" ? temp : "\(temp) \(pop)"
            return WxWidgetSnapshot.Chip(label: period.name, value: value)
        }
    }

    private static func percent(_ value: Double?) -> String {
        guard let value else { return "—" }
        return "\(Int(value.rounded()))%"
    }

    private static func clockText(_ date: Date?) -> String {
        guard let date else { return "" }
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeStyle = .short
        formatter.dateStyle = .none
        return formatter.string(from: date)
    }
}

enum WxPublishISO {
    static func date(_ raw: String?) -> Date? {
        guard let raw else { return nil }
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
