package tokenizer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/blevesearch/vellum"
)

// Dictionary holds German compound word components in an FST for fast lookups.
type Dictionary struct {
	fst     *vellum.FST
	words   map[string]struct{} // Source of truth for modifications
	source  dictSource          // Where the word list is read from / written to
	fstPath string              // FST is always a local file
	mu      sync.RWMutex
}

// NewDictionary loads the German compound word components dictionary.
//
// If path begins with "s3://" (e.g. "s3://my-bucket/dicts/german.txt") the
// word list is read from S3 and the FST is built into a fresh file under
// os.TempDir()/german-tokenizer/. Otherwise path is a local .txt file and
// the FST is cached alongside it (".txt" → ".fst") — existing behavior.
func NewDictionary(path string) (*Dictionary, error) {
	d := &Dictionary{words: make(map[string]struct{}, 35000)}

	if strings.HasPrefix(path, "s3://") {
		s3src, err := newS3Source(path)
		if err != nil {
			return nil, err
		}
		fstPath, err := s3FSTPath(s3src.bucket, s3src.key)
		if err != nil {
			return nil, err
		}
		d.source = s3src
		d.fstPath = fstPath

		if err := d.loadFromSource(); err != nil {
			return nil, err
		}
		// S3 mode: load source, build FST locally, do NOT save back.
		// Works under read-only S3 credentials; the temp FST is per-task scratch
		// rebuilt on every startup (no cross-process FST sharing on ECS).
		if _, err := d.buildFST(); err != nil {
			return nil, err
		}
		return d, nil
	}

	// Local file mode (unchanged behavior).
	d.source = fileSource{path: path}
	d.fstPath = strings.TrimSuffix(path, ".txt") + ".fst"
	if err := d.loadFromSource(); err != nil {
		return nil, err
	}
	if err := d.loadOrBuildFST(); err != nil {
		return nil, err
	}
	return d, nil
}

// s3FSTPath returns where to put the locally-built FST for an S3-backed dictionary.
// Slashes in the key are flattened so the result is a single filename.
func s3FSTPath(bucket, key string) (string, error) {
	dir := filepath.Join(os.TempDir(), "german-tokenizer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, bucket+"_"+strings.ReplaceAll(key, "/", "_")+".fst"), nil
}

// loadFromSource reads the word list from the backing source into the word set.
func (d *Dictionary) loadFromSource() error {
	words, err := d.source.load()
	if err != nil {
		return err
	}
	for _, word := range words {
		d.words[word] = struct{}{}
	}
	return nil
}

// loadOrBuildFST loads existing FST or builds a new one (persisting via source).
func (d *Dictionary) loadOrBuildFST() error {
	if fst, err := vellum.Open(d.fstPath); err == nil {
		d.fst = fst
		return nil
	}

	return d.rebuildFST()
}

// Contains checks if a word exists in the dictionary (case-insensitive).
// Always uses FST for lookups.
func (d *Dictionary) Contains(word string) bool {
	lower := strings.ToLower(word)

	d.mu.RLock()
	defer d.mu.RUnlock()

	_, exists, _ := d.fst.Get([]byte(lower))
	return exists
}

// AddWord adds a word to the dictionary and rebuilds FST.
func (d *Dictionary) AddWord(word string) error {
	lower := strings.ToLower(word)

	d.mu.Lock()
	defer d.mu.Unlock()

	d.words[lower] = struct{}{}
	return d.rebuildFST()
}

// RemoveWord removes a word from the dictionary and rebuilds FST.
func (d *Dictionary) RemoveWord(word string) error {
	lower := strings.ToLower(word)

	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.words, lower)
	return d.rebuildFST()
}

// RebuildFST rebuilds the FST from the current word set and saves to disk.
func (d *Dictionary) RebuildFST() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.rebuildFST()
}

// buildFST rebuilds the FST in-memory from d.words and writes it to fstPath.
// Returns the sorted word slice so callers that also persist via source can
// reuse the already-sorted list. Does NOT call source.save.
// Caller must hold the write lock (or be the constructor, pre-publication).
func (d *Dictionary) buildFST() ([]string, error) {
	if d.fst != nil {
		d.fst.Close()
		d.fst = nil
	}

	sortedWords := make([]string, 0, len(d.words))
	for word := range d.words {
		sortedWords = append(sortedWords, word)
	}
	sort.Strings(sortedWords)

	fstFile, err := os.Create(d.fstPath)
	if err != nil {
		return nil, err
	}

	builder, err := vellum.New(fstFile, nil)
	if err != nil {
		fstFile.Close()
		return nil, err
	}

	for _, word := range sortedWords {
		if err := builder.Insert([]byte(word), 0); err != nil {
			builder.Close()
			fstFile.Close()
			return nil, err
		}
	}

	if err := builder.Close(); err != nil {
		fstFile.Close()
		return nil, err
	}
	fstFile.Close()

	fst, err := vellum.Open(d.fstPath)
	if err != nil {
		return nil, err
	}
	d.fst = fst
	return sortedWords, nil
}

// rebuildFST builds the FST and persists the word list back through the source.
// Caller must hold the write lock.
func (d *Dictionary) rebuildFST() error {
	sortedWords, err := d.buildFST()
	if err != nil {
		return err
	}
	return d.source.save(sortedWords)
}

// Close releases FST resources.
func (d *Dictionary) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.fst != nil {
		err := d.fst.Close()
		d.fst = nil
		return err
	}
	return nil
}

// WordCount returns the number of words in the dictionary.
func (d *Dictionary) WordCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.words)
}
