import Foundation

/// Single swappable weather backend boundary for the Mac (and future) shells.
/// Today: `WxCLIBackend` shells out to Go `wx --json`.
/// Later: c-shared / GTK host must expose the same JSON contract — not a second NWS client.
protocol WeatherBackend: Sendable {
    func fetch(location: String?, units: String?, priority: TaskPriority, hourly: Bool) async throws -> WxPayload
    func fetchRadar(location: String?, product: String?, radiusKm: Double?, bbox: RadarBBox?, raw: Bool, loop: Bool, frames: Int, priority: TaskPriority) async throws -> RadarPayload
    func fetchCPC(location: String?) async throws -> CPCPayloadDTO
    func fetchSPC(location: String?) async throws -> SPCPayloadDTO
    func fetchChase() async throws -> ChasePayloadDTO
    func fetchHistory(location: String?, days: Int, units: String?) async throws -> HistoryPayloadDTO
    func fetchNowcast(location: String?, units: String?) async throws -> NowcastPayloadDTO
    func fetchClimate(location: String?, units: String?) async throws -> ClimatePayloadDTO
    func fetchSounding(location: String?, station: String?, units: String?) async throws -> SoundingPayloadDTO
    func fetchTropics(location: String?, storm: String?, units: String?) async throws -> TropicsPayloadDTO
    func persistConfig(location: String?, units: String?) async throws
    var isAvailable: Bool { get }
}

extension WeatherBackend {
    func fetch(location: String?, units: String?) async throws -> WxPayload {
        try await fetch(location: location, units: units, priority: .userInitiated, hourly: false)
    }

    func fetchRadar(location: String?, product: String?, radiusKm: Double?, bbox: RadarBBox? = nil, raw: Bool = true, loop: Bool = true, frames: Int = 8) async throws -> RadarPayload {
        try await fetchRadar(location: location, product: product, radiusKm: radiusKm, bbox: bbox, raw: raw, loop: loop, frames: frames, priority: .userInitiated)
    }

    func fetchHistory(location: String?, days: Int = 14, units: String? = nil) async throws -> HistoryPayloadDTO {
        try await fetchHistory(location: location, days: days, units: units)
    }

    func fetchSounding(location: String? = nil, station: String? = nil, units: String? = nil) async throws -> SoundingPayloadDTO {
        try await fetchSounding(location: location, station: station, units: units)
    }

    func fetchTropics(location: String? = nil, storm: String? = nil, units: String? = nil) async throws -> TropicsPayloadDTO {
        try await fetchTropics(location: location, storm: storm, units: units)
    }
}

/// PATH-based Go CLI adapter (data path A).
struct WxCLIBackend: WeatherBackend {
    var isAvailable: Bool { WxCLI.locateBinary() != nil }

    func fetch(location: String?, units: String?, priority: TaskPriority, hourly: Bool) async throws -> WxPayload {
        try await Task.detached(priority: priority) {
            try WxCLI.fetch(location: location, units: units, hourly: hourly)
        }.value
    }

    func fetchRadar(location: String?, product: String?, radiusKm: Double?, bbox: RadarBBox? = nil, raw: Bool = true, loop: Bool = true, frames: Int = 8, priority: TaskPriority = .userInitiated) async throws -> RadarPayload {
        try await Task.detached(priority: priority) {
            try WxCLI.fetchRadar(location: location, product: product, radiusKm: radiusKm, bbox: bbox, raw: raw, loop: loop, frames: frames)
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

    func fetchNowcast(location: String?, units: String?) async throws -> NowcastPayloadDTO {
        try await Task.detached(priority: .userInitiated) {
            try WxCLI.fetchNowcast(location: location, units: units)
        }.value
    }

    func fetchClimate(location: String?, units: String?) async throws -> ClimatePayloadDTO {
        try await Task.detached(priority: .userInitiated) {
            try WxCLI.fetchClimate(location: location, units: units)
        }.value
    }

    func fetchSounding(location: String?, station: String?, units: String?) async throws -> SoundingPayloadDTO {
        try await Task.detached(priority: .userInitiated) {
            try WxCLI.fetchSounding(location: location, station: station, units: units)
        }.value
    }

    func fetchTropics(location: String?, storm: String?, units: String?) async throws -> TropicsPayloadDTO {
        try await Task.detached(priority: .userInitiated) {
            try WxCLI.fetchTropics(location: location, storm: storm, units: units)
        }.value
    }

    func persistConfig(location: String?, units: String?) async throws {
        try await Task.detached(priority: .userInitiated) {
            guard let binary = WxCLI.locateBinary() else { throw WxCLIError.binaryMissing }
            try WxConfig.persist(location: location, units: units, wxBinary: binary)
        }.value
    }
}
