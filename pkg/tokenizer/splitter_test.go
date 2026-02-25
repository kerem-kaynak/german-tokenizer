package tokenizer

import (
	"testing"
)

func TestSplitWords(t *testing.T) {
	tests := []struct {
		input    string
		expected []RawToken
	}{
		{
			input: "Der fährt!",
			expected: []RawToken{
				{Text: "Der", Type: TokenWord, Start: 0, End: 3},
				{Text: " ", Type: TokenSeparator, Start: 3, End: 4},
				{Text: "fährt", Type: TokenWord, Start: 4, End: 9},
				{Text: "!", Type: TokenSeparator, Start: 9, End: 10},
			},
		},
		{
			input: "Wärmedämmung",
			expected: []RawToken{
				{Text: "Wärmedämmung", Type: TokenWord, Start: 0, End: 12},
			},
		},
		{
			input:    "",
			expected: []RawToken{},
		},
		{
			input: "hello world",
			expected: []RawToken{
				{Text: "hello", Type: TokenWord, Start: 0, End: 5},
				{Text: " ", Type: TokenSeparator, Start: 5, End: 6},
				{Text: "world", Type: TokenWord, Start: 6, End: 11},
			},
		},
		{
			input: "123abc",
			expected: []RawToken{
				{Text: "123abc", Type: TokenWord, Start: 0, End: 6},
			},
		},
		{
			input: "  ",
			expected: []RawToken{
				{Text: "  ", Type: TokenSeparator, Start: 0, End: 2},
			},
		},
	}

	for _, tt := range tests {
		result := SplitWords(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("SplitWords(%q) returned %d tokens, want %d", tt.input, len(result), len(tt.expected))
			continue
		}
		for i, tok := range result {
			if tok.Text != tt.expected[i].Text || tok.Type != tt.expected[i].Type {
				t.Errorf("SplitWords(%q)[%d] = {%q, %v}, want {%q, %v}",
					tt.input, i, tok.Text, tok.Type, tt.expected[i].Text, tt.expected[i].Type)
			}
		}
	}
}

func TestGetTokenType(t *testing.T) {
	tests := []struct {
		input    rune
		expected TokenType
	}{
		{'a', TokenWord},
		{'Z', TokenWord},
		{'ä', TokenWord},
		{'5', TokenWord},
		{' ', TokenSeparator},
		{'.', TokenSeparator},
		{',', TokenSeparator},
		{'!', TokenSeparator},
		{'-', TokenSeparator},
	}

	for _, tt := range tests {
		result := getTokenType(tt.input)
		if result != tt.expected {
			t.Errorf("getTokenType(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
