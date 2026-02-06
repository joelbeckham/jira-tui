.PHONY: build test lint clean run

BINARY := jira-tui
BUILD_DIR := bin

## build: Compile the binary
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/jira-tui

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
