package index

import "github.com/Fiecher/searchinator"

type Index interface {
	Add(doc searchinator.Document, tokens []string) error
	Get(term string) []string
	GetDocument(id string) (searchinator.Document, bool)
	DocumentCount() int
	TermFrequency(term, docID string) int
	DocumentFrequency(term string) int
	AverageDocumentLength() float64
	DocumentLength(docID string) int
}
