package tokenizer

import (
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/blevesearch/vellum"
)

// Dictionary holds German compound word components in an FST for fast lookups.
// The source-of-truth words list is loaded/persisted via a DictSource (local
// file or S3). The FST is always a local file at fstPath.
type Dictionary struct {
	fst     *vellum.FST
	words   map[string]struct{} // Source of truth for modifications
	source  DictSource
	fstPath string
	mu      sync.RWMutex
}

// NewDictionary loads the dictionary from a local .txt file. The FST is built
// at the sibling `.fst` path (e.g. `dict.txt` → `dict.fst`).
//
// This is a thin wrapper over NewDictionaryWithSource using a FileSource.
func NewDictionary(txtPath string) (*Dictionary, error) {
	return NewDictionaryWithSource(&FileSource{Path: txtPath}, defaultFSTPath(txtPath))
}

// NewDictionaryWithSource loads the dictionary from the given source. The
// compiled FST is read from / written to fstPath (always a local file).
func NewDictionaryWithSource(source DictSource, fstPath string) (*Dictionary, error) {
	d := &Dictionary{
		words:   make(map[string]struct{}, 35000),
		source:  source,
		fstPath: fstPath,
	}

	if err := d.loadWords(); err != nil {
		return nil, err
	}

	if err := d.loadOrBuildFST(); err != nil {
		return nil, err
	}

	return d, nil
}

// loadWords pulls raw lines from the source and applies the dictionary's
// parsing rules (skip blank/comment lines, trim, lowercase).
func (d *Dictionary) loadWords() error {
	lines, err := d.source.Load()
	if err != nil {
		return err
	}
	for _, line := range lines {
		word := strings.TrimSpace(line)
		if word == "" || strings.HasPrefix(word, "#") {
			continue
		}
		d.words[strings.ToLower(word)] = struct{}{}
	}
	return nil
}

// loadOrBuildFST loads existing FST or builds a new one.
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

// RebuildFST rebuilds the FST from the current word set and persists to the source.
func (d *Dictionary) RebuildFST() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.rebuildFST()
}

// rebuildFST rebuilds FST without locking (caller must hold lock).
func (d *Dictionary) rebuildFST() error {
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
		return err
	}

	builder, err := vellum.New(fstFile, nil)
	if err != nil {
		fstFile.Close()
		return err
	}

	for _, word := range sortedWords {
		if err := builder.Insert([]byte(word), 0); err != nil {
			builder.Close()
			fstFile.Close()
			return err
		}
	}

	if err := builder.Close(); err != nil {
		fstFile.Close()
		return err
	}
	fstFile.Close()

	fst, err := vellum.Open(d.fstPath)
	if err != nil {
		return err
	}
	d.fst = fst

	return d.source.Save(sortedWords)
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
