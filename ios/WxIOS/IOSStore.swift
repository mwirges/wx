import Combine
import Foundation
import UIKit

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

struct IOSRadarPicture: Identifiable {
    let id: Int
    let image: CGImage
    let validTime: String
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
    @Published private(set) var radarPictures: [IOSRadarPicture] = []
    @Published private(set) var loopFrame = 0
    @Published private(set) var loopTicks = 0
    @Published private(set) var loopPlaying = false
    @Published private(set) var ticksAtPause = 0
    @Published private(set) var ticksWhilePaused = 0
    @Published private(set) var pauseCount = 0
    @Published private(set) var refreshCount = 0
    @Published private(set) var liveStatus = "pending"
    @Published private(set) var liveActivityID = "none"

    let showsPowerProbe: Bool
    let showsLiveProbe: Bool
    let client: any IOSWeatherClient
    private let clock: any IOSClock
    private let locator = IOSLocation()
    private var didBoot = false
    private let staleAfter: TimeInterval = 5 * 60
    private let loopIntervalMs: Int
    private var loopTask: Task<Void, Never>?
    private var refreshTask: Task<Void, Never>?
    private var didConsiderActivity = false

    init(client: (any IOSWeatherClient)? = nil, clock: (any IOSClock)? = nil) {
        let resolved = clock ?? IOSClocks.make()
        self.clock = resolved
        self.showsPowerProbe = resolved is AdjustableClock
        self.showsLiveProbe = ProcessInfo.processInfo.environment["WX_LIVE_ACTIVITY_TEST"] == "1"
        self.client = client ?? IOSClients.make()
        self.loopIntervalMs = IOSLoopTiming.intervalMilliseconds()
    }

    var currentRadarImage: CGImage? {
        guard radarPictures.indices.contains(loopFrame) else { return nil }
        return radarPictures[loopFrame].image
    }

    func advanceTestClock(by seconds: TimeInterval = 6 * 60) {
        guard let adjustable = clock as? AdjustableClock else { return }
        adjustable.advance(by: seconds)
    }

    func injectLiveOnset() async {
        await injectLive(onsetMinutes: 20)
    }

    func injectLiveDry() async {
        await injectLive(onsetMinutes: nil)
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
        if active == foreground {
            syncRadarLoop()
            if active {
                considerStaleRefresh()
            }
            return
        }
        if !active {
            foreground = false
            syncRadarLoop()
            return
        }
        if loopTicks > ticksAtPause {
            ticksWhilePaused += loopTicks - ticksAtPause
        }
        ticksAtPause = loopTicks
        foreground = true
        syncRadarLoop()
        considerStaleRefresh()
    }

    func syncRadarLoop() {
        if foreground && tab == .radar && radarPictures.count > 1 {
            startRadarLoop()
        } else {
            stopRadarLoop()
        }
    }

    func ensure(_ tab: IOSTab) async {
        syncRadarLoop()
        switch tab {
        case .surface, .tactical, .config:
            if payload == nil, !place.isEmpty, !isLoading {
                await refreshSurface()
            }
        case .radar:
            if radar == nil, !radarLoading {
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
        radarLoading = true
        surfaceError = nil
        radarError = nil
        refreshCount += 1
        let client = self.client
        let units = self.units
        // One radio wake: conditions (the forecast document), nowcast, alerts,
        // and radar all start before any of them is awaited.
        async let forecastRaw = fetchIOSWeather(client, args: [
            "forecast", "--hourly", "--hours", "24",
            "--units", units, "--location", whereTo,
        ])
        async let nowcastRaw = fetchIOSWeather(client, args: [
            "nowcast", "--units", units, "--location", whereTo,
        ])
        async let alertsRaw = fetchIOSWeather(client, args: [
            "alerts", "--units", units, "--location", whereTo,
        ])
        async let radarRaw = fetchIOSWeather(client, args: [
            "radar", "--raw", "--loop", "--frames", "6",
            "--location", whereTo, "--product", "composite-reflectivity",
        ])
        let forecastResult = await forecastRaw
        let nowcastResult = await nowcastRaw
        let alertsResult = await alertsRaw
        let radarResult = await radarRaw

        switch forecastResult {
        case .success(let raw):
            do {
                payload = try BridgeJSON.decode(WxPayload.self, from: raw)
                lastRefreshed = clock.now()
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

        switch alertsResult {
        case .success(let raw):
            if let alertsPayload = try? BridgeJSON.decode(WxPayload.self, from: raw) {
                if payload == nil {
                    payload = alertsPayload
                } else {
                    payload?.alerts = alertsPayload.alerts
                }
            }
        case .failure:
            break
        }

        switch radarResult {
        case .success(let raw):
            do {
                try applyRadar(raw)
            } catch {
                radarError = error.localizedDescription
            }
        case .failure(let error):
            radarError = error.localizedDescription
        }

        isLoading = false
        radarLoading = false
        syncRadarLoop()
        await publishSharedCache()
    }

    func loadRadar() async {
        await loadExtra(
            args: ["radar", "--raw", "--loop", "--frames", "6", "--location", place, "--product", "composite-reflectivity"],
            loading: \.radarLoading,
            error: \.radarError
        ) { raw in
            try self.applyRadar(raw)
        }
        syncRadarLoop()
        await publishSharedCache()
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

    private func publishSharedCache() async {
        let snapshot = WidgetPublish.snapshot(
            place: place,
            units: units,
            payload: payload,
            nowcast: nowcast,
            now: clock.now()
        )
        var cacheError: String?
        do {
            try WidgetPublish.store(snapshot, radarBase64: radar?.imageBase64)
        } catch {
            cacheError = error.localizedDescription
        }
        let shouldSync = snapshot.cue.shouldRun || showsLiveProbe || !didConsiderActivity
        didConsiderActivity = true
        guard shouldSync else {
            if let cacheError, showsLiveProbe {
                liveStatus = "failed: cache \(cacheError)"
            }
            return
        }
        let result = await NowcastLiveActivitySync.sync(cue: snapshot.cue)
        liveActivityID = result.id
        if let cacheError {
            liveStatus = "failed: cache \(cacheError)"
        } else {
            liveStatus = result.status
        }
    }

    private func injectLive(onsetMinutes: Double?) async {
        let now = clock.now()
        let next = onsetMinutes.map { now.addingTimeInterval($0 * 60) }
        let sample = NowcastWindowSample(
            now: now,
            isActivePrecip: false,
            nextPrecipTime: next,
            intervals: []
        )
        let headline = onsetMinutes.map { "Rain starting in ~\(Int($0))m" } ?? "Dry next 6h"
        let cue = LiveActivityCue.make(
            sample: sample,
            location: place.isEmpty ? "New Haven, IN" : place,
            headline: headline,
            phase: onsetMinutes == nil ? "none" : "rain",
            sparkline: onsetMinutes == nil ? "" : " ▂▄▆",
            temperatureText: IOSFormat.observation(payload?.conditions, units: units),
            detail: headline,
            cessationText: ""
        )
        let result = await NowcastLiveActivitySync.sync(cue: cue)
        liveStatus = result.status
        liveActivityID = result.id
    }

    private func considerStaleRefresh() {
        guard refreshTask == nil else { return }
        guard didBoot, let last = lastRefreshed else { return }
        guard clock.now().timeIntervalSince(last) > staleAfter else { return }
        refreshTask = Task { @MainActor [weak self] in
            await self?.refreshVisible()
            self?.refreshTask = nil
        }
    }

    private func refreshVisible() async {
        guard !place.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            surfaceError = "Set a place in Config."
            return
        }
        await refreshSurface()
        switch tab {
        case .severe:
            await loadSevere()
        case .tropics:
            await loadTropics()
        case .surface, .tactical, .config, .radar:
            break
        }
    }

    private func invalidate() {
        payload = nil
        nowcast = nil
        radar = nil
        radarPictures = []
        loopFrame = 0
        spc = nil
        tropics = nil
        lastRefreshed = nil
        stopRadarLoop()
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

    private func applyRadar(_ raw: String) throws {
        let decoded = try BridgeJSON.decode(RadarPayload.self, from: raw)
        radar = decoded
        radarPictures = pictures(from: decoded)
        if radarPictures.isEmpty || loopFrame >= radarPictures.count {
            loopFrame = 0
        }
    }

    private func pictures(from radar: RadarPayload) -> [IOSRadarPicture] {
        var sources: [(String, String)] = radar.frames.map { frame in
            (frame.validTime ?? radar.validTime, frame.imageBase64)
        }
        if sources.isEmpty, !radar.imageBase64.isEmpty {
            sources = [(radar.validTime, radar.imageBase64)]
        }
        var pictures: [IOSRadarPicture] = []
        pictures.reserveCapacity(sources.count)
        for (index, source) in sources.enumerated() {
            guard let data = Data(base64Encoded: source.1, options: .ignoreUnknownCharacters),
                  let image = UIImage(data: data) else {
                continue
            }
            let decoded = image.preparingForDisplay() ?? image
            guard let cgImage = decoded.cgImage else { continue }
            pictures.append(IOSRadarPicture(id: index, image: cgImage, validTime: source.0))
        }
        return pictures
    }

    private func startRadarLoop() {
        guard radarPictures.count > 1 else {
            stopRadarLoop()
            return
        }
        guard loopTask == nil else {
            loopPlaying = true
            return
        }
        loopPlaying = true
        let intervalNs = UInt64(loopIntervalMs) * 1_000_000
        loopTask = Task { @MainActor [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: intervalNs)
                guard let self, !Task.isCancelled else { return }
                guard self.foreground, self.tab == .radar, self.radarPictures.count > 1 else {
                    self.stopRadarLoop()
                    return
                }
                self.loopFrame = (self.loopFrame + 1) % self.radarPictures.count
                self.loopTicks += 1
            }
        }
    }

    private func stopRadarLoop() {
        let wasPlaying = loopTask != nil || loopPlaying
        if wasPlaying {
            ticksAtPause = loopTicks
            pauseCount += 1
        }
        loopPlaying = false
        loopTask?.cancel()
        loopTask = nil
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
        let result = await fetchIOSWeather(client, args: args)
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
