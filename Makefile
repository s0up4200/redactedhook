# Makefile

# Variables
BINARY_NAME = RedactedHook
BINDIR = ./bin
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
VERSION = $(shell git describe --tags)
LDFLAGS = -ldflags "-X main.Version=$(VERSION)"
DOCKER_IMAGE_NAME = redactedhook
DOCKER_TAG = latest

# Targets
.PHONY: all build clean test docker-build

all: build

build:
	mkdir -p $(BINDIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINDIR)/$(BINARY_NAME)

clean:
	$(GOCLEAN)
	rm -rf $(BINDIR)

test:
	$(GOTEST) -v ./...

docker-build:
	docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .
