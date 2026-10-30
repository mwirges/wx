import Darwin
import XCTest

/// Release and Debug both run this. It does not set WX_FIXTURE, so WxRunJSON
/// is the live bridge. It locks HOME/.cache first. Without the cache fix the
/// forecast command returns command_error cache init and the temperature stays
/// an em dash.
final class WxIOSLiveBridgeUITests: XCTestCase {
    private let capturePointer = "/Users/mwirges/code/tmp/wx-iosdevice-shots/capture-dir.txt"

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testUnwritableHomeCacheStillLoads() throws {
        let container = try appContainer()
        try lockDotCache(container)

        let app = XCUIApplication()
        app.launchEnvironment = [
            "WX_SKIP_LOCATION": "1",
            "WX_DEFAULT_LOCATION": "New Haven, IN",
        ]
        app.launch()

        let temp = app.descendants(matching: .any)["surface.temp"]
        XCTAssertTrue(temp.waitForExistence(timeout: 20), dump(app))

        var label = temp.label
        let deadline = Date().addingTimeInterval(40)
        while Date() < deadline {
            if app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "cache init")).firstMatch.exists {
                XCTFail("cache init reached the UI. \(dump(app))")
            }
            label = temp.label
            if label.contains("°") {
                break
            }
            Thread.sleep(forTimeInterval: 0.5)
        }
        XCTAssertTrue(label.contains("°"), "temperature stayed empty (\(label)). \(dump(app))")
        XCTAssertFalse(label.contains("—"), "temperature is a dash. \(dump(app))")
        XCTAssertFalse(app.staticTexts["No day periods"].exists, dump(app))
        XCTAssertFalse(app.staticTexts["No hourly periods"].exists, dump(app))
        XCTAssertEqual(app.tabBars.count, 0, "system tab bar is visible. \(dump(app))")
        XCTAssertTrue(app.descendants(matching: .any)["tabbar.custom"].exists, dump(app))
        XCTAssertFalse(app.tabBars.buttons["More"].exists, dump(app))
        let nowcast = app.descendants(matching: .any)["surface.nowcast"].label
        if nowcast.range(of: "clear", options: .caseInsensitive) != nil {
            XCTAssertFalse(app.staticTexts["NONE"].exists, "dry nowcast still tagged NONE. \(dump(app))")
        }

        try captureIfRequested(app)
    }

    /// Make HOME/.cache unwritable inside the app container. MkdirAll of
    /// .cache/wx then fails the way the device sandbox fails.
    private func lockDotCache(_ container: String) throws {
        let fm = FileManager.default
        let dot = (container as NSString).appendingPathComponent(".cache")
        let wx = (dot as NSString).appendingPathComponent("wx")
        if fm.fileExists(atPath: dot) {
            guard chmod(dot, 0o755) == 0 else {
                XCTFail("could not chmod \(dot) writable to reset it")
                return
            }
        }
        if fm.fileExists(atPath: wx) {
            try fm.removeItem(atPath: wx)
        }
        if !fm.fileExists(atPath: dot) {
            try fm.createDirectory(atPath: dot, withIntermediateDirectories: true)
        }
        guard chmod(dot, 0o555) == 0 else {
            XCTFail("could not lock \(dot)")
            return
        }
        do {
            try fm.createDirectory(atPath: wx, withIntermediateDirectories: true)
            XCTFail("HOME/.cache stayed writable at \(dot)")
        } catch {
            // permission denied is the device-like condition
        }
    }

    private func appContainer() throws -> String {
        if let dir = try container(restrictToPreferredUDID: true) {
            return dir
        }
        if let dir = try container(restrictToPreferredUDID: false) {
            return dir
        }
        XCTFail("wx iOS data container not installed")
        return ""
    }

    /// UI tests run inside the simulator container, so NSHomeDirectory is not
    /// the Mac user. The container path still sits under CoreSimulator/Devices.
    private func simulatorDevicesRoot() -> String {
        let marker = "/Library/Developer/CoreSimulator/Devices"
        let home = NSHomeDirectory()
        if let range = home.range(of: marker) {
            return String(home[..<range.upperBound])
        }
        return (home as NSString).appendingPathComponent("Library/Developer/CoreSimulator/Devices")
    }

    private func container(restrictToPreferredUDID: Bool) throws -> String? {
        let root = simulatorDevicesRoot()
        let fm = FileManager.default
        guard let devices = try? fm.contentsOfDirectory(atPath: root) else {
            XCTFail("no simulator devices under \(root)")
            return nil
        }
        let preferred = restrictToPreferredUDID ? ProcessInfo.processInfo.environment["SIMULATOR_UDID"] : nil
        var matches: [(String, Date)] = []
        for device in devices {
            if let preferred, !preferred.isEmpty, device != preferred {
                continue
            }
            let apps = (root as NSString).appendingPathComponent("\(device)/data/Containers/Data/Application")
            guard let uuids = try? fm.contentsOfDirectory(atPath: apps) else { continue }
            for uuid in uuids {
                let dir = (apps as NSString).appendingPathComponent(uuid)
                let meta = (dir as NSString).appendingPathComponent(".com.apple.mobile_container_manager.metadata.plist")
                guard let data = fm.contents(atPath: meta),
                      let plist = try? PropertyListSerialization.propertyList(from: data, format: nil) as? [String: Any],
                      plist["MCMMetadataIdentifier"] as? String == "com.mwirges.wx.ios" else {
                    continue
                }
                let modified = (try? fm.attributesOfItem(atPath: dir)[.modificationDate] as? Date) ?? .distantPast
                matches.append((dir, modified))
            }
        }
        return matches.max(by: { $0.1 < $1.1 })?.0
    }

    private func captureIfRequested(_ app: XCUIApplication) throws {
        guard let raw = try? String(contentsOfFile: capturePointer, encoding: .utf8) else { return }
        let dir = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !dir.isEmpty else { return }
        try FileManager.default.createDirectory(atPath: dir, withIntermediateDirectories: true)
        shot(dir, "01-surface")
        app.swipeUp()
        shot(dir, "02-surface-scrolled")
        app.swipeUp()
        shot(dir, "03-surface-hourly")
        let tabs = [
            ("Tactical", "04-tactical", 2.0),
            ("Radar", "05-radar", 6.0),
            ("Severe", "06-severe", 6.0),
            ("Tropics", "07-tropics", 6.0),
            ("Config", "08-config", 2.0),
        ]
        for (tab, name, pause) in tabs {
            let button = app.buttons["tab.\(tab)"]
            XCTAssertTrue(button.waitForExistence(timeout: 5), "missing \(tab). \(dump(app))")
            button.tap()
            Thread.sleep(forTimeInterval: pause)
            shot(dir, name)
        }
        app.buttons["tab.Surface"].tap()
        Thread.sleep(forTimeInterval: 1)
        shot(dir, "09-surface-return")
    }

    private func shot(_ dir: String, _ name: String) {
        let png = (dir as NSString).appendingPathComponent("\(name).png")
        try? FileManager.default.removeItem(atPath: png)
        let request = "/tmp/wx-ios-shot-request.txt"
        try? png.write(toFile: request, atomically: true, encoding: .utf8)
        let deadline = Date().addingTimeInterval(20)
        while Date() < deadline {
            if FileManager.default.fileExists(atPath: png) {
                return
            }
            Thread.sleep(forTimeInterval: 0.2)
        }
        XCTFail("host did not capture \(png)")
    }

    private func dump(_ app: XCUIApplication) -> String {
        let labels = app.staticTexts.allElementsBoundByIndex.prefix(40).map(\.label).joined(separator: " | ")
        return "static texts: \(labels)"
    }
}
