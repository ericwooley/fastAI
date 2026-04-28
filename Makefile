SHELL := /usr/bin/env bash

BINARY := fastAI
CMD := ./cmd/fastAI
BIN_DIR := tmp/bin
BIN := $(BIN_DIR)/$(BINARY)
GO_BUILD_FLAGS ?= -trimpath -ldflags "-s -w"

.DEFAULT_GOAL := help

.PHONY: help build install run login test test-unit test-integration test-e2e fmt vet check clean

help: ## Show available make targets.
	@awk 'BEGIN {FS = ":.*## "; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the local CLI binary into tmp/bin/fastAI.
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BIN) $(CMD)

build-dev: ## Build the local CLI binary with debug symbols for development.
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN) $(CMD)

install: ## Install the CLI into the OS-appropriate user bin directory.
	./scripts/install.sh

run: build ## Run the local CLI. Pass args with ARGS='--model ... prompt'.
	$(BIN) $(ARGS)

login: build ## Authenticate with GitHub Copilot using the local binary.
	$(BIN) login

test: ## Run all tests.
	go test ./...

test-unit: ## Run internal package tests.
	go test ./internal/...

test-integration: ## Run integration tests.
	go test ./test/integration/...

test-e2e: ## Run end-to-end CLI tests.
	go test ./test/e2e/...

fmt: ## Format Go code.
	go fmt ./...

vet: ## Run Go static analysis.
	go vet ./...

check: fmt vet test ## Format, vet, and test the full project.

clean: ## Remove local build artifacts.
	rm -rf $(BIN_DIR)
