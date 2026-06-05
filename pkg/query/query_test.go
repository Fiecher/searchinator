package query_test

import (
	"sort"
	"testing"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/analysis"
	"github.com/Fiecher/searchinator/pkg/index"
	"github.com/Fiecher/searchinator/pkg/query"
)

func testAnalyzer() analysis.Analyzer {
	return analysis.NewPipelineAnalyzer(
		analysis.NewWhitespaceTokenizer(),
		analysis.NewLowercaseFilter(),
		analysis.NewPunctuationFilter(),
	)
}

func buildIndex(t *testing.T, docs []searchinator.Document) (index.Index, analysis.Analyzer) {
	t.Helper()
	a := testAnalyzer()
	idx := index.NewInvertedIndex()
	for _, d := range docs {
		if err := idx.Add(d, a.Analyze(d.Text)); err != nil {
			t.Fatalf("add %q: %v", d.ID, err)
		}
	}
	return idx, a
}

func matchIDs(t *testing.T, idx index.Index, a analysis.Analyzer, q string) []string {
	t.Helper()
	parsed, err := query.Parse(q, a)
	if err != nil {
		t.Fatalf("parse %q: %v", q, err)
	}
	set := parsed.Match(idx)
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func corpus() []searchinator.Document {
	return []searchinator.Document{
		{ID: "a", Text: "go has memory safety", Meta: map[string]any{"year": 2012, "tag": "go"}},
		{ID: "b", Text: "rust has memory safety without gc", Meta: map[string]any{"year": 2015, "tag": "rust"}},
		{ID: "c", Text: "go is fast and simple", Meta: map[string]any{"year": 2009, "tag": "go"}},
		{ID: "d", Text: "python is simple and slow", Meta: map[string]any{"year": 2020, "tag": "python"}},
	}
}

func TestParse_AND(t *testing.T) {
	idx, a := buildIndex(t, corpus())
	got := matchIDs(t, idx, a, "go AND safety")
	if !eq(got, []string{"a"}) {
		t.Errorf("go AND safety = %v, want [a]", got)
	}
}

func TestParse_ImplicitAND(t *testing.T) {
	idx, a := buildIndex(t, corpus())
	got := matchIDs(t, idx, a, "go safety")
	if !eq(got, []string{"a"}) {
		t.Errorf("implicit AND = %v, want [a]", got)
	}
}

func TestParse_OR(t *testing.T) {
	idx, a := buildIndex(t, corpus())
	got := matchIDs(t, idx, a, "rust OR python")
	if !eq(got, []string{"b", "d"}) {
		t.Errorf("rust OR python = %v, want [b d]", got)
	}
}

func TestParse_NOT(t *testing.T) {
	idx, a := buildIndex(t, corpus())

	got := matchIDs(t, idx, a, "simple AND NOT go")
	if !eq(got, []string{"d"}) {
		t.Errorf("simple AND NOT go = %v, want [d]", got)
	}
}

func TestParse_Parentheses(t *testing.T) {
	idx, a := buildIndex(t, corpus())

	got := matchIDs(t, idx, a, "(memory OR fast) AND go")
	if !eq(got, []string{"a", "c"}) {
		t.Errorf("(memory OR fast) AND go = %v, want [a c]", got)
	}
}

func TestParse_PrecedenceAndOverOr(t *testing.T) {
	idx, a := buildIndex(t, corpus())

	got := matchIDs(t, idx, a, "go AND fast OR python")
	if !eq(got, []string{"c", "d"}) {
		t.Errorf("precedence = %v, want [c d]", got)
	}
}

func TestParse_Phrase(t *testing.T) {
	idx, a := buildIndex(t, corpus())
	got := matchIDs(t, idx, a, `"memory safety"`)
	if !eq(got, []string{"a", "b"}) {
		t.Errorf("phrase = %v, want [a b]", got)
	}
}

func TestParse_FilterNumericGreater(t *testing.T) {
	idx, a := buildIndex(t, corpus())
	got := matchIDs(t, idx, a, "year>2012")
	if !eq(got, []string{"b", "d"}) {
		t.Errorf("year>2012 = %v, want [b d]", got)
	}
}

func TestParse_FilterNumericGTE(t *testing.T) {
	idx, a := buildIndex(t, corpus())
	got := matchIDs(t, idx, a, "year>=2015")
	if !eq(got, []string{"b", "d"}) {
		t.Errorf("year>=2015 = %v, want [b d]", got)
	}
}

func TestParse_FilterStringEquals(t *testing.T) {
	idx, a := buildIndex(t, corpus())
	got := matchIDs(t, idx, a, "tag=go")
	if !eq(got, []string{"a", "c"}) {
		t.Errorf("tag=go = %v, want [a c]", got)
	}
}

func TestParse_FilterNotEquals(t *testing.T) {
	idx, a := buildIndex(t, corpus())
	got := matchIDs(t, idx, a, "simple AND tag!=python")
	if !eq(got, []string{"c"}) {
		t.Errorf("simple AND tag!=python = %v, want [c]", got)
	}
}

func TestParse_FilterCombinedWithTerm(t *testing.T) {
	idx, a := buildIndex(t, corpus())

	got := matchIDs(t, idx, a, "safety AND year<2015")
	if !eq(got, []string{"a"}) {
		t.Errorf("safety AND year<2015 = %v, want [a]", got)
	}
}

func TestParse_Terms_ExcludesNegativeAndFilter(t *testing.T) {
	a := testAnalyzer()
	parsed, err := query.Parse("go AND NOT slow AND year>2010", a)
	if err != nil {
		t.Fatal(err)
	}
	got := parsed.Terms()
	if !eq(got, []string{"go"}) {
		t.Errorf("Terms = %v, want [go] (NOT and filter terms excluded)", got)
	}
}

func TestParse_Empty(t *testing.T) {
	idx, a := buildIndex(t, corpus())
	got := matchIDs(t, idx, a, "   ")
	if len(got) != 0 {
		t.Errorf("empty query = %v, want []", got)
	}
}

func TestParse_UnbalancedParen(t *testing.T) {
	a := testAnalyzer()
	if _, err := query.Parse("(go OR rust", a); err == nil {
		t.Error("expected error for unbalanced parenthesis")
	}
}
