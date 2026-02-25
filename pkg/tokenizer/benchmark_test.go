package tokenizer

import (
	"testing"
)

func BenchmarkTokenize_SingleWord(b *testing.B) {
	dictPath := getTestDictPath()
	tok, err := NewTokenizer(dictPath, Config{
		Cache:             true,
		LowercaseOriginal: true,
		Normalizers:       allNormalizersEnabled(),
	})
	if err != nil {
		b.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.Tokenize("Wärmedämmung")
	}
}

func BenchmarkTokenize_LongCompound(b *testing.B) {
	dictPath := getTestDictPath()
	tok, err := NewTokenizer(dictPath, Config{
		Cache:             true,
		LowercaseOriginal: true,
		Normalizers:       allNormalizersEnabled(),
	})
	if err != nil {
		b.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.Tokenize("Wärmedämmverbundsystem")
	}
}

func BenchmarkTokenize_Sentence(b *testing.B) {
	dictPath := getTestDictPath()
	tok, err := NewTokenizer(dictPath, Config{
		Cache:             true,
		LowercaseOriginal: true,
		Normalizers:       allNormalizersEnabled(),
	})
	if err != nil {
		b.Fatalf("Failed to create tokenizer: %v", err)
	}
	defer tok.Close()

	sentence := "Der Brandschutzkonzept und die Wärmedämmung der Stahlbetondecke"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.Tokenize(sentence)
	}
}

func BenchmarkNormalizer_FullPipeline(b *testing.B) {
	n := NewNormalizer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n.Normalize("Wärmedämmung")
	}
}

func BenchmarkCompoundSplitter_Split(b *testing.B) {
	dictPath := getTestDictPath()
	dict, err := NewDictionary(dictPath)
	if err != nil {
		b.Fatalf("Failed to load components: %v", err)
	}
	defer dict.Close()

	splitter := NewCompoundSplitter(dict)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitter.ClearCache() // Clear cache to measure actual splitting
		splitter.Split("brandschutzkonzept")
	}
}

func BenchmarkCompoundSplitter_CacheHit(b *testing.B) {
	dictPath := getTestDictPath()
	dict, err := NewDictionary(dictPath)
	if err != nil {
		b.Fatalf("Failed to load components: %v", err)
	}
	defer dict.Close()

	splitter := NewCompoundSplitter(dict)
	splitter.Split("brandschutzkonzept") // Prime the cache

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitter.Split("brandschutzkonzept")
	}
}

func BenchmarkDictionary_Contains(b *testing.B) {
	dictPath := getTestDictPath()
	dict, err := NewDictionary(dictPath)
	if err != nil {
		b.Fatalf("Failed to load dictionary: %v", err)
	}
	defer dict.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dict.Contains("brand")
	}
}
