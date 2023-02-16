.DEFAULT_GOAL := build
NAME ?= $(shell basename $(shell pwd))
export VERSION := v$(shell cat VERSION)
export OUT_PATH ?= out
export CGO_ENABLED ?= 0

OS_ARCHS ?= darwin/amd64 darwin/arm64 \
			freebsd/amd64 freebsd/arm64 freebsd/386 freebsd/arm \
			linux/amd64 linux/arm64 linux/arm linux/s390x linux/ppc64le linux/386 \
			windows/amd64.exe windows/386.exe
GO_LDFLAGS ?= '-extldflags "-static"'

build:
	@mkdir -p $(OUT_PATH)
	go build -a -ldflags $(GO_LDFLAGS) -o $(OUT_PATH)/$(NAME) ./cmd/$(NAME)/...

TARGETS = $(foreach OSARCH,$(OS_ARCHS),${OUT_PATH}/$(NAME)-$(subst /,-,$(OSARCH)))

$(TARGETS):
	@mkdir -p $(OUT_PATH)
	GOOS=$(firstword $(subst -, ,$(subst $(OUT_PATH)/$(NAME)-,,$@))) \
			 GOARCH=$(lastword $(subst .exe,,$(subst -, ,$(subst $(OUT_PATH)/$(NAME)-,,$@)))) \
			 go build -a -ldflags $(GO_LDFLAGS) -o $@ ./cmd/$(NAME)/...

MAKEFLAGS += -j
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
upload-release: generate-checksums-file
upload-release:
	ci/upload-release.sh

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
