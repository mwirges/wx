# wx — Claude Code project guide

`wx` is a Go CLI tool that fetches and displays current weather conditions,
forecasts, and active alerts. It targets `weather.gov` (NWS) for US locations,
with an extensible provider architecture for adding additional data sources.

## Commands

```bash
make build        # compile → build/wx (current OS/arch)
make build-all    # cross-compile for darwin/linux/windows
make test         # go test ./... -count=1
make test-verbose # same with -v
make vet          # go vet ./...
make clean        # remove build/
```

**Always run `make test` after any code change.**

## Usage

```bash
wx                             # current conditions, auto-detect location
wx --forecast                  # include 7-day forecast
wx --alerts                    # include active alerts
wx --location "Kansas City, MO"
wx --location 64101
wx --units metric
wx --json                      # force JSON (default when not a TTY)
wx --no-cache                  # bypass cache

wx config                      # show current config + file path
wx config set --location "Chicago, IL"
wx config set --units metric
```

## Package layout

```
main.go                        entry point → cmd.NewApp().Run()
cmd/
  app.go                       CLI wiring (urfave/cli v2), main action
  config_cmd.go                `wx config` subcommand

internal/
  cache/       cache.go        In-memory TTL cache; NewNoOp() for tests
  config/      config.go       Load/Save ~/.config/wx/config.json
  location/
    location.go                Location struct + Resolve(input string) logic
    ipgeo.go                   IP-based auto-detect (ip-api.com)
    geocode.go                 Nominatim geocoding for "City, ST" / zip
  models/
    conditions.go              CurrentConditions — all SI units, nil = missing
    forecast.go                Forecast + ForecastPeriod
    alert.go                   Alert
  output/
    output.go                  Render() — picks pretty vs JSON by TTY
    pretty.go                  Colorized output via lipgloss; icon + data layout
    icons.go                   ASCII weather icons (5 lines × 11 chars, per-line color)
    json.go                    JSON output with unit conversions
  provider/
    provider.go                WeatherProvider interface + global registry
    nws/                       NWS (weather.gov) implementation
      nws.go                   Provider struct + Register() in init()
      client.go                HTTP client with retries
      points.go                /points + /stations grid resolution (cached 24h)
      observe.go               /observations/latest → CurrentConditions (cached 10m)
      forecast.go              /forecast → Forecast (cached 1h)
      alerts.go                /alerts/active → []Alert (cached 5m)
```

## Radar (`wx radar`)

```bash
wx radar                        # current composite reflectivity
wx radar --interactive          # interactive mode: keyboard controls for product, zoom, loop
wx radar --loop                 # 6-frame animated loop (Ctrl+C to exit)
wx radar --loop --frames 12 --interval 400
wx radar --product base-reflectivity
wx radar --radius 150           # km radius around location
wx radar --station KIWX         # center on a specific NEXRAD station
wx radar --no-inline            # force half-block rendering
```

### Interactive mode (`--interactive / -i`)

Full-screen TUI powered by bubbletea. Keyboard shortcuts:

| Key   | Action |
|-------|--------|
| `p`   | Cycle radar product (composite / base reflectivity) |
| `+/-` | Zoom in / out (50–500 km radius presets) |
| `l`   | Toggle loop animation |
| `←/→` | Step through loop frames manually |
| `r`   | Refresh current data |
| `q`   | Quit |

### Data sources

- **Current + loop frames** — Iowa State IEM radmap (`mesonet.agron.iastate.edu`), 5-min intervals, with geographic overlay layers (state borders, county lines, city labels, interstates)
- **Station lookup** — NWS API (`api.weather.gov/radar/stations/{ID}`)

### Rendering

- **Half-block mode** (default): `▀` characters with ANSI truecolor (fg = top pixel, bg = bottom pixel), giving 2× vertical resolution. Programmatic city labels (~90 major US cities) overlaid in white text. Works in all truecolor terminals.
- **Inline image mode** (auto-detected): sends full 1600×1600 PNG via iTerm2 or Kitty graphics protocol. Supported by iTerm2, Kitty, Ghostty, WezTerm. Use `--no-inline` to force half-block.
- Source images are 1600×1600 from IEM for high-quality rendering in both modes.

### Radar caching (two-tier)

`image.Image` cannot be JSON-serialised through `cache.Cache`, so radar uses two layers:

| Layer | Type | Scope | TTL |
|-------|------|-------|-----|
| L1 | In-process `sync.RWMutex` map | Single invocation | 5 min (current) / forever (historical) |
| L2 | `cache.Cache` disk (PNG bytes as base64 JSON) | Across invocations | 5 min (current) / 24 h (historical) |

Both layers live in `RadarProvider`. L1 avoids re-decoding PNG during loop animation; L2 avoids re-fetching the same frame on a subsequent `wx radar` call within the TTL.

### Adding a new radar product

1. Add a `Product` const in `internal/radar/radar.go`
2. Add the WMS layer name to `nwsWMSLayers` in `internal/provider/nws/radar.go`
3. Add the IEM product code to `iemProducts` (for loop support)

## Key design decisions

### Adding a new weather provider

1. Create `internal/provider/yourprovider/` with a package that implements
   `provider.WeatherProvider`.
2. Call `provider.Register(...)` from the package's `init()` function.
3. Import the package with `_` in `cmd/app.go` (see the existing NWS import).
4. `Supports()` should return `true` only for locations the provider can serve
   (NWS checks `loc.CountryCode == "US"`).

### CurrentConditions model

All measurements are stored in SI units:
- Temperature, dew point, wind chill, heat index → **Celsius**
- Wind speed, gusts → **km/h**
- Pressure → **hPa** (sea-level)
- Visibility → **meters**
- Humidity → percent 0–100

All measurement fields are `*float64`; `nil` means the station did not report
that value. `ConditionCode` is a normalized string (see `icons.go` for the
known codes) used to select the ASCII icon.

### Output routing

`output.Render()` checks whether stdout is a TTY
(`golang.org/x/term.IsTerminal`). TTY → pretty; pipe/redirect → JSON.
`--json` overrides to always produce JSON. Unit conversions happen at render
time; the model always stores SI.

### Icon system

Icons live in `output/icons.go` as `weatherIcon{lines [5]string, colors [5]lipgloss.Color}`.
Each line is ≤11 visible chars; `lipgloss.Style.Width(iconWidth)` pads to 13
at render time. To add an icon: add a new entry to `conditionIcons` and map
to it from `mapNWSIconCode` (or the equivalent in a future provider).

### Config file

`~/.config/wx/config.json` — fields: `default_location`, `units`.
Written atomically via temp-file + rename (`config.Save`).
Precedence for all settings: **CLI flag > config file > built-in default**.

## Testing conventions

- Every new function or package gets tests in `_test.go` alongside the code.
- Provider tests use `httptest.NewServer` with a mock NWS mux — no real HTTP.
- Output tests use `captureStdout` (pipe trick in `json_test.go`) for JSON,
  and direct unit tests for formatter functions in `pretty_test.go`.
- Cache tests always use `cache.NewNoOp()` to stay stateless.
- `make test` must pass before any commit.
