package semantic

import (
	"math"
	"testing"

	"github.com/Fiecher/searchinator"
)

func TestHashEmbedderDimAndNormalization(t *testing.T) {
	e := NewHashEmbedder(64)
	if e.Dim() != 64 {
		t.Fatalf("Dim() = %d, want 64", e.Dim())
	}
	vec, err := e.Embed("compiled statically typed language")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 64 {
		t.Fatalf("len(vec) = %d, want 64", len(vec))
	}
	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}
	if math.Abs(sum-1) > 1e-6 {
		t.Fatalf("vector not unit length: ||v||^2 = %v", sum)
	}
}

func TestHashEmbedderEmptyTextIsZero(t *testing.T) {
	vec, err := NewHashEmbedder(32).Embed("   !!!   ")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	for _, v := range vec {
		if v != 0 {
			t.Fatalf("expected zero vector for empty text, got %v", vec)
		}
	}
}

func TestHashEmbedderDeterministic(t *testing.T) {
	e := NewHashEmbedder(128)
	a, _ := e.Embed("go is a compiled language")
	b, _ := e.Embed("go is a compiled language")
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("embeddings not deterministic at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestBruteForceIndexSearchOrdering(t *testing.T) {
	idx := NewBruteForceIndex()

	idx.Add("a", []float32{1, 0})
	idx.Add("b", []float32{0, 1})
	idx.Add("c", []float32{float32(math.Sqrt2 / 2), float32(math.Sqrt2 / 2)})

	got := idx.Search([]float32{1, 0}, 3)
	if len(got) != 3 {
		t.Fatalf("got %d matches, want 3", len(got))
	}
	if got[0].ID != "a" {
		t.Fatalf("top match = %q, want \"a\"", got[0].ID)
	}
	if got[len(got)-1].ID != "b" {
		t.Fatalf("last match = %q, want \"b\"", got[len(got)-1].ID)
	}
	if math.Abs(got[0].Score-1) > 1e-6 {
		t.Fatalf("top score = %v, want 1", got[0].Score)
	}
}

func TestBruteForceIndexTopK(t *testing.T) {
	idx := NewBruteForceIndex()
	idx.Add("a", []float32{1, 0})
	idx.Add("b", []float32{0, 1})
	if got := idx.Search([]float32{1, 0}, 1); len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("top-1 = %+v, want single match \"a\"", got)
	}
}

func TestBruteForceIndexRemove(t *testing.T) {
	idx := NewBruteForceIndex()
	idx.Add("a", []float32{1, 0})
	idx.Add("b", []float32{0, 1})
	idx.Remove("a")
	if idx.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", idx.Len())
	}
	got := idx.Search([]float32{1, 0}, 5)
	if len(got) != 1 || got[0].ID != "b" {
		t.Fatalf("after remove got %+v, want only \"b\"", got)
	}
}

func TestBruteForceIndexDimMismatchSkipped(t *testing.T) {
	idx := NewBruteForceIndex()
	idx.Add("ok", []float32{1, 0})
	idx.Add("bad", []float32{1, 0, 0})
	got := idx.Search([]float32{1, 0}, 5)
	if len(got) != 1 || got[0].ID != "ok" {
		t.Fatalf("mismatched-dim vector not skipped: %+v", got)
	}
}

func TestEngineSearchRanksRelevantHigher(t *testing.T) {
	eng := newTestEngine(t)
	docs := []searchinator.Document{
		{ID: "go", Text: "go is a compiled statically typed language for concurrency"},
		{ID: "python", Text: "python is an interpreted dynamically typed language"},
		{ID: "rust", Text: "rust is a systems language focused on memory safety"},
	}
	if err := eng.Index(docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := eng.Search("compiled concurrency language", 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}
	if results[0].Document.ID != "go" {
		t.Fatalf("top result = %q, want \"go\"", results[0].Document.ID)
	}

	for i := 1; i < len(results); i++ {
		if results[i-1].Score < results[i].Score {
			t.Fatalf("results not sorted descending: %v < %v at %d",
				results[i-1].Score, results[i].Score, i)
		}
	}
}

func TestEngineRankedIDsAndDelete(t *testing.T) {
	eng := newTestEngine(t)
	_ = eng.Index([]searchinator.Document{
		{ID: "a", Text: "alpha beta gamma"},
		{ID: "b", Text: "delta epsilon"},
	})
	ids, err := eng.RankedIDs("alpha beta", 0)
	if err != nil {
		t.Fatalf("RankedIDs: %v", err)
	}
	if len(ids) == 0 || ids[0] != "a" {
		t.Fatalf("RankedIDs top = %v, want \"a\" first", ids)
	}

	eng.Delete("a")
	if eng.DocumentCount() != 1 {
		t.Fatalf("DocumentCount after delete = %d, want 1", eng.DocumentCount())
	}
	ids, _ = eng.RankedIDs("alpha beta", 0)
	for _, id := range ids {
		if id == "a" {
			t.Fatalf("deleted document still returned: %v", ids)
		}
	}
}

func TestNewEngineValidation(t *testing.T) {
	if _, err := NewEngine(nil, NewBruteForceIndex()); err == nil {
		t.Fatal("expected error for nil embedder")
	}
	if _, err := NewEngine(NewHashEmbedder(16), nil); err == nil {
		t.Fatal("expected error for nil index")
	}
}

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	eng, err := NewEngine(NewHashEmbedder(256), NewBruteForceIndex())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return eng
}
