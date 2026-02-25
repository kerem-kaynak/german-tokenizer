package main

import (
	"fmt"
	"os"

	"github.com/kerem-kaynak/german-tokenizer/pkg/tokenizer"
)

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	dictPath := os.Args[1]
	command := os.Args[2]

	dict, err := tokenizer.NewDictionary(dictPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading dictionary: %v\n", err)
		os.Exit(1)
	}
	defer dict.Close()

	switch command {
	case "add":
		if len(os.Args) < 4 {
			fmt.Println("Error: add requires at least one word")
			os.Exit(1)
		}
		for _, word := range os.Args[3:] {
			if err := dict.AddWord(word); err != nil {
				fmt.Fprintf(os.Stderr, "Error adding word '%s': %v\n", word, err)
				os.Exit(1)
			}
			fmt.Printf("Added: %s\n", word)
		}
		fmt.Printf("Total words: %d\n", dict.WordCount())

	case "remove":
		if len(os.Args) < 4 {
			fmt.Println("Error: remove requires at least one word")
			os.Exit(1)
		}
		for _, word := range os.Args[3:] {
			if err := dict.RemoveWord(word); err != nil {
				fmt.Fprintf(os.Stderr, "Error removing word '%s': %v\n", word, err)
				os.Exit(1)
			}
			fmt.Printf("Removed: %s\n", word)
		}
		fmt.Printf("Total words: %d\n", dict.WordCount())

	case "contains":
		if len(os.Args) < 4 {
			fmt.Println("Error: contains requires a word")
			os.Exit(1)
		}
		word := os.Args[3]
		if dict.Contains(word) {
			fmt.Printf("'%s' exists in dictionary\n", word)
		} else {
			fmt.Printf("'%s' NOT in dictionary\n", word)
			os.Exit(1)
		}

	case "rebuild":
		if err := dict.RebuildFST(); err != nil {
			fmt.Fprintf(os.Stderr, "Error rebuilding FST: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("FST rebuilt. Total words: %d\n", dict.WordCount())

	case "stats":
		fmt.Printf("Dictionary: %s\n", dictPath)
		fmt.Printf("Word count: %d\n", dict.WordCount())

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: dictmgr <dictionary.txt> <command> [args...]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  add <word> [word...]    Add words to dictionary")
	fmt.Println("  remove <word> [word...] Remove words from dictionary")
	fmt.Println("  contains <word>         Check if word exists")
	fmt.Println("  rebuild                 Rebuild FST from text file")
	fmt.Println("  stats                   Show dictionary statistics")
}
