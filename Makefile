.PHONY: build test lint clean run build-all tools

BINARY := jira-tui
BUILD_DIR := bin
MY_DIR := /mnt/c/tools/jira-tui
VERSION ?= dev
LDFLAGS := -s -w
RSRC := $(shell go env GOPATH)/bin/rsrc
ICON := icon.ico
SYSO := cmd/jira-tui/rsrc_windows_amd64.syso

## tools: Install build tool dependencies
tools:
	go install github.com/akavel/rsrc@latest

## build: Compile the binary for the current platform
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/jira-tui

## windows-icon: Generate Windows resource file with embedded icon
windows-icon: $(SYSO)
$(SYSO): $(ICON)
	$(RSRC) -ico $(ICON) -o $(SYSO)

## build-all: Cross-compile for Linux amd64, macOS ARM, and Windows amd64
build-all: windows-icon
	@mkdir -p $(BUILD_DIR)
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64       ./cmd/jira-tui
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64      ./cmd/jira-tui
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/jira-tui

build-for-me: windows-icon
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/jira-tui
	@mkdir -p $(MY_DIR)
	cp $(BUILD_DIR)/$(BINARY)-windows-amd64.exe $(MY_DIR)/jira-tui.exe
	cp -r $(BUILD_DIR)/.jira-tui $(MY_DIR)/.jira-tui


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
