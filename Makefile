# Simple Makefile to build and run the zone-names Temporal worker

# Load .env if present to provide environment variables for local development
ifneq (,$(wildcard .env))
include .env
export
endif

# Configurable variables
IMAGE          ?= zone-names-worker
TAG            ?= latest
DOCKER_PLATFORM?= linux/amd64
BIN_DIR        ?= bin
BIN            ?= $(BIN_DIR)/worker
PKG            ?= ./cmd/worker

# Temporal defaults (override at invocation)
# Temporal / Worker defaults (can be provided via .env)
TEMPORAL_TARGET_HOST ?= 127.0.0.1:7233
TEMPORAL_ADDRESS     ?= $(TEMPORAL_TARGET_HOST)
TEMPORAL_NAMESPACE?= default
TEMPORAL_TASK_QUEUE?= zone-names
LOG_LEVEL         ?= info
METRICS_ADDR      ?= :9090
ZN_TMP_DIR        ?= /tmp/zone-names

# AWS defaults (override as needed)
AWS_REGION  ?= us-east-1
AWS_PROFILE ?=

# Helper to pass-through AWS creds/profile if present
AWS_ENV = \
	-e AWS_REGION=$(AWS_REGION) \
	$(if $(AWS_PROFILE),-e AWS_PROFILE=$(AWS_PROFILE),) \
	$(if $(AWS_ACCESS_KEY_ID),-e AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID),) \
	$(if $(AWS_SECRET_ACCESS_KEY),-e AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY),) \
	$(if $(AWS_SESSION_TOKEN),-e AWS_SESSION_TOKEN=$(AWS_SESSION_TOKEN),)

# Use .env for docker run if present
DOCKER_ENV_FILE := $(if $(wildcard .env),--env-file .env,)

.PHONY: all build test docker-build docker-push docker-run clean tidy

all: build

$(BIN): ## Build the worker binary locally
	@mkdir -p $(BIN_DIR)
	GO111MODULE=on CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BIN) $(PKG)

build: $(BIN)

test: ## Run all unit tests
	go test ./...

 tidy: ## Update go.sum and tidy modules
	go mod tidy

docker-build: ## Build the worker container image
	docker buildx build --platform=$(DOCKER_PLATFORM) -t $(IMAGE):$(TAG) .

docker-push: ## Push the image to the configured registry (ensure IMAGE is a registry ref)
	docker buildx build --platform=$(DOCKER_PLATFORM) -t $(IMAGE):$(TAG) --push .

docker-run: ## Run the worker container with environment configured
	docker run --rm \
		--name zone-names-worker \
		$(DOCKER_ENV_FILE) \
		-e TEMPORAL_TARGET_HOST=$(TEMPORAL_TARGET_HOST) \
		-e TEMPORAL_ADDRESS=$(TEMPORAL_ADDRESS) \
		-e TEMPORAL_NAMESPACE=$(TEMPORAL_NAMESPACE) \
		-e TEMPORAL_TASK_QUEUE=$(TEMPORAL_TASK_QUEUE) \
		-e LOG_LEVEL=$(LOG_LEVEL) \
		-e METRICS_ADDR=$(METRICS_ADDR) \
		-e ZN_TMP_DIR=$(ZN_TMP_DIR) \
		$(AWS_ENV) \
		$(IMAGE):$(TAG)

clean: ## Remove built artifacts
	rm -rf $(BIN_DIR)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ---- Dev stack (docker compose) ----
.PHONY: dev-up dev-down dev-logs dev-ps dev-worker-logs dev-restart
dev-up: ## Start Temporal, UI, MinIO, and worker via docker compose (build if needed)
	docker compose up -d --build
dev-down: ## Stop and remove the dev stack
	docker compose down -v
dev-logs: ## Tail logs of the dev stack
	docker compose logs -f --tail=200
dev-ps: ## List dev stack services
	docker compose ps
dev-worker-logs: ## Tail only the worker service logs
	docker compose logs -f --tail=200 worker
dev-restart: ## Rebuild and restart just the worker service
	docker compose up -d --build worker
