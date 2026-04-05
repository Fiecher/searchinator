package index

import (
	"errors"

	"github.com/Fiecher/searchinator"
)

type postings map[string]map[string]int

type InvertedIndex struct {
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

	if _, exists := idx.documents[doc.ID]; exists {
		idx.remove(doc.ID)
	}

	idx.documents[doc.ID] = doc

	idx.docLengths[doc.ID] = len(tokens)
	idx.totalTokens += len(tokens)

	for _, term := range tokens {
		if idx.postings[term] == nil {
			idx.postings[term] = make(map[string]int)
		}
		idx.postings[term][doc.ID]++
	}

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
	doc, ok := idx.documents[id]
	return doc, ok
}

func (idx *InvertedIndex) DocumentCount() int {
	return len(idx.documents)
}

func (idx *InvertedIndex) TermFrequency(term, docID string) int {
	docs, ok := idx.postings[term]
	if !ok {
		return 0
	}
	return docs[docID]
}

func (idx *InvertedIndex) DocumentFrequency(term string) int {
	return len(idx.postings[term])
}

func (idx *InvertedIndex) AverageDocumentLength() float64 {
	n := len(idx.documents)
	if n == 0 {
		return 0
	}
	return float64(idx.totalTokens) / float64(n)
}

func (idx *InvertedIndex) DocumentLength(docID string) int {
	return idx.docLengths[docID]
}
