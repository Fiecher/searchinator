package engine_test

import (
	"testing"

	"github.com/Fiecher/searchinator"
)

func boolCorpus() []searchinator.Document {
	return []searchinator.Document{
		{ID: "a", Text: "go has memory safety", Meta: map[string]any{"year": 2012, "tag": "go"}},
		{ID: "b", Text: "rust has memory safety without gc", Meta: map[string]any{"year": 2015, "tag": "rust"}},
		{ID: "c", Text: "go is fast and simple", Meta: map[string]any{"year": 2009, "tag": "go"}},
		{ID: "d", Text: "python is simple and slow", Meta: map[string]any{"year": 2020, "tag": "python"}},
	}
}

func mustSearchBool(t *testing.T, e interface {
	SearchBool(string) ([]searchinator.Result, error)
}, q string) []string {
	t.Helper()
	results, err := e.SearchBool(q)
	if err != nil {
		t.Fatalf("SearchBool(%q): %v", q, err)
	}
	return resultIDs(results)
}

func TestEngine_SearchBool_AND(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, boolCorpus())
	if got := mustSearchBool(t, e, "go AND safety"); len(got) != 1 || got[0] != "a" {
		t.Errorf("go AND safety = %v, want [a]", got)
	}
}

func TestEngine_SearchBool_NOT(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, boolCorpus())
	if got := mustSearchBool(t, e, "simple AND NOT go"); len(got) != 1 || got[0] != "d" {
		t.Errorf("simple AND NOT go = %v, want [d]", got)
	}
}

func TestEngine_SearchBool_Filter(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, boolCorpus())
	got := mustSearchBool(t, e, "safety AND year<2015")
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("safety AND year<2015 = %v, want [a]", got)
	}
}

func TestEngine_SearchBool_RanksByPositiveTerms(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "memory safety memory safety memory"},
		{ID: "b", Text: "memory safety once"},
	})

	got := mustSearchBool(t, e, "memory OR safety")
	if len(got) != 2 || got[0] != "a" {
		t.Errorf("ranking order = %v, want a first", got)
	}
}

func TestEngine_SearchBool_NoMatch(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, boolCorpus())
	if got := mustSearchBool(t, e, "go AND rust"); len(got) != 0 {
		t.Errorf("go AND rust = %v, want []", got)
	}
}

func TestEngine_SearchBool_ParseError(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, boolCorpus())
	if _, err := e.SearchBool("(go OR"); err == nil {
		t.Error("expected parse error for unbalanced parenthesis")
	}
}
