package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kerem-kaynak/german-tokenizer/pkg/tokenizer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tokenize <dictionary_path> [text]")
		fmt.Println("       tokenize <dictionary_path>          (interactive mode)")
		os.Exit(1)
	}

	dictPath := os.Args[1]

	tok, err := tokenizer.NewTokenizer(dictPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading dictionary: %v\n", err)
		os.Exit(1)
	}
	defer tok.Close()

	// If text provided as argument, tokenize and exit
	if len(os.Args) > 2 {
		text := strings.Join(os.Args[2:], " ")
		tokens := tok.Tokenize(text)
		output, _ := json.Marshal(tokens)
		fmt.Println(string(output))
		return
	}

	// Interactive mode
	fmt.Println("German Tokenizer (interactive mode)")
	fmt.Printf("Dictionary loaded: %d words\n", tok.DictionaryWordCount())
	fmt.Println("Type a word or sentence, press Enter to tokenize. Ctrl+C to exit.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		text := scanner.Text()
		if text == "" {
			continue
		}

		tokens := tok.Tokenize(text)
		output, _ := json.Marshal(tokens)
		fmt.Printf("  %s\n\n", output)
	}
}
