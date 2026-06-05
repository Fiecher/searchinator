package ranking_test

import (
	"testing"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/index"
	"github.com/Fiecher/searchinator/pkg/ranking"
)

type constRanker struct{ v float64 }

func (c constRanker) Score([]string, searchinator.Document, index.Index) float64 { return c.v }

func buildIdx(t *testing.T, docs map[string][]string) *index.InvertedIndex {
	t.Helper()
	idx := index.NewInvertedIndex()
	for id, toks := range docs {
		if err := idx.Add(searchinator.Document{ID: id}, toks); err != nil {
			t.Fatal(err)
		}
	}
	return idx
}

func TestProximity_TightMatchScoresHigher(t *testing.T) {
	idx := buildIdx(t, map[string][]string{

		"a": {"memory", "safety", "rules"},

		"b": {"memory", "x", "y", "z", "safety"},
	})
	r := ranking.NewProximity(constRanker{v: 1.0}, 2.0)

	sa := r.Score([]string{"memory", "safety"}, searchinator.Document{ID: "a"}, idx)
	sb := r.Score([]string{"memory", "safety"}, searchinator.Document{ID: "b"}, idx)

	if !(sa > sb) {
		t.Errorf("adjacent terms should outscore distant: a=%.4f b=%.4f", sa, sb)
	}
}

func TestProximity_FallsBackToBaseWhenSingleTerm(t *testing.T) {
	idx := buildIdx(t, map[string][]string{"a": {"memory", "safety"}})
	r := ranking.NewProximity(constRanker{v: 3.0}, 5.0)
	got := r.Score([]string{"memory"}, searchinator.Document{ID: "a"}, idx)
	if got != 3.0 {
		t.Errorf("single-term query should equal base score 3.0, got %.4f", got)
	}
}

func TestProximity_OnlyOneTermPresent(t *testing.T) {
	idx := buildIdx(t, map[string][]string{"a": {"memory", "rules"}})
	r := ranking.NewProximity(constRanker{v: 1.0}, 5.0)
	got := r.Score([]string{"memory", "safety"}, searchinator.Document{ID: "a"}, idx)
	if got != 1.0 {
		t.Errorf("no bonus when only one query term present, got %.4f", got)
	}
}
