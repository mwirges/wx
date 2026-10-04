# GTK desk shell. Weather stays in the Go CLI.
ROOT := $(abspath ..)

.PHONY: desk test run

## desk: Build the wx CLI and syntax-check the GTK shell
desk:
	$(MAKE) -C $(ROOT) build
	python3 -m compileall -q $(ROOT)/linux/wxdesk
	chmod +x $(ROOT)/linux/wx-desk

## test: Shell tests. Uses WX_CONFIG under /tmp. Does not read the user config.
test:
	WX_CONFIG=/tmp/wx-linux-test-config.json PYTHONPATH=$(ROOT)/linux python3 -m unittest discover -s $(ROOT)/linux/tests -v

## run: Build and open the desk
run: desk
	$(ROOT)/linux/wx-desk
