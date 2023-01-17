.DEFAULT_GOAL := build
NAME ?= $(shell basename $(shell pwd))
export VERSION := v$(shell cat VERSION)
export OUT_PATH ?= bin
export CGO_ENABLED ?= 0

BUILD_ARCHS ?= -arch '386' -arch 'arm' -arch 'amd64' -arch 'arm64' -arch 's390x' -arch 'ppc64le'
BUILD_PLATFORMS ?= -osarch 'darwin/amd64' -osarch 'darwin/arm64' -os 'linux' -os 'freebsd' -os 'windows' ${BUILD_ARCHS}
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

.PHONY: clean
clean:
	rm -fr $(OUT_PATH)
