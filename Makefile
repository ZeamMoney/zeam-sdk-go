SHELL := /bin/bash

GATEWAY_TAG ?= main
GATEWAY_RAW_BASE := https://raw.githubusercontent.com/BeamMoney/zeam-api-gateway.go/$(GATEWAY_TAG)

GO      ?= go
GOFUMPT ?= gofumpt
GOLANGCI ?= golangci-lint

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: tidy
tidy: ## go mod tidy
	$(GO) mod tidy

.PHONY: fmt
fmt: ## gofumpt the tree
	$(GOFUMPT) -w .

.PHONY: lint
lint: ## golangci-lint + gofumpt check
	$(GOFUMPT) -l . | tee /dev/stderr | test ! -s
	$(GOLANGCI) run ./...

.PHONY: test-unit
test-unit: ## Fast unit tests (no build tags)
	$(GO) test ./...

.PHONY: test-race
test-race: ## Unit tests with -race
	$(GO) test -race ./...

.PHONY: test-integration
test-integration: ## httptest round-trip tests (build tag integration)
	$(GO) test -race -tags=integration -timeout=300s ./...

.PHONY: test-contract
test-contract: ## Live-gateway smoke (requires ZEAM_API_URL + ZEAM_CONTRACT_TESTS=1)
	$(GO) test -race -tags=contract -timeout=600s ./test/contract/...

.PHONY: vet
vet: ## go vet
	$(GO) vet ./...

.PHONY: govulncheck
govulncheck: ## Vulnerability scan
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: sync-spec
sync-spec: ## Pull OpenAPI spec from gateway at GATEWAY_TAG
	@echo "Syncing OpenAPI spec from $(GATEWAY_RAW_BASE)"
	curl -sSfL $(GATEWAY_RAW_BASE)/docs/openapi.yaml -o api/openapi.yaml
	shasum -a 256 api/openapi.yaml | awk '{print $$1}' > api/openapi.sha256

.PHONY: generate
generate: ## Regenerate internal/wire from api/openapi.yaml
	$(GO) generate ./...

.PHONY: build-examples
build-examples: ## Compile every example
	@for dir in ./examples/*/; do \
		echo "Building $$dir"; \
		$(GO) build -o /dev/null $$dir; \
	done

.PHONY: check
check: fmt vet lint test-race ## Full pre-commit gate

.DEFAULT_GOAL := help
