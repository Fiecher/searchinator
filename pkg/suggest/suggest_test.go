package suggest

import "testing"

func TestSuggest_CorrectsTypo(t *testing.T) {
	s := New([]string{"memory", "safety", "concurrency", "programming"}, nil)
	got, ok := s.Suggest("memmory")
	if !ok || got != "memory" {
		t.Errorf("Suggest(memmory) = %q,%v want memory,true", got, ok)
	}
}

func TestSuggest_NoCorrectionForKnownWord(t *testing.T) {
	s := New([]string{"memory", "safety"}, map[string]int{"memory": 3, "safety": 2})
	if got, ok := s.Suggest("memory"); ok {
		t.Errorf("known word should not be corrected, got %q", got)
	}
}

func TestSuggest_PrefersHigherFrequency(t *testing.T) {

	s := New([]string{"like", "life"}, map[string]int{"like": 1, "life": 10})
	got, ok := s.Suggest("lite")
	if !ok || got != "life" {
		t.Errorf("Suggest(lite) = %q,%v want life,true (higher freq)", got, ok)
	}
}

func TestSuggest_CorrectQuery(t *testing.T) {
	s := New([]string{"memory", "safety", "rust"}, nil)
	got, changed := s.Correct("memmory safty")
	if !changed || got != "memory safety" {
		t.Errorf("Correct = %q,%v want 'memory safety',true", got, changed)
	}
}

func TestSuggest_NoMatchTooFar(t *testing.T) {
	s := New([]string{"memory"}, nil, WithMaxDistance(1))
	if got, ok := s.Suggest("xyzzy"); ok {
		t.Errorf("distant word should not match, got %q", got)
	}
}
