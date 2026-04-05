package engine_test

import (
	"testing"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/analysis"
	"github.com/Fiecher/searchinator/engine"
	"github.com/Fiecher/searchinator/ranking"
)

func defaultEngine(t *testing.T) *engine.Engine {
	t.Helper()
	e, err := engine.NewEngine(engine.DefaultConfig())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return e
}

func mustIndex(t *testing.T, e *engine.Engine, docs []searchinator.Document) {
	t.Helper()
	if err := e.Index(docs); err != nil {
		t.Fatalf("Index: %v", err)
	}
}

func mustSearch(t *testing.T, e *engine.Engine, query string) []searchinator.Result {
	t.Helper()
	results, err := e.Search(query)
	if err != nil {
		t.Fatalf("Search(%q): %v", query, err)
	}
	return results
}

func resultIDs(results []searchinator.Result) []string {
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.Document.ID
	}
	return ids
}

func isSortedDesc(results []searchinator.Result) bool {
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			return false
		}
	}
	return true
}

func TestNewEngine_ValidConfig(t *testing.T) {
	e, err := engine.NewEngine(engine.DefaultConfig())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestNewEngine_InvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  engine.Config
	}{
		{
			name: "nil analyzer",
			cfg: engine.Config{
				Analyzer: nil,
				Ranker:   ranking.NewBM25(ranking.DefaultBM25Params()),
			},
		},
		{
			name: "nil ranker",
			cfg: engine.Config{
				Analyzer: analysis.NewPipelineAnalyzer(analysis.NewWhitespaceTokenizer()),
				Ranker:   nil,
			},
		},
		{
			name: "both nil",
			cfg:  engine.Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := engine.NewEngine(tt.cfg)
			if err == nil {
				t.Error("expected error for invalid config, got nil")
			}
			if e != nil {
				t.Error("expected nil engine for invalid config")
			}
		})
	}
}

func TestEngine_Index_EmptySlice(t *testing.T) {
	e := defaultEngine(t)
	if err := e.Index([]searchinator.Document{}); err != nil {
		t.Errorf("indexing empty slice should not error: %v", err)
	}
}

func TestEngine_Index_EmptyDocumentID(t *testing.T) {
	e := defaultEngine(t)
	err := e.Index([]searchinator.Document{
		{ID: "", Text: "valid text"},
	})
	if err == nil {
		t.Error("expected error for document with empty ID")
	}
}

func TestEngine_Index_Idempotent(t *testing.T) {
	e := defaultEngine(t)

	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "golang is great"},
	})
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "python is great"},
	})

	results := mustSearch(t, e, "golang")
	if len(results) != 0 {
		t.Errorf("after re-index: expected 0 results for 'golang', got %d", len(results))
	}

	results = mustSearch(t, e, "python")
	if len(results) != 1 || results[0].Document.ID != "1" {
		t.Errorf("after re-index: expected doc '1' for 'python', got %v", resultIDs(results))
	}
}

func TestEngine_Search_EmptyQuery(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "hello world"},
	})

	results := mustSearch(t, e, "")
	if len(results) != 0 {
		t.Errorf("empty query: expected 0 results, got %d", len(results))
	}
}

func TestEngine_Search_PunctuationOnlyQuery(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "hello world"},
	})

	results := mustSearch(t, e, "!!! ???")
	if len(results) != 0 {
		t.Errorf("punctuation query: expected 0 results, got %d", len(results))
	}
}

func TestEngine_Search_NoMatchingDocuments(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "golang search engine"},
	})

	results := mustSearch(t, e, "python")
	if len(results) != 0 {
		t.Errorf("no match: expected 0 results, got %d", len(results))
	}
}

func TestEngine_Search_NeverReturnsNil(t *testing.T) {
	e := defaultEngine(t)
	results, err := e.Search("anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results == nil {
		t.Error("Search returned nil, want empty non-nil slice")
	}
}

func TestEngine_Search_EmptyIndex(t *testing.T) {
	e := defaultEngine(t)
	results := mustSearch(t, e, "golang")
	if len(results) != 0 {
		t.Errorf("empty index: expected 0 results, got %d", len(results))
	}
}

func TestEngine_Search_SingleDocument_Matches(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "the quick brown fox"},
	})

	results := mustSearch(t, e, "fox")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Document.ID != "1" {
		t.Errorf("expected doc ID '1', got %q", results[0].Document.ID)
	}
	if results[0].Score <= 0 {
		t.Errorf("expected positive score, got %v", results[0].Score)
	}
}

func TestEngine_Search_CaseInsensitive(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "The Quick Brown Fox"},
	})

	for _, query := range []string{"fox", "FOX", "Fox", "fOx"} {
		results := mustSearch(t, e, query)
		if len(results) != 1 {
			t.Errorf("query %q: expected 1 result, got %d", query, len(results))
		}
	}
}

func TestEngine_Search_MultiTermQuery_UnionOfPostings(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "go-doc", Text: "golang is a compiled language"},
		{ID: "py-doc", Text: "python is an interpreted language"},
		{ID: "rs-doc", Text: "rust is a systems language"},
	})

	results := mustSearch(t, e, "golang python")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(results), resultIDs(results))
	}

	ids := resultIDs(results)
	wantIDs := map[string]bool{"go-doc": true, "py-doc": true}
	for _, id := range ids {
		if !wantIDs[id] {
			t.Errorf("unexpected doc in results: %q", id)
		}
	}
}

func TestEngine_Search_ResultsSortedByScoreDescending(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "go go go go go"},
		{ID: "2", Text: "go programming"},
		{ID: "3", Text: "python ruby java go"},
	})

	results := mustSearch(t, e, "go")
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	if !isSortedDesc(results) {
		t.Errorf("results not sorted by score descending: %v", results)
	}

	if results[0].Document.ID != "1" {
		t.Errorf("expected doc '1' to rank first, got %q", results[0].Document.ID)
	}
}

func TestEngine_Search_TieBreak_ByDocumentID(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "beta", Text: "golang search"},
		{ID: "alpha", Text: "golang search"},
	})

	results := mustSearch(t, e, "golang")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Document.ID != "alpha" {
		t.Errorf("tie-break: expected 'alpha' first, got %q", results[0].Document.ID)
	}
}

func TestEngine_Integration_FullPipeline(t *testing.T) {
	e := defaultEngine(t)

	corpus := []searchinator.Document{
		{
			ID:   "go-spec",
			Text: "Go is an open source programming language that makes it easy to build simple reliable and efficient software",
		},
		{
			ID:   "go-tour",
			Text: "A Tour of Go covers the most important features of the Go programming language Go Go Go",
		},
		{
			ID:   "python-intro",
			Text: "Python is a high level general purpose programming language with dynamic semantics",
		},
		{
			ID:   "rust-book",
			Text: "The Rust programming language helps you write fast reliable software without a garbage collector",
		},
		{
			ID:   "unrelated",
			Text: "Recipes for banana bread and chocolate cake with detailed baking instructions",
		},
	}
	mustIndex(t, e, corpus)

	tests := []struct {
		name        string
		query       string
		wantFirst   string
		wantPresent []string
		wantAbsent  []string
	}{
		{
			name:        "go-specific query ranks go docs first",
			query:       "Go programming language",
			wantFirst:   "go-tour",
			wantPresent: []string{"go-spec", "python-intro", "rust-book"},
			wantAbsent:  []string{"unrelated"},
		},
		{
			name:        "reliable software matches go and rust",
			query:       "reliable software",
			wantPresent: []string{"go-spec", "rust-book"},
			wantAbsent:  []string{"unrelated", "python-intro"},
		},
		{
			name:       "unrelated query returns no results",
			query:      "banana chocolate cake",
			wantFirst:  "unrelated",
			wantAbsent: []string{"go-spec", "go-tour", "python-intro", "rust-book"},
		},
		{
			name:        "case insensitive query",
			query:       "PROGRAMMING LANGUAGE",
			wantPresent: []string{"go-spec", "go-tour", "python-intro", "rust-book"},
			wantAbsent:  []string{"unrelated"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := mustSearch(t, e, tt.query)

			if !isSortedDesc(results) {
				t.Error("results not sorted by score descending")
			}

			if tt.wantFirst != "" {
				if len(results) == 0 {
					t.Fatalf("expected results, got none")
				}
				if results[0].Document.ID != tt.wantFirst {
					t.Errorf("top result: got %q, want %q (scores: %v)",
						results[0].Document.ID, tt.wantFirst, results)
				}
			}

			returned := make(map[string]bool, len(results))
			for _, r := range results {
				returned[r.Document.ID] = true
			}

			for _, id := range tt.wantPresent {
				if !returned[id] {
					t.Errorf("expected doc %q in results, but it was absent", id)
				}
			}
			for _, id := range tt.wantAbsent {
				if returned[id] {
					t.Errorf("expected doc %q to be absent, but it appeared in results", id)
				}
			}
		})
	}
}

func TestEngine_Integration_MetaPreserved(t *testing.T) {
	e := defaultEngine(t)
	doc := searchinator.Document{
		ID:   "1",
		Text: "golang search engine library",
		Meta: map[string]any{
			"author":  "Fiecher",
			"version": 2,
			"tags":    []string{"go", "search"},
		},
	}
	mustIndex(t, e, []searchinator.Document{doc})

	results := mustSearch(t, e, "golang")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0].Document
	if got.Meta["author"] != "Fiecher" {
		t.Errorf("meta['author']: got %v, want 'Fiecher'", got.Meta["author"])
	}
	if got.Meta["version"] != 2 {
		t.Errorf("meta['version']: got %v, want 2", got.Meta["version"])
	}
}

func TestEngine_Integration_CustomConfig(t *testing.T) {
	customAnalyzer := analysis.NewPipelineAnalyzer(
		analysis.NewWhitespaceTokenizer(),
		analysis.NewLowercaseFilter(),
	)
	cfg := engine.Config{
		Analyzer: customAnalyzer,
		Ranker:   ranking.NewBM25(ranking.BM25Params{K1: 2.0, B: 0.5}),
	}
	e, err := engine.NewEngine(cfg)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	mustIndex(t, e, []searchinator.Document{
		{ID: "1", Text: "hello, world"},
	})

	results := mustSearch(t, e, "hello,")
	if len(results) != 1 {
		t.Errorf("custom config: expected 1 result for 'hello,', got %d", len(results))
	}

	results = mustSearch(t, e, "hello")
	if len(results) != 0 {
		t.Errorf("custom config: expected 0 results for 'hello', got %d", len(results))
	}
}
