# wx — iOS Mobile Architecture & Engineering Specification

**Document Version:** 1.0  
**Status:** Draft / For Review  
**Date:** 2026-09-28  
**Target Platform:** iOS 17.0+ / iPadOS 17.0+ (Universal SwiftUI App)

---

## 1. Executive Summary & Goals

The goal of this specification is to define the architecture for extending `wx` into a mobile application for iPhone and iPad while **preserving 100% of the core Go implementation**.

### Core Tenets
1. **Single Source of Truth**: The Go engine (`internal/*`) remains the sole provider of meteorological ingestion, caching, astronomical calculation, thermodynamic sounding analysis, nowcasting precipitation math, and radar compositing.
2. **Zero Commercial APIs**: Retain the project's zero-config, zero-signup, pure public open-meteo/NOAA foundation.
3. **App Store 100% Compliant**: Strictly adhere to Apple sandbox rules and App Store Review Guidelines (no subprocesses, no JIT, no banned APIs).
4. **Negligible Battery Impact**: Design for modern iOS power management—ephemeral execution bursts, cooperative thread parking, zero idle CPU, and radio-tail-energy minimization.
5. **Maximum Code Reuse**: Share `WeatherModels.swift`, `WxTheme.swift`, and all console HUD views with the existing macOS app.

---

## 2. Platform Constraints & Architectural Pivot

### 2.1 The iOS Sandbox Dilemma
On macOS, `wx.app` bundles the Go CLI as a standalone helper binary (`wx-cli`) and executes it using `Foundation.Process` (formerly `NSTask`), communicating over stdout/stderr JSON pipes.

On iOS, **Apple's sandbox strictly forbids spawning child processes**. 
- `fork()`, `exec()`, and `posix_spawn()` are blocked at the kernel level and return `EPERM`.
- `Foundation.Process` does not exist in the iOS SDK.
- You cannot bundle an executable binary inside an `.ipa` and invoke it via CLI pipes.

### 2.2 The Solution: Go In-Process C-Archive / XCFramework
Instead of compiling an executable binary, Go compiles into a universal static C-archive (`-buildmode=c-archive`) packaged into an Apple **XCFramework** (`WxCore.xcframework`):

```
+-------------------------------------------------------------------+
|                        iOS App Process Space                       |
|                                                                   |
|   +-----------------------------------------------------------+   |
|   |                   SwiftUI Application Layer               |   |
|   |      (Views, ViewModels, WidgetKit, CoreLocation)         |   |
|   +-----------------------------------------------------------+   |
|                                 |                                 |
|                                 v Swift FFI / async               |
|   +-----------------------------------------------------------+   |
|   |                    WxBridge (Swift C-API)                 |   |
|   +-----------------------------------------------------------+   |
|                                 |                                 |
|                                 v Cgo Export Interface            |
|   +-----------------------------------------------------------+   |
|   |             WxCore.xcframework (Static Library)           |   |
|   |                                                           |   |
|   |   +-------------------+   +---------------------------+   |   |
|   |   |  Go Runtime       |   |  pkg/mobile Bridge        |   |   |
|   |   |  (GC, Scheduler)  |   |  (Cgo String / Data API)  |   |   |
|   |   +-------------------+   +---------------------------+   |   |
|   |             |                           |                 |   |
|   |             v                           v                 |   |
|   |   +-----------------------------------------------+       |   |
|   |   |  Core Go Subsystems                           |       |   |
|   |   |  - internal/provider (NWS / Open-Meteo)       |       |   |
|   |   |  - internal/radar (MRMS / NEXRAD / Compositor)|       |   |
|   |   |  - internal/sounding (NSHARP / Thermodynamics)|       |   |
|   |   |  - internal/nowcast (15m Precip Timeline)     |       |   |
|   |   |  - internal/climate (ACIS Normals / Records)  |       |   |
|   |   |  - internal/tropics (NHC Storm Tracker)       |       |   |
|   |   |  - internal/astro (Offline Solar / Lunar)     |       |   |
|   |   +-----------------------------------------------+       |   |
|   +-----------------------------------------------------------+   |
+-------------------------------------------------------------------+
```

### 2.3 Advantages Over Subprocesses
1. **Zero Spawning Latency**: Eliminates the 20–50ms process creation and teardown overhead on every fetch.
2. **Direct Memory Transfers**: Large payloads (e.g. multi-frame radar base64 PNGs) pass directly in memory via pointers rather than piped OS file descriptors.
3. **Unified Codebase**: The macOS desktop app can optionally adopt the same in-process XCFramework bridge, obsoleting subprocess pipes entirely.

---

## 3. Go ⇄ Swift In-Process Bridge Specification

### 3.1 Go Cgo Export Package (`pkg/mobile/bridge.go`)
A dedicated, lightweight package exposes C-compatible entry points:

```go
package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"strings"
	"time"
	"unsafe"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	// Providers & subsystems
)

//export WxRunJSON
func WxRunJSON(cArgs *C.char) *C.char {
	argsStr := C.GoString(cArgs)
	args := strings.Fields(argsStr)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	jsonOutput, err := executeInternal(ctx, args)
	if err != nil {
		jsonOutput = `{"error": "` + err.Error() + `"}`
	}

	return C.CString(jsonOutput)
}

//export WxFreeString
func WxFreeString(ptr *C.char) {
	C.free(unsafe.Pointer(ptr))
}

func main() {} // Required for -buildmode=c-archive
```

### 3.2 Swift FFI Wrapper (`WxBridge.swift`)
Swift manages memory lifecycles safely and executes calls asynchronously off the main thread:

```swift
import Foundation
import WxCore // Imported from WxCore.xcframework

enum WxBridge {
    static func run(_ command: String) async throws -> String {
        try await Task.detached(priority: .userInitiated) {
            guard let cCommand = command.cString(using: .utf8) else {
                throw WxBridgeError.invalidCommandString
            }

            guard let cResult = WxRunJSON(cCommand) else {
                throw WxBridgeError.nullResponse
            }
            defer { WxFreeString(cResult) }

            return String(cString: cResult)
        }.value
    }
}
```

### 3.3 Swift Store Integration
The existing `WeatherStore.swift` and `WeatherModels.swift` consume the bridge with zero model changes:

```swift
func refreshNowcast(location: String) async {
    do {
        let jsonStr = try await WxBridge.run("nowcast \(location) --json")
        let payload = try JSONDecoder().decode(NowcastPayloadDTO.self, from: Data(jsonStr.utf8))
        self.nowcastPayload = payload
    } catch {
        self.errorMessage = error.localizedDescription
    }
}
```

---

## 4. Battery, Energy & iOS Lifecycle Management

Mobile power management is fundamentally different from desktop. The following practices ensure minimal battery consumption:

### 4.1 Cooperative Go Thread Parking
- In Go 1.14+, when all goroutines are idle, the runtime enters cooperative sleep (`runtime.notesleep`) on Mach semaphores.
- In `wx`, an update runs for **150–300ms** to scrape/parse feeds, and then Go drops to **0.0% CPU**. 
- It does not spin, poll, or burn background cycles.

### 4.2 Handling iOS App Lifecycle (Suspension)
- When the user leaves `wx` or locks the phone, iOS allows ~5 seconds of cleanup time and then sends `SIGSTOP`, freezing the entire process in RAM.
- While suspended, the app consumes **0 CPU cycles, 0 network bandwidth, and 0 battery**.
- On foreground return (`scenePhase == .active`), the app checks timestamps and only triggers a refresh if the data exceeds the standard 5-minute cache TTL.

### 4.3 Avoiding the 4 Mobile Battery Traps

| Threat | The Risk | `wx` Mitigation |
| :--- | :--- | :--- |
| **GPS Tracking** | Continuous tracking (`startUpdatingLocation`) burns **10–15% battery/hr**. | Use **One-Shot Location** (`CLLocationManager.requestLocation()`). Powers up the GPS radio, gets a fix in ~1.5s, and powers down immediately. For tracking while moving, use **Significant Change Service** (cell-tower handoffs only; ~0% battery). |
| **Cellular Tail Energy** | Staggered HTTP calls keep the 5G/LTE radio in high-power state for 10–15s after each call. | **Batch all requests in parallel**. Fire conditions, nowcast, alerts, and radar simultaneously. The cellular modem powers up once, transfers all data in <1s, and returns to sleep. |
| **Radar Animation** | 60fps loops burning GPU/CPU when screen is locked or tab is hidden. | Automatically pause animations on `scenePhase != .active`. On iPhone the loop runs only on the Radar tab. On the iPad split desk the loop runs while the radar pane is on screen, and pauses when that pane is gone. Render radar loops at **10–15 fps** (matching radar scan dwell intervals). Cache decoded `CGImage` frames in memory; never re-decode base64 PNGs on every frame. |
| **Memory / Jetsam Kills** | Go's default `GOGC=100` allows heap to double before GC runs, risking OS termination. | Call `debug.SetMemoryLimit(40 * 1024 * 1024)` (`GOMEMLIMIT=40MiB`) and `debug.SetGCPercent(50)` at startup. Keeps peak resident memory well below iOS warning thresholds. |

---

## 5. UI Architecture & Screen Adaptation

### 5.1 Reusable macOS Code (No Changes Required)
- **`WeatherModels.swift`**: All DTO structs (`ConditionsDTO`, `ForecastDTO`, `AlertDTO`, `NowcastPayloadDTO`, `SoundingReportDTO`, `TropicsPayloadDTO`, `ClimateReportDTO`) are pure Swift `Codable`.
- **`WxTheme.swift`**: All SNW cyberpunk colors, font definitions, and corner brackets (`SNWCornerBrackets`, `SNWConsoleCard`, `SNWMetricTile`).
- **Telemetry Views**: `NowBlockView`, `PeriodsListView`, `NowcastCardView`, `StormChaseView`, `TropicsView`, `ClimateTrendsView`, `CommandGridView`.

### 5.2 iOS Navigation Shell
On iPhone, and whenever the horizontal size class is compact, replace macOS `NSWindow` / `NSStatusItem` / `NSPopover` with an iOS-native `TabView`:

```
+-----------------------------------------------------------+
| [Header] NEW HAVEN, IN      [SYNC // 7:42 PM]    [RADAR]   |
+-----------------------------------------------------------+
|                                                           |
|                     ACTIVE VIEW TAB                       |
|                                                           |
|   - SURFACE:     NowBlock + Nowcast + 5-Day Forecast      |
|   - TACTICAL:    Full Sensor Matrix + Spaceweather + AQI  |
|   - RADAR:       Interactive Mosaic + Loop Transport      |
|   - SEVERE:      SPC Convective Outlooks + Storm Chase    |
|   - TROPICS:     NHC Storm Tracker + Invest Outlooks      |
|                                                           |
+-----------------------------------------------------------+
|  [☀️ Surface] [📡 Radar] [⚡ Severe] [🌀 Tropics] [⚙️ Config]|
+-----------------------------------------------------------+
```

### 5.3 iPadOS Adaptive Layout

`TARGETED_DEVICE_FAMILY` is `1,2` (iPhone and iPad). The Xcode project stays simulator-only: `SDKROOT` is `iphonesimulator`, signing is off, and there is no Apple Developer portal, team, or certificate. The widget extension uses the same device family, still does not link `WxCore`, and still does not call NWS.

The shell reads the horizontal size class and the device idiom. The split is shown only when both say iPad and regular width. A nil size class stays on the phone tabs until SwiftUI reports regular.

- **iPad, regular width** (full-screen iPad, portrait or landscape): a split desk, the same idea as the Mac dual desk (`DeskWindowView` when the desk tab is dual and the window is wide). Atmospheric telemetry is the left pane: the existing `SurfaceView` (observation, nowcast, days, hourly), titled `TELEMETRY`, identifier `desk.telemetry`. The radar array is the right pane: the existing `RadarTabView`, titled `RADAR`, identifier `desk.radar`. The pair is `desk.split`. The left column is 40% of the width, clamped to 340–480 points. A one-point cyan rule separates the panes. The custom tab bar stays. Surface and Radar both show this desk. Tactical, Severe, Tropics, and Config stay one full-width page. The radar loop runs while this desk is on screen, because the radar pane is visible, and pauses when the user leaves the desk or the scene is inactive.
- **iPhone, and any compact width** (including iPad Slide Over and a narrow split): the `TabView` in §5.2 is unchanged. The radar loop still runs only on the Radar tab. The split identifiers are absent.

The Go core stays the only weather implementation. The split is a SwiftUI arrangement of the same shell views. It does not fetch weather on its own. `refreshSurface` already batches the forecast, nowcast, alerts, and radar in one radio wake, so opening the desk does not add a second radar call.

---

## 6. Mobile-Native Capabilities (WidgetKit & Live Activities)

### 6.1 Lock Screen & Home Screen Widgets (`WidgetExtension`)
WidgetKit extensions run in a separate process with a strict **30MB RAM ceiling**. The extension bundle id is `com.mwirges.wx.ios.widget`. It does not link `WxCore` and it does not call NWS. The Go core remains the only weather implementation. The shell writes a small snapshot after each forecast/nowcast refresh.

- **Lock Screen**:
  - `accessoryInline`: `☀️ 72° · Dry next 6h`
  - `accessoryRectangular`: Current temp, high/low, and the precip headline plus a short sparkline.
  - `accessoryCircular`: Current temperature.
- **Home Screen**:
  - `systemSmall`: Current observation card + one radar thumbnail.
  - `systemMedium`: Precip headline, up to six hourly chips, and up to three forecast periods.
- **Radar thumbnail**: One frame, the current `image_base64` only (not the loop). ImageIO downsamples it to a longest edge of 72 px. The widget ignores a file larger than 32 KB or an image whose longest edge is over 72 px, so a full scan cannot land in the widget process.
- **Data sharing**: The app and the widget read `widget-snapshot.json` and `radar-thumb.png` from the App Group container `group.com.mwirges.wx`. Those ids are in the local entitlements files used for simulator builds. There is no Apple Developer portal registration in this slice. When `containerURL(forSecurityApplicationGroupIdentifier:)` is nil, the shell writes the same files under Caches/`wx-widget-cache`. That fallback is per-process, so the widget only sees the snapshot when the group container is available.

### 6.2 Live Activities & Dynamic Island
The Live Activity widget lives in the same widget extension. Updates are local ActivityKit only: `Activity.request` / `update` / `end` with `pushType: nil`. No APNs, push token, or server. The app `Info.plist` sets `NSSupportsLiveActivities`.

**Trigger rule.** The window is the next 60 minutes from the shell clock.

- **Start** when the nowcast shows precipitation onset in that window. Onset is `next_precip_time` from now through 60 minutes, precipitation already falling (`is_active_precip`), or a precipitating nowcast interval that overlaps the window. A precipitating interval has phase rain, snow, freezing-rain, ice-pellets, or mixed, or a positive rate or accumulation. Probability alone does not count.
- **Update** the running activity locally when a newer nowcast still shows precipitation in the window.
- **End** the activity when a nowcast shows no precipitation in the window.

Dynamic Island compact is `💧` plus the minutes label (`15m`, or `now` when precipitation is already falling). Expanded shows the temperature, the minutes label, the intensity sparkline, and the cessation time when the nowcast has one. The lock screen shows the same fields.

---

## 7. Toolchain & Build System (`Makefile`)

`make ios-framework` builds `./pkg/mobile` (`-buildmode=c-archive`, `CGO_ENABLED=1`, `GOOS=ios`, `GOARCH=arm64`) for two slices and packages `build/WxCore.xcframework`.

- Device arm64 uses clang from `xcrun --sdk iphoneos --find clang`, `-isysroot` set to the iphoneos SDK, and `-target arm64-apple-ios17.0`.
- Simulator arm64 uses clang from `xcrun --sdk iphonesimulator --find clang`, `-isysroot` set to the iphonesimulator SDK, and `-target arm64-apple-ios17.0-simulator`.
- Each slice gets a `Headers` directory with the generated header and `module.modulemap` (`module WxCore`).
- An existing `build/WxCore.xcframework` is removed first. `xcodebuild -create-xcframework -library ... -headers ...` writes the new one.

`make ios-framework-test` depends on `ios-framework`. It checks that `Info.plist` lists `ios-arm64` and `ios-arm64-simulator` (`plutil` and `PlistBuddy`), that each slice has `Headers` and `Headers/module.modulemap`, and that `nm` shows `WxRunJSON` and `WxFreeString` exported from each slice archive. Any miss exits nonzero.

`make clean` removes `build/`, including the xcframework. `make ios-build` builds the simulator shell and the widget extension (`com.mwirges.wx.ios.widget`) for iPhone and iPad (`UIDeviceFamily` 1 and 2). `make ios-ui-test` runs the widget timeline tests and the shell UI tests on the pinned iPhone simulator (`IOS_SIMULATOR` in `ios/Makefile`), then `WxPadSplitUITests` on the pinned iPad simulator (`IPAD_SIMULATOR`). Those tests use bundled fixtures. They do not call NWS. The iPad test asserts `desk.split`, `desk.telemetry`, and `desk.radar` are on screen together, along with the surface and radar fixture strings.

---

## 8. Implementation Milestones

1. **Milestone 1: The Go Mobile Bridge (`pkg/mobile`)**
   - Implement `bridge.go` exposing `WxRunJSON` and memory release functions.
   - Verify static compilation and link against a test C command-line tool.
2. **Milestone 2: `WxCore.xcframework` Automation**
   - `make ios-framework` assembles `build/WxCore.xcframework` (device arm64 and simulator arm64, headers, `module WxCore`).
   - `make ios-framework-test` checks the slices, module map, and exported `WxRunJSON` / `WxFreeString`.
3. **Milestone 3: iOS Shell & SwiftUI Binding**
   - Create `ios/` Xcode project referencing `WxCore.xcframework`.
   - Implement `WxBridge.swift` and connect to existing `WeatherStore`.
   - Wire `TabView` with `Surface`, `Tactical`, `Radar`, and `Tropics`. On iPad regular width, show the split desk from §5.3.
4. **Milestone 4: Power & Touch Polish**
   - Hook up One-Shot CoreLocation GPS.
   - Implement background suspension pauses for radar animations.
   - Configure memory limits (`GOMEMLIMIT=40MiB`).
5. **Milestone 5: Widgets & Live Activities**
   - Implement `WidgetExtension` using App Group shared disk cache.
   - Implement Dynamic Island nowcasting timeline.
   - Live Activity trigger: start on precipitation onset within 60 minutes, update locally as the nowcast changes, end when the window is dry. See §6.2.
