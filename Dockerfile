# syntax=docker/dockerfile:1.7

# ---- Builder stage ----
FROM golang:1.24-bookworm AS builder

# Configure environment for a static build
ENV CGO_ENABLED=0 \
    GO111MODULE=on

WORKDIR /src

# Cache deps first
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the rest of the source
COPY . .

# Build the worker binary for Linux; allow GOARCH override at build time
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV GOOS=${TARGETOS} GOARCH=${TARGETARCH}
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker && \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    go build -trimpath -ldflags="-s -w" -o /out/importer ./cmd/importer

# Prepare a world-writable scratch dir to seed the runtime volume perms
RUN mkdir -p /seed-tmp/zone-names && chmod 0777 /seed-tmp/zone-names

# ---- Runtime stage ----
# Distroless static includes CA certs and runs as nonroot
FROM gcr.io/distroless/static:nonroot AS runtime

WORKDIR /
COPY --from=builder /out/worker /worker
COPY --from=builder /out/api /api
COPY --from=builder /out/importer /importer
COPY --from=builder /seed-tmp/zone-names /var/zone-names

# Default environment (override at runtime as needed)
# Temporal connection
ENV TEMPORAL_ADDRESS=127.0.0.1:7233 \
    TEMPORAL_TARGET_HOST=127.0.0.1:7233 \
    TEMPORAL_NAMESPACE=default \
    TEMPORAL_TASK_QUEUE=zone-names \
    LOG_LEVEL=info \
    ZN_TMP_DIR=/var/zone-names

# Optional metrics port if you expose a Prometheus HTTP endpoint later
EXPOSE 9090

USER nonroot
ENTRYPOINT ["/worker"]
