package tokenizer

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func getTestDictPath() string {
	// Try to find the compound word components dictionary relative to the test file
	paths := []string{
		"../../dictionaries/german_compound_word_components.txt",
		"dictionaries/german_compound_word_components.txt",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Try absolute path from working directory
	wd, _ := os.Getwd()
	return filepath.Join(wd, "../../dictionaries/german_compound_word_components.txt")
}

// testConfig returns a Config with all features enabled for tests.
func testConfig() Config {
	return Config{
		Cache: true,
		Normalizers: NormalizerConfig{
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
}

// flatten returns the union of every Whole and every Parts entry across the
// result, so tests can assert "this token appears anywhere in the output"
// without caring about per-word grouping.
func flatten(words []WordTokens) map[string]bool {
	seen := make(map[string]bool)
	for _, w := range words {
		seen[w.Whole] = true
		for _, p := range w.Parts {
			seen[p] = true
		}
	}
	return seen
}

func TestTokenizer_Tokenize(t *testing.T) {
	dictPath := getTestDictPath()
	tok, err := NewTokenizer(dictPath, testConfig())
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	tests := []struct {
		input    string
		contains []string // tokens that must be present (as Whole or as a Part)
	}{
		{
			input:    "Wärmedämmung",
			contains: []string{"wärmedämmung"}, // Whole preserves umlauts; parts are stemmed
		},
		{
			input:    "Brandschutzkonzept",
			contains: []string{"brandschutzkonzept", "brand", "schutz", "konzept"},
		},
		{
			input:    "Stahlbetondecke",
			contains: []string{"stahlbetondecke", "stahl", "beton"}, // decke might be stemmed
		},
		{
			input:    "Größe",
			contains: []string{"größe"},
		},
		{
			input:    "Haus",
			contains: []string{"haus"},
		},
	}

	for _, tt := range tests {
		result := tok.Tokenize(tt.input)
		seen := flatten(result)

		for _, expected := range tt.contains {
			if !seen[expected] {
				t.Errorf("Tokenize(%q) missing expected token %q, got %+v", tt.input, expected, result)
			}
		}
	}
}

// TestTokenizer_Structure pins the exact WordTokens shape: Whole is the
// lowercased-with-umlauts input word, Parts are the normalized splitter
// segments in order, one entry per input word.
func TestTokenizer_Structure(t *testing.T) {
	tok, err := NewTokenizer(getTestDictPath(), testConfig())
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	result := tok.Tokenize("Brandschutzkonzept")

	if len(result) != 1 {
		t.Fatalf("expected 1 WordTokens, got %d: %+v", len(result), result)
	}
	if result[0].Whole != "brandschutzkonzept" {
		t.Errorf("Whole: want %q, got %q", "brandschutzkonzept", result[0].Whole)
	}
	want := []string{"brand", "schutz", "konzept"}
	if !slices.Equal(result[0].Parts, want) {
		t.Errorf("Parts: want %v, got %v", want, result[0].Parts)
	}
}

// TestTokenizer_NoCrossWordDedup pins the "one entry per detected word, no
// global dedup" contract that lets the query side weight by word position.
func TestTokenizer_NoCrossWordDedup(t *testing.T) {
	tok, err := NewTokenizer(getTestDictPath(), testConfig())
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	result := tok.Tokenize("Haus Haus")

	if len(result) != 2 {
		t.Fatalf("expected 2 WordTokens for repeated input, got %d: %+v", len(result), result)
	}
	for i, w := range result {
		if w.Whole != "haus" {
			t.Errorf("result[%d].Whole: want %q, got %q", i, "haus", w.Whole)
		}
	}
}

func TestTokenizer_MultipleWords(t *testing.T) {
	dictPath := getTestDictPath()
	tok, err := NewTokenizer(dictPath, testConfig())
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	result := tok.Tokenize("Der fährt")

	if len(result) != 2 {
		t.Errorf("Tokenize('Der fährt') expected 2 WordTokens (one per word), got %d: %+v", len(result), result)
	}

	seen := flatten(result)
	for _, expected := range []string{"der", "fährt"} {
		if !seen[expected] {
			t.Errorf("Tokenize('Der fährt') missing expected token %q, got %+v", expected, result)
		}
	}
}

func TestTokenizer_WithCustomNormalizer(t *testing.T) {
	dictPath := getTestDictPath()

	// Create tokenizer with custom normalizer config (without stemming)
	tok, err := NewTokenizer(dictPath, Config{
		Cache: true,
		Normalizers: NormalizerConfig{
			NFKDDecompose:        true,
			RemoveControlChars:   false,
			Lowercase:            true,
			NormalizeQuotes:      false,
			ExpandLigatures:      false,
			ConvertEszett:        true,
			RemoveCombiningMarks: true,
			StemGerman:           false, // No stemming
		},
	})
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	result := tok.Tokenize("Größe")

	// Without stemming, parts should contain "grosse" (not the stemmed form)
	seen := flatten(result)
	if !seen["grosse"] {
		t.Errorf("Expected 'grosse' in result, got %+v", result)
	}
}

func TestTokenizer_WithoutCache(t *testing.T) {
	dictPath := getTestDictPath()

	cfg := testConfig()
	cfg.Cache = false

	tok, err := NewTokenizer(dictPath, cfg)
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	if tok.CacheEnabled() {
		t.Error("Expected cache to be disabled")
	}

	// Should still tokenize correctly
	result := tok.Tokenize("Brandschutzkonzept")
	seen := flatten(result)

	if !seen["brand"] {
		t.Errorf("Expected 'brand' in result, got %+v", result)
	}
}

func TestTokenizer_DictionaryWordCount(t *testing.T) {
	dictPath := getTestDictPath()
	tok, err := NewTokenizer(dictPath, testConfig())
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	count := tok.DictionaryWordCount()
	if count < 1000 {
		t.Errorf("Expected at least 1000 words in dictionary, got %d", count)
	}
}
