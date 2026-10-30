import XCTest

/// iPad regular width shows radar beside the telemetry. Fixtures only. No live NWS.
/// Skips on iPhone, where the custom tab bar stays the shell.
final class WxPadSplitUITests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testRadarSitsBesideTelemetry() throws {
        try XCTSkipIf(UIDevice.current.userInterfaceIdiom != .pad, "split pane runs on iPad")

        let app = XCUIApplication()
        app.launchEnvironment = [
            "WX_UI_TEST": "1",
            "WX_FIXTURE": "1",
            "WX_SKIP_LOCATION": "1",
            "WX_DEFAULT_LOCATION": "New Haven, IN",
        ]
        app.launch()

        XCTAssertTrue(mark(app, "desk.split").waitForExistence(timeout: 15), dump(app))
        XCTAssertTrue(mark(app, "desk.telemetry").waitForExistence(timeout: 5), dump(app))
        XCTAssertTrue(mark(app, "desk.radar").exists, dump(app))
        XCTAssertTrue(mark(app, "surface.root").exists, dump(app))
        XCTAssertTrue(mark(app, "radar.root").exists, dump(app))
        XCTAssertTrue(text(app, "Fixture Partly Cloudy").waitForExistence(timeout: 15), dump(app))
        XCTAssertTrue(text(app, "Fixture Composite").waitForExistence(timeout: 15), dump(app))
    }

    private func mark(_ app: XCUIApplication, _ identifier: String) -> XCUIElement {
        app.descendants(matching: .any)[identifier]
    }

    private func text(_ app: XCUIApplication, _ label: String) -> XCUIElement {
        app.staticTexts[label]
    }

    private func dump(_ app: XCUIApplication) -> String {
        let ids = ["desk.split", "desk.telemetry", "desk.radar", "surface.root", "radar.root"]
            .map { id in "\(id)=\(mark(app, id).exists)" }
            .joined(separator: " ")
        let labels = app.staticTexts.allElementsBoundByIndex.prefix(40).map(\.label).joined(separator: " | ")
        return "\(ids) static texts: \(labels)"
    }
}
