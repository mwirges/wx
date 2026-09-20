import Foundation
import Combine
import AppKit

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

    private let backend: WeatherBackend
    private var refreshTimer: Timer?
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
        refreshTimer?.invalidate()
        refreshTimer = Timer.scheduledTimer(withTimeInterval: refreshInterval, repeats: true) { [weak self] _ in
            Task { @MainActor in
                await self?.refresh()
            }
        }
    }

    func stop() {
        refreshTimer?.invalidate()
        refreshTimer = nil
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
}
