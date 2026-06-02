// Package s3source provides an S3-backed implementation of tokenizer.DictSource.
//
// The dictionary `.txt` is the source of truth and lives in S3; the compiled
// FST stays a local artifact whose path is chosen by the caller of
// tokenizer.NewDictionaryWithSource (S3Source itself never touches the
// filesystem).
package s3source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3API is the minimal S3 surface s3source needs. The aws-sdk-go-v2 *s3.Client
// satisfies this interface, and tests can provide a fake.
type S3API interface {
	GetObject(ctx context.Context, in *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, in *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// Source is a tokenizer.DictSource backed by a single S3 object.
type Source struct {
	client S3API
	bucket string
	key    string
}

// New constructs a Source from an explicit S3 client and an s3://bucket/key URI.
// Use this in tests (with a fake client) or when you want to control the AWS
// SDK client configuration yourself.
func New(client S3API, uri string) (*Source, error) {
	if client == nil {
		return nil, errors.New("s3source: client is nil")
	}
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return nil, err
	}
	return &Source{client: client, bucket: bucket, key: key}, nil
}

// NewFromURI constructs a Source using a default aws-sdk-go-v2 client built
// from the ambient AWS configuration (env vars, shared config, IAM role).
func NewFromURI(ctx context.Context, uri string) (*Source, error) {
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return nil, err
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("s3source: load AWS config: %w", err)
	}
	return &Source{client: s3.NewFromConfig(cfg), bucket: bucket, key: key}, nil
}

// Bucket returns the S3 bucket name.
func (s *Source) Bucket() string { return s.bucket }

// Key returns the S3 object key.
func (s *Source) Key() string { return s.key }

// Load fetches the object from S3 and returns its lines verbatim.
// A trailing empty line is dropped so round-tripping with Save is stable.
func (s *Source) Load() ([]string, error) {
	out, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3source: get s3://%s/%s: %w", s.bucket, s.key, err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("s3source: read s3://%s/%s: %w", s.bucket, s.key, err)
	}

	text := string(data)
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

// Save serializes words as "word\n" lines and overwrites the S3 object.
func (s *Source) Save(words []string) error {
	var buf bytes.Buffer
	for _, w := range words {
		buf.WriteString(w)
		buf.WriteByte('\n')
	}

	_, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s.key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("text/plain"),
	})
	if err != nil {
		return fmt.Errorf("s3source: put s3://%s/%s: %w", s.bucket, s.key, err)
	}
	return nil
}

// parseS3URI parses "s3://bucket/key" into bucket + key.
func parseS3URI(uri string) (string, string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", "", fmt.Errorf("s3source: parse %q: %w", uri, err)
	}
	if u.Scheme != "s3" {
		return "", "", fmt.Errorf("s3source: scheme must be s3, got %q", u.Scheme)
	}
	bucket := u.Host
	key := strings.TrimPrefix(u.Path, "/")
	if bucket == "" {
		return "", "", fmt.Errorf("s3source: missing bucket in %q", uri)
	}
	if key == "" {
		return "", "", fmt.Errorf("s3source: missing key in %q", uri)
	}
	return bucket, key, nil
}
