package analysis

import (
	"strings"
	"unicode"
)

type LowercaseFilter struct{}

func NewLowercaseFilter() *LowercaseFilter {
	return &LowercaseFilter{}
}

func (f *LowercaseFilter) Filter(tokens []string) []string {
	out := make([]string, len(tokens))
	for i, t := range tokens {
		out[i] = strings.ToLower(t)
	}
	return out
}

type PunctuationFilter struct{}

func NewPunctuationFilter() *PunctuationFilter {
	return &PunctuationFilter{}
}

func (f *PunctuationFilter) Filter(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		cleaned := strings.TrimFunc(t, func(r rune) bool {
			return unicode.IsPunct(r) || unicode.IsSymbol(r)
		})
		if cleaned != "" {
			out = append(out, cleaned)
		}
	}
	return out
}
