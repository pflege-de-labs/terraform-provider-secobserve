# Development entrypoints. See CONTRIBUTING.md for the full workflow.

BINARY      := terraform-provider-secobserve
# GOBIN wins when set (mise, asdf and friends set it); GOPATH/bin otherwise.
GOBIN       := $(shell go env GOBIN)
ifeq ($(strip $(GOBIN)),)
GOBIN       := $(shell go env GOPATH)/bin
endif
COMPOSE     := docker compose --env-file test/.env -f test/docker-compose.yml
COMPOSE_OIDC:= $(COMPOSE) -f test/docker-compose.oidc.yml
BASE_URL    ?= http://localhost:8000

# Apple `container` targets. HOST_ADDRESS overrides how the container reaches
# the host; empty means auto-detect. See test/apple-container.md.
CONTAINER_NAME ?= secobserve-backend
HOST_ADDRESS   ?=

# The parent directory may contain an unrelated go.work; the provider is a
# standalone module and must not be pulled into someone else's workspace.
export GOWORK := off

# Pinned tool versions. oapi-codegen is not here: it is a go.mod tool
# dependency, so `go generate` pins it and Renovate tracks it with everything
# else in go.mod.
# renovate: datasource=github-releases depName=hashicorp/terraform-plugin-docs
TFPLUGINDOCS_VERSION := v0.23.0
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION:= v2.13.2

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the provider binary
	go build -o $(BINARY) .

.PHONY: install
install: ## Build and install into GOPATH/bin for dev_overrides
	go install .

.PHONY: test
test: ## Run unit tests
	go test ./... -timeout 5m

.PHONY: testacc
testacc: ## Run acceptance tests against the containerized instance
	TF_ACC=1 go test ./... -v -timeout 30m

.PHONY: testacc-oidc
testacc-oidc: ## Run the OIDC-specific acceptance tests (needs the Keycloak overlay)
	TF_ACC=1 go test ./... -v -timeout 30m -tags oidc -run OIDC

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## Format Go and Terraform sources
	gofmt -w .
	@command -v terraform >/dev/null && terraform fmt -recursive ./examples || true

.PHONY: up
up: ## Start the containerized SecObserve and mint an API token
	$(COMPOSE) up -d
	@./test/bootstrap.sh

.PHONY: up-oidc
up-oidc: ## Start the containerized SecObserve with Keycloak
	$(COMPOSE_OIDC) up -d
	@./test/bootstrap.sh

.PHONY: down
down: ## Stop the stack and delete its volumes
	$(COMPOSE_OIDC) down --volumes --remove-orphans

.PHONY: logs
logs: ## Tail the backend logs
	$(COMPOSE) logs -f backend

# Apple `container` instead of Docker Compose, with PostgreSQL on the host.
# Kept as separate targets so it is always obvious which stack is being driven.
# See test/apple-container.md for the one-time host setup.
.PHONY: container-up
container-up: ## Start SecObserve under Apple container (host PostgreSQL)
	@./test/container-up.sh $(HOST_ADDRESS)

.PHONY: container-down
container-down: ## Stop and remove the Apple container (leaves the host database alone)
	@container stop $(CONTAINER_NAME) 2>/dev/null || true
	@container rm $(CONTAINER_NAME) 2>/dev/null || true
	@echo "removed $(CONTAINER_NAME); the host database was not touched"

.PHONY: container-logs
container-logs: ## Follow the Apple container's logs
	container logs --follow $(CONTAINER_NAME)

.PHONY: schema
schema: ## Refetch the OpenAPI schema from the running instance
	@test -n "$(SECOBSERVE_API_TOKEN)" || { echo "SECOBSERVE_API_TOKEN is not set; run: eval \"\$$(./test/bootstrap.sh)\"" >&2; exit 1; }
	curl -fsS -H "Authorization: APIToken $(SECOBSERVE_API_TOKEN)" \
		"$(BASE_URL)/api/oa3/schema/?format=json" \
		| python3 -m json.tool --sort-keys > api/openapi.json
	@echo "wrote api/openapi.json"

.PHONY: generate
generate: ## Regenerate the API client and the provider documentation
	go generate ./...

.PHONY: docs
docs: ## Regenerate docs/ with tfplugindocs
	$(GOBIN)/tfplugindocs generate --provider-name secobserve

.PHONY: tools
tools: ## Install the pinned development tools
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@$(TFPLUGINDOCS_VERSION)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: clean
clean: ## Remove build artifacts
	rm -f $(BINARY)
