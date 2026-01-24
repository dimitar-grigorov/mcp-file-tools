.PHONY: build test lint clean install run help

BINARY_NAME=mcp-file-tools
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

## build: Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/mcp-file-tools

## test: Run tests
test:
	go test -v -race ./...

## test-cover: Run tests with coverage
test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## lint: Run linters
lint:
	go vet ./...
	go fmt ./...

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -f coverage.out coverage.html
	rm -rf dist/

## install: Install binary to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/mcp-file-tools

## run: Build and run
run: build
	./$(BINARY_NAME)

## tidy: Tidy go modules
tidy:
	go mod tidy

## build-all: Build for all platforms
build-all:
	mkdir -p dist
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/mcp-file-tools
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-arm64.exe ./cmd/mcp-file-tools
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/mcp-file-tools
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/mcp-file-tools
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/mcp-file-tools
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/mcp-file-tools

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/ /'

.DEFAULT_GOAL := help
