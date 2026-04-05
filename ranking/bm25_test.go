package ranking_test

import (
	"math"
	"testing"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/index"
	"github.com/Fiecher/searchinator/ranking"
)

func buildIndex(docs []searchinator.Document, tokens [][]string) *index.InvertedIndex {
	idx := index.NewInvertedIndex()
	for i, doc := range docs {
		if err := idx.Add(doc, tokens[i]); err != nil {
			panic("buildIndex: " + err.Error())
		}
	}
	return idx
}

func doc(id string) searchinator.Document {
	return searchinator.Document{ID: id}
}

func TestBM25_ImplementsRanker(t *testing.T) {
	var _ ranking.Ranker = ranking.NewBM25(ranking.DefaultBM25Params())
}

func TestBM25_Score_EmptyQuery(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("1")},
		[][]string{{"go", "search"}},
	)
	ranker := ranking.NewBM25(ranking.DefaultBM25Params())
	score := ranker.Score([]string{}, doc("1"), idx)
	if score != 0 {
		t.Errorf("empty query: Score() = %v, want 0", score)
	}
}

func TestBM25_Score_EmptyIndex(t *testing.T) {
	idx := index.NewInvertedIndex()
	ranker := ranking.NewBM25(ranking.DefaultBM25Params())
	score := ranker.Score([]string{"go"}, doc("1"), idx)
	if score != 0 {
		t.Errorf("empty index: Score() = %v, want 0", score)
	}
}

func TestBM25_Score_TermAbsentFromDocument(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("1"), doc("2")},
		[][]string{{"go"}, {"python"}},
	)
	ranker := ranking.NewBM25(ranking.DefaultBM25Params())
	score := ranker.Score([]string{"go"}, doc("2"), idx)
	if score != 0 {
		t.Errorf("absent term: Score() = %v, want 0", score)
	}
}

func TestBM25_Score_NonNegative(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("1"), doc("2"), doc("3")},
		[][]string{
			{"the", "a", "is"},
			{"the", "a"},
			{"the"},
		},
	)
	ranker := ranking.NewBM25(ranking.DefaultBM25Params())

	for _, term := range []string{"the", "a", "is"} {
		score := ranker.Score([]string{term}, doc("1"), idx)
		if score < 0 {
			t.Errorf("Score(%q) = %v, want >= 0", term, score)
		}
	}
}

func TestBM25_HigherTF_GivesHigherScore(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("rich"), doc("sparse")},
		[][]string{
			{"go", "go", "go", "go"},
			{"go"},
		},
	)
	ranker := ranking.NewBM25(ranking.DefaultBM25Params())

	scoreRich := ranker.Score([]string{"go"}, doc("rich"), idx)
	scoreSparse := ranker.Score([]string{"go"}, doc("sparse"), idx)

	if scoreRich <= scoreSparse {
		t.Errorf("higher TF should score higher: rich=%v sparse=%v", scoreRich, scoreSparse)
	}
}

func TestBM25_RareTerm_GivesHigherIDF(t *testing.T) {
	docs := []searchinator.Document{
		doc("target"), doc("2"), doc("3"), doc("4"), doc("5"),
	}
	tokens := [][]string{
		{"rare", "common"},
		{"common"},
		{"common"},
		{"common"},
		{"common"},
	}
	idx := buildIndex(docs, tokens)
	ranker := ranking.NewBM25(ranking.DefaultBM25Params())

	scoreRare := ranker.Score([]string{"rare"}, doc("target"), idx)
	scoreCommon := ranker.Score([]string{"common"}, doc("target"), idx)

	if scoreRare <= scoreCommon {
		t.Errorf("rare term should score higher: rare=%v common=%v", scoreRare, scoreCommon)
	}
}

func TestBM25_LongerDocument_PenalisedByLengthNorm(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("short"), doc("long")},
		[][]string{
			{"go"},
			{"go", "filler", "filler", "filler", "filler", "filler", "filler", "filler"},
		},
	)
	ranker := ranking.NewBM25(ranking.DefaultBM25Params())

	scoreShort := ranker.Score([]string{"go"}, doc("short"), idx)
	scoreLong := ranker.Score([]string{"go"}, doc("long"), idx)

	if scoreShort <= scoreLong {
		t.Errorf("shorter doc should score higher: short=%v long=%v", scoreShort, scoreLong)
	}
}

func TestBM25_NoLengthNorm_WithBZero(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("short"), doc("long")},
		[][]string{
			{"go"},
			{"go", "filler", "filler", "filler", "filler"},
		},
	)
	params := ranking.BM25Params{K1: 1.2, B: 0}
	ranker := ranking.NewBM25(params)

	scoreShort := ranker.Score([]string{"go"}, doc("short"), idx)
	scoreLong := ranker.Score([]string{"go"}, doc("long"), idx)

	if math.Abs(scoreShort-scoreLong) > 1e-9 {
		t.Errorf("b=0: scores should be equal: short=%v long=%v", scoreShort, scoreLong)
	}
}

func TestBM25_MultiTermQuery_SumsTermScores(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("1"), doc("2")},
		[][]string{
			{"go", "search"},
			{"python"},
		},
	)
	ranker := ranking.NewBM25(ranking.DefaultBM25Params())

	scoreA := ranker.Score([]string{"go"}, doc("1"), idx)
	scoreB := ranker.Score([]string{"search"}, doc("1"), idx)
	scoreBoth := ranker.Score([]string{"go", "search"}, doc("1"), idx)

	expected := scoreA + scoreB
	if math.Abs(scoreBoth-expected) > 1e-9 {
		t.Errorf("multi-term score: got %v, want %v (sum of %v + %v)", scoreBoth, expected, scoreA, scoreB)
	}
}

func TestBM25_Ordering(t *testing.T) {
	idx := buildIndex(
		[]searchinator.Document{doc("best"), doc("mid"), doc("none")},
		[][]string{
			{"go", "go", "go"},
			{"go", "filler", "filler", "filler", "filler"},
			{"python", "rust"},
		},
	)
	ranker := ranking.NewBM25(ranking.DefaultBM25Params())
	query := []string{"go"}

	scoreBest := ranker.Score(query, doc("best"), idx)
	scoreMid := ranker.Score(query, doc("mid"), idx)
	scoreNone := ranker.Score(query, doc("none"), idx)

	if !(scoreBest > scoreMid) {
		t.Errorf("expected best > mid: %v > %v", scoreBest, scoreMid)
	}
	if scoreNone != 0 {
		t.Errorf("expected none = 0, got %v", scoreNone)
	}
}

func TestDefaultBM25Params(t *testing.T) {
	p := ranking.DefaultBM25Params()
	if p.K1 != 1.2 {
		t.Errorf("default K1 = %v, want 1.2", p.K1)
	}
	if p.B != 0.75 {
		t.Errorf("default B = %v, want 0.75", p.B)
	}
}
