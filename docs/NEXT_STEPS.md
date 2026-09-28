# wx — Next Slice Handoff & Implementation Candidates

Generated: 2026-09-28  
Status: All roadmap items 1 through 10 delivered and verified (`make test && make vet` passing across all packages, `make mac-dmg` verified).

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

### 6. Radar Export Tooling (PNG / Animated GIF) — Completed
* CLI:
  - `wx radar --save frame.png` saves current radar image with city labels directly to disk.
  - `wx radar --loop --save-gif radar.gif` (and `wx radar --save-gif radar.gif` or `--save animation.gif`) fetches chronological radar frames, overlays city labels, and compiles into an animated GIF with extended live scan dwell time.
* Mac App:
  - "EXPORT SCAN" menu in `RadarTransportBar` supporting 1-click "Copy Frame to Clipboard", "Save Frame (PNG)...", and "Export Loop (Animated GIF)...".
  - Native ImageIO GIF generation and NSSavePanel integration.
  - Ambient toast confirmation banner (`COPIED`, `SAVED`).

---

### 7. Quantitative Precip Nowcasting & Rain Timeline — Completed
* High-resolution 15-minute / hourly precipitation onset, intensity rate, accumulation, and phase discrimination (Rain, Snow, Freezing Rain, Ice Pellets).
* CLI: `wx nowcast [location]` (aliases: `precip`, `rain`, `qpf`) with:
  - Onset / cessation countdowns ("Rain starting in ~20m", "Stopping in ~15m", or "Dry next 6h").
  - 15-min / hourly precipitation probability, intensity rate (`in/h` or `mm/h`), and liquid/snow accumulation totals.
  - Unicode intensity sparkline bar chart (` ▂▃▄▅▆▇█`).
  - Full `--json` payload support.
* Unified across core models (`models.Nowcast` in `models.CurrentConditions`), terminal output, and JSON serialization.
* TUI: Precipitation sparkline and onset/cessation indicator integrated directly into `wx monitor` observation block.
* Mac App: `NowcastCardView` with dynamic timeline bars, status badge, accumulation summary, and interval inspection integrated into `SURFACE` and `TACTICAL` tabs, plus ambient `PopoverView` HUD.

---

### 8. NOAA Climate Normals & Departure Telemetry — Completed
* Official NOAA ACIS 30-year (1991–2020) Climate Normals, all-time daily records (sampled across up to 130+ years of station history with tie-year tracking), and real-time departure anomaly calculation.
* CLI: `wx climate [location]` (aliases: `normals`, `records`) with:
  - Daily 30-year normals: Normal High, Normal Low, Normal Mean, Normal Precipitation.
  - All-time daily records: Record High, Record Low, Record Precipitation, Coldest High, Warmest Low with tie years.
  - Departure anomaly telemetry: `+4.2°F Departure (Above Normal)` or `-12.3°F Departure (Significantly Below Normal)`.
  - Monthly normals baseline: Monthly Average High/Low and Total Normal Precipitation.
  - Automatic station discovery with active-station date range filtering and national grid fallback (`GridData`).
  - Full `--json` payload support.
* TUI: Real-time climate departure anomaly badge and normals integrated into `wx monitor` observation block.
* Mac App: Dedicated Climate Normals and All-Time Records telemetry card integrated into the `CLIMATE` tab, featuring departure anomaly badge, 30-year daily baseline, historical extremes, and monthly statistics alongside historical charts.

---

### 9. Atmospheric Sounding & Convective Instability (`wx sounding` / `wx cape`) — **COMPLETED**
* **CLI Subcommand**: `wx sounding [location]` (aliases: `cape`, `instability`, `skewt`)
  - Curated ~75 NOAA NWS Radiosonde (RAOB) launch sites across CONUS/AK/HI/PR with spatial Haversine distance lookup.
  - Multi-cycle SPC observation scraper (`YYMMDDHH_OBS`) parsing full NSHARP tabular soundings (`.txt`) and linking Skew-T thermodynamic diagrams (`.gif`).
  - High-resolution fallback provider via Open-Meteo model soundings for international coordinates or offline stations with dynamic Cartesian vector shear calculations (0-1km and 0-6km bulk shear).
  - Telemetry indices: SBCAPE, MLCAPE, MUCAPE, CIN, Lifted Index, Precipitable Water (PWAT), Freezing Level, DCAPE, 0-1km & 0-6km Bulk Shear, 0-1km & 0-3km Storm-Relative Helicity (SRH), Significant Tornado Parameter (STP), Supercell Composite Parameter (SCP), and 700-500hPa & 850-500hPa lapse rates.
  - Convective severity classification: `SEVERE`, `ELEVATED`, `MODERATE`, `MARGINAL`, `STABLE / WEAK`.
  - Mandatory pressure levels table (surface, 925, 850, 700, 500, 300, 250, 200 hPa) with relative humidity visual gradient bars and wind barbs.
  - Full `--json` payload support.
* **Storm Chase Integration**:
  - `wx chase` clusters now resolve and display the nearest upper-air sounding station (`SoundingStation`).
* **Mac App Integration**:
  - Upper-Air Sounding & Convective Instability HUD card integrated into the `STORM CHASE` console.
  - Displays station metadata, distance, convective risk badge, 8-KPI telemetry grid (SBCAPE, MLCAPE, CIN, LI, 0-6km Shear, 0-1km Shear, 0-1km SRH, PWAT), and direct action button to open NOAA SPC Skew-T diagram.
  - Direct "LOAD SOUNDING" action on every storm cluster card to immediately inspect upper-air thermodynamic profile for that cell cluster.

---

### 10. NOAA National Hurricane Center (NHC) Tropical Tracker (`wx tropics` / `wx nhc`) — **COMPLETED**
* **CLI Subcommand**: `wx tropics [location]` (aliases: `nhc`, `hurricane`, `cyclone`, `tropical`)
  - Direct ingestion of official NOAA NHC `CurrentStorms.json` feed covering Atlantic, Eastern Pacific, and Central Pacific basins.
  - Ingestion of Tropical Weather Outlook (TWO) bulletins for active invest disturbances (AL9x, EP9x, CP9x) with 48-hour and 7-day genesis probabilities (Low, Medium, High).
  - Saffir-Simpson Hurricane Wind Scale classification: Category 1 through Category 5 (Major Hurricane), Tropical Storm, Tropical Depression, Potential Tropical Cyclone.
  - Full telemetry: maximum sustained winds (mph, kt, km/h), minimum central pressure (mb, inHg), movement compass heading (degrees and cardinal) & speed, decimal coordinates, proximity text to landmarks/coastlines, and Haversine distance from user reference location.
  - Scrapes active public advisory bulletins for storm headlines, landfall warnings, and active coastal watches/warnings.
  - Direct links to NOAA NHC interactive track/cone of uncertainty graphics, forecast discussions, public advisories, and KMZ/GIS layers.
  - Filtering by specific storm name or ID via `--storm <name|id>` flag.
  - Full `--json` payload support.
* **Mac App Integration**:
  - Dedicated `TROPICS` console tab added to DeskWindowView mode selector.
  - Real-time active storm telemetry cards with Saffir-Simpson color coding (Cat 5 Magenta, Cat 4 Red, Cat 3 Amber, Cat 1-2 Gold, TS Cyan, TD Green).
  - Quick action buttons to open official NHC Track & Cone graphics in browser, or jump radar array directly to cyclone coordinates.
  - Invest disturbance cards with 48h and 7d probability pills.
  - Direct action links for Atlantic and Eastern Pacific 7-Day Graphical Tropical Weather Outlook maps.



