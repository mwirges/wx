# wx iOS shell (SwiftUI). No team ID or certs in the repo.
# Debug build / ui-test: pinned simulator, signing off.
# archive: unsigned Release device .xcarchive (generic/platform=iOS). Does not export or upload.
# Local App Group entitlements live in WxIOS.entitlements and WxWidget.entitlements.
XCODEGEN ?= /opt/homebrew/bin/xcodegen
SCHEME   := WxIOS
CONFIG   := Debug
DERIVED  := build
# Xcode 27, runtime iOS 27.0, device "iPhone 17". The id lives only here.
# name= is omitted: xcodebuild will not match this UDID when the name contains a space.
IOS_SIMULATOR := platform=iOS Simulator,id=C5357135-AE11-4E33-B95C-B4D7D9031047,OS=27.0
# Xcode 27, runtime iOS 27.0, device "iPad Pro 11-inch (M5)". The id lives only here.
IPAD_SIMULATOR := platform=iOS Simulator,id=B9A7FDAF-37E8-45BE-A3DA-5902B7A84D63,OS=27.0

comma := ,
empty :=
space := $(empty) $(empty)
IOS_SIM_ID := $(patsubst id=%,%,$(filter id=%,$(subst $(comma),$(space),$(IOS_SIMULATOR))))

APP := $(DERIVED)/Build/Products/$(CONFIG)-iphonesimulator/wx.app
SIGN_OVERRIDES := CODE_SIGNING_ALLOWED=NO CODE_SIGNING_REQUIRED=NO CODE_SIGN_IDENTITY= DEVELOPMENT_TEAM=
ARCHIVE := $(DERIVED)/WxIOS.xcarchive
ARCHIVE_DERIVED := $(DERIVED)/archive-derived

.DEFAULT_GOAL := build

.PHONY: project build sim ui-test check-app archive

## project: Regenerate the Xcode project via xcodegen
project:
	$(XCODEGEN) generate --spec project.yml

## build: xcodegen + xcodebuild for the pinned simulator, no signing
build: project
	xcodebuild -project WxIOS.xcodeproj -scheme $(SCHEME) -configuration $(CONFIG) \
	  -destination '$(IOS_SIMULATOR)' -derivedDataPath $(DERIVED) \
	  $(SIGN_OVERRIDES) build
	@$(MAKE) check-app

## check-app: Bundle id, When-In-Use purpose string, and no Always / entitlements
check-app:
	@set -eu; \
	INFO="$(APP)/Info.plist"; \
	test -f "$$INFO"; \
	id="$$(plutil -extract CFBundleIdentifier raw -o - "$$INFO")"; \
	test "$$id" = "com.mwirges.wx.ios"; \
	family="$$(plutil -extract UIDeviceFamily json -o - "$$INFO" | tr -d '[:space:]')"; \
	test "$$family" = "[1,2]"; \
	plutil -extract NSLocationWhenInUseUsageDescription raw -o - "$$INFO" >/dev/null; \
	if plutil -extract NSLocationAlwaysAndWhenInUseUsageDescription raw -o - "$$INFO" >/dev/null 2>&1; then \
	  echo "ios: NSLocationAlwaysAndWhenInUseUsageDescription is set" >&2; exit 1; \
	fi; \
	if plutil -extract NSLocationAlwaysUsageDescription raw -o - "$$INFO" >/dev/null 2>&1; then \
	  echo "ios: NSLocationAlwaysUsageDescription is set" >&2; exit 1; \
	fi; \
	if find "$(APP)" -name '*.entitlements' | grep -q .; then \
	  echo "ios: entitlements file in the app bundle" >&2; exit 1; \
	fi; \
	for name in forecast nowcast radar spc tropics; do \
	  if ! find "$(APP)" -name "$$name.json" | grep -q .; then \
	    echo "ios: missing fixture $$name.json" >&2; exit 1; \
	  fi; \
	done; \
	live="$$(plutil -extract NSSupportsLiveActivities raw -o - "$$INFO")"; \
	test "$$live" = "true"; \
	APPEX="$$(find "$(APP)/PlugIns" -maxdepth 1 -name '*.appex' | head -1)"; \
	test -n "$$APPEX"; \
	test -f "$$APPEX/Info.plist"; \
	wid="$$(plutil -extract CFBundleIdentifier raw -o - "$$APPEX/Info.plist")"; \
	test "$$wid" = "com.mwirges.wx.ios.widget"; \
	wfamily="$$(plutil -extract UIDeviceFamily json -o - "$$APPEX/Info.plist" | tr -d '[:space:]')"; \
	test "$$wfamily" = "[1,2]"; \
	point="$$(plutil -extract NSExtension.NSExtensionPointIdentifier raw -o - "$$APPEX/Info.plist")"; \
	test "$$point" = "com.apple.widgetkit-extension"; \
	test -f "$$APPEX/WxWidget"; \
	if find "$$APPEX" -name 'WxCore*' | grep -q .; then \
	  echo "ios: widget links WxCore" >&2; exit 1; \
	fi; \
	bytes="$$(wc -c < "$$APPEX/WxWidget" | tr -d ' ')"; \
	if [ "$$bytes" -gt 8388608 ]; then \
	  echo "ios: widget binary is $$bytes bytes" >&2; exit 1; \
	fi; \
	echo "ios app ok $$id family $$family widget $$wid family $$wfamily simulator $(IOS_SIM_ID)"

## sim: Install and launch on the pinned simulator. Fixture data, no live NWS.
sim: build
	xcrun simctl boot "$(IOS_SIM_ID)" >/dev/null 2>&1 || true
	xcrun simctl bootstatus "$(IOS_SIM_ID)" -b
	xcrun simctl install "$(IOS_SIM_ID)" "$(APP)"
	SIMCTL_CHILD_WX_FIXTURE=1 \
	SIMCTL_CHILD_WX_UI_TEST=1 \
	SIMCTL_CHILD_WX_SKIP_LOCATION=1 \
	SIMCTL_CHILD_WX_DEFAULT_LOCATION="New Haven, IN" \
	xcrun simctl launch --terminate-running-process "$(IOS_SIM_ID)" com.mwirges.wx.ios

## ui-test: iPhone widget + shell UI tests (fixtures and the live cache bridge), then the iPad split.
ui-test: build
	xcodebuild -project WxIOS.xcodeproj -scheme $(SCHEME) -configuration $(CONFIG) \
	  -destination '$(IOS_SIMULATOR)' -derivedDataPath $(DERIVED) \
	  -parallel-testing-enabled NO \
	  -maximum-concurrent-test-simulator-destinations 1 \
	  -only-testing:WxWidgetTests \
	  -only-testing:WxIOSUITests \
	  -skip-testing:WxIOSUITests/WxPadSplitUITests \
	  $(SIGN_OVERRIDES) test
	xcodebuild -project WxIOS.xcodeproj -scheme $(SCHEME) -configuration $(CONFIG) \
	  -destination '$(IPAD_SIMULATOR)' -derivedDataPath $(DERIVED) \
	  -parallel-testing-enabled NO \
	  -maximum-concurrent-test-simulator-destinations 1 \
	  -only-testing:WxIOSUITests/WxPadSplitUITests \
	  $(SIGN_OVERRIDES) test

## archive: Unsigned Release device xcarchive, then check bundle contents. No export.
archive: project
	@test -f ../build/WxCore.xcframework/ios-arm64/libwxcore.a || \
	  (echo "ios: missing device slice ../build/WxCore.xcframework/ios-arm64/libwxcore.a" >&2; exit 1)
	@sed -i '' 's/revealArchiveInOrganizer = "YES"/revealArchiveInOrganizer = "NO"/' \
	  WxIOS.xcodeproj/xcshareddata/xcschemes/WxIOS.xcscheme
	rm -rf "$(ARCHIVE)"
	xcodebuild -project WxIOS.xcodeproj -scheme $(SCHEME) -configuration Release \
	  -destination 'generic/platform=iOS' \
	  -archivePath "$(ARCHIVE)" \
	  -derivedDataPath "$(ARCHIVE_DERIVED)" \
	  $(SIGN_OVERRIDES) \
	  ENABLE_BITCODE=NO \
	  DEBUG_INFORMATION_FORMAT=dwarf-with-dsym \
	  VALIDATE_PRODUCT=NO \
	  archive
	@../scripts/ios-archive-check.sh "$(CURDIR)/$(ARCHIVE)"
