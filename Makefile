VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint build-linux build-darwin build-all

build:
	go build -ldflags "$(LDFLAGS)" -o bin/launchpad ./cmd/launchpad

test:
	go test ./... -v

lint:
	go vet ./...

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/launchpad-linux-amd64 ./cmd/launchpad

build-darwin:
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/launchpad-darwin-arm64 ./cmd/launchpad

build-all: build-linux build-darwin
