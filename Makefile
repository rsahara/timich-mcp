.DEFAULT_GOAL := help

-include ../../versions.mk

TIMICH_MCP_VERSION ?= 0.1.0
TIMICH_MCP_DIST_REPO ?= rsahara/timich-mcp
TIMICH_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
TIMICH_BUILT_AT ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
TIMICH_MCP_LDFLAGS ?= -X main.version=$(TIMICH_MCP_VERSION) -X main.commit=$(TIMICH_COMMIT) -X main.builtAt=$(TIMICH_BUILT_AT)
GOCACHE ?= $(abspath build/go-build-cache)
export GOCACHE

BUILD_DIR ?= build
BINARY ?= $(BUILD_DIR)/timich-mcp
GO_BUILD_FLAGS ?=
DIST_DIR ?= dist
DIST_OS ?= $(GOOS)
DIST_ARCH ?= $(GOARCH)
DIST_NAME ?= timich-mcp_$(TIMICH_MCP_VERSION)_$(DIST_OS)_$(DIST_ARCH)
DIST_STAGE := build/dist/$(DIST_NAME)
DIST_STAGE_ABS := $(abspath $(DIST_STAGE))
DIST_ARCHIVE := $(DIST_DIR)/$(DIST_NAME).tar.gz
DIST_BINARY_EXT := $(if $(filter windows,$(DIST_OS)),.exe,)
MCP_DIST_REPO ?= $(TIMICH_MCP_DIST_REPO)
MCP_DIST_TAG ?= v$(TIMICH_MCP_VERSION)
MCP_DIST_TITLE ?= Timich MCP $(TIMICH_MCP_VERSION)
MCP_DIST_NOTES ?= Timich MCP $(TIMICH_MCP_VERSION) release artifacts.

.PHONY: help build test run dist publish-dist

help:
	@echo "timich-mcp"
	@echo ""
	@echo "Available targets:"
	@echo "  make build   Build the timich-mcp binary"
	@echo "  make test    Run timich-mcp Go tests"
	@echo "  make run     Run timich-mcp as a stdio MCP server"
	@echo "  make dist    Build a timich-mcp release tarball"
	@echo "  make publish-dist"
	@echo "                Build and upload the tarball to GitHub Releases"

build:
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -ldflags "$(TIMICH_MCP_LDFLAGS)" -o $(BINARY) ./cmd/timich-mcp

test:
	go test ./...

run:
	go run ./cmd/timich-mcp serve

dist:
	@rm -rf "$(DIST_STAGE)"
	@mkdir -p "$(DIST_STAGE)" "$(DIST_DIR)"
	GOOS=$(DIST_OS) GOARCH=$(DIST_ARCH) CGO_ENABLED=0 \
		$(MAKE) build \
		BUILD_DIR="$(DIST_STAGE_ABS)" \
		BINARY="$(DIST_STAGE_ABS)/timich-mcp$(DIST_BINARY_EXT)" \
		GO_BUILD_FLAGS="-trimpath -buildvcs=false" \
		TIMICH_MCP_VERSION="$(TIMICH_MCP_VERSION)" \
		TIMICH_COMMIT="$(TIMICH_COMMIT)" \
		TIMICH_BUILT_AT="$(TIMICH_BUILT_AT)"
	@printf '%s\n' \
		"TIMICH_MCP_VERSION=$(TIMICH_MCP_VERSION)" \
		"TIMICH_COMMIT=$(TIMICH_COMMIT)" \
		"TIMICH_BUILT_AT=$(TIMICH_BUILT_AT)" \
		"GOOS=$(DIST_OS)" \
		"GOARCH=$(DIST_ARCH)" > "$(DIST_STAGE)/VERSION"
	@printf '%s\n' \
		'{' \
		'  "mcpVersion": "$(TIMICH_MCP_VERSION)",' \
		'  "commit": "$(TIMICH_COMMIT)",' \
		'  "builtAt": "$(TIMICH_BUILT_AT)",' \
		'  "goos": "$(DIST_OS)",' \
		'  "goarch": "$(DIST_ARCH)"' \
		'}' > "$(DIST_STAGE)/BUILDINFO.json"
	@cp README.md "$(DIST_STAGE)/README.md"
	@tar -C "build/dist" -czf "$(DIST_ARCHIVE)" "$(DIST_NAME)"
	@archive_sha="$$(shasum -a 256 "$(DIST_ARCHIVE)" | awk '{print $$1}')" && \
		printf '%s  %s\n' "$$archive_sha" "$(DIST_NAME).tar.gz" > "$(DIST_ARCHIVE).sha256"
	@echo "Wrote $(DIST_ARCHIVE)"

publish-dist: dist
	@command -v gh >/dev/null 2>&1 || { echo "gh is required to publish release assets."; exit 1; }
	@if gh release view "$(MCP_DIST_TAG)" --repo "$(MCP_DIST_REPO)" >/dev/null 2>&1; then \
		echo "Using existing release $(MCP_DIST_REPO) $(MCP_DIST_TAG)"; \
	else \
		gh release create "$(MCP_DIST_TAG)" \
			--repo "$(MCP_DIST_REPO)" \
			--title "$(MCP_DIST_TITLE)" \
			--notes "$(MCP_DIST_NOTES)"; \
	fi
	@gh release upload "$(MCP_DIST_TAG)" "$(DIST_ARCHIVE)" "$(DIST_ARCHIVE).sha256" --repo "$(MCP_DIST_REPO)" --clobber
	@echo "Published MCP artifacts to $(MCP_DIST_REPO) $(MCP_DIST_TAG)"
