package semantic

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Fiecher/searchinator"
)

type WordPredictor interface {
	Predict(description string, limit int) ([]string, error)
}

type CorpusPredictor struct {
	analyze   func(string) []string
	vocab     map[string]int
	cooc      map[string]map[string]int
	surface   map[string]map[string]int
	thesaurus map[string][]string
}

func NewCorpusPredictor(docs []searchinator.Document, analyze func(string) []string) *CorpusPredictor {
	p := &CorpusPredictor{
		analyze:   analyze,
		vocab:     make(map[string]int),
		cooc:      make(map[string]map[string]int),
		surface:   make(map[string]map[string]int),
		thesaurus: defaultThesaurus(),
	}
	p.Index(docs)
	return p
}

func (p *CorpusPredictor) Index(docs []searchinator.Document) {
	for _, d := range docs {

		var stems []string
		seen := make(map[string]struct{})
		for _, w := range predictTokenize(d.Text) {
			st := p.stem(w)
			if st == "" {
				continue
			}
			row := p.surface[st]
			if row == nil {
				row = make(map[string]int)
				p.surface[st] = row
			}
			row[w]++
			if _, ok := seen[st]; ok {
				continue
			}
			seen[st] = struct{}{}
			stems = append(stems, st)
		}
		for _, t := range stems {
			p.vocab[t]++
		}
		for _, a := range stems {
			row := p.cooc[a]
			if row == nil {
				row = make(map[string]int)
				p.cooc[a] = row
			}
			for _, b := range stems {
				if a != b {
					row[b]++
				}
			}
		}
	}
}

func (p *CorpusPredictor) stem(word string) string {
	if p.analyze == nil {
		return word
	}
	toks := p.analyze(word)
	if len(toks) == 0 {
		return ""
	}
	return toks[len(toks)-1]
}

func (p *CorpusPredictor) render(stem string) string {
	best, bestN := stem, -1
	for s, n := range p.surface[stem] {
		if n > bestN || (n == bestN && s < best) {
			best, bestN = s, n
		}
	}
	return best
}

func (p *CorpusPredictor) Predict(description string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 6
	}
	scores := make(map[string]float64)
	for _, qw := range uniqueTerms(predictTokenize(description)) {
		for _, assoc := range p.thesaurus[qw] {
			st := p.stem(assoc)
			if st != "" && p.vocab[st] > 0 {
				scores[st] += 2.0
			}
		}
		q := p.stem(qw)
		if q == "" {
			continue
		}
		row, inCorpus := p.cooc[q]
		if !inCorpus {
			continue
		}

		scores[q] += 1.0
		for term, count := range row {
			scores[term] += float64(count) / float64(1+p.vocab[term])
		}
	}

	type kv struct {
		word  string
		score float64
	}
	ranked := make([]kv, 0, len(scores))
	for w, s := range scores {
		ranked = append(ranked, kv{w, s})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].word < ranked[j].word
	})

	out := make([]string, 0, limit)
	for _, e := range ranked {
		out = append(out, p.render(e.word))
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func defaultThesaurus() map[string][]string {
	return map[string][]string{
		"память":             {"памяти", "владения", "заимствования", "указатель"},
		"памяти":             {"владения", "заимствования", "сборщика", "указатель"},
		"утечка":             {"памяти", "сборщика", "владения"},
		"утечки":             {"памяти", "сборщика", "владения"},
		"безопасный":         {"безопасности", "владения", "заимствования", "компилятор"},
		"безопасность":       {"безопасности", "владения", "заимствования"},
		"мусор":              {"сборщика", "мусора", "владения"},
		"мусора":             {"сборщика", "владения", "заимствования"},
		"сборка":             {"сборщика", "мусора"},
		"быстрый":            {"компилируемый", "эффективного", "скорость", "системный"},
		"скорость":           {"компилируемый", "эффективного", "системный"},
		"производительность": {"компилируемый", "эффективного", "системный"},
		"параллельный":       {"конкурентности", "параллелизма"},
		"параллельное":       {"конкурентности", "параллелизма"},
		"параллельных":       {"конкурентности", "параллелизма"},
		"параллелизм":        {"конкурентности", "параллелизма"},
		"параллельность":     {"конкурентности", "параллелизма"},
		"многопоточность":    {"конкурентности", "параллелизма"},
		"выполнение":         {"конкурентности", "параллелизма"},
		"типизация":          {"типизацией", "статической", "типов", "строгой"},
		"типы":               {"типов", "статической", "типизацией"},
		"функциональный":     {"функциональный", "монады", "ленивыми", "вычислениями"},
		"функциональное":     {"функциональный", "монады", "ленивыми", "вычислениями"},
		"функциональная":     {"функциональный", "монады", "ленивыми", "вычислениями"},
		"функции":            {"функциональный", "монады", "высшего", "порядка"},
		"объект":             {"объектно", "классов", "байткод"},
		"объекты":            {"объектно", "классов", "байткод"},
		"ооп":                {"объектно", "классов"},
		"мобильный":          {"apple", "ios", "macos", "swift"},
		"телефон":            {"apple", "ios", "macos"},
		"эппл":               {"apple", "ios", "macos", "swift"},
		"веб":                {"javascript", "typescript"},
		"браузер":            {"javascript", "typescript"},
		"сайт":               {"javascript", "typescript"},
		"система":            {"системный", "операционных", "встраиваемых", "ядер"},
		"системный":          {"низкоуровневый", "операционных", "встраиваемых"},
	}
}

func predictTokenize(text string) []string {
	raw := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := raw[:0]
	for _, w := range raw {
		if utf8.RuneCountInString(w) >= 3 {
			out = append(out, w)
		}
	}
	return out
}

func uniqueTerms(terms []string) []string {
	seen := make(map[string]struct{}, len(terms))
	out := terms[:0]
	for _, t := range terms {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
