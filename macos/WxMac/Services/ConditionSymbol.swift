import SwiftUI

enum ConditionSymbol {
    static func systemName(for code: String?) -> String {
        switch (code ?? "").lowercased() {
        case "clear-day": return "sun.max.fill"
        case "clear-night": return "moon.stars.fill"
        case "partly-cloudy-day": return "cloud.sun.fill"
        case "partly-cloudy-night": return "cloud.moon.fill"
        case "cloudy": return "cloud.fill"
        case "rain": return "cloud.rain.fill"
        case "heavy-rain": return "cloud.heavyrain.fill"
        case "snow": return "cloud.snow.fill"
        case "sleet": return "cloud.sleet.fill"
        case "thunder": return "cloud.bolt.rain.fill"
        case "fog": return "cloud.fog.fill"
        case "wind": return "wind"
        default: return "cloud.sun.fill"
        }
    }
}
