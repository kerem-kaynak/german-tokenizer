package tokenizer

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
		Cache:             true,
		LowercaseOriginal: true,
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

func TestTokenizer_Tokenize(t *testing.T) {
	dictPath := getTestDictPath()
	tok, err := NewTokenizer(dictPath, testConfig())
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
	tok, err := NewTokenizer(dictPath, testConfig())
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
	tok, err := NewTokenizer(dictPath, testConfig())
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

	cfg := testConfig()
	cfg.LowercaseOriginal = false

	tok, err := NewTokenizer(dictPath, cfg)
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
	resultSet := make(map[string]bool)
	for _, tok := range result {
		resultSet[tok] = true
	}

	if !resultSet["brand"] {
		t.Errorf("Expected 'brand' in result, got %v", result)
	}
}

// detailedConfig matches the parity contract (Design A in the search-migration
// plan): no stemming, full folding. TokenizeDetailed is meant to be used by
// the HTTP service that ships with this config.
func detailedConfig() Config {
	cfg := testConfig()
	cfg.Normalizers.StemGerman = false
	return cfg
}

func TestTokenizer_TokenizeDetailed_Compound(t *testing.T) {
	tok, err := NewTokenizer(getTestDictPath(), detailedConfig())
	if err != nil {
		t.Fatalf("NewTokenizer: %v", err)
	}
	defer tok.Close()

	got := tok.TokenizeDetailed("Brandschutzkonzept")
	if len(got) != 1 {
		t.Fatalf("expected 1 word, got %d (%v)", len(got), got)
	}
	if got[0].Original != "brandschutzkonzept" {
		t.Errorf("Original = %q, want 'brandschutzkonzept'", got[0].Original)
	}
	wantSegs := []string{"brand", "schutz", "konzept"}
	if !reflect.DeepEqual(got[0].Segments, wantSegs) {
		t.Errorf("Segments = %v, want %v", got[0].Segments, wantSegs)
	}
}

func TestTokenizer_TokenizeDetailed_UmlautPreservedInOriginal(t *testing.T) {
	tok, err := NewTokenizer(getTestDictPath(), detailedConfig())
	if err != nil {
		t.Fatalf("NewTokenizer: %v", err)
	}
	defer tok.Close()

	got := tok.TokenizeDetailed("Wärmedämmung")
	if len(got) != 1 {
		t.Fatalf("expected 1 word, got %d", len(got))
	}
	if got[0].Original != "wärmedämmung" {
		t.Errorf("Original = %q, want 'wärmedämmung' (umlauts preserved)", got[0].Original)
	}
	// Segments are normalized → umlauts folded.
	for _, s := range got[0].Segments {
		if strings.ContainsAny(s, "äöüÄÖÜ") {
			t.Errorf("Segment %q still contains umlaut; normalization should fold", s)
		}
	}
}

func TestTokenizer_TokenizeDetailed_CompoundRecallTargets(t *testing.T) {
	// These are the recall targets called out in the PRD:
	// wand → Außenwand, Brandschutz → Brandschutzkonzept (already covered above).
	tok, err := NewTokenizer(getTestDictPath(), detailedConfig())
	if err != nil {
		t.Fatalf("NewTokenizer: %v", err)
	}
	defer tok.Close()

	got := tok.TokenizeDetailed("Außenwand")
	if len(got) != 1 {
		t.Fatalf("expected 1 word, got %d", len(got))
	}
	if got[0].Original != "außenwand" {
		t.Errorf("Original = %q, want 'außenwand'", got[0].Original)
	}
	// Segments must contain 'wand' (normalized) so a query for 'wand' recalls this.
	found := false
	for _, s := range got[0].Segments {
		if s == "wand" {
			found = true
		}
	}
	if !found {
		t.Errorf("Außenwand segments missing 'wand': %v", got[0].Segments)
	}
}

func TestTokenizer_TokenizeDetailed_NonCompound(t *testing.T) {
	tok, err := NewTokenizer(getTestDictPath(), detailedConfig())
	if err != nil {
		t.Fatalf("NewTokenizer: %v", err)
	}
	defer tok.Close()

	got := tok.TokenizeDetailed("Haus")
	if len(got) != 1 {
		t.Fatalf("expected 1 word, got %d", len(got))
	}
	if got[0].Original != "haus" {
		t.Errorf("Original = %q, want 'haus'", got[0].Original)
	}
	// Splitter returns [word] when no split applies → Segments has the lone form.
	if len(got[0].Segments) != 1 || got[0].Segments[0] != "haus" {
		t.Errorf("Segments = %v, want ['haus']", got[0].Segments)
	}
}

func TestTokenizer_TokenizeDetailed_MultiWord(t *testing.T) {
	tok, err := NewTokenizer(getTestDictPath(), detailedConfig())
	if err != nil {
		t.Fatalf("NewTokenizer: %v", err)
	}
	defer tok.Close()

	got := tok.TokenizeDetailed("Das Brandschutzkonzept der Außenwand")
	if len(got) != 4 {
		t.Fatalf("expected 4 words, got %d (%v)", len(got), got)
	}
	wantOriginals := []string{"das", "brandschutzkonzept", "der", "außenwand"}
	for i, w := range wantOriginals {
		if got[i].Original != w {
			t.Errorf("word %d Original = %q, want %q", i, got[i].Original, w)
		}
	}
}

func TestTokenizer_TokenizeDetailed_ConsistentWithTokenize(t *testing.T) {
	// Invariant: with LowercaseOriginal=true, the in-order union of
	// Original ++ Segments across all words, globally deduped, equals Tokenize.
	cfg := detailedConfig()
	cfg.LowercaseOriginal = true
	tok, err := NewTokenizer(getTestDictPath(), cfg)
	if err != nil {
		t.Fatalf("NewTokenizer: %v", err)
	}
	defer tok.Close()

	inputs := []string{
		"Brandschutzkonzept",
		"Wärmedämmung",
		"Außenwand Müllraum",
		"Das Brandschutzkonzept der Außenwand",
		"Haus",
		"Größe",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			detailed := tok.TokenizeDetailed(input)

			seen := make(map[string]struct{})
			var rebuilt []string
			for _, w := range detailed {
				if _, ok := seen[w.Original]; !ok {
					seen[w.Original] = struct{}{}
					rebuilt = append(rebuilt, w.Original)
				}
				for _, s := range w.Segments {
					if _, ok := seen[s]; !ok {
						seen[s] = struct{}{}
						rebuilt = append(rebuilt, s)
					}
				}
			}

			tokens := tok.Tokenize(input)
			if !reflect.DeepEqual(rebuilt, tokens) {
				t.Errorf("union mismatch for %q:\n  Tokenize     = %v\n  TokenizeDetailed-union = %v",
					input, tokens, rebuilt)
			}
		})
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
