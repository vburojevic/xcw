# XcodeConsoleWatcher (xcw) Makefile

BINARY_NAME=xcw
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

LDFLAGS=-ldflags "-X github.com/vburojevic/xcw/internal/cli.Version=$(VERSION) -X github.com/vburojevic/xcw/internal/cli.Commit=$(COMMIT)"

.PHONY: all build clean test install uninstall release help

all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/xcw

## build-release: Build optimized release binary
build-release:
	@echo "Building release $(BINARY_NAME)..."
	CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/xcw

## install: Install to /usr/local/bin
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	install -m 755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

## uninstall: Remove from /usr/local/bin
uninstall:
	@echo "Removing $(BINARY_NAME) from /usr/local/bin..."
	rm -f /usr/local/bin/$(BINARY_NAME)

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf dist/

## test: Run tests
test:
	@echo "Running tests..."
	go test -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## lint: Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

## docs: Regenerate machine-readable help docs (docs/help.json)
docs:
	./scripts/gen-readme.sh

## schema: Regenerate JSON schema from CLI output
schema:
	./scripts/gen-schema.sh

## tidy: Tidy modules
tidy:
	@echo "Tidying modules..."
	go mod tidy

## run-list: Run list command (for testing)
run-list: build
	./$(BINARY_NAME) list

## run-tail: Run tail command (for testing)
run-tail: build
	./$(BINARY_NAME) tail -a com.apple -l debug

## release: Build release with goreleaser
release:
	@echo "Building release with goreleaser..."
	goreleaser release --clean

## release-snapshot: Build snapshot release
release-snapshot:
	@echo "Building snapshot release..."
	goreleaser release --snapshot --clean

## help: Show this help
help:
	@echo "XcodeConsoleWatcher (xcw) - Build Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
