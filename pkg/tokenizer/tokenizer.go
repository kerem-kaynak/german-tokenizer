package tokenizer

// Config holds all tokenizer configuration. All fields must be explicitly set.
type Config struct {
	Cache       bool
	Normalizers NormalizerConfig
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
	dict       *Dictionary
	normalizer *Normalizer
	splitter   *CompoundSplitter
}

// NewTokenizer creates a tokenizer with explicit configuration.
//
// Example usage:
//
//	tok, _ := NewTokenizer(dictPath, Config{
//	    Cache: true,
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
		dict:       dict,
		normalizer: normalizer,
		splitter:   splitter,
	}, nil
}

// WordTokens is the per-word output of Tokenize. Whole is the lowercase form of
// the input word with umlauts preserved (via LowercaseOnly). Parts is the
// normalized compound segments in splitter order.
type WordTokens struct {
	Whole string   `json:"whole"`
	Parts []string `json:"parts"`
}

// Tokenize returns one WordTokens entry per detected word, preserving per-word
// boundaries. No cross-word deduplication.
func (t *Tokenizer) Tokenize(text string) []WordTokens {
	rawTokens := SplitWords(text)
	var results []WordTokens

	for _, raw := range rawTokens {
		if raw.Type != TokenWord {
			continue
		}

		segments := t.splitter.Split(raw.Text)
		parts := make([]string, 0, len(segments))
		for _, seg := range segments {
			parts = append(parts, t.normalizer.Normalize(seg))
		}

		results = append(results, WordTokens{
			Whole: t.normalizer.LowercaseOnly(raw.Text),
			Parts: parts,
		})
	}

	return results
}

// AddWord adds a word to the dictionary.
// Rebuilds FST immediately and persists to disk.
func (t *Tokenizer) AddWord(word string) error {
	return t.dict.AddWord(word)
}

// RemoveWord removes a word from the dictionary.
// Rebuilds FST immediately and persists to disk.
func (t *Tokenizer) RemoveWord(word string) error {
	return t.dict.RemoveWord(word)
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
