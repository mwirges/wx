# wx Makefile
#
# Default target builds for the current OS/arch.
# Use `make build-all` for cross-platform releases.
# All artifacts land in ./build/ and are removed by `make clean`.

MODULE  := github.com/mwirges/wx
BINARY  := wx
CMD     := .

BUILD_DIR := build

# Detect host platform.
GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Embed version: check VERSION file first, then git tag, otherwise fallback.
VERSION := $(strip $(shell cat VERSION 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || echo 1.1.0))
LDFLAGS := -ldflags "-X github.com/mwirges/wx/cmd.Version=$(VERSION)"

# Cross-compilation targets: OS/ARCH pairs.
PLATFORMS := \
  darwin/amd64 \
  darwin/arm64 \
  linux/amd64  \
  linux/arm64  \
  windows/amd64

.DEFAULT_GOAL := build

# ── Primary targets ──────────────────────────────────────────────────────────

## build: Compile for the current OS/arch → build/wx[.exe]
.PHONY: build
build:
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BUILD_DIR)/$(call exe,$(BINARY),$(GOOS)) $(CMD)
	@echo "Built $(BUILD_DIR)/$(call exe,$(BINARY),$(GOOS))  [$(GOOS)/$(GOARCH)]"

## test: Run the full test suite
.PHONY: test
test:
	go test ./... -count=1

## test-verbose: Run tests with verbose output
.PHONY: test-verbose
test-verbose:
	go test ./... -v -count=1

## vet: Run go vet, including the Linux desk module
.PHONY: vet
vet:
	go vet ./...
	$(MAKE) -C linux vet

## mobile-bridge-test: Build the c-archive bridge and compare it with wx --json
.PHONY: mobile-bridge-test
mobile-bridge-test:
	@./scripts/mobile-bridge-test.sh

# iOS 17 is the floor in docs/ios-mobile-architecture.md.
# CC is clang from `xcrun --sdk <sdk> --find clang`, plus -isysroot and -target,
# so each c-archive carries the device or simulator platform.
IOS_MIN := 17.0

## ios-framework: Build device and simulator arm64 into build/WxCore.xcframework
.PHONY: ios-framework
ios-framework:
	@echo "==> Compiling Go C-archive for iOS device (arm64)..."
	@set -eu; \
	sdk="$$(xcrun --sdk iphoneos --show-sdk-path)"; \
	clang="$$(xcrun --sdk iphoneos --find clang)"; \
	mkdir -p "$(BUILD_DIR)/ios/arm64-device/Headers"; \
	CGO_ENABLED=1 GOOS=ios GOARCH=arm64 \
	CC="\"$$clang\" -isysroot \"$$sdk\" -target arm64-apple-ios$(IOS_MIN)" \
	go build -buildmode=c-archive -o "$(BUILD_DIR)/ios/arm64-device/libwxcore.a" ./pkg/mobile; \
	cp "$(BUILD_DIR)/ios/arm64-device/libwxcore.h" "$(BUILD_DIR)/ios/arm64-device/Headers/libwxcore.h"; \
	printf '%s\n' 'module WxCore {' '    header "libwxcore.h"' '    export *' '}' \
	  > "$(BUILD_DIR)/ios/arm64-device/Headers/module.modulemap"
	@echo "==> Compiling Go C-archive for iOS simulator (arm64)..."
	@set -eu; \
	sdk="$$(xcrun --sdk iphonesimulator --show-sdk-path)"; \
	clang="$$(xcrun --sdk iphonesimulator --find clang)"; \
	mkdir -p "$(BUILD_DIR)/ios/arm64-simulator/Headers"; \
	CGO_ENABLED=1 GOOS=ios GOARCH=arm64 \
	CC="\"$$clang\" -isysroot \"$$sdk\" -target arm64-apple-ios$(IOS_MIN)-simulator" \
	go build -buildmode=c-archive -o "$(BUILD_DIR)/ios/arm64-simulator/libwxcore.a" ./pkg/mobile; \
	cp "$(BUILD_DIR)/ios/arm64-simulator/libwxcore.h" "$(BUILD_DIR)/ios/arm64-simulator/Headers/libwxcore.h"; \
	printf '%s\n' 'module WxCore {' '    header "libwxcore.h"' '    export *' '}' \
	  > "$(BUILD_DIR)/ios/arm64-simulator/Headers/module.modulemap"
	@echo "==> Packaging $(BUILD_DIR)/WxCore.xcframework..."
	rm -rf "$(BUILD_DIR)/WxCore.xcframework"
	xcodebuild -create-xcframework \
	  -library "$(BUILD_DIR)/ios/arm64-device/libwxcore.a" -headers "$(BUILD_DIR)/ios/arm64-device/Headers" \
	  -library "$(BUILD_DIR)/ios/arm64-simulator/libwxcore.a" -headers "$(BUILD_DIR)/ios/arm64-simulator/Headers" \
	  -output "$(BUILD_DIR)/WxCore.xcframework"
	@echo "Built $(BUILD_DIR)/WxCore.xcframework"

## ios-framework-test: Check WxCore.xcframework slices, module map, and exports
.PHONY: ios-framework-test
ios-framework-test: ios-framework
	@./scripts/ios-framework-test.sh

## build-all: Cross-compile for all supported platforms → build/wx-{os}-{arch}[.exe]
.PHONY: build-all
build-all: $(foreach p,$(PLATFORMS),build-$(subst /,-,$(p)))

# Generate one phony rule per platform using target-specific variables so that
# each rule captures its own OS and ARCH at expansion time.
define PLATFORM_RULE
.PHONY: build-$(subst /,-,$(1))
build-$(subst /,-,$(1)): _OS   := $(word 1,$(subst /, ,$(1)))
build-$(subst /,-,$(1)): _ARCH := $(word 2,$(subst /, ,$(1)))
build-$(subst /,-,$(1)):
	@mkdir -p $(BUILD_DIR)
	GOOS=$$(_OS) GOARCH=$$(_ARCH) go build $(LDFLAGS) \
	  -o $(BUILD_DIR)/$(BINARY)-$$(_OS)-$$(_ARCH)$$(call ext,$$(_OS)) \
	  $(CMD)
	@echo "Built $(BUILD_DIR)/$(BINARY)-$$(_OS)-$$(_ARCH)$$(call ext,$$(_OS))"
endef
$(foreach p,$(PLATFORMS),$(eval $(call PLATFORM_RULE,$(p))))

## clean: Remove the build directory
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	@echo "Removed $(BUILD_DIR)/"

## help: Show this help message
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | column -t -s ':'

# ── Helpers ──────────────────────────────────────────────────────────────────

# exe(name, os) → name.exe on Windows, name elsewhere.
exe = $(1)$(call ext,$(2))

# ext(os) → .exe on Windows, empty elsewhere.
ext = $(if $(filter windows,$(1)),.exe,)

# ── Mac app (SwiftUI under macos/) ───────────────────────────────────────────

## mac-build: Build the macOS SwiftUI app (macos/)
.PHONY: mac-build
mac-build:
	$(MAKE) -C macos build

## linux-desk: Build the CLI and the Fyne desk binary
.PHONY: linux-desk
linux-desk:
	$(MAKE) -C linux desk

## linux-test: Test the Fyne desk shell. Not part of `make test`.
.PHONY: linux-test
linux-test:
	$(MAKE) -C linux test

## linux-run: Build and launch the Fyne desk
.PHONY: linux-run
linux-run:
	$(MAKE) -C linux run

## linux-deb: Package the Linux CLI and desk as build/wx_VERSION_arch.deb
.PHONY: linux-deb
linux-deb:
	$(MAKE) -C linux deb VERSION=$(VERSION)

## mac-ui-test: XCUITest the Mac app. Not part of `make test`.
.PHONY: mac-ui-test
mac-ui-test:
	$(MAKE) -C macos ui-test

## mac-run: Build and open the macOS app
.PHONY: mac-run
mac-run:
	$(MAKE) -C macos run

## mac-dmg: Create a distributable macOS DMG installer (build/wx.dmg)
.PHONY: mac-dmg
mac-dmg:
	$(MAKE) -C macos dmg

## dmg: Alias for mac-dmg
.PHONY: dmg
dmg: mac-dmg

## mac-notarize: Notarize and staple build/wx.dmg with Apple Notary Service
.PHONY: mac-notarize
mac-notarize:
	$(MAKE) -C macos notarize

## notarize: Alias for mac-notarize
.PHONY: notarize
notarize: mac-notarize

# ── Release Pipeline ──────────────────────────────────────────────────────────

## release: Build, test, package DMG, commit, tag, and publish GitHub release (Usage: make release [VERSION=X.Y.Z])
.PHONY: release
release:
	@./scripts/release.sh $(if $(VERSION),--version $(VERSION),--current)

## release-patch: Bump patch version and release
.PHONY: release-patch
release-patch:
	@./scripts/release.sh --patch

## release-minor: Bump minor version and release
.PHONY: release-minor
release-minor:
	@./scripts/release.sh --minor

## release-major: Bump major version and release
.PHONY: release-major
release-major:
	@./scripts/release.sh --major

## bump: Synchronize and bump version metadata across all project files (Usage: make bump [VERSION=X.Y.Z])
.PHONY: bump
bump:
	@./scripts/bump-version.sh $(if $(VERSION),--version $(VERSION),--minor)


