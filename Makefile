BINARY_NAME := mts
SRC_DIR := src
BUILD_DIR := ./cmd/mts
GO := go
LDFLAGS := -s -w
VERSION := $(shell [ -n "$$MTS_VERSION" ] && echo "$$MTS_VERSION" || (git describe --tags --exact-match 2>/dev/null || git describe --always --dirty 2>/dev/null || echo "dev"))
GIT_COMMIT ?= $(shell if [ -n "$$MTS_COMMIT" ]; then echo "$$MTS_COMMIT"; else git rev-parse --short HEAD 2>/dev/null || echo "unknown"; fi)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_FLAGS := -X main.Version=$(VERSION) -X main.buildCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)
PREFIX ?= /usr/local
DESTDIR ?=

all: build

build:
	cd $(SRC_DIR) && CGO_ENABLED=0 $(GO) build -trimpath -mod=mod -ldflags "$(BUILD_FLAGS)" -o ../$(BINARY_NAME) $(BUILD_DIR)

build-release:
	cd $(SRC_DIR) && CGO_ENABLED=0 $(GO) build -trimpath -mod=mod -ldflags "$(LDFLAGS) $(BUILD_FLAGS)" -o ../$(BINARY_NAME) $(BUILD_DIR)

build-all:
	cd $(SRC_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath -mod=mod -ldflags "$(LDFLAGS) $(BUILD_FLAGS)" -o ../$(BINARY_NAME)_x86_64 $(BUILD_DIR)
	cd $(SRC_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -trimpath -mod=mod -ldflags "$(LDFLAGS) $(BUILD_FLAGS)" -o ../$(BINARY_NAME)_aarch64 $(BUILD_DIR)

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)_x86_64 $(BINARY_NAME)_aarch64

test:
	cd $(SRC_DIR) && CGO_ENABLED=0 $(GO) test -count=1 -short ./...

vet:
	cd $(SRC_DIR) && CGO_ENABLED=0 $(GO) vet ./...

install:
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	cp $(BINARY_NAME) $(DESTDIR)$(PREFIX)/bin/$(BINARY_NAME)

install-man:
	mkdir -p $(DESTDIR)$(PREFIX)/share/man/man1
	cp man/mts.1 $(DESTDIR)$(PREFIX)/share/man/man1/mts.1

release: clean build-all
	@echo "Binaries built with version: $(VERSION) and ready for release: $(BINARY_NAME)_x86_64, $(BINARY_NAME)_aarch64"

check-release:
	@if git describe --tags --exact-match >/dev/null 2>&1; then \
		if git diff-index --quiet HEAD --; then \
			GIT_TAG=$$(git describe --tags --exact-match); \
			if grep -q "version = \"$$GIT_TAG\"" flake.nix; then \
				echo "Repository is clean and tagged. Tag $$GIT_TAG matches version in flake.nix. Ready for release."; \
			else \
				FLAKE_VERSION=$$(grep -o 'version = "[^"]*"' flake.nix | cut -d'"' -f2); \
				echo "Error: Git tag ($$GIT_TAG) does not match version in flake.nix ($$FLAKE_VERSION)"; \
				exit 1; \
			fi; \
		else \
			echo "Repository is tagged but has uncommitted changes. Please commit or stash changes before release."; \
			exit 1; \
		fi; \
	else \
		echo "Repository is not tagged. Please create a tag before release."; \
		exit 1; \
	fi

help:
	@echo "make build              Build the binary"
	@echo "make build-release      Build the binary with version information"
	@echo "make build-all          Build linux amd64+arm64 binaries"
	@echo "make clean              Clean up build artifacts"
	@echo "make test               Run unit tests"
	@echo "make vet                Run go vet"
	@echo "make install            Install the binary locally"
	@echo "make install-man        Install the man page (skip if man-db absent)"
	@echo "make release            Build and prepare for release"
	@echo "make check-release      Verify if the repository is tagged and clean"
	@echo "make help               Show this message"
