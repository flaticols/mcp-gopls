.PHONY: build run install test lint format clean help

# Binary name
BINARY=mcpgopls
BUILD_DIR=bin

# Go build flags
GOFLAGS=-v

# Go packages
PACKAGES=./...

help:
	@echo "Available commands:"
	@echo "  make build     - Build the binary"
	@echo "  make run       - Run the application"
	@echo "  make install   - Install the binary"
	@echo "  make test      - Run tests"
	@echo "  make lint      - Run linter"
	@echo "  make format    - Format code"
	@echo "  make clean     - Remove build artifacts"

build:
	mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/mcpgopls

run:
	go run ./cmd/mcpgopls

install:
	go install $(GOFLAGS) ./cmd/mcpgopls

test:
	go test $(PACKAGES)

test-verbose:
	go test -v $(PACKAGES)

lint:
	golangci-lint run

format:
	gofmt -w .
	goimports -w .

clean:
	rm -rf $(BUILD_DIR)