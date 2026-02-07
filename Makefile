.PHONY: build test lint clean run

# Build variables
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(shell git rev-parse --short HEAD 2>/dev/null) -X main.date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Default target
all: build

build:
	go build -ldflags "$(LDFLAGS)" -o pulse ./cmd/pulse

build-race:
	go build -race -ldflags "$(LDFLAGS)" -o pulse ./cmd/pulse

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

clean:
	rm -f pulse coverage.out coverage.html

run: build
	./pulse start --addr localhost:3002

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/pulse

# Database operations (for future persistence layer)
db-migrate:
	@echo "Migration not yet implemented"

db-reset:
	@echo "Reset not yet implemented"

# Release preparation
release: clean build
	@echo "Building release $(VERSION)"
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o pulse-darwin-amd64 ./cmd/pulse
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o pulse-darwin-arm64 ./cmd/pulse
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o pulse-linux-amd64 ./cmd/pulse
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o pulse-linux-arm64 ./cmd/pulse
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o pulse-windows-amd64.exe ./cmd/pulse
	@echo "Release binaries created"
