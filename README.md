# German Tokenizer

A high-performance German text tokenizer for search and NLP applications. Specializes in compound word decomposition, a critical feature for German text processing.

[![Go Reference](https://pkg.go.dev/badge/github.com/kerem-kaynak/german-tokenizer.svg)](https://pkg.go.dev/github.com/kerem-kaynak/german-tokenizer)

## Features

- **Compound word decomposition**: Splits German compounds into constituent parts (e.g., "Brandschutzkonzept" → ["brand", "schutz", "konzept"])
- **FST-based dictionary lookups**: Uses finite state transducers for O(n) dictionary lookups where n is word length
- **Configurable normalization pipeline**: NFKD decomposition, lowercase, ß→ss conversion, German stemming, and more
- **LRU cache**: 100k entry cache for compound splits (~10MB memory)
- **Dual output**: Emits both original tokens (with umlauts preserved) and normalized/stemmed tokens
- **Runtime dictionary updates**: Add or remove words without restarting

## Installation

```bash
go get github.com/kerem-kaynak/german-tokenizer
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/kerem-kaynak/german-tokenizer/pkg/tokenizer"
)

func main() {
    tok, err := tokenizer.NewTokenizer("path/to/dictionary.txt", tokenizer.Config{
        Cache:             true,
        LowercaseOriginal: true,
        Normalizers: tokenizer.NormalizerConfig{
            NFKDDecompose:        true,
            RemoveControlChars:   true,
            Lowercase:            true,
            NormalizeQuotes:      true,
            ExpandLigatures:      true,
            ConvertEszett:        true,
            RemoveCombiningMarks: true,
            StemGerman:           true,
        },
    })
    if err != nil {
        panic(err)
    }
    defer tok.Close()

    tokens := tok.Tokenize("Brandschutzkonzept")
    fmt.Println(tokens)
    // Output: [brandschutzkonzept brand schutz konzept]
}
```

## Dictionary

The tokenizer requires a dictionary of German compound word components. A dictionary with ~15,000 words is included at `dictionaries/german_compound_word_components.txt`.

**Dictionary source**: The included dictionary is derived from [uschindler/german-decompounder](https://github.com/uschindler/german-decompounder), which was created based on [Björn Jacke's igerman98](https://www.j3e.de/ispell/igerman98/) dictionary. The dictionary contains component parts commonly used to form German compound words (not the compounds themselves).

You can also use your own dictionary - one word per line, lowercase.

### Runtime Dictionary Updates

Words can be added or removed at runtime. Changes are immediately persisted to disk and the FST is rebuilt:

```go
// Add a word - FST is rebuilt immediately
err := tok.AddWord("neueswort")

// Remove a word - FST is rebuilt immediately
err := tok.RemoveWord("alteswort")
```

## Configuration

All configuration is explicit. No hidden defaults.

### Config struct

```go
type Config struct {
    Cache             bool             // Enable LRU cache for compound splits
    LowercaseOriginal bool             // Include lowercase original in output
    Normalizers       NormalizerConfig // Which normalizers to apply
}

type NormalizerConfig struct {
    NFKDDecompose        bool // Unicode NFKD decomposition
    RemoveControlChars   bool // Remove control characters
    Lowercase            bool // Convert to lowercase
    NormalizeQuotes      bool // Normalize „" » « to ASCII quotes
    ExpandLigatures      bool // æ→ae, œ→oe
    ConvertEszett        bool // ß→ss
    RemoveCombiningMarks bool // Remove combining diacritics (ä→a after NFKD)
    StemGerman           bool // Apply Snowball German stemmer
}
```

### Example configurations

**Full normalization (search indexing)**:
```go
tokenizer.Config{
    Cache:             true,
    LowercaseOriginal: true,
    Normalizers: tokenizer.NormalizerConfig{
        NFKDDecompose:        true,
        RemoveControlChars:   true,
        Lowercase:            true,
        NormalizeQuotes:      true,
        ExpandLigatures:      true,
        ConvertEszett:        true,
        RemoveCombiningMarks: true,
        StemGerman:           true,
    },
}
```

**Preserve umlauts (exact matching)**:
```go
tokenizer.Config{
    Cache:             true,
    LowercaseOriginal: true,
    Normalizers: tokenizer.NormalizerConfig{
        NFKDDecompose:        false,
        RemoveControlChars:   true,
        Lowercase:            true,
        NormalizeQuotes:      true,
        ExpandLigatures:      false,
        ConvertEszett:        false,
        RemoveCombiningMarks: false,
        StemGerman:           false,
    },
}
```

**No cache (memory constrained)**:
```go
tokenizer.Config{
    Cache:             false,  // Disable cache
    LowercaseOriginal: true,
    Normalizers:       // ...
}
```

## How It Works

### 1. Word Splitting

Input text is split into words using Unicode letter/number detection:

```
"Der Brandschutzkonzept" → ["Der", "Brandschutzkonzept"]
```

### 2. Compound Decomposition

Each word is decomposed using a greedy left-to-right algorithm:

```
"Brandschutzkonzept"
  ├─ Try "Brandschutzkonzept" → not in dictionary
  ├─ Try "Brandschutzkonzep" → not in dictionary
  ├─ ...
  ├─ Try "Brand" → IN DICTIONARY ✓
  │   └─ Recurse on "schutzkonzept"
  │       ├─ Try "Schutz" → IN DICTIONARY ✓
  │       │   └─ Recurse on "konzept"
  │       │       └─ Try "Konzept" → IN DICTIONARY ✓
  └─ Result: ["brand", "schutz", "konzept"]
```

Dictionary lookups use:
1. Direct FST lookup
2. Umlaut normalization (ä→a, ö→o, ü→u, ß→ss)
3. Suffix stripping for inflected forms

### 3. Token Output

For each word, the tokenizer outputs:
1. **Lowercase original** (if enabled): Preserves umlauts for exact matching
2. **Normalized segments**: Each compound part, normalized and stemmed

```
Input: "Wärmedämmung"
Output: ["wärmedämmung", "warm", "dammung"]
         ↑ original      ↑ normalized segments
```

Tokens are deduplicated using set semantics.

### 4. Normalization Pipeline

Each segment passes through the configured normalizers in order:

```
"Größe"
  → NFKD: "Gro\u0308ße" (ö decomposed to o + combining umlaut)
  → Lowercase: "gro\u0308ße"
  → ConvertEszett: "gro\u0308sse"
  → RemoveCombiningMarks: "grosse"
  → StemGerman: "gross"
```

### 5. FST Dictionary

The dictionary uses a Finite State Transducer (FST) via [blevesearch/vellum](https://github.com/blevesearch/vellum):

- **Memory efficient**: FST is smaller than a hash map for large dictionaries
- **Fast lookups**: O(n) where n is the word length, not dictionary size
- **Prefix queries**: Can efficiently find all words with a given prefix

### 6. LRU Cache

Compound splits are cached using an LRU cache (100k entries, ~10MB):

- **Cache hit**: Return cached result immediately
- **Cache miss**: Compute split, store in cache
- **Eviction**: Least recently used entries are evicted when cache is full

## CLI Tools

### Tokenize

```bash
# Build all binaries
make build

# Tokenize a single input
make run TEXT="Brandschutzkonzept"
# Output: ["brandschutzkonzept","brand","schutz","konzept"]

# Interactive mode
make demo
> Wärmedämmung
  ["wärmedämmung","warm","dammung"]
```

### Dictionary Management

```bash
# Show dictionary statistics
make dict-stats

# Check if a word exists
make dict-contains WORD=haus

# Add a word
make dict-add WORD=neueswort

# Remove a word
make dict-remove WORD=alteswort
```

### Throughput Benchmarking

```bash
make throughput
```

## Performance

Benchmarks on Apple M4 Pro:

| Operation | Throughput | Latency |
|-----------|------------|---------|
| Single word tokenization | 720k ops/sec | 1.4μs |
| Long compound tokenization | 530k ops/sec | 1.9μs |
| Sentence (10 words) | 200k ops/sec | 4.9μs |
| Dictionary lookup | 273M ops/sec | 4ns |
| Normalizer (full pipeline) | 1.4M ops/sec | 736ns |
| Cache hit | 54M ops/sec | 19ns |

Run benchmarks on your hardware:

```bash
# Go micro-benchmarks (per-function timing)
make bench

# Throughput test (words/sec with colored output)
make throughput
```

## API Reference

### Tokenizer

```go
// Create tokenizer
tok, err := tokenizer.NewTokenizer(dictPath string, cfg Config) (*Tokenizer, error)

// Tokenize text
tokens := tok.Tokenize(text string) []string

// Dictionary management (FST rebuilt immediately, persisted to disk)
err := tok.AddWord(word string) error
err := tok.RemoveWord(word string) error

// Cache management
tok.CacheSize() int
tok.ClearCache()
tok.CacheEnabled() bool

// Info
tok.DictionaryWordCount() int
tok.LowercaseOriginalEnabled() bool

// Cleanup
tok.Close() error
```

### Normalizer (standalone)

```go
// Create with all normalizers
norm := tokenizer.NewNormalizer()

// Create with specific normalizers
norm := tokenizer.NewNormalizerWithSteps(
    tokenizer.NFKDDecompose,
    tokenizer.Lowercase,
    tokenizer.StemGerman,
)

// Normalize text
result := norm.Normalize(text string) string

// Lowercase only (preserves umlauts)
result := norm.LowercaseOnly(text string) string
```

### Individual normalizer functions

All normalizer functions are exported and can be used standalone:

```go
tokenizer.NFKDDecompose(s string) string
tokenizer.RemoveControlChars(s string) string
tokenizer.Lowercase(s string) string
tokenizer.NormalizeQuotes(s string) string
tokenizer.ExpandLigatures(s string) string
tokenizer.ConvertEszett(s string) string
tokenizer.RemoveCombiningMarks(s string) string
tokenizer.StemGerman(s string) string
```

## Development

```bash
# Run tests
make test

# Run micro-benchmarks
make bench

# Run throughput test
make throughput

# Build binaries
make build

# Format code
make fmt

# Lint
make lint
```

## License

MIT License - see [LICENSE](LICENSE)
