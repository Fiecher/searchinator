package engine

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/highlight"
	"github.com/Fiecher/searchinator/pkg/index"
	"github.com/Fiecher/searchinator/pkg/query"
	"github.com/Fiecher/searchinator/pkg/suggest"
	"github.com/Fiecher/searchinator/pkg/trie"
)

type Stats struct {
	DocumentCount         int
	TermCount             int
	AverageDocumentLength float64
}

type Engine struct {
	config Config
	index  index.Index

	vocab      *trie.Trie
	vocabDirty bool
}

func NewEngine(cfg Config) (*Engine, error) {
	return NewEngineWithIndex(cfg, index.NewInvertedIndex())
}

func NewEngineWithIndex(cfg Config, idx index.Index) (*Engine, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("engine: invalid config: %w", err)
	}
	if idx == nil {
		return nil, errors.New("engine: index must not be nil")
	}
	return &Engine{
		config: cfg,
		index:  idx,
	}, nil
}

func (e *Engine) Flush() error {
	if f, ok := e.index.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

func (e *Engine) Index(docs []searchinator.Document) error {
	for _, doc := range docs {
		tokens := e.config.Analyzer.Analyze(doc.Text)
		if err := e.index.Add(doc, tokens); err != nil {
			return fmt.Errorf("engine: failed to index document %q: %w", doc.ID, err)
		}
	}
	e.vocabDirty = true
	return nil
}

func (e *Engine) Delete(id string) error {
	if err := e.index.Remove(id); err != nil {
		return fmt.Errorf("engine: %w", err)
	}
	e.vocabDirty = true
	return nil
}

func (e *Engine) Search(query string) ([]searchinator.Result, error) {
	terms, phrases := e.parseQuery(query)

	scoreTerms := append([]string{}, terms...)
	for _, ph := range phrases {
		scoreTerms = append(scoreTerms, ph...)
	}
	if len(scoreTerms) == 0 {
		return []searchinator.Result{}, nil
	}

	var candidates map[string]struct{}
	if len(phrases) > 0 {
		candidates = e.phraseCandidates(phrases)
	} else {
		candidates = e.collectCandidates(terms)
	}
	if len(candidates) == 0 {
		return []searchinator.Result{}, nil
	}

	results := make([]searchinator.Result, 0, len(candidates))
	for id := range candidates {
		doc, ok := e.index.GetDocument(id)
		if !ok {
			continue
		}
		score := e.config.Ranker.Score(scoreTerms, doc, e.index)
		results = append(results, searchinator.Result{
			Document: doc,
			Score:    score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Document.ID < results[j].Document.ID
	})

	return results, nil
}

func (e *Engine) parseQuery(query string) (terms []string, phrases [][]string) {
	segments := strings.Split(query, "\"")
	for i, seg := range segments {
		analyzed := e.config.Analyzer.Analyze(seg)
		if len(analyzed) == 0 {
			continue
		}
		if i%2 == 1 {
			phrases = append(phrases, analyzed)
		} else {
			terms = append(terms, analyzed...)
		}
	}
	return terms, phrases
}

func (e *Engine) phraseCandidates(phrases [][]string) map[string]struct{} {
	var result map[string]struct{}
	for i, ph := range phrases {
		matching := e.docsMatchingPhrase(ph)
		if i == 0 {
			result = matching
		} else {
			for id := range result {
				if _, ok := matching[id]; !ok {
					delete(result, id)
				}
			}
		}
		if len(result) == 0 {
			return result
		}
	}
	return result
}

func (e *Engine) docsMatchingPhrase(phrase []string) map[string]struct{} {
	out := make(map[string]struct{})
	if len(phrase) == 0 {
		return out
	}
	for _, docID := range e.index.Get(phrase[0]) {
		if query.PhraseMatch(e.index, docID, phrase) {
			out[docID] = struct{}{}
		}
	}
	return out
}

func (e *Engine) SearchBool(q string) ([]searchinator.Result, error) {
	parsed, err := query.Parse(q, e.config.Analyzer)
	if err != nil {
		return nil, fmt.Errorf("engine: %w", err)
	}

	matches := parsed.Match(e.index)
	if len(matches) == 0 {
		return []searchinator.Result{}, nil
	}

	scoreTerms := parsed.Terms()
	results := make([]searchinator.Result, 0, len(matches))
	for id := range matches {
		doc, ok := e.index.GetDocument(id)
		if !ok {
			continue
		}
		var score float64
		if len(scoreTerms) > 0 {
			score = e.config.Ranker.Score(scoreTerms, doc, e.index)
		}
		results = append(results, searchinator.Result{Document: doc, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Document.ID < results[j].Document.ID
	})

	return results, nil
}

func (e *Engine) SearchParallel(query string, workers int) ([]searchinator.Result, error) {
	terms, phrases := e.parseQuery(query)

	scoreTerms := append([]string{}, terms...)
	for _, ph := range phrases {
		scoreTerms = append(scoreTerms, ph...)
	}
	if len(scoreTerms) == 0 {
		return []searchinator.Result{}, nil
	}

	var candidates map[string]struct{}
	if len(phrases) > 0 {
		candidates = e.phraseCandidates(phrases)
	} else {
		candidates = e.collectCandidates(terms)
	}
	if len(candidates) == 0 {
		return []searchinator.Result{}, nil
	}

	if workers <= 0 {
		workers = 4
	}
	ids := make([]string, 0, len(candidates))
	for id := range candidates {
		ids = append(ids, id)
	}

	jobs := make(chan string)
	out := make(chan searchinator.Result)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				doc, ok := e.index.GetDocument(id)
				if !ok {
					continue
				}
				out <- searchinator.Result{
					Document: doc,
					Score:    e.config.Ranker.Score(scoreTerms, doc, e.index),
				}
			}
		}()
	}
	go func() {
		for _, id := range ids {
			jobs <- id
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(out)
	}()

	results := make([]searchinator.Result, 0, len(ids))
	for r := range out {
		results = append(results, r)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Document.ID < results[j].Document.ID
	})
	return results, nil
}

func (e *Engine) SearchWildcard(pattern string) ([]searchinator.Result, error) {
	terms := e.ExpandWildcard(pattern)
	if len(terms) == 0 {
		return []searchinator.Result{}, nil
	}
	candidates := e.collectCandidates(terms)
	return e.rankCandidates(terms, candidates), nil
}

func (e *Engine) ExpandWildcard(pattern string) []string {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return nil
	}

	if strings.HasSuffix(pattern, "*") {
		stem := pattern[:len(pattern)-1]
		if !strings.ContainsAny(stem, "*?") {
			e.ensureVocab()
			return e.vocab.WithPrefix(stem)
		}
	}

	var out []string
	for _, term := range e.index.Terms() {
		if wildcardMatch(pattern, term) {
			out = append(out, term)
		}
	}
	return out
}

func (e *Engine) ensureVocab() {
	if e.vocab == nil || e.vocabDirty {
		t := trie.New()
		for _, term := range e.index.Terms() {
			t.Insert(term)
		}
		e.vocab = t
		e.vocabDirty = false
	}
}

func (e *Engine) rankCandidates(scoreTerms []string, candidates map[string]struct{}) []searchinator.Result {
	results := make([]searchinator.Result, 0, len(candidates))
	for id := range candidates {
		doc, ok := e.index.GetDocument(id)
		if !ok {
			continue
		}
		results = append(results, searchinator.Result{
			Document: doc,
			Score:    e.config.Ranker.Score(scoreTerms, doc, e.index),
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Document.ID < results[j].Document.ID
	})
	return results
}

func wildcardMatch(glob, s string) bool {
	g, str := []rune(glob), []rune(s)
	gi, si := 0, 0
	star, mark := -1, 0
	for si < len(str) {
		if gi < len(g) && (g[gi] == '?' || g[gi] == str[si]) {
			gi++
			si++
		} else if gi < len(g) && g[gi] == '*' {
			star = gi
			mark = si
			gi++
		} else if star != -1 {
			gi = star + 1
			mark++
			si = mark
		} else {
			return false
		}
	}
	for gi < len(g) && g[gi] == '*' {
		gi++
	}
	return gi == len(g)
}

func (e *Engine) Suggester() *suggest.Suggester {
	terms := e.index.Terms()
	freq := make(map[string]int, len(terms))
	for _, t := range terms {
		freq[t] = e.index.DocumentFrequency(t)
	}
	return suggest.New(terms, freq)
}

func (e *Engine) DidYouMean(query string) (string, bool) {
	return e.Suggester().Correct(query)
}

func (e *Engine) SearchN(query string, limit int) ([]searchinator.Result, error) {
	results, err := e.Search(query)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (e *Engine) Save(w io.Writer) error {
	idx, ok := e.index.(*index.InvertedIndex)
	if !ok {
		return errors.New("engine: index type does not support Save")
	}
	return idx.Save(w)
}

func (e *Engine) Load(r io.Reader) error {
	idx, ok := e.index.(*index.InvertedIndex)
	if !ok {
		return errors.New("engine: index type does not support Load")
	}
	if err := idx.Load(r); err != nil {
		return err
	}
	e.vocabDirty = true
	return nil
}

func (e *Engine) Facet(query, field string) (map[string]int, error) {
	results, err := e.Search(query)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, r := range results {
		if r.Document.Meta == nil {
			continue
		}
		if v, ok := r.Document.Meta[field]; ok {
			counts[fmt.Sprintf("%v", v)]++
		}
	}
	return counts, nil
}

func (e *Engine) Snippet(docID, query string) (string, bool) {
	doc, ok := e.index.GetDocument(docID)
	if !ok {
		return "", false
	}
	terms := e.config.Analyzer.Analyze(query)
	return highlight.Snippet(doc.Text, terms, e.config.Analyzer.Analyze, highlight.DefaultOptions()), true
}

func (e *Engine) Highlights(docID, query string) ([]highlight.Span, bool) {
	doc, ok := e.index.GetDocument(docID)
	if !ok {
		return nil, false
	}
	terms := e.config.Analyzer.Analyze(query)
	return highlight.HighlightSpans(doc.Text, terms, e.config.Analyzer.Analyze, highlight.DefaultOptions()), true
}

func (e *Engine) TermOccurrences(docID, query string) int {
	terms := e.config.Analyzer.Analyze(query)
	n := 0
	for _, t := range terms {
		n += e.index.TermFrequency(t, docID)
	}
	return n
}

func (e *Engine) Stats() Stats {
	return Stats{
		DocumentCount:         e.index.DocumentCount(),
		TermCount:             len(e.index.Terms()),
		AverageDocumentLength: e.index.AverageDocumentLength(),
	}
}

func VocabularyFromIndex(e *Engine) []string {
	return e.index.Terms()
}

func (e *Engine) collectCandidates(queryTerms []string) map[string]struct{} {
	seen := make(map[string]struct{})
	for _, term := range queryTerms {
		for _, id := range e.index.Get(term) {
			seen[id] = struct{}{}
		}
	}
	return seen
}

func validateConfig(cfg Config) error {
	if cfg.Analyzer == nil {
		return errors.New("analyzer must not be nil")
	}
	if cfg.Ranker == nil {
		return errors.New("ranker must not be nil")
	}
	return nil
}
