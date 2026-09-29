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
| **Radar Animation** | 60fps loops burning GPU/CPU when screen is locked or tab is hidden. | Automatically pause animations on `scenePhase != .active` or when navigating away from the radar tab. Render radar loops at **10–15 fps** (matching radar scan dwell intervals). Cache decoded `CGImage` frames in memory; never re-decode base64 PNGs on every frame. |
| **Memory / Jetsam Kills** | Go's default `GOGC=100` allows heap to double before GC runs, risking OS termination. | Call `debug.SetMemoryLimit(40 * 1024 * 1024)` (`GOMEMLIMIT=40MiB`) and `debug.SetGCPercent(50)` at startup. Keeps peak resident memory well below iOS warning thresholds. |

---

## 5. UI Architecture & Screen Adaptation

### 5.1 Reusable macOS Code (No Changes Required)
- **`WeatherModels.swift`**: All DTO structs (`ConditionsDTO`, `ForecastDTO`, `AlertDTO`, `NowcastPayloadDTO`, `SoundingReportDTO`, `TropicsPayloadDTO`, `ClimateReportDTO`) are pure Swift `Codable`.
- **`WxTheme.swift`**: All SNW cyberpunk colors, font definitions, and corner brackets (`SNWCornerBrackets`, `SNWConsoleCard`, `SNWMetricTile`).
- **Telemetry Views**: `NowBlockView`, `PeriodsListView`, `NowcastCardView`, `StormChaseView`, `TropicsView`, `ClimateTrendsView`, `CommandGridView`.

### 5.2 iOS Navigation Shell
Replace macOS `NSWindow` / `NSStatusItem` / `NSPopover` with an iOS-native `TabView`:

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
On iPad (and iPhone in landscape), the interface automatically expands to a **Split-Pane Command Console** (reusing the desktop `Dual` mode), displaying the Radar array side-by-side with Atmospheric Telemetry.

---

## 6. Mobile-Native Capabilities (WidgetKit & Live Activities)

### 6.1 Lock Screen & Home Screen Widgets (`WidgetExtension`)
WidgetKit extensions run in a separate process with a strict **30MB RAM ceiling**:
- **Lock Screen**:
  - `accessoryInline`: `☀️ 72° · Dry next 6h`
  - `accessoryRectangular`: Current temp, high/low, and 15-min precip sparkline.
- **Home Screen**:
  - `systemSmall`: Current observation card + radar thumbnail.
  - `systemMedium`: 6-hour precip timeline + 3-day forecast.
- **Data Sharing**: The iOS App and WidgetExtension share a local cache folder using an **Apple App Group container** (`group.com.mwirges.wx`), allowing the widget to read cached Go payloads instantly without redundant network calls.

### 6.2 Live Activities & Dynamic Island
When active precipitation is detected:
- The app automatically registers an `Activity<NowcastAttributes>`.
- Displays real-time rain countdowns on the Lock Screen and in the Dynamic Island:
  - *Dynamic Island Compact*: `💧 15m`
  - *Dynamic Island Expanded*: Unicode intensity sparkline + estimated cessation time.

---

## 7. Toolchain & Build System (`Makefile`)

To build the iOS static library and framework on macOS:

```makefile
# Target Architectures
IOS_SDK = $(shell xcrun --sdk iphoneos --show-sdk-path)
IOS_SIM_SDK = $(shell xcrun --sdk iphonesimulator --show-sdk-path)

.PHONY: ios-framework
ios-framework:
	@echo "==> Compiling Go C-Archive for iOS Device (arm64)..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
	CC="xcrun --sdk iphoneos clang -arch arm64" \
	go build -buildmode=c-archive -o build/ios/arm64-device/libwxcore.a ./pkg/mobile

	@echo "==> Compiling Go C-Archive for iOS Simulator (arm64)..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
	CC="xcrun --sdk iphonesimulator clang -arch arm64" \
	go build -buildmode=c-archive -o build/ios/arm64-sim/libwxcore.a ./pkg/mobile

	@echo "==> Packaging Apple XCFramework..."
	rm -rf build/WxCore.xcframework
	xcodebuild -create-xcframework \
		-library build/ios/arm64-device/libwxcore.a -headers build/ios/arm64-device/libwxcore.h \
		-library build/ios/arm64-sim/libwxcore.a -headers build/ios/arm64-sim/libwxcore.h \
		-output build/WxCore.xcframework
	@echo "==> Successfully generated build/WxCore.xcframework"

.PHONY: ios-build
ios-build: ios-framework
	@echo "==> Building iOS Application..."
	xcodebuild -project ios/WxApp.xcodeproj -scheme WxApp -destination 'generic/platform=iOS' build

.PHONY: ios-sim
ios-sim: ios-framework
	@echo "==> Running in iOS Simulator..."
	xcodebuild -project ios/WxApp.xcodeproj -scheme WxApp -destination 'platform=iOS Simulator,name=iPhone 16' -derivedDataPath build/ios-sim
	xcrun simctl boot "iPhone 16" || true
	xcrun simctl install booted build/ios-sim/Build/Products/Debug-iphonesimulator/WxApp.app
	xcrun simctl launch booted com.mwirges.wx.ios
```

---

## 8. Implementation Milestones

1. **Milestone 1: The Go Mobile Bridge (`pkg/mobile`)**
   - Implement `bridge.go` exposing `WxRunJSON` and memory release functions.
   - Verify static compilation and link against a test C command-line tool.
2. **Milestone 2: `WxCore.xcframework` Automation**
   - Implement `make ios-framework` to assemble the universal `.xcframework`.
   - Add automated headers and modulemap.
3. **Milestone 3: iOS Shell & SwiftUI Binding**
   - Create `ios/` Xcode project referencing `WxCore.xcframework`.
   - Implement `WxBridge.swift` and connect to existing `WeatherStore`.
   - Wire `TabView` with `Surface`, `Tactical`, `Radar`, and `Tropics`.
4. **Milestone 4: Power & Touch Polish**
   - Hook up One-Shot CoreLocation GPS.
   - Implement background suspension pauses for radar animations.
   - Configure memory limits (`GOMEMLIMIT=40MiB`).
5. **Milestone 5: Widgets & Live Activities**
   - Implement `WidgetExtension` using App Group shared disk cache.
   - Implement Dynamic Island nowcasting timeline.
