package engine_test

import (
	"testing"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/engine"
)

func fieldedEngine(t *testing.T, boosts map[string]float64) *engine.FieldedEngine {
	t.Helper()
	fe, err := engine.NewFieldedEngine(engine.DefaultConfig(), boosts)
	if err != nil {
		t.Fatalf("NewFieldedEngine: %v", err)
	}
	return fe
}

func TestFieldedEngine_TitleBoostWins(t *testing.T) {
	fe := fieldedEngine(t, map[string]float64{"title": 5, "body": 1})
	err := fe.Index([]searchinator.Document{
		{ID: "a", Fields: map[string]string{
			"title": "rust safety",
			"body":  "a language",
		}},
		{ID: "b", Fields: map[string]string{
			"title": "a language",
			"body":  "rust safety discussed at length here",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	results, err := fe.Search("rust safety")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	if results[0].Document.ID != "a" {
		t.Errorf("title match (a) should rank first, got %s", results[0].Document.ID)
	}
}

func TestFieldedEngine_RequiresBoosts(t *testing.T) {
	if _, err := engine.NewFieldedEngine(engine.DefaultConfig(), nil); err == nil {
		t.Error("expected error with no field boosts")
	}
}

func TestFieldedEngine_EmptyQuery(t *testing.T) {
	fe := fieldedEngine(t, map[string]float64{"title": 1})
	res, err := fe.Search("   ")
	if err != nil || len(res) != 0 {
		t.Errorf("empty query: got %v err=%v, want empty", res, err)
	}
}
