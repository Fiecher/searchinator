package ranking

import (
	"math"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/index"
)

type TFIDF struct{}

func NewTFIDF() *TFIDF {
	return &TFIDF{}
}

func (t *TFIDF) Score(queryTerms []string, doc searchinator.Document, idx index.Index) float64 {
	n := idx.DocumentCount()
	if n == 0 || len(queryTerms) == 0 {
		return 0
	}

	var score float64
	for _, term := range queryTerms {
		tf := idx.TermFrequency(term, doc.ID)
		if tf == 0 {
			continue
		}
		df := idx.DocumentFrequency(term)
		if df == 0 {
			continue
		}
		tfWeight := 1 + math.Log(float64(tf))
		idf := math.Log(float64(n) / float64(df))
		score += tfWeight * idf
	}
	return score
}
