import XCTest

/// Drives wx.app against a stub CLI. Asserts which processes spawn, not pixels.
final class RadarRefreshUITests: XCTestCase {
    private var scratch: URL!
    private var logURL: URL!
    private var configURL: URL!

    override func setUpWithError() throws {
        continueAfterFailure = false
        scratch = URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true)
            .appendingPathComponent("wx-ui-\(UUID().uuidString)", isDirectory: true)
        try FileManager.default.createDirectory(at: scratch, withIntermediateDirectories: true)
        logURL = scratch.appendingPathComponent("stub.log")
        configURL = scratch.appendingPathComponent("config.json")
        FileManager.default.createFile(atPath: logURL.path, contents: Data())
    }

    override func tearDownWithError() throws {
        if let scratch {
            try? FileManager.default.removeItem(at: scratch)
        }
    }

    func testTimerRunsWhileDeskOpenAndStopsAfterClose() throws {
        let app = try launch(radarSeconds: "2", refreshSeconds: "600")
        XCTAssertTrue(app.windows["wx"].waitForExistence(timeout: 5))

        _ = try waitRadarCount(atLeast: 1, timeout: 5)
        Thread.sleep(forTimeInterval: 1)
        let burst = radarCount()

        Thread.sleep(forTimeInterval: 5)
        XCTAssertGreaterThanOrEqual(radarCount(), burst + 1, "2s timer did not fetch while the desk was open")

        app.typeKey("w", modifierFlags: .command)
        Thread.sleep(forTimeInterval: 0.5)
        let closed = radarCount()
        Thread.sleep(forTimeInterval: 6)
        XCTAssertEqual(radarCount(), closed, "radar kept refreshing after the desk closed")
        app.terminate()
    }

    func testLaunchAndLocationOneShotsStillFetch() throws {
        let app = try launch(radarSeconds: "600", refreshSeconds: "600")
        XCTAssertTrue(app.windows["wx"].waitForExistence(timeout: 5))

        _ = try waitRadarCount(atLeast: 1, timeout: 5)
        Thread.sleep(forTimeInterval: 1)
        let burst = radarCount()
        Thread.sleep(forTimeInterval: 5)
        XCTAssertEqual(radarCount(), burst, "timer fetched during the one-shot case")

        let field = locationField(in: app)
        XCTAssertTrue(field.waitForExistence(timeout: 5))
        field.click()
        field.typeText("46802\n")

        let after = try waitRadarCount(atLeast: burst + 1, timeout: 10)
        let added = radarLines().dropFirst(burst)
        XCTAssertTrue(added.contains { $0.contains("--location") && $0.contains("46802") },
                      "location submit did not fetch radar with --location 46802 (count \(after))")

        app.typeKey("w", modifierFlags: .command)
        Thread.sleep(forTimeInterval: 0.5)
        let closed = radarCount()
        Thread.sleep(forTimeInterval: 5)
        XCTAssertEqual(radarCount(), closed, "radar fetched after the desk closed")
        app.terminate()
    }

    func testClosedDeskKeepsMenuBarRefreshAndSkipsDeskFetches() throws {
        let config = #"{"favorites":[{"name":"Fav","value":"FavTown"}]}"#
        try config.write(to: configURL, atomically: true, encoding: .utf8)

        let app = try launch(radarSeconds: "600", refreshSeconds: "8")
        XCTAssertTrue(app.windows["wx"].waitForExistence(timeout: 5))
        try waitUntil(timeout: 8) {
            self.lineCount(containing: "outlook") >= 1
                && self.lineCount(containing: "chase") >= 1
                && self.lineCount(containing: "FavTown") >= 1
                && self.menuBarFetches() >= 1
        }

        let outlook = lineCount(containing: "outlook")
        let chase = lineCount(containing: "chase")
        let favorite = lineCount(containing: "FavTown")
        let menu = menuBarFetches()

        app.typeKey("w", modifierFlags: .command)
        try waitUntil(timeout: 12) { self.menuBarFetches() > menu }

        XCTAssertEqual(lineCount(containing: "outlook"), outlook, "outlook fetched after the desk closed")
        XCTAssertEqual(lineCount(containing: "chase"), chase, "chase fetched after the desk closed")
        XCTAssertEqual(lineCount(containing: "FavTown"), favorite, "favorites grid fetched after the desk closed")
        app.terminate()
    }

    func testDeskWindowUnloadsOnCloseAndReopens() throws {
        let app = try launch(radarSeconds: "600", refreshSeconds: "600")
        XCTAssertTrue(app.windows["wx"].waitForExistence(timeout: 5))

        app.typeKey("w", modifierFlags: .command)
        try waitUntil(timeout: 3) { !app.windows["wx"].exists }

        app.menuBars.menuBarItems["File"].click()
        app.menuItems["Open Desk Window"].click()
        XCTAssertTrue(app.windows["wx"].waitForExistence(timeout: 5))
        app.terminate()
    }

    func testSteadyPipWhileHazardAndNoneWhenClear() throws {
        let alerted = try launch(radarSeconds: "600", refreshSeconds: "600", alert: true)
        XCTAssertTrue(alerted.windows["wx"].waitForExistence(timeout: 5))
        let first = try waitForStatus(alerted, timeout: 8) { $0.contains("🔴") }
        Thread.sleep(forTimeInterval: 2)
        let second = statusTitle(alerted)
        XCTAssertTrue(first.contains("🔴") && second.contains("🔴"), "pip was not steady: \(first) then \(second)")
        XCTAssertFalse(first.contains("⭕") || second.contains("⭕"), "pip blinked off")
        alerted.terminate()

        let clear = try launch(radarSeconds: "600", refreshSeconds: "600", alert: false)
        XCTAssertTrue(clear.windows["wx"].waitForExistence(timeout: 5))
        let quiet = try waitForStatus(clear, timeout: 8) { !$0.contains("🔴") && !$0.contains("🟠") && ($0.contains("--") || $0.contains("°")) }
        Thread.sleep(forTimeInterval: 2)
        let quietLater = statusTitle(clear)
        XCTAssertFalse(quiet.contains("🔴") || quiet.contains("🟠") || quietLater.contains("🔴") || quietLater.contains("🟠"),
                       "clear status showed a pip: \(quiet) then \(quietLater)")
        clear.terminate()
    }

    func testHourlyStripOnDeskAndPopover() throws {
        let config = #"{"favorites":[{"name":"Fav","value":"FavTown"}]}"#
        try config.write(to: configURL, atomically: true, encoding: .utf8)

        let app = try launch(radarSeconds: "600", refreshSeconds: "600")
        XCTAssertTrue(app.windows["wx"].waitForExistence(timeout: 5))

        try waitUntil(timeout: 8) {
            self.stubLines().contains { line in
                line.contains("--json") && line.contains("--forecast") && line.contains("--alerts")
                    && line.contains("--hourly") && line.contains("--hours") && !line.contains("FavTown")
            } && self.stubLines().contains { $0.contains("FavTown") }
        }
        XCTAssertFalse(stubLines().contains { $0.contains("FavTown") && $0.contains("--hourly") },
                       "favorites grid asked for hourly")

        XCTAssertTrue(app.staticTexts["Tonight"].waitForExistence(timeout: 5))
        XCTAssertTrue(waitForLabel(app, containing: "72°", timeout: 5), "desk did not show 72°")

        app.typeKey("w", modifierFlags: .command)
        try waitUntil(timeout: 3) { !app.windows["wx"].exists }

        app.statusItems.firstMatch.click()
        XCTAssertTrue(app.staticTexts["Tonight"].waitForExistence(timeout: 5))
        XCTAssertTrue(waitForLabel(app, containing: "72°", timeout: 5), "popover did not show 72°")
        app.terminate()
    }

    private func launch(radarSeconds: String, refreshSeconds: String, alert: Bool = false) throws -> XCUIApplication {
        let stub = try stubBinary()
        let app = XCUIApplication()
        app.launchEnvironment = [
            "WX_UI_TEST": "1",
            "WX_BINARY": stub,
            "WX_STUB_LOG": logURL.path,
            "WX_CONFIG": configURL.path,
            "WX_RADAR_REFRESH_SECONDS": radarSeconds,
            "WX_REFRESH_SECONDS": refreshSeconds,
        ]
        if alert {
            app.launchEnvironment["WX_STUB_ALERT"] = "warning"
        }
        app.launch()
        return app
    }

    private func stubBinary() throws -> String {
        let bundle = Bundle(for: RadarRefreshUITests.self)
        let url = bundle.url(forResource: "wx-stub", withExtension: nil)
            ?? bundle.url(forResource: "wx-stub", withExtension: nil, subdirectory: "Fixtures")
        guard let url else {
            XCTFail("wx-stub is not in the test bundle")
            return ""
        }
        return url.path
    }

    private func radarLines() -> [String] {
        guard let text = try? String(contentsOf: logURL, encoding: .utf8) else { return [] }
        return text.split(separator: "\n", omittingEmptySubsequences: true).map(String.init).filter { $0.contains("radar") }
    }

    private func radarCount() -> Int { radarLines().count }

    private func stubLines() -> [String] {
        guard let text = try? String(contentsOf: logURL, encoding: .utf8) else { return [] }
        return text.split(separator: "\n", omittingEmptySubsequences: true).map(String.init)
    }

    private func lineCount(containing needle: String) -> Int {
        stubLines().filter { $0.contains(needle) }.count
    }

    /// Menu-bar fetch is `--json --forecast --alerts` with no favorite location.
    private func menuBarFetches() -> Int {
        stubLines().filter {
            $0.contains("--json") && $0.contains("--forecast") && $0.contains("--alerts") && !$0.contains("FavTown")
        }.count
    }

    private func waitUntil(timeout: TimeInterval, _ condition: () -> Bool) throws {
        let deadline = Date().addingTimeInterval(timeout)
        while !condition() && Date() < deadline {
            Thread.sleep(forTimeInterval: 0.2)
        }
        XCTAssertTrue(condition(), "timed out. stub log:\n\(stubLines().joined(separator: "\n"))")
    }

    private func waitRadarCount(atLeast minimum: Int, timeout: TimeInterval) throws -> Int {
        let deadline = Date().addingTimeInterval(timeout)
        var count = radarCount()
        while count < minimum && Date() < deadline {
            Thread.sleep(forTimeInterval: 0.2)
            count = radarCount()
        }
        XCTAssertGreaterThanOrEqual(count, minimum, "stub log never reached \(minimum) radar calls:\n\(radarLines().joined(separator: "\n"))")
        return count
    }

    private func statusTitle(_ app: XCUIApplication) -> String {
        let item = app.statusItems.firstMatch
        return [item.title, item.label, item.value as? String ?? ""].joined(separator: " ")
    }

    private func waitForStatus(_ app: XCUIApplication, timeout: TimeInterval, _ condition: (String) -> Bool) throws -> String {
        let deadline = Date().addingTimeInterval(timeout)
        var title = statusTitle(app)
        while !condition(title) && Date() < deadline {
            Thread.sleep(forTimeInterval: 0.2)
            title = statusTitle(app)
        }
        XCTAssertTrue(condition(title), "status item never matched. title=\(title)")
        return title
    }

    private func waitForLabel(_ app: XCUIApplication, containing text: String, timeout: TimeInterval) -> Bool {
        let el = app.descendants(matching: .any).matching(NSPredicate(format: "label CONTAINS %@", text)).firstMatch
        return el.waitForExistence(timeout: timeout)
    }

    private func locationField(in app: XCUIApplication) -> XCUIElement {
        let fields = app.textFields.matching(identifier: "wx.location")
        for field in fields.allElementsBoundByIndex where field.isHittable {
            return field
        }
        return fields.firstMatch
    }
}
