package index

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Fiecher/searchinator"
)

func tok(text string) []string { return strings.Fields(strings.ToLower(text)) }

func addDoc(t *testing.T, idx interface {
	Add(searchinator.Document, []string) error
}, id, text string) {
	t.Helper()
	d := searchinator.Document{ID: id, Text: text}
	if err := idx.Add(d, tok(text)); err != nil {
		t.Fatalf("Add(%q): %v", id, err)
	}
}

func sortedIDs(ids []string) []string {
	out := append([]string{}, ids...)
	sort.Strings(out)
	return out
}

func TestSegmented_FlushAndReopen(t *testing.T) {
	dir := t.TempDir()

	si, err := OpenSegmented(dir)
	if err != nil {
		t.Fatal(err)
	}
	addDoc(t, si, "a", "go systems programming")
	addDoc(t, si, "b", "python scripting language")
	if err := si.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if si.SegmentCount() != 1 {
		t.Fatalf("SegmentCount = %d, want 1", si.SegmentCount())
	}

	re, err := OpenSegmented(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := re.DocumentCount(); got != 2 {
		t.Errorf("after reopen DocumentCount = %d, want 2", got)
	}
	if got := sortedIDs(re.Get("programming")); len(got) != 1 || got[0] != "a" {
		t.Errorf("Get(programming) = %v, want [a]", got)
	}
	doc, ok := re.GetDocument("b")
	if !ok || doc.Text != "python scripting language" {
		t.Errorf("GetDocument(b) = %+v ok=%v", doc, ok)
	}
}

func TestSegmented_BufferNotDurableUntilFlush(t *testing.T) {
	dir := t.TempDir()
	si, err := OpenSegmented(dir)
	if err != nil {
		t.Fatal(err)
	}
	addDoc(t, si, "a", "unflushed document")

	re, err := OpenSegmented(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := re.DocumentCount(); got != 0 {
		t.Errorf("reopen without flush: DocumentCount = %d, want 0", got)
	}
}

func TestSegmented_DeleteTombstonePersists(t *testing.T) {
	dir := t.TempDir()
	si, err := OpenSegmented(dir)
	if err != nil {
		t.Fatal(err)
	}
	addDoc(t, si, "a", "alpha shared")
	addDoc(t, si, "b", "beta shared")
	if err := si.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := si.Remove("a"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if got := si.DocumentCount(); got != 1 {
		t.Errorf("after delete DocumentCount = %d, want 1", got)
	}
	if got := si.Get("shared"); len(got) != 1 || got[0] != "b" {
		t.Errorf("Get(shared) = %v, want [b]", got)
	}
	if df := si.DocumentFrequency("shared"); df != 1 {
		t.Errorf("DocumentFrequency(shared) = %d, want 1", df)
	}

	re, err := OpenSegmented(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := re.DocumentCount(); got != 1 {
		t.Errorf("reopen DocumentCount = %d, want 1", got)
	}
	if _, ok := re.GetDocument("a"); ok {
		t.Error("deleted document a reappeared after reopen")
	}
}

func TestSegmented_ReindexOverride(t *testing.T) {
	dir := t.TempDir()
	si, err := OpenSegmented(dir)
	if err != nil {
		t.Fatal(err)
	}
	addDoc(t, si, "a", "first version compiled")
	if err := si.Flush(); err != nil {
		t.Fatal(err)
	}

	addDoc(t, si, "a", "second version interpreted")

	if got := si.DocumentCount(); got != 1 {
		t.Errorf("DocumentCount = %d, want 1 (override, not duplicate)", got)
	}
	if got := si.Get("compiled"); len(got) != 0 {
		t.Errorf("Get(compiled) = %v, want [] (old version shadowed)", got)
	}
	if got := si.Get("interpreted"); len(got) != 1 || got[0] != "a" {
		t.Errorf("Get(interpreted) = %v, want [a]", got)
	}
	if df := si.DocumentFrequency("version"); df != 1 {
		t.Errorf("DocumentFrequency(version) = %d, want 1", df)
	}

	if err := si.Flush(); err != nil {
		t.Fatal(err)
	}
	re, err := OpenSegmented(dir)
	if err != nil {
		t.Fatal(err)
	}
	if doc, ok := re.GetDocument("a"); !ok || doc.Text != "second version interpreted" {
		t.Errorf("after reopen GetDocument(a) = %+v ok=%v", doc, ok)
	}
	if got := re.DocumentCount(); got != 1 {
		t.Errorf("after reopen DocumentCount = %d, want 1", got)
	}
}

func TestSegmented_ParityWithInverted(t *testing.T) {
	docs := []struct{ id, text string }{
		{"go", "go systems programming concurrency compiled"},
		{"py", "python scripting language readable programming"},
		{"rs", "rust systems memory safety concurrency compiled"},
		{"hs", "haskell functional programming lazy pure"},
		{"c", "c systems programming memory low level"},
	}

	ref := NewInvertedIndex()
	dir := t.TempDir()
	seg, err := OpenSegmented(dir)
	if err != nil {
		t.Fatal(err)
	}

	for i, d := range docs {
		ref.Add(searchinator.Document{ID: d.id, Text: d.text}, tok(d.text))
		addDoc(t, seg, d.id, d.text)
		if i%2 == 1 {
			if err := seg.Flush(); err != nil {
				t.Fatal(err)
			}
		}
	}

	if ref.DocumentCount() != seg.DocumentCount() {
		t.Fatalf("DocumentCount: ref=%d seg=%d", ref.DocumentCount(), seg.DocumentCount())
	}
	if ref.AverageDocumentLength() != seg.AverageDocumentLength() {
		t.Errorf("AvgDocLen: ref=%v seg=%v", ref.AverageDocumentLength(), seg.AverageDocumentLength())
	}

	terms := []string{"programming", "systems", "concurrency", "memory", "compiled", "functional", "missing"}
	for _, term := range terms {
		if ref.DocumentFrequency(term) != seg.DocumentFrequency(term) {
			t.Errorf("DF(%q): ref=%d seg=%d", term, ref.DocumentFrequency(term), seg.DocumentFrequency(term))
		}
		gotRef := sortedIDs(ref.Get(term))
		gotSeg := sortedIDs(seg.Get(term))
		if strings.Join(gotRef, ",") != strings.Join(gotSeg, ",") {
			t.Errorf("Get(%q): ref=%v seg=%v", term, gotRef, gotSeg)
		}
		for _, d := range docs {
			if ref.TermFrequency(term, d.id) != seg.TermFrequency(term, d.id) {
				t.Errorf("TF(%q,%q): ref=%d seg=%d", term, d.id,
					ref.TermFrequency(term, d.id), seg.TermFrequency(term, d.id))
			}
		}
	}
}

func TestSegmented_Concurrent(t *testing.T) {
	dir := t.TempDir()
	si, err := OpenSegmented(dir)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			id := fmt.Sprintf("d%d", i)
			_ = si.Add(searchinator.Document{ID: id}, tok("programming systems concurrency"))
			if i%25 == 0 {
				_ = si.Flush()
			}
		}
	}()

	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				_ = si.Get("programming")
				_ = si.DocumentCount()
				_ = si.DocumentFrequency("systems")
			}
		}()
	}

	wg.Wait()
	if err := si.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := si.DocumentCount(); got != 200 {
		t.Errorf("DocumentCount = %d, want 200", got)
	}
	if got := len(si.Get("programming")); got != 200 {
		t.Errorf("Get(programming) returned %d docs, want 200", got)
	}
}
