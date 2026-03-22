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

# Embed version from git tag if available, otherwise "dev".
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

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

## vet: Run go vet
.PHONY: vet
vet:
	go vet ./...

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
