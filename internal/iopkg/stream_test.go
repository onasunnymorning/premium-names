package iopkg

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type fakeS3 struct {
	getBody       []byte
	getErr        error
	putLastBucket string
	putLastKey    string
	putLastBody   []byte
	putErr        error
}

func (f *fakeS3) GetObject(ctx context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if f.getErr != nil { return nil, f.getErr }
	rc := io.NopCloser(bytes.NewReader(f.getBody))
	cl := int64(len(f.getBody))
	return &s3.GetObjectOutput{Body: rc, ContentLength: &cl}, nil
}
func (f *fakeS3) PutObject(ctx context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if f.putErr != nil { return nil, f.putErr }
	f.putLastBucket = aws.ToString(in.Bucket)
	f.putLastKey = aws.ToString(in.Key)
	if in.Body != nil {
		b, _ := io.ReadAll(in.Body)
		f.putLastBody = b
	}
	return &s3.PutObjectOutput{}, nil
}

func withFakeS3(t *testing.T, f *fakeS3) func() {
	old := newS3Client
	newS3Client = func(ctx context.Context) (s3iface, error) { return f, nil }
	return func() { newS3Client = old }
}

func TestOpenFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "z.txt")
	content := "hello world\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil { t.Fatal(err) }
	rc, sz, err := Open("file://" + p)
	if err != nil { t.Fatalf("Open err: %v", err) }
	defer rc.Close()
	if sz != int64(len(content)) { t.Fatalf("size got %d want %d", sz, len(content)) }
	b, _ := io.ReadAll(rc)
	if string(b) != content { t.Fatalf("content mismatch: %q", string(b)) }
}

func TestCreateWriterFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "out.txt")
	w, c, err := CreateWriter("file://" + p)
	if err != nil { t.Fatalf("CreateWriter err: %v", err) }
	_, _ = w.Write([]byte("abc"))
	if err := c.Close(); err != nil { t.Fatalf("close err: %v", err) }
	b, _ := os.ReadFile(p)
	if string(b) != "abc" { t.Fatalf("file content: %q", string(b)) }
}

func TestOpenS3Mock(t *testing.T) {
	f := &fakeS3{ getBody: []byte("data-from-s3") }
	defer withFakeS3(t, f)()
	rc, sz, err := Open("s3://bucket/key/path.txt")
	if err != nil { t.Fatalf("Open s3 err: %v", err) }
	defer rc.Close()
	if sz != int64(len(f.getBody)) { t.Fatalf("size got %d want %d", sz, len(f.getBody)) }
	b, _ := io.ReadAll(rc)
	if string(b) != string(f.getBody) { t.Fatalf("content mismatch: %q", string(b)) }
}

func TestCreateWriterS3Mock(t *testing.T) {
	f := &fakeS3{}
	defer withFakeS3(t, f)()
	w, c, err := CreateWriter("s3://mybucket/dir/name.txt")
	if err != nil { t.Fatalf("CreateWriter s3 err: %v", err) }
	_, _ = w.Write([]byte("payload"))
	if err := c.Close(); err != nil { t.Fatalf("close err: %v", err) }
	if f.putLastBucket != "mybucket" { t.Fatalf("bucket %q", f.putLastBucket) }
	if f.putLastKey != "dir/name.txt" { t.Fatalf("key %q", f.putLastKey) }
	if string(f.putLastBody) != "payload" { t.Fatalf("body %q", string(f.putLastBody)) }
}
