package tokenizer

import (
	"bufio"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/blevesearch/vellum"
)

// Dictionary holds German compound word components in an FST for fast lookups.
type Dictionary struct {
	fst     *vellum.FST
	words   map[string]struct{} // Source of truth for modifications
	fstPath string
	txtPath string
	mu      sync.RWMutex
}

// NewDictionary loads the German compound word components dictionary from file into an FST.
// If FST doesn't exist, builds it from the text file.
func NewDictionary(txtPath string) (*Dictionary, error) {
	fstPath := strings.TrimSuffix(txtPath, ".txt") + ".fst"

	d := &Dictionary{
		words:   make(map[string]struct{}, 35000),
		fstPath: fstPath,
		txtPath: txtPath,
	}

	if err := d.loadTextFile(); err != nil {
		return nil, err
	}

	if err := d.loadOrBuildFST(); err != nil {
		return nil, err
	}

	return d, nil
}

// loadTextFile reads words from the source text file.
func (d *Dictionary) loadTextFile() error {
	file, err := os.Open(d.txtPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word == "" || strings.HasPrefix(word, "#") {
			continue
		}
		d.words[strings.ToLower(word)] = struct{}{}
	}
	return scanner.Err()
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

// RebuildFST rebuilds the FST from the current word set and saves to disk.
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

	// Create FST file
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

	return d.saveTextFile()
}

// saveTextFile writes the current word set back to the text file.
func (d *Dictionary) saveTextFile() error {
	sortedWords := make([]string, 0, len(d.words))
	for word := range d.words {
		sortedWords = append(sortedWords, word)
	}
	sort.Strings(sortedWords)

	file, err := os.Create(d.txtPath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, word := range sortedWords {
		if _, err := file.WriteString(word + "\n"); err != nil {
			return err
		}
	}
	return nil
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
