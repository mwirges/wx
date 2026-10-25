import Combine
import Foundation

enum IOSTab: String, CaseIterable, Identifiable {
    case surface = "Surface"
    case tactical = "Tactical"
    case radar = "Radar"
    case severe = "Severe"
    case tropics = "Tropics"
    case config = "Config"

    var id: String { rawValue }

    var symbol: String {
        switch self {
        case .surface: return "sun.max"
        case .tactical: return "square.grid.2x2"
        case .radar: return "dot.radiowaves.left.and.right"
        case .severe: return "cloud.bolt"
        case .tropics: return "hurricane"
        case .config: return "gearshape"
        }
    }
}

@MainActor
final class IOSStore: ObservableObject {
    @Published var tab: IOSTab = .surface
    @Published var place: String = ""
    @Published var units: String = "imperial"
    @Published var payload: WxPayload?
    @Published var nowcast: NowcastPayloadDTO?
    @Published var radar: RadarPayload?
    @Published var spc: SPCPayloadDTO?
    @Published var tropics: TropicsPayloadDTO?
    @Published var isLoading = false
    @Published var radarLoading = false
    @Published var severeLoading = false
    @Published var tropicsLoading = false
    @Published var surfaceError: String?
    @Published var radarError: String?
    @Published var severeError: String?
    @Published var tropicsError: String?
    @Published var locationNote: String?
    @Published var lastRefreshed: Date?
    @Published private(set) var foreground = true

    let client: any IOSWeatherClient
    private let locator = IOSLocation()
    private var didBoot = false
    private let staleAfter: TimeInterval = 5 * 60

    init(client: (any IOSWeatherClient)? = nil) {
        self.client = client ?? IOSClients.make()
    }

    func boot() async {
        guard !didBoot else { return }
        didBoot = true
        seedFromEnvironment()
        let cfg = WxConfig.load()
        units = cfg.units == "metric" ? "metric" : "imperial"
        let saved = (cfg.defaultLocation ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let resolved = await resolvePlace(saved: saved)
        place = resolved
        guard !resolved.isEmpty else {
            surfaceError = "Set a place in Config."
            return
        }
        await refreshSurface()
    }

    func noteForeground(_ active: Bool) {
        foreground = active
        guard active, didBoot, let last = lastRefreshed else { return }
        guard Date().timeIntervalSince(last) > staleAfter else { return }
        Task { await refreshVisible() }
    }

    func ensure(_ tab: IOSTab) async {
        switch tab {
        case .surface, .tactical, .config:
            if payload == nil, !place.isEmpty {
                await refreshSurface()
            }
        case .radar:
            if radar == nil {
                await loadRadar()
            }
        case .severe:
            if spc == nil {
                await loadSevere()
            }
        case .tropics:
            if tropics == nil {
                await loadTropics()
            }
        }
    }

    func save(place: String, units: String) {
        let clean = place.trimmingCharacters(in: .whitespacesAndNewlines)
        let nextUnits = units == "metric" ? "metric" : "imperial"
        var cfg = WxConfig.load()
        cfg.defaultLocation = clean.isEmpty ? nil : clean
        cfg.units = nextUnits
        WxConfig.save(cfg)
        if !clean.isEmpty {
            WxConfig.addRecent(location: clean)
        }
        self.place = clean
        self.units = nextUnits
        locationNote = nil
        invalidate()
        Task { await refreshVisible() }
    }

    func useCurrentLocation() async {
        do {
            let found = try await locator.oneShot()
            locationNote = nil
            save(place: found, units: units)
        } catch {
            let saved = place.trimmingCharacters(in: .whitespacesAndNewlines)
            if saved.isEmpty {
                locationNote = "Location denied. Set a place in Config."
                surfaceError = locationNote
            } else {
                locationNote = "Location denied. Using saved place."
            }
        }
    }

    func refreshSurface() async {
        let whereTo = place.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !whereTo.isEmpty else {
            surfaceError = "Set a place in Config."
            return
        }
        isLoading = true
        surfaceError = nil
        let forecastArgs = [
            "forecast", "--alerts", "--hourly", "--hours", "24",
            "--units", units, "--location", whereTo,
        ]
        let nowcastArgs = ["nowcast", "--units", units, "--location", whereTo]
        async let forecastRaw = capture(forecastArgs)
        async let nowcastRaw = capture(nowcastArgs)
        let forecastResult = await forecastRaw
        let nowcastResult = await nowcastRaw
        isLoading = false

        switch forecastResult {
        case .success(let raw):
            do {
                payload = try BridgeJSON.decode(WxPayload.self, from: raw)
                lastRefreshed = Date()
            } catch {
                surfaceError = error.localizedDescription
            }
        case .failure(let error):
            surfaceError = error.localizedDescription
        }

        switch nowcastResult {
        case .success(let raw):
            do {
                nowcast = try BridgeJSON.decode(NowcastPayloadDTO.self, from: raw)
            } catch {
                if surfaceError == nil {
                    surfaceError = error.localizedDescription
                }
            }
        case .failure(let error):
            if surfaceError == nil {
                surfaceError = error.localizedDescription
            }
        }
    }

    func loadRadar() async {
        await loadExtra(
            args: ["radar", "--raw", "--location", place, "--product", "composite-reflectivity"],
            loading: \.radarLoading,
            error: \.radarError
        ) { raw in
            self.radar = try BridgeJSON.decode(RadarPayload.self, from: raw)
        }
    }

    func loadSevere() async {
        await loadExtra(
            args: ["spc", "--location", place],
            loading: \.severeLoading,
            error: \.severeError
        ) { raw in
            self.spc = try BridgeJSON.decode(SPCPayloadDTO.self, from: raw)
        }
    }

    func loadTropics() async {
        await loadExtra(
            args: ["tropics", "--location", place],
            loading: \.tropicsLoading,
            error: \.tropicsError
        ) { raw in
            self.tropics = try BridgeJSON.decode(TropicsPayloadDTO.self, from: raw)
        }
    }

    var nowcastHeadline: String? {
        if let headline = nowcast?.nowcast?.headline, !headline.isEmpty {
            return headline
        }
        if let headline = payload?.conditions?.nowcast?.headline, !headline.isEmpty {
            return headline
        }
        return payload?.nowcast?.headline
    }

    var refreshedLabel: String {
        guard let lastRefreshed else { return "SYNC" }
        let formatter = DateFormatter()
        formatter.timeStyle = .short
        return formatter.string(from: lastRefreshed)
    }

    private func refreshVisible() async {
        guard !place.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            surfaceError = "Set a place in Config."
            return
        }
        await refreshSurface()
        switch tab {
        case .radar:
            await loadRadar()
        case .severe:
            await loadSevere()
        case .tropics:
            await loadTropics()
        case .surface, .tactical, .config:
            break
        }
    }

    private func invalidate() {
        payload = nil
        nowcast = nil
        radar = nil
        spc = nil
        tropics = nil
        lastRefreshed = nil
    }

    private func resolvePlace(saved: String) async -> String {
        let env = ProcessInfo.processInfo.environment
        if env["WX_SKIP_LOCATION"] == "1" || env["WX_UI_TEST"] == "1" {
            return saved
        }
        if env["WX_LOCATION_MODE"] == "denied" {
            return denied(saved)
        }
        do {
            let found = try await locator.oneShot()
            locationNote = nil
            return found
        } catch {
            return denied(saved)
        }
    }

    private func denied(_ saved: String) -> String {
        if saved.isEmpty {
            locationNote = "Location denied. Set a place in Config."
            return ""
        }
        locationNote = "Location denied. Using saved place."
        return saved
    }

    private func seedFromEnvironment() {
        let env = ProcessInfo.processInfo.environment
        guard let raw = env["WX_DEFAULT_LOCATION"]?.trimmingCharacters(in: .whitespacesAndNewlines),
              !raw.isEmpty else {
            return
        }
        var cfg = WxConfig.load()
        cfg.defaultLocation = raw
        if let units = env["WX_UNITS"], units == "metric" || units == "imperial" {
            cfg.units = units
        }
        WxConfig.save(cfg)
    }

    private func capture(_ args: [String]) async -> Result<String, Error> {
        do {
            return .success(try await client.run(args: args))
        } catch {
            return .failure(error)
        }
    }

    private func loadExtra(
        args: [String],
        loading: ReferenceWritableKeyPath<IOSStore, Bool>,
        error: ReferenceWritableKeyPath<IOSStore, String?>,
        apply: (String) throws -> Void
    ) async {
        let whereTo = place.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !whereTo.isEmpty else { return }
        self[keyPath: loading] = true
        self[keyPath: error] = nil
        let result = await capture(args)
        self[keyPath: loading] = false
        switch result {
        case .success(let raw):
            do {
                try apply(raw)
            } catch let failure {
                self[keyPath: error] = failure.localizedDescription
            }
        case .failure(let failure):
            self[keyPath: error] = failure.localizedDescription
        }
    }
}

enum IOSFormat {
    static func temperature(f: Double?, c: Double?, units: String) -> String {
        if units == "metric", let c {
            return String(format: "%.0f°", c)
        }
        if let f {
            return String(format: "%.0f°", f)
        }
        if let c {
            return String(format: "%.0f°", c)
        }
        return "—"
    }

    static func observation(_ conditions: Conditions?, units: String) -> String {
        temperature(f: conditions?.temperatureF, c: conditions?.temperatureC, units: units)
    }

    static func period(_ period: Period, units: String) -> String {
        temperature(f: period.temperatureF, c: period.temperatureC, units: units)
    }

    static func hourLabel(_ period: Period) -> String {
        guard let raw = period.startTime, !raw.isEmpty else { return period.name }
        let iso = ISO8601DateFormatter()
        iso.formatOptions = [.withInternetDateTime]
        guard let date = iso.date(from: raw) else { return period.name }
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = .current
        formatter.dateFormat = "h a"
        return formatter.string(from: date)
    }
}
