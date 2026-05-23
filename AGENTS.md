# wx — Agent guide

Go CLI for US weather (NWS). Read CLAUDE.md for full architecture details.

## Commands

```bash
make build        # → build/wx
make test         # go test ./... -count=1
make vet          # go vet ./...
make build-all    # cross-compile darwin/linux/windows
make help         # list targets
```

**Always run `make test && make vet` after code changes.**

## Package layout (complete)

```
main.go              → cmd.NewApp().Run(os.Args)
cmd/
  app.go             urfave/cli v2 wiring; blank-import provider packages here
  config_cmd.go      wx config subcommand
  monitor_cmd.go     wx monitor TUI
  radar_cmd.go       wx radar subcommand
internal/
  cache/             TTL cache; NewNoOp() for tests
  config/            ~/.config/wx/config.json (atomic save via temp+rename)
  location/          Resolve(), ipgeo (ipinfo.io), geocode (Nominatim)
  models/            CurrentConditions (SI), Forecast, Alert
  monitor/           Monitor TUI (bubbletea)
  output/            Render() — TTY→pretty, pipe→JSON; lipgloss icons
  provider/          WeatherProvider interface + registry
  provider/nws/      NWS implementation; Register() in init()
  radar/             RadarProvider, interactive TUI, render (half-block/inline PNG)
```

## Provider architecture

New providers: implement `provider.WeatherProvider`, call `provider.Register()` from `init()`, blank-import in `cmd/app.go`. `Supports()` gates by country code (NWS checks `loc.CountryCode == "US"`).

## Model conventions

All measurements in SI: °C, km/h, hPa, meters. All value fields are `*float64` (`nil` = not reported). `ConditionCode` is a normalized string mapped in `output/icons.go`. Unit conversions happen only at render time.

## Output routing

`output.Render()` auto-detects TTY via `golang.org/x/term.IsTerminal`. TTY → pretty/lipgloss; pipe → JSON. `--json` forces JSON.

## Radar caching (two-tier)

`image.Image` can't be JSON-serialized, so `RadarProvider` uses:
- L1: in-process `sync.RWMutex` map (5 min TTL, single invocation)
- L2: `cache.Cache` disk as base64 PNG (5 min current / 24h historical)

## Config file

`~/.config/wx/config.json`. Precedence: CLI flag > config > built-in default.

## Testing

- Tests in `_test.go` alongside code
- Provider tests: `httptest.NewServer` with mock NWS mux
- Cache tests: use `cache.NewNoOp()` to stay stateless
- JSON tests: `captureStdout` pipe trick

## Go version

Requires Go 1.26.1 (see go.mod).
