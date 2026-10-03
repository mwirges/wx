// JSON DTOs mirroring `wx --json` output only. No weather business logic.
import Foundation

struct WxPayload: Decodable, Sendable {
    var conditions: Conditions?
    var forecast: Forecast?
    var alerts: [Alert]
    var warning: String?
    var astronomy: AstronomyDTO?
    var airQuality: AirQualityDTO?
    var nowcast: NowcastDTO?

    enum CodingKeys: String, CodingKey {
        case conditions, forecast, alerts, warning, astronomy, nowcast
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
        nowcast = try c.decodeIfPresent(NowcastDTO.self, forKey: .nowcast)
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
    var civilDawn: String?
    var civilDusk: String?
    var nauticalDawn: String?
    var nauticalDusk: String?
    var astroDawn: String?
    var astroDusk: String?
    var goldenHourMorningStart: String?
    var goldenHourMorningEnd: String?
    var goldenHourEveningStart: String?
    var goldenHourEveningEnd: String?
    var solarElevationDeg: Double?
    var solarAzimuthDeg: Double?
    var currentPeriod: String?
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
        case civilDawn = "civil_dawn"
        case civilDusk = "civil_dusk"
        case nauticalDawn = "nautical_dawn"
        case nauticalDusk = "nautical_dusk"
        case astroDawn = "astro_dawn"
        case astroDusk = "astro_dusk"
        case goldenHourMorningStart = "golden_hour_morning_start"
        case goldenHourMorningEnd = "golden_hour_morning_end"
        case goldenHourEveningStart = "golden_hour_evening_start"
        case goldenHourEveningEnd = "golden_hour_evening_end"
        case solarElevationDeg = "solar_elevation_deg"
        case solarAzimuthDeg = "solar_azimuth_deg"
        case currentPeriod = "current_period"
        case moonPhase = "moon_phase"
        case moonPhaseIcon = "moon_phase_icon"
        case moonIlluminationPct = "moon_illumination_pct"
        case moonAgeDays = "moon_age_days"
    }

    private func formatIsoTime(_ iso: String?) -> String? {
        guard let iso, let date = ISO8601DateFormatter().date(from: iso) else { return nil }
        let formatter = DateFormatter()
        formatter.timeStyle = .short
        return formatter.string(from: date)
    }

    var sunriseFormatted: String? { formatIsoTime(sunrise) }
    var sunsetFormatted: String? { formatIsoTime(sunset) }
    var solarNoonFormatted: String? { formatIsoTime(solarNoon) }
    var civilDawnFormatted: String? { formatIsoTime(civilDawn) }
    var civilDuskFormatted: String? { formatIsoTime(civilDusk) }
    var nauticalDawnFormatted: String? { formatIsoTime(nauticalDawn) }
    var nauticalDuskFormatted: String? { formatIsoTime(nauticalDusk) }
    var astroDawnFormatted: String? { formatIsoTime(astroDawn) }
    var astroDuskFormatted: String? { formatIsoTime(astroDusk) }

    var goldenHourMorningFormatted: String? {
        guard let s = formatIsoTime(goldenHourMorningStart), let e = formatIsoTime(goldenHourMorningEnd) else { return nil }
        return "\(s) – \(e)"
    }

    var goldenHourEveningFormatted: String? {
        guard let s = formatIsoTime(goldenHourEveningStart), let e = formatIsoTime(goldenHourEveningEnd) else { return nil }
        return "\(s) – \(e)"
    }

    var solarAzimuthCompass: String {
        guard let az = solarAzimuthDeg else { return "" }
        let dirs = ["N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"]
        let idx = Int((az + 11.25) / 22.5) % 16
        return dirs[max(0, min(15, idx))]
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
    var healthAdvisory: String?
    var smokeAdvisory: String?

    enum CodingKeys: String, CodingKey {
        case aqi, category, ozone, no2, co, so2
        case uvIndex = "uv_index"
        case uvCategory = "uv_category"
        case pm25 = "pm2_5"
        case pm10 = "pm10"
        case healthAdvisory = "health_advisory"
        case smokeAdvisory = "smoke_advisory"
    }

    var resolvedAdvisory: String {
        if let ha = healthAdvisory, !ha.isEmpty {
            return ha
        }
        guard let aqi = aqi else { return "Air quality data is currently unavailable." }
        switch aqi {
        case ..<51:
            return "Air quality is satisfactory, and air pollution poses little or no risk."
        case 51...100:
            return "Air quality is acceptable; sensitive individuals should consider limiting prolonged outdoor exertion."
        case 101...150:
            return "Members of sensitive groups may experience health effects. The general public is less likely to be affected."
        case 151...200:
            return "Some members of the general public may experience health effects; sensitive groups may experience more serious effects."
        case 201...300:
            return "Health alert: The risk of health effects is increased for everyone. Limit outdoor activities."
        default:
            return "Health warning of emergency conditions: everyone is more likely to be affected. Avoid outdoor exertion."
        }
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
    var nowcast: NowcastDTO?

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
        case nowcast
    }
}

struct Forecast: Decodable, Sendable {
    var generatedAt: String?
    var periods: [Period]
    var hourly: [Period]?

    enum CodingKeys: String, CodingKey {
        case generatedAt = "generated_at"
        case periods
        case hourly
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        generatedAt = try c.decodeIfPresent(String.self, forKey: .generatedAt)
        periods = try c.decodeIfPresent([Period].self, forKey: .periods) ?? []
        hourly = try c.decodeIfPresent([Period].self, forKey: .hourly)
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

    var isWarning: Bool {
        event.localizedCaseInsensitiveContains("warning") ||
        severity?.localizedCaseInsensitiveCompare("extreme") == .orderedSame ||
        severity?.localizedCaseInsensitiveCompare("severe") == .orderedSame
    }

    var isWatch: Bool {
        event.localizedCaseInsensitiveContains("watch")
    }

    var isAdvisory: Bool {
        event.localizedCaseInsensitiveContains("advisory") ||
        event.localizedCaseInsensitiveContains("statement")
    }
}

enum MenuBarFormat: String, CaseIterable, Identifiable, Codable, Sendable {
    case compact = "compact"
    case standard = "standard"
    case tactical = "tactical"

    var id: String { rawValue }

    var displayName: String {
        switch self {
        case .compact: return "Compact (72°)"
        case .standard: return "Standard (☀️ 72°)"
        case .tactical: return "Tactical ([FWA] 72° ↘12mph)"
        }
    }
}

struct RadarFrame: Decodable, Sendable {
    var validTime: String?
    var imageBase64: String

    enum CodingKeys: String, CodingKey {
        case validTime = "valid_time"
        case imageBase64 = "image_base64"
    }
}

struct RadarBBox: Decodable, Sendable, Equatable {
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
    var soundingStation: String?
    var cells: [AlertCellDTO]

    enum CodingKeys: String, CodingKey {
        case id, name, states, score, cells
        case centerLat = "center_lat"
        case centerLon = "center_lon"
        case nearestRadar = "nearest_radar"
        case soundingStation = "sounding_station"
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

struct PrecipIntervalDTO: Decodable, Identifiable, Sendable {
    var id: String { startTime ?? UUID().uuidString }
    var startTime: String?
    var endTime: String?
    var probability: Double?
    var rateMmh: Double?
    var rateInh: Double?
    var accumMm: Double?
    var accumIn: Double?
    var phase: String?
    var summary: String?

    enum CodingKeys: String, CodingKey {
        case startTime = "start_time"
        case endTime = "end_time"
        case probability
        case rateMmh = "rate_mmh"
        case rateInh = "rate_inh"
        case accumMm = "accum_mm"
        case accumIn = "accum_in"
        case phase, summary
    }

    var formattedTime: String {
        guard let startTime else { return "—" }
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        var date = formatter.date(from: startTime)
        if date == nil {
            formatter.formatOptions = [.withInternetDateTime]
            date = formatter.date(from: startTime)
        }
        guard let date else {
            if startTime.contains("T") {
                let parts = startTime.split(separator: "T")
                if parts.count > 1 {
                    return String(parts[1].prefix(5))
                }
            }
            return startTime
        }
        let out = DateFormatter()
        out.timeStyle = .short
        return out.string(from: date)
    }

    var sparkRune: Character {
        let rate = rateInh ?? 0
        if rate <= 0.001 { return " " }
        if rate < 0.03 { return " " }
        if rate < 0.08 { return "▂" }
        if rate < 0.15 { return "▃" }
        if rate < 0.25 { return "▄" }
        if rate < 0.40 { return "▅" }
        if rate < 0.60 { return "▆" }
        if rate < 0.90 { return "▇" }
        return "█"
    }
}

struct NowcastDTO: Decodable, Sendable {
    var generatedAt: String?
    var location: String?
    var headline: String?
    var isActivePrecip: Bool?
    var summary: String?
    var primaryPhase: String?
    var totalLiquidMm: Double?
    var totalLiquidIn: Double?
    var totalSnowCm: Double?
    var totalSnowIn: Double?
    var peakRateMmh: Double?
    var peakRateInh: Double?
    var peakTime: String?
    var nextPrecipTime: String?
    var precipEndTime: String?
    var intervals: [PrecipIntervalDTO]?

    enum CodingKeys: String, CodingKey {
        case generatedAt = "generated_at"
        case location, headline
        case isActivePrecip = "is_active_precip"
        case summary
        case primaryPhase = "primary_phase"
        case totalLiquidMm = "total_liquid_mm"
        case totalLiquidIn = "total_liquid_in"
        case totalSnowCm = "total_snow_cm"
        case totalSnowIn = "total_snow_in"
        case peakRateMmh = "peak_rate_mmh"
        case peakRateInh = "peak_rate_inh"
        case peakTime = "peak_time"
        case nextPrecipTime = "next_precip_time"
        case precipEndTime = "precip_end_time"
        case intervals
    }
}

struct NowcastPayloadDTO: Decodable, Sendable {
    var location: String?
    var nowcast: NowcastDTO?
}

struct DailyNormalsDTO: Decodable, Sendable {
    var date: String?
    var normalHighF: Double?
    var normalHighC: Double?
    var normalLowF: Double?
    var normalLowC: Double?
    var normalMeanF: Double?
    var normalMeanC: Double?
    var normalPrecipIn: Double?
    var normalPrecipMm: Double?

    enum CodingKeys: String, CodingKey {
        case date
        case normalHighF = "normal_high_f"
        case normalHighC = "normal_high_c"
        case normalLowF = "normal_low_f"
        case normalLowC = "normal_low_c"
        case normalMeanF = "normal_mean_f"
        case normalMeanC = "normal_mean_c"
        case normalPrecipIn = "normal_precip_in"
        case normalPrecipMm = "normal_precip_mm"
    }
}

struct DailyRecordDTO: Decodable, Sendable {
    var valueF: Double?
    var valueC: Double?
    var valueIn: Double?
    var valueMm: Double?
    var years: [Int]?

    enum CodingKeys: String, CodingKey {
        case valueF = "value_f"
        case valueC = "value_c"
        case valueIn = "value_in"
        case valueMm = "value_mm"
        case years
    }
}

struct DailyRecordsDTO: Decodable, Sendable {
    var recordHigh: DailyRecordDTO?
    var recordLow: DailyRecordDTO?
    var recordPrecip: DailyRecordDTO?
    var coldestHigh: DailyRecordDTO?
    var warmestLow: DailyRecordDTO?
    var periodOfRecord: String?
    var totalYearsSampled: Int?

    enum CodingKeys: String, CodingKey {
        case recordHigh = "record_high"
        case recordLow = "record_low"
        case recordPrecip = "record_precip"
        case coldestHigh = "coldest_high"
        case warmestLow = "warmest_low"
        case periodOfRecord = "period_of_record"
        case totalYearsSampled = "total_years_sampled"
    }
}

struct ClimateDepartureDTO: Decodable, Sendable {
    var observedCurrentF: Double?
    var observedCurrentC: Double?
    var departureCurrentF: Double?
    var departureCurrentC: Double?
    var summary: String?

    enum CodingKeys: String, CodingKey {
        case observedCurrentF = "observed_current_f"
        case observedCurrentC = "observed_current_c"
        case departureCurrentF = "departure_current_f"
        case departureCurrentC = "departure_current_c"
        case summary
    }
}

struct MonthlyNormalsDTO: Decodable, Sendable {
    var monthName: String?
    var normalAvgHighF: Double?
    var normalAvgHighC: Double?
    var normalAvgLowF: Double?
    var normalAvgLowC: Double?
    var normalTotalPrecipIn: Double?
    var normalTotalPrecipMm: Double?

    enum CodingKeys: String, CodingKey {
        case monthName = "month_name"
        case normalAvgHighF = "normal_avg_high_f"
        case normalAvgHighC = "normal_avg_high_c"
        case normalAvgLowF = "normal_avg_low_f"
        case normalAvgLowC = "normal_avg_low_c"
        case normalTotalPrecipIn = "normal_total_precip_in"
        case normalTotalPrecipMm = "normal_total_precip_mm"
    }
}

struct ClimateReportDTO: Decodable, Sendable {
    var date: String?
    var location: String?
    var stationId: String?
    var stationName: String?
    var latitude: Double?
    var longitude: Double?
    var elevationFt: Double?
    var elevationM: Double?
    var normalsPeriod: String?
    var todayNormals: DailyNormalsDTO?
    var records: DailyRecordsDTO?
    var departure: ClimateDepartureDTO?
    var monthlyNormals: MonthlyNormalsDTO?

    enum CodingKeys: String, CodingKey {
        case date, location
        case stationId = "station_id"
        case stationName = "station_name"
        case latitude, longitude
        case elevationFt = "elevation_ft"
        case elevationM = "elevation_m"
        case normalsPeriod = "normals_period"
        case todayNormals = "today_normals"
        case records, departure
        case monthlyNormals = "monthly_normals"
    }
}

struct ClimatePayloadDTO: Decodable, Sendable {
    var location: String?
    var climate: ClimateReportDTO?
}

struct SoundingLevelDTO: Decodable, Sendable, Identifiable {
    var id: String { String(format: "%.0f", pressureHpa) }
    var pressureHpa: Double
    var heightM: Double?
    var heightFt: Double?
    var tempC: Double?
    var tempF: Double?
    var dewpointC: Double?
    var dewpointF: Double?
    var windDirDeg: Double?
    var windSpeedKt: Double?
    var windSpeedKph: Double?
    var windSpeedMph: Double?
    var rhPct: Double?

    enum CodingKeys: String, CodingKey {
        case pressureHpa = "pressure_hpa"
        case heightM = "height_m"
        case heightFt = "height_ft"
        case tempC = "temp_c"
        case tempF = "temp_f"
        case dewpointC = "dewpoint_c"
        case dewpointF = "dewpoint_f"
        case windDirDeg = "wind_dir_deg"
        case windSpeedKt = "wind_speed_kt"
        case windSpeedKph = "wind_speed_kph"
        case windSpeedMph = "wind_speed_mph"
        case rhPct = "rh_pct"
    }
}

struct ConvectiveIndicesDTO: Decodable, Sendable {
    var sbcapeJkg: Double?
    var mlcapeJkg: Double?
    var mucapeJkg: Double?
    var sbcinJkg: Double?
    var mlcinJkg: Double?
    var mucinJkg: Double?
    var sbliC: Double?
    var mlliC: Double?
    var muliC: Double?
    var pwatIn: Double?
    var pwatMm: Double?
    var freezingLevelM: Double?
    var freezingLevelFt: Double?
    var dcapeJkg: Double?
    var bulkShear01Kt: Double?
    var bulkShear03Kt: Double?
    var bulkShear06Kt: Double?
    var srh01M2s2: Double?
    var srh03M2s2: Double?
    var stp: Double?
    var scp: Double?
    var ship: Double?
    var lapseRate700_500: Double?
    var lapseRate850_500: Double?
    var instabilitySummary: String?
    var shearSummary: String?
    var convectiveRisk: String?

    enum CodingKeys: String, CodingKey {
        case sbcapeJkg = "sbcape_jkg"
        case mlcapeJkg = "mlcape_jkg"
        case mucapeJkg = "mucape_jkg"
        case sbcinJkg = "sbcin_jkg"
        case mlcinJkg = "mlcin_jkg"
        case mucinJkg = "mucin_jkg"
        case sbliC = "sbli_c"
        case mlliC = "mlli_c"
        case muliC = "muli_c"
        case pwatIn = "pwat_in"
        case pwatMm = "pwat_mm"
        case freezingLevelM = "freezing_level_m"
        case freezingLevelFt = "freezing_level_ft"
        case dcapeJkg = "dcape_jkg"
        case bulkShear01Kt = "bulk_shear_0_1km_kt"
        case bulkShear03Kt = "bulk_shear_0_3km_kt"
        case bulkShear06Kt = "bulk_shear_0_6km_kt"
        case srh01M2s2 = "srh_0_1km_m2s2"
        case srh03M2s2 = "srh_0_3km_m2s2"
        case stp, scp, ship
        case lapseRate700_500 = "lapse_rate_700_500_c_km"
        case lapseRate850_500 = "lapse_rate_850_500_c_km"
        case instabilitySummary = "instability_summary"
        case shearSummary = "shear_summary"
        case convectiveRisk = "convective_risk"
    }
}

struct SoundingReportDTO: Decodable, Sendable {
    var timestamp: String?
    var location: String?
    var stationId: String?
    var stationName: String?
    var distanceKm: Double?
    var distanceMiles: Double?
    var provider: String?
    var skewtImageUrl: String?
    var indices: ConvectiveIndicesDTO?
    var levels: [SoundingLevelDTO]?

    enum CodingKeys: String, CodingKey {
        case timestamp, location
        case stationId = "station_id"
        case stationName = "station_name"
        case distanceKm = "distance_km"
        case distanceMiles = "distance_miles"
        case provider
        case skewtImageUrl = "skewt_image_url"
        case indices, levels
    }
}

struct SoundingPayloadDTO: Decodable, Sendable {
    var location: String?
    var sounding: SoundingReportDTO?
}

struct TropicalStormDTO: Decodable, Identifiable, Sendable {
    var id: String
    var binNumber: String?
    var name: String
    var classification: String
    var classificationName: String?
    var category: Int
    var categoryLabel: String?
    var intensityKt: Int
    var windSpeedMph: Int
    var windSpeedKmh: Int
    var pressureMb: Int
    var pressureInHg: Double
    var latitude: Double
    var longitude: Double
    var locationText: String
    var movementDir: Int
    var movementCompass: String
    var movementSpeedMph: Int
    var movementSpeedKmh: Int
    var headline: String?
    var proximityText: String?
    var distanceKm: Double?
    var distanceMiles: Double?
    var watchesWarnings: [String]?
    var advisoryNumber: String?
    var advisoryTime: String?
    var publicAdvisoryUrl: String?
    var forecastDiscussionUrl: String?
    var graphicsUrl: String?
    var trackConeKmz: String?
    var lastUpdate: String?

    enum CodingKeys: String, CodingKey {
        case id
        case binNumber = "bin_number"
        case name, classification
        case classificationName = "classification_name"
        case category
        case categoryLabel = "category_label"
        case intensityKt = "intensity_kt"
        case windSpeedMph = "wind_speed_mph"
        case windSpeedKmh = "wind_speed_kmh"
        case pressureMb = "pressure_mb"
        case pressureInHg = "pressure_inhg"
        case latitude, longitude
        case locationText = "location_text"
        case movementDir = "movement_dir"
        case movementCompass = "movement_compass"
        case movementSpeedMph = "movement_speed_mph"
        case movementSpeedKmh = "movement_speed_kmh"
        case headline
        case proximityText = "proximity_text"
        case distanceKm = "distance_km"
        case distanceMiles = "distance_miles"
        case watchesWarnings = "watches_warnings"
        case advisoryNumber = "advisory_number"
        case advisoryTime = "advisory_time"
        case publicAdvisoryUrl = "public_advisory_url"
        case forecastDiscussionUrl = "forecast_discussion_url"
        case graphicsUrl = "graphics_url"
        case trackConeKmz = "track_cone_kmz"
        case lastUpdate = "last_update"
    }
}

struct TropicalDisturbanceDTO: Decodable, Identifiable, Sendable {
    var id: String
    var basin: String
    var name: String
    var chance48h: Int
    var category48h: String
    var chance7d: Int
    var category7d: String
    var summary: String?

    enum CodingKeys: String, CodingKey {
        case id, basin, name
        case chance48h = "chance_48h"
        case category48h = "category_48h"
        case chance7d = "chance_7d"
        case category7d = "category_7d"
        case summary
    }
}

struct TropicsReportDTO: Decodable, Sendable {
    var generatedAt: String?
    var totalActive: Int
    var storms: [TropicalStormDTO]
    var disturbances: [TropicalDisturbanceDTO]?
    var atlanticOutlookUrl: String?
    var pacificOutlookUrl: String?
    var referenceLocation: String?

    enum CodingKeys: String, CodingKey {
        case generatedAt = "generated_at"
        case totalActive = "total_active"
        case storms, disturbances
        case atlanticOutlookUrl = "atlantic_outlook_url"
        case pacificOutlookUrl = "pacific_outlook_url"
        case referenceLocation = "reference_location"
    }
}

struct TropicsPayloadDTO: Decodable, Sendable {
    var location: String?
    var tropics: TropicsReportDTO?
}



