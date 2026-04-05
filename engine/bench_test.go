package engine_test

import (
	"fmt"
	"testing"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/engine"
)

func corpusOfSize(n int) []searchinator.Document {
	words := []string{
		"go", "golang", "search", "engine", "index", "query", "rank",
		"bm25", "token", "filter", "analyzer", "document", "library",
		"fast", "reliable", "efficient", "simple", "language", "program",
	}
	docs := make([]searchinator.Document, n)
	for i := range n {
		w1 := words[i%len(words)]
		w2 := words[(i+3)%len(words)]
		w3 := words[(i+7)%len(words)]
		docs[i] = searchinator.Document{
			ID:   fmt.Sprintf("doc-%d", i),
			Text: fmt.Sprintf("%s %s %s goes well with %s and %s", w1, w2, w3, w1, w2),
		}
	}
	return docs
}

func BenchmarkEngine_Index(b *testing.B) {
	for _, n := range []int{100, 1_000, 10_000} {
		corpus := corpusOfSize(n)
		b.Run(fmt.Sprintf("docs=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				e, _ := engine.NewEngine(engine.DefaultConfig())
				_ = e.Index(corpus)
			}
		})
	}
}

func BenchmarkEngine_Search(b *testing.B) {
	queries := []string{"go", "golang search", "reliable efficient language"}

	for _, n := range []int{100, 1_000, 10_000} {
		corpus := corpusOfSize(n)
		e, _ := engine.NewEngine(engine.DefaultConfig())
		_ = e.Index(corpus)

		for _, q := range queries {
			b.Run(fmt.Sprintf("docs=%d/query=%q", n, q), func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					_, _ = e.Search(q)
				}
			})
		}
	}
}

func BenchmarkEngine_IndexAndSearch(b *testing.B) {
	corpus := corpusOfSize(1_000)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		e, _ := engine.NewEngine(engine.DefaultConfig())
		_ = e.Index(corpus)
		_, _ = e.Search("golang search engine")
	}
}
