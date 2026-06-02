package tokenizer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DictSource abstracts how the dictionary's source-of-truth `.txt` is loaded
// and persisted. The compiled FST is always a local artifact and is owned by
// Dictionary; sources only transport the words list.
//
// Load returns the raw lines from the underlying storage (no comment-stripping,
// no lowercasing — Dictionary applies those rules). Save receives words that
// are already sorted and lowercased and persists them as one word per line.
type DictSource interface {
	Load() ([]string, error)
	Save(words []string) error
}

// FileSource reads/writes the dictionary `.txt` from a local file path.
type FileSource struct {
	Path string
}

// Load reads the file at FileSource.Path and returns its lines verbatim.
func (f *FileSource) Load() ([]string, error) {
	file, err := os.Open(f.Path)
	if err != nil {
		return nil, fmt.Errorf("filesource: open %q: %w", f.Path, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("filesource: scan %q: %w", f.Path, err)
	}
	return lines, nil
}

// Save writes the given words to FileSource.Path, one per line.
// Writes go to a sibling temp file and are renamed into place for atomicity.
func (f *FileSource) Save(words []string) error {
	dir := filepath.Dir(f.Path)
	tmp, err := os.CreateTemp(dir, filepath.Base(f.Path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("filesource: create temp in %q: %w", dir, err)
	}
	tmpPath := tmp.Name()

	w := bufio.NewWriter(tmp)
	for _, word := range words {
		if _, err := w.WriteString(word + "\n"); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("filesource: write %q: %w", tmpPath, err)
		}
	}
	if err := w.Flush(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("filesource: flush %q: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("filesource: close %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, f.Path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("filesource: rename %q -> %q: %w", tmpPath, f.Path, err)
	}
	return nil
}

// defaultFSTPath derives the conventional FST sibling path for a given .txt.
func defaultFSTPath(txtPath string) string {
	return strings.TrimSuffix(txtPath, ".txt") + ".fst"
}
