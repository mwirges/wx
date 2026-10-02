import Foundation

struct WxLocationEntry: Codable, Identifiable, Hashable, Sendable {
    var id: String { name }
    var name: String
    var value: String
}

struct WxPerLocationSettings: Codable, Sendable {
    var units: String?
    var provider: String?
    var defaultRadarProduct: String?
    var defaultRadarRadius: Double?
    var radarStation: String?

    enum CodingKeys: String, CodingKey {
        case units
        case provider
        case defaultRadarProduct = "default_radar_product"
        case defaultRadarRadius = "default_radar_radius"
        case radarStation = "radar_station"
    }
}

struct WxConfigFile: Codable, Sendable {
    var defaultLocation: String?
    var units: String?
    var provider: String?
    var notifications: Bool?
    var menuBarFormat: String?
    var favorites: [WxLocationEntry]?
    var recentLocations: [String]?
    var perLocation: [String: WxPerLocationSettings]?

    enum CodingKeys: String, CodingKey {
        case defaultLocation = "default_location"
        case units
        case provider
        case notifications
        case menuBarFormat = "menu_bar_format"
        case favorites
        case recentLocations = "recent_locations"
        case perLocation = "per_location"
    }
}

enum WxConfig {
    static var configURL: URL {
        if let override = ProcessInfo.processInfo.environment["WX_CONFIG"]?
            .trimmingCharacters(in: .whitespacesAndNewlines), !override.isEmpty {
            return URL(fileURLWithPath: override)
        }
        return FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent(".config/wx/config.json")
    }

    static func load() -> WxConfigFile {
        guard let data = try? Data(contentsOf: configURL),
              let decoded = try? JSONDecoder().decode(WxConfigFile.self, from: data) else {
            return WxConfigFile(defaultLocation: nil, units: "imperial", provider: nil, notifications: nil, favorites: [], recentLocations: [])
        }
        return decoded
    }

    static func save(_ config: WxConfigFile) {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        guard let data = try? encoder.encode(config) else { return }
        let dir = configURL.deletingLastPathComponent()
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        try? data.write(to: configURL, options: .atomic)
    }

    static func addFavorite(name: String, value: String) {
        var cfg = load()
        var favs = cfg.favorites ?? []
        let cleanName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        let cleanVal = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !cleanName.isEmpty && !cleanVal.isEmpty else { return }

        if let idx = favs.firstIndex(where: { $0.name.caseInsensitiveCompare(cleanName) == .orderedSame }) {
            favs[idx] = WxLocationEntry(name: cleanName, value: cleanVal)
        } else {
            favs.append(WxLocationEntry(name: cleanName, value: cleanVal))
        }
        cfg.favorites = favs
        save(cfg)
    }

    static func removeFavorite(nameOrValue: String) {
        var cfg = load()
        var favs = cfg.favorites ?? []
        favs.removeAll(where: {
            $0.name.caseInsensitiveCompare(nameOrValue) == .orderedSame ||
            $0.value.caseInsensitiveCompare(nameOrValue) == .orderedSame
        })
        cfg.favorites = favs
        save(cfg)
    }

    static func addRecent(location: String) {
        let clean = location.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !clean.isEmpty else { return }
        var cfg = load()
        var recents = cfg.recentLocations ?? []
        recents.removeAll(where: { $0.caseInsensitiveCompare(clean) == .orderedSame })
        recents.insert(clean, at: 0)
        if recents.count > 10 {
            recents = Array(recents.prefix(10))
        }
        cfg.recentLocations = recents
        save(cfg)
    }

    static func clearRecents() {
        var cfg = load()
        cfg.recentLocations = []
        save(cfg)
    }

    static func setMenuBarFormat(_ format: String) {
        var cfg = load()
        cfg.menuBarFormat = format
        save(cfg)
    }

    /// Persist default location and units via atomic save
    @discardableResult
    static func persist(location: String?, units: String?, wxBinary: String) throws -> String {
        var cfg = load()
        if let location, !location.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            cfg.defaultLocation = location.trimmingCharacters(in: .whitespacesAndNewlines)
            addRecent(location: cfg.defaultLocation!)
        }
        if let units, !units.isEmpty {
            cfg.units = units
        }
        save(cfg)
        return "saved"
    }
}
