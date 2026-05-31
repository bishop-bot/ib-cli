.PHONY: build test lint clean install run help

# Binary name
BINARY_NAME=ibcli
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOFMT=$(GOCMD) fmt
GOMOD=$(GOCMD) mod
VERSION ?= dev
COMMIT ?= unknown
LDFLAGS := -ldflags "-X github.com/bishop-bot/ibcli-go/cmd.Version=$(VERSION) -X github.com/bishop-bot/ibcli-go/cmd.Commit=$(COMMIT)"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

build-all: ## Build for multiple platforms
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .

test: ## Run tests
	$(GOTEST) -v -race -cover ./...

test-unit: ## Run unit tests only
	$(GOTEST) -v -short ./...

lint: ## Run linters (requires golangci-lint)
	golangci-lint run ./...

fmt: ## Format code
	$(GOFMT) ./...

tidy: ## Tidy dependencies
	$(GOMOD) tidy

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*

install: ## Install binary to GOBIN
	$(GOBUILD) $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME) .

run: ## Run the CLI
	$(GOCMD) run main.go -- $(ARGS)

# Development helpers
dev: ## Build with dev version
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

prod: VERSION=prod ## Build for production
prod: COMMIT=$(shell git rev-parse --short HEAD)
prod: build

# Docker helpers (for running with gateway)
docker-build:
	docker build -t $(BINARY_NAME) .

# Self-documented: see all targets
.DEFAULT_GOAL := help