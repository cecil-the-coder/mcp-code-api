# MCP Code API - Go Implementation Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=mcp-code-api
BINARY_UNIX=$(BINARY_NAME)_unix

# Build targets
.PHONY: all build clean test coverage deps help install run lint format

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v .

test:
	$(GOTEST) -v ./...

coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

clean: 
	$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_UNIX)
	@rm -f coverage.out

deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Development targets
install:
	$(GOBUILD) -o $(BINARY_NAME) -v .
	@echo "Installing to /usr/local/bin..."
	@sudo cp $(BINARY_NAME) /usr/local/bin/ || cp $(BINARY_NAME) ~/bin/ 2>/dev/null || echo "Please copy $(BINARY_NAME) to your PATH manually"

run:
	$(GOBUILD) -o $(BINARY_NAME) -v .
	./$(BINARY_NAME)

# Cross-compilation
linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v .

# Code quality
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2)
	$(shell go env GOPATH)/bin/golangci-lint run

format:
	$(GOCMD) fmt ./...
	@echo "Code formatted successfully"

# Release targets
release: clean test build
	@echo "Creating release..."
	@mkdir -p release
	@cp $(BINARY_NAME) release/
	@tar -czf release/$(BINARY_NAME)-$(shell git describe --tags --always --dirty).tar.gz $(BINARY_NAME)
	@echo "Release created in release/ directory"

# Development utilities
watch:
	@which fswatch > /dev/null || (echo "Installing fswatch..." && brew install fswatch || sudo apt-get install fswatch)
	fswatch -o . | xargs -n1 -I{} make build

# Docker targets
docker-build:
	docker build -t $(BINARY_NAME) .

docker-run:
	docker run --rm -it $(BINARY_NAME)

# Help target
help:
	@echo "Available targets:"
	@echo "  all       - Run tests and build"
	@echo "  build     - Build the binary"
	@echo "  test      - Run tests"
	@echo "  coverage  - Run tests with coverage report"
	@echo "  clean     - Clean build artifacts"
	@echo "  deps      - Download dependencies"
	@echo "  install   - Build and install to system PATH"
	@echo "  run       - Build and run the binary"
	@echo "  linux     - Cross-compile for Linux"
	@echo "  lint      - Run linter"
	@echo "  format    - Format code"
	@echo "  release   - Create a release package"
	@echo "  watch     - Watch for changes and rebuild"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run  - Run Docker container"
	@echo "  help      - Show this help message"

# Default target
default: build