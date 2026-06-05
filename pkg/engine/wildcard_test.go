package engine_test

import (
	"sort"
	"testing"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/engine"
)

func TestEngine_SearchWildcard_Prefix(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "programming language"},
		{ID: "b", Text: "program files"},
		{ID: "c", Text: "rust safety"},
	})

	got := resultIDs(mustWildcard(t, e, "program*"))
	sort.Strings(got)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("program* = %v, want [a b]", got)
	}
}

func TestEngine_SearchWildcard_QuestionMark(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "cat"},
		{ID: "b", Text: "cot"},
		{ID: "c", Text: "cart"},
	})
	got := resultIDs(mustWildcard(t, e, "c?t"))
	sort.Strings(got)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("c?t = %v, want [a b]", got)
	}
}

func TestEngine_SearchWildcard_RebuildsAfterDelete(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "programming"},
		{ID: "b", Text: "program"},
	})

	_ = mustWildcard(t, e, "program*")

	if err := e.Delete("a"); err != nil {
		t.Fatal(err)
	}
	got := resultIDs(mustWildcard(t, e, "program*"))
	if len(got) != 1 || got[0] != "b" {
		t.Errorf("after delete, program* = %v, want [b]", got)
	}
}

func mustWildcard(t *testing.T, e *engine.Engine, pattern string) []searchinator.Result {
	t.Helper()
	res, err := e.SearchWildcard(pattern)
	if err != nil {
		t.Fatalf("SearchWildcard(%q): %v", pattern, err)
	}
	return res
}
