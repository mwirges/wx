import Foundation
import Combine
import AppKit
import UserNotifications

struct LocationGridCardData: Identifiable, Sendable {
    var id: String { locationKey }
    var locationKey: String
    var displayName: String
    var payload: WxPayload?
    var isLoading: Bool
    var errorMessage: String?
    var lastUpdated: Date?
}

struct DecodedRadarFrame: Identifiable, Sendable {
    let id: Int
    let validTime: String
    let date: Date?
    let image: NSImage
    let label: String
    let isLive: Bool
}

enum DeskTab: String, CaseIterable, Identifiable {
    case dual = "Tactical"
    case weather = "Weather"
    case radar = "Radar"
    case outlooks = "Outlooks"
    case chase = "Storm Chase"
    case climate = "Climate"
    case tropics = "Tropics"
    case grid = "Grid"

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
    @Published var menuBarFormat: MenuBarFormat = .standard
    @Published var lastRefreshed: Date?
    @Published var deskWindowOpen = false
    @Published var selectedDeskTab: DeskTab = .dual

    @Published var radarPayload: RadarPayload?
    @Published var radarImage: NSImage?
    @Published var radarFrames: [DecodedRadarFrame] = []
    @Published var activeFrameIndex: Int = 0
    @Published var isLoopPlaying: Bool = true
    @Published var loopStepMs: Int = 380
    @Published var loopDwellMs: Int = 1100
    @Published var isRadarLoading = false
    @Published var radarErrorMessage: String?
    @Published var selectedRadarProduct: String = "composite-reflectivity"
    @Published var selectedRadarRadius: Double = 200

    @Published var cpcPayload: CPCPayloadDTO?
    @Published var isCPCLoading = false
    @Published var cpcErrorMessage: String?

    @Published var chasePayload: ChasePayloadDTO?
    @Published var isChaseLoading = false
    @Published var chaseErrorMessage: String?

    @Published var spcPayload: SPCPayloadDTO?
    @Published var isSPCLoading = false
    @Published var spcErrorMessage: String?

    @Published var historyPayload: HistoryPayloadDTO?
    @Published var isHistoryLoading = false
    @Published var historyErrorMessage: String?
    @Published var historyDaysCount: Int = 14

    @Published var nowcastPayload: NowcastPayloadDTO?
    @Published var isNowcastLoading = false
    @Published var nowcastErrorMessage: String?

    @Published var climatePayload: ClimatePayloadDTO?
    @Published var isClimateLoading = false
    @Published var climateErrorMessage: String?

    @Published var soundingPayload: SoundingPayloadDTO?
    @Published var isSoundingLoading = false
    @Published var soundingErrorMessage: String?

    @Published var tropicsPayload: TropicsPayloadDTO?
    @Published var isTropicsLoading = false
    @Published var tropicsErrorMessage: String?

    @Published var favorites: [WxLocationEntry] = []
    @Published var recentLocations: [String] = []
    @Published var gridCards: [LocationGridCardData] = []
    @Published var isGridLoading = false
    @Published var isLocating = false

    private var notifiedAlertIDs = Set<String>()
    private var loopTask: Task<Void, Never>?

    let locationManager = LocationManager()

    private let backend: WeatherBackend
    private var refreshTask: Task<Void, Never>?
    private let refreshInterval: TimeInterval = 5 * 60

    init(backend: WeatherBackend = WxCLIBackend()) {
        self.backend = backend
        let cfg = WxConfig.load()
        locationInput = cfg.defaultLocation ?? ""
        let u = (cfg.units ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        units = (u == "metric") ? "metric" : "imperial"
        if let fmt = cfg.menuBarFormat, let parsed = MenuBarFormat(rawValue: fmt.lowercased()) {
            menuBarFormat = parsed
        } else {
            menuBarFormat = .standard
        }
        favorites = cfg.favorites ?? []
        recentLocations = cfg.recentLocations ?? []
    }

    func reloadConfig() {
        let cfg = WxConfig.load()
        favorites = cfg.favorites ?? []
        recentLocations = cfg.recentLocations ?? []
    }

    func isFavorite(_ loc: String) -> Bool {
        let clean = loc.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !clean.isEmpty else { return false }
        return favorites.contains {
            $0.name.caseInsensitiveCompare(clean) == .orderedSame ||
            $0.value.caseInsensitiveCompare(clean) == .orderedSame
        }
    }

    func toggleFavorite(_ loc: String) {
        let clean = loc.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !clean.isEmpty else { return }
        if isFavorite(clean) {
            WxConfig.removeFavorite(nameOrValue: clean)
        } else {
            let displayName = payload?.conditions?.location ?? clean
            WxConfig.addFavorite(name: displayName, value: clean)
        }
        reloadConfig()
    }

    func addFavorite(name: String, value: String) {
        WxConfig.addFavorite(name: name, value: value)
        reloadConfig()
    }

    func removeFavorite(_ nameOrValue: String) {
        WxConfig.removeFavorite(nameOrValue: nameOrValue)
        reloadConfig()
    }

    func clearRecents() {
        WxConfig.clearRecents()
        reloadConfig()
    }

    func selectLocation(_ loc: String) {
        let clean = loc.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !clean.isEmpty else { return }
        locationInput = clean
        WxConfig.addRecent(location: clean)
        reloadConfig()
        Task {
            await applyLocationAndUnits()
        }
    }

    func useCurrentLocation() {
        isLocating = true
        errorMessage = nil
        locationManager.requestLocation { [weak self] result in
            guard let self else { return }
            self.isLocating = false
            switch result {
            case .success(let loc):
                self.selectLocation(loc)
            case .failure(let err):
                self.errorMessage = err.localizedDescription
            }
        }
    }

    func start() {
        Task {
            await refresh()
            await refreshCPC()
            await refreshChase()
            if !favorites.isEmpty {
                await refreshGrid()
            }
            if selectedDeskTab == .dual || selectedDeskTab == .radar {
                await refreshRadar()
            }
        }
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
                await self?.refreshCPC()
                await self?.refreshChase()
                if let favs = self?.favorites, !favs.isEmpty {
                    await self?.refreshGrid()
                }
            }
        }
    }

    func stop() {
        refreshTask?.cancel()
        refreshTask = nil
        stopLoopTimer()
    }

    func applyLocationAndUnits() async {
        do {
            guard backend.isAvailable else {
                errorMessage = WxCLIError.binaryMissing.errorDescription
                return
            }
            let clean = locationInput.trimmingCharacters(in: .whitespacesAndNewlines)
            if !clean.isEmpty {
                WxConfig.addRecent(location: clean)
                reloadConfig()
            }
            try await backend.persistConfig(
                location: clean,
                units: units
            )
        } catch {
            errorMessage = error.localizedDescription
        }
        await refresh()
        await refreshCPC()
        if radarPayload != nil || selectedDeskTab == .radar || selectedDeskTab == .dual {
            await refreshRadar()
        }
    }

    func parseRadarDate(_ str: String) -> Date? {
        let f1 = ISO8601DateFormatter()
        f1.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = f1.date(from: str) { return d }
        let f2 = ISO8601DateFormatter()
        f2.formatOptions = [.withInternetDateTime]
        return f2.date(from: str)
    }

    func startLoopTimer() {
        loopTask?.cancel()
        guard isLoopPlaying, radarFrames.count > 1 else { return }
        loopTask = Task { @MainActor [weak self] in
            while !Task.isCancelled {
                guard let self = self, self.isLoopPlaying, self.radarFrames.count > 1 else { break }
                let isLast = (self.activeFrameIndex == self.radarFrames.count - 1)
                let dwell = isLast ? self.loopDwellMs : self.loopStepMs
                try? await Task.sleep(nanoseconds: UInt64(dwell) * 1_000_000)
                guard !Task.isCancelled, self.isLoopPlaying, self.radarFrames.count > 1 else { break }
                let nextIdx = (self.activeFrameIndex + 1) % self.radarFrames.count
                self.activeFrameIndex = nextIdx
                self.radarImage = self.radarFrames[nextIdx].image
            }
        }
    }

    func stopLoopTimer() {
        loopTask?.cancel()
        loopTask = nil
    }

    func toggleLoop() {
        isLoopPlaying.toggle()
        if isLoopPlaying {
            startLoopTimer()
        } else {
            stopLoopTimer()
        }
    }

    func stepFrameForward() {
        stopLoopTimer()
        isLoopPlaying = false
        guard !radarFrames.isEmpty else { return }
        activeFrameIndex = (activeFrameIndex + 1) % radarFrames.count
        radarImage = radarFrames[activeFrameIndex].image
    }

    func stepFrameBackward() {
        stopLoopTimer()
        isLoopPlaying = false
        guard !radarFrames.isEmpty else { return }
        activeFrameIndex = (activeFrameIndex - 1 + radarFrames.count) % radarFrames.count
        radarImage = radarFrames[activeFrameIndex].image
    }

    func seekFrame(to index: Int) {
        stopLoopTimer()
        isLoopPlaying = false
        guard !radarFrames.isEmpty else { return }
        let clamped = max(0, min(radarFrames.count - 1, index))
        activeFrameIndex = clamped
        radarImage = radarFrames[clamped].image
    }

    func jumpToLive() {
        guard !radarFrames.isEmpty else { return }
        seekFrame(to: radarFrames.count - 1)
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
                raw: true,
                loop: true,
                frames: 8
            )
            radarPayload = res

            var decoded: [DecodedRadarFrame] = []
            if !res.frames.isEmpty {
                for (idx, f) in res.frames.enumerated() {
                    guard let data = Data(base64Encoded: f.imageBase64),
                          let img = NSImage(data: data) else { continue }
                    let valid = f.validTime ?? res.validTime
                    let d = parseRadarDate(valid)
                    let isLive = (idx == res.frames.count - 1)
                    let label: String
                    if let date = d {
                        let mins = Int(round(Date().timeIntervalSince(date) / 60.0))
                        label = (isLive || mins <= 2) ? "LIVE" : "-\(mins)m"
                    } else {
                        label = isLive ? "LIVE" : "F\(idx+1)"
                    }
                    decoded.append(DecodedRadarFrame(
                        id: idx,
                        validTime: valid,
                        date: d,
                        image: img,
                        label: label,
                        isLive: isLive
                    ))
                }
            } else if let data = Data(base64Encoded: res.imageBase64),
                      let img = NSImage(data: data) {
                decoded = [DecodedRadarFrame(
                    id: 0,
                    validTime: res.validTime,
                    date: parseRadarDate(res.validTime),
                    image: img,
                    label: "LIVE",
                    isLive: true
                )]
            }

            if !decoded.isEmpty {
                radarFrames = decoded
                activeFrameIndex = decoded.count - 1
                radarImage = decoded.last?.image
                if isLoopPlaying && decoded.count > 1 {
                    startLoopTimer()
                }
            } else {
                radarFrames = []
                radarImage = nil
                radarErrorMessage = "Failed to decode radar frame telemetry"
            }
        } catch {
            radarErrorMessage = error.localizedDescription
        }
    }

    func refreshCPC() async {
        isCPCLoading = true
        cpcErrorMessage = nil
        defer { isCPCLoading = false }

        guard backend.isAvailable else {
            cpcErrorMessage = WxCLIError.binaryMissing.errorDescription
            return
        }

        let loc = locationInput.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            let res = try await backend.fetchCPC(location: loc.isEmpty ? nil : loc)
            cpcPayload = res
        } catch {
            cpcErrorMessage = error.localizedDescription
        }
    }

    func refreshChase() async {
        isChaseLoading = true
        chaseErrorMessage = nil
        defer { isChaseLoading = false }

        guard backend.isAvailable else {
            chaseErrorMessage = WxCLIError.binaryMissing.errorDescription
            return
        }

        do {
            let res = try await backend.fetchChase()
            chasePayload = res
            if let spc = res.spc {
                spcPayload = spc
            }
        } catch {
            chaseErrorMessage = error.localizedDescription
        }
    }

    func refreshSPC() async {
        isSPCLoading = true
        spcErrorMessage = nil
        defer { isSPCLoading = false }

        guard backend.isAvailable else {
            spcErrorMessage = WxCLIError.binaryMissing.errorDescription
            return
        }

        let loc = locationInput.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            let res = try await backend.fetchSPC(location: loc.isEmpty ? nil : loc)
            spcPayload = res
        } catch {
            spcErrorMessage = error.localizedDescription
        }
    }

    func refreshNowcast() async {
        isNowcastLoading = true
        nowcastErrorMessage = nil
        defer { isNowcastLoading = false }

        guard backend.isAvailable else {
            nowcastErrorMessage = WxCLIError.binaryMissing.errorDescription
            return
        }

        let loc = locationInput.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            let res = try await backend.fetchNowcast(location: loc.isEmpty ? nil : loc, units: units)
            nowcastPayload = res
        } catch {
            nowcastErrorMessage = error.localizedDescription
        }
    }

    func chaseCluster(_ cluster: StormClusterDTO) {
        let coord = String(format: "%.4f,%.4f", cluster.centerLat, cluster.centerLon)
        selectLocation(coord)
        selectedDeskTab = .radar
        selectedRadarRadius = 250
        Task {
            await refreshRadar()
        }
    }

    func setRadarRadius(_ radius: Double) {
        guard selectedRadarRadius != radius else { return }
        selectedRadarRadius = radius
        Task {
            await refreshRadar()
        }
    }

    func refreshHistory() async {
        isHistoryLoading = true
        historyErrorMessage = nil
        defer { isHistoryLoading = false }

        guard backend.isAvailable else {
            historyErrorMessage = WxCLIError.binaryMissing.errorDescription
            return
        }

        let loc = locationInput.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            let res = try await backend.fetchHistory(
                location: loc.isEmpty ? nil : loc,
                days: historyDaysCount,
                units: units
            )
            historyPayload = res
        } catch {
            historyErrorMessage = error.localizedDescription
        }
    }

    func refreshClimate() async {
        isClimateLoading = true
        climateErrorMessage = nil
        defer { isClimateLoading = false }

        guard backend.isAvailable else {
            climateErrorMessage = WxCLIError.binaryMissing.errorDescription
            return
        }

        let loc = locationInput.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            let res = try await backend.fetchClimate(
                location: loc.isEmpty ? nil : loc,
                units: units
            )
            climatePayload = res
        } catch {
            climateErrorMessage = error.localizedDescription
        }
    }

    func refreshSounding(station: String? = nil) async {
        isSoundingLoading = true
        soundingErrorMessage = nil
        defer { isSoundingLoading = false }

        guard backend.isAvailable else {
            soundingErrorMessage = WxCLIError.binaryMissing.errorDescription
            return
        }

        let loc = locationInput.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            let res = try await backend.fetchSounding(
                location: loc.isEmpty ? nil : loc,
                station: station,
                units: units
            )
            soundingPayload = res
        } catch {
            soundingErrorMessage = error.localizedDescription
        }
    }

    func refreshTropics(storm: String? = nil) async {
        isTropicsLoading = true
        tropicsErrorMessage = nil
        defer { isTropicsLoading = false }

        guard backend.isAvailable else {
            tropicsErrorMessage = WxCLIError.binaryMissing.errorDescription
            return
        }

        let loc = locationInput.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            let res = try await backend.fetchTropics(
                location: loc.isEmpty ? nil : loc,
                storm: storm,
                units: units
            )
            tropicsPayload = res
        } catch {
            tropicsErrorMessage = error.localizedDescription
        }
    }

    func refreshGrid() async {
        isGridLoading = true
        defer { isGridLoading = false }

        guard backend.isAvailable else { return }

        let favs = favorites
        guard !favs.isEmpty else {
            gridCards = []
            return
        }

        var currentCards = gridCards
        var updatedCards: [LocationGridCardData] = []
        for f in favs {
            let key = f.value.isEmpty ? f.name : f.value
            let name = f.name.isEmpty ? f.value : f.name
            if let existing = currentCards.first(where: { $0.locationKey.caseInsensitiveCompare(key) == .orderedSame }) {
                var c = existing
                c.isLoading = true
                updatedCards.append(c)
            } else {
                updatedCards.append(LocationGridCardData(
                    locationKey: key,
                    displayName: name,
                    payload: nil,
                    isLoading: true,
                    errorMessage: nil,
                    lastUpdated: nil
                ))
            }
        }
        gridCards = updatedCards

        let currentUnits = self.units
        await withTaskGroup(of: (String, Result<WxPayload, Error>).self) { group in
            for f in favs {
                let key = f.value.isEmpty ? f.name : f.value
                group.addTask { [backend = self.backend] in
                    do {
                        let p = try await backend.fetch(location: key, units: currentUnits)
                        return (key, .success(p))
                    } catch {
                        return (key, .failure(error))
                    }
                }
            }

            for await (key, res) in group {
                if let idx = self.gridCards.firstIndex(where: { $0.locationKey.caseInsensitiveCompare(key) == .orderedSame }) {
                    self.gridCards[idx].isLoading = false
                    switch res {
                    case .success(let p):
                        self.gridCards[idx].payload = p
                        self.gridCards[idx].lastUpdated = Date()
                        self.gridCards[idx].errorMessage = nil
                        if let locName = p.conditions?.location, !locName.isEmpty {
                            self.gridCards[idx].displayName = locName
                        }
                    case .failure(let err):
                        self.gridCards[idx].errorMessage = err.localizedDescription
                    }
                }
            }
        }
    }

    func refreshSingleGridCard(_ locationKey: String) async {
        guard let idx = gridCards.firstIndex(where: { $0.locationKey.caseInsensitiveCompare(locationKey) == .orderedSame }) else { return }
        gridCards[idx].isLoading = true
        let currentUnits = self.units
        do {
            let p = try await backend.fetch(location: locationKey, units: currentUnits)
            gridCards[idx].payload = p
            gridCards[idx].lastUpdated = Date()
            gridCards[idx].errorMessage = nil
            if let locName = p.conditions?.location, !locName.isEmpty {
                gridCards[idx].displayName = locName
            }
        } catch {
            gridCards[idx].errorMessage = error.localizedDescription
        }
        gridCards[idx].isLoading = false
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
            notifySevereAlertsIfNeeded(result.alerts)
        } catch {
            errorMessage = error.localizedDescription
            updateStatusItemChrome()
        }
    }

    private func notifySevereAlertsIfNeeded(_ alerts: [Alert]) {
        let center = UNUserNotificationCenter.current()
        for alert in alerts {
            guard !notifiedAlertIDs.contains(alert.id) else { continue }
            notifiedAlertIDs.insert(alert.id)

            let isWarning = alert.event.localizedCaseInsensitiveContains("warning")
            let isExtreme = alert.severity?.localizedCaseInsensitiveContains("extreme") ?? false
            let isSevere = alert.severity?.localizedCaseInsensitiveContains("severe") ?? false

            guard isWarning || isExtreme || isSevere else { continue }

            let content = UNMutableNotificationContent()
            content.title = "⚠️ " + alert.event.uppercased()
            if let area = alert.area, !area.isEmpty {
                content.subtitle = area
            } else if let loc = payload?.conditions?.location {
                content.subtitle = loc
            }
            content.body = alert.headline ?? alert.description ?? "Severe weather warning in effect."
            content.sound = .defaultCritical

            let request = UNNotificationRequest(
                identifier: alert.id,
                content: content,
                trigger: nil
            )
            center.add(request) { err in
                if let err {
                    print("[Severe Alert Notification Error] \(err.localizedDescription)")
                }
            }
        }
    }

    func setMenuBarFormat(_ format: MenuBarFormat) {
        menuBarFormat = format
        WxConfig.setMenuBarFormat(format.rawValue)
        updateStatusItemChrome()
    }

    var activeAlerts: [Alert] {
        payload?.alerts ?? []
    }

    var hasActiveWarning: Bool {
        activeAlerts.contains { $0.isWarning }
    }

    var hasActiveWatchOrAdvisory: Bool {
        activeAlerts.contains { $0.isWatch || $0.isAdvisory }
    }

    var tacticalStationTag: String {
        if let st = payload?.conditions?.station, !st.isEmpty {
            let upper = st.uppercased()
            if upper.count == 4 && upper.hasPrefix("K") {
                return String(upper.dropFirst())
            }
            if upper.count <= 4 && upper != "OPENMETEO" {
                return upper
            }
        }
        if let loc = payload?.conditions?.location, !loc.isEmpty {
            let words = loc.split(separator: " ")
            if let first = words.first {
                let letters = first.filter { $0.isLetter }.prefix(3).uppercased()
                if !letters.isEmpty {
                    return String(letters)
                }
            }
        }
        return "WX"
    }

    static func windArrow(for direction: String?) -> String {
        guard let dir = direction?.uppercased().trimmingCharacters(in: .whitespacesAndNewlines) else { return "" }
        switch dir {
        case "N": return "↓"
        case "NNE", "NE": return "↙"
        case "ENE", "E": return "←"
        case "ESE", "SE": return "↖"
        case "SSE", "S": return "↑"
        case "SSW", "SW": return "↗"
        case "WSW", "W": return "→"
        case "WNW", "NW": return "↘"
        case "NNW": return "↓"
        default: return ""
        }
    }

    var tacticalWindString: String {
        guard let c = payload?.conditions else { return "" }
        let speed: String
        if units == "metric" {
            guard let w = c.windKph else { return "" }
            speed = String(format: "%.0fkm/h", w)
        } else {
            guard let w = c.windMph else { return "" }
            speed = String(format: "%.0fmph", w)
        }
        let arrow = Self.windArrow(for: c.windDirection)
        return "\(arrow)\(speed)"
    }

    var tacticalStatusText: String {
        let tag = tacticalStationTag
        let temp = displayTemp
        let wind = tacticalWindString
        if wind.isEmpty {
            return "[\(tag)] \(temp)"
        }
        return "[\(tag)] \(temp) \(wind)"
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
