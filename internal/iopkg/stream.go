package iopkg

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// s3iface is the minimal subset of s3 client methods we use; allows test fakes.
type s3iface interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// newS3Client constructs an s3 client; overridden in tests.
var newS3Client = func(ctx context.Context) (s3iface, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil { return nil, err }
	return s3.NewFromConfig(cfg), nil
}

// Open returns a ReadCloser and (if known) size for file:// or s3:// URIs.
func Open(uri string) (io.ReadCloser, int64, error) {
	u, err := url.Parse(uri)
	if err != nil { return nil, 0, err }
	switch u.Scheme {
	case "file", "":
		p := strings.TrimPrefix(uri, "file://")
		f, err := os.Open(p)
		if err != nil { return nil, 0, err }
		st, _ := f.Stat()
		var sz int64
		if st != nil { sz = st.Size() }
		return f, sz, nil
	case "s3":
		ctx := context.Background()
		cl, err := newS3Client(ctx)
		if err != nil { return nil, 0, err }
		bkt := u.Host
		key := strings.TrimPrefix(u.Path, "/")
		// Use GetObject streaming
		resp, err := cl.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bkt), Key: aws.String(key),
		})
		if err != nil { return nil, 0, err }
		var sz int64 = 0
		if resp.ContentLength != nil { sz = *resp.ContentLength }
		return resp.Body, sz, nil
	default:
		return nil, 0, errors.New("unsupported scheme: " + u.Scheme)
	}
}

func OpenReader(uri string) (io.ReadCloser, error) {
	rc, _, err := Open(uri)
	return rc, err
}

// Create creates a local file (file scheme). For S3 use CreateWriter with s3://.
func Create(path string) (io.Writer, io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return nil, nil, err }
	f, err := os.Create(path)
	if err != nil { return nil, nil, err }
	return f, f, nil
}

// CreateWriter supports file:// and s3://
func CreateWriter(uri string) (io.Writer, io.Closer, error) {
	if strings.HasPrefix(uri, "file://") || !strings.Contains(uri, "://") {
		p := strings.TrimPrefix(uri, "file://")
		return Create(p)
	}
	u, err := url.Parse(uri)
	if err != nil { return nil, nil, err }
	switch u.Scheme {
	case "s3":
		// buffer in memory and upload on Close (simple & safe)
		var buf bytes.Buffer
		type s3closer struct{
			io.Writer
			done bool
			upload func([]byte) error
		}
		sc := &s3closer{
			Writer: &buf,
			upload: func(b []byte) error {
				ctx := context.Background()
				cl, err := newS3Client(ctx)
				if err != nil { return err }
				_, err = cl.PutObject(ctx, &s3.PutObjectInput{
					Bucket: aws.String(u.Host),
					Key:    aws.String(strings.TrimPrefix(u.Path, "/")),
					Body:   bytes.NewReader(b),
				})
				return err
			},
		}
		return sc, closerFunc(func() error {
			if sc.done { return nil }
			sc.done = true
			return sc.upload(buf.Bytes())
		}), nil
	default:
		return nil, nil, errors.New("unsupported scheme for CreateWriter: " + u.Scheme)
	}
}

type closerFunc func() error
func (f closerFunc) Close() error { return f() }
