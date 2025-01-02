# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=mi-top

# Version information
VERSION=0.1.0
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-s -w \
	-X main.Version=${VERSION} \
	-X main.BuildTime=${BUILD_TIME} \
	-X main.GitCommit=${GIT_COMMIT}

.PHONY: all build clean test deps help install uninstall version

all: deps build ## Build the project

build: ## Build the binary
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)

clean: ## Clean build files
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test: ## Run tests
	$(GOTEST) -v ./...

deps: ## Get dependencies
	$(GOMOD) download

install: build ## Install the binary to /usr/local/bin
	sudo cp $(BINARY_NAME) /usr/local/bin/

uninstall: ## Remove the binary from /usr/local/bin
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

version: ## Show version information
	@echo "Version:    ${VERSION}"
	@echo "Git commit: ${GIT_COMMIT}"
	@echo "Build time: ${BUILD_TIME}"

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help