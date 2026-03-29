BIN=akctl
PKG="github.com/elliottpolk/akctl"
BIN_VERSION=`cat .version`
MAJOR_VERSION=`cut -d. -f 1 < .version`

COMPILED=`date +%s`
COMMIT=`git rev-parse --short HEAD`

GOOS?=linux
GOARCH?=amd64

BUILD_DIR=./build/bin
BUILD_CANDIDATE_TAG=dc

M = $(shell printf "\033[34;1m◉\033[0m")

default: all ;                                              		@ ## defaulting to clean and build

.PHONY: all
all: clean build

.PHONY: clean
clean: ; $(info $(M) running clean ...)                             @ ## clean up the old build dir
	@rm -vrf build "v$(MAJOR_VERSION)"

.PHONY: test
test: unit-test;													@ ## wrapper to run all testing

.PHONY: update-deps
update-deps: ;														@ ## pull the latest deps
	@go mod edit -go=$$(go version | awk '{print $$3}' | tr -d 'go' | cut -d. -f1-2)
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go get -v -u ./...
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go mod tidy

.PHONY: unit-test
unit-test: update-deps; $(info $(M) running unit tests ...)          @ ## run the unit tests
	@go test -v -cover ./...

.PHONY: build
build: build-dir update-deps; $(info $(M) building ...)             @ ## build the binary
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-ldflags "-X main.version=$(BIN_VERSION) -X main.compiled=$(COMPILED) -X main.githash=$(COMMIT)" \
		-o ./build/bin/$(BIN) ./cmd/*.go

.PHONY: build-dir
build-dir: ;
	@[ ! -d "${BUILD_DIR}" ] && mkdir -vp "${BUILD_DIR}" || true

.PHONY: install
install: ; $(info $(M) installing locally ...) 						@ ## install binary locally
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-ldflags "-X main.version=$(BIN_VERSION) -X main.compiled=$(COMPILED) -X main.githash=$(COMMIT)" \
		-o $(GOPATH)/bin/$(BIN) ./cmd/*.go

.PHONY: build-cli
build-cli: clean build-dir update-deps; $(info $(M) building various cli versions ...)			@ ## build out the various OS and architecture builds
	@GOOS=linux GOARCH=amd64 go build \
		-ldflags "-X main.version=$(BIN_VERSION) -X main.compiled=$(COMPILED) -X main.githash=$(COMMIT)" \
		-o ./build/bin/$(BIN)-linux-amd64 ./cmd/*.go
	@GOOS=windows GOARCH=amd64 go build \
		-ldflags "-X main.version=$(BIN_VERSION) -X main.compiled=$(COMPILED) -X main.githash=$(COMMIT)" \
		-o ./build/bin/$(BIN)-windows-amd64.exe ./cmd/*.go
	@GOOS=darwin GOARCH=amd64 go build \
		-ldflags "-X main.version=$(BIN_VERSION) -X main.compiled=$(COMPILED) -X main.githash=$(COMMIT)" \
		-o ./build/bin/$(BIN)-darwin-amd64 ./cmd/*.go
	@GOOS=darwin GOARCH=arm64 go build \
		-ldflags "-X main.version=$(BIN_VERSION) -X main.compiled=$(COMPILED) -X main.githash=$(COMMIT)" \
		-o ./build/bin/$(BIN)-darwin-arm64 ./cmd/*.go
	@tree ./build/bin

.PHONY: help
help:
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

