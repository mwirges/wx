import AppKit
import ApplicationServices
import XCTest

/// Background refresh and the status-item HUD must not pull the desk forward.
/// Release: `make mac-ui-test CONFIG=Release`.
/// This class only: `xcodebuild -project macos/WxMac.xcodeproj -scheme WxMac -configuration Release -destination 'platform=macOS,arch=arm64' -derivedDataPath macos/build -only-testing:WxMacUITests/FocusStealUITests test`
final class FocusStealUITests: XCTestCase {
    private var scratch: URL!
    private var logURL: URL!
    private var configURL: URL!
    private var probeDir: URL?
    private var app: XCUIApplication?
    /// PID of the wx.app this test launched. Never the installed /Applications copy.
    private var launchedPID: pid_t?
    private var clickDebug = ""

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
        closeFinderProbe()
        app?.terminate()
        app = nil
        launchedPID = nil
        if let scratch {
            try? FileManager.default.removeItem(at: scratch)
        }
    }

    func testBackgroundRefreshDoesNotMakeWxFrontmost() throws {
        let app = try launch(refreshSeconds: "2", radarSeconds: "600")
        XCTAssertTrue(app.windows["wx"].waitForExistence(timeout: 8), "desk missing after launch")
        let wxPID = try requireLaunchedPID()
        _ = try waitMenuBarFetches(atLeast: 1, timeout: 8)

        let finderPID = try openFinderProbe()
        XCTAssertTrue(waitUntil(timeout: 8) {
            self.frontPID() == finderPID && self.finderCoversDesk(wxPID: wxPID, finderPID: finderPID)
        }, "Finder probe did not cover the desk before refresh. \(zOrderDebug())")

        let baseline = menuBarFetches()
        XCTAssertTrue(waitUntil(timeout: 12) {
            self.menuBarFetches() > baseline
        }, "background refresh did not run. log:\n\(stubLines().joined(separator: "\n"))")

        let deadline = Date().addingTimeInterval(1.5)
        while Date() < deadline {
            XCTAssertNotEqual(frontPID(), wxPID, "wx became frontmost during refresh. \(zOrderDebug())")
            XCTAssertTrue(
                finderCoversDesk(wxPID: wxPID, finderPID: finderPID),
                "desk moved in front of Finder during refresh. \(zOrderDebug())"
            )
            XCTAssertFalse(deskIsKey(wxPID: wxPID), "desk became key during refresh. \(focusDebug()) \(zOrderDebug())")
            Thread.sleep(forTimeInterval: 0.25)
        }
    }

    func testStatusPopoverDoesNotBringDeskKeyOrFront() throws {
        let app = try launch(refreshSeconds: "600", radarSeconds: "600")
        XCTAssertTrue(app.windows["wx"].waitForExistence(timeout: 8), "desk missing after launch")
        let wxPID = try requireLaunchedPID()
        let finderPID = try openFinderProbe()
        XCTAssertTrue(waitUntil(timeout: 8) {
            self.frontPID() == finderPID && self.finderCoversDesk(wxPID: wxPID, finderPID: finderPID)
        }, "Finder probe did not cover the desk before the popover. \(zOrderDebug())")

        openHUDWithoutActivating(pid: wxPID)
        let hud = app.descendants(matching: .any).matching(identifier: "wx.hud").firstMatch
        let opened = hud.waitForExistence(timeout: 4) || waitUntil(timeout: 2) { self.hudLayerIsUp(wxPID: wxPID) }
        XCTAssertTrue(opened, "status item path did not open the HUD. \(clickDebug) \(zOrderDebug())")

        XCTAssertNotEqual(frontPID(), wxPID, "opening the HUD made wx frontmost. \(zOrderDebug())")
        XCTAssertTrue(
            finderCoversDesk(wxPID: wxPID, finderPID: finderPID),
            "opening the HUD brought the desk in front of Finder. \(zOrderDebug())"
        )
        XCTAssertFalse(deskIsKey(wxPID: wxPID), "opening the HUD made the desk key. \(focusDebug())")
    }

    // MARK: - Launch

    private func launch(refreshSeconds: String, radarSeconds: String) throws -> XCUIApplication {
        let before = wxPIDsOutsideApplications()
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
        self.app = app
        app.launch()
        let after = wxPIDsOutsideApplications()
        let fresh = after.subtracting(before)
        if fresh.count == 1, let pid = fresh.first {
            launchedPID = pid
        } else if after.count == 1, let pid = after.first, before.isEmpty {
            launchedPID = pid
        } else {
            launchedPID = after.subtracting(before).first ?? after.first
        }
        return app
    }

    private func stubBinary() throws -> String {
        let bundle = Bundle(for: FocusStealUITests.self)
        let url = bundle.url(forResource: "wx-stub", withExtension: nil)
            ?? bundle.url(forResource: "wx-stub", withExtension: nil, subdirectory: "Fixtures")
        guard let url else {
            XCTFail("wx-stub is not in the test bundle")
            return ""
        }
        return url.path
    }

    /// Running wx processes whose executable is not /Applications/wx.app.
    private func wxPIDsOutsideApplications() -> Set<pid_t> {
        Set(NSWorkspace.shared.runningApplications.compactMap { running -> pid_t? in
            let path = running.executableURL?.path ?? ""
            guard path.hasSuffix("/Contents/MacOS/wx") else { return nil }
            guard !path.hasPrefix("/Applications/wx.app/") else { return nil }
            return running.processIdentifier
        })
    }

    private func requireLaunchedPID() throws -> pid_t {
        if let launchedPID { return launchedPID }
        let pids = wxPIDsOutsideApplications()
        if pids.count == 1, let pid = pids.first {
            launchedPID = pid
            return pid
        }
        XCTFail("could not identify the test wx pid. candidates=\(pids)")
        throw XCTSkip("no launched pid")
    }

    // MARK: - Finder probe

    /// Opens one uniquely named Finder window and activates Finder.
    /// tearDown closes only that window. POSIX file coercion stays outside
    /// the Finder tell, or Finder answers -600.
    private func openFinderProbe() throws -> pid_t {
        let dir = URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true)
            .appendingPathComponent("wx-82-focus-\(UUID().uuidString)", isDirectory: true)
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        probeDir = dir
        // NSWorkspace opens the folder without a Finder tell. Activate after,
        // so the probe window is the front document window.
        guard NSWorkspace.shared.open(dir) else {
            XCTFail("NSWorkspace did not open \(dir.path)")
            return -1
        }
        guard let finder = NSWorkspace.shared.runningApplications.first(where: { $0.bundleIdentifier == "com.apple.finder" }) else {
            XCTFail("Finder is not running")
            return -1
        }
        finder.activate(options: [.activateIgnoringOtherApps])
        return finder.processIdentifier
    }

    private func closeFinderProbe() {
        guard let dir = probeDir else { return }
        let script = """
        set probe to POSIX file "\(dir.path)" as alias
        tell application "Finder"
          close (every window whose target is probe)
        end tell
        """
        try? runAppleScript(script, failOnError: false)
        try? FileManager.default.removeItem(at: dir)
        probeDir = nil
    }

    /// Opens the HUD through the same `togglePopover` the status item uses.
    /// `XCUIElement.click()` on the status item logs `Activate` before the event
    /// and that activation orders the desk forward by itself, so it cannot prove
    /// this behavior. Quartz clicks from the runner never reached the menu bar.
    private func openHUDWithoutActivating(pid: pid_t) {
        clickDebug = "distributed wx.uitest.togglePopover pid=\(pid)"
        DistributedNotificationCenter.default().postNotificationName(
            Notification.Name("wx.uitest.togglePopover"),
            object: nil,
            userInfo: nil,
            deliverImmediately: true
        )
    }

    /// NSPopover windows sit above layer 0. The desk is the large layer-0 window.
    private func hudLayerIsUp(wxPID: pid_t) -> Bool {
        guard let raw = CGWindowListCopyWindowInfo([.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID) as? [[String: Any]] else {
            return false
        }
        for dict in raw {
            let pid = (dict[kCGWindowOwnerPID as String] as? NSNumber)?.int32Value ?? -1
            let layer = (dict[kCGWindowLayer as String] as? NSNumber)?.intValue ?? 0
            let bounds = dict[kCGWindowBounds as String] as? [String: Any]
            let width = (bounds?["Width"] as? NSNumber)?.doubleValue ?? 0
            let height = (bounds?["Height"] as? NSNumber)?.doubleValue ?? 0
            if pid == wxPID && layer > 0 && width >= 200 && height >= 200 {
                return true
            }
        }
        return false
    }

    private func runAppleScript(_ source: String, failOnError: Bool = true) throws {
        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: "/usr/bin/osascript")
        proc.arguments = ["-e", source]
        let out = Pipe()
        let err = Pipe()
        proc.standardOutput = out
        proc.standardError = err
        try proc.run()
        proc.waitUntilExit()
        if proc.terminationStatus != 0 {
            let message = String(data: err.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
            if failOnError {
                XCTFail("osascript \(proc.terminationStatus): \(message)")
            }
            throw NSError(domain: "osascript", code: Int(proc.terminationStatus), userInfo: [NSLocalizedDescriptionKey: message])
        }
    }

    // MARK: - Focus

    private func frontPID() -> pid_t? {
        NSWorkspace.shared.frontmostApplication?.processIdentifier
    }

    private struct Win {
        let pid: pid_t
        let name: String
        let owner: String
        let layer: Int
        let width: Double
        let height: Double
    }

    /// Front to back. Layer 0, large enough to be a document window rather than the HUD.
    private func documentWindows() -> [Win] {
        guard let raw = CGWindowListCopyWindowInfo([.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID) as? [[String: Any]] else {
            return []
        }
        return raw.compactMap { dict in
            let pid = (dict[kCGWindowOwnerPID as String] as? NSNumber)?.int32Value ?? -1
            let name = dict[kCGWindowName as String] as? String ?? ""
            let owner = dict[kCGWindowOwnerName as String] as? String ?? ""
            let layer = (dict[kCGWindowLayer as String] as? NSNumber)?.intValue ?? 0
            let bounds = dict[kCGWindowBounds as String] as? [String: Any]
            let width = (bounds?["Width"] as? NSNumber)?.doubleValue ?? 0
            let height = (bounds?["Height"] as? NSNumber)?.doubleValue ?? 0
            guard layer == 0, width >= 500, height >= 400 else { return nil }
            return Win(pid: pid, name: name, owner: owner, layer: layer, width: width, height: height)
        }
    }

    private func finderCoversDesk(wxPID: pid_t, finderPID: pid_t) -> Bool {
        let windows = documentWindows()
        guard let desk = windows.firstIndex(where: { $0.pid == wxPID }) else { return false }
        guard let finder = windows.firstIndex(where: { $0.pid == finderPID }) else { return false }
        return finder < desk
    }

    private func deskIsKey(wxPID: pid_t) -> Bool {
        guard let focus = focusedWindow() else { return false }
        if focus.pid != wxPID { return false }
        if focus.title == "wx" { return true }
        // Title can be empty when the window list redacts names. A standard window
        // owned by wx that is the focused window is the desk, not the HUD panel.
        return focus.role == "AXWindow" && focus.subrole == "AXStandardWindow"
    }

    private struct Focused {
        let pid: pid_t
        let title: String
        let role: String
        let subrole: String
    }

    private func focusedWindow() -> Focused? {
        let system = AXUIElementCreateSystemWide()
        var value: CFTypeRef?
        guard AXUIElementCopyAttributeValue(system, kAXFocusedWindowAttribute as CFString, &value) == .success,
              let value else {
            return nil
        }
        let element = value as! AXUIElement
        var pid: pid_t = 0
        AXUIElementGetPid(element, &pid)
        func stringAttr(_ name: String) -> String {
            var raw: CFTypeRef?
            AXUIElementCopyAttributeValue(element, name as CFString, &raw)
            return raw as? String ?? ""
        }
        return Focused(
            pid: pid,
            title: stringAttr(kAXTitleAttribute),
            role: stringAttr(kAXRoleAttribute),
            subrole: stringAttr(kAXSubroleAttribute)
        )
    }

    private func zOrderDebug() -> String {
        let rows = documentWindows().prefix(8).map { win in
            "pid=\(win.pid) owner=\(win.owner) name=\(win.name) \(Int(win.width))x\(Int(win.height))"
        }
        return "frontPID=\(frontPID() ?? -1) docs=[\(rows.joined(separator: " | "))]"
    }

    private func focusDebug() -> String {
        guard let focus = focusedWindow() else { return "focus=none" }
        return "focus pid=\(focus.pid) title=\(focus.title) role=\(focus.role) subrole=\(focus.subrole)"
    }

    // MARK: - Stub log

    private func stubLines() -> [String] {
        guard let text = try? String(contentsOf: logURL, encoding: .utf8) else { return [] }
        return text.split(separator: "\n", omittingEmptySubsequences: true).map(String.init)
    }

    private func menuBarFetches() -> Int {
        stubLines().filter {
            $0.contains("--json") && $0.contains("--forecast") && $0.contains("--alerts")
        }.count
    }

    private func waitMenuBarFetches(atLeast minimum: Int, timeout: TimeInterval) throws -> Int {
        let deadline = Date().addingTimeInterval(timeout)
        var count = menuBarFetches()
        while count < minimum && Date() < deadline {
            Thread.sleep(forTimeInterval: 0.2)
            count = menuBarFetches()
        }
        XCTAssertGreaterThanOrEqual(count, minimum, "stub log never reached \(minimum) menu-bar fetches:\n\(stubLines().joined(separator: "\n"))")
        return count
    }

    @discardableResult
    private func waitUntil(timeout: TimeInterval, _ condition: () -> Bool) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if condition() { return true }
            Thread.sleep(forTimeInterval: 0.2)
        }
        return condition()
    }
}
