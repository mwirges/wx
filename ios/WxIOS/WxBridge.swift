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
    static func run(args: [String]) async throws -> String {
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
