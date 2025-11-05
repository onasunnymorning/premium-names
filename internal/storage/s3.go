package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client *s3.Client
}

// NewS3 creates an S3 client honoring env configuration for MinIO.
// Env support: AWS_REGION, AWS_ENDPOINT_URL_S3, AWS_S3_FORCE_PATH_STYLE.
func NewS3(ctx context.Context) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if ep := os.Getenv("AWS_ENDPOINT_URL_S3"); ep != "" {
			o.BaseEndpoint = aws.String(ep)
		}
		if strings.EqualFold(os.Getenv("AWS_S3_FORCE_PATH_STYLE"), "true") {
			o.UsePathStyle = true
		}
	})
	return &S3Client{client: client}, nil
}

func parseS3(uri string) (bucket, key string, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", "", err
	}
	if u.Scheme != "s3" {
		return "", "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
	bucket = u.Host
	key = strings.TrimPrefix(u.Path, "/")
	if bucket == "" || key == "" {
		return "", "", errors.New("invalid s3 uri")
	}
	return
}

func (s *S3Client) Get(ctx context.Context, uri string) (io.ReadCloser, int64, error) {
	if strings.HasPrefix(uri, "file://") {
		p := strings.TrimPrefix(uri, "file://")
		f, err := os.Open(p)
		if err != nil {
			return nil, 0, err
		}
		info, _ := f.Stat()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		return f, size, nil
	}
	b, k, err := parseS3(uri)
	if err != nil {
		return nil, 0, err
	}
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &b, Key: &k})
	if err != nil {
		return nil, 0, err
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return out.Body, size, nil
}

func (s *S3Client) Put(ctx context.Context, uri string, body io.Reader) (string, error) {
	b, k, err := parseS3(uri)
	if err != nil {
		return "", err
	}
	uploader := manager.NewUploader(s.client)
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{Bucket: &b, Key: &k, Body: body})
	if err != nil {
		return "", err
	}
	return uri, nil
}
