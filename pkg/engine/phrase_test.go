package engine_test

import (
	"testing"

	"github.com/Fiecher/searchinator"
)

func TestEngine_PhraseSearch_MatchesConsecutiveTerms(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "rust achieves memory safety without a garbage collector"},
		{ID: "b", Text: "safety memory is reversed and should not match the phrase"},
		{ID: "c", Text: "go has memory safety in its runtime"},
	})

	results := mustSearch(t, e, `"memory safety"`)
	ids := resultIDs(results)

	got := map[string]bool{}
	for _, id := range ids {
		got[id] = true
	}
	if !got["a"] || !got["c"] {
		t.Errorf("phrase \"memory safety\" should match a and c, got %v", ids)
	}
	if got["b"] {
		t.Errorf("doc b has reversed words and must NOT match phrase, got %v", ids)
	}
}

func TestEngine_PhraseSearch_NoMatch(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "memory safety matters"},
	})

	results := mustSearch(t, e, `"garbage collector"`)
	if len(results) != 0 {
		t.Errorf("expected 0 results for absent phrase, got %d", len(results))
	}
}

func TestEngine_PhraseSearch_SingleWordPhraseEqualsTerm(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "go language"},
		{ID: "b", Text: "rust language"},
	})

	quoted := resultIDs(mustSearch(t, e, `"go"`))
	plain := resultIDs(mustSearch(t, e, "go"))

	if len(quoted) != 1 || quoted[0] != "a" {
		t.Errorf("single-word phrase: got %v, want [a]", quoted)
	}
	if len(plain) != len(quoted) {
		t.Errorf("single-word phrase should equal plain term search: %v vs %v", quoted, plain)
	}
}

func TestEngine_PhraseSearch_RequiredPlusLooseTerm(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "memory safety in rust"},
		{ID: "b", Text: "memory safety in go"},
		{ID: "c", Text: "rust without the required phrase words adjacent: safety then memory"},
	})

	results := mustSearch(t, e, `"memory safety" rust`)
	ids := resultIDs(results)

	if len(ids) != 2 {
		t.Fatalf("expected 2 docs satisfying phrase, got %v", ids)
	}
	if ids[0] != "a" {
		t.Errorf("doc a (phrase + rust) should rank first, got order %v", ids)
	}
}

func TestEngine_PhraseSearch_TwoPhrasesAreANDed(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "memory safety and garbage collector together"},
		{ID: "b", Text: "memory safety but no gc phrase here"},
	})

	results := mustSearch(t, e, `"memory safety" "garbage collector"`)
	ids := resultIDs(results)
	if len(ids) != 1 || ids[0] != "a" {
		t.Errorf("both phrases required (AND): got %v, want [a]", ids)
	}
}
