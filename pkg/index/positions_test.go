package index

import (
	"reflect"
	"testing"
)

func TestInvertedIndex_Positions(t *testing.T) {
	idx := NewInvertedIndex()
	mustAdd(t, idx, newDoc("1", ""), []string{"go", "is", "go", "fast", "go"})

	tests := []struct {
		term  string
		docID string
		want  []int
	}{
		{"go", "1", []int{0, 2, 4}},
		{"is", "1", []int{1}},
		{"fast", "1", []int{3}},
		{"missing", "1", []int{}},
		{"go", "nonexistent", []int{}},
	}

	for _, tt := range tests {
		got := idx.Positions(tt.term, tt.docID)
		if len(got) == 0 && len(tt.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Positions(%q, %q) = %v, want %v", tt.term, tt.docID, got, tt.want)
		}
	}
}

func TestInvertedIndex_Positions_ReturnsCopy(t *testing.T) {
	idx := NewInvertedIndex()
	mustAdd(t, idx, newDoc("1", ""), []string{"a", "a", "a"})

	got := idx.Positions("a", "1")
	got[0] = 999

	again := idx.Positions("a", "1")
	if again[0] == 999 {
		t.Error("Positions returned a slice aliasing internal state; expected a copy")
	}
}

func TestInvertedIndex_Positions_TermFrequencyConsistent(t *testing.T) {
	idx := NewInvertedIndex()
	mustAdd(t, idx, newDoc("1", ""), []string{"go", "go", "go", "rust"})

	if tf, np := idx.TermFrequency("go", "1"), len(idx.Positions("go", "1")); tf != np {
		t.Errorf("TermFrequency=%d but len(Positions)=%d, want equal", tf, np)
	}
}
