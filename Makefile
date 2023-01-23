.DEFAULT_GOAL := build
NAME ?= $(shell basename $(shell pwd))
export VERSION := v$(shell cat VERSION)
export OUT_PATH ?= bin
export CGO_ENABLED ?= 0

BUILD_ARCHS ?= -arch '386' -arch 'arm' -arch 'amd64' -arch 'arm64' -arch 's390x' -arch 'ppc64le'
BUILD_PLATFORMS ?= -osarch 'darwin/amd64' -osarch 'darwin/arm64' -os 'linux' -os 'freebsd' -os 'windows' ${BUILD_ARCHS}
OS_ARCHS ?= darwin/amd64 darwin/arm64 \
			freebsd/amd64 freebsd/arm64 freebsd/386 freebsd/arm \
			linux/amd64 linux/arm64 linux/arm linux/s390x linux/ppc64le linux/386 \
			windows/amd64.exe windows/386.exe
GO_LDFLAGS ?= '-extldflags "-static"'

GOX=gox

$(GOX):
	@go install github.com/mitchellh/gox@v1.0.1

build:
	@mkdir -p $(OUT_PATH)
	go build -a -ldflags $(GO_LDFLAGS) -o $(OUT_PATH)/$(NAME) ./cmd/$(NAME)/...

all: $(GOX)
	# Building $(NAME) version $(VERSION) for $(BUILD_PLATFORMS)
	$(GOX) $(BUILD_PLATFORMS) \
		-ldflags $(GO_LDFLAGS) \
		-output="$(OUT_PATH)/$(NAME)-{{.OS}}-{{.Arch}}" \
		./...

TARGETS = $(foreach OSARCH,$(OS_ARCHS),${OUT_PATH}/$(NAME)-$(subst /,-,$(OSARCH)))

$(TARGETS):
	@mkdir -p $(OUT_PATH)
	GOOS=$(firstword $(subst -, ,$(subst $(OUT_PATH)/$(NAME)-,,$@))) \
			 GOARCH=$(lastword $(subst .exe,,$(subst -, ,$(subst $(OUT_PATH)/$(NAME)-,,$@)))) \
			 go build -a -ldflags $(GO_LDFLAGS) -o $@ ./cmd/$(NAME)/...

MAKEFLAGS += -j
all-2:$(TARGETS)

.PHONY: clean
clean:
	rm -fr $(OUT_PATH)
