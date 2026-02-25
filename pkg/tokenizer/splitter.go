package tokenizer

import (
	"unicode"
)

// TokenType identifies the type of token.
type TokenType int

const (
	TokenWord TokenType = iota
	TokenSeparator
)

// RawToken represents a token before normalization.
type RawToken struct {
	Text  string
	Type  TokenType
	Start int
	End   int
}

// SplitWords splits text into words and separators.
// Word characters: letters and numbers.
// Separators: whitespace, punctuation, symbols.
func SplitWords(text string) []RawToken {
	var tokens []RawToken
	runes := []rune(text)

	if len(runes) == 0 {
		return tokens
	}

	start := 0
	currentType := getTokenType(runes[0])

	for i := 1; i <= len(runes); i++ {
		var nextType TokenType
		if i < len(runes) {
			nextType = getTokenType(runes[i])
		} else {
			nextType = TokenType(-1) // Force flush
		}

		if nextType != currentType {
			tokens = append(tokens, RawToken{
				Text:  string(runes[start:i]),
				Type:  currentType,
				Start: start,
				End:   i,
			})
			start = i
			currentType = nextType
		}
	}

	return tokens
}

// getTokenType determines if a rune is a word character or separator.
func getTokenType(r rune) TokenType {
	if unicode.IsLetter(r) || unicode.IsNumber(r) {
		return TokenWord
	}
	return TokenSeparator
}
