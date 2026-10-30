import Foundation
import WxCore

enum WxBridgeError: LocalizedError {
    case invalidArgs
    case nullResponse

    var errorDescription: String? {
        switch self {
        case .invalidArgs:
            return "Could not encode the wx command."
        case .nullResponse:
            return "WxRunJSON returned nothing."
        }
    }
}

/// In-process call into WxCore. The argument is a JSON array of strings,
/// for example ["forecast","New Haven, IN"]. The location stays one element.
enum WxBridge {
    /// Point the Go cache at FileManager Caches before any WxRunJSON.
    /// The iOS sandbox rejects mkdir of HOME/.cache. The widget snapshot
    /// cache is a different directory; this one is only the Go JSON cache.
    static func prepareCacheDir() {
        let caches = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first
            ?? URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true)
        let dir = caches.appendingPathComponent("wx", isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        setenv("WX_CACHE_DIR", dir.path, 1)
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
