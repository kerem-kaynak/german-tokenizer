package tokenizer

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// fakeSource captures Save calls and returns canned Load output.
type fakeSource struct {
	lines     []string
	saveCalls [][]string
	loadErr   error
	saveErr   error
}

func (f *fakeSource) Load() ([]string, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	out := make([]string, len(f.lines))
	copy(out, f.lines)
	return out, nil
}

func (f *fakeSource) Save(words []string) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	captured := make([]string, len(words))
	copy(captured, words)
	f.saveCalls = append(f.saveCalls, captured)
	return nil
}

func TestNewDictionaryWithSource_LoadsWords(t *testing.T) {
	src := &fakeSource{lines: []string{"# header", "", "haus", "Wand", "decke"}}
	fstPath := filepath.Join(t.TempDir(), "test.fst")

	dict, err := NewDictionaryWithSource(src, fstPath)
	if err != nil {
		t.Fatalf("NewDictionaryWithSource: %v", err)
	}
	defer dict.Close()

	if !dict.Contains("haus") {
		t.Error("expected 'haus' in dict")
	}
	if !dict.Contains("wand") {
		t.Error("expected 'wand' in dict (lowercased from 'Wand')")
	}
	if !dict.Contains("decke") {
		t.Error("expected 'decke' in dict")
	}
	if dict.Contains("nope") {
		t.Error("'nope' should not be in dict")
	}

	// rebuildFST during construction (no FST file existed) calls Save once.
	if len(src.saveCalls) != 1 {
		t.Fatalf("expected 1 Save call during construction, got %d", len(src.saveCalls))
	}
	got := src.saveCalls[0]
	want := []string{"decke", "haus", "wand"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Save got %v, want %v", got, want)
	}
}

func TestNewDictionaryWithSource_AddWordPersists(t *testing.T) {
	src := &fakeSource{lines: []string{"haus", "wand"}}
	fstPath := filepath.Join(t.TempDir(), "test.fst")

	dict, err := NewDictionaryWithSource(src, fstPath)
	if err != nil {
		t.Fatalf("NewDictionaryWithSource: %v", err)
	}
	defer dict.Close()

	before := len(src.saveCalls)
	if err := dict.AddWord("Brandschutz"); err != nil {
		t.Fatalf("AddWord: %v", err)
	}

	if len(src.saveCalls) != before+1 {
		t.Fatalf("expected one additional Save after AddWord, got %d (before=%d)",
			len(src.saveCalls), before)
	}

	last := src.saveCalls[len(src.saveCalls)-1]
	if !sort.StringsAreSorted(last) {
		t.Errorf("Save payload not sorted: %v", last)
	}
	found := false
	for _, w := range last {
		if w == "brandschutz" {
			found = true
		}
	}
	if !found {
		t.Errorf("AddWord('Brandschutz') did not appear (lowercased) in Save payload: %v", last)
	}
	if !dict.Contains("brandschutz") {
		t.Error("Contains('brandschutz') = false after AddWord")
	}
}

func TestNewDictionaryWithSource_RemoveWordPersists(t *testing.T) {
	src := &fakeSource{lines: []string{"haus", "wand", "decke"}}
	fstPath := filepath.Join(t.TempDir(), "test.fst")

	dict, err := NewDictionaryWithSource(src, fstPath)
	if err != nil {
		t.Fatalf("NewDictionaryWithSource: %v", err)
	}
	defer dict.Close()

	if err := dict.RemoveWord("wand"); err != nil {
		t.Fatalf("RemoveWord: %v", err)
	}

	last := src.saveCalls[len(src.saveCalls)-1]
	for _, w := range last {
		if w == "wand" {
			t.Errorf("RemoveWord('wand') still present in Save payload: %v", last)
		}
	}
	if dict.Contains("wand") {
		t.Error("Contains('wand') = true after RemoveWord")
	}
}
