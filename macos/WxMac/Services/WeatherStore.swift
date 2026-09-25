import Foundation
import Combine
import AppKit

enum DeskTab: String, CaseIterable, Identifiable {
    case weather = "Weather"
    case radar = "Radar"

    var id: String { rawValue }
}

/// Presentation store. Fetches only through `WeatherBackend` (default: WxCLI).
/// Views stay dumb — no provider/cache/NWS policy here.
@MainActor
final class WeatherStore: ObservableObject {
    @Published var payload: WxPayload?
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var locationInput: String = ""
    @Published var units: String = "imperial"
    @Published var lastRefreshed: Date?
    @Published var deskWindowOpen = false
    @Published var selectedDeskTab: DeskTab = .weather

    @Published var radarPayload: RadarPayload?
    @Published var radarImage: NSImage?
    @Published var isRadarLoading = false
    @Published var radarErrorMessage: String?
    @Published var selectedRadarProduct: String = "composite-reflectivity"
    @Published var selectedRadarRadius: Double = 200

    private let backend: WeatherBackend
    private var refreshTask: Task<Void, Never>?
    private let refreshInterval: TimeInterval = 5 * 60

    init(backend: WeatherBackend = WxCLIBackend()) {
        self.backend = backend
        let cfg = WxConfig.load()
        locationInput = cfg.defaultLocation ?? ""
        let u = (cfg.units ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        units = (u == "metric") ? "metric" : "imperial"
    }

    func start() {
        Task { await refresh() }
        refreshTask?.cancel()
        let interval = refreshInterval
        refreshTask = Task { [weak self] in
            while !Task.isCancelled {
                do {
                    try await Task.sleep(nanoseconds: UInt64(interval * 1_000_000_000))
                } catch {
                    return
                }
                guard !Task.isCancelled else { return }
                await self?.refresh()
            }
        }
    }

    func stop() {
        refreshTask?.cancel()
        refreshTask = nil
    }

    func applyLocationAndUnits() async {
        do {
            guard backend.isAvailable else {
                errorMessage = WxCLIError.binaryMissing.errorDescription
                return
            }
            try await backend.persistConfig(
                location: locationInput.trimmingCharacters(in: .whitespacesAndNewlines),
                units: units
            )
        } catch {
            errorMessage = error.localizedDescription
        }
        await refresh()
        if radarPayload != nil || selectedDeskTab == .radar {
            await refreshRadar()
        }
    }

    func refreshRadar() async {
        isRadarLoading = true
        radarErrorMessage = nil
        defer { isRadarLoading = false }

        guard backend.isAvailable else {
            radarErrorMessage = WxCLIError.binaryMissing.errorDescription
            return
        }

        let loc = locationInput.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            let res = try await backend.fetchRadar(
                location: loc.isEmpty ? nil : loc,
                product: selectedRadarProduct,
                radiusKm: selectedRadarRadius,
                raw: true
            )
            radarPayload = res
            if let data = Data(base64Encoded: res.imageBase64),
               let img = NSImage(data: data) {
                radarImage = img
            } else {
                radarImage = nil
                radarErrorMessage = "Failed to decode radar image"
            }
        } catch {
            radarErrorMessage = error.localizedDescription
        }
    }

    func refresh() async {
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }

        guard backend.isAvailable else {
            errorMessage = WxCLIError.binaryMissing.errorDescription
            updateStatusItemChrome()
            return
        }

        let loc = locationInput.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            let result = try await backend.fetch(
                location: loc.isEmpty ? nil : loc,
                units: units
            )
            payload = result
            lastRefreshed = Date()
            updateStatusItemChrome()
        } catch {
            errorMessage = error.localizedDescription
            updateStatusItemChrome()
        }
    }

    var displayTemp: String {
        guard let c = payload?.conditions else { return "--" }
        if units == "metric" {
            if let t = c.temperatureC { return String(format: "%.0f°", t) }
        } else {
            if let t = c.temperatureF { return String(format: "%.0f°", t) }
        }
        return "--"
    }

    var statusSymbol: String {
        ConditionSymbol.systemName(for: payload?.conditions?.conditionCode)
    }

    private func updateStatusItemChrome() {
        NotificationCenter.default.post(name: .wxWeatherDidUpdate, object: nil)
    }
}

extension Notification.Name {
    static let wxWeatherDidUpdate = Notification.Name("wxWeatherDidUpdate")
    static let wxOpenDeskWindow = Notification.Name("wxOpenDeskWindow")
    static let wxOpenDeskRadar = Notification.Name("wxOpenDeskRadar")
}
