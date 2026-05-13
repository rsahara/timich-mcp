.DEFAULT_GOAL := help

-include ../../versions.mk

BUILD_DIR ?= build
BINARY ?= $(BUILD_DIR)/timich-mcp
GO_BUILD_FLAGS ?=

.PHONY: help build test run

help:
	@echo "timich-mcp"
	@echo ""
	@echo "Available targets:"
	@echo "  make build   Build the timich-mcp binary"
	@echo "  make test    Run timich-mcp Go tests"
	@echo "  make run     Run timich-mcp as a stdio MCP server"

build:
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -ldflags "$(TIMICH_MCP_LDFLAGS)" -o $(BINARY) ./cmd/timich-mcp

test:
	go test ./...

run:
	go run ./cmd/timich-mcp serve
