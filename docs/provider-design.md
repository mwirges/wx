# wx Provider Architecture & Strategy (v2)

This document formalizes the weather provider architecture for `wx`, establishing the design principles, interface contract, priority tiers, fallback mechanics, and guidelines for adding new providers.

---

## 1. Design Philosophy

`wx` is built around four non-negotiable principles:

1. **Zero Configuration, Zero API Keys, Zero Cost:**
   Users must never be asked to sign up for an account, generate an API key, enter billing details, or configure credentials. Any weather data source integrated into `wx` must be publicly accessible and free for individual use.
2. **NWS-First for the United States:**
   The US National Weather Service (`weather.gov`) remains the primary, authoritative data source for all United States locations. It provides high-accuracy radar, localized gridded forecasts, and official CAP/SAME alerts without rate limits or commercial paywalls.
3. **Seamless Global Fallback:**
   When a user queries coordinates or locations outside the United States (or when NWS experiences an upstream outage), `wx` must automatically and transparently fall back to an authoritative, keyless global provider (e.g., Open-Meteo) without failing or requiring the user to specify flags.
4. **Pure Go Business Logic & SI Internal Representation:**
   All normalization, data modeling, ephemeris calculations, and unit conversions remain strictly in Go. Providers fetch upstream data and normalize into `wx`'s standard SI models (`°C`, `km/h`, `hPa`, `meters`). Unit conversions to imperial (`°F`, `mph`, `inHg`, `miles`) occur exclusively at display/render time.

---

## 2. Provider Interface Contract

All weather data sources implement the `provider.WeatherProvider` interface defined in `internal/provider/provider.go`:

```go
type WeatherProvider interface {
    // Name returns a unique, lowercase provider identifier (e.g. "nws", "openmeteo").
    Name() string

    // Supports returns true if the provider can serve data for the given location.
    Supports(loc location.Location) bool

    // CurrentConditions fetches the latest observed weather conditions.
    CurrentConditions(ctx context.Context, loc location.Location, c *cache.Cache) (*models.CurrentConditions, error)

    // Forecast fetches the 7-day or hourly forecast. When hourly is true, periods represent hourly slices.
    Forecast(ctx context.Context, loc location.Location, hourly bool, c *cache.Cache) (*models.Forecast, error)

    // Alerts fetches active weather alerts/watches/warnings. Returns an empty slice if unsupported or none active.
    Alerts(ctx context.Context, loc location.Location, c *cache.Cache) ([]models.Alert, error)
}
```

### Data Normalization Rules

Providers must adhere to these structural conventions:
- **Temperature & Dewpoint:** Normalized in Celsius (`°C`).
- **Wind Speed & Gusts:** Normalized in kilometers per hour (`km/h`).
- **Pressure:** Atmospheric pressure at mean sea level in hectopascals (`hPa`).
- **Visibility:** Distance in meters (`m`).
- **Condition Code:** A normalized slug string mapped in `output/icons.go` (`clear-day`, `clear-night`, `partly-cloudy-day`, `partly-cloudy-night`, `cloudy`, `rain`, `heavy-rain`, `snow`, `sleet`, `thunder`, `fog`, `wind`).
- **Alerts:** For providers without a Common Alerting Protocol (CAP) or government alert feed, `Alerts(...)` returns `[]models.Alert{}, nil`.

---

## 3. Provider Registry & Priority Tiers

The provider registry manages available data sources and their selection hierarchy using priority tiers:

```
  ┌────────────────────────────────────────────────────────┐
  │                   Location Resolution                  │
  │               (US or International)                    │
  └──────────────────────────┬─────────────────────────────┘
                             │
                             ▼
  ┌────────────────────────────────────────────────────────┐
  │                 Provider Registry                      │
  │                                                        │
  │  Priority 100: Specialized (e.g. NWS for US)           │
  │  Priority  50: Regional / National                     │
  │  Priority  10: Global Fallback (e.g. Open-Meteo)       │
  └──────────────────────────┬─────────────────────────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
    [loc.CountryCode == "US"]     [loc.CountryCode != "US"]
              │                             │
              ▼                             ▼
       NWS Provider               Open-Meteo Provider
    (with Open-Meteo fallback)          (Global)
```

### Priority Constants

```go
const (
    PrioritySpecialized = 100 // High-fidelity national/regional services (e.g. NWS for US)
    PriorityStandard    = 50  // Secondary or specialized sources
    PriorityFallback    = 10  // Broad global fallback sources (e.g. Open-Meteo)
)
```

Providers register via `provider.RegisterWithPriority(p, priority)` inside their package `init()` function and are blank-imported in `cmd/app.go`.

---

## 4. Provider Selection & Fallback Semantics

Selection precedence is strictly defined:

1. **Explicit CLI Flag:** `--provider <name>` (or `-p <name>`) forces the requested provider. If the requested provider does not support the location (e.g. `--provider nws` for Paris, France), `wx` exits with a descriptive error.
2. **Per-Location Config Setting:** `per_location.<location>.provider` configured in `~/.config/wx/config.json`.
3. **Global Config Setting:** `provider` configured in `~/.config/wx/config.json`.
4. **Automatic Priority Resolution:** `provider.ForLocation(loc)` selects the highest-priority registered provider returning `Supports(loc) == true`.
5. **Runtime Failure Fallback:** If the primary provider encounters an upstream network or HTTP 5xx failure during live data acquisition, `wx` transparently attempts `provider.FallbackForLocation(loc, failedProvider)` to query the next available provider, outputting a subtle warning to `stderr` without failing the user's command.

---

## 5. Open-Meteo Adapter Architecture

Open-Meteo (`https://api.open-meteo.com`) serves as the reference global fallback provider:

- **Keyless & Open:** Free access under Attribution 4.0 International (CC BY 4.0) for non-commercial/open-source use.
- **Universal Lat/Lon Support:** Provides seamless weather for any coordinate pair on Earth.
- **Native SI Units:** Returns temperatures in °C, wind speeds in km/h, and pressures in hPa, avoiding intermediary conversion round-trips.
- **WMO Code Mapping:** Open-Meteo emits standard WMO (World Meteorological Organization) weather interpretation codes (0–99), which `internal/provider/openmeteo/wmo.go` maps directly to `wx` condition codes and human descriptions.
- **Day/Night Period Synthesis:** For standard 7-day daily forecasts, Open-Meteo daily aggregates (`temperature_2m_max`, `temperature_2m_min`, dominant wind, max PoP) are split into alternating Day and Night periods ("Today", "Tonight", "Monday", "Monday Night", ...), matching the familiar NWS visual rhythm in terminal and JSON outputs.

---

## 6. Extending wx: Adding New Providers

To add a new provider (e.g., Environment Canada, Deutscher Wetterdienst, ECMWF):

1. Create package `internal/provider/<provider_name>/`.
2. Implement `provider.WeatherProvider`.
3. In `init()`, call:
   ```go
   func init() {
       provider.RegisterWithPriority(New(), provider.PrioritySpecialized) // or PriorityStandard
   }
   ```
4. Add blank import in `cmd/app.go`:
   ```go
   _ "github.com/mwirges/wx/internal/provider/<provider_name>"
   ```
5. Ensure tests in `internal/provider/<provider_name>/` utilize `httptest.NewServer` to stay stateless and offline.
