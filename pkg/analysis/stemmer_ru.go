package analysis

import "strings"

type RussianStemmer struct{}

func NewRussianStemmer() *RussianStemmer { return &RussianStemmer{} }

func (s *RussianStemmer) Filter(tokens []string) []string {
	out := make([]string, len(tokens))
	for i, t := range tokens {
		out[i] = stemRussian(t)
	}
	return out
}

const ruVowels = "аеиоуыэюя"

func isRuVowel(r rune) bool { return strings.ContainsRune(ruVowels, r) }

var (
	ruPerfectiveG1 = []string{"вшись", "вши", "в"}
	ruPerfectiveG2 = []string{"ившись", "ывшись", "ивши", "ывши", "ив", "ыв"}

	ruReflexive = []string{"ся", "сь"}

	ruAdjective = []string{
		"ими", "ыми", "его", "ого", "ему", "ому",
		"ее", "ие", "ые", "ое", "ей", "ий", "ый", "ой",
		"ем", "им", "ым", "ом", "их", "ых", "ую", "юю",
		"ая", "яя", "ою", "ею",
	}
	ruParticipleG1 = []string{"ющ", "вш", "ем", "нн", "щ"}
	ruParticipleG2 = []string{"ивш", "ывш", "ующ"}

	ruVerbG1 = []string{
		"ете", "йте", "ешь", "нно",
		"ла", "на", "ли", "ем", "ло", "но", "ет", "ют", "ны", "ть",
		"й", "л", "н",
	}
	ruVerbG2 = []string{
		"ейте", "уйте", "ила", "ыла", "ена", "ите", "или", "ыли", "ило",
		"ыло", "ено", "ует", "уют", "ены", "ить", "ыть", "ишь",
		"ей", "уй", "ил", "ыл", "им", "ым", "ен", "ят", "ит", "ыт", "ую", "ю",
	}

	ruNoun = []string{
		"иями", "ями", "ами", "иях", "ях", "ием", "ием",
		"иям", "ям", "ией", "ев", "ов", "ие", "ье", "еи", "ии",
		"ей", "ой", "ий", "ам", "ем", "ом", "ах", "ию", "ью", "ия", "ья",
		"а", "е", "и", "й", "о", "у", "ы", "ь", "ю", "я",
	}

	ruDerivational = []string{"ость", "ост"}
	ruSuperlative  = []string{"ейше", "ейш"}
)

func stemRussian(word string) string {
	w := []rune(strings.ToLower(word))
	for i := range w {
		if w[i] == 'ё' {
			w[i] = 'е'
		}
	}

	rv := computeRV(w)
	_, r2 := computeR1R2(w)

	if nw, ok := tryRemoveG1(w, rv, ruPerfectiveG1); ok {
		w = nw
	} else if nw, ok := tryRemove(w, rv, ruPerfectiveG2); ok {
		w = nw
	} else {
		if nw, ok := tryRemove(w, rv, ruReflexive); ok {
			w = nw
		}
		if nw, ok := removeAdjectival(w, rv); ok {
			w = nw
		} else if nw, ok := tryRemoveG1(w, rv, ruVerbG1); ok {
			w = nw
		} else if nw, ok := tryRemove(w, rv, ruVerbG2); ok {
			w = nw
		} else if nw, ok := tryRemove(w, rv, ruNoun); ok {
			w = nw
		}
	}

	if nw, ok := tryRemove(w, rv, []string{"и"}); ok {
		w = nw
	}

	if nw, ok := tryRemove(w, r2, ruDerivational); ok {
		w = nw
	}

	if endsInRegion(w, rv, "нн") {
		w = w[:len(w)-1]
	}
	if nw, ok := tryRemove(w, rv, ruSuperlative); ok {
		w = nw
		if endsInRegion(w, rv, "нн") {
			w = w[:len(w)-1]
		}
	}
	if endsInRegion(w, rv, "ь") {
		w = w[:len(w)-1]
	}

	return string(w)
}

func computeRV(w []rune) int {
	for i := 0; i < len(w); i++ {
		if isRuVowel(w[i]) {
			return i + 1
		}
	}
	return len(w)
}

func computeR1R2(w []rune) (r1, r2 int) {
	n := len(w)
	r1, r2 = n, n
	i := 0
	for ; i+1 < n; i++ {
		if isRuVowel(w[i]) && !isRuVowel(w[i+1]) {
			r1 = i + 2
			break
		}
	}
	for i = r1; i+1 < n; i++ {
		if isRuVowel(w[i]) && !isRuVowel(w[i+1]) {
			r2 = i + 2
			break
		}
	}
	return r1, r2
}

func endsInRegion(w []rune, regionStart int, suffix string) bool {
	sr := []rune(suffix)
	off := len(w) - len(sr)
	if off < 0 || off < regionStart {
		return false
	}
	for i, r := range sr {
		if w[off+i] != r {
			return false
		}
	}
	return true
}

func tryRemove(w []rune, regionStart int, suffixes []string) ([]rune, bool) {
	for _, suf := range suffixes {
		if endsInRegion(w, regionStart, suf) {
			return w[:len(w)-len([]rune(suf))], true
		}
	}
	return w, false
}

func tryRemoveG1(w []rune, regionStart int, suffixes []string) ([]rune, bool) {
	for _, suf := range suffixes {
		if !endsInRegion(w, regionStart, suf) {
			continue
		}
		off := len(w) - len([]rune(suf))
		if off-1 < 0 {
			continue
		}
		if prev := w[off-1]; prev == 'а' || prev == 'я' {
			return w[:off], true
		}
	}
	return w, false
}

func removeAdjectival(w []rune, rv int) ([]rune, bool) {
	nw, ok := tryRemove(w, rv, ruAdjective)
	if !ok {
		return w, false
	}
	w = nw
	if pw, ok := tryRemoveG1(w, rv, ruParticipleG1); ok {
		w = pw
	} else if pw, ok := tryRemove(w, rv, ruParticipleG2); ok {
		w = pw
	}
	return w, true
}
