package index

import (
	"sync"
	"testing"

	"github.com/Fiecher/searchinator"
)

func TestInvertedIndex_Remove(t *testing.T) {
	t.Run("removes existing document", func(t *testing.T) {
		idx := NewInvertedIndex()
		mustAdd(t, idx, newDoc("1", ""), []string{"go", "search"})

		if err := idx.Remove("1"); err != nil {
			t.Fatalf("Remove: unexpected error: %v", err)
		}

		if n := idx.DocumentCount(); n != 0 {
			t.Errorf("DocumentCount() = %d, want 0", n)
		}
		if ids := idx.Get("go"); len(ids) != 0 {
			t.Errorf("Get(\"go\") after Remove = %v, want []", ids)
		}
	})

	t.Run("returns error for missing document", func(t *testing.T) {
		idx := NewInvertedIndex()
		if err := idx.Remove("nonexistent"); err == nil {
			t.Error("expected error for missing document, got nil")
		}
	})

	t.Run("cleans up empty postings entries", func(t *testing.T) {
		idx := NewInvertedIndex()
		mustAdd(t, idx, newDoc("1", ""), []string{"unique"})
		_ = idx.Remove("1")

		if ids := idx.Get("unique"); len(ids) != 0 {
			t.Errorf("Get(\"unique\") after Remove = %v, want []", ids)
		}
	})

	t.Run("totalTokens updated correctly", func(t *testing.T) {
		idx := NewInvertedIndex()
		mustAdd(t, idx, newDoc("1", ""), []string{"a", "b", "c"})
		mustAdd(t, idx, newDoc("2", ""), []string{"d", "e"})

		_ = idx.Remove("1")

		if avg := idx.AverageDocumentLength(); avg != 2.0 {
			t.Errorf("AverageDocumentLength() after Remove = %v, want 2.0", avg)
		}
	})
}

func TestInvertedIndex_Terms(t *testing.T) {
	t.Run("empty index returns empty slice", func(t *testing.T) {
		idx := NewInvertedIndex()
		terms := idx.Terms()
		if terms == nil {
			t.Error("Terms() returned nil, want empty non-nil slice")
		}
		if len(terms) != 0 {
			t.Errorf("Terms() = %v, want []", terms)
		}
	})

	t.Run("returns all indexed terms", func(t *testing.T) {
		idx := NewInvertedIndex()
		mustAdd(t, idx, newDoc("1", ""), []string{"go", "search", "index"})
		mustAdd(t, idx, newDoc("2", ""), []string{"go", "rust"})

		terms := idx.Terms()
		want := map[string]bool{"go": true, "search": true, "index": true, "rust": true}

		if len(terms) != len(want) {
			t.Errorf("Terms() count = %d, want %d", len(terms), len(want))
		}
		for _, term := range terms {
			if !want[term] {
				t.Errorf("unexpected term in Terms(): %q", term)
			}
		}
	})

	t.Run("updates after remove", func(t *testing.T) {
		idx := NewInvertedIndex()
		mustAdd(t, idx, newDoc("1", ""), []string{"only"})
		_ = idx.Remove("1")

		terms := idx.Terms()
		if len(terms) != 0 {
			t.Errorf("Terms() after Remove = %v, want []", terms)
		}
	})
}

func TestInvertedIndex_ConcurrentReadWrite(t *testing.T) {
	idx := NewInvertedIndex()

	for i := 0; i < 10; i++ {
		id := string(rune('a' + i))
		mustAdd(t, idx, searchinator.Document{ID: id, Text: ""}, []string{"term"})
	}

	var wg sync.WaitGroup
	const goroutines = 20

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = idx.Get("term")
			_ = idx.DocumentCount()
			_ = idx.AverageDocumentLength()
			_ = idx.Terms()
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = idx.Add(searchinator.Document{ID: "z", Text: ""}, []string{"term"})
	}()

	wg.Wait()
}
