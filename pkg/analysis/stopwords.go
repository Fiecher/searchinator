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
