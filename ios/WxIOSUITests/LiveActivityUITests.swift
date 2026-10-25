import XCTest

/// Starts and ends a local Live Activity on the simulator. Fixtures only.
/// Skips when ActivityKit is turned off or the simulator will not start one.
final class WxIOSLiveActivityUITests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testStartsAndEndsWhenNowcastOnsetEntersAndLeavesTheWindow() throws {
        let app = XCUIApplication()
        app.launchEnvironment = [
            "WX_UI_TEST": "1",
            "WX_FIXTURE": "1",
            "WX_SKIP_LOCATION": "1",
            "WX_DEFAULT_LOCATION": "New Haven, IN",
            "WX_LIVE_ACTIVITY_TEST": "1",
            "WX_NOW": "2026-10-25T18:00:00Z",
        ]
        app.launch()

        XCTAssertTrue(app.staticTexts["Fixture Partly Cloudy"].waitForExistence(timeout: 15), dump(app))
        let initial = waitUntil(app, "live.status") { $0 != "pending" && !$0.isEmpty }
        try skipIfUnavailable(initial)

        XCTAssertNotEqual(initial, "active", "dry fixture started a Live Activity. \(dump(app))")

        let onset = app.buttons["live.inject.onset"]
        XCTAssertTrue(onset.waitForExistence(timeout: 5), dump(app))
        onset.tap()
        let started = waitUntil(app, "live.status") { $0 == "active" || $0 == "unavailable" || $0.hasPrefix("failed") }
        try skipIfUnavailable(started)
        XCTAssertEqual(started, "active", dump(app))
        XCTAssertNotEqual(read(app, "live.id"), "none", dump(app))

        app.buttons["live.inject.dry"].tap()
        let ended = waitUntil(app, "live.status") { $0 == "ended" || $0 == "inactive" || $0 == "unavailable" || $0.hasPrefix("failed") }
        try skipIfUnavailable(ended)
        XCTAssertTrue(ended == "ended" || ended == "inactive", dump(app))
        XCTAssertEqual(read(app, "live.id"), "none", dump(app))
    }

    private func skipIfUnavailable(_ status: String) throws {
        if status == "unavailable" || status.hasPrefix("unavailable") {
            throw XCTSkip("ActivityKit unavailable: \(status)")
        }
    }

    private func read(_ app: XCUIApplication, _ id: String, file: StaticString = #filePath, line: UInt = #line) -> String {
        let element = app.staticTexts[id]
        XCTAssertTrue(element.waitForExistence(timeout: 5), "missing \(id). \(dump(app))", file: file, line: line)
        return element.label
    }

    private func waitUntil(
        _ app: XCUIApplication,
        _ id: String,
        timeout: TimeInterval = 15,
        where predicate: (String) -> Bool,
        file: StaticString = #filePath,
        line: UInt = #line
    ) -> String {
        let deadline = Date().addingTimeInterval(timeout)
        var last = ""
        while Date() < deadline {
            let element = app.staticTexts[id]
            if element.exists {
                last = element.label
                if predicate(last) {
                    return last
                }
            }
            RunLoop.current.run(until: Date().addingTimeInterval(0.05))
        }
        XCTFail("timed out waiting for \(id), last \(last). \(dump(app))", file: file, line: line)
        return last
    }

    private func dump(_ app: XCUIApplication) -> String {
        let labels = app.staticTexts.allElementsBoundByIndex.prefix(40).map(\.label).joined(separator: " | ")
        return "static texts: \(labels)"
    }
}
