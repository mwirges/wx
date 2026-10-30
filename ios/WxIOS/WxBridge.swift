import Foundation
import WxCore

enum WxBridgeError: LocalizedError {
    case invalidArgs
    case nullResponse
    case cacheDirRejected

    var errorDescription: String? {
        switch self {
        case .invalidArgs:
            return "Could not encode the wx command."
        case .nullResponse:
            return "WxRunJSON returned nothing."
        case .cacheDirRejected:
            return "WxSetCacheDir rejected the Caches path."
        }
    }
}

/// In-process call into WxCore. The argument is a JSON array of strings,
/// for example ["forecast","New Haven, IN"]. The location stays one element.
enum WxBridge {
    private static var cacheDirPrepared = false

    /// Point the Go cache at FileManager Caches once before any WxRunJSON.
    /// The iOS sandbox rejects mkdir of HOME/.cache. Swift setenv alone is
    /// invisible to Go across the c-archive boundary, so this calls the
    /// exported WxSetCacheDir which runs os.Setenv on the Go side.
    static func prepareCacheDir() {
        if cacheDirPrepared { return }
        let caches = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first
            ?? URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true)
        let dir = caches.appendingPathComponent("wx", isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        // Keep libc setenv for any non-Go readers; Go must see WxSetCacheDir.
        setenv("WX_CACHE_DIR", dir.path, 1)
        let rc = dir.path.withCString { pointer in
            WxSetCacheDir(UnsafeMutablePointer(mutating: pointer))
        }
        if rc != 0 {
            // Do not crash startup; forecast will surface command_error if Go
            // still cannot write. The live-bridge UITest covers the happy path.
            return
        }
        cacheDirPrepared = true
    }

    static func run(args: [String]) async throws -> String {
        prepareCacheDir()
        let data = try JSONEncoder().encode(args)
        let json = String(decoding: data, as: UTF8.self)
        return try await Task.detached(priority: .userInitiated) {
            try call(json)
        }.value
    }

    private static func call(_ json: String) throws -> String {
        try json.withCString { pointer in
            guard let out = WxRunJSON(UnsafeMutablePointer(mutating: pointer)) else {
                throw WxBridgeError.nullResponse
            }
            defer { WxFreeString(out) }
            return String(cString: out)
        }
    }
}
