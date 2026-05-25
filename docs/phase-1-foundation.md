# Phase 1: Foundation — Tier 1 Implementation Plan

**Project:** wx (github.com/mwirges/wx)  
**Phase:** 1 of 3 (Tier 1 Features)  
**Status:** For review and feedback  
**Depends on:** Current main branch (clean, all tests passing)  
**Target Outcome:** A complete, data-rich core weather experience with excellent scripting ergonomics and the architectural patterns needed for Phases 2 and 3.

---

## 1. Overview & Objectives

### Primary Goals
- Deliver the highest-value missing features that make `wx` feel like a **complete** terminal weather tool.
- Establish durable patterns for:
  - Richer forecast data
  - Extensible output formatting (short + templated)
  - User preferences stored in `config.json` (favorites, recents)
  - Pure-computation astronomy (no new runtime HTTP dependencies)
- Preserve the project's core principles: NWS-first, zero-config for the end user, SI models, clean TTY/JSON routing, excellent caching.

### Non-Goals (for this phase)
- No new external data sources at runtime (sunrise/sunset will be calculated locally).
- No historical/climate data or second providers (those are Phase 3).
- No major TUI rewrites (monitor/radar polish is Phase 2).

### Success Statement
After Phase 1, a user running `wx --forecast --hourly --short` or using the new templating should get genuinely useful, script-friendly output, and the internal models + config will be ready for per-location customization and richer features in later phases.

---

## 2. Features In Scope (Prioritized)

| Priority | Feature | Why It Matters | Rough Complexity |
|----------|---------|----------------|--------------------|
| P1 | Hourly forecast (`--hourly` flag + `wx hourly` command) | The provider already supports it internally; the CLI never exposes it. | Low-Medium |
| P1 | Richer `Period` data (PoP, dewpoint, humidity, etc.) | Current forecasts are text-only. NWS hourly endpoint already returns quantitative fields. | Medium |
| P1 | One-line / statusline mode (`--short` + `--format template=...`) | Highest-ROI scripting feature for prompts, tmux, scripts, `watch`. | Medium |
| P2 | Sunrise, sunset, day length (astronomy) | Extremely common request. Achievable with pure Go (NOAA solar equations). | Medium |
| P2 | Severity-aware alerts (summary line + better pretty rendering) | Current alerts are a flat dump. | Low |
| P2 | Location favorites + recent locations (persisted in config.json) | Dramatically improves monitor, radar, and daily UX. Foundation for Phase 2 per-location config. | Medium |
| P3 | Minor hygiene | Fix dead `version` injection in Makefile + `cmd/app.go`. Add basic data age indicators where easy. | Low |

All features must respect:
- SI units in models.
- `*float64` for optional measurements.
- Atomic config writes.
- Existing cache TTL discipline.
- No breaking changes to the `WeatherProvider` interface.

---

## 3. Data Model Changes

### 3.1 `models.Period` (forecast.go)

**Current:**
```go
type Period struct {
    Name         string
    StartTime    time.Time
    EndTime      time.Time
    IsDaytime    bool
    TempC        float64
    WindKPH      float64
    WindDir      string
    ShortDesc    string
    DetailedDesc string
}
```

**Proposed (additive):**
```go
type Period struct {
    Name         string
    StartTime    time.Time
    EndTime      time.Time
    IsDaytime    bool
    TempC        float64
    WindKPH      float64
    WindDir      string
    ShortDesc    string
    DetailedDesc string

    // New in Phase 1
    ProbabilityOfPrecipitation *float64 // 0-100, nil if not reported (standardized to *float64 for model consistency)
    HumidityPct                *float64 // 0-100
    DewPointC                  *float64
    // WindGustKPH could be added later if the hourly endpoint provides it cleanly
}
```

**Notes:**
- Use pointer for PoP and humidity so "not reported" is distinguishable from 0.
- Keep `TempC`, `WindKPH` as value types for backward compatibility (they were always present).
- Update `jsonPeriod` in `output/json.go` to include the new fields (additive JSON is fine).

### 3.2 Astronomy Data

**Option A (preferred for simplicity):** New small struct, attached to `CurrentConditions` for the current day.

```go
// Astronomy contains calculated sun/moon data for the location's date.
type Astronomy struct {
    Sunrise   time.Time // local time
    Sunset    time.Time // local time
    DayLength time.Duration
    // MoonPhase string (optional stretch for Phase 1)
}
```

Attach as:
```go
type CurrentConditions struct {
    ...
    Astronomy *Astronomy // new, nil if calculation failed
}
```

**Option B:** Separate top-level in render data. Option A is simpler for "current conditions + sun times."

We will decide during implementation, but the plan recommends Option A with a pure function `CalculateAstronomy(lat, lon, date time.Time) (*Astronomy, error)`.

### 3.3 Alert Enhancements (minor)

The existing `Alert` struct already has `Severity` and `Urgency`. Phase 1 will focus on **rendering** improvements rather than model changes. A small `IsWarning() bool` helper can be added to `models/alert.go`.

### 3.4 JSON Output Impact

- All new fields are **additive** (`omitempty` where appropriate).
- Minor breaking change tolerance accepted per project guidance: we will bump the conceptual "data version" in docs if needed and note it in the README.

---

## 4. NWS Provider Changes

### 4.1 Forecast Parsing (`internal/provider/nws/forecast.go`)

- Extend the internal `forecastPeriod` struct to capture the full rich fields returned by `/gridpoints/.../forecast/hourly`:
  ```go
  type forecastPeriod struct {
      // existing fields...
      ProbabilityOfPrecipitation struct {
          Value *float64 `json:"value"` // NWS returns integer; we store as *float64 for model consistency
      } `json:"probabilityOfPrecipitation"`
      Dewpoint struct {
          Value *float64 `json:"value"`
          // unitCode
      } `json:"dewpoint"`
      RelativeHumidity struct {
          Value *float64 `json:"value"`
      } `json:"relativeHumidity"`
      // ... windGust if present, etc.
  }
  ```
- In the `Period` construction loop, populate the new pointer fields (convert units as needed — NWS already gives Celsius for dewpoint in the grid response).
- The 12-hour textual forecast (`/forecast`) will continue to return minimal data (many quantitative fields are null or absent there). This is acceptable; rich data primarily comes with `--hourly`.

### 4.2 Astronomy

- New package/file: `internal/astro/astro.go` (the authoritative home for the pure calculation logic and public API).
- Pure Go implementation of a simplified NOAA Solar Calculator (based on the well-known Meeus/Almanac equations or the public domain NOAA reference implementation).
- Must be deterministic, well-tested (edge cases: equator, high latitudes, DST transitions, polar day/night).
- No external HTTP at runtime. Optionally seed from a small embedded table for speed, but pure calculation is preferred.
- Cache results for the day (trivial in-memory or via the existing disk cache keyed by lat/lon/date).

### 4.3 Cache Keys

- New cache key patterns:
  - `nws:forecast-hourly:...` (already partially used)
  - `nws:astro:lat,lon:date` (daily TTL)

---

## 5. CLI & Command Surface

### 5.1 Main `wx` Action (cmd/app.go)

- New flag:
  ```go
  &cli.BoolFlag{
      Name:    "hourly",
      Aliases: []string{"H"},
      Usage:   "show hourly forecast (implies --forecast)",
  },
  ```
- Update the concurrent fetch logic:
  ```go
  showHourly := c.Bool("hourly")
  if showHourly {
      showForecast = true
  }
  ...
  fc, fcErr = prov.Forecast(ctx, loc, showHourly, ch)
  ```
- Pass `ShowHourly` through `RenderOptions`.

### 5.2 New or Enhanced Subcommands

- `wx hourly` (convenience wrapper for `wx --hourly`).
- `wx locations` (or extend `wx config`):
  - `wx locations list`
  - `wx locations add "Chicago, IL"`
  - `wx locations remove "Chicago, IL"`
  - `wx locations recent` (last 5–10 auto-tracked)
- Keep `wx config` for global settings; locations become a first-class section.

### 5.3 Output Flags (new in RenderOptions)

```go
type RenderOptions struct {
    ...
    Short          bool
    Template       string   // raw template string or path (prefixed with @)
    ShowHourly     bool
    ShowAstronomy  bool     // default true when available
}
```

---

## 6. Output System Design (Critical Foundation for All Phases)

### 6.1 Short / Compact Mode

- When `Short: true`:
  - Single line (or very few) optimized for prompts.
  - Example default: `KC 72°F (feels 68) Clear · 10mph NW · 42%  ↑6:42 ↓7:58`
  - Respects units.
  - Includes astronomy when available.
- Controlled by `--short` / `-s`.

### 6.2 Templating

**Recommended UX (resolved after review feedback):**

We will use two distinct, conventional flags rather than overloading `--format`:

- `--short` / `-s` — Built-in compact one-line mode (the 80% case for prompts/scripts).
- `--template '...' ` (or `-T`) — Literal Go `text/template` string or `@/path/to/file`.
  - Example: `--template '{{.Conditions.Location}}: {{.Conditions.TempF}}°F {{.Conditions.Description}}'`
  - File form: `--template @/etc/wx/prompt.tmpl`
- `--format name` — For named built-in formats (e.g. `--format prompt`, `--format jsonl`, future ones).

This avoids magic prefix parsing inside a single flag value and follows patterns users expect from other CLIs (`--template` is common in tools like `git`, `kubectl`, etc.).

Predefined variables in templates: `.Conditions`, `.Forecast`, `.Alerts`, `.Astronomy`, `.Now`, plus helpers (`formatTemp`, `compass`, `time`, `age`).

Security: templates execute locally only.

### 6.3 Implementation Location

- New file: `internal/output/short.go`
- New file or section: `internal/output/template.go`
- Extend `renderPretty` and `renderJSON` paths (JSON output should also respect some template concepts or stay rich).
- Pretty short mode still uses lipgloss for color when the terminal supports it (or plain text).

### 6.4 Data Age / Freshness (light)

- Add `LastUpdated time.Time` or age helpers on render data.
- In short mode and monitor, surface "· 4m ago" when data is from cache.

---

## 7. Config Schema Evolution

**Current example:**
```json
{
  "default_location": "Kansas City, MO",
  "units": "imperial",
  "notifications": true
}
```

**Phase 1 target:**
```json
{
  "default_location": "Kansas City, MO",
  "units": "imperial",
  "notifications": true,
  "favorites": [
    {"name": "Home", "value": "Kansas City, MO"},
    {"name": "Work", "value": "64101"}
  ],
  "recent_locations": ["Chicago, IL", "Denver, CO"]
}
```

- `recent_locations` is auto-managed (append on successful resolve, deduped, capped at 8–10).
- `favorites` are user-managed via `wx locations`.
- Add the following types in `internal/config/config.go`:

```go
type LocationEntry struct {
    Name  string `json:"name"`
    Value string `json:"value"` // same formats as --location (zip, "City, ST", etc.)
}

type Config struct {
    // ... existing fields ...
    Favorites       []LocationEntry `json:"favorites,omitempty"`
    RecentLocations []string        `json:"recent_locations,omitempty"`
}
```

- Update `Save` / `Load` (additive — old files continue to work).
- Provide `config.MigrateIfNeeded` or just handle missing fields gracefully.

---

## 8. Astronomy Implementation Strategy

- Create `internal/astro/astro.go`.
- Public API:
  ```go
  func SunriseSunset(lat, lon float64, date time.Time) (sunrise, sunset time.Time, err error)
  func DayLength(sunrise, sunset time.Time) time.Duration
  ```
- Reference implementation: Port/adapt the well-known NOAA/GSFC pure algorithm (public domain equations). Several clean Go ports exist; we will pick or write one with excellent tests.
- Handle:
  - Location in any timezone (return times in location's local time or UTC + offset note).
  - Polar day / night (sunrise/sunset can be nil or special values).
- Tests: Table-driven with known good values for major US cities + edge cases (Fairbanks, Quito, etc.).
- No new Go modules.

---

## 9. Detailed File-by-File Change Inventory

| File | Type of Change | Notes |
|------|----------------|-------|
| `internal/models/conditions.go` | Add `Astronomy *Astronomy` | Small |
| `internal/models/forecast.go` | Extend `Period` | Core data change |
| `internal/models/astro.go` (new) | New astronomy types | |
| `internal/config/config.go` | Add Favorites + RecentLocations fields + helpers | |
| `internal/provider/nws/forecast.go` | Richer response struct + population logic | |
| `internal/astro/astro.go` (new) | Pure astronomy calculator + public API (see Section 8) | |
| `internal/output/output.go` | Extend RenderOptions | |
| `internal/output/short.go` (new) | Short renderer | |
| `internal/output/template.go` (new) | Template engine + helpers | |
| `internal/output/json.go` | Extend json* structs (additive) | |
| `internal/output/pretty.go` | Update for hourly sections + astro + alert summary | |
| `cmd/app.go` | New --hourly flag, pass-through, version fix | |
| `cmd/config_cmd.go` or new `cmd/locations_cmd.go` | Location management commands | |
| `cmd/radar_cmd.go`, `cmd/monitor_cmd.go` | Minor integration (use favorites) | |
| `README.md`, `internal/output/help text` | Updates | |
| `*_test.go` (many) | New tests for all new paths | |

---

## 10. Recommended Implementation Order (Milestones)

1. **M1: Data Model & Provider** — Extend `Period`, update NWS forecast parser, add tests. Verify richer hourly JSON works.
2. **M2: Astronomy Core** — Land the pure calculator + tests. Wire into `CurrentConditions`.
3. **M3: Config Evolution** — Add favorites/recent fields + migration behavior + basic `wx locations` commands.
4. **M4: Output — Short Mode** — Implement `--short` default format. Works for current conditions + limited forecast.
5. **M5: Output — Templating** — Full template support + helpers. `--template '...' ` flag (plus `--format name` for named formats).
6. **M6: CLI Integration & Hourly** — Wire `--hourly`, `wx hourly`, update main action and render paths.
7. **M7: Alert Polish + Freshness** — Severity summary, age indicators in short/pretty.
8. **M8: TUI Integration & Polish** — Light use of favorites in monitor/radar; docs + README.
9. **M9: Tests, Vet, Cross-check** — Full coverage on new code, golden JSON tests, `make test && make vet`.
10. **M10: Documentation & Release Prep** — Update all help text, examples, changelog notes.

---

## 11. Testing Strategy

- **Unit tests** for astronomy (many edge cases, including polar).
- **Provider tests** — extend existing `httptest` mux to return rich hourly responses with PoP/dewpoint.
- **Output tests** — new tests in `json_test.go` and `pretty_test.go` for short mode and templates. Use golden files or capture for templates.
- **Config tests** — roundtrip favorites + recents, old config compatibility.
- **Integration** — `captureStdout` style tests for `wx --short`, `wx --hourly --json`.
- **Property / table-driven** for format helpers.

---

## 12. Risks, Open Questions & Compatibility

**Risks**
- Astronomy accuracy at extreme latitudes.
- Template syntax discoverability (mitigate with good examples and `--help`).
- JSON consumers surprised by new fields (document clearly).

**Open Questions for Reviewer (updated after initial review)**
- Exact default short format string (we'll propose 2–3 options).
- Should `--hourly` imply `--forecast` or require both?
- Do we want a small number of named template shortcuts (`--format prompt`) in Phase 1?
- Naming of the locations subcommand (`wx locations` vs `wx config locations`)?

**Resolved in this revision:**
- Astronomy package location standardized to `internal/astro/astro.go` (pure calculation, not provider-specific). All references updated.

**Compatibility**
- Old config files load fine.
- Old JSON consumers continue to work (new fields are additive or optional).
- Existing CLI flags and behavior unchanged unless the new flag is used.

---

## 13. Exit Criteria (Verifiable Checklist)

- [ ] `make test && make vet` passes cleanly.
- [ ] `wx --hourly` and `wx hourly` produce hourly periods (pretty + JSON).
- [ ] Rich fields (PoP, dewpoint, humidity) appear in hourly JSON and pretty output.
- [ ] `wx --short` produces a single useful line.
- [ ] `wx --template '...'` (or `@file`) works with a documented set of fields and helpers.
- [ ] `wx locations add/list/remove` works and persists to `~/.config/wx/config.json`.
- [ ] Sunrise/sunset appears in current conditions (pretty + JSON) for US locations.
- [ ] Alerts render with a severity summary line.
- [ ] Favorites and recents influence location resolution and are visible in `wx config show`.
- [ ] README and `--help` output are updated with new capabilities.
- [ ] All new code has tests; no dead code from the version fix.
- [ ] No new runtime dependencies.

---

**End of Phase 1 Plan**

This document is intentionally detailed so you can review the architecture, data shapes, and sequencing before any code is written.

Next documents (Phase 2 and Phase 3) will heavily reference this one and only describe the *incremental* changes.

Please provide feedback, preferred options, or scope adjustments. Once approved, I can generate the matching detailed task lists or begin implementation spikes.