.DEFAULT_GOAL := build

export NAME ?= $(shell basename $(shell pwd))
export VERSION := v$(shell cat VERSION)
export OUT_PATH ?= out
export CGO_ENABLED ?= 0

export CHECKSUMS_FILE_NAME := release.sha256
export CHECKSUMS_FILE := $(OUT_PATH)/$(CHECKSUMS_FILE_NAME)

OS_ARCHS ?= darwin/amd64 darwin/arm64 \
			freebsd/amd64 freebsd/arm64 freebsd/386 freebsd/arm \
			linux/amd64 linux/arm64 linux/arm linux/s390x linux/ppc64le linux/386 \
			windows/amd64.exe windows/386.exe
GO_LDFLAGS ?= '-extldflags "-static"'

build:
	@mkdir -p $(OUT_PATH)
	go build -a -ldflags $(GO_LDFLAGS) -o $(OUT_PATH)/$(NAME) ./cmd/$(NAME)/...

.PHONY: .mods
.mods:
	@go mod download

TARGETS = $(foreach OSARCH,$(OS_ARCHS),${OUT_PATH}/$(NAME)-$(subst /,-,$(OSARCH)))

$(TARGETS): .mods
	@mkdir -p $(OUT_PATH)
	GOOS=$(firstword $(subst -, ,$(subst $(OUT_PATH)/$(NAME)-,,$@))) \
			 GOARCH=$(lastword $(subst .exe,,$(subst -, ,$(subst $(OUT_PATH)/$(NAME)-,,$@)))) \
			 go build -a -ldflags $(GO_LDFLAGS) -o $@ ./cmd/$(NAME)/...

MAKEFLAGS += -j$(shell nproc)
all:$(TARGETS)

.PHONY: test
test:
	go test ./...

.PHONY: shellcheck
shellcheck:
	shellcheck $(shell find ci -name "*.sh")

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

.PHONY: do-release
do-release:
	git tag -s $(VERSION) -m "Version $(VERSION)"
	git push origin $(VERSION)
