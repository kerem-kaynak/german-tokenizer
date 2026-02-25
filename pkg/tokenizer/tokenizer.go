package tokenizer

// Option configures a Tokenizer.
type Option func(*config)

// config holds tokenizer configuration with defaults.
type config struct {
	cache                    bool
	includeLowercaseOriginal bool
	normalizerSteps          []NormalizerFunc
}

// defaultConfig returns configuration with sensible defaults.
func defaultConfig() *config {
	return &config{
		cache:                    true,
		includeLowercaseOriginal: true,
		normalizerSteps:          nil, // nil means use default normalizer
	}
}

// WithCache enables compound splitting cache (default: true).
func WithCache(enabled bool) Option {
	return func(c *config) {
		c.cache = enabled
	}
}

// WithLowercaseOriginal includes lowercase original token in output (default: true).
// When enabled, "Wärmedämmung" outputs both "wärmedämmung" and normalized segments.
// When disabled, only normalized segments are output.
func WithLowercaseOriginal(enabled bool) Option {
	return func(c *config) {
		c.includeLowercaseOriginal = enabled
	}
}

// WithNormalizerSteps sets custom normalization steps.
// Pass nil or omit to use default pipeline (all steps).
// Example: WithNormalizerSteps(NFKDDecompose, Lowercase, StemGerman)
func WithNormalizerSteps(steps ...NormalizerFunc) Option {
	return func(c *config) {
		c.normalizerSteps = steps
	}
}

// Tokenizer is the main German tokenizer.
type Tokenizer struct {
	dict                     *Dictionary
	normalizer               *Normalizer
	splitter                 *CompoundSplitter
	includeLowercaseOriginal bool
}

// NewTokenizer creates a tokenizer with the given options.
//
// Example usage:
//
//	// Default configuration (cache on, lowercase original on, all normalizers)
//	tok, _ := NewTokenizer(dictPath)
//
//	// Disable cache
//	tok, _ := NewTokenizer(dictPath, WithCache(false))
//
//	// Disable lowercase original output
//	tok, _ := NewTokenizer(dictPath, WithLowercaseOriginal(false))
//
//	// Custom normalizer pipeline (skip eszett conversion)
//	tok, _ := NewTokenizer(dictPath, WithNormalizerSteps(
//	    NFKDDecompose,
//	    Lowercase,
//	    RemoveCombiningMarks,
//	    StemGerman,
//	))
//
//	// Combine options
//	tok, _ := NewTokenizer(dictPath,
//	    WithCache(false),
//	    WithLowercaseOriginal(false),
//	    WithNormalizerSteps(Lowercase, StemGerman),
//	)
func NewTokenizer(dictPath string, opts ...Option) (*Tokenizer, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	dict, err := NewDictionary(dictPath)
	if err != nil {
		return nil, err
	}

	// Build normalizer
	var normalizer *Normalizer
	if cfg.normalizerSteps != nil {
		normalizer = NewNormalizerWithSteps(cfg.normalizerSteps...)
	} else {
		normalizer = NewNormalizer()
	}

	// Build compound splitter
	var splitter *CompoundSplitter
	if cfg.cache {
		splitter = NewCompoundSplitter(dict)
	} else {
		splitter = NewCompoundSplitterNoCache(dict)
	}

	return &Tokenizer{
		dict:                     dict,
		normalizer:               normalizer,
		splitter:                 splitter,
		includeLowercaseOriginal: cfg.includeLowercaseOriginal,
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

		// Add lowercase original (preserves umlauts) if enabled
		if t.includeLowercaseOriginal {
			original := t.normalizer.LowercaseOnly(raw.Text)
			if _, exists := resultSet[original]; !exists {
				resultSet[original] = struct{}{}
				results = append(results, original)
			}
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

// LowercaseOriginalEnabled returns true if lowercase original output is enabled.
func (t *Tokenizer) LowercaseOriginalEnabled() bool {
	return t.includeLowercaseOriginal
}
