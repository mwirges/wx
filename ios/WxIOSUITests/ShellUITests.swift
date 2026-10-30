import XCTest

/// Taps every iPhone tab and checks Surface against the bundled fixture.
/// Launch env forces fixtures. This test does not call NWS.
final class WxIOSShellUITests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testEveryTabAndSurfaceFixture() throws {
        let app = XCUIApplication()
        app.launchEnvironment = [
            "WX_UI_TEST": "1",
            "WX_FIXTURE": "1",
            "WX_SKIP_LOCATION": "1",
            "WX_DEFAULT_LOCATION": "New Haven, IN",
        ]
        app.launch()

        assertSurface(app)
        XCTAssertEqual(app.tabBars.count, 0, "system tab bar is visible. \(dump(app))")
        XCTAssertTrue(app.descendants(matching: .any)["tabbar.custom"].exists, dump(app))
        XCTAssertFalse(app.tabBars.buttons["More"].exists, "system More tab is visible. \(dump(app))")
        XCTAssertFalse(app.descendants(matching: .any)["desk.split"].exists, "iPhone showed the iPad split. \(dump(app))")
        open(app, "Tactical")
        XCTAssertTrue(text(app, "Tactical sensors").waitForExistence(timeout: 5), dump(app))
        open(app, "Radar")
        XCTAssertTrue(text(app, "Fixture Composite").waitForExistence(timeout: 5), dump(app))
        open(app, "Severe")
        XCTAssertTrue(text(app, "Fixture Slight").waitForExistence(timeout: 5), dump(app))
        open(app, "Tropics")
        XCTAssertTrue(text(app, "Fixture Storm").waitForExistence(timeout: 5), dump(app))
        open(app, "Config")
        XCTAssertTrue(text(app, "Saved place").waitForExistence(timeout: 5), dump(app))
        let field = app.textFields["config.place"]
        XCTAssertTrue(field.waitForExistence(timeout: 3), dump(app))
        XCTAssertEqual(field.value as? String, "New Haven, IN", dump(app))
        open(app, "Surface")
        assertSurface(app)

        XCTAssertFalse(app.buttons["tab.Climate"].exists)
        XCTAssertFalse(app.buttons["tab.Grid"].exists)
        XCTAssertFalse(app.buttons["tab.CPC"].exists)
    }

    private func assertSurface(_ app: XCUIApplication, file: StaticString = #filePath, line: UInt = #line) {
        XCTAssertTrue(text(app, "Fixture Partly Cloudy").waitForExistence(timeout: 15), dump(app), file: file, line: line)
        XCTAssertTrue(text(app, "Fixture Dry next 6h").exists, dump(app), file: file, line: line)
        XCTAssertTrue(text(app, "Fixture Tuesday").exists, dump(app), file: file, line: line)
        XCTAssertTrue(text(app, "15:00").exists, dump(app), file: file, line: line)
        XCTAssertTrue(text(app, "70°").exists, dump(app), file: file, line: line)
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
