import XCTest

/// Power behavior against bundled fixtures. The clock is frozen, so a
/// foreground return inside five minutes must not fetch again. No live NWS.
final class WxIOSPowerUITests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testLoopPausesOffRadarTab() throws {
        let app = launch()
        XCTAssertTrue(text(app, "Fixture Partly Cloudy").waitForExistence(timeout: 15), dump(app))
        _ = waitUntil(app, "power.frameCount") { (Int($0) ?? 0) >= 2 }

        XCTAssertEqual(read(app, "power.loopState"), "paused", dump(app))
        let idle = read(app, "power.loopTicks")
        Thread.sleep(forTimeInterval: 1.2)
        XCTAssertEqual(read(app, "power.loopTicks"), idle, "loop advanced off the radar tab. \(dump(app))")

        open(app, "Radar")
        _ = waitUntil(app, "power.loopTicks") { ticks in
            (Int(ticks) ?? 0) >= (Int(idle) ?? 0) + 3
        }
        XCTAssertEqual(read(app, "power.loopState"), "playing", dump(app))

        open(app, "Surface")
        XCTAssertEqual(read(app, "power.loopState"), "paused", dump(app))
        let held = read(app, "power.loopTicks")
        Thread.sleep(forTimeInterval: 1.2)
        XCTAssertEqual(read(app, "power.loopTicks"), held, "loop kept playing after leaving Radar. \(dump(app))")
        XCTAssertEqual(read(app, "power.ticksAtPause"), held, dump(app))

        open(app, "Radar")
        _ = waitUntil(app, "power.loopTicks") { ticks in
            (Int(ticks) ?? 0) > (Int(held) ?? 0)
        }
        XCTAssertEqual(read(app, "power.loopState"), "playing", dump(app))
    }

    func testLoopPausesWhenBackgrounded() throws {
        let app = launch()
        XCTAssertTrue(text(app, "Fixture Partly Cloudy").waitForExistence(timeout: 15), dump(app))
        _ = waitUntil(app, "power.frameCount") { (Int($0) ?? 0) >= 2 }
        open(app, "Radar")
        _ = waitUntil(app, "power.loopState") { $0 == "playing" }
        _ = waitUntil(app, "power.loopTicks") { (Int($0) ?? 0) >= 2 }
        let pauses = Int(read(app, "power.pauseCount")) ?? 0

        leaveAndReturn(app, seconds: 1.2)

        _ = waitUntil(app, "power.pauseCount") { (Int($0) ?? 0) > pauses }
        _ = waitUntil(app, "power.loopState") { $0 == "playing" }
        XCTAssertEqual(read(app, "power.ticksWhilePaused"), "0", "loop advanced while inactive. \(dump(app))")
    }

    func testForegroundSkipsRefreshInsideFiveMinutes() throws {
        let app = launch()
        XCTAssertTrue(text(app, "Fixture Partly Cloudy").waitForExistence(timeout: 15), dump(app))
        _ = waitUntil(app, "power.fetchCount") { $0 == "1" }

        leaveAndReturn(app, seconds: 0.8)
        Thread.sleep(forTimeInterval: 0.6)

        XCTAssertEqual(read(app, "power.fetchCount"), "1", "refetched inside 5 minutes. \(dump(app))")
        XCTAssertTrue(text(app, "Fixture Partly Cloudy").exists, dump(app))
    }

    func testForegroundRefreshesWhenClockIsOlderThanFiveMinutes() throws {
        let app = launch()
        XCTAssertTrue(text(app, "Fixture Partly Cloudy").waitForExistence(timeout: 15), dump(app))
        _ = waitUntil(app, "power.fetchCount") { $0 == "1" }

        let advance = app.buttons["power.clock.advance"]
        XCTAssertTrue(advance.waitForExistence(timeout: 5), dump(app))
        advance.tap()
        Thread.sleep(forTimeInterval: 0.4)
        XCTAssertEqual(read(app, "power.fetchCount"), "1", "advancing the clock refreshed while foregrounded. \(dump(app))")

        leaveAndReturn(app, seconds: 0.6)
        _ = waitUntil(app, "power.fetchCount") { $0 == "2" }
        XCTAssertTrue(text(app, "Fixture Partly Cloudy").waitForExistence(timeout: 5), dump(app))
    }

    private func launch() -> XCUIApplication {
        let app = XCUIApplication()
        app.launchEnvironment = [
            "WX_UI_TEST": "1",
            "WX_FIXTURE": "1",
            "WX_SKIP_LOCATION": "1",
            "WX_DEFAULT_LOCATION": "New Haven, IN",
            "WX_CLOCK": "frozen",
            "WX_NOW": "2026-10-25T18:00:00Z",
        ]
        app.launch()
        return app
    }

    private func leaveAndReturn(_ app: XCUIApplication, seconds: TimeInterval) {
        XCUIDevice.shared.press(.home)
        Thread.sleep(forTimeInterval: seconds)
        app.activate()
        XCTAssertTrue(app.wait(for: .runningForeground, timeout: 10), "app did not return. \(dump(app))")
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

    private func open(_ app: XCUIApplication, _ name: String, file: StaticString = #filePath, line: UInt = #line) {
        let button = app.buttons["tab.\(name)"]
        XCTAssertTrue(button.waitForExistence(timeout: 5), "tab \(name) missing. \(dump(app))", file: file, line: line)
        button.tap()
    }

    private func text(_ app: XCUIApplication, _ label: String) -> XCUIElement {
        app.staticTexts[label]
    }

    private func dump(_ app: XCUIApplication) -> String {
        let labels = app.staticTexts.allElementsBoundByIndex.prefix(40).map(\.label).joined(separator: " | ")
        return "static texts: \(labels)"
    }
}
