# Fyne desk shell. Weather stays in the Go CLI.
ROOT := $(abspath ..)
VERSION ?= $(shell cat $(ROOT)/VERSION)
ARCH ?= $(shell dpkg --print-architecture 2>/dev/null || go env GOARCH)

.PHONY: desk test vet run deb

## desk: Build the wx CLI and the Fyne desk binary
desk:
	$(MAKE) -C $(ROOT) build
	mkdir -p $(ROOT)/build
	CGO_ENABLED=1 go build -o $(ROOT)/build/wx-desk ./cmd/wxdesk

## test: Shell tests. Uses WX_CONFIG under /tmp. Does not read the user config.
test:
	WX_CONFIG=/tmp/wx-linux-test-config.json CGO_ENABLED=1 go test ./... -count=1

## vet: vet the desk module
vet:
	CGO_ENABLED=1 go vet ./...

## run: Build and open the desk
run: desk
	$(ROOT)/build/wx-desk

## deb: Package the CLI and the desk as build/wx_VERSION_ARCH.deb
deb: desk
	@rm -rf $(ROOT)/build/deb-root
	go run ./packaging/stage \
		-icon $(ROOT)/macos/WxMac/AppIcon-1024.png \
		-templates packaging \
		-dest $(ROOT)/build/deb-root \
		-bins $(ROOT)/build \
		-version $(VERSION) \
		-arch $(ARCH) \
		-date $(shell date -u +%F)
	dpkg-deb --root-owner-group -Zxz --build $(ROOT)/build/deb-root \
		$(ROOT)/build/wx_$(VERSION)_$(ARCH).deb
	@list=$(ROOT)/build/deb.list; \
		dpkg-deb -c $(ROOT)/build/wx_$(VERSION)_$(ARCH).deb > $$list; \
		grep -q './usr/bin/wx$$' $$list; \
		grep -q './usr/bin/wx-desk$$' $$list; \
		grep -q './usr/share/applications/wx-desk.desktop$$' $$list; \
		grep -q './usr/share/icons/hicolor/256x256/apps/wx.png$$' $$list; \
		rm -f $$list
	@echo "Built $(ROOT)/build/wx_$(VERSION)_$(ARCH).deb"
