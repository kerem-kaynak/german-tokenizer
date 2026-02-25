package tokenizer

import (
	"os"
	"path/filepath"
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

// allNormalizersEnabled returns a NormalizerConfig with all normalizers enabled.
// Used in tests for convenience.
func allNormalizersEnabled() NormalizerConfig {
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

func TestTokenizer_Tokenize(t *testing.T) {
	dictPath := getTestDictPath()
	tok, err := NewTokenizer(dictPath, Config{
		Cache:             true,
		LowercaseOriginal: true,
		Normalizers:       allNormalizersEnabled(),
	})
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	tests := []struct {
		input    string
		contains []string // tokens that must be present (normalized forms)
	}{
		{
			input:    "Wärmedämmung",
			contains: []string{"wärmedämmung"}, // Original preserved, segments are normalized
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
		resultSet := make(map[string]bool)
		for _, tok := range result {
			resultSet[tok] = true
		}

		for _, expected := range tt.contains {
			if !resultSet[expected] {
				t.Errorf("Tokenize(%q) missing expected token %q, got %v", tt.input, expected, result)
			}
		}
	}
}

func TestTokenizer_Deduplication(t *testing.T) {
	dictPath := getTestDictPath()
	tok, err := NewTokenizer(dictPath, Config{
		Cache:             true,
		LowercaseOriginal: true,
		Normalizers:       allNormalizersEnabled(),
	})
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	// "Haus" should deduplicate to just "haus" (original and normalized are the same)
	result := tok.Tokenize("Haus")

	// Count occurrences of "haus"
	count := 0
	for _, token := range result {
		if token == "haus" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected 'haus' to appear exactly once, got %d times in %v", count, result)
	}
}

func TestTokenizer_MultipleWords(t *testing.T) {
	dictPath := getTestDictPath()
	tok, err := NewTokenizer(dictPath, Config{
		Cache:             true,
		LowercaseOriginal: true,
		Normalizers:       allNormalizersEnabled(),
	})
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	result := tok.Tokenize("Der fährt")

	// Should contain tokens from both words
	resultSet := make(map[string]bool)
	for _, tok := range result {
		resultSet[tok] = true
	}

	expected := []string{"der", "fährt"}
	for _, e := range expected {
		if !resultSet[e] {
			t.Errorf("Tokenize('Der fährt') missing expected token %q, got %v", e, result)
		}
	}
}

func TestTokenizer_WithCustomNormalizer(t *testing.T) {
	dictPath := getTestDictPath()

	// Create tokenizer with custom normalizer config (without stemming)
	tok, err := NewTokenizer(dictPath, Config{
		Cache:             true,
		LowercaseOriginal: true,
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

	// Without stemming, should get "grosse" not stemmed form
	resultSet := make(map[string]bool)
	for _, tok := range result {
		resultSet[tok] = true
	}

	if !resultSet["grosse"] {
		t.Errorf("Expected 'grosse' in result, got %v", result)
	}
}

func TestTokenizer_WithoutLowercaseOriginal(t *testing.T) {
	dictPath := getTestDictPath()

	tok, err := NewTokenizer(dictPath, Config{
		Cache:             true,
		LowercaseOriginal: false,
		Normalizers:       allNormalizersEnabled(),
	})
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	result := tok.Tokenize("Brandschutzkonzept")

	// Without lowercase original, should NOT contain "brandschutzkonzept"
	// but should contain the normalized segments
	resultSet := make(map[string]bool)
	for _, tok := range result {
		resultSet[tok] = true
	}

	if resultSet["brandschutzkonzept"] {
		t.Errorf("Expected 'brandschutzkonzept' to NOT be in result with LowercaseOriginal=false, got %v", result)
	}

	// Should still have segments
	if !resultSet["brand"] {
		t.Errorf("Expected 'brand' in result, got %v", result)
	}
}

func TestTokenizer_WithoutCache(t *testing.T) {
	dictPath := getTestDictPath()

	tok, err := NewTokenizer(dictPath, Config{
		Cache:             false,
		LowercaseOriginal: true,
		Normalizers:       allNormalizersEnabled(),
	})
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	if tok.CacheEnabled() {
		t.Error("Expected cache to be disabled")
	}

	// Should still tokenize correctly
	result := tok.Tokenize("Brandschutzkonzept")
	resultSet := make(map[string]bool)
	for _, tok := range result {
		resultSet[tok] = true
	}

	if !resultSet["brand"] {
		t.Errorf("Expected 'brand' in result, got %v", result)
	}
}

func TestTokenizer_DictionaryWordCount(t *testing.T) {
	dictPath := getTestDictPath()
	tok, err := NewTokenizer(dictPath, Config{
		Cache:             true,
		LowercaseOriginal: true,
		Normalizers:       allNormalizersEnabled(),
	})
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	count := tok.DictionaryWordCount()
	if count < 1000 {
		t.Errorf("Expected at least 1000 words in dictionary, got %d", count)
	}
}
