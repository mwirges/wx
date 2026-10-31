import Foundation
import Security

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

struct WxMCPSettings: Codable, Sendable, Equatable {
    var enabled: Bool?
    var port: Int?
    var requireToken: Bool?
    var tokenFile: String?

    enum CodingKeys: String, CodingKey {
        case enabled
        case port
        case requireToken = "require_token"
        case tokenFile = "token_file"
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
    var mcp: WxMCPSettings?

    enum CodingKeys: String, CodingKey {
        case defaultLocation = "default_location"
        case units
        case provider
        case notifications
        case menuBarFormat = "menu_bar_format"
        case favorites
        case recentLocations = "recent_locations"
        case perLocation = "per_location"
        case mcp
    }
}

enum WxMCPTokenError: LocalizedError {
    case symlink
    case unreadable
    case mode
    case random

    var errorDescription: String? {
        switch self {
        case .symlink:
            return "The MCP token file is a symlink. Refusing to use it."
        case .unreadable:
            return "The MCP token file is missing or empty."
        case .mode:
            return "The MCP token file must be mode 0600."
        case .random:
            return "Could not generate an MCP token."
        }
    }
}

enum WxConfig {
    static var configURL: URL {
        if let override = ProcessInfo.processInfo.environment["WX_CONFIG"]?
            .trimmingCharacters(in: .whitespacesAndNewlines), !override.isEmpty {
            return URL(fileURLWithPath: override)
        }
        #if os(iOS)
        // homeDirectoryForCurrentUser is unavailable in the iOS sandbox.
        let base = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first
            ?? URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true)
        return base.appendingPathComponent("wx-config.json")
        #else
        return FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent(".config/wx/config.json")
        #endif
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

    static func updateMCP(_ mutate: (inout WxMCPSettings) -> Void) {
        var cfg = load()
        var mcp = cfg.mcp ?? WxMCPSettings()
        mutate(&mcp)
        cfg.mcp = mcp
        save(cfg)
    }

    static func mcpTokenURL() -> URL {
        let cfg = load()
        if let custom = cfg.mcp?.tokenFile?.trimmingCharacters(in: .whitespacesAndNewlines), !custom.isEmpty {
            if custom.hasPrefix("/") {
                return URL(fileURLWithPath: custom)
            }
            return configURL.deletingLastPathComponent().appendingPathComponent(custom)
        }
        return configURL.deletingLastPathComponent().appendingPathComponent("mcp.token")
    }

    /// Creates mcp.token (mode 0600) when it is missing. Does not print the token.
    static func ensureMCPToken() throws -> String {
        let url = mcpTokenURL()
        if let existing = try? readTokenFile(url), !existing.isEmpty {
            return existing
        }
        var bytes = [UInt8](repeating: 0, count: 32)
        let status = SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes)
        guard status == errSecSuccess else { throw WxMCPTokenError.random }
        let token = bytes.map { String(format: "%02x", $0) }.joined()
        try writeTokenFile(url, token: token)
        return token
    }

    static func readMCPToken() throws -> String {
        let token = try readTokenFile(mcpTokenURL())
        if token.isEmpty { throw WxMCPTokenError.unreadable }
        return token
    }

    private static func readTokenFile(_ url: URL) throws -> String {
        let values = try url.resourceValues(forKeys: [.isSymbolicLinkKey])
        if values.isSymbolicLink == true {
            throw WxMCPTokenError.symlink
        }
        let attrs = try FileManager.default.attributesOfItem(atPath: url.path)
        let mode = (attrs[.posixPermissions] as? NSNumber)?.uint16Value ?? 0
        if mode & 0o077 != 0 {
            throw WxMCPTokenError.mode
        }
        let text = try String(contentsOf: url, encoding: .utf8)
            .trimmingCharacters(in: .whitespacesAndNewlines)
        if text.isEmpty || text.contains(" ") || text.contains("\n") {
            throw WxMCPTokenError.unreadable
        }
        return text
    }

    private static func writeTokenFile(_ url: URL, token: String) throws {
        if let values = try? url.resourceValues(forKeys: [.isSymbolicLinkKey]), values.isSymbolicLink == true {
            throw WxMCPTokenError.symlink
        }
        let dir = url.deletingLastPathComponent()
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        let data = Data((token + "\n").utf8)
        try data.write(to: url, options: .atomic)
        try FileManager.default.setAttributes(
            [.posixPermissions: NSNumber(value: Int16(0o600))],
            ofItemAtPath: url.path
        )
    }
}
