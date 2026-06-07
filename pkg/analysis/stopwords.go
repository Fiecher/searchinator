package analysis

import "strings"

type StopWordsFilter struct {
	stop map[string]struct{}
}

func NewStopWordsFilter(words []string) *StopWordsFilter {
	set := make(map[string]struct{}, len(words))
	for _, w := range words {
		set[strings.ToLower(w)] = struct{}{}
	}
	return &StopWordsFilter{stop: set}
}

func (f *StopWordsFilter) Filter(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if _, ok := f.stop[strings.ToLower(t)]; ok {
			continue
		}
		out = append(out, t)
	}
	return out
}

func DefaultEnglishStopWords() []string {
	return []string{
		"a", "an", "and", "are", "as", "at", "be", "but", "by", "for",
		"from", "has", "have", "he", "in", "is", "it", "its", "of", "on",
		"or", "that", "the", "this", "to", "was", "were", "will", "with",
		"you", "your",
	}
}

func DefaultRussianStopWords() []string {
	return []string{
		"и", "в", "во", "не", "что", "он", "на", "я", "с", "со",
		"как", "а", "то", "все", "она", "так", "его", "но", "да", "ты",
		"к", "у", "же", "вы", "за", "бы", "по", "только", "ее", "мне",
		"было", "вот", "от", "меня", "еще", "нет", "о", "из", "ему", "теперь",
		"даже", "ну", "ли", "если", "или", "быть", "был", "него", "до", "вас",
		"для", "при", "это", "этот", "эта", "эти", "тот", "та", "те", "там",
		"чтобы", "чем", "чём", "который", "которая", "которые", "которых",
		"также", "тоже", "уже", "ещё", "над", "под", "без", "между", "через",
		"их", "им", "них", "ним", "она", "оно", "они", "мы", "вам", "нам",
		"свой", "своя", "свои", "своего", "своих", "себя", "сам", "сама",
		"может", "можно", "нужно", "благодаря", "включая", "поддерживает",
	}
}
