import SwiftUI

enum ConditionSymbol {
    static func systemName(for code: String?, isNight: Bool = false) -> String {
        if let known = explicitSystemName(for: code, isNight: isNight) {
            return known
        }
        return isNight ? "moon.stars.fill" : "cloud.sun.fill"
    }

    /// Determines whether current conditions occur at night using solar ephemeris,
    /// condition codes, and lead forecast periods (handling ASOS stations like KFWA
    /// that omit `condition_code` in METAR observations).
    static func isNight(
        conditions: Conditions?,
        astronomy: AstronomyDTO? = nil,
        forecast: Forecast? = nil,
        now: Date = Date()
    ) -> Bool {
        let astro = astronomy ?? conditions?.astronomy
        if astro?.isPolarNight == true {
            return true
        }
        if astro?.isPolarDay == true {
            return false
        }
        if let elev = astro?.solarElevationDeg {
            return elev < 0
        }
        if let period = astro?.currentPeriod?.lowercased(), !period.isEmpty {
            if period.contains("night") || period.contains("twilight") || period.contains("blue") {
                return true
            }
            if period.contains("day") || period.contains("golden") {
                return false
            }
        }
        let code = (conditions?.conditionCode ?? "").lowercased()
        if code.contains("night") {
            return true
        }
        if code.contains("day") {
            return false
        }
        if let hourlyLead = forecast?.currentHourlyPeriod(at: now), let dt = hourlyLead.isDaytime {
            return !dt
        }
        if let dailyLead = forecast?.periods.first {
            return !dailyLead.resolvedIsDaytime
        }
        return false
    }

    static func isNight(for payload: WxPayload?, now: Date = Date()) -> Bool {
        isNight(
            conditions: payload?.conditions,
            astronomy: payload?.conditions?.astronomy ?? payload?.astronomy,
            forecast: payload?.forecast,
            now: now
        )
    }

    /// Resolves the SF Symbol for current conditions across ASOS/METAR, solar ephemeris,
    /// active nowcast precipitation, and lead hourly/daily forecast descriptions.
    static func resolvedSystemName(
        conditions: Conditions?,
        astronomy: AstronomyDTO? = nil,
        forecast: Forecast? = nil,
        nowcast: NowcastDTO? = nil,
        now: Date = Date()
    ) -> String {
        let night = isNight(conditions: conditions, astronomy: astronomy, forecast: forecast, now: now)
        let isDay = !night

        if let known = explicitSystemName(for: conditions?.conditionCode, isNight: night) {
            return known
        }

        let nc = nowcast ?? conditions?.nowcast
        if nc?.isActivePrecip == true {
            let phase = (nc?.primaryPhase ?? "").lowercased()
            if phase.contains("snow") {
                return "cloud.snow.fill"
            }
            if phase.contains("sleet") || phase.contains("ice") || phase.contains("freez") {
                return "cloud.sleet.fill"
            }
            return "cloud.rain.fill"
        }

        if let sym = symbolFromDescription(conditions?.description, isDay: isDay) {
            return sym
        }
        if let sym = symbolFromDescription(forecast?.currentHourlyPeriod(at: now)?.shortDescription, isDay: isDay) {
            return sym
        }
        if let sym = symbolFromDescription(forecast?.periods.first?.shortDescription, isDay: isDay) {
            return sym
        }

        return night ? "moon.stars.fill" : "cloud.sun.fill"
    }

    static func resolvedSystemName(for payload: WxPayload?, nowcast: NowcastDTO? = nil, now: Date = Date()) -> String {
        resolvedSystemName(
            conditions: payload?.conditions,
            astronomy: payload?.conditions?.astronomy ?? payload?.astronomy,
            forecast: payload?.forecast,
            nowcast: nowcast ?? payload?.conditions?.nowcast ?? payload?.nowcast,
            now: now
        )
    }

    /// Resolves human-readable sky condition text when automated ASOS stations
    /// return an empty/nil `description`, falling back to lead hourly/daily forecasts.
    static func resolvedDescription(conditions: Conditions?, forecast: Forecast? = nil, now: Date = Date()) -> String? {
        if let d = conditions?.description?.trimmingCharacters(in: .whitespacesAndNewlines), !d.isEmpty {
            return d
        }
        if let h = forecast?.currentHourlyPeriod(at: now)?.shortDescription?.trimmingCharacters(in: .whitespacesAndNewlines), !h.isEmpty {
            return h
        }
        if let p = forecast?.periods.first?.shortDescription?.trimmingCharacters(in: .whitespacesAndNewlines), !p.isEmpty {
            return p
        }
        if let code = conditions?.conditionCode?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased(), !code.isEmpty {
            switch code {
            case "clear", "clear-day", "clear-night": return "Clear"
            case "partly-cloudy", "partly_cloudy", "partly-cloudy-day", "partly-cloudy-night": return "Partly Cloudy"
            case "cloudy": return "Cloudy"
            case "rain": return "Rain"
            case "heavy-rain": return "Heavy Rain"
            case "snow": return "Snow"
            case "sleet": return "Wintry Mix"
            case "thunder": return "Thunderstorms"
            case "fog": return "Fog"
            case "wind": return "Windy"
            default: break
            }
        }
        return nil
    }

    static func resolvedDescription(for payload: WxPayload?, now: Date = Date()) -> String? {
        resolvedDescription(conditions: payload?.conditions, forecast: payload?.forecast, now: now)
    }

    /// Tints weather symbols coherently for day vs night (lunar periwinkle at night,
    /// solar gold for daytime sun, luminous cyan for precip, cool silver for overcast/fog).
    static func tint(for symbol: String, isNight: Bool) -> Color {
        if symbol.contains("bolt") {
            return WxGlassTheme.gold
        }
        if symbol.contains("snow") || symbol.contains("sleet") {
            return WxGlassTheme.cyanSoft
        }
        if symbol.contains("rain") || symbol.contains("drizzle") {
            return WxGlassTheme.cyan
        }
        if symbol == "cloud.fill" || symbol == "cloud.fog.fill" {
            return WxGlassTheme.textSecondary
        }
        if symbol == "wind" {
            return WxGlassTheme.cyanSoft
        }
        if symbol.contains("moon") || isNight {
            return WxGlassTheme.periwinkle
        }
        if symbol.contains("sun") {
            return WxGlassTheme.gold
        }
        return WxGlassTheme.cyan
    }

    private static func explicitSystemName(for code: String?, isNight: Bool) -> String? {
        let normalized = (code ?? "").trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalized.isEmpty else { return nil }
        switch normalized {
        case "clear", "clear-day": return isNight ? "moon.stars.fill" : "sun.max.fill"
        case "clear-night": return "moon.stars.fill"
        case "partly-cloudy", "partly_cloudy", "partly-cloudy-day": return isNight ? "cloud.moon.fill" : "cloud.sun.fill"
        case "partly-cloudy-night": return "cloud.moon.fill"
        case "cloudy": return "cloud.fill"
        case "rain": return "cloud.rain.fill"
        case "heavy-rain": return "cloud.heavyrain.fill"
        case "snow": return "cloud.snow.fill"
        case "sleet": return "cloud.sleet.fill"
        case "thunder": return "cloud.bolt.rain.fill"
        case "fog": return "cloud.fog.fill"
        case "wind": return "wind"
        default: return nil
        }
    }

    static func symbolFromDescription(_ raw: String?, isDay: Bool) -> String? {
        let desc = (raw ?? "").trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !desc.isEmpty else { return nil }

        if desc.contains("thunder") || desc.contains("t-storm") {
            return "cloud.bolt.rain.fill"
        }
        if desc.contains("snow") || desc.contains("flurr") || desc.contains("blizzard") {
            return "cloud.snow.fill"
        }
        if desc.contains("sleet") || desc.contains("ice") || desc.contains("freez") || desc.contains("wintry") {
            return "cloud.sleet.fill"
        }
        if desc.contains("heavy rain") || desc.contains("downpour") {
            return "cloud.heavyrain.fill"
        }
        if desc.contains("storm") && !desc.contains("dust") {
            return "cloud.bolt.rain.fill"
        }
        if desc.contains("rain") || desc.contains("shower") || desc.contains("drizzle") || desc.contains("sprinkle") {
            if desc.contains("chance") || desc.contains("slight") || desc.contains("isolated") || desc.contains("scattered") {
                return isDay ? "cloud.sun.rain.fill" : "cloud.moon.rain.fill"
            }
            return "cloud.rain.fill"
        }
        if desc.contains("fog") || desc.contains("mist") || desc.contains("haze") || desc.contains("smoke") || desc.contains("dust") {
            return "cloud.fog.fill"
        }
        if desc.contains("wind") || desc.contains("breez") || desc.contains("blustery") {
            if desc.contains("clear") || desc.contains("sunny") || desc.contains("fair") {
                return isDay ? "sun.max.fill" : "moon.stars.fill"
            }
            if desc.contains("partly") || desc.contains("mostly") {
                return isDay ? "cloud.sun.fill" : "cloud.moon.fill"
            }
            return "wind"
        }
        if desc.contains("overcast") || (desc.contains("cloudy") && !desc.contains("partly") && !desc.contains("mostly clear") && !desc.contains("mostly sunny")) {
            if desc.contains("mostly cloudy") || desc.contains("considerable") {
                return isDay ? "cloud.sun.fill" : "cloud.moon.fill"
            }
            return "cloud.fill"
        }
        if desc.contains("partly") || desc.contains("mostly sunny") || desc.contains("mostly clear") || desc.contains("few clouds") || desc.contains("scattered clouds") {
            return isDay ? "cloud.sun.fill" : "cloud.moon.fill"
        }
        if desc.contains("sunny") || desc.contains("clear") || desc.contains("fair") {
            return isDay ? "sun.max.fill" : "moon.stars.fill"
        }
        return nil
    }

    /// Maps forecast Period descriptions + day/night flag to cohesive SF Symbols.
    static func periodSymbol(for period: Period) -> String {
        let isDay = period.resolvedIsDaytime
        if let sym = symbolFromDescription(period.shortDescription, isDay: isDay) {
            return sym
        }
        return isDay ? "sun.max.fill" : "moon.stars.fill"
    }

    /// Maps lunar phase names to crisp SF Symbols (replacing raw OS color emojis).
    static func moonSymbol(for phase: String?) -> String {
        let lower = (phase ?? "").lowercased()
        if lower.contains("new") {
            return "moonphase.new.moon"
        }
        if lower.contains("waxing crescent") {
            return "moonphase.waxing.crescent"
        }
        if lower.contains("first quarter") {
            return "moonphase.first.quarter"
        }
        if lower.contains("waxing gibbous") || lower.contains("waxing") {
            return "moonphase.waxing.gibbous"
        }
        if lower.contains("full") {
            return "moonphase.full.moon"
        }
        if lower.contains("waning gibbous") {
            return "moonphase.waning.gibbous"
        }
        if lower.contains("last quarter") || lower.contains("third quarter") {
            return "moonphase.last.quarter"
        }
        if lower.contains("waning crescent") || lower.contains("waning") {
            return "moonphase.waning.crescent"
        }
        return "moon.stars.fill"
    }
}
