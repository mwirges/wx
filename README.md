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
