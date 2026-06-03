package tokenizer

import "testing"

func TestParseS3URI(t *testing.T) {
	tests := []struct {
		uri     string
		bucket  string
		key     string
		wantErr bool
	}{
		{"s3://my-bucket/dict.txt", "my-bucket", "dict.txt", false},
		{"s3://my-bucket/path/to/dict.txt", "my-bucket", "path/to/dict.txt", false},
		{"s3://my-bucket/", "", "", true},            // empty key
		{"s3:///key.txt", "", "", true},              // empty bucket
		{"s3://my-bucket", "", "", true},             // no key separator
		{"https://my-bucket/dict.txt", "", "", true}, // wrong scheme
		{"", "", "", true},                           // empty
	}
	for _, tt := range tests {
		bucket, key, err := parseS3URI(tt.uri)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseS3URI(%q) err=%v, wantErr=%v", tt.uri, err, tt.wantErr)
			continue
		}
		if tt.wantErr {
			continue
		}
		if bucket != tt.bucket || key != tt.key {
			t.Errorf("parseS3URI(%q) = (%q, %q), want (%q, %q)", tt.uri, bucket, key, tt.bucket, tt.key)
		}
	}
}
