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

    private func launch(radarSeconds: String, refreshSeconds: String) throws -> XCUIApplication {
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

    private func locationField(in app: XCUIApplication) -> XCUIElement {
        let fields = app.textFields.matching(identifier: "wx.location")
        for field in fields.allElementsBoundByIndex where field.isHittable {
            return field
        }
        return fields.firstMatch
    }
}
