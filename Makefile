VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")

.PHONY: build test lint

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/launchpad ./cmd/launchpad

test:
	go test ./... -v

lint:
	go vet ./...

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/launchpad-linux-amd64 ./cmd/launchpad
