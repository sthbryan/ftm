# FTM Makefile
#
# Run `make help` for the target list. Targets carry their own `##` description,
# so the help output cannot drift from what is actually here.

BINARY_NAME=ftm
BUILD_DIR=bin
PKG=./cmd/ftm
DESKTOP_DIR=./desktop

# git describe includes the leading "v" from tags; strip it so callers can print "v$(Version)".
VERSION=$(shell (git describe --tags --always --dirty 2>/dev/null || echo dev) | sed 's/^v//')
# Windows resources need a strictly numeric version, so drop any git describe suffix.
WINRES_VERSION=$(shell echo $(VERSION) | sed 's/[-+].*//')
LDFLAGS=-ldflags "-X github.com/sthbryan/ftm/internal/version.Version=$(VERSION)"
CGO_ENABLED=0

# Install location (override with `make install BINDIR=/path/to/bin`)
GOBIN := $(shell go env GOPATH)/bin
BINDIR ?= $(GOBIN)
DESTDIR ?=
INSTALL_PATH=$(DESTDIR)$(BINDIR)/$(BINARY_NAME)

# Wails v3 defaults to webkitgtk-6.0 (GTK4) on Linux, but the bundled AppImage
# helpers target WebKitGTK 4.1 (GTK3), so force the gtk3 tag on Linux hosts.
DESKTOP_OUT := $(DESKTOP_DIR)/build/bin
DESKTOP_BIN := ftm-desktop
ifeq ($(OS),Windows_NT)
DESKTOP_BIN := ftm-desktop.exe
endif
UNAME_S := $(shell uname -s)
DESKTOP_DEV_TAGS_ARG :=
DESKTOP_PROD_TAGS_ARG := production
ifeq ($(UNAME_S),Linux)
DESKTOP_DEV_TAGS_ARG := gtk3
DESKTOP_PROD_TAGS_ARG := production,gtk3
endif

.DEFAULT_GOAL := build

##@ Build

.PHONY: web
web: ## Build the Preact UI into internal/transport/httpapi/static (+ desktop dist)
	@echo "Building web UI..."
	./scripts/build-web-assets.sh
	mkdir -p $(DESKTOP_DIR)/build
	cp $(DESKTOP_DIR)/icon.png $(DESKTOP_DIR)/build/appicon.png

.PHONY: build
build: ## Build the CLI with whatever is already embedded in internal/transport/httpapi/static
	@echo "Building $(BINARY_NAME) for current platform... ($(VERSION))"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(PKG)

.PHONY: build-full
build-full: web build ## Build the web UI and then the CLI

.PHONY: install
install: web build ## Build and copy the binary to $BINDIR (default $(go env GOPATH)/bin)
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	install -d $(DESTDIR)$(BINDIR)
	install -m 0755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)
	@echo "Installed: $(INSTALL_PATH)"

.PHONY: uninstall
uninstall: ## Remove the installed binary from $BINDIR
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(INSTALL_PATH)

##@ Desktop

.PHONY: desktop
desktop: web ## Build the Wails v3 desktop shell for the host OS
	@echo "Building desktop (production tags)..."
	@mkdir -p $(DESKTOP_OUT)
	CGO_ENABLED=1 go build $(if $(DESKTOP_PROD_TAGS_ARG),-tags $(DESKTOP_PROD_TAGS_ARG),) -ldflags "-s -w" -o $(DESKTOP_OUT)/$(DESKTOP_BIN) ./desktop

.PHONY: desktop-dev
desktop-dev: web ## Build the desktop shell without production tags
	@echo "Building desktop (dev)..."
	@mkdir -p $(DESKTOP_OUT)
	CGO_ENABLED=1 go build $(if $(DESKTOP_DEV_TAGS_ARG),-tags $(DESKTOP_DEV_TAGS_ARG),) -o $(DESKTOP_OUT)/$(DESKTOP_BIN) ./desktop

.PHONY: desktop-winres
desktop-winres: ## Regenerate the Windows icon/manifest resources after changing desktop/icon.png
	@echo "Generating Windows resources (icon, manifest, version info)..."
	go run github.com/tc-hib/go-winres@v0.3.3 make \
		--in $(DESKTOP_DIR)/winres/winres.json \
		--out $(DESKTOP_DIR)/rsrc \
		--arch amd64,arm64 \
		--product-version $(VERSION) \
		--file-version $(WINRES_VERSION)

##@ Develop

.PHONY: run
run: ## go run the CLI against the assets already embedded (fast; Go changes only)
	go run $(PKG)

.PHONY: dev
dev: web run ## Rebuild the web UI, then go run the CLI

##@ Quality

.PHONY: test
test: ## Run the Go tests
	@echo "Running tests..."
	go test ./...

.PHONY: test-race
test-race: ## Run the Go tests with the race detector
	@echo "Running tests with -race..."
	go test -race ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

.PHONY: fmt
fmt: ## Run gofmt -w and fail if anything is still unformatted
	@echo "Running gofmt -w..."
	gofmt -w .
	@echo "Checking for unformatted files..."
	@test -z "$$(gofmt -l .)" || (echo "ERROR: gofmt found unformatted files:" && gofmt -l . && exit 1)
	@echo "All files formatted."

##@ Release

# Forward all args to scripts/bump-version.sh, e.g.
#   make version                 # interactive menu
#   make version ARGS="patch"    # bump patch
#   make version ARGS="0.11.0 --dry-run"
.PHONY: version
version: ## Bump the version, commit, tag and push (ARGS forwarded to scripts/bump-version.sh)
	@./scripts/bump-version.sh $(ARGS)

.PHONY: release
release: version ## Alias of version

##@ Housekeeping

.PHONY: clean
clean: ## Remove build artifacts (leaves internal/transport/httpapi/static, which build embeds)
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME) ftm-*
	rm -rf $(DESKTOP_DIR)/build/bin/*
	rm -rf $(DESKTOP_DIR)/frontend/dist
	@go clean -testcache 2>/dev/null || true

.PHONY: help
help: ## Show this help
	@awk 'BEGIN { FS = ":.*##"; printf "\nftm — make targets\n" } \
		/^##@/ { printf "\n%s\n", substr($$0, 5); next } \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  %-16s %s\n", $$1, $$2 } \
		END { printf "\n" }' $(MAKEFILE_LIST)
