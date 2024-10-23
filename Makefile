.PHONY: all deps test build clean install lint coverage version help
.POSIX:
.SUFFIXES:

# Project variables
SERVICE := redactedhook
PREFIX := /usr/local
BINDIR := bin

# Go related variables
GO := go
GOFLAGS := -trimpath -mod=readonly

# Git information
GIT_COMMIT := $(shell git rev-parse --short HEAD 2> /dev/null)
GIT_TAG := $(shell git describe --abbrev=0 --tags 2> /dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker flags
LDFLAGS := -X main.commit=$(GIT_COMMIT) \
           -X main.version=$(GIT_TAG) \
           -X main.buildDate=$(BUILD_DATE)

# Default target
all: clean build ## Build the project

# Show help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod verify

lint: ## Run linters
	$(GO) vet ./...
	@command -v golangci-lint >/dev/null 2>&1 || { echo >&2 "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run

test: ## Run tests
	$(GO) test -v -race $(shell go list ./... | grep -v test/integration)

coverage: ## Run tests with coverage
	$(GO) test -v -race -coverprofile=coverage.out $(shell go list ./... | grep -v test/integration)
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

build: deps ## Build the binary
	$(GO) build -ldflags "$(LDFLAGS)" $(GOFLAGS) -o $(BINDIR)/$(SERVICE) ./cmd/redactedhook/main.go

build/docker: ## Build Docker image
	docker build -t $(SERVICE):$(GIT_TAG) -f Dockerfile . \
		--build-arg GIT_TAG=$(GIT_TAG) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT)

clean: ## Clean build artifacts
	rm -rf $(BINDIR)
	rm -f coverage.out coverage.html

install: all ## Install the binary
	@echo "Installing to $(DESTDIR)$(PREFIX)/$(BINDIR)"
	@mkdir -p $(DESTDIR)$(PREFIX)/$(BINDIR)
	@cp -f $(BINDIR)/$(SERVICE) $(DESTDIR)$(PREFIX)/$(BINDIR)

version: ## Display version information
	@echo "Version:    $(GIT_TAG)"
	@echo "Commit:     $(GIT_COMMIT)"
	@echo "Built:      $(BUILD_DATE)"
