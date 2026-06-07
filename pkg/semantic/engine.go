package semantic

import (
	"fmt"

	"github.com/Fiecher/searchinator"
)

type Engine struct {
	embedder Embedder
	index    VectorIndex
	docs     map[string]searchinator.Document
}

func NewEngine(embedder Embedder, index VectorIndex) (*Engine, error) {
	if embedder == nil {
		return nil, fmt.Errorf("semantic: embedder must not be nil")
	}
	if index == nil {
		return nil, fmt.Errorf("semantic: index must not be nil")
	}
	return &Engine{
		embedder: embedder,
		index:    index,
		docs:     make(map[string]searchinator.Document),
	}, nil
}

func (e *Engine) Index(docs []searchinator.Document) error {
	for _, doc := range docs {
		vec, err := e.embedder.Embed(doc.Text)
		if err != nil {
			return fmt.Errorf("semantic: failed to embed document %q: %w", doc.ID, err)
		}
		e.index.Add(doc.ID, vec)
		e.docs[doc.ID] = doc
	}
	return nil
}

func (e *Engine) Delete(id string) {
	e.index.Remove(id)
	delete(e.docs, id)
}

func (e *Engine) Search(query string, limit int) ([]searchinator.Result, error) {
	qv, err := e.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("semantic: failed to embed query: %w", err)
	}
	matches := e.index.Search(qv, limit)
	results := make([]searchinator.Result, 0, len(matches))
	for _, m := range matches {
		doc, ok := e.docs[m.ID]
		if !ok {
			continue
		}
		results = append(results, searchinator.Result{Document: doc, Score: m.Score})
	}
	return results, nil
}

func (e *Engine) RankedIDs(query string, limit int) ([]string, error) {
	results, err := e.Search(query, limit)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.Document.ID
	}
	return ids, nil
}

func (e *Engine) DocumentCount() int { return len(e.docs) }
