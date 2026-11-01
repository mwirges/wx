import Darwin
import XCTest

/// Production Mac token write must be mode 0600 before the destination name exists.
/// Data.write(.atomic) under umask 022 leaves group/world bits until a later mode
/// change, so a write-then-adjust implementation fails the pre-rename lstat and
/// the source check for setAttributes / chmod.
final class MCPTokenFileTests: XCTestCase {
    private var scratch: URL!

    override func setUpWithError() throws {
        scratch = URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true)
            .appendingPathComponent("wx-mcp-token-\(UUID().uuidString)", isDirectory: true)
        try FileManager.default.createDirectory(at: scratch, withIntermediateDirectories: true)
    }

    override func tearDownWithError() throws {
        if let scratch {
            try? FileManager.default.removeItem(at: scratch)
        }
        scratch = nil
    }

    func testWriteTokenFileMode0600BeforeVisibleUnderUmask022() throws {
        let previous = umask(mode_t(0o022))
        defer { _ = umask(previous) }

        let token = String(repeating: "a", count: 32)
        let dest = scratch.appendingPathComponent("mcp.token")

        var observed: UInt16?
        var destVisibleDuringCreate = true
        var samePath = true
        var sameDirectory = false
        try WxConfig.writeTokenFile(dest, token: token) { temp in
            observed = self.posixMode(temp)
            destVisibleDuringCreate = FileManager.default.fileExists(atPath: dest.path)
            samePath = temp.standardizedFileURL.path == dest.standardizedFileURL.path
            sameDirectory = temp.deletingLastPathComponent().standardizedFileURL.path
                == dest.deletingLastPathComponent().standardizedFileURL.path
        }

        XCTAssertEqual(observed, UInt16(0o600), "temp mode before rename")
        XCTAssertFalse(destVisibleDuringCreate, "destination was visible before rename")
        XCTAssertFalse(samePath, "hook observed the destination rather than the temp")
        XCTAssertTrue(sameDirectory, "temp was not created in the destination directory")

        let finalMode = try XCTUnwrap(posixMode(dest))
        XCTAssertEqual(finalMode, UInt16(0o600))
        XCTAssertEqual(finalMode & 0o077, UInt16(0), "group or world bits are set")

        let payload = try Data(contentsOf: dest)
        XCTAssertEqual(payload.count, token.utf8.count + 1, "payload length")
        XCTAssertEqual(payload.last, UInt8(ascii: "\n"))

        let broken = scratch.appendingPathComponent("broken.token")
        try writeVisibleWithoutModeFix(broken, token: token)
        let brokenMode = try XCTUnwrap(posixMode(broken))
        XCTAssertNotEqual(brokenMode, UInt16(0o600))
        XCTAssertNotEqual(brokenMode & 0o077, UInt16(0), "umask 022 write without a create mode of 0600 stayed private")

        let source = try String(contentsOf: repoSource("macos/WxMac/Services/WxConfig.swift"), encoding: .utf8)
        let writer = try XCTUnwrap(writerSource(in: source))
        XCTAssertTrue(writer.contains("O_CREAT"))
        XCTAssertTrue(writer.contains("O_EXCL"))
        XCTAssertTrue(writer.contains("O_WRONLY"))
        XCTAssertTrue(writer.contains("0o600"))
        XCTAssertFalse(writer.contains("setAttributes"), "write-then-setAttributes is still in the writer")
        XCTAssertFalse(writer.contains("chmod"), "writer adjusts mode after create")
    }

    /// Deliberately broken stand-in for Data.write then a later mode change,
    /// with that later change skipped. Under umask 022 this is not 0600.
    private func writeVisibleWithoutModeFix(_ url: URL, token: String) throws {
        let data = Data((token + "\n").utf8)
        try data.write(to: url, options: .atomic)
    }

    private func posixMode(_ url: URL) -> UInt16? {
        var st = stat()
        let rc = url.path.withCString { lstat($0, &st) }
        if rc != 0 { return nil }
        return UInt16(st.st_mode & mode_t(0o777))
    }

    private func writerSource(in source: String) -> String? {
        guard let start = source.range(of: "static func writeTokenFile(") else { return nil }
        return String(source[start.lowerBound...])
    }

    private func repoSource(_ relative: String) -> URL {
        var url = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
        for part in relative.split(separator: "/") {
            url.appendPathComponent(String(part))
        }
        return url
    }
}
