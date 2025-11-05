FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /zone-names ./cmd/api

FROM alpine:latest
WORKDIR /app
COPY --from=builder /zone-names .
COPY web ./web

EXPOSE 8080
CMD ["./zone-names"]
