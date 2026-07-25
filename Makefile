SHELL := /usr/bin/env bash

BINARY := fastAI
CMD := ./cmd/fastAI
BIN_DIR := tmp/bin
BIN := $(BIN_DIR)/$(BINARY)
GO_BUILD_FLAGS ?= -trimpath -ldflags "-s -w"

.DEFAULT_GOAL := help

.PHONY: help build install run login test test-unit test-integration test-e2e test-release fmt fmt-check vet check pre-commit install-hooks clean

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
	$(BIN) login copilot

test: ## Run all tests.
	go test ./...

test-unit: ## Run internal package tests.
	go test ./internal/...

test-integration: ## Run integration tests.
	go test ./test/integration/...

test-e2e: ## Run end-to-end CLI tests.
	go test ./test/e2e/...

test-release: ## Run release, commit-policy, and Homebrew packaging tests.
	bash ./scripts/validate-commit-message.test.sh
	bash ./scripts/release-plan.test.sh
	bash ./scripts/release-workflow.test.sh
	bash ./scripts/render-homebrew-formula.test.sh
	bash ./scripts/package-release.test.sh
	@set -e; for file in scripts/*.sh .githooks/*; do bash -n "$$file"; done

fmt: ## Format Go code.
	go fmt ./...

fmt-check: ## Verify Go formatting without changing files.
	@unformatted="$$(gofmt -l $$(find cmd internal test -type f -name '*.go' -print))"; \
	if [ -n "$$unformatted" ]; then \
		printf 'Go files need formatting:\n%s\n' "$$unformatted" >&2; \
		exit 1; \
	fi

vet: ## Run Go static analysis.
	go vet ./...

check: fmt-check vet test test-release ## Verify formatting, vet, and run all tests.

pre-commit: fmt-check test-unit ## Run the fast checks enforced by the pre-commit hook.
	git diff --check --cached

install-hooks: ## Install this repository's pre-commit and commit-msg hooks.
	./scripts/install-git-hooks.sh

clean: ## Remove local build artifacts.
	rm -rf $(BIN_DIR)
