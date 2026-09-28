# wx — Next Slice Handoff & Implementation Candidates

Generated: 2026-09-27  
Status: Items 1 & 2 delivered and verified (`make test && make vet` passing across all 18 packages, `make mac-dmg` verified). Ready for next slice.

---

## Completed Slices

### 1. Multi-Radar Composite Mosaic (Zoom-Out Coverage) — Completed (babf266)
* Multi-station tile blending with NOAA MRMS national composite fallback.
* Terminal radar header displaying contributing station count.
* MapKit dynamic scale switching and radar radius expansion chips.

### 2. SPC Convective Outlooks & Mesoscale Discussions — Completed
* Ingests official NOAA SPC Day 1 / Day 2 / Day 3 categorical risk boundaries (TSTM, MRGL, SLGT, ENH, MDT, HIGH) and probabilistic layers (Tornado, Hail, Damaging Wind).
* Surfaces active Mesoscale Discussions (MCD) with watch probability, affected areas, and summaries.
* Monitors active Tornado and Severe Thunderstorm Watches nationwide.
* Exposes `wx spc [location]`, `wx outlook --spc`, `wx chase --spc`, and interactive `s` key in `wx chase`.
* Auto-correlates remote storm chase clusters with active MCDs and convective risk ceilings.
* High-visibility SPC Convective Intelligence & MCD Console in the Mac App `STORM CHASE` tab.

---

## Upcoming Candidates (In Priority Order)

## 3. Air Quality Index (AQI) & Smoke Plume Console
* **Goal**: Surface real-time air quality metrics alongside atmospheric telemetry.
* **Deliverables**:
  - Expose `wx aqi [location]` in the CLI using the pre-existing `internal/airquality` engine (EPA AirNow & Open-Meteo fallback).
  - Add an AQI widget to the Mac App `SURFACE` / `TACTICAL` tabs (US AQI rating, PM2.5, PM10, Ozone, health advisories).
  - Add AQI bar indicator to `wx monitor` TUI.

---

## 4. Astro & Ephemeris HUD (Solar Arc & Lunar Cycle)
* **Goal**: Offline, pure-calculation astronomy metrics (zero API calls needed).
* **Deliverables**:
  - Expose `wx astro [location]` in CLI via `internal/astro`.
  - Display solar elevation arc, sunrise/sunset, civil/nautical/astronomical twilight, golden hour, and moon phase illumination (with Unicode glyphs).
  - Add an Ephemeris card to the Mac App desk window and `wx monitor`.

---

## 5. Menu Bar Customization & Ambient HUD
* **Goal**: Fine-grained user control over the macOS menu bar footprint.
* **Deliverables**:
  - Configurable status item formats:
    - Compact: `72°`
    - Standard: `☀️ 72°`
    - Tactical: `[FWA] 72° ↘12mph`
    - Hazard alert indicator: pulsing amber/red pip when severe warnings are active.
  - Multi-location dropdown menu: quick preview cards for all pinned Command Grid locations directly from the menu bar item.

---

## 6. Radar Export Tooling (PNG / Animated GIF)
* **Goal**: Allow saving and sharing radar scans directly from CLI and Mac App.
* **Deliverables**:
  - CLI: `wx radar --save frame.png` and `wx radar --loop --save-gif radar.gif`.
  - Mac App: "Export Scan" button in `RadarTransportBar` to copy current radar frame or export 8-frame loop GIF.
