package suggest

import (
	"strings"

	"github.com/Fiecher/searchinator/pkg/analysis"
)

type Suggester struct {
	maxDistance int
	minOverlap  float64
	terms       []string
	freq        map[string]int
	bigrams     map[string][]string
}

type Option func(*Suggester)

func WithMaxDistance(d int) Option { return func(s *Suggester) { s.maxDistance = d } }

func WithMinOverlap(f float64) Option { return func(s *Suggester) { s.minOverlap = f } }

func New(vocab []string, freq map[string]int, opts ...Option) *Suggester {
	s := &Suggester{
		maxDistance: 2,
		minOverlap:  0.3,
		freq:        freq,
		bigrams:     make(map[string][]string),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.freq == nil {
		s.freq = make(map[string]int)
	}

	seen := make(map[string]struct{})
	for _, term := range vocab {
		if _, dup := seen[term]; dup {
			continue
		}
		seen[term] = struct{}{}
		s.terms = append(s.terms, term)
		for _, bg := range bigrams(term) {
			s.bigrams[bg] = append(s.bigrams[bg], term)
		}
	}
	return s
}

func (s *Suggester) Suggest(word string) (string, bool) {
	if _, ok := s.freq[word]; ok {
		return "", false
	}
	if contains(s.terms, word) {
		return "", false
	}

	wbg := bigrams(word)
	if len(wbg) == 0 {
		return "", false
	}

	counts := make(map[string]int)
	for _, bg := range wbg {
		for _, term := range s.bigrams[bg] {
			counts[term]++
		}
	}

	best := ""
	bestDist := s.maxDistance + 1
	bestFreq := -1
	for term, shared := range counts {
		if float64(shared)/float64(len(wbg)) < s.minOverlap {
			continue
		}
		d := analysis.EditDistance(word, term)
		if d > s.maxDistance {
			continue
		}
		f := s.freq[term]
		if d < bestDist || (d == bestDist && f > bestFreq) {
			best, bestDist, bestFreq = term, d, f
		}
	}
	if best == "" {
		return "", false
	}
	return best, true
}

func (s *Suggester) Correct(query string) (string, bool) {
	fields := strings.Fields(query)
	changed := false
	for i, f := range fields {
		if sug, ok := s.Suggest(strings.ToLower(f)); ok {
			fields[i] = sug
			changed = true
		}
	}
	return strings.Join(fields, " "), changed
}

func bigrams(s string) []string {
	r := []rune(s)
	if len(r) < 2 {
		if len(r) == 1 {
			return []string{string(r)}
		}
		return nil
	}
	out := make([]string, 0, len(r)-1)
	for i := 0; i+1 < len(r); i++ {
		out = append(out, string(r[i:i+2]))
	}
	return out
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
