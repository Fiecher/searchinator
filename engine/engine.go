package engine

import (
	"errors"
	"fmt"
	"sort"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/index"
)

type Engine struct {
	config Config
	index  index.Index
}

func NewEngine(cfg Config) (*Engine, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("engine: invalid config: %w", err)
	}
	return &Engine{
		config: cfg,
		index:  index.NewInvertedIndex(),
	}, nil
}

func (e *Engine) Index(docs []searchinator.Document) error {
	for _, doc := range docs {
		tokens := e.config.Analyzer.Analyze(doc.Text)
		if err := e.index.Add(doc, tokens); err != nil {
			return fmt.Errorf("engine: failed to index document %q: %w", doc.ID, err)
		}
	}
	return nil
}

func (e *Engine) Search(query string) ([]searchinator.Result, error) {
	queryTerms := e.config.Analyzer.Analyze(query)
	if len(queryTerms) == 0 {
		return []searchinator.Result{}, nil
	}

	candidates := e.collectCandidates(queryTerms)
	if len(candidates) == 0 {
		return []searchinator.Result{}, nil
	}

	results := make([]searchinator.Result, 0, len(candidates))
	for id := range candidates {
		doc, ok := e.index.GetDocument(id)
		if !ok {
			continue
		}
		score := e.config.Ranker.Score(queryTerms, doc, e.index)
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
