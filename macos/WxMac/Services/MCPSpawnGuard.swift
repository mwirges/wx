import Foundation

/// Spawn-pid check for `MCPServerSupervisor.spawn`.
///
/// pid 0, pid 1, and any negative id are refused. `terminate` is accepted so
/// a test can prove the closure is not run. This function never invokes it.
enum MCPSpawnGuard {
    static func rejectInvalidSpawnPID(_ pid: Int32, terminate: () -> Void) throws {
        if pid <= 1 {
            throw MCPSpawnPIDError.refused(pid)
        }
        // Keep the closure referenced so callers can pass it. Do not call it.
        _ = terminate
    }
}

enum MCPSpawnPIDError: LocalizedError, Equatable {
    case refused(Int32)

    var errorDescription: String? {
        switch self {
        case .refused(let pid):
            return "wx-cli returned pid \(pid)"
        }
    }
}
