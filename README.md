# wx

A terminal weather suite and native macOS app. Fetches current conditions, forecasts, astronomical ephemerides, and active alerts from the [National Weather Service](https://www.weather.gov) (US) and [Open-Meteo](https://open-meteo.com) (Global) APIs — no API key required.

Features colorized truecolor output in the terminal, interactive Doppler radar, piped JSON for shell scripting, and a native SwiftUI macOS app with live radar maps.

---

## Screenshots

### macOS App (`wx.app`)

Full-featured dual-pane Command Console (`WX.DESK`) featuring live atmospheric telemetry, synoptic forecasts, tactical sensor controls, and interactive vector Apple Maps Doppler radar overlay.

<p align="center">
  <img src="docs/screenshots/wx-mac-desk.png" alt="wx macOS App - Atmospheric Telemetry Console" width="850">
</p>

### CLI App

Colorized terminal weather output with ASCII weather art, sun/moon astronomical ephemerides, 7-day forecast, and Unicode half-block Doppler radar.

| Forecast & Conditions (`wx --forecast`) | Doppler Radar (`wx radar --no-inline`) |
|:---:|:---:|
| <img src="docs/screenshots/wx-cli-forecast.png" alt="wx CLI Forecast & Current Conditions" width="420"> | <img src="docs/screenshots/wx-cli-radar.png" alt="wx CLI Doppler Radar" width="420"> |

---

## Installation

### macOS App (DMG Installer)

Install the native macOS SwiftUI app via disk image:

1. Download `wx.dmg` or build it locally with:
   ```bash
   make dmg
   # (or: make mac-dmg)
   ```
2. Open `build/wx.dmg` and drag **`wx.app`** into **`/Applications`**.

> **Note:** The `wx.app` bundle includes both the native macOS menu bar HUD / Telemetry Console and the embedded `wx-cli` binary inside `wx.app/Contents/MacOS/wx-cli`.

### CLI App

```bash
# Build locally
make build          # → build/wx

# Install to $GOPATH/bin
go install github.com/mwirges/wx@latest
```

## Usage

```
wx [options]

Options:
  -l, --location <value>   Zip code, "City, ST", or omit to use config/auto-detect
  -f, --forecast           Show 7-day forecast
  -H, --hourly             Show hourly forecast
  -s, --short              Single-line summary for statuslines/prompts
  -T, --template <string>  Format output with a Go template or @file
  -a, --alerts             Show active weather alerts
  -u, --units <value>      imperial (default) or metric
      --no-cache           Bypass the local cache
  -j, --json               Force JSON output even in a terminal
      --exit-code-on-alerts Exit 2 on warnings, 1 on watches/advisories, 0 on normal
      --help               Show help
      --version            Print version
```

### Examples

```bash
# Current conditions, auto-detected location
wx

# Single-line compact summary with feels-like & observation age (great for tmux / shell prompts / waybar)
wx --short
# Output: Fort Wayne, IN: 63°F (feels 60°F) · N 0 mph · 63% hum · ↑7:32 AM ↓7:32 PM 🌔 · 4m ago

# Hourly forecast (table layout with precip % and humidity)
wx --hourly
wx hourly
wx hourly --hours 12

# Custom templating (inline or from file with @)
wx -T '{{.Conditions.Location}}: {{.Conditions.TempStr}} ({{.Conditions.Description}})'
wx -T '{{.Conditions.TempF | printf "%.0f"}}°F | {{.Astronomy.MoonPhaseIcon}} {{.Astronomy.MoonPhase}} | {{.Conditions.Astronomy.Sunrise}} - {{.Conditions.Astronomy.Sunset}} [{{.Freshness.AgeString}}]'
wx -T @~/.config/wx/tmux.tmpl

# Automation / scripting with alert exit codes
wx --exit-code-on-alerts
# Exit status: 2 = Severe Warning, 1 = Watch/Advisory, 0 = Normal

# Specific city or zip
wx -l "Kansas City, MO"
wx -l 64101

# Forecast + alerts together
wx -l "Chicago, IL" --forecast --alerts

# Positional location argument or favorite alias
wx "Chicago, IL"
wx Home

# Metric units
wx -l "Denver, CO" --forecast --units metric

# JSON output (also automatic when piped)
wx -l 10001 --json
wx -l 10001 | jq .conditions.temperature_f

# Skip the cache (force a fresh fetch)
wx --no-cache
```

## Radar

```bash
wx radar                          # composite reflectivity, auto-detected location
wx radar --interactive            # full-screen interactive TUI (remembers product/zoom per location)
wx radar --save /tmp/radar.png    # save high-resolution radar image directly to PNG
wx radar --loop                   # 6-frame animated loop (Ctrl+C to exit)
wx radar --loop --frames 12 --interval 400
wx radar --product base-reflectivity
wx radar --product storm-relative-velocity
wx radar --product echo-tops
wx radar --radius 150             # km radius around location (default 200)
wx radar --station KIWX           # center on a specific NEXRAD station
wx radar --no-inline              # force half-block rendering
```

**Products:**

| Product | Description |
|---------|-------------|
| `composite-reflectivity` | National CONUS mosaic (default) |
| `base-reflectivity` | Single-station, lower-tilt scan |
| `storm-relative-velocity` | Velocity adjusted for storm motion |
| `echo-tops` | MRMS enhanced echo tops (cloud heights, kft) |
| `precip-type` | MRMS surface precipitation classification (rain, snow, ice) |
| `one-hour-precip` | 1-hour quantitative precipitation accumulation (QPE) |
| `storm-total-precip` | Storm total precipitation accumulation |

**Rendering:** wx auto-detects your terminal. In iTerm2, Kitty, Ghostty, and WezTerm it sends a full 1600×1600 PNG via inline image protocol. In all other terminals it uses Unicode half-block characters (`▀`) with ANSI truecolor. Use `--no-inline` to force half-block.

### Interactive mode (`--interactive`)

Full-screen TUI with live radar. Press `R` in `wx monitor` to open the radar panel there instead.
When exiting interactive radar, your selected product and zoom radius are automatically saved to your per-location preferences.

| Key | Action |
|-----|--------|
| `p` | Cycle product (composite → base → SRV → echo tops → precip type → 1h → storm total) |
| `+` / `-` | Zoom in / out |
| `l` | Toggle loop animation |
| `space` | Pause / resume loop |
| `←` / `→` | Step through frames manually |
| `<` / `>` | Adjust loop speed slower / faster |
| `r` | Refresh |
| `q` | Quit |

## Monitor

Full-screen live weather dashboard that refreshes automatically.

```bash
wx monitor                        # current conditions + forecast, auto-detected location
wx monitor --location "Chicago, IL"
wx monitor --units metric
wx monitor --interval 5m          # refresh interval (default 15m)
wx monitor --notify               # enable desktop notifications for new weather alerts
wx monitor --hourly               # start directly in hourly forecast mode
```

The monitor shows current conditions, active alerts, and a scrollable forecast. Press `R` to toggle a live radar panel, and `H` to toggle between 7-day and hourly forecasts.

| Key | Action |
|-----|--------|
| `R` | Toggle radar panel (splits screen left/right) |
| `H` | Toggle between hourly and daily forecast |
| `r` | Refresh weather now |
| `l` | Change location (type to filter; `tab`/`↑`/`↓` to cycle favorites & recents) |
| `↑` / `↓` | Scroll forecast |
| `q` | Quit |

## Config file

Use `wx config set` to write your preferences, or edit `~/.config/wx/config.json` directly.

```bash
# Set a default location so you never need --location
wx config set --location "Kansas City, MO"
wx config set --location 64101

# Set default units
wx config set --units metric

# Set default desktop notifications
wx config set --notifications true

# Set multiple options at once
wx config set --location "Denver, CO" --units imperial --notifications true

# Clear a value (pass empty string)
wx config set --location ""

# Show current config (includes favorites, recents, and per-location settings)
wx config
wx config show
```

`wx config show` prints the config file path and all current values. The file itself is plain JSON at `~/.config/wx/config.json`:

```json
{
  "default_location": "Kansas City, MO",
  "units": "imperial",
  "notifications": true
}
```

`default_location` accepts any value that `--location` accepts: a zip code, a `"City, ST"` string, or leave it unset to fall back to IP-based auto-detection.

**Precedence:** `--location` flag → positional arg → per-location config → `default_location` in config → IP auto-detect.
**Units precedence:** `--units` flag → per-location config → `units` in config → `imperial`.
**Notifications precedence:** `--notify` flag → `notifications` in config → `false` (disabled by default).

## Favorite Locations & Recents

Manage named favorite location aliases, customize per-location defaults, and view recent searches:

```bash
# Add favorites with optional per-location preferences
wx locations add Home "Fort Wayne, IN" --station KIWX --radar-radius 120
wx locations add Work 46802 --units metric
wx locations add "Chicago, IL"

# Configure per-location settings for an existing location or favorite
wx locations config Home --radar-product base-reflectivity --radar-radius 150
wx locations config Home              # show custom settings
wx locations config Home --clear      # reset to defaults

# Use your favorite alias anywhere
wx Home
wx Home --forecast
wx radar Home
wx monitor Home

# List favorites and recent searches (shows custom overrides)
wx locations

# View recent searches or clear history
wx locations recents
wx locations clear-recents

# Remove a favorite
wx locations rm Home
```

Favorite aliases, per-location settings, and recent searches are persisted in `~/.config/wx/config.json`:
```json
{
  "default_location": "Fort Wayne, IN",
  "units": "imperial",
  "favorites": [
    {"name": "Home", "value": "Fort Wayne, IN"},
    {"name": "Work", "value": "46802"}
  ],
  "recent_locations": [
    "Fort Wayne, Indiana",
    "Chicago, Illinois"
  ],
  "per_location": {
    "Home": {
      "radar_station": "KIWX",
      "default_radar_radius": 120,
      "default_radar_product": "base-reflectivity"
    },
    "Work": {
      "units": "metric"
    }
  }
}
```

## Cache

Responses are cached in `~/.cache/wx/` to avoid unnecessary API calls:

| Data | TTL |
|------|-----|
| Current conditions | 10 minutes |
| Forecast | 1 hour |
| Alerts | 5 minutes |
| Grid/station lookup | 24 hours |
| Geocoding (zip/city) | 24 hours |
| IP geolocation | 1 hour |

Use `--no-cache` to bypass the cache for a single invocation.

## JSON output

When stdout is not a terminal (pipe, redirect, cron job), wx automatically outputs JSON. You can also force it with `--json`.

```bash
wx -l 64101 --forecast --alerts --json | jq .
```

```json
{
  "conditions": {
    "station": "KMKC",
    "observed_at": "2026-03-22T22:10:00Z",
    "location": "Kansas City, MO",
    "description": "Clear",
    "temperature_f": 62.6,
    "temperature_c": 17,
    "humidity_pct": 39.17
  },
  "forecast": { ... },
  "alerts": [ ... ]
}
```

## Data sources

| Purpose | Service |
|---------|---------|
| US Weather | [api.weather.gov](https://api.weather.gov) (NWS) — Primary US data source, no key required |
| Global Weather | [api.open-meteo.com](https://open-meteo.com) — Global fallback & international coverage, no key required |
| Zip / city geocoding | [Nominatim](https://nominatim.openstreetmap.org) (OpenStreetMap) |
| IP geolocation | [ipinfo.io](https://ipinfo.io) (free tier) |

## Weather Providers & Architecture

`wx` uses a tiered, prioritized provider registry (`docs/provider-design.md`):

1. **National Weather Service (`nws`)**: High-fidelity data for US coordinates (`PrioritySpecialized: 100`).
2. **Open-Meteo (`openmeteo`)**: Keyless, universal fallback for international coordinates (`PriorityFallback: 10`).

Selection is automatic: US locations query NWS by default; international locations (e.g. Toronto, London, Tokyo, Paris) automatically query Open-Meteo. If NWS encounters an upstream service failure, `wx` transparently falls back to Open-Meteo.

You can explicitly force a provider via CLI or config:
```bash
wx --provider openmeteo "Fort Wayne, IN"
wx config set --provider openmeteo
```

To add another provider (e.g. Environment Canada, DWD, ECMWF), implement `provider.WeatherProvider`, register with `provider.RegisterWithPriority(...)` in `init()`, and blank-import in `cmd/app.go`. See `docs/provider-design.md` for full design documentation.

## macOS App

The repository includes a native SwiftUI application for macOS 14+ (`macos/`):

- **Menu Bar HUD (`WX.HUD`)**: Sits unobtrusively in the macOS menu bar showing live temperatures and status glyphs; clicks toggle an atmospheric telemetry popover.
- **Telemetry Console (`WX.DESK`)**: A tactical command console featuring:
  - Surface sensor telemetry (temperature, wind, humidity, dew point, pressure, visibility).
  - Astronomical ephemeris logging (sunrise/sunset, daylight duration, moon phase, age, and illumination).
  - Synoptic forecast logs.
  - Interactive Apple Maps Doppler radar array with MRMS composite and NEXRAD product selection.
- **Embedded Engine**: The app bundle automatically embeds and invokes the Go CLI binary (`wx.app/Contents/MacOS/wx-cli`).
- **DMG Distribution**: Built into a standalone, compressed disk image installer via `make dmg` or `make mac-dmg`.

## Development

```bash
make build        # build CLI for current platform (build/wx)
make test         # run Go test suite
make test-verbose # verbose test output
make vet          # go vet
make build-all    # cross-compile CLI for darwin/linux/windows
make mac-build    # build macOS SwiftUI app (macos/build/Build/Products/Debug/wx.app)
make mac-run      # build and launch macOS app
make mac-dmg      # package release macOS app into build/wx.dmg (alias: make dmg)
make clean        # remove build/
```
