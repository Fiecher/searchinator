package trie

import (
	"reflect"
	"testing"
)

func TestTrie_WithPrefix(t *testing.T) {
	tr := New()
	for _, w := range []string{"go", "golang", "gopher", "rust", "ruby"} {
		tr.Insert(w)
	}

	got := tr.WithPrefix("go")
	want := []string{"go", "golang", "gopher"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("WithPrefix(go) = %v, want %v", got, want)
	}

	if got := tr.WithPrefix("ru"); !reflect.DeepEqual(got, []string{"ruby", "rust"}) {
		t.Errorf("WithPrefix(ru) = %v, want [ruby rust]", got)
	}

	if got := tr.WithPrefix("zzz"); got != nil {
		t.Errorf("WithPrefix(zzz) = %v, want nil", got)
	}
}

func TestTrie_ContainsAndLen(t *testing.T) {
	tr := New()
	tr.Insert("go")
	tr.Insert("go")
	tr.Insert("golang")

	if tr.Len() != 2 {
		t.Errorf("Len = %d, want 2", tr.Len())
	}
	if !tr.Contains("go") || tr.Contains("g") || tr.Contains("golanggo") {
		t.Error("Contains gave wrong results")
	}
}

func TestTrie_EmptyPrefixReturnsAll(t *testing.T) {
	tr := New()
	for _, w := range []string{"b", "a", "c"} {
		tr.Insert(w)
	}
	if got := tr.WithPrefix(""); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("WithPrefix(empty) = %v, want sorted all", got)
	}
}
