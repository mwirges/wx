# iOS TestFlight prep

Audit of the iPhone and iPad app in `ios/` on branch `chore/ios-testflight-prep`. This change prepares an unsigned Release device archive. It does not sign, export, upload, or touch the developer portal.

Marketing version is `VERSION` (currently 1.2.0). The iOS app and the widget extension share that version and one build number (currently 1). `scripts/bump-version.sh` updates both plists together. The macOS build number stays on its own counter.

## Already OK

- Bundle IDs were already `com.mwirges.wx.ios` (app, product name `wx`) and `com.mwirges.wx.ios.widget` (extension, executable `WxWidget`, package type `XPC!`, extension point `com.apple.widgetkit-extension`). The same extension hosts the home-screen widget and the Live Activity.
- `NSLocationWhenInUseUsageDescription` was already set. The shell calls `requestWhenInUseAuthorization` and one-shot `requestLocation()`. It does not request Always, start significant-change updates, or call `requestTemporaryFullAccuracyAuthorization`. There is no Always purpose string, and `make` `check-app` still rejects one if it appears.
- No other purpose strings are required. The app does not touch the camera, photo library, microphone, contacts, Bluetooth, local network, tracking, or Face ID. Radar stays in memory and in the app-group file. It is not saved to Photos.
- App Group `group.com.mwirges.wx` was already the only entitlement on both targets (`WxIOS.entitlements`, `WxWidget.entitlements`). The app writes `widget-snapshot.json` and `radar-thumb.png` through `FileManager.containerURL`. The widget only reads those files. Neither target uses `UserDefaults` or the app-group defaults suite.
- `UILaunchScreen` was already an empty dictionary (modern launch screen, no storyboard).
- `UIApplicationSceneManifest` was already present. The process is a SwiftUI `App` / `WindowGroup`.
- iPhone orientations were already portrait and both landscapes. iPad orientations were already all four. `UIRequiresFullScreen` is unset, so iPad multitasking stays available.
- `TARGETED_DEVICE_FAMILY` was already `1,2` (iPhone and iPad). Deployment target was already iOS 17.0, which is the floor in `docs/ios-mobile-architecture.md`.
- `NSSupportsLiveActivities` was already true on the app. `NowcastLiveActivitySync` requests a local activity with `pushType: nil`. There is no frequent-updates key and no push entitlement. Live Activities do not need a separate portal capability for this local-only use. Do not turn on Push Notifications unless the app later sends push updates.
- The widget must not link `WxCore`. `check-app` still enforces that. The extension reads the shared cache only.
- `WxCore.xcframework` is a static c-archive (`embed: false`), built `-target arm64-apple-ios17.0` for the device slice and the matching simulator triple for Debug. It is not an embedded `.framework`, so App Store checks for a framework-bundle `CFBundleShortVersionString` / `MinimumOSVersion` do not apply to a copied WxCore bundle. Those keys are on the app and the appex instead. The device binary's minimum OS is checked in the archive.
- Bitcode is off. Architectures are arm64 only (`EXCLUDED_ARCHS` drops i386 and x86_64).
- Transport is HTTPS only (`https://` weather, radar, and geocoder hosts). Clients keep Go's default `http.Transport`: no `InsecureSkipVerify`, no private root pool. On iOS, `crypto/x509` verifies through the platform trust store (see README "TLS and certificate authorities"). The record protocol is standard TLS in `crypto/tls`, not a private cipher and not file encryption. That is exempt encryption, so `ITSAppUsesNonExemptEncryption` is false. `notarytool` is the macOS notarization path and is not used for this iOS upload.

## Missing, and fixed here

- No app icon. The 1024×1024 marketing image is generated from `macos/WxMac/AppIcon-1024.png`. The Mac artwork has an alpha channel (transparent corners around the rounded radar). App Store rejects a marketing icon with alpha, so the iOS image is flattened onto black and stored as 8-bit RGB with no alpha at `ios/WxIOS/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png`. Xcode 14 and later treat that single universal iOS 1024 image as the full icon set, including the App Store marketing icon. `CFBundleIconName` is `AppIcon`.
- No widget icon. WidgetKit and Live Activities draw the containing app's icon. An extension icon set is not required. A partial one fails validation, so the appex has no `CFBundleIconName` and no asset catalog.
- No privacy manifests. Added:
  - `ios/WxIOS/PrivacyInfo.xcprivacy` for the app
  - `ios/WxWidget/PrivacyInfo.xcprivacy` for the extension
  - `ios/WxCore/PrivacyInfo.xcprivacy`, copied to the root of `build/WxCore.xcframework` by `make ios-framework` so the static archive's manifest travels with the framework
- `NSPrivacyTracking` is false on all three. There are no tracking domains.
- Collected data, app only: precise location and coarse location, each not linked to identity, not used for tracking, purpose App Functionality. `IOSLocation` asks for `kCLLocationAccuracyHundredMeters` and formats a fallback coordinate to four decimal places, which is precise location. If the person turns Precise Location off, Core Location still returns a reduced-accuracy fix and the app uses it, which is coarse location. The widget does not call Core Location. Its manifest collects nothing.
- Required-reason APIs actually referenced by the device `c-archive` (`nm -u` on `libwxcore.a`): `stat`, `lstat`, `fstat`, `fstatat`, and `mach_absolute_time`. Not referenced: `statfs` / `fstatfs` (disk space), `getattrlist`, `NSUserDefaults`, `activeInputModes`. `sysctl` is present because the Go runtime reads `hw.ncpu` and `hw.pagesize`. It is not `kern.boottime`, and `sysctl` by itself is not a required-reason API.
  - File timestamp `C617.1`: Go `os` stats cache and config files inside the app container. The value is not shown to the user and not sent off device. Declared on the app and on WxCore.
  - System boot time `35F9.1`: the Go runtime's monotonic clock (`mach_absolute_time`) for timers, the scheduler, and GC. Those readings are not sent off device. Declared on the app and on WxCore.
  - UserDefaults `CA92.1` / `1C8F.1` are not declared. The app group is a directory of files, not a defaults suite.
  - The widget manifest leaves `NSPrivacyAccessedAPITypes` empty. `make ios-archive` fails if the built app or appex imports a required-reason symbol the manifest does not name.
- Version scheme. `scripts/bump-version.sh` now sets `CFBundleShortVersionString` from `VERSION` on `ios/WxIOS/Info.plist` and `ios/WxWidget/Info.plist`, and writes the same `CFBundleVersion` to both. A normal bump increments the iOS build from the app plist. `--current` leaves the build number alone. The macOS build number is still independent.
- `ITSAppUsesNonExemptEncryption` = false on the app and the widget. `MinimumOSVersion` 17.0 is in both source plists and is checked on the built bundles.
- Release no longer uses the simulator SDK. In `ios/project.yml`, Debug keeps `SDKROOT=iphonesimulator` and `SUPPORTED_PLATFORMS=iphonesimulator`. Release uses `iphoneos` and `dwarf-with-dsym`. Debug simulator builds still pass `CODE_SIGNING_ALLOWED=NO`. Signing stays off in the project. `DEVELOPMENT_TEAM` stays empty.
- `ios/ExportOptions.plist`: method `app-store-connect`, `signingStyle` automatic, `destination` export, `uploadSymbols` true, `manageAppVersionAndBuildNumber` false. `teamID` is the placeholder `$(WX_TEAM_ID)`. `xcodebuild -exportArchive` does not expand that string. Substitute it into a temp plist at upload time. Do not commit a real team ID.
- `make ios-archive` (root `Makefile` and `ios/Makefile`) builds a Release `.xcarchive` for `generic/platform=iOS` with `CODE_SIGNING_ALLOWED=NO`, then `scripts/ios-archive-check.sh` checks the archive. It is not part of `make test`. The target does not run `-exportArchive`, `altool`, `notarytool`, or Transporter. The archive is `ios/build/WxIOS.xcarchive` (gitignored). It is unsigned and is not the archive to upload.
- The widget embed uses code sign on copy so a later signed archive re-signs the appex. The unsigned prep build still passes `CODE_SIGNING_ALLOWED=NO`.

## Archive checks

`scripts/ios-archive-check.sh` requires all of the following:

- `wx.app` and `WxWidget.appex`, executables `wx` and `WxWidget`, arm64 only, `platform IOS`, `minos 17.0`, `DTPlatformName` `iphoneos`, no simulator platform, no bitcode, no team identifier on the signature
- Matching marketing version (`VERSION`) and build number on the app, the appex, and the archive's `ApplicationProperties`
- Bundle IDs, device family `[1,2]`, launch screen, scene manifest, iPad's four orientations, Live Activities, When In Use purpose string, no Always keys, encryption flag false
- `Assets.car` contains `AppIcon` and a 1024 image. The source marketing PNG is 1024×1024 with `hasAlpha` no
- Privacy manifests in the app and the appex, tracking false, precise and coarse location on the app only
- Required-reason symbols in each binary are covered by that binary's manifest
- dSYMs for the app and the widget
- WxCore is not copied into the app or the appex. If a `Frameworks` bundle appears later, its Info.plist must have `CFBundleShortVersionString` and `MinimumOSVersion`
- Source entitlements still name `group.com.mwirges.wx`, and `ExportOptions.plist` still has the placeholder team ID

## Still outside the repo

Portal records, the distribution certificate, the App Store Connect app, nutrition labels, TestFlight testers, and the signed export are Matt's steps. The command list is `~/code/tmp/wx-testflight-checklist.md`.
