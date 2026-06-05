package engine_test

import (
	"fmt"
	"testing"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/engine"
)

func TestEngine_SearchParallel_MatchesSearch(t *testing.T) {
	e := defaultEngine(t)
	docs := make([]searchinator.Document, 0, 200)
	for i := 0; i < 200; i++ {
		docs = append(docs, searchinator.Document{
			ID:   fmt.Sprintf("d%d", i),
			Text: "go rust memory safety concurrency systems programming language",
		})
	}
	mustIndex(t, e, docs)

	seq := resultIDs(mustSearch(t, e, "memory safety"))
	par, err := e.SearchParallel("memory safety", 8)
	if err != nil {
		t.Fatal(err)
	}
	got := resultIDs(par)

	if len(seq) != len(got) {
		t.Fatalf("result count differs: seq=%d par=%d", len(seq), len(got))
	}
	for i := range seq {
		if seq[i] != got[i] {
			t.Errorf("order differs at %d: seq=%s par=%s", i, seq[i], got[i])
		}
	}
}

func TestEngine_SearchParallel_EmptyQuery(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{{ID: "a", Text: "go"}})
	res, err := e.SearchParallel("", 4)
	if err != nil || len(res) != 0 {
		t.Errorf("empty query: got %v err=%v", res, err)
	}
}

const parBenchQuery = "golang search engine reliable efficient language"

func benchEngine(b *testing.B) *engine.Engine {
	b.Helper()
	e, _ := engine.NewEngine(engine.DefaultConfig())
	_ = e.Index(corpusOfSize(10_000))
	return e
}

func BenchmarkSearch_Sequential(b *testing.B) {
	e := benchEngine(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Search(parBenchQuery)
	}
}

func BenchmarkSearch_Parallel(b *testing.B) {
	e := benchEngine(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.SearchParallel(parBenchQuery, 8)
	}
}
