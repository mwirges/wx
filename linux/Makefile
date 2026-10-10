# Fyne desk shell. Weather stays in the Go CLI.
ROOT := $(abspath ..)

.PHONY: desk test vet run

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
