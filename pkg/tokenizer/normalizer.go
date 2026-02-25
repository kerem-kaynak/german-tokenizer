package tokenizer

import (
	"strings"
	"unicode"

	"github.com/kljensen/snowball"
	"golang.org/x/text/unicode/norm"
)

// NormalizerFunc defines a single normalization step.
type NormalizerFunc func(string) string

// Normalizer applies a configurable pipeline of normalization steps.
type Normalizer struct {
	steps []NormalizerFunc
}

// NewNormalizer creates a normalizer with the default pipeline.
func NewNormalizer() *Normalizer {
	return &Normalizer{
		steps: []NormalizerFunc{
			NFKDDecompose,
			RemoveControlChars,
			Lowercase,
			NormalizeQuotes,
			ExpandLigatures,
			ConvertEszett,
			RemoveCombiningMarks,
			StemGerman,
		},
	}
}

// NewNormalizerWithSteps creates a normalizer with a custom pipeline.
func NewNormalizerWithSteps(steps ...NormalizerFunc) *Normalizer {
	return &Normalizer{steps: steps}
}

// Normalize applies all configured steps in order.
func (n *Normalizer) Normalize(s string) string {
	for _, step := range n.steps {
		s = step(s)
	}
	return s
}

// LowercaseOnly lowercases without other transformations (preserves umlauts).
func (n *Normalizer) LowercaseOnly(s string) string {
	return strings.ToLower(s)
}

// NFKDDecompose applies Unicode NFKD normalization.
// Decomposes ä → a + combining_umlaut, ﬁ → fi, etc.
func NFKDDecompose(s string) string {
	return norm.NFKD.String(s)
}

// RemoveControlChars removes Unicode control characters.
func RemoveControlChars(s string) string {
	var result strings.Builder
	result.Grow(len(s))
	for _, r := range s {
		if !unicode.IsControl(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// Lowercase converts to lowercase.
func Lowercase(s string) string {
	return strings.ToLower(s)
}

// quoteReplacements maps fancy quotes to ASCII.
var quoteReplacements = map[rune]rune{
	'\u201E': '"',  // „ German opening quote
	'\u201C': '"',  // " left double quote
	'\u201D': '"',  // " right double quote
	'\u00AB': '"',  // « left-pointing double angle
	'\u00BB': '"',  // » right-pointing double angle
	'\u2018': '\'', // ' left single quote
	'\u2019': '\'', // ' right single quote
	'\u201A': '\'', // ‚ single low-9 quote
	'\u2039': '\'', // ‹ single left-pointing angle
	'\u203A': '\'', // › single right-pointing angle
}

// NormalizeQuotes converts fancy quotes to ASCII.
func NormalizeQuotes(s string) string {
	var result strings.Builder
	result.Grow(len(s))
	for _, r := range s {
		if replacement, ok := quoteReplacements[r]; ok {
			result.WriteRune(replacement)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ExpandLigatures expands æ→ae, œ→oe.
func ExpandLigatures(s string) string {
	s = strings.ReplaceAll(s, "æ", "ae")
	s = strings.ReplaceAll(s, "Æ", "ae")
	s = strings.ReplaceAll(s, "œ", "oe")
	s = strings.ReplaceAll(s, "Œ", "oe")
	return s
}

// ConvertEszett converts ß to ss.
// Critical: NFKD does not decompose ß, so explicit conversion is needed.
func ConvertEszett(s string) string {
	return strings.ReplaceAll(s, "ß", "ss")
}

// RemoveCombiningMarks removes Unicode combining characters (category Mn).
// Removes umlaut dots after NFKD decomposition.
func RemoveCombiningMarks(s string) string {
	var result strings.Builder
	result.Grow(len(s))
	for _, r := range s {
		if !unicode.Is(unicode.Mn, r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// StemGerman applies the German Snowball stemmer.
func StemGerman(s string) string {
	stemmed, err := snowball.Stem(s, "german", true)
	if err != nil {
		return s
	}
	return stemmed
}
