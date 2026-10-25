import XCTest

final class LiveActivityRuleTests: XCTestCase {
    private let now = Date(timeIntervalSince1970: 1_761_422_400)

    func testOnsetInsideSixtyMinutesStarts() {
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: false,
            nextPrecipTime: now.addingTimeInterval(20 * 60),
            intervals: []
        )
        XCTAssertTrue(LiveActivityRule.showsPrecipitation(in: sample))
        XCTAssertEqual(LiveActivityRule.minutesLabel(in: sample), "20m")
    }

    func testOnsetAtSixtyMinutesStarts() {
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: false,
            nextPrecipTime: now.addingTimeInterval(LiveActivityRule.windowSeconds),
            intervals: []
        )
        XCTAssertTrue(LiveActivityRule.showsPrecipitation(in: sample))
        XCTAssertEqual(LiveActivityRule.minutesLabel(in: sample), "60m")
    }

    func testOnsetPastSixtyMinutesDoesNotStart() {
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: false,
            nextPrecipTime: now.addingTimeInterval(LiveActivityRule.windowSeconds + 1),
            intervals: []
        )
        XCTAssertFalse(LiveActivityRule.showsPrecipitation(in: sample))
        XCTAssertEqual(LiveActivityRule.minutesLabel(in: sample), "—")
    }

    func testActivePrecipitationStaysInTheWindow() {
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: true,
            nextPrecipTime: now.addingTimeInterval(3 * 60 * 60),
            intervals: []
        )
        XCTAssertTrue(LiveActivityRule.showsPrecipitation(in: sample))
        XCTAssertEqual(LiveActivityRule.minutesLabel(in: sample), "now")
    }

    func testPrecipitatingIntervalInsideTheWindowStarts() {
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: false,
            nextPrecipTime: nil,
            intervals: [
                NowcastIntervalSample(
                    start: now.addingTimeInterval(45 * 60),
                    end: now.addingTimeInterval(60 * 60),
                    precipitating: true
                ),
            ]
        )
        XCTAssertTrue(LiveActivityRule.showsPrecipitation(in: sample))
        XCTAssertEqual(LiveActivityRule.minutesLabel(in: sample), "45m")
    }

    func testPrecipitatingIntervalOutsideTheWindowEnds() {
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: false,
            nextPrecipTime: nil,
            intervals: [
                NowcastIntervalSample(
                    start: now.addingTimeInterval(70 * 60),
                    end: now.addingTimeInterval(85 * 60),
                    precipitating: true
                ),
            ]
        )
        XCTAssertFalse(LiveActivityRule.showsPrecipitation(in: sample))
    }

    func testIntervalCoveringNowIsNow() {
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: false,
            nextPrecipTime: nil,
            intervals: [
                NowcastIntervalSample(
                    start: now.addingTimeInterval(-5 * 60),
                    end: now.addingTimeInterval(10 * 60),
                    precipitating: true
                ),
            ]
        )
        XCTAssertTrue(LiveActivityRule.showsPrecipitation(in: sample))
        XCTAssertEqual(LiveActivityRule.minutesLabel(in: sample), "now")
    }

    func testEndedIntervalDoesNotCount() {
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: false,
            nextPrecipTime: now.addingTimeInterval(-10 * 60),
            intervals: [
                NowcastIntervalSample(
                    start: now.addingTimeInterval(-30 * 60),
                    end: now.addingTimeInterval(-60),
                    precipitating: true
                ),
            ]
        )
        XCTAssertFalse(LiveActivityRule.showsPrecipitation(in: sample))
    }

    func testDryWindowDoesNotRun() {
        let sample = NowcastWindowSample(now: now, isActivePrecip: false, nextPrecipTime: nil, intervals: [])
        XCTAssertFalse(LiveActivityRule.showsPrecipitation(in: sample))
        let cue = LiveActivityCue.make(
            sample: sample,
            location: "New Haven, IN",
            headline: "Dry next 6h",
            phase: "none",
            sparkline: "",
            temperatureText: "72°",
            detail: "No precipitation in the fixture window.",
            cessationText: ""
        )
        XCTAssertFalse(cue.shouldRun)
        XCTAssertEqual(cue.minutesLabel, "—")
    }

    func testKnownPhasesAndRates() {
        XCTAssertTrue(LiveActivityRule.isPrecipitating(phase: "rain", rateInh: 0, rateMmh: 0, accumIn: 0, accumMm: 0))
        XCTAssertTrue(LiveActivityRule.isPrecipitating(phase: " Snow ", rateInh: nil, rateMmh: nil, accumIn: nil, accumMm: nil))
        XCTAssertTrue(LiveActivityRule.isPrecipitating(phase: "none", rateInh: 0.02, rateMmh: 0, accumIn: 0, accumMm: 0))
        XCTAssertFalse(LiveActivityRule.isPrecipitating(phase: "none", rateInh: 0, rateMmh: 0, accumIn: 0, accumMm: 0))
        XCTAssertFalse(LiveActivityRule.isPrecipitating(phase: nil, rateInh: nil, rateMmh: nil, accumIn: nil, accumMm: nil))
    }

    func testCueUsesTheRuleForOnset() {
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: false,
            nextPrecipTime: now.addingTimeInterval(15 * 60),
            intervals: []
        )
        let cue = LiveActivityCue.make(
            sample: sample,
            location: "New Haven, IN",
            headline: "Rain starting in ~15m",
            phase: "rain",
            sparkline: " ▂▄",
            temperatureText: "72°",
            detail: "Light rain",
            cessationText: ""
        )
        XCTAssertTrue(cue.shouldRun)
        XCTAssertEqual(cue.minutesLabel, "15m")
        XCTAssertEqual(cue.headline, "Rain starting in ~15m")
    }
}
