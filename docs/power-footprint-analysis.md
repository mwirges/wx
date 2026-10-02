# wx macOS App: Power & Energy Footprint Analysis

## 1. Executive Summary

This document analyzes the power and energy consumption profile of `wx.app` on macOS. In standard operation, `wx` has been reported by macOS Activity Monitor, `powerd`, and the system battery status menu as **"Using Significant Energy"**.

This analysis identifies the primary architectural drivers of energy consumption across the macOS AppKit/SwiftUI shell, background Concurrency tasks, MapKit vector view lifecycle, and Go CLI subprocess execution. It provides concrete measurements and actionable mitigation strategies to reduce the application's energy footprint to near-zero when idling in the background.

---

## 2. Diagnostic Methodology & Live Profiling

### 2.1 macOS Energy Impact Metrics
macOS flags applications as "Using Significant Energy" based on a combination of:
1. **CPU C-State Invalidation**: Frequent timer wakeups preventing CPU cores and package power from entering deep low-power sleep states (deep C-states).
2. **Periodic Heavy Core Bursts**: Sustained or high-frequency multi-threaded CPU bursts on Performance cores (P-cores).
3. **WindowServer Compositing Frequency**: Frequent mutation of UI elements (such as `NSStatusItem` in the system menu bar), which forces macOS `WindowServer` to wake up the GPU, re-composite display surfaces, and drive display refresh cycles.
4. **App Nap Prevention**: Zero-tolerance timers, active `CVDisplayLink` threads, or high-priority Quality-of-Service (`.userInitiated`) tasks that inhibit macOS App Nap.

### 2.2 Live Process Profiling Data
Sampling and tracing active execution on macOS (Apple Silicon arm64) revealed:
* **Nominal Idle Instruction Rate**: ~1.5 to 3.5 million CPU instructions and ~20–40 context switches per second while idling in the background.
* **Single Background Radar Cycle (Measured via `/usr/bin/time -l`)**:
  - **Go Process (`wx-cli radar`)**:
    - **11.95 billion instructions retired** per invocation.
    - **2.41 billion CPU cycles elapsed** per invocation.
    - Peak RSS: **177.6 MB**.
  - **Swift Process (`wx`)**:
    - Reading 5–15 MB Base64 JSON across stdout `Pipe`.
    - `JSONDecoder` parsing multi-megabyte payloads.
    - Decoding 8 Base64 strings into uncompressed `NSImage` bitmaps.
    - Invalidation and redraw of MapKit overlays.
* **Process Memory Footprint**:
  - **Resident Footprint**: ~181.5 MB.
  - **Peak Memory Footprint**: ~348.1 MB (driven by MapKit vector tile caching and decoded radar bitmap frames).

---

## 3. Root Cause Analysis: The 6 Energy Drivers

### Culprit 1: Unconditional 2-Minute Background Radar Loop (Primary Offender)
* **Code Reference**: [`macos/WxMac/Services/WeatherStore.swift`](file:///Users/mwirges/code/wx/macos/WxMac/Services/WeatherStore.swift#L231-L247)
* **Mechanism**:
  In `WeatherStore.start()`, `radarRefreshTask` schedules an independent loop every 2 minutes:
  ```swift
  radarRefreshTask = Task { [weak self] in
      while !Task.isCancelled {
          try await Task.sleep(nanoseconds: UInt64(radarInterval * 1_000_000_000)) // 120s
          guard !Task.isCancelled, let self = self else { return }
          if self.deskWindowOpen || self.selectedDeskTab == .dual || self.selectedDeskTab == .radar || self.radarPayload != nil {
              await self.refreshRadar()
          }
      }
  }
  ```
* **Why it fails**:
  - `selectedDeskTab` is initialized to `.dual` by default.
  - As soon as radar is fetched once, `radarPayload != nil` remains true indefinitely.
  - `deskWindowOpen` is **never updated** when the user closes the window because `deskWindow` has no `NSWindowDelegate`.
  - **Impact**: The app executes the 8-frame MRMS radar pipeline in the background **every 120 seconds, 24/7**, even when the window is closed and the display is asleep.
  - **Cumulative Energy Waste**: 30 times an hour, 720 times a day = **~8.6 trillion CPU instructions per day** burned generating radar composites that no user is viewing.

### Culprit 2: Zero-Tolerance 800ms Status Item Alert Pip Timer
* **Code Reference**: [`macos/WxMac/StatusItemController.swift`](file:///Users/mwirges/code/wx/macos/WxMac/StatusItemController.swift#L70-L87)
* **Mechanism**:
  ```swift
  private func refreshPipState() {
      let hasHazard = store.hasActiveWarning || store.hasActiveWatchOrAdvisory
      if hasHazard {
          if pipTimer == nil {
              pipPulseOn = true
              pipTimer = Timer.scheduledTimer(withTimeInterval: 0.8, repeats: true) { [weak self] _ in
                  Task { @MainActor in
                      self?.pipPulseOn.toggle()
                      self?.refreshButton()
                  }
              }
          }
      } else {
          pipTimer?.invalidate()
          pipTimer = nil
      }
  }
  ```
* **Why it fails**:
  - NWS weather advisories (e.g., Frost Advisory, Wind Advisory, Small Craft Advisory, Winter Weather Advisory) routinely remain in effect for 12 to 48 consecutive hours.
  - `Timer.scheduledTimer` without explicit tolerance sets `tolerance = 0`, disabling macOS timer coalescing.
  - Every 800 ms (1.25 Hz, 4,500 times per hour), the main thread wakes up, toggles `pipPulseOn`, and mutates the `NSStatusItem` button title or image.
  - Mutating an `NSStatusItem` button forces macOS `WindowServer` to wake up, re-render, and composite the system menu bar at 1.25 Hz. This constant wakeup cadence prevents CPU cores from remaining in low-power sleep states.

### Culprit 3: Redundant 5-Minute Background Subprocess Bursts
* **Code Reference**: [`macos/WxMac/Services/WeatherStore.swift`](file:///Users/mwirges/code/wx/macos/WxMac/Services/WeatherStore.swift#L214-L229)
* **Mechanism**:
  Every 5 minutes (300 seconds), `refreshTask` spawns multiple CLI processes in rapid succession:
  1. `wx --json` (Current conditions + hourly + synoptic forecast)
  2. `wx outlook --json` (CPC 6-10 day, 8-14 day, 3-4 week outlooks)
  3. `wx spc --json` (SPC convective outlooks & storm chase data)
  4. `wx <loc> --json` for every saved favorite location in the Grid matrix.
* **Why it fails**:
  - NOAA CPC outlooks update **once per day** (at approximately 20:00 UTC / 3:00 PM EST). Polling them every 5 minutes in the background produces no new data.
  - SPC convective outlooks update 4–5 times per day.
  - Grid favorites are only visible when the user selects the "Grid" tab.
  - When the desk window is closed, the status item only requires current temperature, condition icon, and hazard state. Spawning 4 to 10 Go runtimes every 5 minutes triggers repetitive CPU and I/O spikes.

### Culprit 4: Zombie MapKit / Desk Window Retention & Periodic Timelines
* **Code Reference**: [`macos/WxMac/AppDelegate.swift`](file:///Users/mwirges/code/wx/macos/WxMac/AppDelegate.swift#L406-L424) & [`macos/WxMac/Views/RadarPanelView.swift`](file:///Users/mwirges/code/wx/macos/WxMac/Views/RadarPanelView.swift#L288-L306)
* **Mechanism**:
  - `AppDelegate.deskWindow` is created with `isReleasedWhenClosed = false`.
  - When the user clicks the red (X) button, the window is ordered out (`orderOut`), but `AppDelegate.deskWindow` retains the window, its `NSHostingController`, `DeskWindowView`, `RadarPanelView`, and `MKMapView`.
  - `MKMapView` retains its Metal rendering context and `CVDisplayLink` thread (`CVDisplayLink::runIOThread()`).
  - In `RadarPanelView.swift`:
    ```swift
    TimelineView(.periodic(from: .now, by: 5.0)) { timeline in
        ...
    }
    ```
    This `TimelineView` evaluates every 5 seconds to compute the relative timestamp ("15s AGO"), driving SwiftUI view evaluation even when the window is hidden.
  - When background radar refreshes complete every 2 minutes, SwiftUI calls `RadarMapView.updateNSView`, forcing MapKit overlay updates and raster invalidations in an offscreen window.

### Culprit 5: Unbounded Radar Loop Player
* **Code Reference**: [`macos/WxMac/Services/WeatherStore.swift`](file:///Users/mwirges/code/wx/macos/WxMac/Services/WeatherStore.swift#L291-L305)
* **Mechanism**:
  - If the user clicks "LOOP" on the radar transport bar, `loopTask` cycles through frames every **380 ms (~2.63 Hz)**.
  - There is no check for window visibility, window miniaturization, or tab changes.
  - If the window is closed or another tab is selected while the loop is running, `loopTask` continues cycling frames indefinitely, modifying published properties and triggering continuous SwiftUI view invalidations.

### Culprit 6: Subprocess Quality-of-Service (QoS) Misconfiguration
* **Code Reference**: [`macos/WxMac/Services/WeatherBackend.swift`](file:///Users/mwirges/code/wx/macos/WxMac/Services/WeatherBackend.swift#L44-L100)
* **Mechanism**:
  - All background fetches use `Task.detached(priority: .userInitiated)`.
  - Apple Silicon's thread scheduler places `.userInitiated` work on high-power Performance cores (P-cores) at maximum clock speeds and voltages.
  - Background periodic polling should run with `.utility` or `.background` QoS, allowing macOS to route execution to energy-efficient (E-cores) without triggering battery penalties.

---

## 4. Architectural Mitigation Strategies

The following strategies provide a roadmap to eliminate wasteful energy consumption while preserving instant responsiveness when the user interacts with the app.

### Strategy A: Strict Window Visibility Lifecycle Gating (High ROI)
1. Implement `NSWindowDelegate` on `deskWindow` (`windowDidBecomeKey`, `windowDidResignKey`, `windowWillClose`, `windowDidMiniaturize`, `windowDidDeminiaturize`).
2. Maintain an accurate `store.deskWindowOpen` / `store.isDeskWindowVisible` state.
3. When the window is closed, miniaturized, or occluded:
   - **Immediately suspend `radarRefreshTask`**.
   - **Immediately pause `loopTask`** if currently running.
   - **Suspend non-essential background refreshes** (CPC, SPC, Grid favorites).
4. When the window re-opens:
   - Refresh radar if cached data is older than 5 minutes.
   - Resume live polling.

### Strategy B: Lightweight Menu-Bar-Only Background Mode
When the desk window is closed:
1. The app enters an ultra-low-power background mode.
2. Only `wx --json` (current conditions & alerts) runs on an extended interval (e.g. 15–20 minutes) with a 2-minute timer tolerance.
3. This allows macOS App Nap to engage fully, letting the CPU remain in deep sleep states.

### Strategy C: Redesign Hazard Pip Animation
Replace the sub-second 0.8s timer with an energy-efficient alternative:
1. **Static Visual Distinction**: Use a solid indicator (🔴 for Warnings, 🟠 for Watches/Advisories) rather than an oscillating timer.
2. **Hardware-Composited CALayer / CoreAnimation**: If pulsing is desired, use an `NSSymbolPulseEffect` or an AppKit layer animation rather than mutating `NSStatusItem.button.title` every 800ms.
3. **Slow Coalesced Pulse**: Alternatively, pulse every 10–15 seconds with a 2–3 second timer tolerance.

### Strategy D: Unload / Dismantle MapKit on Window Close
1. Nil out `hostingController.rootView` or destroy the `deskWindow` instance on close (`deskWindow = nil`), recreating it when `showDeskWindow()` is called.
2. This drops idle RAM footprint from **181.5 MB down to ~35 MB** and terminates the `CVDisplayLink` thread when the window is not in use.

### Strategy E: Background QoS and Timer Tolerance
1. Dispatch periodic background fetches with `.utility` priority instead of `.userInitiated`, allowing macOS to schedule them onto E-cores.
2. Apply explicit `.tolerance` to all scheduled timers (`timer.tolerance = interval * 0.1`) to enable hardware-level timer coalescing.

---

## 5. Summary Matrix

| Problem | Root Cause | Target Fix | Expected Energy Savings |
| :--- | :--- | :--- | :--- |
| **Heavy 2-min CPU spike** | Unconditional 8-frame MRMS radar fetch in background | Suspend radar refresh when desk window is closed | **~85% total reduction** in background CPU instructions |
| **Constant menu bar redraw** | 800ms zero-tolerance `pipTimer` modifying `NSStatusItem` | Static alert badge or hardware `NSSymbolPulseEffect` | Eliminates **4,500 wakeups/hr** & `WindowServer` GPU compositing |
| **5-min subprocess bursts** | Unconditional polling of CPC, SPC, and Grid favorites | Lazy fetch only when respective tab is visible | Cuts 4–8 process forks per 5m down to 1 every 15m |
| **High idle memory (180MB+)** | Retained `MKMapView` and Metal `CVDisplayLink` | Dismantle window on close; re-instantiate on show | **~80% RAM reduction** when window closed |
| **Offscreen loop animation** | 380ms `loopTask` continues when window closed | Auto-pause radar loop on window resign/hide | Eliminates sub-second background UI invalidations |
