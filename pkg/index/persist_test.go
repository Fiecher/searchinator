package index

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/Fiecher/searchinator"
)

func TestInvertedIndex_SaveLoad_RoundTrip(t *testing.T) {
	src := NewInvertedIndex()
	mustAdd(t, src, searchinator.Document{
		ID: "1", Text: "go rules", Meta: map[string]any{"year": 2009, "tag": "lang"},
	}, []string{"go", "rules", "go"})
	mustAdd(t, src, searchinator.Document{ID: "2", Text: "rust"}, []string{"rust"})

	var buf bytes.Buffer
	if err := src.Save(&buf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	dst := NewInvertedIndex()
	if err := dst.Load(&buf); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if dst.DocumentCount() != 2 {
		t.Errorf("DocumentCount = %d, want 2", dst.DocumentCount())
	}
	if !reflect.DeepEqual(dst.Positions("go", "1"), []int{0, 2}) {
		t.Errorf("positions lost: %v", dst.Positions("go", "1"))
	}
	if tf := dst.TermFrequency("go", "1"); tf != 2 {
		t.Errorf("TermFrequency = %d, want 2", tf)
	}
	doc, ok := dst.GetDocument("1")
	if !ok || doc.Meta["year"] != 2009 || doc.Meta["tag"] != "lang" {
		t.Errorf("meta not preserved: %+v ok=%v", doc.Meta, ok)
	}
	if dst.AverageDocumentLength() != src.AverageDocumentLength() {
		t.Errorf("avg length mismatch: %v vs %v", dst.AverageDocumentLength(), src.AverageDocumentLength())
	}
}

func TestInvertedIndex_LoadInvalidData(t *testing.T) {
	idx := NewInvertedIndex()
	if err := idx.Load(bytes.NewReader([]byte("not gob"))); err == nil {
		t.Error("expected error loading garbage data")
	}
}
