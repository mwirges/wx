import Foundation

enum IOSClientError: LocalizedError {
    case bridge(code: String, message: String)
    case decode(String)
    case missingFixture(String)

    var errorDescription: String? {
        switch self {
        case .bridge(let code, let message):
            return "\(code): \(message)"
        case .decode(let message):
            return "Could not read wx JSON: \(message)"
        case .missingFixture(let name):
            return "Missing fixture \(name).json"
        }
    }
}

protocol IOSWeatherClient: Sendable {
    func run(args: [String]) async throws -> String
}

enum IOSClients {
    /// UI tests and `make ios-sim` set WX_FIXTURE or WX_UI_TEST.
    /// Those paths never call WxRunJSON, so they never reach NWS.
    static func make() -> any IOSWeatherClient {
        let env = ProcessInfo.processInfo.environment
        let fixture = env["WX_FIXTURE"] == "1" || env["WX_UI_TEST"] == "1"
        if fixture {
            return FixtureClient()
        }
        return BridgeClient()
    }
}

struct BridgeClient: IOSWeatherClient {
    func run(args: [String]) async throws -> String {
        try await WxBridge.run(args: args)
    }
}

struct FixtureClient: IOSWeatherClient {
    func run(args: [String]) async throws -> String {
        guard let name = args.first, !name.isEmpty else {
            throw IOSClientError.missingFixture("(empty)")
        }
        let bundle = Bundle.main
        let url = bundle.url(forResource: name, withExtension: "json", subdirectory: "Fixtures")
            ?? bundle.url(forResource: name, withExtension: "json")
        guard let url else {
            throw IOSClientError.missingFixture(name)
        }
        return try String(contentsOf: url, encoding: .utf8)
    }
}

enum BridgeJSON {
    private struct Envelope: Decodable {
        struct Item: Decodable {
            var code: String
            var message: String
        }
        var error: Item
    }

    static func decode<T: Decodable>(_ type: T.Type, from raw: String) throws -> T {
        let data = Data(raw.utf8)
        if let env = try? JSONDecoder().decode(Envelope.self, from: data) {
            throw IOSClientError.bridge(code: env.error.code, message: env.error.message)
        }
        do {
            return try JSONDecoder().decode(type, from: data)
        } catch {
            throw IOSClientError.decode(error.localizedDescription)
        }
    }
}
