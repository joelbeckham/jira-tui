.PHONY: build test lint clean run build-all

BINARY := jira-tui
BUILD_DIR := bin
VERSION ?= dev
LDFLAGS := -s -w

## build: Compile the binary for the current platform
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/jira-tui

## build-all: Cross-compile for Linux amd64, macOS ARM, and Windows amd64
build-all:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64       ./cmd/jira-tui
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64      ./cmd/jira-tui
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/jira-tui

## run: Build and run the TUI
run: build
	./$(BUILD_DIR)/$(BINARY)

## test: Run all tests
test:
	go test ./... -v

## test-short: Run tests without verbose output
test-short:
	go test ./...

## test-coverage: Run tests with coverage report
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

## lint: Run linter
lint:
	golangci-lint run ./...

## fmt: Format all Go files
fmt:
	gofmt -w .

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

## help: Show this help message
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
