# wx — Next Slice Handoff & Implementation Candidates

Generated: 2026-09-27  
Status: Items 1, 2, 3, 4 & 5 delivered and verified (`make test && make vet` passing across all 18 packages, `make mac-dmg` verified). Ready for Item 6.

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

### 3. Air Quality Index (AQI) & Smoke Plume Console — Completed
* Exposes `wx aqi [location]` in CLI (aliases: `air`, `airquality`, `smoke`) with EPA AirNow category ratings, visual spectrum gauge, EPA health advisories, smoke plume detection, and full atmospheric chemistry (PM2.5, PM10, O3, NO2, CO, SO2, UV).
* High-visibility Air Quality & Smoke Plume Console card added to macOS App (`SURFACE` and `TACTICAL` tabs) with interactive AQI meter, category badge, EPA health guidance, and particulate breakdown.
* AQI bar indicator and real-time particulate telemetry integrated directly into `wx monitor` TUI.

---

### 4. Astro & Ephemeris HUD (Solar Arc & Lunar Cycle) — Completed
* Pure offline astronomical ephemeris calculation with zero external APIs using NOAA solar position & Meeus algorithms.
* Exposes `wx astro [location]` in CLI (aliases: `ephemeris`, `sun`, `moon`) with `--date` support.
* Displays solar elevation arc, twilight horizons (civil, nautical, astronomical dawn/dusk), morning/evening golden hour, daylight duration, and lunar illumination/cycle progress meter.
* High-visibility Ephemeris HUD card integrated into macOS App (`SURFACE` & `TACTICAL` tabs) and `wx monitor` TUI with real-time solar elevation and twilight phase badge.

---

### 5. Menu Bar Customization & Ambient HUD — Completed
* Configurable macOS menu bar format settings (`compact`, `standard`, `tactical`):
  - Compact: `72°`
  - Standard: `☀️ 72°`
  - Tactical: `[FWA] 72° ↘12mph` (station abbreviation, temperature, cardinal wind direction arrow, and wind speed)
* Real-time pulsing amber/red hazard pip alert indicator when active severe warnings or watches/advisories are in effect.
* Multi-location dropdown menu with quick preview cards for all pinned Command Grid locations directly from the menu bar item (with condition symbol, temperature, wind, and warning badges).
* Pinned Command Grid stations preview bar integrated directly into ambient `PopoverView` HUD with 1-click station switching.
* CLI `--menu-bar-format` configuration support via `wx config set --menu-bar-format <compact|standard|tactical>`.

---

## Upcoming Candidates (In Priority Order)

---

## 6. Radar Export Tooling (PNG / Animated GIF)
* **Goal**: Allow saving and sharing radar scans directly from CLI and Mac App.
* **Deliverables**:
  - CLI: `wx radar --save frame.png` and `wx radar --loop --save-gif radar.gif`.
  - Mac App: "Export Scan" button in `RadarTransportBar` to copy current radar frame or export 8-frame loop GIF.
