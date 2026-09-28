# wx — Next Slice Handoff & Implementation Candidates

Generated: 2026-09-27  
Status: Options 1–5 successfully delivered and verified (`make test && make vet` passing, `make mac-dmg` verified). Ready for fresh session.

---

## 1. Multi-Radar Composite Mosaic (Zoom-Out Coverage)
* **Goal**: When zoomed out beyond single-station range (>250–300 miles) or viewing regional/national scales, composite multiple adjacent NEXRAD stations or fetch NOAA MRMS CONUS seamless mosaic tiles.
* **Architecture**:
  - `internal/radar/`: Multi-station tile blending or MRMS national composite layer provider.
  - Mac App (`RadarMapView.swift`): Switch smoothly between local NEXRAD high-res single-site and CONUS mosaic overlay based on MapKit zoom region span (`region.span.latitudeDelta`).
  - Terminal (`internal/radar/render.go`): Multi-station sampling when bounding box spans multiple radars.

---

## 2. SPC (Storm Prediction Center) Convective Outlooks & Mesoscale Discussions
* **Goal**: Deepen the remote storm chasing and hazard awareness capabilities with official SPC severe weather outlooks.
* **Deliverables**:
  - Ingest SPC Day 1 / Day 2 / Day 3 categorical risk boundaries (General Thunder, Marginal, Slight, Enhanced, Moderate, High) and probabilistic layers (Tornado, Hail, Damaging Wind).
  - Surface active Mesoscale Discussions (MCD) and Tornado/Severe Thunderstorm Watches.
  - Wire into `wx chase` / `wx outlook --spc` and the Mac App `STORM CHASE` console.

---

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
