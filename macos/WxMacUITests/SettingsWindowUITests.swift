import XCTest

/// Settings must open from the status menu and from ⌘,, then the MCP toggle
/// must change mcp.enabled in the scratch config. Release run:
/// `make mac-ui-test CONFIG=Release`.
final class SettingsWindowUITests: XCTestCase {
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
        try #"{"mcp":{"enabled":false}}"#.write(to: configURL, atomically: true, encoding: .utf8)
    }

    override func tearDownWithError() throws {
        app?.terminate()
        app = nil
        if let scratch {
            try? FileManager.default.removeItem(at: scratch)
        }
    }

    func testStatusMenuAndCommandCommaOpenSettingsAndPersistMCPToggle() throws {
        let app = try launch()
        let desk = app.windows["wx"]
        XCTAssertTrue(desk.waitForExistence(timeout: 8), "desk window missing")
        XCTAssertTrue(app.menuBars.menuBarItems["wx"].waitForExistence(timeout: 5), "app menu missing, so ⌘, is not installed")

        let fromStatus = openSettingsFromStatusMenu(app)
        XCTAssertTrue(fromStatus.waitForExistence(timeout: 5), "status menu did not open Settings. windows=\(windowTitles(app))")
        XCTAssertEqual(fromStatus.title, "Settings")
        XCTAssertTrue(desk.exists, "desk disappeared when Settings opened")
        let toggle = mcpToggle(in: fromStatus)
        XCTAssertTrue(toggle.waitForExistence(timeout: 5), "MCP toggle missing from the status-menu Settings window. \(snapshot(app))")

        closeSettings(app)
        XCTAssertFalse(app.windows["Settings"].exists, "Settings window still up after close. windows=\(windowTitles(app))")
        XCTAssertTrue(desk.exists)

        app.activate()
        app.typeKey(",", modifierFlags: .command)
        let fromComma = app.windows["Settings"]
        XCTAssertTrue(fromComma.waitForExistence(timeout: 5), "⌘, did not open Settings. windows=\(windowTitles(app))")
        XCTAssertEqual(fromComma.title, "Settings")
        let commaToggle = mcpToggle(in: fromComma)
        XCTAssertTrue(commaToggle.waitForExistence(timeout: 5), "MCP toggle missing after ⌘,. \(snapshot(app))")

        // load() arms `ready` on the next turn so the initial value does not write config.
        Thread.sleep(forTimeInterval: 0.5)
        clickMCPToggle(commaToggle, in: fromComma)
        try waitMCPEnabled(true)

        clickMCPToggle(commaToggle, in: fromComma)
        try waitMCPEnabled(false)
    }

    private func launch() throws -> XCUIApplication {
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
        self.app = app
        app.launch()
        return app
    }

    private func stubBinary() throws -> String {
        let bundle = Bundle(for: SettingsWindowUITests.self)
        let url = bundle.url(forResource: "wx-stub", withExtension: nil)
            ?? bundle.url(forResource: "wx-stub", withExtension: nil, subdirectory: "Fixtures")
        guard let url else {
            XCTFail("wx-stub is not in the test bundle")
            return ""
        }
        return url.path
    }

    @discardableResult
    private func openSettingsFromStatusMenu(_ app: XCUIApplication, file: StaticString = #filePath, line: UInt = #line) -> XCUIElement {
        let status = app.statusItems.firstMatch
        XCTAssertTrue(status.waitForExistence(timeout: 8), "status item missing", file: file, line: line)
        status.rightClick()
        let item = settingsMenuItem(app)
        if !item.waitForExistence(timeout: 4) {
            status.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.5)).rightClick()
        }
        let resolved = settingsMenuItem(app)
        XCTAssertTrue(
            resolved.waitForExistence(timeout: 4),
            "Settings… missing from the status menu. \(describeMenus(app))",
            file: file,
            line: line
        )
        resolved.click()
        return app.windows["Settings"]
    }

    /// Identifier only. The app menu also has a Settings… item, and matching
    /// that title would not prove the status menu opened.
    private func settingsMenuItem(_ app: XCUIApplication) -> XCUIElement {
        let pred = NSPredicate(format: "identifier == %@", "wx.status.settings")
        let inApp = app.menuItems.matching(pred).firstMatch
        let inStatus = app.statusItems.firstMatch.menus.menuItems.matching(pred).firstMatch
        // Prefer a hittable status-menu row once the popup is open.
        if inStatus.exists { return inStatus }
        return inApp
    }

    private func mcpToggle(in settings: XCUIElement) -> XCUIElement {
        settings.descendants(matching: .any).matching(identifier: "mcp-enabled-toggle").firstMatch
    }

    private func clickMCPToggle(_ toggle: XCUIElement, in settings: XCUIElement) {
        if toggle.isHittable {
            toggle.click()
            return
        }
        let sw = settings.switches["mcp-enabled-toggle"]
        if sw.exists && sw.isHittable {
            sw.click()
            return
        }
        let box = settings.checkBoxes["mcp-enabled-toggle"]
        if box.exists && box.isHittable {
            box.click()
            return
        }
        toggle.click()
    }

    private func closeSettings(_ app: XCUIApplication) {
        let settings = app.windows["Settings"]
        let close = settings.buttons[XCUIIdentifierCloseWindow]
        if close.waitForExistence(timeout: 2) && close.isHittable {
            close.click()
        } else {
            settings.click()
            app.typeKey("w", modifierFlags: .command)
        }
        let deadline = Date().addingTimeInterval(4)
        while app.windows["Settings"].exists && Date() < deadline {
            Thread.sleep(forTimeInterval: 0.1)
        }
    }

    private func waitMCPEnabled(_ expected: Bool, timeout: TimeInterval = 5, file: StaticString = #filePath, line: UInt = #line) throws {
        let deadline = Date().addingTimeInterval(timeout)
        var last = ""
        while Date() < deadline {
            last = (try? String(contentsOf: configURL, encoding: .utf8)) ?? ""
            if mcpEnabled(in: last) == expected { return }
            Thread.sleep(forTimeInterval: 0.1)
        }
        XCTFail("mcp.enabled never became \(expected). config:\n\(last)", file: file, line: line)
    }

    private func mcpEnabled(in text: String) -> Bool? {
        guard let data = text.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let mcp = obj["mcp"] as? [String: Any] else {
            return nil
        }
        return mcp["enabled"] as? Bool
    }

    private func windowTitles(_ app: XCUIApplication) -> String {
        app.windows.allElementsBoundByIndex.map(\.title).joined(separator: ", ")
    }

    private func describeMenus(_ app: XCUIApplication) -> String {
        let items = app.menuItems.allElementsBoundByIndex.prefix(30).map { item in
            "\(item.identifier):\(item.title)"
        }
        return "menuItems[\(items.joined(separator: " | "))] windows=\(windowTitles(app))"
    }

    private func snapshot(_ app: XCUIApplication) -> String {
        let texts = app.staticTexts.allElementsBoundByIndex.prefix(12).map { "\($0.identifier):\($0.label)" }
        let toggles = app.switches.allElementsBoundByIndex.prefix(8).map { "\($0.identifier):\($0.label)" }
        return "texts[\(texts.joined(separator: " | "))] switches[\(toggles.joined(separator: " | "))]"
    }
}
