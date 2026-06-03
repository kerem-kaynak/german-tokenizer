package tokenizer

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// parseS3URI splits "s3://bucket/key" (key may contain "/"). Both parts required.
func parseS3URI(uri string) (bucket, key string, err error) {
	rest, ok := strings.CutPrefix(uri, "s3://")
	if !ok {
		return "", "", fmt.Errorf("not an s3 URI: %q", uri)
	}
	bucket, key, ok = strings.Cut(rest, "/")
	if !ok || bucket == "" || key == "" {
		return "", "", fmt.Errorf("invalid s3 URI %q: expected s3://bucket/key", uri)
	}
	return bucket, key, nil
}

// s3Source reads and writes the dictionary as a single S3 object.
// Credentials and region come from the AWS default config chain
// (env vars, shared config files, container/IAM role).
type s3Source struct {
	client *s3.Client
	bucket string
	key    string
}

func newS3Source(uri string) (s3Source, error) {
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return s3Source{}, err
	}
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return s3Source{}, fmt.Errorf("load aws config: %w", err)
	}
	return s3Source{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
		key:    key,
	}, nil
}

func (s s3Source) load() ([]string, error) {
	out, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get s3://%s/%s: %w", s.bucket, s.key, err)
	}
	defer out.Body.Close()
	return parseDictLines(out.Body)
}

func (s s3Source) save(words []string) error {
	data := serializeDict(words)
	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(s.key),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
	})
	if err != nil {
		return fmt.Errorf("s3 put s3://%s/%s: %w", s.bucket, s.key, err)
	}
	return nil
}
