package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kerem-kaynak/german-tokenizer/pkg/tokenizer"
)

const (
	iterations = 100000
	warmup     = 1000
	boxWidth   = 62

	// ANSI color codes
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorDim    = "\033[2m"
)

var line = strings.Repeat("─", boxWidth)

func main() {
	dictPath := "dictionaries/german_compound_word_components.txt"
	if len(os.Args) > 1 {
		dictPath = os.Args[1]
	}

	// Load tokenizer
	fmt.Print("Loading German compound word components dictionary... ")
	start := time.Now()
	tok, err := tokenizer.NewTokenizer(dictPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer tok.Close()
	fmt.Printf("done (%d words in %v)\n", tok.DictionaryWordCount(), time.Since(start).Round(time.Millisecond))
	fmt.Printf("Iterations: %d (warmup: %d)\n", iterations, warmup)
	fmt.Println("Reference: 1 second = 1,000,000,000 ns")
	fmt.Println()

	// Test data
	singleWord := "Wärmedämmung"
	longCompound := "Wärmedämmverbundsystem"
	sentence := "Der Brandschutzkonzept und die Wärmedämmung der Stahlbetondecke"

	// Full pipeline benchmarks
	printHeader("FULL PIPELINE THROUGHPUT")
	bench("Single word", func() { tok.Tokenize(singleWord) })
	bench("Long compound", func() { tok.Tokenize(longCompound) })
	bench("Sentence (10 words)", func() { tok.Tokenize(sentence) })
	printFooter()
	fmt.Println()

	// Component breakdown
	printHeader("COMPONENT BREAKDOWN")

	bench("Dictionary lookup", func() {
		tok.DictionaryWordCount()
	})

	norm := tokenizer.NewNormalizer()
	bench("Normalizer (full)", func() {
		norm.Normalize("Wärmedämmung")
	})

	bench("Normalizer (lowercase)", func() {
		norm.LowercaseOnly("Wärmedämmung")
	})

	tok.ClearCache()
	tok.Tokenize(singleWord)
	bench("Split (cache hit)", func() {
		tok.Tokenize(singleWord)
	})

	bench("Split (cache miss)", func() {
		tok.ClearCache()
		tok.Tokenize(singleWord)
	})
	printFooter()
	fmt.Println()

	// Normalizer steps
	printHeader("NORMALIZER STEPS BREAKDOWN")
	bench("NFKD decompose", func() {
		tokenizer.NFKDDecompose("Wärmedämmung")
	})
	bench("Remove control chars", func() {
		tokenizer.RemoveControlChars("Wärmedämmung")
	})
	bench("Lowercase", func() {
		tokenizer.Lowercase("Wärmedämmung")
	})
	bench("Normalize quotes", func() {
		tokenizer.NormalizeQuotes("\u201EWärmedämmung\u201C")
	})
	bench("Expand ligatures", func() {
		tokenizer.ExpandLigatures("Wärmedämmung")
	})
	bench("Convert Eszett to ss", func() {
		tokenizer.ConvertEszett("Größe")
	})
	bench("Remove combining marks", func() {
		tokenizer.RemoveCombiningMarks("Wa\u0308rme")
	})
	bench("Stem German", func() {
		tokenizer.StemGerman("warme")
	})
	printFooter()
}

func bench(name string, fn func()) {
	for i := 0; i < warmup; i++ {
		fn()
	}

	start := time.Now()
	for i := 0; i < iterations; i++ {
		fn()
	}
	elapsed := time.Since(start)

	opsPerSec := float64(iterations) / elapsed.Seconds()
	nsPerOp := float64(elapsed.Nanoseconds()) / float64(iterations)

	// Truncate name if too long
	displayName := name
	if len(displayName) > 26 {
		displayName = displayName[:26]
	}

	// Format with colors - build plain string for padding, colored for display
	plain := fmt.Sprintf("  %-26s %10.0f ops/sec %8.0f ns", displayName, opsPerSec, nsPerOp)
	padded := padLine(plain)

	// Now colorize the padded string
	colored := fmt.Sprintf("  %-26s %s%10.0f%s ops/sec %s%8.0f%s ns",
		displayName,
		colorGreen, opsPerSec, colorReset,
		colorYellow, nsPerOp, colorReset)

	// Calculate how much padding we added
	extraPad := len(padded) - len(plain)
	if extraPad > 0 {
		colored += strings.Repeat(" ", extraPad)
	}

	fmt.Println(colorDim + "│" + colorReset + colored + colorDim + "│" + colorReset)
}

func padLine(content string) string {
	if len(content) >= boxWidth {
		return content[:boxWidth]
	}
	return content + strings.Repeat(" ", boxWidth-len(content))
}

func printHeader(title string) {
	fmt.Println(colorDim + "┌" + line + "┐" + colorReset)
	printTitleRow("  " + title)
	fmt.Println(colorDim + "├" + line + "┤" + colorReset)
}

func printFooter() {
	fmt.Println(colorDim + "└" + line + "┘" + colorReset)
}

func printTitleRow(content string) {
	fmt.Println(colorDim + "│" + colorReset + colorCyan + padLine(content) + colorReset + colorDim + "│" + colorReset)
}
