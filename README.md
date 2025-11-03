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

Start the stack:

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

### Running the worker

Build locally and run directly, or use the container image:

```bash
# local build
make build

# run the worker (connecting to the compose Temporal at localhost)
TEMPORAL_ADDRESS=127.0.0.1:7233 LOG_LEVEL=info bin/worker

# or run the container
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

MinIO will auto-create a bucket named `zone-names` via the `minio-init` one-shot job.

## Start a workflow (example)

```bash
tctl workflow start   --taskqueue zone-names   --workflow ZoneNamesWorkflow   --input '{ 
    "ZoneURI":"s3://your-bucket/path/org.zone.gz",
    "OutputURI":"s3://your-bucket/output/org/names.txt",
    "Shards":64,
    "Filters":["A","AAAA","CNAME"],
    "IDNMode":"none"
  }'
```

### S3 credentials
Relies on default AWS credential chain (env, shared config, role, etc.).

### Notes
- `ZN_TMP_DIR` must be a fast local disk with enough space.
- To write to `file://` instead of S3, set `OutputURI` accordingly.
- `IDNMode`: `alabel`, `ulabel`, or `none`.
- `Filters` empty = include all types.
