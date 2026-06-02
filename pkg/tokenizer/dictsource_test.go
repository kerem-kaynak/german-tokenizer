package tokenizer

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFileSource_LoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dict.txt")

	original := "# comment line\n\nhaus\nwand\nDecke\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	fs := &FileSource{Path: path}
	lines, err := fs.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := []string{"# comment line", "", "haus", "wand", "Decke"}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("Load() = %#v, want %#v", lines, want)
	}
}

func TestFileSource_SaveAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dict.txt")

	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	fs := &FileSource{Path: path}
	if err := fs.Save([]string{"alpha", "beta", "gamma"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	if string(got) != "alpha\nbeta\ngamma\n" {
		t.Errorf("Save wrote %q", got)
	}

	// No leftover tmp files in the directory.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "dict.txt" {
			t.Errorf("unexpected leftover file %q", e.Name())
		}
	}
}

func TestFileSource_LoadMissingFile(t *testing.T) {
	fs := &FileSource{Path: filepath.Join(t.TempDir(), "nope.txt")}
	if _, err := fs.Load(); err == nil {
		t.Error("expected error for missing file")
	}
}
