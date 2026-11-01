import XCTest

/// pid <= 1 must throw and must not run the terminate closure.
/// The old spawn guard called Process.terminate() before throwing, which fails
/// the terminateCalled assertion. A spawn that does not call rejectInvalidSpawnPID
/// fails the source assertion.
final class MCPSpawnGuardTests: XCTestCase {
    func testRejectInvalidSpawnPIDDoesNotTerminateForZeroOneOrNegative() throws {
        let forbidden: [Int32] = [0, 1, -1]
        for pid in forbidden {
            var terminateCalled = false
            XCTAssertThrowsError(
                try MCPSpawnGuard.rejectInvalidSpawnPID(pid) {
                    terminateCalled = true
                }
            ) { error in
                guard case MCPSpawnPIDError.refused(let got) = error else {
                    return XCTFail("unexpected error for pid \(pid): \(error)")
                }
                XCTAssertEqual(got, pid)
            }
            XCTAssertFalse(terminateCalled, "terminate closure ran for pid \(pid)")
        }

        var terminateCalled = false
        XCTAssertNoThrow(
            try MCPSpawnGuard.rejectInvalidSpawnPID(42) {
                terminateCalled = true
            }
        )
        XCTAssertFalse(terminateCalled, "terminate closure ran for a normal pid")

        let supervisor = try String(contentsOf: repoSource("macos/WxMac/Services/MCPServerSupervisor.swift"), encoding: .utf8)
        let guardSource = try String(contentsOf: repoSource("macos/WxMac/Services/MCPSpawnGuard.swift"), encoding: .utf8)
        XCTAssertTrue(supervisor.contains("rejectInvalidSpawnPID"))
        XCTAssertFalse(supervisor.contains(".terminate("), "spawn must not call Process.terminate")
        XCTAssertFalse(guardSource.contains("terminate()"), "helper must not invoke the terminate closure")
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
