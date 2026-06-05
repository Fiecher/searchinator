package ranking_test

import (
	"math"
	"testing"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/ranking"
)

func TestTFIDF_ImplementsRanker(t *testing.T) {
	var _ ranking.Ranker = ranking.NewTFIDF()
}

func TestTFIDF_Score_EmptyQuery(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("1")},
		[][]string{{"go", "search"}},
	)
	if score := ranking.NewTFIDF().Score([]string{}, doc("1"), idx); score != 0 {
		t.Errorf("empty query: Score() = %v, want 0", score)
	}
}

func TestTFIDF_Score_EmptyIndex(t *testing.T) {
	idx := buildIndex(nil, nil)
	if score := ranking.NewTFIDF().Score([]string{"go"}, doc("1"), idx); score != 0 {
		t.Errorf("empty index: Score() = %v, want 0", score)
	}
}

func TestTFIDF_Score_TermAbsent(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("1"), doc("2")},
		[][]string{{"go"}, {"python"}},
	)
	if score := ranking.NewTFIDF().Score([]string{"go"}, doc("2"), idx); score != 0 {
		t.Errorf("absent term: Score() = %v, want 0", score)
	}
}

func TestTFIDF_HigherTF_GivesHigherScore(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("rich"), doc("sparse"), doc("other")},
		[][]string{
			{"go", "go", "go"},
			{"go"},
			{"python"},
		},
	)
	r := ranking.NewTFIDF()
	rich := r.Score([]string{"go"}, doc("rich"), idx)
	sparse := r.Score([]string{"go"}, doc("sparse"), idx)
	if rich <= sparse {
		t.Errorf("higher TF should score higher: rich=%v sparse=%v", rich, sparse)
	}
}

func TestTFIDF_RareTerm_GivesHigherIDF(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("t"), doc("2"), doc("3"), doc("4")},
		[][]string{
			{"rare", "common"},
			{"common"},
			{"common"},
			{"common"},
		},
	)
	r := ranking.NewTFIDF()
	rare := r.Score([]string{"rare"}, doc("t"), idx)
	common := r.Score([]string{"common"}, doc("t"), idx)
	if rare <= common {
		t.Errorf("rare term should score higher: rare=%v common=%v", rare, common)
	}
}

func TestTFIDF_TermInEveryDoc_HasZeroIDF(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("1"), doc("2")},
		[][]string{{"go"}, {"go"}},
	)

	if score := ranking.NewTFIDF().Score([]string{"go"}, doc("1"), idx); score != 0 {
		t.Errorf("term in every doc: Score() = %v, want 0", score)
	}
}

func TestTFIDF_MultiTermQuery_SumsTermScores(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("1"), doc("2")},
		[][]string{{"go", "search"}, {"python"}},
	)
	r := ranking.NewTFIDF()
	a := r.Score([]string{"go"}, doc("1"), idx)
	b := r.Score([]string{"search"}, doc("1"), idx)
	both := r.Score([]string{"go", "search"}, doc("1"), idx)
	if math.Abs(both-(a+b)) > 1e-9 {
		t.Errorf("multi-term: got %v, want %v", both, a+b)
	}
}
