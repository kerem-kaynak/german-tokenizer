package s3source

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// fakeS3 implements S3API with in-memory storage.
type fakeS3 struct {
	objects map[string][]byte // key: "bucket/key"
	getErr  error
	putErr  error

	lastPutBody []byte
}

func (f *fakeS3) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	key := *in.Bucket + "/" + *in.Key
	data, ok := f.objects[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(data))}, nil
}

func (f *fakeS3) PutObject(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if f.putErr != nil {
		return nil, f.putErr
	}
	data, err := io.ReadAll(in.Body)
	if err != nil {
		return nil, err
	}
	if f.objects == nil {
		f.objects = make(map[string][]byte)
	}
	key := *in.Bucket + "/" + *in.Key
	f.objects[key] = data
	f.lastPutBody = data
	return &s3.PutObjectOutput{}, nil
}

func TestParseS3URI(t *testing.T) {
	cases := []struct {
		uri        string
		wantBucket string
		wantKey    string
		wantErr    bool
	}{
		{"s3://bucket/key", "bucket", "key", false},
		{"s3://my-bucket/path/to/dict.txt", "my-bucket", "path/to/dict.txt", false},
		{"http://bucket/key", "", "", true},
		{"s3://bucket", "", "", true},
		{"s3:///key", "", "", true},
		{"", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.uri, func(t *testing.T) {
			b, k, err := parseS3URI(c.uri)
			if c.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", c.uri)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if b != c.wantBucket || k != c.wantKey {
				t.Errorf("got (%q, %q), want (%q, %q)", b, k, c.wantBucket, c.wantKey)
			}
		})
	}
}

func TestNew_RejectsNilClient(t *testing.T) {
	if _, err := New(nil, "s3://b/k"); err == nil {
		t.Error("expected error for nil client")
	}
}

func TestSource_LoadRoundTrip(t *testing.T) {
	fake := &fakeS3{
		objects: map[string][]byte{
			"my-bucket/dict.txt": []byte("haus\nwand\ndecke\n"),
		},
	}
	src, err := New(fake, "s3://my-bucket/dict.txt")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	lines, err := src.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"haus", "wand", "decke"}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("Load = %v, want %v", lines, want)
	}
}

func TestSource_LoadEmptyObject(t *testing.T) {
	fake := &fakeS3{
		objects: map[string][]byte{"b/k": []byte("")},
	}
	src, _ := New(fake, "s3://b/k")
	lines, err := src.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected no lines, got %v", lines)
	}
}

func TestSource_Save(t *testing.T) {
	fake := &fakeS3{}
	src, err := New(fake, "s3://my-bucket/dict.txt")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := src.Save([]string{"alpha", "beta", "gamma"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	want := "alpha\nbeta\ngamma\n"
	if string(fake.lastPutBody) != want {
		t.Errorf("PutObject body = %q, want %q", fake.lastPutBody, want)
	}
}

func TestSource_LoadError(t *testing.T) {
	fake := &fakeS3{getErr: errors.New("network broken")}
	src, _ := New(fake, "s3://b/k")
	_, err := src.Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "s3source") {
		t.Errorf("error not wrapped with s3source prefix: %v", err)
	}
}

func TestSource_SaveError(t *testing.T) {
	fake := &fakeS3{putErr: errors.New("access denied")}
	src, _ := New(fake, "s3://b/k")
	err := src.Save([]string{"x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "s3source") {
		t.Errorf("error not wrapped: %v", err)
	}
}

func TestSource_LoadThenSaveRoundTrip(t *testing.T) {
	fake := &fakeS3{
		objects: map[string][]byte{"b/k": []byte("haus\nwand\n")},
	}
	src, _ := New(fake, "s3://b/k")

	lines, err := src.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := src.Save(lines); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if string(fake.lastPutBody) != "haus\nwand\n" {
		t.Errorf("round-trip body = %q", fake.lastPutBody)
	}
}
