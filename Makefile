.DEFAULT_GOAL := build

export NAME ?= $(shell basename $(shell pwd))
export VERSION := v1.4.3 # x-releaser-pleaser-version
export OUT_PATH ?= out
export CGO_ENABLED ?= 0

export CHECKSUMS_FILE_NAME := release.sha256
export CHECKSUMS_FILE := $(OUT_PATH)/$(CHECKSUMS_FILE_NAME)

REVISION := $(shell git rev-parse --short=8 HEAD || echo unknown)
REFERENCE := $(shell git show-ref | grep "$(REVISION)" | grep -v HEAD | awk '{print $$2}' | sed 's|refs/remotes/origin/||' | sed 's|refs/heads/||' | sort | head -n 1)
BUILT := $(shell date -u +%Y-%m-%dT%H:%M:%S%z)
PKG = $(shell go list .)

OS_ARCHS ?= darwin/amd64 darwin/arm64 \
			freebsd/amd64 freebsd/arm64 freebsd/386 freebsd/arm \
			linux/amd64 linux/arm64 linux/arm linux/s390x linux/ppc64le linux/386 \
			windows/amd64.exe windows/386.exe
GO_LDFLAGS ?= -X $(PKG).NAME=$(NAME) -X $(PKG).VERSION=$(VERSION) \
              -X $(PKG).REVISION=$(REVISION) -X $(PKG).BUILT=$(BUILT) \
              -X $(PKG).REFERENCE=$(REFERENCE) \
              -w -extldflags '-static'

build:
	@mkdir -p $(OUT_PATH)
	go build -a -ldflags "$(GO_LDFLAGS)" -o $(OUT_PATH)/$(NAME) ./cmd/$(NAME)/...

.PHONY: .mods
.mods:
	go mod download

TARGETS = $(foreach OSARCH,$(OS_ARCHS),${OUT_PATH}/$(NAME)-$(subst /,-,$(OSARCH)))

$(TARGETS): .mods
	@mkdir -p $(OUT_PATH)
	GOOS=$(firstword $(subst -, ,$(subst $(OUT_PATH)/$(NAME)-,,$@))) \
			 GOARCH=$(lastword $(subst .exe,,$(subst -, ,$(subst $(OUT_PATH)/$(NAME)-,,$@)))) \
			 go build -a -ldflags "$(GO_LDFLAGS)" -o $@ ./cmd/$(NAME)/...

MAKEFLAGS += -j$(shell nproc)
all:$(TARGETS)

.PHONY: test
test: .mods
	go test -v -timeout=30m ./...

coverage:
	go test -v -timeout=30m -coverprofile=coverage.tmp -covermode count ./...
	grep -v -e 'zz_.*.go' coverage.tmp > coverage.txt
	go run github.com/boumenot/gocover-cobertura < coverage.txt > coverage.xml
	go tool cover -func=coverage.txt

.PHONY: clean
clean:
	rm -fr $(OUT_PATH)

.PHONY: upload-release
upload-release: sign-checksums-file
upload-release:
	ci/upload-release.sh

.PHONY: sign-checksums-file
sign-checksums-file: generate-checksums-file
	ci/sign-checksums-file.sh

.PHONY: generate-checksums-file
generate-checksums-file:
	ci/generate-checksums-file.sh

.PHONY: release
release:
	ci/release.sh

release-oci-artifacts:
	ci/release-oci-artifacts.sh
