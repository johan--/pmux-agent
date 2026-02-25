# Makefile for pmux-agent

.PHONY: build test test-integration test-stress clean

# Build the pmux binary
build:
	go build -o bin/pmux ./cmd/pmux

# Run unit tests
test:
	go test ./...

# Run integration tests (requires tmux)
test-integration:
	go test -tags=integration -race -timeout=120s ./test/integration/... -v

# Run stress tests (requires tmux, may take several minutes)
test-stress:
	go test -tags=stress -race -timeout=300s ./test/stress/... -v

# Run all tests (unit + integration + stress)
test-all: test test-integration test-stress

# Clean build artifacts
clean:
	rm -rf bin/
