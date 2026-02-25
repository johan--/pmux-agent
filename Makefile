# Makefile for pmux-agent

.PHONY: build test test-integration clean

# Build the pmux binary
build:
	go build -o bin/pmux ./cmd/pmux

# Run unit tests
test:
	go test ./...

# Run integration tests (requires tmux)
test-integration:
	go test -tags=integration -race -timeout=120s ./test/integration/... -v

# Run all tests (unit + integration)
test-all: test test-integration

# Clean build artifacts
clean:
	rm -rf bin/
