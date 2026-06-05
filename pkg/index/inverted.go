package index

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/compress"
)

type postings map[string]map[string][]int

type InvertedIndex struct {
	mu          sync.RWMutex
	postings    postings
	documents   map[string]searchinator.Document
	docLengths  map[string]int
	totalTokens int
}

func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		postings:   make(postings),
		documents:  make(map[string]searchinator.Document),
		docLengths: make(map[string]int),
	}
}

func (idx *InvertedIndex) Add(doc searchinator.Document, tokens []string) error {
	if doc.ID == "" {
		return errors.New("index: document ID must not be empty")
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	if _, exists := idx.documents[doc.ID]; exists {
		idx.remove(doc.ID)
	}

	idx.documents[doc.ID] = doc
	idx.docLengths[doc.ID] = len(tokens)
	idx.totalTokens += len(tokens)

	for pos, term := range tokens {
		if idx.postings[term] == nil {
			idx.postings[term] = make(map[string][]int)
		}
		idx.postings[term][doc.ID] = append(idx.postings[term][doc.ID], pos)
	}

	return nil
}

func (idx *InvertedIndex) Remove(docID string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if _, ok := idx.documents[docID]; !ok {
		return fmt.Errorf("index: document %q not found", docID)
	}
	idx.remove(docID)
	return nil
}

func (idx *InvertedIndex) remove(docID string) {
	idx.totalTokens -= idx.docLengths[docID]
	delete(idx.docLengths, docID)
	delete(idx.documents, docID)

	for term, docs := range idx.postings {
		if _, ok := docs[docID]; ok {
			delete(docs, docID)
			if len(docs) == 0 {
				delete(idx.postings, term)
			}
		}
	}
}

func (idx *InvertedIndex) Get(term string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	docs, ok := idx.postings[term]
	if !ok {
		return []string{}
	}

	ids := make([]string, 0, len(docs))
	for id := range docs {
		ids = append(ids, id)
	}
	return ids
}

func (idx *InvertedIndex) GetDocument(id string) (searchinator.Document, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	doc, ok := idx.documents[id]
	return doc, ok
}

func (idx *InvertedIndex) DocumentCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.documents)
}

func (idx *InvertedIndex) TermFrequency(term, docID string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	docs, ok := idx.postings[term]
	if !ok {
		return 0
	}
	return len(docs[docID])
}

func (idx *InvertedIndex) DocumentFrequency(term string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.postings[term])
}

func (idx *InvertedIndex) AverageDocumentLength() float64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	n := len(idx.documents)
	if n == 0 {
		return 0
	}
	return float64(idx.totalTokens) / float64(n)
}

func (idx *InvertedIndex) DocumentLength(docID string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.docLengths[docID]
}

func (idx *InvertedIndex) TotalTokens() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.totalTokens
}

type CompressionStats struct {
	RawBytes        int
	CompressedBytes int
	Ratio           float64
}

func (idx *InvertedIndex) CompressionStats() CompressionStats {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var raw, comp int
	for _, docs := range idx.postings {
		for _, positions := range docs {
			raw += len(positions) * 8
			comp += len(compress.EncodeGaps(positions))
		}
	}

	var ratio float64
	if raw > 0 {
		ratio = float64(comp) / float64(raw)
	}
	return CompressionStats{RawBytes: raw, CompressedBytes: comp, Ratio: ratio}
}

func (idx *InvertedIndex) DocumentIDs() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	ids := make([]string, 0, len(idx.documents))
	for id := range idx.documents {
		ids = append(ids, id)
	}
	return ids
}

func (idx *InvertedIndex) Terms() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	terms := make([]string, 0, len(idx.postings))
	for term := range idx.postings {
		terms = append(terms, term)
	}
	return terms
}

func (idx *InvertedIndex) Positions(term, docID string) []int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	docs, ok := idx.postings[term]
	if !ok {
		return []int{}
	}
	pos := docs[docID]
	out := make([]int, len(pos))
	copy(out, pos)
	return out
}
