package tokenizer

// Config holds all tokenizer configuration. All fields must be explicitly set.
type Config struct {
	Cache             bool
	LowercaseOriginal bool
	Normalizers       NormalizerConfig
}

// NormalizerConfig specifies which normalization steps to apply.
// Each step must be explicitly enabled or disabled.
type NormalizerConfig struct {
	NFKDDecompose        bool
	RemoveControlChars   bool
	Lowercase            bool
	NormalizeQuotes      bool
	ExpandLigatures      bool
	ConvertEszett        bool
	RemoveCombiningMarks bool
	StemGerman           bool
}

// DefaultConfig returns the standard configuration with all features enabled.
func DefaultConfig() Config {
	return Config{
		Cache:             true,
		LowercaseOriginal: true,
		Normalizers:       DefaultNormalizerConfig(),
	}
}

// DefaultNormalizerConfig returns all normalizers enabled.
func DefaultNormalizerConfig() NormalizerConfig {
	return NormalizerConfig{
		NFKDDecompose:        true,
		RemoveControlChars:   true,
		Lowercase:            true,
		NormalizeQuotes:      true,
		ExpandLigatures:      true,
		ConvertEszett:        true,
		RemoveCombiningMarks: true,
		StemGerman:           true,
	}
}

// buildNormalizer creates a Normalizer from the config.
func (nc NormalizerConfig) buildNormalizer() *Normalizer {
	var steps []NormalizerFunc

	if nc.NFKDDecompose {
		steps = append(steps, NFKDDecompose)
	}
	if nc.RemoveControlChars {
		steps = append(steps, RemoveControlChars)
	}
	if nc.Lowercase {
		steps = append(steps, Lowercase)
	}
	if nc.NormalizeQuotes {
		steps = append(steps, NormalizeQuotes)
	}
	if nc.ExpandLigatures {
		steps = append(steps, ExpandLigatures)
	}
	if nc.ConvertEszett {
		steps = append(steps, ConvertEszett)
	}
	if nc.RemoveCombiningMarks {
		steps = append(steps, RemoveCombiningMarks)
	}
	if nc.StemGerman {
		steps = append(steps, StemGerman)
	}

	return NewNormalizerWithSteps(steps...)
}

// Tokenizer is the main German tokenizer.
type Tokenizer struct {
	dict                     *Dictionary
	normalizer               *Normalizer
	splitter                 *CompoundSplitter
	includeLowercaseOriginal bool
}

// NewTokenizer creates a tokenizer with explicit configuration.
//
// Example usage:
//
//	tok, _ := NewTokenizer(dictPath, Config{
//	    Cache:             true,
//	    LowercaseOriginal: true,
//	    Normalizers: NormalizerConfig{
//	        NFKDDecompose:        true,
//	        RemoveControlChars:   true,
//	        Lowercase:            true,
//	        NormalizeQuotes:      true,
//	        ExpandLigatures:      true,
//	        ConvertEszett:        true,
//	        RemoveCombiningMarks: true,
//	        StemGerman:           true,
//	    },
//	})
//
//	// Or use defaults:
//	tok, _ := NewTokenizer(dictPath, DefaultConfig())
func NewTokenizer(dictPath string, cfg Config) (*Tokenizer, error) {
	dict, err := NewDictionary(dictPath)
	if err != nil {
		return nil, err
	}

	// Build normalizer from config
	normalizer := cfg.Normalizers.buildNormalizer()

	// Build compound splitter
	var splitter *CompoundSplitter
	if cfg.Cache {
		splitter = NewCompoundSplitter(dict)
	} else {
		splitter = NewCompoundSplitterNoCache(dict)
	}

	return &Tokenizer{
		dict:                     dict,
		normalizer:               normalizer,
		splitter:                 splitter,
		includeLowercaseOriginal: cfg.LowercaseOriginal,
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
