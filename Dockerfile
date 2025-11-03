# syntax=docker/dockerfile:1.7

# ---- Builder stage ----
FROM golang:1.23-bookworm AS builder

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
    go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# ---- Runtime stage ----
# Distroless static includes CA certs and runs as nonroot
FROM gcr.io/distroless/static:nonroot AS runtime

WORKDIR /
COPY --from=builder /out/worker /worker

# Default environment (override at runtime as needed)
# Temporal connection
ENV TEMPORAL_ADDRESS=127.0.0.1:7233 \
    TEMPORAL_TARGET_HOST=127.0.0.1:7233 \
    TEMPORAL_NAMESPACE=default \
    TEMPORAL_TASK_QUEUE=zone-names \
    LOG_LEVEL=info

# Optional metrics port if you expose a Prometheus HTTP endpoint later
EXPOSE 9090

USER nonroot
ENTRYPOINT ["/worker"]
