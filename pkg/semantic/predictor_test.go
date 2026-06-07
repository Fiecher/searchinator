package semantic

import (
	"strings"
	"testing"

	"github.com/Fiecher/searchinator"
)

func predictorCorpus() []searchinator.Document {
	return []searchinator.Document{
		{ID: "rust", Text: "Rust системный язык программирования безопасность памяти без сборщика мусора владения заимствования компилятор"},
		{ID: "go", Text: "Go компилируемый язык программирования конкурентности параллелизма эффективного"},
		{ID: "haskell", Text: "Haskell функциональный язык программирования монады ленивыми вычислениями типов"},
		{ID: "swift", Text: "Swift компилируемый язык программирования apple ios macos"},
	}
}

func TestCorpusPredictor_CooccurrenceSurfacesRelatedWords(t *testing.T) {
	p := NewCorpusPredictor(predictorCorpus(), nil)
	got, err := p.Predict("сборка мусора и владение памятью", 6)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected predicted words, got none")
	}

	want := map[string]bool{"владения": true, "заимствования": true, "сборщика": true, "компилятор": true}
	hit := false
	for _, w := range got {
		if want[w] {
			hit = true
		}
	}
	if !hit {
		t.Errorf("Predict(...) = %v, want at least one of %v", got, want)
	}
}

func TestCorpusPredictor_ThesaurusExpandsAbsentWord(t *testing.T) {
	p := NewCorpusPredictor(predictorCorpus(), nil)

	got, err := p.Predict("мобильный телефон", 6)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "apple") && !strings.Contains(joined, "ios") && !strings.Contains(joined, "macos") {
		t.Errorf("Predict(...) = %v, want apple/ios/macos via thesaurus", got)
	}
}

func TestCorpusPredictor_RespectsLimit(t *testing.T) {
	p := NewCorpusPredictor(predictorCorpus(), nil)
	got, _ := p.Predict("язык программирования", 3)
	if len(got) > 3 {
		t.Errorf("len = %d, want <= 3", len(got))
	}
}

func TestCorpusPredictor_UnknownWordsYieldNothing(t *testing.T) {
	p := NewCorpusPredictor(predictorCorpus(), nil)
	got, _ := p.Predict("кулинария рецепты борщ", 6)
	if len(got) != 0 {
		t.Errorf("Predict(unrelated) = %v, want empty", got)
	}
}

func TestParseWords(t *testing.T) {
	got := parseWords("1. владения, 2. заимствования; сборщика. владения", 6)
	want := []string{"1", "владения", "2", "заимствования", "сборщика"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("parseWords = %v, want %v", got, want)
	}
}

func TestNewLLMPredictorValidation(t *testing.T) {
	if _, err := NewLLMPredictor(LLMConfig{Model: "m"}); err == nil {
		t.Error("expected error for empty endpoint")
	}
	if _, err := NewLLMPredictor(LLMConfig{Endpoint: "http://x"}); err == nil {
		t.Error("expected error for empty model")
	}
	if _, err := NewLLMPredictor(LLMConfig{Endpoint: "http://x", Model: "m"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
