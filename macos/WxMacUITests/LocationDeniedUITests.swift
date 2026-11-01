import XCTest

/// Location-denied UX through the shipped wx.app (WX_UI_TEST=1, stub CLI).
///
/// XCUITest cannot flip this Mac's TCC Location decision for com.mwirges.wx
/// without writing the user privacy database. These tests therefore set
/// WX_LOCATION_AUTH, which injects CLAuthorizationStatus only (and
/// WX_LOCATION_FAIL, which injects the CLError a failed fix would deliver).
/// No view is stubbed. A Darwin notification
/// `com.mwirges.wx.uitest.location-auth.<status>` flips that injected status
/// mid-run. The real System Settings prompt is manual on a Release build.
final class LocationDeniedUITests: XCTestCase {
    private let accessOff = "Location access is off for wx"
    private let settingsURL = "x-apple.systempreferences:com.apple.preference.security?Privacy_LocationServices"
    private let usage = "wx uses your location to provide accurate local weather observations, alerts, and Doppler radar telemetry."

    private var scratch: URL!
    private var logURL: URL!
    private var configURL: URL!
    private var openLogURL: URL!
    private var app: XCUIApplication?

    override func setUpWithError() throws {
        continueAfterFailure = false
        scratch = URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true)
            .appendingPathComponent("wx-ui-\(UUID().uuidString)", isDirectory: true)
        try FileManager.default.createDirectory(at: scratch, withIntermediateDirectories: true)
        logURL = scratch.appendingPathComponent("stub.log")
        configURL = scratch.appendingPathComponent("config.json")
        openLogURL = scratch.appendingPathComponent("open-url.log")
        FileManager.default.createFile(atPath: logURL.path, contents: Data())
        try "{}".write(to: configURL, atomically: true, encoding: .utf8)
    }

    override func tearDownWithError() throws {
        app?.terminate()
        app = nil
        if let scratch {
            try? FileManager.default.removeItem(at: scratch)
        }
    }

    func testDeskGPSDeniedShowsAccessOffAndOpensSettings() throws {
        let app = try launch(auth: "denied", recordOpens: true)
        let window = app.windows["wx"]
        XCTAssertTrue(window.waitForExistence(timeout: 8))
        XCTAssertTrue(window.staticTexts["TACTICAL SENSOR CONTROL"].waitForExistence(timeout: 8))
        XCTAssertTrue(app.staticTexts["Tonight"].waitForExistence(timeout: 10), "forecast never loaded. \(debugLog())")

        let gps = element(window, "wx.use-current-location")
        XCTAssertTrue(waitHittable(gps, timeout: 5), "location.fill button not hittable")
        gps.click()

        assertAccessOff(in: window, app: app)
        assertNoRawCoreLocation(app)

        let settings = element(window, "wx.open-location-settings")
        XCTAssertTrue(waitHittable(settings, timeout: 5), "Settings control not hittable. frame=\(settings.frame)")
        settings.click()

        XCTAssertTrue(waitForOpenURL(), "open-url log missing \(settingsURL). log=\(openLogText())")
        let settingsApp = XCUIApplication(bundleIdentifier: "com.apple.systempreferences")
        if settingsApp.wait(for: .runningForeground, timeout: 3) {
            settingsApp.typeKey("h", modifierFlags: .command)
        }
    }

    func testRestrictedMatchesDeniedCopy() throws {
        let app = try launch(auth: "restricted")
        let window = app.windows["wx"]
        XCTAssertTrue(window.waitForExistence(timeout: 8))
        XCTAssertTrue(app.staticTexts["Tonight"].waitForExistence(timeout: 10))
        let gps = element(window, "wx.use-current-location")
        XCTAssertTrue(waitHittable(gps, timeout: 5))
        gps.click()
        assertAccessOff(in: window, app: app)
        XCTAssertTrue(waitHittable(element(window, "wx.open-location-settings"), timeout: 5))
        assertNoRawCoreLocation(app)
    }

    /// Use Current Location is ⌘⇧L. ⌘L recenters the radar.
    func testUseCurrentLocationMenuShowsAccessOff() throws {
        let app = try launch(auth: "denied")
        let window = app.windows["wx"]
        XCTAssertTrue(window.waitForExistence(timeout: 8))
        XCTAssertTrue(app.staticTexts["Tonight"].waitForExistence(timeout: 10))
        app.activate()
        // ⌘⇧L is the menu key equivalent. XCTest sometimes drops that chord
        // (DisplayManager inf rect), so the View menu item is the fallback.
        app.typeKey("l", modifierFlags: [.command, .shift])
        if !element(window, "wx.location-denied").waitForExistence(timeout: 2) {
            app.menuBars.menuBarItems["View"].click()
            let item = app.menuItems["Use Current Location"]
            XCTAssertTrue(item.waitForExistence(timeout: 3), "View menu is missing Use Current Location")
            item.click()
        }
        assertAccessOff(in: window, app: app)
        assertNoRawCoreLocation(app)
    }

    /// Popover ControlsBar is the tactical HUD (`menu_bar_format=tactical`).
    /// Compact and standard popovers do not host the location bar.
    func testPopoverUseCurrentLocationShowsAccessOff() throws {
        try #"{"menu_bar_format":"tactical"}"#.write(to: configURL, atomically: true, encoding: .utf8)
        let app = try launch(auth: "denied")
        let window = app.windows["wx"]
        XCTAssertTrue(window.waitForExistence(timeout: 8))
        XCTAssertTrue(window.staticTexts["TACTICAL SENSOR CONTROL"].waitForExistence(timeout: 8))

        app.typeKey("w", modifierFlags: .command)
        let closed = Date().addingTimeInterval(4)
        while app.windows["wx"].exists && Date() < closed {
            Thread.sleep(forTimeInterval: 0.1)
        }
        XCTAssertFalse(app.windows["wx"].exists, "desk still open")

        app.statusItems.firstMatch.click()
        let card = app.staticTexts["TACTICAL SENSOR CONTROL"]
        if !card.waitForExistence(timeout: 6) {
            app.statusItems.firstMatch.click()
        }
        XCTAssertTrue(card.waitForExistence(timeout: 6), "tactical popover did not show. \(snapshot(app))")

        let gps = element(app, "wx.use-current-location")
        XCTAssertTrue(gps.waitForExistence(timeout: 5), "popover location control missing. \(snapshot(app))")
        XCTAssertTrue(waitHittable(gps, timeout: 4), "popover location control not hittable. \(snapshot(app))")
        gps.click()

        assertAccessOff(in: app, app: app)
        let settings = element(app, "wx.open-location-settings")
        XCTAssertTrue(waitHittable(settings, timeout: 5), "popover Settings control not hittable. \(snapshot(app))")
        XCTAssertTrue(element(app, "wx.locations-menu").exists, "popover locations menu missing")
        assertNoRawCoreLocation(app)
    }

    func testCoreLocationFailuresAreFriendly() throws {
        try assertFailure(auth: "authorizedWhenInUse", fail: "denied", accessOff: true, friendly: accessOff)
        try assertFailure(auth: "authorizedWhenInUse", fail: "locationUnknown", accessOff: false,
                          friendly: "Location is temporarily unavailable. Try again.")
        try assertFailure(auth: "authorizedWhenInUse", fail: "network", accessOff: false,
                          friendly: "Location lookup needs a network connection.")
        try assertFailure(auth: "authorizedWhenInUse", fail: "headingFailure", accessOff: false,
                          friendly: "Compass heading is unavailable.")
    }

    func testNotDeterminedWaitsThenResolvesOnGrant() throws {
        let built = try builtAppURL()
        let entitlements = try codesignEntitlements(of: built)
        XCTAssertTrue(
            entitlements.contains("com.apple.security.personal-information.location"),
            "Debug wx.app is missing the location entitlement.\n\(entitlements)"
        )
        XCTAssertFalse(
            entitlements.contains("com.apple.security.app-sandbox"),
            "Debug wx.app picked up the app sandbox; that is issue #78.\n\(entitlements)"
        )
        let info = NSDictionary(contentsOf: built.appendingPathComponent("Contents/Info.plist")) as? [String: Any]
        XCTAssertEqual(info?["NSLocationWhenInUseUsageDescription"] as? String, usage)
        XCTAssertEqual(info?["NSLocationUsageDescription"] as? String, usage)

        let app = try launch(auth: "notDetermined")
        let window = app.windows["wx"]
        XCTAssertTrue(window.waitForExistence(timeout: 8))
        XCTAssertTrue(app.staticTexts["Tonight"].waitForExistence(timeout: 10), debugLog())
        let before = locationFetches()

        let gps = element(window, "wx.use-current-location")
        XCTAssertTrue(waitHittable(gps, timeout: 5))
        gps.click()
        Thread.sleep(forTimeInterval: 1.5)

        XCTAssertFalse(element(window, "wx.location-denied").exists, "denied UI before authorization was decided. \(snapshot(app))")
        XCTAssertFalse(element(app, "wx.location-alert").exists, snapshot(app))
        assertNoRawCoreLocation(app)
        XCTAssertEqual(locationFetches(), before, "requestLocation ran before authorization changed.\n\(debugLog())")

        postLocationAuth("authorizedWhenInUse")
        XCTAssertTrue(waitForLocationFetch(above: before, timeout: 20), "grant did not fetch --location.\n\(debugLog())")
        let field = locationField(in: window)
        XCTAssertTrue(field.waitForExistence(timeout: 3))
        let value = field.value as? String ?? ""
        XCTAssertFalse(value.isEmpty)
        XCTAssertNotEqual(value, "Zip or City, ST")
        XCTAssertFalse(value.contains("kCLErrorDomain"))
        XCTAssertFalse(value.contains("error 1"))
        assertNoRawCoreLocation(app)
        XCTAssertFalse(element(window, "wx.location-denied").exists)
    }

    func testDeniedThenReallowClearsAlertAndSecondTapFetches() throws {
        let app = try launch(auth: "denied")
        let window = app.windows["wx"]
        XCTAssertTrue(window.waitForExistence(timeout: 8))
        XCTAssertTrue(app.staticTexts["Tonight"].waitForExistence(timeout: 10), debugLog())
        let before = locationFetches()

        let gps = element(window, "wx.use-current-location")
        XCTAssertTrue(waitHittable(gps, timeout: 5))
        gps.click()
        assertAccessOff(in: window, app: app)
        assertNoRawCoreLocation(app)

        postLocationAuth("authorizedWhenInUse")
        let deadline = Date().addingTimeInterval(5)
        while element(window, "wx.location-denied").exists && Date() < deadline {
            Thread.sleep(forTimeInterval: 0.1)
        }
        XCTAssertFalse(element(window, "wx.location-denied").exists, "stale denial stayed after access was granted. \(snapshot(app))")
        XCTAssertFalse(window.staticTexts[accessOff].exists)
        XCTAssertFalse(element(app, "wx.location-alert").exists, snapshot(app))
        assertNoRawCoreLocation(app)
        XCTAssertEqual(locationFetches(), before, "re-allow started a fetch before the second tap.\n\(debugLog())")

        XCTAssertTrue(waitHittable(gps, timeout: 5), "GPS button stayed disabled after re-allow")
        gps.click()
        XCTAssertTrue(waitForLocationFetch(above: before, timeout: 20), "second GPS tap did not fetch --location.\n\(debugLog())")
        let value = locationField(in: window).value as? String ?? ""
        XCTAssertFalse(value.isEmpty)
        XCTAssertNotEqual(value, "Zip or City, ST")
        XCTAssertFalse(value.contains("kCLErrorDomain"))
        XCTAssertFalse(element(window, "wx.location-denied").exists)
        let alert = element(app, "wx.location-alert")
        if alert.exists {
            XCTAssertFalse(alert.label.contains("kCLErrorDomain"))
            XCTAssertFalse(alert.label.contains("error 1"))
        }
        assertNoRawCoreLocation(app)
    }

    // MARK: - Helpers

    private func assertAccessOff(in root: XCUIElement, app: XCUIApplication, file: StaticString = #filePath, line: UInt = #line) {
        let denied = element(root, "wx.location-denied")
        XCTAssertTrue(denied.waitForExistence(timeout: 5), "wx.location-denied missing. \(snapshot(app))", file: file, line: line)
        let value = denied.value as? String ?? ""
        XCTAssertTrue(
            denied.label == accessOff || value == accessOff,
            "denied copy label=\(denied.label) value=\(value). \(snapshot(app))",
            file: file,
            line: line
        )
    }

    private func assertFailure(auth: String, fail: String, accessOff: Bool, friendly: String) throws {
        let app = try launch(auth: auth, fail: fail)
        let window = app.windows["wx"]
        XCTAssertTrue(window.waitForExistence(timeout: 8))
        XCTAssertTrue(app.staticTexts["Tonight"].waitForExistence(timeout: 10), debugLog())
        let gps = element(window, "wx.use-current-location")
        XCTAssertTrue(waitHittable(gps, timeout: 5))
        gps.click()
        if accessOff {
            assertAccessOff(in: window, app: app)
            XCTAssertTrue(element(window, "wx.open-location-settings").exists)
        } else {
            let alert = element(window, "wx.location-alert")
            XCTAssertTrue(alert.waitForExistence(timeout: 5), "\(fail): \(snapshot(app))")
            XCTAssertTrue(alert.label.contains(friendly), "label=\(alert.label) value=\(String(describing: alert.value))")
            XCTAssertFalse(alert.label.contains("kCLErrorDomain"))
            XCTAssertFalse(alert.label.contains("error 1"))
        }
        assertNoRawCoreLocation(app)
        app.terminate()
        self.app = nil
    }

    private func launch(auth: String, fail: String? = nil, recordOpens: Bool = false) throws -> XCUIApplication {
        app?.terminate()
        FileManager.default.createFile(atPath: logURL.path, contents: Data())
        let stub = try stubBinary()
        let app = XCUIApplication()
        app.launchEnvironment = [
            "WX_UI_TEST": "1",
            "WX_BINARY": stub,
            "WX_STUB_LOG": logURL.path,
            "WX_CONFIG": configURL.path,
            "WX_RADAR_REFRESH_SECONDS": "600",
            "WX_REFRESH_SECONDS": "600",
            "WX_LOCATION_AUTH": auth,
        ]
        if let fail {
            app.launchEnvironment["WX_LOCATION_FAIL"] = fail
        }
        if recordOpens {
            FileManager.default.createFile(atPath: openLogURL.path, contents: Data())
            app.launchEnvironment["WX_OPEN_URL_LOG"] = openLogURL.path
        }
        self.app = app
        app.launch()
        return app
    }

    private func stubBinary() throws -> String {
        let bundle = Bundle(for: LocationDeniedUITests.self)
        let url = bundle.url(forResource: "wx-stub", withExtension: nil)
            ?? bundle.url(forResource: "wx-stub", withExtension: nil, subdirectory: "Fixtures")
        guard let url else {
            XCTFail("wx-stub is not in the test bundle")
            return ""
        }
        return url.path
    }

    private func element(_ root: XCUIElement, _ identifier: String) -> XCUIElement {
        root.descendants(matching: .any).matching(identifier: identifier).firstMatch
    }

    private func waitHittable(_ element: XCUIElement, timeout: TimeInterval) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if element.exists && element.isHittable { return true }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return element.exists && element.isHittable
    }

    private func assertNoRawCoreLocation(_ app: XCUIApplication, file: StaticString = #filePath, line: UInt = #line) {
        let pred = NSPredicate(
            format: "label CONTAINS[c] 'kCLErrorDomain' OR label CONTAINS 'error 1' OR value CONTAINS[c] 'kCLErrorDomain' OR value CONTAINS 'error 1'"
        )
        let hit = app.descendants(matching: .any).matching(pred).firstMatch
        if hit.exists {
            XCTFail("raw CoreLocation text label=\(hit.label) value=\(String(describing: hit.value))", file: file, line: line)
        }
    }

    private func postLocationAuth(_ token: String) {
        let name = "com.mwirges.wx.uitest.location-auth.\(token)" as CFString
        CFNotificationCenterPostNotification(
            CFNotificationCenterGetDarwinNotifyCenter(),
            CFNotificationName(name),
            nil,
            nil,
            true
        )
    }

    private func stubLines() -> [String] {
        guard let text = try? String(contentsOf: logURL, encoding: .utf8) else { return [] }
        return text.split(separator: "\n", omittingEmptySubsequences: true).map(String.init)
    }

    private func locationFetches() -> Int {
        stubLines().filter { $0.contains("--location") }.count
    }

    private func waitForLocationFetch(above baseline: Int, timeout: TimeInterval) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if locationFetches() > baseline { return true }
            Thread.sleep(forTimeInterval: 0.2)
        }
        return locationFetches() > baseline
    }

    private func waitForOpenURL() -> Bool {
        let deadline = Date().addingTimeInterval(4)
        while Date() < deadline {
            if openLogText().contains(settingsURL) { return true }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return openLogText().contains(settingsURL)
    }

    private func openLogText() -> String {
        (try? String(contentsOf: openLogURL, encoding: .utf8)) ?? ""
    }

    private func locationField(in root: XCUIElement) -> XCUIElement {
        let fields = root.textFields.matching(identifier: "wx.location")
        for field in fields.allElementsBoundByIndex where field.exists {
            return field
        }
        return fields.firstMatch
    }

    private func debugLog() -> String {
        "stub log:\n" + stubLines().joined(separator: "\n")
    }

    private func snapshot(_ app: XCUIApplication) -> String {
        let texts = app.staticTexts.allElementsBoundByIndex.prefix(12).map { "\($0.identifier):\($0.label)" }
        let buttons = app.buttons.allElementsBoundByIndex.prefix(12).map { "\($0.identifier):\($0.label)" }
        return "texts[\(texts.joined(separator: " | "))] buttons[\(buttons.joined(separator: " | "))]"
    }

    private func builtAppURL() throws -> URL {
        let bundleURL = Bundle(for: LocationDeniedUITests.self).bundleURL
        var derived = bundleURL
        derived.deleteLastPathComponent() // PlugIns
        derived.deleteLastPathComponent() // Contents
        derived.deleteLastPathComponent() // WxMacUITests-Runner.app
        let fromBundle = derived.appendingPathComponent("wx.app")

        let source = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent() // WxMacUITests
            .deletingLastPathComponent() // macos
            .appendingPathComponent("build/Build/Products/Debug/wx.app")

        for url in [fromBundle, source] where FileManager.default.fileExists(atPath: url.path) {
            return url
        }
        XCTFail("Debug wx.app not found at \(fromBundle.path) or \(source.path)")
        struct MissingApp: Error {}
        throw MissingApp()
    }

    private func codesignEntitlements(of app: URL) throws -> String {
        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: "/usr/bin/codesign")
        proc.arguments = ["-d", "--entitlements", ":-", app.path]
        let out = Pipe()
        let err = Pipe()
        proc.standardOutput = out
        proc.standardError = err
        try proc.run()
        proc.waitUntilExit()
        let stdout = String(data: out.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
        let stderr = String(data: err.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
        return stdout + "\n" + stderr
    }
}
