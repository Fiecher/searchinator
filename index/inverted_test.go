package index_test

import (
	"testing"

	"github.com/Fiecher/searchinator"
)

func newDoc(id, text string) searchinator.Document {
	return searchinator.Document{ID: id, Text: text}
}

func mustAdd(t *testing.T, idx *InvertedIndex, doc searchinator.Document, tokens []string) {
	if t != nil {
		t.Helper()
	}
	if err := idx.Add(doc, tokens); err != nil {
		if t != nil {
			t.Fatalf("Add(%q): unexpected error: %v", doc.ID, err)
		}
		panic("mustAdd: " + err.Error())
	}
}

func containsAll(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	set := make(map[string]struct{}, len(got))
	for _, v := range got {
		set[v] = struct{}{}
	}
	for _, v := range want {
		if _, ok := set[v]; !ok {
			return false
		}
	}
	return true
}

func TestInvertedIndex_AddAndGet(t *testing.T) {
	tests := []struct {
		name    string
		docs    []searchinator.Document
		tokens  [][]string
		query   string
		wantIDs []string
	}{
		{
			name:    "single document single term",
			docs:    []searchinator.Document{newDoc("1", "")},
			tokens:  [][]string{{"go"}},
			query:   "go",
			wantIDs: []string{"1"},
		},
		{
			name: "multiple documents share a term",
			docs: []searchinator.Document{
				newDoc("1", ""), newDoc("2", ""), newDoc("3", ""),
			},
			tokens:  [][]string{{"go", "search"}, {"go", "index"}, {"rust"}},
			query:   "go",
			wantIDs: []string{"1", "2"},
		},
		{
			name:    "term not in index",
			docs:    []searchinator.Document{newDoc("1", "")},
			tokens:  [][]string{{"go"}},
			query:   "python",
			wantIDs: []string{},
		},
		{
			name:    "empty index",
			docs:    nil,
			tokens:  nil,
			query:   "go",
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := NewInvertedIndex()
			for i, doc := range tt.docs {
				mustAdd(t, idx, doc, tt.tokens[i])
			}

			got := idx.Get(tt.query)

			if len(got) == 0 && len(tt.wantIDs) == 0 {
				return
			}
			if !containsAll(got, tt.wantIDs) {
				t.Errorf("Get(%q) = %v, want %v", tt.query, got, tt.wantIDs)
			}
		})
	}
}

func TestInvertedIndex_Add_EmptyID(t *testing.T) {
	idx := NewInvertedIndex()
	err := idx.Add(newDoc("", "text"), []string{"text"})
	if err == nil {
		t.Fatal("expected error for empty document ID, got nil")
	}
}

func TestInvertedIndex_ReIndex(t *testing.T) {
	idx := NewInvertedIndex()

	mustAdd(t, idx, newDoc("1", ""), []string{"go", "search"})

	mustAdd(t, idx, newDoc("1", ""), []string{"rust"})

	if ids := idx.Get("go"); len(ids) != 0 {
		t.Errorf("after re-index: Get(\"go\") = %v, want []", ids)
	}
	if ids := idx.Get("search"); len(ids) != 0 {
		t.Errorf("after re-index: Get(\"search\") = %v, want []", ids)
	}

	if ids := idx.Get("rust"); !containsAll(ids, []string{"1"}) {
		t.Errorf("after re-index: Get(\"rust\") = %v, want [1]", ids)
	}

	if n := idx.DocumentCount(); n != 1 {
		t.Errorf("DocumentCount() = %d, want 1", n)
	}
}

func TestInvertedIndex_GetDocument(t *testing.T) {
	idx := NewInvertedIndex()
	doc := newDoc("42", "hello world")
	mustAdd(t, idx, doc, []string{"hello", "world"})

	got, ok := idx.GetDocument("42")
	if !ok {
		t.Fatal("GetDocument(\"42\"): expected true, got false")
	}
	if got.ID != doc.ID || got.Text != doc.Text {
		t.Errorf("GetDocument returned wrong document: %+v", got)
	}

	_, ok = idx.GetDocument("nonexistent")
	if ok {
		t.Error("GetDocument(\"nonexistent\"): expected false, got true")
	}
}

func TestInvertedIndex_DocumentCount(t *testing.T) {
	idx := NewInvertedIndex()

	if n := idx.DocumentCount(); n != 0 {
		t.Errorf("empty index: DocumentCount() = %d, want 0", n)
	}

	mustAdd(t, idx, newDoc("1", ""), []string{"a"})
	mustAdd(t, idx, newDoc("2", ""), []string{"b"})

	if n := idx.DocumentCount(); n != 2 {
		t.Errorf("DocumentCount() = %d, want 2", n)
	}

	mustAdd(t, idx, newDoc("1", ""), []string{"c"})
	if n := idx.DocumentCount(); n != 2 {
		t.Errorf("after re-index: DocumentCount() = %d, want 2", n)
	}
}

func TestInvertedIndex_TermFrequency(t *testing.T) {
	idx := NewInvertedIndex()
	mustAdd(t, idx, newDoc("1", ""), []string{"go", "go", "go", "search"})

	tests := []struct {
		term  string
		docID string
		want  int
	}{
		{"go", "1", 3},
		{"search", "1", 1},
		{"missing", "1", 0},
		{"go", "nonexistent", 0},
	}

	for _, tt := range tests {
		got := idx.TermFrequency(tt.term, tt.docID)
		if got != tt.want {
			t.Errorf("TermFrequency(%q, %q) = %d, want %d", tt.term, tt.docID, got, tt.want)
		}
	}
}

func TestInvertedIndex_DocumentFrequency(t *testing.T) {
	idx := NewInvertedIndex()
	mustAdd(t, idx, newDoc("1", ""), []string{"go", "search"})
	mustAdd(t, idx, newDoc("2", ""), []string{"go"})
	mustAdd(t, idx, newDoc("3", ""), []string{"rust"})

	tests := []struct {
		term string
		want int
	}{
		{"go", 2},
		{"search", 1},
		{"rust", 1},
		{"java", 0},
	}

	for _, tt := range tests {
		got := idx.DocumentFrequency(tt.term)
		if got != tt.want {
			t.Errorf("DocumentFrequency(%q) = %d, want %d", tt.term, got, tt.want)
		}
	}
}

func TestInvertedIndex_AverageDocumentLength(t *testing.T) {
	tests := []struct {
		name  string
		setup func(idx *InvertedIndex)
		want  float64
	}{
		{
			name:  "empty index",
			setup: func(_ *InvertedIndex) {},
			want:  0,
		},
		{
			name: "single document",
			setup: func(idx *InvertedIndex) {
				mustAdd(nil, idx, newDoc("1", ""), []string{"a", "b", "c"})
			},
			want: 3.0,
		},
		{
			name: "two documents same length",
			setup: func(idx *InvertedIndex) {
				mustAdd(nil, idx, newDoc("1", ""), []string{"a", "b"})
				mustAdd(nil, idx, newDoc("2", ""), []string{"c", "d"})
			},
			want: 2.0,
		},
		{
			name: "two documents different lengths",
			setup: func(idx *InvertedIndex) {
				mustAdd(nil, idx, newDoc("1", ""), []string{"a"})
				mustAdd(nil, idx, newDoc("2", ""), []string{"b", "c", "d"})
			},
			want: 2.0,
		},
		{
			name: "average updates after re-index",
			setup: func(idx *InvertedIndex) {
				mustAdd(nil, idx, newDoc("1", ""), []string{"a", "b", "c"})
				mustAdd(nil, idx, newDoc("2", ""), []string{"d", "e"})
				mustAdd(nil, idx, newDoc("1", ""), []string{"x"})
			},
			want: 1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := NewInvertedIndex()
			tt.setup(idx)
			got := idx.AverageDocumentLength()
			if got != tt.want {
				t.Errorf("AverageDocumentLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvertedIndex_DocumentLength(t *testing.T) {
	idx := NewInvertedIndex()
	mustAdd(t, idx, newDoc("1", ""), []string{"a", "b", "c"})
	mustAdd(t, idx, newDoc("2", ""), []string{"x"})

	tests := []struct {
		docID string
		want  int
	}{
		{"1", 3},
		{"2", 1},
		{"nonexistent", 0},
	}

	for _, tt := range tests {
		got := idx.DocumentLength(tt.docID)
		if got != tt.want {
			t.Errorf("DocumentLength(%q) = %d, want %d", tt.docID, got, tt.want)
		}
	}
}

func TestInvertedIndex_Get_NeverReturnsNil(t *testing.T) {
	idx := NewInvertedIndex()
	result := idx.Get("anything")
	if result == nil {
		t.Error("Get() on empty index returned nil, want empty non-nil slice")
	}
}
