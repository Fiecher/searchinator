package engine_test

import (
	"testing"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/engine"
)

func TestEngine_Delete_ExistingDocument(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "golang search engine"},
	})

	if err := e.Delete("1"); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	results := mustSearch(t, e, "golang")
	if len(results) != 0 {
		t.Errorf("after Delete: expected 0 results, got %d", len(results))
	}
}

func TestEngine_Delete_MissingDocument(t *testing.T) {
	e := defaultEngine(t)
	if err := e.Delete("nonexistent"); err == nil {
		t.Error("expected error when deleting nonexistent document, got nil")
	}
}

func TestEngine_Delete_UpdatesStats(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "golang"},
		{ID: "2", Text: "python"},
	})

	_ = e.Delete("1")
	s := e.Stats()
	if s.DocumentCount != 1 {
		t.Errorf("Stats.DocumentCount after Delete = %d, want 1", s.DocumentCount)
	}
}

func TestEngine_Stats_EmptyIndex(t *testing.T) {
	e := defaultEngine(t)
	s := e.Stats()

	if s.DocumentCount != 0 {
		t.Errorf("DocumentCount = %d, want 0", s.DocumentCount)
	}
	if s.TermCount != 0 {
		t.Errorf("TermCount = %d, want 0", s.TermCount)
	}
	if s.AverageDocumentLength != 0 {
		t.Errorf("AverageDocumentLength = %v, want 0", s.AverageDocumentLength)
	}
}

func TestEngine_Stats_AfterIndexing(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "go golang"},
		{ID: "2", Text: "python ruby"},
	})

	s := e.Stats()
	if s.DocumentCount != 2 {
		t.Errorf("DocumentCount = %d, want 2", s.DocumentCount)
	}
	if s.TermCount == 0 {
		t.Error("TermCount should be > 0 after indexing")
	}
	if s.AverageDocumentLength != 2.0 {
		t.Errorf("AverageDocumentLength = %v, want 2.0", s.AverageDocumentLength)
	}
}

func TestEngine_SearchN_LimitResults(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "go go go"},
		{ID: "2", Text: "go language"},
		{ID: "3", Text: "go program"},
	})

	results, err := e.SearchN("go", 2)
	if err != nil {
		t.Fatalf("SearchN: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("SearchN(limit=2) returned %d results, want 2", len(results))
	}
}

func TestEngine_SearchN_ZeroLimitReturnsAll(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "go go go"},
		{ID: "2", Text: "go language"},
		{ID: "3", Text: "go program"},
	})

	all := mustSearch(t, e, "go")
	limited, err := e.SearchN("go", 0)
	if err != nil {
		t.Fatalf("SearchN: %v", err)
	}
	if len(limited) != len(all) {
		t.Errorf("SearchN(limit=0) returned %d results, want %d (all)", len(limited), len(all))
	}
}

func TestEngine_SearchN_LimitLargerThanResults(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "go language"},
	})

	results, err := e.SearchN("go", 100)
	if err != nil {
		t.Fatalf("SearchN: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchN(limit=100) returned %d results, want 1", len(results))
	}
}

func TestEngine_SearchN_ResultsSortedDescending(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "go go go go go"},
		{ID: "2", Text: "go language"},
		{ID: "3", Text: "go program go"},
	})

	results, err := e.SearchN("go", 2)
	if err != nil {
		t.Fatalf("SearchN: %v", err)
	}
	if !isSortedDesc(results) {
		t.Error("SearchN results not sorted by score descending")
	}
}

func TestVocabularyFromIndex_EmptyEngine(t *testing.T) {
	e := defaultEngine(t)
	vocab := engine.VocabularyFromIndex(e)
	if vocab == nil {
		t.Error("VocabularyFromIndex returned nil, want empty non-nil slice")
	}
	if len(vocab) != 0 {
		t.Errorf("VocabularyFromIndex on empty engine = %v, want []", vocab)
	}
}

func TestVocabularyFromIndex_ContainsIndexedTerms(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "golang search"},
		{ID: "2", Text: "python index"},
	})

	vocab := engine.VocabularyFromIndex(e)
	want := map[string]bool{"golang": true, "search": true, "python": true, "index": true}

	if len(vocab) != len(want) {
		t.Errorf("VocabularyFromIndex count = %d, want %d", len(vocab), len(want))
	}
	for _, term := range vocab {
		if !want[term] {
			t.Errorf("unexpected term in vocabulary: %q", term)
		}
	}
}

func TestFuzzyConfig_FindsTypos(t *testing.T) {
	plain, err := engine.NewEngine(engine.DefaultConfig())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	docs := []searchinator.Document{
		{ID: "go", Text: "Go is a compiled statically typed language"},
		{ID: "rust", Text: "Rust is a systems language for memory safety"},
	}
	mustIndex(t, plain, docs)

	vocab := engine.VocabularyFromIndex(plain)
	fe, err := engine.NewEngine(engine.FuzzyConfig(vocab, 1))
	if err != nil {
		t.Fatalf("NewEngine FuzzyConfig: %v", err)
	}
	mustIndex(t, fe, docs)

	results, err := fe.Search("compied")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("FuzzyConfig: expected results for typo 'compied', got none")
	}
	found := false
	for _, r := range results {
		if r.Document.ID == "go" {
			found = true
		}
	}
	if !found {
		t.Error("FuzzyConfig: expected 'go' doc in results for 'compied' (typo of 'compiled')")
	}
}

func TestFuzzyConfig_ExactMatchStillWorks(t *testing.T) {
	plain, err := engine.NewEngine(engine.DefaultConfig())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	docs := []searchinator.Document{
		{ID: "go", Text: "Go is a compiled language"},
	}
	mustIndex(t, plain, docs)

	vocab := engine.VocabularyFromIndex(plain)
	fe, err := engine.NewEngine(engine.FuzzyConfig(vocab, 1))
	if err != nil {
		t.Fatalf("NewEngine FuzzyConfig: %v", err)
	}
	mustIndex(t, fe, docs)

	results, err := fe.Search("compiled")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].Document.ID != "go" {
		t.Errorf("FuzzyConfig exact match: got %v, want [go]", resultIDs(results))
	}
}
