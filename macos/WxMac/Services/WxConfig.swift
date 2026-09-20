import Foundation

struct WxConfigFile: Codable, Sendable {
    var defaultLocation: String?
    var units: String?
    var notifications: Bool?

    enum CodingKeys: String, CodingKey {
        case defaultLocation = "default_location"
        case units
        case notifications
    }
}

enum WxConfig {
    static var configURL: URL {
        FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent(".config/wx/config.json")
    }

    static func load() -> WxConfigFile {
        guard let data = try? Data(contentsOf: configURL),
              let decoded = try? JSONDecoder().decode(WxConfigFile.self, from: data) else {
            return WxConfigFile(defaultLocation: nil, units: "imperial", notifications: nil)
        }
        return decoded
    }

    /// Persist via `wx config set` so atomic save stays one implementation.
    @discardableResult
    static func persist(location: String?, units: String?, wxBinary: String) throws -> String {
        var args = ["config", "set"]
        if let location {
            args += ["--location", location]
        }
        if let units, !units.isEmpty {
            args += ["--units", units]
        }
        guard args.count > 2 else { return "" }
        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: wxBinary)
        proc.arguments = args
        let out = Pipe()
        let err = Pipe()
        proc.standardOutput = out
        proc.standardError = err
        try proc.run()
        proc.waitUntilExit()
        if proc.terminationStatus != 0 {
            let msg = String(data: err.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? "config set failed"
            throw WxCLIError.configFailed(msg.trimmingCharacters(in: .whitespacesAndNewlines))
        }
        return String(data: out.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
    }
}
