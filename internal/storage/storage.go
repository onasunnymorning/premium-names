package storage

import (
	"context"
	"io"
)

// ObjectStore defines minimal methods for upload/import needs.
type ObjectStore interface {
	// Get returns a reader for the given URI (s3://bucket/key or file://path).
	Get(ctx context.Context, uri string) (io.ReadCloser, int64, error)
	// Put writes content to the given URI (s3://bucket/key); returns final URI.
	Put(ctx context.Context, uri string, body io.Reader) (string, error)
}
