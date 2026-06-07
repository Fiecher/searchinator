package analysis

import "testing"

func TestRussianStemmerConflation(t *testing.T) {
	groups := [][]string{
		{"красивый", "красивая", "красивые", "красивого", "красивыми"},
		{"программирование", "программирования", "программированию"},
		{"владение", "владения", "владению", "владениями"},
		{"заимствование", "заимствования", "заимствованию"},
		{"язык", "языка", "языку", "языком", "языки"},
	}
	for _, g := range groups {
		stems := make([]string, len(g))
		for i, w := range g {
			stems[i] = stemRussian(w)
		}
		for i := 1; i < len(stems); i++ {
			if stems[i] != stems[0] {
				t.Errorf("conflation failed for %v: %q (%q) != %q (%q)",
					g, g[i], stems[i], g[0], stems[0])
			}
		}
	}
}

func TestRussianStemmerLatinUnchanged(t *testing.T) {
	for _, w := range []string{"javascript", "rust", "apple", "ios", "macos"} {
		if got := stemRussian(w); got != w {
			t.Errorf("stemRussian(%q) = %q, want unchanged", w, got)
		}
	}
}

func TestRussianStemmerYoNormalization(t *testing.T) {
	if a, b := stemRussian("ещё"), stemRussian("еще"); a != b {
		t.Errorf("ё normalization failed: %q != %q", a, b)
	}
}

func TestRussianStemmerFilter(t *testing.T) {
	s := NewRussianStemmer()
	in := []string{"красивый", "языки"}
	out := s.Filter(in)
	if len(out) != len(in) {
		t.Fatalf("Filter len = %d, want %d", len(out), len(in))
	}
	if out[0] != stemRussian("красивый") || out[1] != stemRussian("языки") {
		t.Errorf("Filter = %v, want stemmed forms", out)
	}
}
