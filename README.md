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

## Start a workflow (example)

Use the Makefile helper (tctl must be installed locally):

```bash
make start-workflow START_INPUT=examples/request.example.json
```

Or invoke tctl directly (note the correct flag is --workflow_type):

```bash
tctl workflow start \
  --taskqueue zone-names \
  --workflow_type ZoneNamesWorkflow \
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
