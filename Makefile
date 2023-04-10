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

# Targets
.PHONY: all build clean test

all: build

build:
	mkdir -p $(BINDIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINDIR)/$(BINARY_NAME)

clean:
	$(GOCLEAN)
	rm -rf $(BINDIR)

test:
	$(GOTEST) -v ./...
