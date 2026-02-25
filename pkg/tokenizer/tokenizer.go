package tokenizer

// Tokenizer is the main German tokenizer.
type Tokenizer struct {
	dict       *Dictionary
	normalizer *Normalizer
	splitter   *CompoundSplitter
}

// NewTokenizer creates a tokenizer with the default normalizer pipeline.
func NewTokenizer(dictPath string) (*Tokenizer, error) {
	dict, err := NewDictionary(dictPath)
	if err != nil {
		return nil, err
	}

	return &Tokenizer{
		dict:       dict,
		normalizer: NewNormalizer(),
		splitter:   NewCompoundSplitter(dict),
	}, nil
}

// NewTokenizerWithNormalizer creates a tokenizer with a custom normalizer.
func NewTokenizerWithNormalizer(dictPath string, norm *Normalizer) (*Tokenizer, error) {
	dict, err := NewDictionary(dictPath)
	if err != nil {
		return nil, err
	}

	return &Tokenizer{
		dict:       dict,
		normalizer: norm,
		splitter:   NewCompoundSplitter(dict),
	}, nil
}

// NewTokenizerNoCache creates a tokenizer with caching disabled.
// Use this when memory is constrained or compounds are rarely repeated.
func NewTokenizerNoCache(dictPath string) (*Tokenizer, error) {
	dict, err := NewDictionary(dictPath)
	if err != nil {
		return nil, err
	}

	return &Tokenizer{
		dict:       dict,
		normalizer: NewNormalizer(),
		splitter:   NewCompoundSplitterNoCache(dict),
	}, nil
}

// Tokenize processes input text and returns deduplicated tokens.
func (t *Tokenizer) Tokenize(text string) []string {
	rawTokens := SplitWords(text)

	resultSet := make(map[string]struct{})
	var results []string

	for _, raw := range rawTokens {
		if raw.Type != TokenWord {
			continue
		}

		// Compound decomposition
		segments := t.splitter.Split(raw.Text)

		// Add lowercase original (preserves umlauts)
		original := t.normalizer.LowercaseOnly(raw.Text)
		if _, exists := resultSet[original]; !exists {
			resultSet[original] = struct{}{}
			results = append(results, original)
		}

		// Add normalized+stemmed segments
		for _, seg := range segments {
			normalized := t.normalizer.Normalize(seg)
			if _, exists := resultSet[normalized]; !exists {
				resultSet[normalized] = struct{}{}
				results = append(results, normalized)
			}
		}
	}

	return results
}

// AddWord adds a word to the dictionary.
// Note: Changes are not persisted until RebuildDictionary() is called.
func (t *Tokenizer) AddWord(word string) {
	t.dict.AddWord(word)
}

// RemoveWord removes a word from the dictionary.
// Note: Changes are not persisted until RebuildDictionary() is called.
func (t *Tokenizer) RemoveWord(word string) {
	t.dict.RemoveWord(word)
}

// RebuildDictionary rebuilds the FST and persists changes to disk.
func (t *Tokenizer) RebuildDictionary() error {
	return t.dict.RebuildFST()
}

// Close releases resources (call when done with tokenizer).
func (t *Tokenizer) Close() error {
	return t.dict.Close()
}

// DictionaryWordCount returns the number of words in the dictionary.
func (t *Tokenizer) DictionaryWordCount() int {
	return t.dict.WordCount()
}

// CacheSize returns the number of cached compound splits.
func (t *Tokenizer) CacheSize() int {
	return t.splitter.CacheSize()
}

// ClearCache clears the compound splitting cache.
func (t *Tokenizer) ClearCache() {
	t.splitter.ClearCache()
}

// CacheEnabled returns true if caching is enabled.
func (t *Tokenizer) CacheEnabled() bool {
	return t.splitter.CacheEnabled()
}
