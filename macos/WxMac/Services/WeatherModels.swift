// JSON DTOs mirroring `wx --json` output only. No weather business logic.
import Foundation

struct WxPayload: Decodable, Sendable {
    var conditions: Conditions?
    var forecast: Forecast?
    var alerts: [Alert]
    var warning: String?

    enum CodingKeys: String, CodingKey {
        case conditions, forecast, alerts, warning
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        conditions = try c.decodeIfPresent(Conditions.self, forKey: .conditions)
        forecast = try c.decodeIfPresent(Forecast.self, forKey: .forecast)
        alerts = try c.decodeIfPresent([Alert].self, forKey: .alerts) ?? []
        warning = try c.decodeIfPresent(String.self, forKey: .warning)
    }
}

struct Conditions: Decodable, Sendable {
    var station: String?
    var observedAt: String?
    var location: String?
    var description: String?
    var conditionCode: String?
    var temperatureC: Double?
    var temperatureF: Double?
    var feelsLikeC: Double?
    var feelsLikeF: Double?
    var dewPointC: Double?
    var dewPointF: Double?
    var humidityPct: Double?
    var windKph: Double?
    var windMph: Double?
    var windGustKph: Double?
    var windGustMph: Double?
    var windDirection: String?
    var pressureHpa: Double?
    var pressureInhg: Double?
    var visibilityM: Double?
    var visibilityMi: Double?

    enum CodingKeys: String, CodingKey {
        case station, location, description
        case observedAt = "observed_at"
        case conditionCode = "condition_code"
        case temperatureC = "temperature_c"
        case temperatureF = "temperature_f"
        case feelsLikeC = "feels_like_c"
        case feelsLikeF = "feels_like_f"
        case dewPointC = "dew_point_c"
        case dewPointF = "dew_point_f"
        case humidityPct = "humidity_pct"
        case windKph = "wind_kph"
        case windMph = "wind_mph"
        case windGustKph = "wind_gust_kph"
        case windGustMph = "wind_gust_mph"
        case windDirection = "wind_direction"
        case pressureHpa = "pressure_hpa"
        case pressureInhg = "pressure_inhg"
        case visibilityM = "visibility_m"
        case visibilityMi = "visibility_mi"
    }
}

struct Forecast: Decodable, Sendable {
    var generatedAt: String?
    var periods: [Period]

    enum CodingKeys: String, CodingKey {
        case generatedAt = "generated_at"
        case periods
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        generatedAt = try c.decodeIfPresent(String.self, forKey: .generatedAt)
        periods = try c.decodeIfPresent([Period].self, forKey: .periods) ?? []
    }
}

struct Period: Decodable, Identifiable, Sendable {
    var id: String { "\(name)-\(startTime ?? "")" }
    var name: String
    var startTime: String?
    var isDaytime: Bool?
    var temperatureF: Double?
    var temperatureC: Double?
    var windMph: Double?
    var windKph: Double?
    var windDirection: String?
    var shortDescription: String?
    var detailedDescription: String?
    var probabilityOfPrecipitation: Double?
    var dewPointC: Double?
    var dewPointF: Double?
    var humidityPct: Double?

    enum CodingKeys: String, CodingKey {
        case name
        case startTime = "start_time"
        case isDaytime = "is_daytime"
        case temperatureF = "temperature_f"
        case temperatureC = "temperature_c"
        case windMph = "wind_mph"
        case windKph = "wind_kph"
        case windDirection = "wind_direction"
        case shortDescription = "short_description"
        case detailedDescription = "detailed_description"
        case probabilityOfPrecipitation = "probability_of_precipitation"
        case dewPointC = "dew_point_c"
        case dewPointF = "dew_point_f"
        case humidityPct = "humidity_pct"
    }
}

struct Alert: Decodable, Identifiable, Sendable {
    var id: String { "\(event)-\(effective ?? "")-\(expires ?? "")" }
    var event: String
    var headline: String?
    var severity: String?
    var urgency: String?
    var effective: String?
    var expires: String?
    var area: String?
    var description: String?
    var instruction: String?
}
