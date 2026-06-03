package tokenizer

import (
	"bufio"
	"io"
	"os"
	"sort"
	"strings"
)

// dictSource is the single seam for where the dictionary word list is read from
// and written to. Today there are two implementations: a local file and S3.
type dictSource interface {
	load() ([]string, error)   // lowercased words; blank and "#" comment lines skipped
	save(words []string) error // persist the full word list
}

// parseDictLines reads dictionary words from r using the canonical format:
// one word per line, blank and "#"-comment lines skipped, words lowercased.
// Shared by file and S3 sources so the on-disk format never drifts.
func parseDictLines(r io.Reader) ([]string, error) {
	var words []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word == "" || strings.HasPrefix(word, "#") {
			continue
		}
		words = append(words, strings.ToLower(word))
	}
	return words, scanner.Err()
}

// serializeDict produces the on-disk byte form: sorted words, one per line,
// trailing newline. Byte-identical to the previous saveTextFile output, so
// file-mode round-trips produce no diff.
func serializeDict(words []string) []byte {
	sorted := append([]string(nil), words...)
	sort.Strings(sorted)

	var b strings.Builder
	for _, w := range sorted {
		b.WriteString(w)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

// fileSource reads and writes the dictionary as a local text file.
type fileSource struct {
	path string
}

func (f fileSource) load() ([]string, error) {
	file, err := os.Open(f.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseDictLines(file)
}

func (f fileSource) save(words []string) error {
	return os.WriteFile(f.path, serializeDict(words), 0o644)
}
