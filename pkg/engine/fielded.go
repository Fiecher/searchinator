package engine

import (
	"errors"
	"fmt"
	"sort"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/index"
)

type FieldedEngine struct {
	config  Config
	boosts  map[string]float64
	indexes map[string]*index.InvertedIndex
	docs    map[string]searchinator.Document
}

func NewFieldedEngine(cfg Config, boosts map[string]float64) (*FieldedEngine, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("engine: invalid config: %w", err)
	}
	if len(boosts) == 0 {
		return nil, errors.New("engine: at least one field boost must be configured")
	}
	indexes := make(map[string]*index.InvertedIndex, len(boosts))
	for field := range boosts {
		indexes[field] = index.NewInvertedIndex()
	}
	return &FieldedEngine{
		config:  cfg,
		boosts:  boosts,
		indexes: indexes,
		docs:    make(map[string]searchinator.Document),
	}, nil
}

func (fe *FieldedEngine) Index(docs []searchinator.Document) error {
	for _, doc := range docs {
		fe.docs[doc.ID] = doc
		for field, idx := range fe.indexes {
			content, ok := fe.fieldText(doc, field)
			if !ok {
				continue
			}
			tokens := fe.config.Analyzer.Analyze(content)
			if err := idx.Add(doc, tokens); err != nil {
				return fmt.Errorf("engine: index %q field %q: %w", doc.ID, field, err)
			}
		}
	}
	return nil
}

func (fe *FieldedEngine) fieldText(doc searchinator.Document, field string) (string, bool) {
	if doc.Fields != nil {
		if v, ok := doc.Fields[field]; ok {
			return v, true
		}
	}
	if field == "text" && doc.Text != "" {
		return doc.Text, true
	}
	return "", false
}

func (fe *FieldedEngine) Search(queryText string) ([]searchinator.Result, error) {
	terms := fe.config.Analyzer.Analyze(queryText)
	if len(terms) == 0 {
		return []searchinator.Result{}, nil
	}

	scores := make(map[string]float64)
	for field, idx := range fe.indexes {
		boost := fe.boosts[field]
		if boost == 0 {
			boost = 1
		}
		candidates := make(map[string]struct{})
		for _, term := range terms {
			for _, id := range idx.Get(term) {
				candidates[id] = struct{}{}
			}
		}
		for id := range candidates {
			doc := fe.docs[id]
			scores[id] += boost * fe.config.Ranker.Score(terms, doc, idx)
		}
	}

	results := make([]searchinator.Result, 0, len(scores))
	for id, s := range scores {
		results = append(results, searchinator.Result{Document: fe.docs[id], Score: s})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Document.ID < results[j].Document.ID
	})
	return results, nil
}
