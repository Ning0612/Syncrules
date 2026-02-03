# Syncrules Makefile

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Output directories
DIST := dist

.PHONY: all build build-gui clean test lint fmt vet deps help

all: build

## build: Build the CLI binary
build:
	go build $(LDFLAGS) -o $(DIST)/syncrules ./cmd/syncrules

## build-gui: Build the GUI binary
build-gui:
	go build $(LDFLAGS) -o $(DIST)/syncrules-gui ./cmd/syncrules-gui

## build-all: Build both CLI and GUI
build-all: build build-gui

## clean: Remove build artifacts
clean:
	rm -rf $(DIST)
	go clean -cache

## test: Run all tests
test:
	go test -v ./...

## test-cover: Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## lint: Run linter
lint:
	golangci-lint run

## fmt: Format code
fmt:
	go fmt ./...
	goimports -w .

## vet: Run go vet
vet:
	go vet ./...

## deps: Download dependencies
deps:
	go mod download
	go mod tidy

## deps-update: Update dependencies
deps-update:
	go get -u ./...
	go mod tidy

## install: Install the CLI to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/syncrules

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/ /'
