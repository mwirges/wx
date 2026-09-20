import Foundation

/// Single swappable weather backend boundary for the Mac (and future) shells.
/// Today: `WxCLIBackend` shells out to Go `wx --json`.
/// Later: c-shared / GTK host must expose the same JSON contract — not a second NWS client.
protocol WeatherBackend: Sendable {
    func fetch(location: String?, units: String?) async throws -> WxPayload
    func persistConfig(location: String?, units: String?) async throws
    var isAvailable: Bool { get }
}

/// PATH-based Go CLI adapter (data path A).
struct WxCLIBackend: WeatherBackend {
    var isAvailable: Bool { WxCLI.locateBinary() != nil }

    func fetch(location: String?, units: String?) async throws -> WxPayload {
        try await Task.detached(priority: .userInitiated) {
            try WxCLI.fetch(location: location, units: units)
        }.value
    }

    func persistConfig(location: String?, units: String?) async throws {
        try await Task.detached(priority: .userInitiated) {
            guard let binary = WxCLI.locateBinary() else { throw WxCLIError.binaryMissing }
            try WxConfig.persist(location: location, units: units, wxBinary: binary)
        }.value
    }
}
