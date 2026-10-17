import XCTest

/// Pinned severity banner: above the fold on every desk tab, including radar.
final class AlertBannerUITests: XCTestCase {
    private var scratch: URL!
    private var logURL: URL!
    private var configURL: URL!
    private var app: XCUIApplication?

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
        app?.terminate()
        app = nil
        if let scratch {
            try? FileManager.default.removeItem(at: scratch)
        }
    }

    func testWarningBannerIsPinnedHittableAndOpensDetails() throws {
        let app = try launch(alert: "warning")
        let window = app.windows["wx"]
        XCTAssertTrue(window.waitForExistence(timeout: 8))

        let banner = element(app, "alert-banner")
        XCTAssertTrue(banner.waitForExistence(timeout: 10), "alert-banner was not on screen without scrolling. \(debugLog())")
        assertPinnedInTopBand(banner, of: window)
        XCTAssertTrue(element(app, "alert-banner-warning").waitForExistence(timeout: 2), "alert-banner-warning missing")

        let details = element(app, "alert-banner-details")
        XCTAssertFalse(details.exists)
        banner.click()
        XCTAssertTrue(details.waitForExistence(timeout: 5), "clicking the banner did not show alert-banner-details")
    }

    func testWatchBannerUsesWatchIdentifier() throws {
        let app = try launch(alert: "watch")
        XCTAssertTrue(app.windows["wx"].waitForExistence(timeout: 8))
        XCTAssertTrue(element(app, "alert-banner").waitForExistence(timeout: 10))
        XCTAssertTrue(element(app, "alert-banner-watch").waitForExistence(timeout: 2))
        XCTAssertFalse(element(app, "alert-banner-warning").exists)
    }

    func testNoBannerWhenNoAlert() throws {
        let app = try launch(alert: nil)
        XCTAssertTrue(app.windows["wx"].waitForExistence(timeout: 8))
        XCTAssertTrue(app.staticTexts["Tonight"].waitForExistence(timeout: 8), "forecast never loaded. \(debugLog())")
        XCTAssertFalse(element(app, "alert-banner").exists)
    }

    func testBannerVisibleOnRadarTab() throws {
        let app = try launch(alert: "warning", deskTab: "radar")
        let window = app.windows["wx"]
        XCTAssertTrue(window.waitForExistence(timeout: 8))
        let banner = element(app, "alert-banner")
        XCTAssertTrue(banner.waitForExistence(timeout: 10), "banner missing on the radar tab. \(debugLog())")
        assertPinnedInTopBand(banner, of: window)
        XCTAssertTrue(element(app, "alert-banner-warning").exists)
    }

    private func assertPinnedInTopBand(_ banner: XCUIElement, of window: XCUIElement, file: StaticString = #filePath, line: UInt = #line) {
        let deadline = Date().addingTimeInterval(3)
        while !banner.isHittable && Date() < deadline {
            Thread.sleep(forTimeInterval: 0.1)
        }
        let frame = banner.frame
        let windowFrame = window.frame
        XCTAssertTrue(banner.isHittable, "banner not hittable. frame=\(frame) window=\(windowFrame)", file: file, line: line)
        XCTAssertLessThanOrEqual(
            frame.minY,
            windowFrame.minY + 200,
            "banner frame.minY is not within the top 200pt. frame=\(frame) window=\(windowFrame)",
            file: file,
            line: line
        )
        let expanded = windowFrame.insetBy(dx: -1, dy: -1)
        XCTAssertTrue(
            expanded.contains(frame),
            "banner is not fully inside the window. frame=\(frame) window=\(windowFrame)",
            file: file,
            line: line
        )
    }

    private func launch(alert: String?, deskTab: String? = nil) throws -> XCUIApplication {
        let stub = try stubBinary()
        let app = XCUIApplication()
        app.launchEnvironment = [
            "WX_UI_TEST": "1",
            "WX_BINARY": stub,
            "WX_STUB_LOG": logURL.path,
            "WX_CONFIG": configURL.path,
            "WX_RADAR_REFRESH_SECONDS": "600",
            "WX_REFRESH_SECONDS": "600",
        ]
        if let alert {
            app.launchEnvironment["WX_STUB_ALERT"] = alert
        }
        if let deskTab {
            app.launchEnvironment["WX_DESK_TAB"] = deskTab
        }
        self.app = app
        app.launch()
        return app
    }

    private func stubBinary() throws -> String {
        let bundle = Bundle(for: AlertBannerUITests.self)
        let url = bundle.url(forResource: "wx-stub", withExtension: nil)
            ?? bundle.url(forResource: "wx-stub", withExtension: nil, subdirectory: "Fixtures")
        guard let url else {
            XCTFail("wx-stub is not in the test bundle")
            return ""
        }
        return url.path
    }

    private func element(_ app: XCUIApplication, _ identifier: String) -> XCUIElement {
        app.descendants(matching: .any).matching(identifier: identifier).firstMatch
    }

    private func debugLog() -> String {
        let text = (try? String(contentsOf: logURL, encoding: .utf8)) ?? ""
        return "stub log:\n\(text)"
    }
}
