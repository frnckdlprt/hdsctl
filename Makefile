.DEFAULT_GOAL := build

VERSION := 0.1
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GOFLAGS := -mod=mod
LD_FLAGS= \
	-X github.com/frnckdlprt/hdsctl/version.Version=$(VERSION) \
	-X github.com/frnckdlprt/hdsctl/version.BuildDate=$(BUILD_DATE)

GO_SOURCE := $(shell find ./ -name "*.go")

build: build/hdsctl

setup:
	sudo chown $(USER):$(USER) /dev/bus/usb/$(shell lsusb | grep PDS6062T | awk '{print $$2 "/" substr($$4,1,length($$4)-1)}')

build/hdsctl: $(GO_SOURCE)
	@mkdir -p $(@D)
	@go build $(GOFLAGS) -o $@ -ldflags="$(LD_FLAGS)" cmd/hdsctl/main.go

.PHONY: clean
clean:
	rm -rf build

