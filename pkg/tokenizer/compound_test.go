package tokenizer

import (
	"testing"
)

func TestCompoundSplitter_Split(t *testing.T) {
	dictPath := getTestDictPath()
	dict, err := NewDictionary(dictPath)
	if err != nil {
		t.Fatalf("Failed to load components: %v", err)
	}
	defer dict.Close()

	splitter := NewCompoundSplitter(dict)

	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "brandschutzkonzept",
			expected: []string{"brand", "schutz", "konzept"},
		},
		{
			input:    "stahlbetondecke",
			expected: []string{"stahl", "beton", "decke"},
		},
		{
			input:    "wärmedämmung",
			expected: []string{"wärme", "dämmung"},
		},
		{
			// Non-splittable word
			input:    "beton",
			expected: []string{"beton"},
		},
		{
			// Simple word that shouldn't split
			input:    "haus",
			expected: []string{"haus"},
		},
	}

	for _, tt := range tests {
		result := splitter.Split(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("Split(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, seg := range result {
			if seg != tt.expected[i] {
				t.Errorf("Split(%q)[%d] = %q, want %q", tt.input, i, seg, tt.expected[i])
			}
		}
	}
}

func TestCompoundSplitter_Cache(t *testing.T) {
	dictPath := getTestDictPath()
	dict, err := NewDictionary(dictPath)
	if err != nil {
		t.Fatalf("Failed to load components: %v", err)
	}
	defer dict.Close()

	splitter := NewCompoundSplitter(dict)

	// First call should add to cache
	splitter.Split("brandschutzkonzept")
	if splitter.CacheSize() != 1 {
		t.Errorf("Expected cache size 1 after first split, got %d", splitter.CacheSize())
	}

	// Second call with same word should use cache
	splitter.Split("brandschutzkonzept")
	if splitter.CacheSize() != 1 {
		t.Errorf("Expected cache size 1 after second split (cache hit), got %d", splitter.CacheSize())
	}

	// Different word should add to cache
	splitter.Split("stahlbetondecke")
	if splitter.CacheSize() != 2 {
		t.Errorf("Expected cache size 2 after different word, got %d", splitter.CacheSize())
	}

	// Clear cache
	splitter.ClearCache()
	if splitter.CacheSize() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", splitter.CacheSize())
	}
}

func TestNormalizeUmlauts(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"wärme", "warme"},
		{"größe", "grosse"},
		{"über", "uber"},
		{"haus", "haus"},
		{"äöüß", "aouss"},  // ä→a, ö→o, ü→u, ß→ss = aouss
	}

	for _, tt := range tests {
		result := normalizeUmlauts(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeUmlauts(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
