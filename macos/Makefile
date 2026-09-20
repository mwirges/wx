# wx Mac app (SwiftUI) — slice 0.1a
XCODEGEN ?= /opt/homebrew/bin/xcodegen
SCHEME   := WxMac
CONFIG   := Debug
DERIVED  := build

.DEFAULT_GOAL := build

.PHONY: project build run clean test-go

## project: Regenerate Xcode project via xcodegen
project:
	$(XCODEGEN) generate

## build: xcodegen + xcodebuild Debug
build: project
	xcodebuild -project WxMac.xcodeproj -scheme $(SCHEME) -configuration $(CONFIG) -derivedDataPath $(DERIVED) build

## run: build then open the app
run: build
	open "$(DERIVED)/Build/Products/$(CONFIG)/wx.app"

## clean: remove derived data + xcodeproj
clean:
	rm -rf $(DERIVED) WxMac.xcodeproj

## test-go: run Go suite from repo root (sanity)
test-go:
	$(MAKE) -C .. test
