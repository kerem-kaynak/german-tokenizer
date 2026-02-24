# German Tokenizer - Go Implementation Plan

## Context

We need a standalone German tokenizer that:

- Decomposes compound words (Dampfschifffahrtskapitän → Dampf + schifffahrts + kapitän)
- Emits both original (umlauts preserved) AND normalized/stemmed segments
- Uses set semantics to deduplicate identical tokens
- Replicates charabia's full normalization pipeline
- Will eventually become a web service, but starts as a Go library with CLI testing

---

## Pipeline Overview

```
Input: "Der Wärmedämmung fährt"
              │
              ▼
┌────────────────────────────────────────────────────────────────────┐
│ STEP 1: WORD SPLITTING                                             │
│                                                                    │
│ Split on whitespace and punctuation, preserving separators         │
│ "Der Wärmedämmung fährt" → ["Der", " ", "Wärmedämmung", " ", "fährt"]
└────────────────────────────────────────────────────────────────────┘
              │
              ▼
     ┌────────┴────────┐
     │  FOR EACH WORD  │  (skip separators)
     └────────┬────────┘
              │
              ▼
┌────────────────────────────────────────────────────────────────────┐
│ STEP 2: COMPOUND DECOMPOSITION                                     │
│                                                                    │
│ Try to split word using dictionary (greedy left-to-right)          │
│                                                                    │
│ "Wärmedämmung" → try splits → ["Wärme", "dämmung"]                 │
│ "fährt" → can't split → ["fährt"]                                  │
│ "Der" → can't split → ["Der"]                                      │
└────────────────────────────────────────────────────────────────────┘
              │
              ▼
┌────────────────────────────────────────────────────────────────────┐
│ STEP 3: SEGMENT VALIDATION                                         │
│                                                                    │
│ Check each segment: must be ≥2 chars AND in dictionary             │
│ If ANY segment invalid → fallback to [original_word]               │
│                                                                    │
│ ["Wärme", "dämmung"] → both valid → keep                           │
│ ["fähr", "t"] → "t" invalid → fallback to ["fährt"]                │
└────────────────────────────────────────────────────────────────────┘
              │
              ▼
┌────────────────────────────────────────────────────────────────────┐
│ STEP 4: TOKEN EMISSION (Set-based)                                 │
│                                                                    │
│ For word "Wärmedämmung" with segments ["Wärme", "dämmung"]:        │
│                                                                    │
│ result_set = {}                                                    │
│ result_set.add(lowercase("Wärmedämmung"))     → "wärmedämmung"     │
│ result_set.add(normalize_stem("Wärme"))       → "warm"             │
│ result_set.add(normalize_stem("dämmung"))     → "dammung"          │
│                                                                    │
│ Output: ["wärmedämmung", "warm", "dammung"]                        │
│                                                                    │
│ For word "Haus" with segments ["Haus"]:                            │
│ result_set.add(lowercase("Haus"))             → "haus"             │
│ result_set.add(normalize_stem("Haus"))        → "haus"  (duplicate)│
│                                                                    │
│ Output: ["haus"]  (deduplicated)                                   │
└────────────────────────────────────────────────────────────────────┘
```

---

## Two Separate Normalizations

### A. Compound Decomposition Normalization (for dictionary lookup ONLY)

Used when trying to find words in dictionary during compound splitting:

```
ä→a, ö→o, ü→u, ß→ss
```

This is a simple character replacement used ONLY to match dictionary entries.
Example: "dämmung" → lookup "dammung" in dictionary

### B. Token Normalization Pipeline (for final output)

The `normalize_stem()` function applies these transformations IN ORDER:

| Step | Operation              | Example                | Go Implementation                |
| ---- | ---------------------- | ---------------------- | -------------------------------- |
| 1    | NFKD Decomposition     | "ä" → "a\u0308"        | `golang.org/x/text/unicode/norm` |
| 2    | Control Char Removal   | "\u0000a" → "a"        | Filter Unicode control category  |
| 3    | Lowercase              | "ABC" → "abc"          | `strings.ToLower()`              |
| 4    | Quote Normalization    | „text" → "text"        | Character mapping                |
| 5    | Ligature Expansion     | "œ" → "oe", "æ" → "ae" | Character mapping                |
| 6    | Combining Mark Removal | "a\u0308" → "a"        | Filter Unicode Mn category       |
| 7    | Stemming               | "laufen" → "lauf"      | Snowball stemmer library         |

**Important**:

- Steps 1 + 6 together convert umlauts: ä → a\u0308 → a
- **NO explicit ß→ss** in this pipeline (charabia doesn't have it)
- ß handling is left to the Snowball stemmer

---

## Project Structure

```
german-tokenizer/
├── go.mod
├── go.sum
├── README.md
├── cmd/
│   ├── tokenize/
│   │   └── main.go              # CLI for tokenization
│   └── dictmgr/
│       └── main.go              # CLI for dictionary management
├── pkg/
│   └── tokenizer/
│       ├── tokenizer.go         # Main Tokenizer struct and API
│       ├── tokenizer_test.go    # Unit tests
│       ├── splitter.go          # Word splitting logic
│       ├── compound.go          # Compound decomposition
│       ├── normalizer.go        # Full normalization pipeline
│       └── dictionary.go        # FST-based dictionary with add/remove
├── dictionaries/
│   ├── german_words.txt         # Word list (source, from charabia)
│   └── german_words.fst         # Compiled FST (auto-generated)
└── testdata/
    └── test_cases.txt           # Test inputs/expected outputs
```

---

## Detailed Implementation Steps

### Step 1: Project Setup

**1.1 Initialize Go module**

```bash
mkdir german-tokenizer && cd german-tokenizer
go mod init github.com/yourorg/german-tokenizer
```

**1.2 Add dependencies**

```bash
go get golang.org/x/text/unicode/norm    # NFKD normalization
go get github.com/kljensen/snowball      # Snowball stemmer
go get github.com/blevesearch/vellum     # FST for lightning-fast lookups
```

**1.3 Create directory structure**

```bash
mkdir -p cmd/tokenize pkg/tokenizer dictionaries testdata
```

**1.4 Copy dictionary**

Copy `charabia/dictionaries/txt/german/words.txt` to `dictionaries/german_words.txt`

This file contains ~300,000 German words, one per line, lowercase.

---

### Step 2: Dictionary Module (`pkg/tokenizer/dictionary.go`)

**Purpose**: FST-based dictionary for lightning-fast O(1) lookups (like charabia).

The FST (Finite State Transducer) provides:

- Extremely fast key existence checks
- Memory-efficient storage (compressed trie structure)
- Faster than HashMap for large dictionaries

```go
package tokenizer

import (
    "bufio"
    "os"
    "sort"
    "strings"

    "github.com/blevesearch/vellum"
)

// Dictionary holds German words in an FST for fast lookups
type Dictionary struct {
    fst      *vellum.FST       // Compiled FST for fast lookups
    words    map[string]struct{} // In-memory set for modifications
    fstPath  string            // Path to FST file
    txtPath  string            // Path to source text file
    dirty    bool              // True if words changed since last FST build
}

// NewDictionary loads an FST dictionary from file
// If FST doesn't exist, builds it from the text file
func NewDictionary(txtPath string) (*Dictionary, error) {
    fstPath := strings.TrimSuffix(txtPath, ".txt") + ".fst"

    d := &Dictionary{
        words:   make(map[string]struct{}, 300000),
        fstPath: fstPath,
        txtPath: txtPath,
    }

    // Load words from text file into memory
    if err := d.loadTextFile(); err != nil {
        return nil, err
    }

    // Try to load existing FST, or build new one
    if err := d.loadOrBuildFST(); err != nil {
        return nil, err
    }

    return d, nil
}

// loadTextFile reads words from the source text file
func (d *Dictionary) loadTextFile() error {
    file, err := os.Open(d.txtPath)
    if err != nil {
        return err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        word := strings.TrimSpace(scanner.Text())
        if word != "" {
            d.words[strings.ToLower(word)] = struct{}{}
        }
    }
    return scanner.Err()
}

// loadOrBuildFST loads existing FST or builds a new one
func (d *Dictionary) loadOrBuildFST() error {
    // Try to load existing FST
    if fst, err := vellum.Open(d.fstPath); err == nil {
        d.fst = fst
        return nil
    }

    // FST doesn't exist or is invalid - build it
    return d.RebuildFST()
}

// Contains checks if a word exists in the FST (case-insensitive)
// This is the hot path - optimized for speed
func (d *Dictionary) Contains(word string) bool {
    lower := strings.ToLower(word)

    // If FST is stale, check in-memory map
    if d.dirty {
        _, ok := d.words[lower]
        return ok
    }

    // Fast FST lookup
    _, exists, _ := d.fst.Get([]byte(lower))
    return exists
}

// AddWord adds a word to the dictionary (both uppercase and lowercase)
// Marks dictionary as dirty - call RebuildFST() to persist
func (d *Dictionary) AddWord(word string) {
    lower := strings.ToLower(word)
    upper := strings.ToUpper(string([]rune(word)[0])) + lower[len(string([]rune(lower)[0])):]

    d.words[lower] = struct{}{}
    d.words[upper] = struct{}{}
    d.dirty = true
}

// RemoveWord removes a word from the dictionary (both cases)
// Marks dictionary as dirty - call RebuildFST() to persist
func (d *Dictionary) RemoveWord(word string) {
    lower := strings.ToLower(word)
    upper := strings.ToUpper(string([]rune(word)[0])) + lower[len(string([]rune(lower)[0])):]

    delete(d.words, lower)
    delete(d.words, upper)
    d.dirty = true
}

// RebuildFST rebuilds the FST from the current word set and saves to disk
func (d *Dictionary) RebuildFST() error {
    // Close existing FST if any
    if d.fst != nil {
        d.fst.Close()
    }

    // Sort words (FST requires sorted input)
    sortedWords := make([]string, 0, len(d.words))
    for word := range d.words {
        sortedWords = append(sortedWords, word)
    }
    sort.Strings(sortedWords)

    // Build FST
    builder, err := vellum.New(d.fstPath, nil)
    if err != nil {
        return err
    }

    for _, word := range sortedWords {
        if err := builder.Insert([]byte(word), 0); err != nil {
            builder.Close()
            return err
        }
    }

    if err := builder.Close(); err != nil {
        return err
    }

    // Reopen the FST for reading
    fst, err := vellum.Open(d.fstPath)
    if err != nil {
        return err
    }
    d.fst = fst
    d.dirty = false

    // Also update the source text file
    return d.saveTextFile()
}

// saveTextFile writes the current word set back to the text file
func (d *Dictionary) saveTextFile() error {
    // Sort for consistent output
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

// Close releases FST resources
func (d *Dictionary) Close() error {
    if d.fst != nil {
        return d.fst.Close()
    }
    return nil
}

// IsDirty returns true if dictionary has unsaved changes
func (d *Dictionary) IsDirty() bool {
    return d.dirty
}

// WordCount returns the number of words in the dictionary
func (d *Dictionary) WordCount() int {
    return len(d.words)
}
```

**Key design decisions:**

- **Dual storage**: In-memory map for modifications, FST for fast lookups
- **Lazy rebuild**: Changes are tracked with `dirty` flag; FST rebuilt on demand
- **Case handling**: `AddWord`/`RemoveWord` handle both upper and lowercase automatically
- **Persistence**: `RebuildFST()` saves both FST and text file

**Test cases for dictionary:**

```go
func TestDictionary(t *testing.T) {
    dict, _ := NewDictionary("dictionaries/german_words.txt")
    defer dict.Close()

    // Fast FST lookup
    assert(dict.Contains("Haus") == true)    // case insensitive
    assert(dict.Contains("haus") == true)
    assert(dict.Contains("xyz123") == false)

    // Add word (adds both cases)
    dict.AddWord("neueswort")
    assert(dict.IsDirty() == true)
    assert(dict.Contains("neueswort") == true)
    assert(dict.Contains("Neueswort") == true)

    // Rebuild FST
    dict.RebuildFST()
    assert(dict.IsDirty() == false)
    assert(dict.Contains("neueswort") == true)  // still exists after rebuild

    // Remove word (removes both cases)
    dict.RemoveWord("neueswort")
    assert(dict.Contains("neueswort") == false)
    assert(dict.Contains("Neueswort") == false)
}
```

---

### Step 3: Normalizer Module (`pkg/tokenizer/normalizer.go`)

**Purpose**: Apply the full normalization pipeline to a string.

```go
package tokenizer

import (
    "strings"
    "unicode"

    "golang.org/x/text/unicode/norm"
    "github.com/kljensen/snowball"
)

// Normalizer handles text normalization
type Normalizer struct{}

// NewNormalizer creates a new normalizer
func NewNormalizer() *Normalizer {
    return &Normalizer{}
}

// Normalize applies the full pipeline: NFKD → control chars → lowercase →
// quotes → ligatures → combining marks → stem
// NOTE: No explicit ß→ss - matches charabia behavior
func (n *Normalizer) Normalize(s string) string {
    s = n.nfkdDecompose(s)
    s = n.removeControlChars(s)
    s = strings.ToLower(s)
    s = n.normalizeQuotes(s)
    s = n.expandLigatures(s)
    s = n.removeCombiningMarks(s)
    s = n.stem(s)
    return s
}

// LowercaseOnly just lowercases (for original tokens)
func (n *Normalizer) LowercaseOnly(s string) string {
    return strings.ToLower(s)
}

// nfkdDecompose applies Unicode NFKD normalization
// This decomposes ä → a + combining_umlaut, ﬁ → fi, etc.
func (n *Normalizer) nfkdDecompose(s string) string {
    return norm.NFKD.String(s)
}

// removeControlChars removes Unicode control characters (C0, C1)
func (n *Normalizer) removeControlChars(s string) string {
    var result strings.Builder
    for _, r := range s {
        if !unicode.IsControl(r) {
            result.WriteRune(r)
        }
    }
    return result.String()
}

// normalizeQuotes converts fancy quotes to ASCII
var quoteReplacements = map[rune]rune{
    '„': '"',  // German opening quote
    '"': '"',  // German closing quote
    '«': '"',  // French quote
    '»': '"',  // French quote
    ''': '\'', // Single quote
    ''': '\'', // Single quote
    '‚': '\'', // Single low quote
    '‹': '\'', // Single angle quote
    '›': '\'', // Single angle quote
}

func (n *Normalizer) normalizeQuotes(s string) string {
    var result strings.Builder
    for _, r := range s {
        if replacement, ok := quoteReplacements[r]; ok {
            result.WriteRune(replacement)
        } else {
            result.WriteRune(r)
        }
    }
    return result.String()
}

// expandLigatures expands æ→ae, œ→oe
func (n *Normalizer) expandLigatures(s string) string {
    s = strings.ReplaceAll(s, "æ", "ae")
    s = strings.ReplaceAll(s, "Æ", "AE")
    s = strings.ReplaceAll(s, "œ", "oe")
    s = strings.ReplaceAll(s, "Œ", "OE")
    return s
}

// removeCombiningMarks removes Unicode combining characters (category Mn)
// This removes the umlaut dots after NFKD decomposition
func (n *Normalizer) removeCombiningMarks(s string) string {
    var result strings.Builder
    for _, r := range s {
        if !unicode.Is(unicode.Mn, r) { // Mn = Mark, Nonspacing
            result.WriteRune(r)
        }
    }
    return result.String()
}

// stem applies German Snowball stemmer
func (n *Normalizer) stem(s string) string {
    stemmed, err := snowball.Stem(s, "german", true)
    if err != nil {
        return s // fallback to original on error
    }
    return stemmed
}
```

**Test cases for normalizer:**

```go
func TestNormalizer(t *testing.T) {
    n := NewNormalizer()

    // Full pipeline tests
    assert(n.Normalize("Wärme") == "warm")       // ä→a, stem: wärme→warm
    assert(n.Normalize("Häuser") == "haus")      // ä→a, stem: häuser→haus
    assert(n.Normalize("fährt") == "fahrt")      // ä→a, stem may not change
    assert(n.Normalize("über") == "uber")        // ü→u
    assert(n.Normalize("Größe") == "gross")      // ö→o, ß→ss (NFKD handles ß)

    // Lowercase only (for originals)
    assert(n.LowercaseOnly("Wärme") == "wärme")  // preserves umlaut
    assert(n.LowercaseOnly("ÜBER") == "über")    // preserves umlaut

    // Quote normalization
    assert(n.Normalize("„Wort"") == "wort")

    // Ligatures (rare but possible)
    assert(n.Normalize("Œuvre") == "oeuvr")      // œ→oe + stem
}
```

**Important edge case - ß handling:**
NFKD does NOT decompose ß. It stays as ß. The stemmer handles ß→ss conversion.
If stemmer doesn't handle it, add explicit replacement:

```go
s = strings.ReplaceAll(s, "ß", "ss")
```

---

### Step 4: Word Splitter Module (`pkg/tokenizer/splitter.go`)

**Purpose**: Split input text into words and separators.

```go
package tokenizer

import (
    "unicode"
)

// Token types
const (
    TokenWord      = "word"
    TokenSeparator = "separator"
)

// RawToken represents a token before normalization
type RawToken struct {
    Text  string
    Type  string
    Start int
    End   int
}

// SplitWords splits text into words and separators
// Separators are: whitespace, punctuation
func SplitWords(text string) []RawToken {
    var tokens []RawToken
    var currentToken strings.Builder
    var currentType string
    start := 0

    runes := []rune(text)
    for i, r := range runes {
        isWordChar := unicode.IsLetter(r) || unicode.IsNumber(r)

        tokenType := TokenSeparator
        if isWordChar {
            tokenType = TokenWord
        }

        // Type changed or first char
        if tokenType != currentType {
            // Flush previous token
            if currentToken.Len() > 0 {
                tokens = append(tokens, RawToken{
                    Text:  currentToken.String(),
                    Type:  currentType,
                    Start: start,
                    End:   i,
                })
            }
            currentToken.Reset()
            currentType = tokenType
            start = i
        }

        currentToken.WriteRune(r)
    }

    // Flush final token
    if currentToken.Len() > 0 {
        tokens = append(tokens, RawToken{
            Text:  currentToken.String(),
            Type:  currentType,
            Start: start,
            End:   len(runes),
        })
    }

    return tokens
}
```

**Test cases for splitter:**

```go
func TestSplitWords(t *testing.T) {
    tokens := SplitWords("Der fährt!")

    // Expected: ["Der", " ", "fährt", "!"]
    assert(len(tokens) == 4)
    assert(tokens[0].Text == "Der" && tokens[0].Type == TokenWord)
    assert(tokens[1].Text == " " && tokens[1].Type == TokenSeparator)
    assert(tokens[2].Text == "fährt" && tokens[2].Type == TokenWord)
    assert(tokens[3].Text == "!" && tokens[3].Type == TokenSeparator)
}
```

---

### Step 5: Compound Decomposition Module (`pkg/tokenizer/compound.go`)

**Purpose**: Split compound words using dictionary lookups.

```go
package tokenizer

import (
    "strings"
)

// Common German suffixes for validation fallback
var germanSuffixes = []string{
    "ungen", "schaft", "heiten", "keiten",
    "ung", "heit", "keit", "tion", "isch", "lich", "chen", "lein",
    "haft", "bar", "sam", "tum", "ig", "er", "en", "em", "es",
    "st", "nd", "te", "el", "le", "se", "ße", "ze",
    "e", "s", "n", "t",
}

// CompoundSplitter handles German compound word decomposition
type CompoundSplitter struct {
    dict *Dictionary
}

// NewCompoundSplitter creates a new splitter with dictionary
func NewCompoundSplitter(dict *Dictionary) *CompoundSplitter {
    return &CompoundSplitter{dict: dict}
}

// Split attempts to decompose a compound word
// Returns segments if successful, or [word] if can't split
func (c *CompoundSplitter) Split(word string) []string {
    // Try greedy left-to-right decomposition
    segments := c.greedySplit(strings.ToLower(word))

    // Validate all segments
    if c.allSegmentsValid(segments) && len(segments) > 1 {
        return segments
    }

    // Fallback: return original word as single segment
    return []string{word}
}

// greedySplit tries to split word from left to right
func (c *CompoundSplitter) greedySplit(word string) []string {
    var segments []string
    remaining := word

    for len(remaining) > 0 {
        found := false
        runes := []rune(remaining)

        // Try longest match first (minimum 2 chars)
        for length := len(runes); length >= 2; length-- {
            prefix := string(runes[:length])

            if c.isValidWord(prefix) {
                segments = append(segments, prefix)
                remaining = string(runes[length:])
                found = true
                break
            }
        }

        if !found {
            // Can't split further - return original
            return []string{word}
        }
    }

    return segments
}

// isValidWord checks if word exists in dictionary (with suffix fallback)
func (c *CompoundSplitter) isValidWord(word string) bool {
    lower := strings.ToLower(word)

    // Direct lookup
    if c.dict.Contains(lower) {
        return true
    }

    // Try with umlaut normalization
    normalized := normalizeUmlauts(lower)
    if normalized != lower && c.dict.Contains(normalized) {
        return true
    }

    // Try suffix stripping
    for _, suffix := range germanSuffixes {
        if strings.HasSuffix(lower, suffix) {
            stem := strings.TrimSuffix(lower, suffix)
            if len([]rune(stem)) >= 2 {
                if c.dict.Contains(stem) {
                    return true
                }
                // Also try normalized stem
                if c.dict.Contains(normalizeUmlauts(stem)) {
                    return true
                }
            }
        }
    }

    return false
}

// allSegmentsValid checks if all segments pass validation
func (c *CompoundSplitter) allSegmentsValid(segments []string) bool {
    for _, seg := range segments {
        if len([]rune(seg)) < 2 {
            return false
        }
        if !c.isValidWord(seg) {
            return false
        }
    }
    return true
}

// normalizeUmlauts converts ä→a, ö→o, ü→u, ß→ss
// Used ONLY for dictionary lookup during compound decomposition
// This is SEPARATE from the token normalization pipeline
func normalizeUmlauts(s string) string {
    replacer := strings.NewReplacer(
        "ä", "a", "Ä", "a",
        "ö", "o", "Ö", "o",
        "ü", "u", "Ü", "u",
        "ß", "ss",
    )
    return replacer.Replace(s)
}
```

**Note on Fugenlaute (linking sounds):**
German compound words often have linking sounds like "s", "n", "en" between components
(e.g., "Dampfschiff**s**fahrts" has "s" after "Dampfschiff").

The dictionary should contain word forms WITH Fugenlaute already (e.g., "schifffahrts",
"verwaltungs", "arbeitnehmer"). The greedy splitter will find these forms directly
without needing to strip Fugenlaute explicitly. This matches charabia's approach.

**Test cases for compound splitter:**

```go
func TestCompoundSplitter(t *testing.T) {
    // Assumes dictionary contains: dampf, schiff, fahrts, kapitän, wärme, dämmung, etc.
    splitter := NewCompoundSplitter(dict)

    // Successful splits
    assert(splitter.Split("Dampfschifffahrtskapitän") ==
           []string{"dampf", "schifffahrts", "kapitän"})
    assert(splitter.Split("Wärmedämmung") == []string{"wärme", "dämmung"})

    // Can't split - return original
    assert(splitter.Split("fährt") == []string{"fährt"})
    assert(splitter.Split("Der") == []string{"Der"})

    // Invalid segment would result - fallback
    // If "t" is invalid: "fährt" → ["fähr", "t"] → invalid → ["fährt"]
    assert(splitter.Split("insgesamt") == []string{"insgesamt"})
}
```

---

### Step 6: Main Tokenizer Module (`pkg/tokenizer/tokenizer.go`)

**Purpose**: Orchestrate all components and provide the main API.

```go
package tokenizer

// Tokenizer is the main German tokenizer
type Tokenizer struct {
    dict       *Dictionary
    normalizer *Normalizer
    splitter   *CompoundSplitter
}

// NewTokenizer creates a tokenizer with the given dictionary path
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

// Tokenize processes input text and returns deduplicated tokens
func (t *Tokenizer) Tokenize(text string) []string {
    // Step 1: Split into words and separators
    rawTokens := SplitWords(text)

    // Result set for deduplication
    resultSet := make(map[string]struct{})
    var results []string

    for _, raw := range rawTokens {
        // Skip separators
        if raw.Type != TokenWord {
            continue
        }

        // Step 2: Compound decomposition
        segments := t.splitter.Split(raw.Text)

        // Step 3 & 4: Emit tokens to set
        // Always add lowercase original
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

// AddWord adds a word to the dictionary (both upper/lowercase)
// Note: Changes are not persisted until RebuildDictionary() is called
func (t *Tokenizer) AddWord(word string) {
    t.dict.AddWord(word)
}

// RemoveWord removes a word from the dictionary (both cases)
// Note: Changes are not persisted until RebuildDictionary() is called
func (t *Tokenizer) RemoveWord(word string) {
    t.dict.RemoveWord(word)
}

// RebuildDictionary rebuilds the FST and persists changes to disk
func (t *Tokenizer) RebuildDictionary() error {
    return t.dict.RebuildFST()
}

// Close releases resources (call when done with tokenizer)
func (t *Tokenizer) Close() error {
    return t.dict.Close()
}
```

**Test cases for tokenizer:**

```go
func TestTokenizer(t *testing.T) {
    tok, _ := NewTokenizer("dictionaries/german_words.txt")
    defer tok.Close()  // Important: release FST resources

    // Compound with umlauts
    assert(tok.Tokenize("Wärmedämmung") ==
           []string{"wärmedämmung", "warm", "dammung"})

    // Standalone with umlaut - original differs from normalized
    assert(tok.Tokenize("fährt") == []string{"fährt", "fahrt"})

    // Standalone without umlaut - deduplicates
    assert(tok.Tokenize("Haus") == []string{"haus"})

    // Multiple words
    assert(tok.Tokenize("Der fährt") == []string{"der", "fährt", "fahrt"})

    // Long compound
    result := tok.Tokenize("Dampfschifffahrtskapitän")
    assert(contains(result, "dampfschifffahrtskapitän")) // original
    assert(contains(result, "dampf"))                    // segment
    assert(contains(result, "schifffahrt"))              // segment (stemmed)
    assert(contains(result, "kapitan"))                  // segment (normalized)
}

func TestTokenizerDictionaryManagement(t *testing.T) {
    tok, _ := NewTokenizer("testdata/test_dict.txt")
    defer tok.Close()

    // Add a custom word
    tok.AddWord("testwort")
    assert(tok.dict.Contains("testwort") == true)
    assert(tok.dict.Contains("Testwort") == true)  // uppercase also added

    // Remove a word
    tok.RemoveWord("testwort")
    assert(tok.dict.Contains("testwort") == false)

    // Rebuild to persist changes
    tok.RebuildDictionary()
}
```

---

### Step 7: CLI Tool (`cmd/tokenize/main.go`)

**Purpose**: Test the tokenizer from command line.

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "strings"

    "github.com/yourorg/german-tokenizer/pkg/tokenizer"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: tokenize <dictionary_path> [text]")
        fmt.Println("       tokenize <dictionary_path>          (interactive mode)")
        os.Exit(1)
    }

    dictPath := os.Args[1]

    tok, err := tokenizer.NewTokenizer(dictPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error loading dictionary: %v\n", err)
        os.Exit(1)
    }

    // If text provided as argument, tokenize and exit
    if len(os.Args) > 2 {
        text := strings.Join(os.Args[2:], " ")
        tokens := tok.Tokenize(text)
        output, _ := json.Marshal(tokens)
        fmt.Println(string(output))
        return
    }

    // Interactive mode
    fmt.Println("German Tokenizer (interactive mode)")
    fmt.Println("Enter text to tokenize (Ctrl+D to exit):")
    fmt.Println()

    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        text := scanner.Text()
        if text == "" {
            continue
        }

        tokens := tok.Tokenize(text)
        output, _ := json.Marshal(tokens)
        fmt.Printf("→ %s\n\n", output)
    }
}
```

**Usage examples:**

```bash
# Build
go build -o tokenize ./cmd/tokenize

# Single tokenization
./tokenize dictionaries/german_words.txt "Wärmedämmung"
# Output: ["wärmedämmung","warm","dammung"]

# Interactive mode
./tokenize dictionaries/german_words.txt
> Der Dampfschifffahrtskapitän fährt
→ ["der","dampfschifffahrtskapitän","dampf","schifffahrt","kapitan","fährt","fahrt"]
```

---

### Step 8: Dictionary Management CLI (`cmd/dictmgr/main.go`)

**Purpose**: Add/remove words from dictionary and rebuild FST.

```go
package main

import (
    "fmt"
    "os"

    "github.com/yourorg/german-tokenizer/pkg/tokenizer"
)

func main() {
    if len(os.Args) < 3 {
        printUsage()
        os.Exit(1)
    }

    dictPath := os.Args[1]
    command := os.Args[2]

    dict, err := tokenizer.NewDictionary(dictPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error loading dictionary: %v\n", err)
        os.Exit(1)
    }
    defer dict.Close()

    switch command {
    case "add":
        if len(os.Args) < 4 {
            fmt.Println("Error: add requires at least one word")
            os.Exit(1)
        }
        for _, word := range os.Args[3:] {
            dict.AddWord(word)
            fmt.Printf("Added: %s (+ uppercase variant)\n", word)
        }
        if err := dict.RebuildFST(); err != nil {
            fmt.Fprintf(os.Stderr, "Error rebuilding FST: %v\n", err)
            os.Exit(1)
        }
        fmt.Printf("FST rebuilt. Total words: %d\n", dict.WordCount())

    case "remove":
        if len(os.Args) < 4 {
            fmt.Println("Error: remove requires at least one word")
            os.Exit(1)
        }
        for _, word := range os.Args[3:] {
            dict.RemoveWord(word)
            fmt.Printf("Removed: %s (+ uppercase variant)\n", word)
        }
        if err := dict.RebuildFST(); err != nil {
            fmt.Fprintf(os.Stderr, "Error rebuilding FST: %v\n", err)
            os.Exit(1)
        }
        fmt.Printf("FST rebuilt. Total words: %d\n", dict.WordCount())

    case "contains":
        if len(os.Args) < 4 {
            fmt.Println("Error: contains requires a word")
            os.Exit(1)
        }
        word := os.Args[3]
        if dict.Contains(word) {
            fmt.Printf("✓ '%s' exists in dictionary\n", word)
        } else {
            fmt.Printf("✗ '%s' NOT in dictionary\n", word)
        }

    case "rebuild":
        if err := dict.RebuildFST(); err != nil {
            fmt.Fprintf(os.Stderr, "Error rebuilding FST: %v\n", err)
            os.Exit(1)
        }
        fmt.Printf("FST rebuilt. Total words: %d\n", dict.WordCount())

    case "stats":
        fmt.Printf("Dictionary: %s\n", dictPath)
        fmt.Printf("Word count: %d\n", dict.WordCount())
        fmt.Printf("Dirty: %v\n", dict.IsDirty())

    default:
        fmt.Printf("Unknown command: %s\n", command)
        printUsage()
        os.Exit(1)
    }
}

func printUsage() {
    fmt.Println("Usage: dictmgr <dictionary.txt> <command> [args...]")
    fmt.Println()
    fmt.Println("Commands:")
    fmt.Println("  add <word> [word...]    Add words (both upper/lowercase)")
    fmt.Println("  remove <word> [word...] Remove words (both cases)")
    fmt.Println("  contains <word>         Check if word exists")
    fmt.Println("  rebuild                 Rebuild FST from text file")
    fmt.Println("  stats                   Show dictionary statistics")
}
```

**Usage examples:**

```bash
# Build
go build -o dictmgr ./cmd/dictmgr

# Add words (adds both "blockchain" and "Blockchain")
./dictmgr dictionaries/german_words.txt add blockchain kryptowährung
# Output:
# Added: blockchain (+ uppercase variant)
# Added: kryptowährung (+ uppercase variant)
# FST rebuilt. Total words: 300002

# Remove words (removes both cases)
./dictmgr dictionaries/german_words.txt remove blockchain
# Output:
# Removed: blockchain (+ uppercase variant)
# FST rebuilt. Total words: 300001

# Check if word exists
./dictmgr dictionaries/german_words.txt contains haus
# Output: ✓ 'haus' exists in dictionary

# Show stats
./dictmgr dictionaries/german_words.txt stats
# Output:
# Dictionary: dictionaries/german_words.txt
# Word count: 300000
# Dirty: false

# Force rebuild FST
./dictmgr dictionaries/german_words.txt rebuild
```

---

## Verification Checklist

After implementation, verify with these test cases:

| Input                        | Expected Output                                                   |
| ---------------------------- | ----------------------------------------------------------------- |
| `"Wärmedämmung"`             | `["wärmedämmung", "warm", "dammung"]`                             |
| `"fährt"`                    | `["fährt", "fahrt"]`                                              |
| `"über"`                     | `["über", "uber"]`                                                |
| `"Haus"`                     | `["haus"]` (deduplicated)                                         |
| `"Der"`                      | `["der"]`                                                         |
| `"Dampfschifffahrtskapitän"` | `["dampfschifffahrtskapitän", "dampf", "schifffahrt", "kapitan"]` |
| `"insgesamt"`                | `["insgesamt"]` (can't split)                                     |
| `"„Wort""`                   | `["wort"]` (quotes normalized)                                    |

---

## Files Summary

| File                          | Purpose                              | Lines (est.)   |
| ----------------------------- | ------------------------------------ | -------------- |
| `pkg/tokenizer/dictionary.go` | FST-based dictionary with add/remove | ~180           |
| `pkg/tokenizer/normalizer.go` | Full normalization pipeline          | ~120           |
| `pkg/tokenizer/splitter.go`   | Split text into words                | ~60            |
| `pkg/tokenizer/compound.go`   | Compound decomposition               | ~120           |
| `pkg/tokenizer/tokenizer.go`  | Main API orchestration               | ~80            |
| `cmd/tokenize/main.go`        | Tokenization CLI                     | ~50            |
| `cmd/dictmgr/main.go`         | Dictionary management CLI            | ~80            |
| Tests                         | Unit tests for each module           | ~250           |
| **Total**                     |                                      | **~940 lines** |

---

## Dependencies

```go
require (
    golang.org/x/text v0.14.0           // NFKD normalization
    github.com/kljensen/snowball v1.0.0 // Snowball stemmer
    github.com/blevesearch/vellum v1.0.0 // FST for lightning-fast dictionary lookups
)
```

---

## Future: Production Web Service

The tokenizer will eventually be deployed as a **production-grade, high-load web service**.

### Basic API (for development/testing)

```go
// cmd/server/main.go
func main() {
    tok, _ := tokenizer.NewTokenizer("dictionaries/german_words.txt")

    http.HandleFunc("/tokenize", func(w http.ResponseWriter, r *http.Request) {
        text := r.URL.Query().Get("text")
        tokens := tok.Tokenize(text)
        json.NewEncoder(w).Encode(tokens)
    })

    http.ListenAndServe(":8080", nil)
}
```

Usage: `GET /tokenize?text=Wärmedämmung` → `["wärmedämmung","warm","dammung"]`

### Production Requirements (to be detailed later)

The production deployment will include:

- **Authentication & Authorization** - API keys, JWT, or OAuth2
- **Rate Limiting** - Per-client request throttling
- **TLS/HTTPS** - Encrypted transport
- **Input Validation** - Request size limits, sanitization
- **Logging & Monitoring** - Structured logs, metrics, health checks
- **Horizontal Scaling** - Stateless design for load balancing
- **Caching** - Response caching for repeated queries
- **Graceful Shutdown** - Connection draining on deploy
- **Error Handling** - Proper HTTP status codes, error responses

Specifics to be discussed before implementation.
