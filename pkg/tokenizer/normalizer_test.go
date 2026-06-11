package tokenizer

import (
	"testing"
)

func TestNFKDDecompose(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ä", "a\u0308"},
		{"ö", "o\u0308"},
		{"ü", "u\u0308"},
		{"ﬁ", "fi"},
		{"ﬂ", "fl"},
		{"hello", "hello"},
	}

	for _, tt := range tests {
		result := NFKDDecompose(tt.input)
		if result != tt.expected {
			t.Errorf("NFKDDecompose(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRemoveControlChars(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello\x00world", "helloworld"},
		{"test\x1fstring", "teststring"},
		{"normal", "normal"},
	}

	for _, tt := range tests {
		result := RemoveControlChars(tt.input)
		if result != tt.expected {
			t.Errorf("RemoveControlChars(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestLowercase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HELLO", "hello"},
		{"Wärme", "wärme"},
		{"ÜBER", "über"},
	}

	for _, tt := range tests {
		result := Lowercase(tt.input)
		if result != tt.expected {
			t.Errorf("Lowercase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"\u201EWort\u201C", "\"Wort\""},
		{"\u00ABtext\u00BB", "\"text\""},
		{"\u2018single\u2019", "'single'"},
	}

	for _, tt := range tests {
		result := NormalizeQuotes(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeQuotes(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExpandLigatures(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"æther", "aether"},
		{"Œuvre", "oeuvre"},
		{"Æsthetic", "aesthetic"},
	}

	for _, tt := range tests {
		result := ExpandLigatures(tt.input)
		if result != tt.expected {
			t.Errorf("ExpandLigatures(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestConvertEszett(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Größe", "Grösse"}, // ConvertEszett only converts ß, not ö
		{"Straße", "Strasse"},
		{"groß", "gross"},
		{"Fuß", "Fuss"},
		{"ß", "ss"},
	}

	for _, tt := range tests {
		result := ConvertEszett(tt.input)
		if result != tt.expected {
			t.Errorf("ConvertEszett(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRemoveCombiningMarks(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a\u0308", "a"}, // ä decomposed
		{"o\u0308", "o"}, // ö decomposed
		{"u\u0308", "u"}, // ü decomposed
		{"normal", "normal"},
	}

	for _, tt := range tests {
		result := RemoveCombiningMarks(tt.input)
		if result != tt.expected {
			t.Errorf("RemoveCombiningMarks(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestStemGerman(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"haus", "haus"},
		{"warm", "warm"},
		{"warme", "warm"},
		{"laufen", "lauf"},
		{"arbeiten", "arbeit"},
	}

	for _, tt := range tests {
		result := StemGerman(tt.input)
		if result != tt.expected {
			t.Errorf("StemGerman(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestStemGerman_SnowballGroundTruth pins the full pipeline against the
// official snowball-data German corpus (github.com/snowballstem/snowball-data).
// The pipeline folds umlauts/ß before stemming; this is output-equivalent to
// canonical Snowball, which folds them internally (prelude ß→ss, postlude
// ä/ö/ü→a/o/u). Note: -ung is only stripped when it lies in region R2, so
// short words like Dämmung keep it while Bezeichnung loses it.
func TestStemGerman_SnowballGroundTruth(t *testing.T) {
	n := NewNormalizer()

	tests := []struct {
		input    string
		expected string
	}{
		{"Dämmung", "dammung"},   // -ung not in R2: kept
		{"Dämmungen", "dammung"}, // plural -en stripped: conflates with singular
		{"Dämme", "damm"},
		{"Dämmerung", "dammer"}, // long enough: -ung in R2, stripped
		{"Bezeichnung", "bezeichn"},
		{"Größe", "gross"},
		{"Meinung", "meinung"},
		{"Wohnung", "wohnung"},
		{"Wärme", "warm"},
		{"schifffahrts", "schifffahrt"}, // linking -s after valid s-ending
	}

	for _, tt := range tests {
		result := n.Normalize(tt.input)
		if result != tt.expected {
			t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizer_Normalize(t *testing.T) {
	n := NewNormalizer()

	// Test the full pipeline output
	tests := []struct {
		input    string
		expected string
	}{
		{"Wärme", "warm"},
		{"Größe", "gross"},
		{"über", "uber"},
		{"HAUS", "haus"},
	}

	for _, tt := range tests {
		result := n.Normalize(tt.input)
		if result != tt.expected {
			t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizer_LowercaseOnly(t *testing.T) {
	n := NewNormalizer()

	tests := []struct {
		input    string
		expected string
	}{
		{"WÄRME", "wärme"},
		{"GRÖßE", "größe"},
		{"ÜBER", "über"},
	}

	for _, tt := range tests {
		result := n.LowercaseOnly(tt.input)
		if result != tt.expected {
			t.Errorf("LowercaseOnly(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewNormalizerWithSteps(t *testing.T) {
	// Custom pipeline: only lowercase and eszett conversion
	n := NewNormalizerWithSteps(Lowercase, ConvertEszett)

	result := n.Normalize("Größe")
	expected := "grösse" // ö preserved, ß → ss
	if result != expected {
		t.Errorf("Custom Normalize(%q) = %q, want %q", "Größe", result, expected)
	}

	// Verify umlauts are preserved (no NFKD or combining marks removal)
	result = n.Normalize("Wärme")
	expected = "wärme"
	if result != expected {
		t.Errorf("Custom Normalize(%q) = %q, want %q", "Wärme", result, expected)
	}
}

func TestFullPipelineUmlautHandling(t *testing.T) {
	n := NewNormalizer()

	// Test that umlauts are properly normalized to ASCII
	result := n.Normalize("ä")
	if result != "a" {
		t.Errorf("Full pipeline 'ä' = %q, want 'a'", result)
	}

	result = n.Normalize("ö")
	if result != "o" {
		t.Errorf("Full pipeline 'ö' = %q, want 'o'", result)
	}

	result = n.Normalize("ü")
	if result != "u" {
		t.Errorf("Full pipeline 'ü' = %q, want 'u'", result)
	}
}

func TestFullPipelineEszettHandling(t *testing.T) {
	n := NewNormalizer()

	// Test that ß is converted to ss
	result := n.Normalize("groß")
	if result != "gross" {
		t.Errorf("Full pipeline 'groß' = %q, want 'gross'", result)
	}
}
