# wx

A terminal weather tool for the US. Fetches current conditions, forecasts, and active alerts from the [National Weather Service](https://www.weather.gov) API — no API key required.

Colorized output when run in a terminal; JSON when piped.

## Installation

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
  -a, --alerts             Show active weather alerts
  -u, --units <value>      imperial (default) or metric
      --no-cache           Bypass the local cache
  -j, --json               Force JSON output even in a terminal
      --help               Show help
      --version            Print version
```

### Examples

```bash
# Current conditions, auto-detected location
wx

# Specific city or zip
wx -l "Kansas City, MO"
wx -l 64101

# Forecast + alerts together
wx -l "Chicago, IL" --forecast --alerts

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
wx radar --interactive            # full-screen interactive TUI
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

**Rendering:** wx auto-detects your terminal. In iTerm2, Kitty, Ghostty, and WezTerm it sends a full 1600×1600 PNG via inline image protocol. In all other terminals it uses Unicode half-block characters (`▀`) with ANSI truecolor. Use `--no-inline` to force half-block.

### Interactive mode (`--interactive`)

Full-screen TUI with live radar. Press `R` in `wx monitor` to open the radar panel there instead.

| Key | Action |
|-----|--------|
| `p` | Cycle product (composite → base refl → SRV → echo tops) |
| `+` / `-` | Zoom in / out |
| `l` | Toggle loop animation |
| `space` | Pause / resume loop |
| `←` / `→` | Step through frames manually |
| `r` | Refresh |
| `q` | Quit |

## Monitor

Full-screen live weather dashboard that refreshes automatically.

```bash
wx monitor                        # current conditions + forecast, auto-detected location
wx monitor --location "Chicago, IL"
wx monitor --units metric
wx monitor --interval 5m          # refresh interval (default 15m)
```

The monitor shows current conditions, active alerts, and a scrollable 7-day forecast. Press `R` to toggle a live radar panel alongside the weather data.

| Key | Action |
|-----|--------|
| `R` | Toggle radar panel (splits screen left/right) |
| `r` | Refresh weather now |
| `l` | Change location |
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

# Set both at once
wx config set --location "Denver, CO" --units imperial

# Clear a value (pass empty string)
wx config set --location ""

# Show current config
wx config
wx config show
```

`wx config show` prints the config file path and all current values. The file itself is plain JSON at `~/.config/wx/config.json`:

```json
{
  "default_location": "Kansas City, MO",
  "units": "imperial"
}
```

`default_location` accepts any value that `--location` accepts: a zip code, a `"City, ST"` string, or leave it unset to fall back to IP-based auto-detection.

**Precedence:** `--location` flag → `default_location` in config → IP auto-detect.
**Units precedence:** `--units` flag → `units` in config → `imperial`.

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
| Weather data | [api.weather.gov](https://api.weather.gov) (NWS) — US only, no key required |
| Zip / city geocoding | [Nominatim](https://nominatim.openstreetmap.org) (OpenStreetMap) |
| IP geolocation | [ipinfo.io](https://ipinfo.io) (free tier) |

## Adding a weather provider

The provider interface supports adding non-US data sources. Create a package under `internal/provider/` that implements `provider.WeatherProvider`, set `Supports()` to return `true` for the target country codes, and register it from `init()`:

```go
func init() {
    provider.Register(New())
}
```

Import the package with a blank import in `cmd/app.go` and it will be selected automatically for non-US locations.

## Development

```bash
make build        # build for current platform
make test         # run tests
make test-verbose # verbose test output
make vet          # go vet
make build-all    # cross-compile for darwin/linux/windows
make clean        # remove build/
```
