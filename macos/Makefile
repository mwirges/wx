# wx Mac app (SwiftUI) — slice 0.1a / W1 quiet build
XCODEGEN ?= /opt/homebrew/bin/xcodegen
SCHEME   := WxMac
CONFIG   := Debug
DERIVED  := build
DEST     := platform=macOS,arch=arm64

.DEFAULT_GOAL := build

.PHONY: project build run clean test-go build-cli embed-cli

## build-cli: Ensure the Go CLI is built in repo root
build-cli:
	$(MAKE) -C .. build

## project: Regenerate Xcode project via xcodegen
project:
	$(XCODEGEN) generate

## build: build-cli + xcodegen + xcodebuild Debug (macOS arm64 only) + embed-cli
build: build-cli project
	xcodebuild -project WxMac.xcodeproj -scheme $(SCHEME) -configuration $(CONFIG) \
	  -destination '$(DEST)' -derivedDataPath $(DERIVED) build
	@$(MAKE) embed-cli

## embed-cli: Copy build/wx into wx.app/Contents/MacOS/wx-cli and sign
embed-cli:
	@mkdir -p "$(DERIVED)/Build/Products/$(CONFIG)/wx.app/Contents/MacOS"
	cp -f "../build/wx" "$(DERIVED)/Build/Products/$(CONFIG)/wx.app/Contents/MacOS/wx-cli"
	chmod +x "$(DERIVED)/Build/Products/$(CONFIG)/wx.app/Contents/MacOS/wx-cli"
	codesign --force --sign - --timestamp=none "$(DERIVED)/Build/Products/$(CONFIG)/wx.app/Contents/MacOS/wx-cli" 2>/dev/null || true

## run: build then open the app
run: build
	open "$(DERIVED)/Build/Products/$(CONFIG)/wx.app"

## clean: remove derived data + xcodeproj
clean:
	rm -rf $(DERIVED) WxMac.xcodeproj

## test-go: run Go suite from repo root (sanity)
test-go:
	$(MAKE) -C .. test
