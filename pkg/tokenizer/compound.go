package tokenizer

import (
	"strings"

	lru "github.com/hashicorp/golang-lru/v2"
)

// CacheSize is the maximum number of entries in the compound split cache.
// At ~100 bytes per entry, 100k entries uses approximately 10MB of memory.
const CacheSize = 100_000

// germanSuffixes for validation fallback during segment validation.
var germanSuffixes = []string{
	"ungen", "schaft", "heiten", "keiten",
	"ung", "heit", "keit", "tion", "isch", "lich", "chen", "lein",
	"haft", "bar", "sam", "tum", "ig", "er", "en", "em", "es",
	"st", "nd", "te", "el", "le", "se", "ße", "ze",
	"e", "s", "n", "t",
}

// CompoundSplitter handles German compound word decomposition.
type CompoundSplitter struct {
	dict  *Dictionary
	cache *lru.Cache[string, []string]
}

// NewCompoundSplitter creates a new splitter with dictionary and LRU cache enabled.
func NewCompoundSplitter(dict *Dictionary) *CompoundSplitter {
	cache, _ := lru.New[string, []string](CacheSize)
	return &CompoundSplitter{
		dict:  dict,
		cache: cache,
	}
}

// NewCompoundSplitterNoCache creates a new splitter without caching.
// Use this when memory is constrained or words are rarely repeated.
func NewCompoundSplitterNoCache(dict *Dictionary) *CompoundSplitter {
	return &CompoundSplitter{
		dict:  dict,
		cache: nil,
	}
}

// Split attempts to decompose a compound word.
// Returns segments if successful, or [word] if can't split.
func (c *CompoundSplitter) Split(word string) []string {
	lower := strings.ToLower(word)

	// If cache is disabled, compute directly
	if c.cache == nil {
		return c.splitUncached(lower)
	}

	// Check cache first (LRU is thread-safe)
	if result, ok := c.cache.Get(lower); ok {
		return result
	}

	// Compute split
	result := c.splitUncached(lower)

	// Store in cache (evicts oldest if at capacity)
	c.cache.Add(lower, result)

	return result
}

// splitUncached performs the actual splitting without cache.
func (c *CompoundSplitter) splitUncached(word string) []string {
	segments := c.greedySplit(word)

	// Validate all segments
	if c.allSegmentsValid(segments) && len(segments) > 1 {
		return segments
	}

	// Fallback: return original word as single segment
	return []string{word}
}

// greedySplit tries to split word from left to right.
func (c *CompoundSplitter) greedySplit(word string) []string {
	var segments []string
	remaining := word

	for len(remaining) > 0 {
		found := false
		runes := []rune(remaining)

		// Try longest match first (minimum 2 chars)
		for length := len(runes); length >= 2; length-- {
			prefix := string(runes[:length])
			rest := string(runes[length:])

			// For final segment (rest is empty), allow suffix-based matching
			// For intermediate segments, use strict direct lookup only
			var isValid bool
			if len(rest) == 0 {
				isValid = c.isValidWord(prefix)
			} else {
				isValid = c.isWordInDict(prefix)
			}

			if isValid {
				segments = append(segments, prefix)
				remaining = rest
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

// isWordInDict checks if word exists in dictionary (direct lookup + umlaut normalization only).
// Used during greedy split to avoid false positives from suffix stripping.
func (c *CompoundSplitter) isWordInDict(word string) bool {
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

	return false
}

// isValidWord checks if word exists in dictionary (with suffix fallback).
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
				if c.dict.Contains(normalizeUmlauts(stem)) {
					return true
				}
			}
		}
	}

	return false
}

// allSegmentsValid checks if all segments pass validation.
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

// normalizeUmlauts converts ä→a, ö→o, ü→u, ß→ss.
// Used ONLY for dictionary lookup during compound decomposition.
// This is SEPARATE from the token normalization pipeline.
func normalizeUmlauts(s string) string {
	replacer := strings.NewReplacer(
		"ä", "a", "Ä", "a",
		"ö", "o", "Ö", "o",
		"ü", "u", "Ü", "u",
		"ß", "ss",
	)
	return replacer.Replace(s)
}

// ClearCache clears the memoization cache.
func (c *CompoundSplitter) ClearCache() {
	if c.cache != nil {
		c.cache.Purge()
	}
}

// CacheSize returns the number of cached entries (0 if cache is disabled).
func (c *CompoundSplitter) CacheSize() int {
	if c.cache == nil {
		return 0
	}
	return c.cache.Len()
}

// CacheEnabled returns true if caching is enabled.
func (c *CompoundSplitter) CacheEnabled() bool {
	return c.cache != nil
}
