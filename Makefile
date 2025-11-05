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

# Database defaults for local runs
DB_HOST    ?= localhost
DB_PORT    ?= 5432
DB_USER    ?= temporal
DB_PASSWORD?= temporal
DB_NAME    ?= premium_names
DB_SSLMODE ?= disable

# S3/MinIO defaults for local runs
AWS_ENDPOINT_URL_S3   ?= http://localhost:9000
AWS_S3_FORCE_PATH_STYLE?= true
IMPORT_BUCKET         ?= zone-names

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

.PHONY: all build test docker-build docker-push docker-run clean tidy lint

all: build

$(BIN): ## Build the worker binary locally
	@mkdir -p $(BIN_DIR)
	GO111MODULE=on CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BIN) $(PKG)

build: $(BIN)

test: ## Run all unit tests
	go test ./...


tidy: ## Update go.sum and tidy modules
	go mod tidy

lint: ## Run vet and format checks
	go vet ./...
	@FMT_OUT=$$(gofmt -s -l .); if [ -n "$$FMT_OUT" ]; then echo "Files need gofmt -s:"; echo "$$FMT_OUT"; exit 1; fi

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
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
	| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}' \
	| sort

# ---- Dev stack (docker compose) ----
.PHONY: dev-up dev-down dev-logs dev-ps dev-worker-logs dev-api-logs dev-importer-logs dev-restart dev-restart-api dev-restart-importer dev-frontend
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

dev-api-logs: ## Tail only the api service logs
	docker compose logs -f --tail=200 api
dev-restart-api: ## Rebuild and restart just the api service
	docker compose up -d --build api

dev-importer-logs: ## Tail only the importer service logs
	docker compose logs -f --tail=200 importer
dev-restart-importer: ## Rebuild and restart just the importer service
	docker compose up -d --build importer

dev-frontend: ## Run the Next.js frontend in ./frontend (installs deps on first run)
	cd frontend && [ -d node_modules ] || npm install
	cd frontend && npm run dev

# ---- DB migrations (using compose migrator) ----
.PHONY: migrate-up migrate-down
migrate-up: ## Apply SQL migrations up to latest using compose migrator
	docker compose run --rm migrator -path /migrations -database "postgres://$${DB_USER:-temporal}:$${DB_PASSWORD:-temporal}@postgres:5432/$${DB_NAME:-premium_names}?sslmode=disable" up
migrate-down: ## Roll back one migration (or add -all via MIGRATE_OPTS)
	docker compose run --rm migrator -path /migrations -database "postgres://$${DB_USER:-temporal}:$${DB_PASSWORD:-temporal}@postgres:5432/$${DB_NAME:-premium_names}?sslmode=disable" down $${MIGRATE_OPTS}

# ---- Run API and Importer locally ----
.PHONY: run-api run-importer
run-api: ## Run API locally (go run)
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) DB_SSLMODE=$(DB_SSLMODE) \
	AWS_ENDPOINT_URL_S3=$(AWS_ENDPOINT_URL_S3) AWS_S3_FORCE_PATH_STYLE=$(AWS_S3_FORCE_PATH_STYLE) IMPORT_BUCKET=$(IMPORT_BUCKET) \
	PORT=8081 go run ./cmd/api
run-importer: ## Run Importer locally (go run)
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) DB_SSLMODE=$(DB_SSLMODE) \
	AWS_ENDPOINT_URL_S3=$(AWS_ENDPOINT_URL_S3) AWS_S3_FORCE_PATH_STYLE=$(AWS_S3_FORCE_PATH_STYLE) IMPORT_BUCKET=$(IMPORT_BUCKET) \
	go run ./cmd/importer

# ---- Workflow helpers ----
.PHONY: start-workflow
# Start the Zone2NamesWorkflow using tctl on your host. Override START_INPUT to point at a JSON file.
START_INPUT ?= examples/request.example.json
start-workflow: ## Start the Zone2NamesWorkflow with tctl using --input_file
	# NOTE: The workflow type is the Go function name registered in the worker
	tctl workflow start --taskqueue $(TEMPORAL_TASK_QUEUE) --workflow_type Zone2NamesWorkflow --input_file $(START_INPUT)
