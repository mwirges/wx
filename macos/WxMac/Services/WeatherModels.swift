// JSON DTOs mirroring `wx --json` output only. No weather business logic.
import Foundation

struct WxPayload: Decodable, Sendable {
    var conditions: Conditions?
    var forecast: Forecast?
    var alerts: [Alert]
    var warning: String?
    var astronomy: AstronomyDTO?
    var airQuality: AirQualityDTO?

    enum CodingKeys: String, CodingKey {
        case conditions, forecast, alerts, warning, astronomy
        case airQuality = "air_quality"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        conditions = try c.decodeIfPresent(Conditions.self, forKey: .conditions)
        forecast = try c.decodeIfPresent(Forecast.self, forKey: .forecast)
        alerts = try c.decodeIfPresent([Alert].self, forKey: .alerts) ?? []
        warning = try c.decodeIfPresent(String.self, forKey: .warning)
        astronomy = try c.decodeIfPresent(AstronomyDTO.self, forKey: .astronomy)
        airQuality = try c.decodeIfPresent(AirQualityDTO.self, forKey: .airQuality)
    }
}

struct AstronomyDTO: Decodable, Sendable {
    var sunrise: String?
    var sunset: String?
    var solarNoon: String?
    var dayLengthSeconds: Int64?
    var dayLength: String?
    var isPolarDay: Bool?
    var isPolarNight: Bool?
    var moonPhase: String?
    var moonPhaseIcon: String?
    var moonIlluminationPct: Double?
    var moonAgeDays: Double?

    enum CodingKeys: String, CodingKey {
        case sunrise, sunset
        case solarNoon = "solar_noon"
        case dayLengthSeconds = "day_length_seconds"
        case dayLength = "day_length"
        case isPolarDay = "is_polar_day"
        case isPolarNight = "is_polar_night"
        case moonPhase = "moon_phase"
        case moonPhaseIcon = "moon_phase_icon"
        case moonIlluminationPct = "moon_illumination_pct"
        case moonAgeDays = "moon_age_days"
    }

    var sunriseFormatted: String? {
        guard let sunrise, let date = ISO8601DateFormatter().date(from: sunrise) else { return nil }
        let formatter = DateFormatter()
        formatter.timeStyle = .short
        return formatter.string(from: date)
    }

    var sunsetFormatted: String? {
        guard let sunset, let date = ISO8601DateFormatter().date(from: sunset) else { return nil }
        let formatter = DateFormatter()
        formatter.timeStyle = .short
        return formatter.string(from: date)
    }
}

struct AirQualityDTO: Decodable, Sendable {
    var aqi: Int?
    var category: String?
    var uvIndex: Double?
    var uvCategory: String?
    var pm25: Double?
    var pm10: Double?
    var ozone: Double?
    var no2: Double?
    var co: Double?
    var so2: Double?

    enum CodingKeys: String, CodingKey {
        case aqi, category, ozone, no2, co, so2
        case uvIndex = "uv_index"
        case uvCategory = "uv_category"
        case pm25 = "pm2_5"
        case pm10 = "pm10"
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
    var astronomy: AstronomyDTO?
    var airQuality: AirQualityDTO?

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
        case astronomy
        case airQuality = "air_quality"
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

struct RadarFrame: Decodable, Sendable {
    var validTime: String?
    var imageBase64: String

    enum CodingKeys: String, CodingKey {
        case validTime = "valid_time"
        case imageBase64 = "image_base64"
    }
}

struct RadarBBox: Decodable, Sendable {
    var minLat: Double
    var minLon: Double
    var maxLat: Double
    var maxLon: Double

    enum CodingKeys: String, CodingKey {
        case minLat = "min_lat"
        case minLon = "min_lon"
        case maxLat = "max_lat"
        case maxLon = "max_lon"
    }
}

struct RadarCenter: Decodable, Sendable {
    var lat: Double
    var lon: Double
}

struct RadarPayload: Decodable, Sendable {
    var product: String
    var productLabel: String
    var location: String
    var station: String?
    var validTime: String
    var imageBase64: String
    var radiusKm: Double?
    var raw: Bool?
    var isComposite: Bool?
    var bbox: RadarBBox?
    var center: RadarCenter?
    var frames: [RadarFrame]
    var stations: [String]?

    enum CodingKeys: String, CodingKey {
        case product
        case productLabel = "product_label"
        case location
        case station
        case validTime = "valid_time"
        case imageBase64 = "image_base64"
        case radiusKm = "radius_km"
        case raw
        case isComposite = "is_composite"
        case bbox
        case center
        case frames
        case stations
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        product = try c.decode(String.self, forKey: .product)
        productLabel = try c.decode(String.self, forKey: .productLabel)
        location = try c.decode(String.self, forKey: .location)
        station = try c.decodeIfPresent(String.self, forKey: .station)
        validTime = try c.decode(String.self, forKey: .validTime)
        imageBase64 = try c.decode(String.self, forKey: .imageBase64)
        radiusKm = try c.decodeIfPresent(Double.self, forKey: .radiusKm)
        raw = try c.decodeIfPresent(Bool.self, forKey: .raw)
        isComposite = try c.decodeIfPresent(Bool.self, forKey: .isComposite)
        bbox = try c.decodeIfPresent(RadarBBox.self, forKey: .bbox)
        center = try c.decodeIfPresent(RadarCenter.self, forKey: .center)
        frames = try c.decodeIfPresent([RadarFrame].self, forKey: .frames) ?? []
        stations = try c.decodeIfPresent([String].self, forKey: .stations)
    }
}

struct HistoryDayDTO: Decodable, Identifiable, Sendable {
    var id: String { date }
    var date: String
    var conditionCode: String?
    var description: String?
    var tempMaxC: Double?
    var tempMaxF: Double?
    var tempMinC: Double?
    var tempMinF: Double?
    var apparentMaxC: Double?
    var apparentMaxF: Double?
    var precipMm: Double?
    var precipIn: Double?
    var windMaxKph: Double?
    var windMaxMph: Double?

    enum CodingKeys: String, CodingKey {
        case date, description
        case conditionCode = "condition_code"
        case tempMaxC = "temperature_max_c"
        case tempMaxF = "temperature_max_f"
        case tempMinC = "temperature_min_c"
        case tempMinF = "temperature_min_f"
        case apparentMaxC = "apparent_max_c"
        case apparentMaxF = "apparent_max_f"
        case precipMm = "precipitation_mm"
        case precipIn = "precipitation_in"
        case windMaxKph = "wind_max_kph"
        case windMaxMph = "wind_max_mph"
    }
}

struct HistorySummaryDTO: Decodable, Sendable {
    var daysCount: Int
    var avgTempMaxC: Double?
    var avgTempMaxF: Double?
    var avgTempMinC: Double?
    var avgTempMinF: Double?
    var totalPrecipMm: Double?
    var totalPrecipIn: Double?
    var maxWindKph: Double?
    var maxWindMph: Double?

    enum CodingKeys: String, CodingKey {
        case daysCount = "days_count"
        case avgTempMaxC = "avg_temperature_max_c"
        case avgTempMaxF = "avg_temperature_max_f"
        case avgTempMinC = "avg_temperature_min_c"
        case avgTempMinF = "avg_temperature_min_f"
        case totalPrecipMm = "total_precipitation_mm"
        case totalPrecipIn = "total_precipitation_in"
        case maxWindKph = "max_wind_kph"
        case maxWindMph = "max_wind_mph"
    }
}

struct HistoryPayload: Decodable, Sendable {
    var location: String?
    var latitude: Double
    var longitude: Double
    var elevationM: Double?
    var days: [HistoryDayDTO]
    var summary: HistorySummaryDTO

    enum CodingKeys: String, CodingKey {
        case location, latitude, longitude, days, summary
        case elevationM = "elevation_m"
    }
}

struct AlertCellDTO: Decodable, Identifiable, Sendable {
    var id: String
    var event: String
    var headline: String?
    var areaDesc: String?
    var description: String?
    var severity: String?
    var urgency: String?
    var certainty: String?
    var senderName: String?
    var states: [String]
    var latitude: Double
    var longitude: Double
    var hasPolygon: Bool
    var hazardText: String?
    var radarSite: String?
    var effective: String?
    var expires: String?

    enum CodingKeys: String, CodingKey {
        case id, event, headline, description, severity, urgency, certainty, states, latitude, longitude
        case areaDesc = "area_desc"
        case senderName = "sender_name"
        case hasPolygon = "has_polygon"
        case hazardText = "hazard_text"
        case radarSite = "radar_site"
        case effective, expires
    }
}

struct StormClusterDTO: Decodable, Identifiable, Sendable {
    var id: Int
    var name: String
    var states: [String]
    var centerLat: Double
    var centerLon: Double
    var nearestRadar: String
    var totalAlerts: Int
    var score: Int
    var hazardsCount: [String: Int]
    var primaryHazard: String
    var spcRisk: String?
    var mcdWatch: String?
    var cells: [AlertCellDTO]

    enum CodingKeys: String, CodingKey {
        case id, name, states, score, cells
        case centerLat = "center_lat"
        case centerLon = "center_lon"
        case nearestRadar = "nearest_radar"
        case totalAlerts = "total_alerts"
        case hazardsCount = "hazards_count"
        case primaryHazard = "primary_hazard"
        case spcRisk = "spc_risk"
        case mcdWatch = "mcd_watch"
    }
}

struct ChasePayloadDTO: Decodable, Sendable {
    var generatedAt: String
    var totalAlerts: Int
    var totalClusters: Int
    var spc: SPCPayloadDTO?
    var clusters: [StormClusterDTO]

    enum CodingKeys: String, CodingKey {
        case clusters, spc
        case generatedAt = "generated_at"
        case totalAlerts = "total_alerts"
        case totalClusters = "total_clusters"
    }
}

struct CPCOutlookItemDTO: Decodable, Identifiable, Sendable {
    var id: String { horizon }
    var horizon: String
    var startDate: String
    var endDate: String
    var tempCategory: String
    var tempProbability: Double
    var precipCategory: String
    var precipProbability: Double

    enum CodingKeys: String, CodingKey {
        case horizon
        case startDate = "start_date"
        case endDate = "end_date"
        case tempCategory = "temp_category"
        case tempProbability = "temp_probability"
        case precipCategory = "precip_category"
        case precipProbability = "precip_probability"
    }
}

struct CPCDroughtDTO: Decodable, Sendable {
    var status: String
    var target: String?
}

struct CPCPatternShiftDTO: Decodable, Sendable {
    var hasShift: Bool
    var summary: String
    var tempShift: String?
    var precipShift: String?
    var confidence: String?

    enum CodingKeys: String, CodingKey {
        case hasShift = "has_shift"
        case summary
        case tempShift = "temp_shift"
        case precipShift = "precip_shift"
        case confidence
    }
}

struct CPCPayloadDTO: Decodable, Sendable {
    var location: String
    var coordinates: [Double]?
    var fetchedAt: String
    var outlooks: [CPCOutlookItemDTO]
    var drought: CPCDroughtDTO?
    var patternShift: CPCPatternShiftDTO

    enum CodingKeys: String, CodingKey {
        case location, coordinates, outlooks, drought
        case fetchedAt = "fetched_at"
        case patternShift = "pattern_shift"
    }
}

typealias HistoryPayloadDTO = HistoryPayload

// ── SPC Convective Outlook & Mesoscale Discussion DTOs ──────────────────────

struct SPCRiskCategoryDTO: Decodable, Sendable {
    var dn: Int
    var code: String
    var name: String
    var description: String
    var color: String
}

struct SPCOutlookItemDTO: Decodable, Identifiable, Sendable {
    var id: String { "Day \(day)" }
    var day: Int
    var valid: String?
    var expires: String?
    var issue: String?
    var category: SPCRiskCategoryDTO
    var tornadoProb: String?
    var tornadoSig: Bool?
    var hailProb: String?
    var hailSig: Bool?
    var windProb: String?
    var windSig: Bool?
    var severeProb: String?
    var severeSig: Bool?

    enum CodingKeys: String, CodingKey {
        case day, valid, expires, issue, category
        case tornadoProb = "tornado_prob"
        case tornadoSig = "tornado_sig"
        case hailProb = "hail_prob"
        case hailSig = "hail_sig"
        case windProb = "wind_prob"
        case windSig = "wind_sig"
        case severeProb = "severe_prob"
        case severeSig = "severe_sig"
    }
}

struct MesoscaleDiscussionDTO: Decodable, Identifiable, Sendable {
    var id: Int
    var name: String
    var title: String?
    var url: String?
    var areasAffected: String?
    var concerning: String?
    var watchProbability: String?
    var summary: String?
    var sent: String?
    var expires: String?
    var lat: Double?
    var lon: Double?

    enum CodingKeys: String, CodingKey {
        case id, name, title, url, summary, sent, expires, lat, lon
        case areasAffected = "areas_affected"
        case concerning
        case watchProbability = "watch_probability"
    }
}

struct SPCWatchDTO: Decodable, Identifiable, Sendable {
    var id: String
    var watchNumber: Int
    var type: String
    var headline: String?
    var areaDesc: String?
    var states: [String]?
    var effective: String?
    var expires: String?
    var active: Bool?
    var severity: String?
    var urgency: String?
    var url: String?

    enum CodingKeys: String, CodingKey {
        case id, type, headline, states, effective, expires, active, severity, urgency, url
        case watchNumber = "watch_number"
        case areaDesc = "area_desc"
    }
}

struct SPCPayloadDTO: Decodable, Sendable {
    var location: String?
    var coordinates: [Double]?
    var fetchedAt: String?
    var day1: SPCOutlookItemDTO?
    var day2: SPCOutlookItemDTO?
    var day3: SPCOutlookItemDTO?
    var activeMCDs: [MesoscaleDiscussionDTO]?
    var activeWatches: [SPCWatchDTO]?
    var maxNationalRisk: SPCRiskCategoryDTO?
    var convectiveSummary: String?

    enum CodingKeys: String, CodingKey {
        case location, coordinates
        case fetchedAt = "fetched_at"
        case day1, day2, day3
        case activeMCDs = "active_mcds"
        case activeWatches = "active_watches"
        case maxNationalRisk = "max_national_risk"
        case convectiveSummary = "convective_summary"
    }
}
