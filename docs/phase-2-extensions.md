# Phase 2: Extensions — Tier 2 Implementation Plan

**Project:** wx  
**Phase:** 2 of 3 (Tier 2 Features)  
**Status:** For review  
**Prerequisite:** Phase 1 Foundation complete and merged (richer `Period`, short + templating output, favorites in config.json, astronomy, hourly support).

---

## 1. Overview

Phase 2 turns the solid data and output foundations from Phase 1 into a **power-user, personalized, and polished** experience.

The focus is on:
- Intelligence (per-location memory)
- Freshness & trust signals
- Ergonomics for both interactive TUI users and automation users
- Incremental radar and monitor improvements that feel like natural extensions

All changes remain strictly NWS-first / zero-config for the end user.

---

## 2. Features In Scope

| Priority | Feature | Builds On (Phase 1) | Complexity |
|----------|---------|---------------------|------------|
| P1 | Per-location configuration (units, default radar product/radius, preferred center) | Config favorites + recent locations | Medium |
| P1 | First-class `FeelsLike` in models + everywhere | Existing `FeelsLikeTemp` helper + JSON anticipation | Low |
| P1 | Excellent data freshness / age UX (short mode, pretty, monitor) | Short mode + freshness helpers started in P1 | Medium |
| P1 | Exit codes for automation (`--exit-code-if-warning`, sensible defaults) | CLI action + RenderOptions | Low |
| P2 | Monitor TUI improvements (location picker/search using favorites/recents, better status) | Favorites in config + freshness | Medium |
| P2 | Radar enhancements (`--save`, additional NWS products if clean, last-used product per location) | Per-location config + Phase 1 output patterns | Medium |
| P2 | Richer short / template helpers (age, feels-like, precip summary) | Templating engine from Phase 1 | Low |
| P3 | Minor radar/monitor polish (keyboard hints, better splitting) | Existing bubbletea code | Low |

---

## 3. Key Architectural Increments

### 3.1 Per-Location Configuration

Extend the config structures created in Phase 1.

**Proposed additions to `internal/config/config.go` (extending Phase 1):**

```go
// LocationEntry was introduced in Phase 1 for favorites.
type PerLocationSettings struct {
    Units               string  `json:"units,omitempty"`            // "imperial" | "metric"
    DefaultRadarProduct string  `json:"default_radar_product,omitempty"`
    DefaultRadarRadius  float64 `json:"default_radar_radius,omitempty"`
    RadarStation        string  `json:"radar_station,omitempty"` // e.g. "KIWX"
}

type Config struct {
    // ... Phase 1 fields (including Favorites []LocationEntry) ...
    RecentLocations []string                       `json:"recent_locations,omitempty"`
    PerLocation     map[string]PerLocationSettings `json:"per_location,omitempty"` // keyed by normalized location key or name
}
```

- Resolution order (highest wins):
  1. Explicit CLI flag
  2. Per-location setting for the resolved location
  3. Global config
  4. Built-in default

Helper methods (new in Phase 2):
- `cfg.GetEffectiveUnits(loc string) string`
- `cfg.GetRadarOptions(loc string) radar.Options`

### 3.2 First-Class Feels Like (Accurate Scope + Locked Design)

**Current state (pre-Phase 2):**
- `output.FeelsLikeTemp(windChill, heatIndex *float64) *float64` already exists in `pretty.go:154` (and is tested).
- The JSON renderer in `json.go:100` already calls this helper and emits `feels_like_c` / `feels_like_f`.
- The "Feels like" line in pretty output already uses the helper.

**What Phase 2 actually delivers:**
Promoting the *computed* apparent temperature to a first-class field on the `CurrentConditions` model itself (`FeelsLikeC *float64`). This makes the value available to all consumers (not just the render layer), improves consistency, and allows providers to pre-compute or override it if desired.

**Locked design decision after review feedback:**

We keep three distinct fields:

- `WindChillC *float64` — Raw NWS-reported wind chill (only when NWS criteria are met).
- `HeatIndexC *float64` — Raw NWS-reported heat index (only when NWS criteria are met).
- `FeelsLikeC *float64` (new on model) — The effective apparent temperature **when a modifier applies**. Computed as:
  1. `WindChillC` if set
  2. `HeatIndexC` if set
  3. `nil` otherwise (no apparent-temperature modifier applies)

**Rationale for `nil` fallback (not `TempC`):**
- Matches the existing `FeelsLikeTemp` helper behavior today.
- `nil` means "no special feels-like value was computed by NWS or our logic" — this is semantically different from "the temperature is X".
- Prevents `FeelsLikeC` from being a redundant copy of `TempC` in the common case.

**JSON output impact:**
All three raw + computed fields will be exposed (additive). The existing `feels_like_*` fields in JSON will now be sourced from the model field instead of (or in addition to) the helper.

A small pure helper `ComputeFeelsLike(...) *float64` will still exist (moved or kept in `internal/models` or output) for reuse during population.

Provider (NWS) may pre-populate `FeelsLikeC`; if absent, the helper can fill it from the raw windchill/heatindex fields.

### 3.3 Freshness & Age System

Introduce a small, reusable concept:

```go
// internal/output/freshness.go (or in render data)
type Freshness struct {
    FetchedAt time.Time
    Source    string // "live" | "cache"
}

func (f Freshness) Age() time.Duration { ... }
func (f Freshness) AgeString() string   { ... } // "just now", "4m ago", "2h ago"
```

- Attach to `RenderData` and to the monitor model.
- Short mode and monitor get prominent age display (color by staleness).
- Radar frames already carry `ValidTime` — unify presentation.

### 3.4 Exit Codes

**Important (resolved after review):** Automatic non-zero exit codes must be opt-in only. Unconditionally exiting with code 1 on any alert would silently break existing scripts and pipelines that do not expect this behavior.

**Recommended design:**

- `--exit-code-on-alerts` (or more granular `--exit-code-on-warnings` / `--exit-code-on-watches`)
- When the flag is present:
  - Warnings → exit 2
  - Watches / Advisories → exit 1 (or a single code; we can decide)
- Default behavior (no flag): always exit 0 on success, just like today. Warnings/alerts are only informational unless the user explicitly opts into stricter automation behavior.

This keeps backward compatibility for all current users while giving power users the automation hook they want. The plan will clearly document the risk of surprising script breakage if we ever considered making it a default.

---

## 4. Monitor TUI Improvements (Phase 2)

- Location input now supports tab-completion or fuzzy search against favorites + recents (using the Phase 1 data).
- Status line (already partially present) now shows strong freshness + last location switch hint.
- "l" key opens an improved location switcher that lists favorites first.
- Optional: persist last radar panel state per location (small addition to `PerLocationSettings`).

No full rewrite — incremental enhancements to `internal/monitor/`.

---

## 5. Radar Enhancements

- `--save /path/to/radar.png` (or `--save` with auto-naming) in `cmd/radar_cmd.go`.
- Persist last-used product + radius in the per-location config (read/write via the helpers from 3.1).
- If NWS/IEM expose additional clean products (e.g., storm total precipitation) that fit the existing `Product` + `IsStationProduct` logic, add them with minimal surface area.
- Export uses the already-rendered (or source) image; keeps the two-tier cache discipline.

---

## 6. File Change Highlights (Incremental from Phase 1)

| File | Changes |
|------|---------|
| `internal/config/config.go` | `PerLocation` map + getters + migration for old favorites |
| `internal/models/conditions.go` | `FeelsLikeC` field (if not done in P1) |
| `internal/output/freshness.go` (new) | Age types + helpers |
| `cmd/app.go` | Exit code logic + per-location config application |
| `internal/monitor/*` | Location picker + freshness display |
| `internal/radar/*` + `cmd/radar_cmd.go` | Save support + last-used prefs |
| `internal/output/short.go` + template helpers | Leverage new FeelsLike + Freshness |

---

## 7. Implementation Order

1. Per-location config storage + getters (foundational for everything else in P2).
2. First-class FeelsLike + freshness system.
3. Exit code support.
4. Radar save + per-location radar prefs.
5. Monitor TUI picker + status improvements.
6. Template/short helper expansions.
7. Tests + docs.

---

## 8. Risks & Mitigations

- Config.json grows in complexity → Keep `wx config show` excellent and add good comments in the saved file.
- Monitor TUI changes can introduce layout bugs → Heavy use of existing render tests + manual verification checklist.
- "Last used radar" can surprise users → Make it opt-in or clearly documented; always overridable on the command line.

---

## 9. Exit Criteria

- [ ] Per-location settings are read and honored for units and radar.
- [ ] `wx --short` shows attractive freshness and feels-like data.
- [ ] Monitor has a pleasant location switcher using favorites/recents.
- [ ] `wx radar --save` works and respects per-location defaults.
- [ ] Exit codes behave as documented.
- [ ] `make test && make vet` clean.
- [ ] README has power-user examples (prompt integration, exit codes, per-city radar).

---

**Phase 2 builds cleanly on the data model, config, and output architecture delivered in Phase 1.**

Please review for scope, sequencing, and any items you want moved earlier or deferred.