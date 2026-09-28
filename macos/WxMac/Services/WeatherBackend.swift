import Foundation

/// Single swappable weather backend boundary for the Mac (and future) shells.
/// Today: `WxCLIBackend` shells out to Go `wx --json`.
/// Later: c-shared / GTK host must expose the same JSON contract — not a second NWS client.
protocol WeatherBackend: Sendable {
    func fetch(location: String?, units: String?) async throws -> WxPayload
    func fetchRadar(location: String?, product: String?, radiusKm: Double?, raw: Bool, loop: Bool, frames: Int) async throws -> RadarPayload
    func fetchCPC(location: String?) async throws -> CPCPayloadDTO
    func fetchSPC(location: String?) async throws -> SPCPayloadDTO
    func fetchChase() async throws -> ChasePayloadDTO
    func fetchHistory(location: String?, days: Int, units: String?) async throws -> HistoryPayloadDTO
    func persistConfig(location: String?, units: String?) async throws
    var isAvailable: Bool { get }
}

extension WeatherBackend {
    func fetchRadar(location: String?, product: String?, radiusKm: Double?, raw: Bool = true, loop: Bool = true, frames: Int = 8) async throws -> RadarPayload {
        try await fetchRadar(location: location, product: product, radiusKm: radiusKm, raw: raw, loop: loop, frames: frames)
    }

    func fetchHistory(location: String?, days: Int = 14, units: String? = nil) async throws -> HistoryPayloadDTO {
        try await fetchHistory(location: location, days: days, units: units)
    }
}

/// PATH-based Go CLI adapter (data path A).
struct WxCLIBackend: WeatherBackend {
    var isAvailable: Bool { WxCLI.locateBinary() != nil }

    func fetch(location: String?, units: String?) async throws -> WxPayload {
        try await Task.detached(priority: .userInitiated) {
            try WxCLI.fetch(location: location, units: units)
        }.value
    }

    func fetchRadar(location: String?, product: String?, radiusKm: Double?, raw: Bool = true, loop: Bool = true, frames: Int = 8) async throws -> RadarPayload {
        try await Task.detached(priority: .userInitiated) {
            try WxCLI.fetchRadar(location: location, product: product, radiusKm: radiusKm, raw: raw, loop: loop, frames: frames)
        }.value
    }

    func fetchCPC(location: String?) async throws -> CPCPayloadDTO {
        try await Task.detached(priority: .userInitiated) {
            try WxCLI.fetchCPC(location: location)
        }.value
    }

    func fetchSPC(location: String?) async throws -> SPCPayloadDTO {
        try await Task.detached(priority: .userInitiated) {
            try WxCLI.fetchSPC(location: location)
        }.value
    }

    func fetchChase() async throws -> ChasePayloadDTO {
        try await Task.detached(priority: .userInitiated) {
            try WxCLI.fetchChase()
        }.value
    }

    func fetchHistory(location: String?, days: Int = 14, units: String? = nil) async throws -> HistoryPayloadDTO {
        try await Task.detached(priority: .userInitiated) {
            try WxCLI.fetchHistory(location: location, days: days, units: units)
        }.value
    }

    func persistConfig(location: String?, units: String?) async throws {
        try await Task.detached(priority: .userInitiated) {
            guard let binary = WxCLI.locateBinary() else { throw WxCLIError.binaryMissing }
            try WxConfig.persist(location: location, units: units, wxBinary: binary)
        }.value
    }
}
