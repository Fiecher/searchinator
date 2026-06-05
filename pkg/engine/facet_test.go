package engine_test

import (
	"testing"

	"github.com/Fiecher/searchinator"
)

func TestEngine_Facet_CountsByField(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "systems programming", Meta: map[string]any{"paradigm": "systems"}},
		{ID: "b", Text: "systems software", Meta: map[string]any{"paradigm": "systems"}},
		{ID: "c", Text: "functional programming", Meta: map[string]any{"paradigm": "functional"}},
	})

	counts, err := e.Facet("programming systems software functional", "paradigm")
	if err != nil {
		t.Fatal(err)
	}
	if counts["systems"] != 2 {
		t.Errorf("systems count = %d, want 2", counts["systems"])
	}
	if counts["functional"] != 1 {
		t.Errorf("functional count = %d, want 1", counts["functional"])
	}
}

func TestEngine_Facet_NumericField(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "go", Meta: map[string]any{"year": 2009}},
		{ID: "b", Text: "go", Meta: map[string]any{"year": 2009}},
	})
	counts, err := e.Facet("go", "year")
	if err != nil {
		t.Fatal(err)
	}
	if counts["2009"] != 2 {
		t.Errorf("year 2009 count = %d, want 2", counts["2009"])
	}
}

func TestEngine_Facet_MissingFieldIgnored(t *testing.T) {
	e := defaultEngine(t)
	mustIndex(t, e, []searchinator.Document{
		{ID: "a", Text: "go", Meta: map[string]any{"paradigm": "concurrent"}},
		{ID: "b", Text: "go"},
	})
	counts, err := e.Facet("go", "paradigm")
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 1 || counts["concurrent"] != 1 {
		t.Errorf("facet = %v, want {concurrent:1}", counts)
	}
}
