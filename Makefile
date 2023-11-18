.PHONY: all deps test build build/app build/docker clean install

SERVICE = redactedhook
GO = go
RM = rm -f
GIT_COMMIT = $(shell git rev-parse HEAD 2> /dev/null)
GIT_TAG = $(shell git describe --abbrev=0 --tags)
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GOFLAGS = "-X main.commit=$(GIT_COMMIT) -X main.version=$(GIT_TAG) -X main.buildDate=$(BUILD_DATE)"
PREFIX = /usr/local
BINDIR = bin

all: clean build

deps:
	@test -z "$(shell git status --porcelain go.mod go.sum)" || $(GO) mod download

test:
	$(GO) test -parallel 4 $(shell $(GO) list ./... | grep -v test/integration)

build: deps build/app

build/app:
	@test -z "$(shell find . -name '*.go' -newer bin/$(SERVICE))" || $(GO) build -ldflags $(GOFLAGS) -o bin/$(SERVICE) ./main.go

build/docker:
	docker build -t redactedhook:dev -f Dockerfile . --build-arg GIT_TAG=$(GIT_TAG) --build-arg GIT_COMMIT=$(GIT_COMMIT)

clean:
	@test ! -d bin || $(RM) -r bin

install: all
	@echo $(DESTDIR)$(PREFIX)/$(BINDIR)
	@mkdir -p $(DESTDIR)$(PREFIX)/$(BINDIR)
	@cp -f bin/$(SERVICE) $(DESTDIR)$(PREFIX)/$(BINDIR)
