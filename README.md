# zone-names (Temporal workflow)

Stream a DNS zone (file or S3), extract owner names, dedupe at scale with Badger, and write:
- `names.txt` (sorted unique)
- `manifest.json` (counts + params)

## Build & Run Worker

```bash
go mod tidy
export TEMPORAL_TARGET_HOST=localhost:7233
export TEMPORAL_NAMESPACE=default
export TEMPORAL_TASK_QUEUE=zone-names
export ZN_TMP_DIR=/tmp/zone-names
go run ./cmd/worker
```

## Local Temporal + MinIO stack

A docker compose stack is provided to run Temporal Server, Temporal UI, and a MinIO S3-compatible service locally.

Start the stack (includes Temporal, UI, MinIO, and the worker):

```bash
make dev-up
```

Open UIs:

- Temporal UI: http://localhost:8080
- MinIO Console: http://localhost:9001 (default user/pass: minioadmin / minioadmin)

Temporal Frontend gRPC is exposed at 127.0.0.1:7233.

Stop the stack:

```bash
make dev-down
```

### Running the worker manually (optional)

You can still build and run the worker locally or via its container if you prefer:

```bash
# local build
make build

# run the worker (connecting to the compose Temporal at localhost)
TEMPORAL_ADDRESS=127.0.0.1:7233 LOG_LEVEL=info bin/worker

# or run the container directly (outside compose)
make docker-build IMAGE=zone-names-worker TAG=dev
make docker-run IMAGE=zone-names-worker TAG=dev TEMPORAL_ADDRESS=host.docker.internal:7233
```

### S3 via MinIO

Point your AWS SDK to MinIO when running locally by exporting credentials:

```bash
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_REGION=us-east-1
export AWS_ENDPOINT_URL_S3=http://localhost:9000
```

MinIO will auto-create a bucket named `zone-names` via the `minio-init` one-shot job. The worker exposes Prometheus metrics at http://localhost:9090/metrics when running under compose.

## HTTP API

The dev stack also includes a small HTTP API (Gin) exposing CRUD and list endpoints for batches, labels, and tags backed by Postgres.

- Base URL: http://localhost:8081/api
- Start via compose: `make dev-up` (the `api` service is part of the stack)

Endpoints (selection):
- POST /api/batches — create a batch
- GET /api/labels — list labels filtered by tags and/or batch: `?tags=tag1,tag2&mode=any|all&batch=123&limit=100&offset=0`
- POST /api/labels/:id/tags — add a tag to a label
- DELETE /api/labels/:id/tags/:tagId — remove a tag from a label
- POST /api/labels/tags/apply — bulk-apply a tag to labels matching a filter
- GET /api/tags?prefix=ca&limit=20 — type-ahead search for tags
- POST /api/tags — create a tag
- PATCH /api/tags/:id — rename a tag and/or change group
- DELETE /api/tags/:id — delete a tag
- GET /api/export — CSV export of labels matching a filter

Notes:
- The API service waits for database migrations to complete.
- Default port is 8081; override with `PORT`.
- Upload and import:
  - POST `/api/batches/{id}/upload` (multipart/form-data, field `file`) uploads to S3/MinIO and enqueues an import job.
  - An `importer` service consumes jobs and processes files.
  - OpenAPI: http://localhost:8081/api/openapi.yaml

### Upload/import pipeline

- Supported input formats: CSV, TSV, TXT, XLSX, XLS (first column is the domain; other columns ignored)
- Tolerant of “dirty” data:
  - Skips empty lines and obvious header rows
  - Handles quoted newlines, variable column counts (CSV/TSV)
  - Normalizes to first label of domain, IDNA punycode, LDH rules
  - In-file dedupe on normalized ASCII label

Processing steps:
1) Upload via `/api/batches/{id}/upload`. File is stored at `s3://$IMPORT_BUCKET/uploads/batch-{id}/...`.
2) Importer downloads, parses first column, normalizes, upserts labels.
3) Bulk links labels to the batch via COPY for high throughput.

## Start a workflow (example)

Use the Makefile helper (tctl must be installed locally):

```bash
make start-workflow START_INPUT=examples/request.example.json
```

Or invoke tctl directly (note the correct flag is --workflow_type):

```bash
tctl workflow start \
  --taskqueue zone-names \
  --workflow_type Zone2NamesWorkflow \
  --input '{"ZoneURI":"s3://zone-names/org/org.txt.gz","OutputURI":"s3://zone-names/org/names.txt","Shards":64,"Filters":["A","AAAA","CNAME"],"IDNMode":"none","KeepScratch":false}'
```

Tip: to avoid shell-escaping issues, use --input_file examples/request.example.json instead of --input.

### S3 credentials
Relies on default AWS credential chain (env, shared config, role, etc.).

### Notes
- `ZN_TMP_DIR` must be a fast local disk with enough space. In Docker (compose), the worker uses `/var/zone-names` by default; change via env.
- To write to `file://` instead of S3, set `OutputURI` accordingly.
- `IDNMode`: `alabel`, `ulabel`, or `none`.
- `Filters` empty = include all types.

## Scratch directory and cleanup

- The worker writes temporary files under a scratch root (`ZN_TMP_DIR`).
- Each workflow execution uses a subdirectory, defaulting to the Temporal `WorkflowId`. Example: `/tmp/zone-names/<workflow-id>/`.
- You can override the subdirectory via the optional input field `ScratchSubdir`. If `ScratchSubdir` is empty or omitted, the workflow id is used.
- Automatic cleanup:
  - On success, the workflow deletes its scratch subdirectory by default.
  - On any failure (partition, dedupe, merge), the workflow attempts to delete the subdirectory before returning an error.
  - To keep artifacts for debugging, set `KeepScratch: true` in the input.

Example `examples/request.example.json` fields:

```json
{
  "ZoneURI": "s3://zone-names/net.txt.gz",
  "OutputURI": "s3://zone-names/net.names.txt",
  "Shards": 32,
  "Filters": ["NS"],
  "IDNMode": "none",
  "ScratchSubdir": "",   
  "KeepScratch": false     
}
```

## Local development quick reference

- Start stack (Temporal, UI, MinIO, Postgres, API, worker, importer):

```bash
make dev-up
```

- Tail logs:

```bash
make dev-logs              # all services
make dev-api-logs          # API only
make dev-worker-logs       # worker only
make dev-importer-logs     # importer only
```

- Rebuild one service in compose:

```bash
make dev-restart-api
make dev-restart-importer
```

- Run locally (outside Docker):

```bash
# API
make run-api

# Importer
make run-importer
```

- Migrations (compose-backed):

```bash
make migrate-up                  # apply up to latest
make migrate-down                # roll back one step
MIGRATE_OPTS=-all make migrate-down  # roll back all
```

### Required env (compose defaults)

Database:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=temporal
export DB_PASSWORD=temporal
export DB_NAME=premium_names
export DB_SSLMODE=disable
```

S3/MinIO:

```bash
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_REGION=us-east-1
export AWS_ENDPOINT_URL_S3=http://localhost:9000
export AWS_S3_FORCE_PATH_STYLE=true
export IMPORT_BUCKET=zone-names
```

## Frontend (Next.js UI)

A lightweight Next.js app provides a simple UI to:
- Browse labels with filters (tags ANY/ALL, optional batch id), export CSV, and bulk-apply a tag to the current result set
- Manage tags (create, rename, delete) with type-ahead search
- Create a batch and upload a file to kick off an import job

Getting started:

```bash
# In another terminal, keep the backend stack running
make dev-up

# Run the frontend (installs deps on first run)
make dev-frontend
```

Then open http://localhost:3000.

Configuration:

- The frontend reads the API base from `NEXT_PUBLIC_API_BASE` (defaults to `http://localhost:8081`).
- You can copy `frontend/.env.example` to `frontend/.env.local` and customize it if needed.

Notes:
- File uploads go directly to the backend API endpoint `/api/batches/:id/upload` using `multipart/form-data`.
- CORS is enabled in the API for local development.
