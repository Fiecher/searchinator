package analysis_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/Fiecher/searchinator/analysis"
)

func TestFuzzyFilter_ExactMatch(t *testing.T) {
	vocab := []string{"search", "index", "query"}
	f := analysis.NewFuzzyFilter(vocab, 1)

	got := f.Filter([]string{"search"})
	if len(got) != 1 || got[0] != "search" {
		t.Errorf("exact match: got %v, want [search]", got)
	}
}

func TestFuzzyFilter_SingleTypo(t *testing.T) {
	tests := []struct {
		name  string
		token string
		vocab []string
		want  string
	}{
		{"transposition", "serach", []string{"search"}, "search"},
		{"deletion", "golan", []string{"golang"}, "golang"},
		{"insertion", "golangg", []string{"golang"}, "golang"},
		{"substitution", "gp", []string{"go"}, "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := analysis.NewFuzzyFilter(tt.vocab, 1)
			got := f.Filter([]string{tt.token})
			if len(got) != 1 || got[0] != tt.want {
				t.Errorf("Filter([%q]) = %v, want [%s]", tt.token, got, tt.want)
			}
		})
	}
}

func TestFuzzyFilter_BeyondMaxDistance(t *testing.T) {
	vocab := []string{"search"}
	f := analysis.NewFuzzyFilter(vocab, 1)

	got := f.Filter([]string{"banana"})
	if len(got) != 1 || got[0] != "banana" {
		t.Errorf("beyond max distance: got %v, want [banana]", got)
	}
}

func TestFuzzyFilter_MaxDistance2(t *testing.T) {
	vocab := []string{"search"}
	f := analysis.NewFuzzyFilter(vocab, 2)

	got := f.Filter([]string{"srach"})
	found := false
	for _, g := range got {
		if g == "search" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("distance-2: Filter([srach]) = %v, want to contain 'search'", got)
	}
}

func TestFuzzyFilter_Deduplication(t *testing.T) {
	vocab := []string{"go", "got"}
	f := analysis.NewFuzzyFilter(vocab, 1)

	got := f.Filter([]string{"go"})

	seen := make(map[string]int)
	for _, g := range got {
		seen[g]++
	}
	for term, count := range seen {
		if count > 1 {
			t.Errorf("duplicate term %q appeared %d times in output", term, count)
		}
	}
}

func TestFuzzyFilter_MultipleTokens(t *testing.T) {
	vocab := []string{"golang", "search", "index"}
	f := analysis.NewFuzzyFilter(vocab, 1)

	got := f.Filter([]string{"golan", "searc"})

	sortedGot := make([]string, len(got))
	copy(sortedGot, got)
	sort.Strings(sortedGot)

	want := []string{"golang", "search"}
	sort.Strings(want)

	if !reflect.DeepEqual(sortedGot, want) {
		t.Errorf("multi-token: got %v, want %v", sortedGot, want)
	}
}

func TestFuzzyFilter_EmptyInput(t *testing.T) {
	f := analysis.NewFuzzyFilter([]string{"search"}, 1)
	got := f.Filter([]string{})
	if len(got) != 0 {
		t.Errorf("empty input: got %v, want []", got)
	}
}

func TestFuzzyFilter_EmptyVocabulary(t *testing.T) {
	f := analysis.NewFuzzyFilter([]string{}, 1)
	got := f.Filter([]string{"search", "index"})
	want := []string{"search", "index"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("empty vocab: got %v, want %v", got, want)
	}
}

func TestFuzzyFilter_UnicodeTokens(t *testing.T) {
	vocab := []string{"поиск"}
	f := analysis.NewFuzzyFilter(vocab, 1)

	got := f.Filter([]string{"поикс"})
	if len(got) != 1 || got[0] != "поиск" {
		t.Errorf("unicode: got %v, want [поиск]", got)
	}
}

func TestFuzzyFilter_ShortTokensSafety(t *testing.T) {
	vocab := []string{"go", "do", "to", "so", "no"}
	f := analysis.NewFuzzyFilter(vocab, 1)

	got := f.Filter([]string{"go"})
	if len(got) == 0 {
		t.Error("short token: expected at least 'go' in results")
	}

	found := false
	for _, g := range got {
		if g == "go" {
			found = true
		}
	}
	if !found {
		t.Errorf("short token: 'go' missing from results %v", got)
	}
}

func TestLevenshtein_Properties(t *testing.T) {
	tests := []struct {
		a, b string
		maxD int
		want bool
	}{
		{"go", "go", 0, true},
		{"search", "search", 0, true},

		{"go", "got", 1, true},
		{"got", "go", 1, true},
		{"gp", "go", 1, true},
		{"og", "go", 1, true},

		{"srach", "search", 2, true},
		{"pythn", "python", 2, true},

		{"banana", "search", 1, false},
		{"abc", "xyz", 1, false},

		{"", "", 0, true},
		{"", "a", 1, true},
		{"a", "", 1, true},
		{"", "ab", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.a+"→"+tt.b, func(t *testing.T) {
			f := analysis.NewFuzzyFilter([]string{tt.b}, tt.maxD)
			got := f.Filter([]string{tt.a})

			matched := false
			for _, g := range got {
				if g == tt.b {
					matched = true
					break
				}
			}

			if matched != tt.want {
				t.Errorf("levenshtein(%q, %q) within %d: matched=%v, want=%v",
					tt.a, tt.b, tt.maxD, matched, tt.want)
			}
		})
	}
}
