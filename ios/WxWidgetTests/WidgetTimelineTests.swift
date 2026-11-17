import CoreGraphics
import ImageIO
import UniformTypeIdentifiers
import XCTest

final class WidgetTimelineTests: XCTestCase {
    func testResolvePrefersAppGroup() {
        let group = URL(fileURLWithPath: "/tmp/wx-group", isDirectory: true)
        let caches = URL(fileURLWithPath: "/tmp/wx-caches", isDirectory: true)
        XCTAssertEqual(WxCacheDirectories.resolve(groupURL: group, cachesURL: caches), group)
    }

    func testResolveFallsBackToCaches() {
        let caches = URL(fileURLWithPath: "/tmp/wx-caches", isDirectory: true)
        let url = WxCacheDirectories.resolve(groupURL: nil, cachesURL: caches)
        XCTAssertEqual(url.lastPathComponent, WxAppGroup.fallbackFolder)
        XCTAssertEqual(url.deletingLastPathComponent().path, caches.path)
    }

    func testMissingCacheIsAPlaceholderTimeline() throws {
        let directory = try makeDirectory()
        defer { try? FileManager.default.removeItem(at: directory) }
        let now = Date(timeIntervalSince1970: 1_761_422_400)
        let plan = WxTimelinePlanBuilder.plan(reading: directory, now: now)
        XCTAssertEqual(plan.snapshot.condition, "No cache")
        XCTAssertFalse(plan.showsRadar)
        XCTAssertEqual(plan.radarByteCount, 0)
        XCTAssertEqual(plan.reloadAfter, now.addingTimeInterval(15 * 60))

        let timeline = WxConditionsProvider(directory: directory).makeTimeline(now: now)
        XCTAssertEqual(timeline.entries.count, 1)
        XCTAssertEqual(timeline.entries[0].snapshot.temperatureText, "—")
        XCTAssertNil(timeline.entries[0].radar)
        XCTAssertEqual(timeline.policy, .after(now.addingTimeInterval(15 * 60)))
    }

    func testProviderReadsFixtureCache() throws {
        let directory = try makeDirectory()
        defer { try? FileManager.default.removeItem(at: directory) }
        let now = Date(timeIntervalSince1970: 1_761_422_400)
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: false,
            nextPrecipTime: now.addingTimeInterval(20 * 60),
            intervals: []
        )
        let snapshot = WxWidgetSnapshot(
            updatedAt: now,
            location: "New Haven, IN",
            units: "imperial",
            temperatureText: "72°",
            condition: "Fixture Partly Cloudy",
            highText: "72°",
            lowText: "51°",
            precipHeadline: "Rain starting in ~20m",
            precipSummary: "Light rain in the fixture window.",
            sparkline: " ▂▄▆",
            hours: [
                WxWidgetSnapshot.Chip(label: "15:00", value: "70° 5%"),
                WxWidgetSnapshot.Chip(label: "16:00", value: "69° 40%"),
            ],
            days: [
                WxWidgetSnapshot.Chip(label: "Fixture Tuesday", value: "72°"),
            ],
            cue: LiveActivityCue.make(
                sample: sample,
                location: "New Haven, IN",
                headline: "Rain starting in ~20m",
                phase: "rain",
                sparkline: " ▂▄▆",
                temperatureText: "72°",
                detail: "Light rain in the fixture window.",
                cessationText: ""
            )
        )
        let thumb = try XCTUnwrap(RadarThumb.png(base64: fixtureRadarBase64))
        try WxWidgetCache.write(snapshot, radarPNG: thumb, to: directory)

        let provider = WxConditionsProvider(directory: directory)
        let timeline = provider.makeTimeline(now: now)
        let entry = try XCTUnwrap(timeline.entries.first)
        XCTAssertEqual(entry.snapshot.temperatureText, "72°")
        XCTAssertEqual(entry.snapshot.precipHeadline, "Rain starting in ~20m")
        XCTAssertEqual(entry.snapshot.highText, "72°")
        XCTAssertEqual(entry.snapshot.lowText, "51°")
        XCTAssertEqual(entry.snapshot.hours.map(\.label), ["15:00", "16:00"])
        XCTAssertTrue(entry.snapshot.cue.shouldRun)
        XCTAssertEqual(entry.snapshot.cue.minutesLabel, "20m")
        let radar = try XCTUnwrap(entry.radar)
        XCTAssertLessThanOrEqual(radar.count, WxWidgetLimits.radarMaxBytes)
        let edge = try XCTUnwrap(RadarThumb.pixelEdge(radar))
        XCTAssertLessThanOrEqual(edge, WxWidgetLimits.radarMaxPixel)

        let plan = WxTimelinePlanBuilder.plan(reading: directory, now: now)
        XCTAssertTrue(plan.showsRadar)
        XCTAssertEqual(plan.radarByteCount, radar.count)
        XCTAssertEqual(plan.snapshot.condition, "Fixture Partly Cloudy")
    }

    func testOversizedRadarIsNotShown() throws {
        let directory = try makeDirectory()
        defer { try? FileManager.default.removeItem(at: directory) }
        let now = Date(timeIntervalSince1970: 1_761_422_400)
        try WxWidgetCache.write(WxWidgetSnapshot.placeholder, radarPNG: nil, to: directory)
        let huge = Data(repeating: 0xAB, count: WxWidgetLimits.radarMaxBytes + 1)
        let radarURL = directory.appendingPathComponent(WxAppGroup.radarFile)
        try huge.write(to: radarURL)

        let plan = WxTimelinePlanBuilder.plan(reading: directory, now: now)
        XCTAssertFalse(plan.showsRadar)
        XCTAssertEqual(plan.radarByteCount, 0)
        XCTAssertNil(WxConditionsProvider(directory: directory).entry(now: now).radar)
    }

    func testDownsampleKeepsASingleSmallFrame() throws {
        let wide = try solidPNG(width: 200, height: 100)
        let edge = try XCTUnwrap(RadarThumb.pixelEdge(wide))
        XCTAssertEqual(edge, 200)
        XCTAssertNil(RadarThumb.accepted(wide))
        let thumb = try XCTUnwrap(RadarThumb.png(data: wide))
        XCTAssertLessThan(thumb.count, WxWidgetLimits.radarMaxBytes)
        let thumbEdge = try XCTUnwrap(RadarThumb.pixelEdge(thumb))
        XCTAssertLessThanOrEqual(thumbEdge, WxWidgetLimits.radarMaxPixel)
        XCTAssertGreaterThan(thumbEdge, 0)
    }

    func testWriteReplacesAStaleRadarFrame() throws {
        let directory = try makeDirectory()
        defer { try? FileManager.default.removeItem(at: directory) }
        let thumb = try XCTUnwrap(RadarThumb.png(base64: fixtureRadarBase64))
        try WxWidgetCache.write(WxWidgetSnapshot.placeholder, radarPNG: thumb, to: directory)
        XCTAssertTrue(FileManager.default.fileExists(atPath: directory.appendingPathComponent(WxAppGroup.radarFile).path))
        try WxWidgetCache.write(WxWidgetSnapshot.placeholder, radarPNG: nil, to: directory)
        XCTAssertFalse(FileManager.default.fileExists(atPath: directory.appendingPathComponent(WxAppGroup.radarFile).path))
    }

    func testNightSnapshotRoundTripsSymbolAndLegacyCacheDecodesCleanly() throws {
        let directory = try makeDirectory()
        defer { try? FileManager.default.removeItem(at: directory) }
        let now = Date(timeIntervalSince1970: 1_761_422_400)

        let nightSnap = WxWidgetSnapshot(
            updatedAt: now,
            location: "New Haven, IN",
            units: "imperial",
            temperatureText: "36°",
            condition: "Mostly Clear",
            highText: "54°",
            lowText: "32°",
            precipHeadline: "Clear",
            precipSummary: "",
            sparkline: "",
            hours: [],
            days: [],
            cue: .idle,
            symbolName: "cloud.moon.fill",
            isNight: true
        )
        try WxWidgetCache.write(nightSnap, radarPNG: nil, to: directory)
        let loaded = WxTimelinePlanBuilder.plan(reading: directory, now: now).snapshot
        XCTAssertEqual(loaded.symbolName, "cloud.moon.fill")
        XCTAssertEqual(loaded.isNight, true)
        XCTAssertEqual(loaded.condition, "Mostly Clear")

        // Simulate legacy snapshot JSON written before symbolName/isNight existed
        let legacyJSON = """
        {
          "updatedAt": "2026-10-25T20:00:00Z",
          "location": "New Haven, IN",
          "units": "imperial",
          "temperatureText": "36°",
          "condition": "Clear",
          "highText": "54°",
          "lowText": "32°",
          "precipHeadline": "Clear",
          "precipSummary": "",
          "sparkline": "",
          "hours": [],
          "days": [],
          "cue": {
            "shouldRun": false,
            "location": "New Haven, IN",
            "headline": "—",
            "phase": "none",
            "minutesLabel": "—",
            "sparkline": "",
            "temperatureText": "36°",
            "detail": "",
            "cessationText": ""
          }
        }
        """
        try Data(legacyJSON.utf8).write(to: directory.appendingPathComponent(WxAppGroup.snapshotFile))
        let legacyLoaded = WxTimelinePlanBuilder.plan(reading: directory, now: now).snapshot
        XCTAssertEqual(legacyLoaded.temperatureText, "36°")
        XCTAssertNil(legacyLoaded.symbolName)
        XCTAssertNil(legacyLoaded.isNight)
    }

    func testWidgetPublishNighttimeASOSSkipsTomorrowHighTrimsExpiredHoursAndHidesDrySparkBars() throws {
        let json = """
        {
          "conditions": {
            "station": "KFWA",
            "location": "New Haven, IN",
            "temperature_f": 34,
            "temperature_c": 1.1,
            "astronomy": {
              "solar_elevation_deg": -66.9,
              "current_period": "Night"
            }
          },
          "forecast": {
            "periods": [
              {
                "name": "Tonight",
                "is_daytime": false,
                "temperature_f": 32,
                "temperature_c": 0,
                "short_description": "Partly Cloudy"
              },
              {
                "name": "Tuesday",
                "is_daytime": true,
                "temperature_f": 54,
                "temperature_c": 12.2,
                "short_description": "Sunny"
              }
            ],
            "hourly": [
              {
                "name": "11 PM",
                "start_time": "2026-11-17T04:00:00Z",
                "is_daytime": false,
                "temperature_f": 36,
                "temperature_c": 2.2,
                "short_description": "Mostly Cloudy"
              },
              {
                "name": "12 AM",
                "start_time": "2026-11-17T05:00:00Z",
                "is_daytime": false,
                "temperature_f": 34,
                "temperature_c": 1.1,
                "short_description": "Mostly Clear"
              }
            ]
          },
          "alerts": []
        }
        """
        let nowcastJSON = """
        {
          "location": "New Haven, IN",
          "nowcast": {
            "headline": "Clear · No precipitation expected next 8 hours",
            "summary": "Conditions remain dry and clear.",
            "is_active_precip": false,
            "primary_phase": "none",
            "intervals": [
              { "start_time": "2026-11-17T05:15:00Z", "end_time": "2026-11-17T05:30:00Z", "rate_inh": 0, "phase": "none" },
              { "start_time": "2026-11-17T05:30:00Z", "end_time": "2026-11-17T05:45:00Z", "rate_inh": 0, "phase": "none" }
            ]
          }
        }
        """
        let payload = try JSONDecoder().decode(WxPayload.self, from: Data(json.utf8))
        let nowcast = try JSONDecoder().decode(NowcastPayloadDTO.self, from: Data(nowcastJSON.utf8))
        let now = try XCTUnwrap(ISO8601DateFormatter().date(from: "2026-11-17T05:22:00Z"))

        let snap = WidgetPublish.snapshot(
            place: "46774",
            units: "imperial",
            payload: payload,
            nowcast: nowcast,
            now: now
        )

        XCTAssertEqual(snap.location, "New Haven, IN")
        XCTAssertEqual(snap.isNight, true)
        XCTAssertEqual(snap.symbolName, "cloud.moon.fill")
        XCTAssertEqual(snap.condition, "Mostly Clear")
        XCTAssertEqual(snap.highText, "—", "Nighttime Tonight lead period must not report tomorrow's daytime high")
        XCTAssertEqual(snap.lowText, "32°")
        XCTAssertEqual(snap.hours.count, 1, "Expired 04:00Z–05:00Z hour must be trimmed at 05:22Z")

        // Dry whitespace-only sparkline must NOT show 16 empty spark bars in WxConditionsView
        let view = WxConditionsView(entry: WxWidgetEntry(date: now, snapshot: snap, radar: nil))
        XCTAssertFalse(view.hasSparkBars)
        XCTAssertEqual(view.rectangularHeaderText, "34°  Low 32°")
    }

    private func makeDirectory() throws -> URL {
        let url = FileManager.default.temporaryDirectory
            .appendingPathComponent("wx-widget-\(UUID().uuidString)", isDirectory: true)
        try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
        return url
    }

    private func solidPNG(width: Int, height: Int) throws -> Data {
        var bytes = [UInt8](repeating: 0, count: width * height * 4)
        for index in stride(from: 0, to: bytes.count, by: 4) {
            bytes[index] = 30
            bytes[index + 1] = 90
            bytes[index + 2] = 160
            bytes[index + 3] = 255
        }
        let encoded: Data? = bytes.withUnsafeMutableBytes { raw in
            guard let ctx = CGContext(
                data: raw.baseAddress,
                width: width,
                height: height,
                bitsPerComponent: 8,
                bytesPerRow: width * 4,
                space: CGColorSpaceCreateDeviceRGB(),
                bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
            ), let cgImage = ctx.makeImage() else {
                return nil
            }
            let out = NSMutableData()
            guard let dest = CGImageDestinationCreateWithData(out, UTType.png.identifier as CFString, 1, nil) else {
                return nil
            }
            CGImageDestinationAddImage(dest, cgImage, nil)
            guard CGImageDestinationFinalize(dest) else { return nil }
            return out as Data
        }
        return try XCTUnwrap(encoded)
    }
}

/// 8×8 fixture frame from ios/WxIOS/Fixtures/radar.json. Not a live scan.
private let fixtureRadarBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAgAAAAICAIAAABLbSncAAAAEUlEQVR4nGN4tkUOK2IYWhIALU1uAZxzUEsAAAAASUVORK5CYII="
