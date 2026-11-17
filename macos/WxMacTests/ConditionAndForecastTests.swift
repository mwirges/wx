import AppKit
import SwiftUI
import XCTest

final class ConditionAndForecastTests: XCTestCase {
    func testNighttimeASOSWithoutConditionCodeUsesLunarSymbolAndForecastDescription() throws {
        let json = """
        {
          "conditions": {
            "station": "KFWA",
            "location": "New Haven, IN",
            "temperature_f": 36,
            "temperature_c": 2.2,
            "astronomy": {
              "solar_elevation_deg": -56.2,
              "current_period": "Night"
            }
          },
          "forecast": {
            "periods": [
              {
                "name": "Tonight",
                "is_daytime": false,
                "temperature_f": 32,
                "temperature_c": 0,
                "short_description": "Clear"
              }
            ],
            "hourly": [
              {
                "name": "11 PM",
                "is_daytime": false,
                "temperature_f": 36,
                "temperature_c": 2,
                "short_description": "Mostly Clear"
              }
            ]
          },
          "alerts": []
        }
        """
        let payload = try JSONDecoder().decode(WxPayload.self, from: Data(json.utf8))

        XCTAssertTrue(ConditionSymbol.isNight(for: payload))
        let symbol = ConditionSymbol.resolvedSystemName(for: payload)
        XCTAssertEqual(symbol, "cloud.moon.fill")
        XCTAssertEqual(ConditionSymbol.resolvedDescription(for: payload), "Mostly Clear")
        XCTAssertEqual(ConditionSymbol.tint(for: symbol, isNight: true), WxGlassTheme.periwinkle)
    }

    func testExpiredLeadHourIsSkippedForHourlyStripAndConditionFallback() throws {
        let json = """
        {
          "conditions": {
            "station": "KFWA",
            "location": "New Haven, IN",
            "temperature_f": 34,
            "astronomy": {
              "solar_elevation_deg": -66.9,
              "current_period": "Night"
            }
          },
          "forecast": {
            "periods": [
              {
                "name": "Tonight",
                "is_daytime": false,
                "temperature_f": 32,
                "temperature_c": 0,
                "short_description": "Partly Cloudy"
              },
              {
                "name": "Tuesday",
                "is_daytime": true,
                "temperature_f": 54,
                "temperature_c": 12.2,
                "short_description": "Sunny"
              }
            ],
            "hourly": [
              {
                "name": "11 PM",
                "start_time": "2026-11-16T23:00:00-05:00",
                "is_daytime": false,
                "temperature_f": 36,
                "temperature_c": 2.2,
                "short_description": "Mostly Cloudy"
              },
              {
                "name": "12 AM",
                "start_time": "2026-11-17T00:00:00-05:00",
                "is_daytime": false,
                "temperature_f": 34,
                "temperature_c": 1.1,
                "short_description": "Clear"
              },
              {
                "name": "1 AM",
                "start_time": "2026-11-17T01:00:00-05:00",
                "is_daytime": false,
                "temperature_f": 33,
                "temperature_c": 0.6,
                "short_description": "Clear"
              }
            ]
          },
          "alerts": []
        }
        """
        let payload = try JSONDecoder().decode(WxPayload.self, from: Data(json.utf8))
        let iso = ISO8601DateFormatter()
        let nowAfterMidnight = try XCTUnwrap(iso.date(from: "2026-11-17T00:22:00-05:00"))

        let active = try XCTUnwrap(payload.forecast?.activeHourlyPeriods(at: nowAfterMidnight))
        XCTAssertEqual(active.map(\.name), ["12 AM", "1 AM"])
        XCTAssertEqual(ConditionSymbol.resolvedDescription(for: payload, now: nowAfterMidnight), "Clear")
        XCTAssertEqual(ConditionSymbol.resolvedSystemName(for: payload, now: nowAfterMidnight), "moon.stars.fill")

        // Static past fixtures (all hours elapsed) must preserve all hours so UI tests keep working.
        let wayLater = try XCTUnwrap(iso.date(from: "2026-12-01T12:00:00Z"))
        let preserved = try XCTUnwrap(payload.forecast?.activeHourlyPeriods(at: wayLater))
        XCTAssertEqual(preserved.count, 3)

        // Lead High/Low at night must NOT report Tuesday's daytime 54° as tonight's High.
        let hiLo = try XCTUnwrap(payload.forecast?.leadHighLow(units: "imperial"))
        XCTAssertNil(hiLo.highText)
        XCTAssertEqual(hiLo.lowText, "32°")
    }

    func testZeroDegreeTemperatureFallbackConvertsUnitsCleanly() {
        let freezingOnlyF = Period(name: "Tonight", isDaytime: false, temperatureF: 32, temperatureC: nil)
        XCTAssertEqual(freezingOnlyF.formattedTemperature(units: "metric"), "0°")
        XCTAssertEqual(freezingOnlyF.formattedTemperature(units: "imperial"), "32°")

        let zeroFahrenheitOnlyC = Period(name: "Arctic Night", isDaytime: false, temperatureF: nil, temperatureC: -17.7778)
        XCTAssertEqual(zeroFahrenheitOnlyC.formattedTemperature(units: "imperial"), "0°")
        XCTAssertEqual(zeroFahrenheitOnlyC.formattedTemperature(units: "metric"), "-18°")
    }

    func testAtmosphereRegimeRespectsPolarNightAndNightForecastWithoutAstronomy() {
        let polarAstro = AstronomyDTO(
            isPolarNight: true,
            solarElevationDeg: -2.0,
            currentPeriod: "Golden Hour"
        )
        XCTAssertEqual(WxAtmosphereRegime.resolve(conditions: nil, astronomy: polarAstro), .orbitalNight)

        let nightForecast = Forecast(periods: [Period(name: "Tonight", isDaytime: false, temperatureF: 32)])
        XCTAssertEqual(WxAtmosphereRegime.resolve(conditions: nil, forecast: nightForecast), .orbitalNight)
    }

    func testClearNightASOSResolvesMoonStarsAndDailyFallbackWhenHourlyMissing() throws {
        let json = """
        {
          "conditions": {
            "station": "KFWA",
            "location": "New Haven, IN",
            "description": "   ",
            "temperature_f": 34,
            "astronomy": {
              "solar_elevation_deg": -18.4,
              "current_period": "Astronomical Twilight"
            }
          },
          "forecast": {
            "periods": [
              {
                "name": "Tonight",
                "is_daytime": false,
                "temperature_f": 30,
                "short_description": "Clear"
              }
            ]
          },
          "alerts": []
        }
        """
        let payload = try JSONDecoder().decode(WxPayload.self, from: Data(json.utf8))

        XCTAssertTrue(ConditionSymbol.isNight(for: payload))
        let symbol = ConditionSymbol.resolvedSystemName(for: payload)
        XCTAssertEqual(symbol, "moon.stars.fill")
        XCTAssertEqual(ConditionSymbol.resolvedDescription(for: payload), "Clear")
        XCTAssertEqual(ConditionSymbol.tint(for: symbol, isNight: true), WxGlassTheme.periwinkle)
    }

    func testActiveSnowNowcastOverridesNilASOSConditionCode() throws {
        let json = """
        {
          "conditions": {
            "station": "KFWA",
            "temperature_f": 28,
            "astronomy": {
              "solar_elevation_deg": -30.0,
              "current_period": "Night"
            },
            "nowcast": {
              "is_active_precip": true,
              "primary_phase": "snow",
              "headline": "Light snow continuing"
            }
          },
          "alerts": []
        }
        """
        let payload = try JSONDecoder().decode(WxPayload.self, from: Data(json.utf8))

        let symbol = ConditionSymbol.resolvedSystemName(for: payload)
        XCTAssertEqual(symbol, "cloud.snow.fill")
        XCTAssertEqual(ConditionSymbol.tint(for: symbol, isNight: true), WxGlassTheme.cyanSoft)
    }

    func testDaytimeFixturePreservesConditionDescriptionAndGoldTint() throws {
        let json = """
        {
          "conditions": {
            "station": "KFWA",
            "location": "New Haven, IN",
            "description": "Fixture Partly Cloudy",
            "condition_code": "partly-cloudy-day",
            "temperature_f": 72
          },
          "alerts": []
        }
        """
        let payload = try JSONDecoder().decode(WxPayload.self, from: Data(json.utf8))

        XCTAssertFalse(ConditionSymbol.isNight(for: payload))
        let symbol = ConditionSymbol.resolvedSystemName(for: payload)
        XCTAssertEqual(symbol, "cloud.sun.fill")
        XCTAssertEqual(ConditionSymbol.resolvedDescription(for: payload), "Fixture Partly Cloudy")
        XCTAssertEqual(ConditionSymbol.tint(for: symbol, isNight: false), WxGlassTheme.gold)
    }

    func testSynopticRowsKeepUnpairedTonightAndTrailingDayCleanlyTyped() throws {
        let periods: [Period] = [
            Period(name: "Tonight", isDaytime: false, temperatureF: 32, temperatureC: 0, shortDescription: "Mostly Clear"),
            Period(name: "Wednesday", isDaytime: true, temperatureF: 54, temperatureC: 12, shortDescription: "Sunny", probabilityOfPrecipitation: 10),
            Period(name: "Wednesday Night", isDaytime: false, temperatureF: 36, temperatureC: 2, shortDescription: "Partly Cloudy", probabilityOfPrecipitation: 25),
            Period(name: "Thursday", isDaytime: true, temperatureF: 62, temperatureC: 17, shortDescription: "Mostly Sunny"),
        ]
        let forecast = Forecast(periods: periods)
        let rows = forecast.synopticRows(units: "imperial")

        XCTAssertEqual(rows.count, 3)

        // Lead nighttime period must be .nightOnly with no fake highValue/highText
        XCTAssertEqual(rows[0].name, "Tonight")
        XCTAssertEqual(rows[0].kind, .nightOnly)
        XCTAssertEqual(rows[0].lowValue, 32)
        XCTAssertNil(rows[0].highValue)
        XCTAssertEqual(rows[0].lowText, "32°")
        XCTAssertNil(rows[0].highText)

        // Paired Wednesday + Wednesday Night
        XCTAssertEqual(rows[1].name, "Wednesday")
        XCTAssertEqual(rows[1].kind, .paired)
        XCTAssertEqual(rows[1].lowValue, 36)
        XCTAssertEqual(rows[1].highValue, 54)
        XCTAssertEqual(rows[1].lowText, "36°")
        XCTAssertEqual(rows[1].highText, "54°")
        XCTAssertEqual(rows[1].precipPct, 25)

        // Trailing unpaired daytime period must be .dayOnly with no fake lowValue/lowText
        XCTAssertEqual(rows[2].name, "Thursday")
        XCTAssertEqual(rows[2].kind, .dayOnly)
        XCTAssertNil(rows[2].lowValue)
        XCTAssertEqual(rows[2].highValue, 62)
        XCTAssertNil(rows[2].lowText)
        XCTAssertEqual(rows[2].highText, "62°")

        // Metric check
        let metricRows = forecast.synopticRows(units: "metric")
        XCTAssertEqual(metricRows[0].lowText, "0°")
        XCTAssertEqual(metricRows[1].lowText, "2°")
        XCTAssertEqual(metricRows[1].highText, "12°")
    }

    @MainActor
    func testMacViewsRenderAndFitCleanlyInHostingViews() throws {
        let json = """
        {
          "conditions": {
            "station": "KFWA",
            "observed_at": "2026-11-17T04:55:00Z",
            "location": "New Haven, IN",
            "temperature_f": 34,
            "temperature_c": 1.1,
            "feels_like_f": 31,
            "feels_like_c": -0.6,
            "dew_point_f": 32,
            "dew_point_c": 0.0,
            "humidity_pct": 93,
            "wind_mph": 6,
            "wind_kph": 10,
            "wind_direction": "SSE",
            "pressure_inhg": 30.12,
            "pressure_hpa": 1020,
            "visibility_mi": 10,
            "visibility_m": 16093,
            "astronomy": {
              "sunrise": "2026-11-17T12:30:00Z",
              "sunset": "2026-11-17T22:20:00Z",
              "day_length": "9h 50m",
              "solar_elevation_deg": -66.9,
              "solar_azimuth_deg": 341.7,
              "current_period": "Night",
              "moon_phase": "First Quarter",
              "moon_illumination_pct": 47.3
            },
            "air_quality": {
              "aqi": 42,
              "category": "Good",
              "uv_index": 0.0,
              "uv_category": "Low"
            }
          },
          "forecast": {
            "periods": [
              { "name": "Tonight", "is_daytime": false, "temperature_f": 32, "temperature_c": 0, "short_description": "Mostly Clear" },
              { "name": "Wednesday", "is_daytime": true, "temperature_f": 54, "temperature_c": 12.2, "short_description": "Rain Likely", "probability_of_precipitation": 100 },
              { "name": "Wednesday Night", "is_daytime": false, "temperature_f": 36, "temperature_c": 2.2, "short_description": "Showers", "probability_of_precipitation": 80 },
              { "name": "Thursday", "is_daytime": true, "temperature_f": 59, "temperature_c": 15.0, "short_description": "Sunny", "probability_of_precipitation": 10 },
              { "name": "Thursday Night", "is_daytime": false, "temperature_f": 38, "temperature_c": 3.3, "short_description": "Clear", "probability_of_precipitation": 0 }
            ],
            "hourly": [
              { "name": "12 AM", "is_daytime": false, "temperature_f": 34, "temperature_c": 1.1, "short_description": "Mostly Clear", "probability_of_precipitation": 0 },
              { "name": "1 AM", "is_daytime": false, "temperature_f": 33, "temperature_c": 0.6, "short_description": "Clear", "probability_of_precipitation": 0 },
              { "name": "2 AM", "is_daytime": false, "temperature_f": 32, "temperature_c": 0.0, "short_description": "Clear", "probability_of_precipitation": 15 },
              { "name": "3 AM", "is_daytime": false, "temperature_f": 32, "temperature_c": 0.0, "short_description": "Partly Cloudy", "probability_of_precipitation": 20 },
              { "name": "4 AM", "is_daytime": false, "temperature_f": 31, "temperature_c": -0.6, "short_description": "Partly Cloudy", "probability_of_precipitation": 0 }
            ]
          },
          "alerts": [
            {
              "event": "Winter Weather Advisory",
              "headline": "Patchy freezing drizzle possible overnight",
              "severity": "Moderate",
              "expires": "2026-11-17T12:00:00Z"
            }
          ]
        }
        """
        let payload = try JSONDecoder().decode(WxPayload.self, from: Data(json.utf8))
        let store = WeatherStore()
        store.payload = payload
        store.locationInput = "New Haven, IN"
        store.lastRefreshed = Date()
        store.gridCards = [
            LocationGridCardData(
                locationKey: "New Haven, IN",
                displayName: "New Haven, IN",
                payload: payload,
                isLoading: false,
                errorMessage: nil,
                lastUpdated: Date()
            )
        ]

        // Verify WeatherStore nighttime ASOS resolution without requiring WindowServer
        XCTAssertTrue(store.isNight)
        XCTAssertEqual(store.statusSymbol, "cloud.moon.fill")
        XCTAssertEqual(store.resolvedConditionDescription, "Mostly Clear")
        XCTAssertEqual(store.displayTemp, "34°")
        XCTAssertEqual(store.tacticalStatusText, "[FWA] 34° ↑6mph")
        XCTAssertEqual(WxTheme.popoverSize(for: .compact), CGSize(width: 360, height: 380))

        // Instantiate view structs directly (verifies full view-body construction & types)
        _ = MenuBarGridCardView(card: store.gridCards[0], units: "imperial", isCurrent: true, onSelect: {})
        _ = PopoverView().environmentObject(store)
        _ = DeskWindowView().environmentObject(store)
        _ = RadarTransportBar(compact: false).environmentObject(store)

        try XCTSkipIf(
            Bundle.main.bundleURL.pathExtension != "app",
            "Skipping AppKit NSHostingView bitmap pass when running in unhosted xctest CLI"
        )

        let captureDir = ProcessInfo.processInfo.environment["WX_CAPTURE_MAC_SHOTS_DIR"]

        // 1. MenuBarGridCardView inside fixed 300x44 NSMenu hosting slot
        let menuCard = MenuBarGridCardView(
            card: store.gridCards[0],
            units: "imperial",
            isCurrent: true,
            onSelect: {}
        )
        let menuFitting = try renderAndMeasure(menuCard, size: CGSize(width: 300, height: 44), name: "menubar-card", captureDir: captureDir)
        XCTAssertLessThanOrEqual(menuFitting.height, 44.5, "MenuBarGridCardView overflowed 44pt NSMenu slot: \(menuFitting)")

        // 2. PopoverView in .compact, .standard, and .tactical
        for fmt in MenuBarFormat.allCases {
            store.menuBarFormat = fmt
            let target = WxTheme.popoverSize(for: fmt)
            let pop = PopoverView().environmentObject(store)
            let fit = try renderAndMeasure(pop, size: target, name: "popover-\(fmt.rawValue)", captureDir: captureDir)
            XCTAssertEqual(fit.width, target.width, accuracy: 1.0)
            XCTAssertEqual(fit.height, target.height, accuracy: 1.0)
        }

        // 3. DeskWindowView in Dual (940x760) and Radar (940x760)
        store.selectedDeskTab = .dual
        _ = try renderAndMeasure(
            DeskWindowView().environmentObject(store),
            size: CGSize(width: 940, height: 760),
            name: "desk-dual-night",
            captureDir: captureDir
        )

        store.selectedDeskTab = .radar
        _ = try renderAndMeasure(
            DeskWindowView().environmentObject(store),
            size: CGSize(width: 940, height: 760),
            name: "desk-radar-night",
            captureDir: captureDir
        )
    }

    @MainActor
    private func renderAndMeasure<V: View>(
        _ view: V,
        size: CGSize,
        name: String,
        captureDir: String?
    ) throws -> CGSize {
        let rect = NSRect(origin: .zero, size: size)
        let hosting = NSHostingView(rootView: view)
        hosting.frame = rect
        let window = NSWindow(
            contentRect: NSRect(x: -4000, y: -4000, width: size.width, height: size.height),
            styleMask: [.borderless],
            backing: .buffered,
            defer: false
        )
        window.appearance = NSAppearance(named: .darkAqua)
        window.contentView = hosting
        hosting.layoutSubtreeIfNeeded()
        let fitting = hosting.fittingSize
        XCTAssertTrue(fitting.width.isFinite && fitting.height.isFinite && fitting.width > 0 && fitting.height > 0)

        let rep = try XCTUnwrap(hosting.bitmapImageRepForCachingDisplay(in: hosting.bounds))
        hosting.cacheDisplay(in: hosting.bounds, to: rep)
        XCTAssertGreaterThan(rep.pixelsWide, 0)
        XCTAssertGreaterThan(rep.pixelsHigh, 0)

        if let dir = captureDir, !dir.isEmpty {
            try FileManager.default.createDirectory(atPath: dir, withIntermediateDirectories: true)
            let data = try XCTUnwrap(rep.representation(using: .png, properties: [:]))
            let path = (dir as NSString).appendingPathComponent("\(name).png")
            try data.write(to: URL(fileURLWithPath: path))
        }
        window.orderOut(nil)
        return fitting
    }
}
