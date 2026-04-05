package ranking

import (
	"math"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/index"
)

type BM25Params struct {
	K1 float64
	B  float64
}

func DefaultBM25Params() BM25Params {
	return BM25Params{K1: 1.2, B: 0.75}
}

type BM25 struct {
	params BM25Params
}

func NewBM25(params BM25Params) *BM25 {
	return &BM25{params: params}
}

func (b *BM25) Score(queryTerms []string, doc searchinator.Document, idx index.Index) float64 {
	n := idx.DocumentCount()
	if n == 0 || len(queryTerms) == 0 {
		return 0
	}

	avgdl := idx.AverageDocumentLength()
	docLen := float64(idx.DocumentLength(doc.ID))

	lengthNorm := 1 - b.params.B + b.params.B*(docLen/safeDivAvg(avgdl))

	var score float64
	for _, term := range queryTerms {
		tf := float64(idx.TermFrequency(term, doc.ID))
		if tf == 0 {
			continue
		}

		df := idx.DocumentFrequency(term)
		idf := okapiIDF(n, df)

		numerator := tf * (b.params.K1 + 1)
		denominator := tf + b.params.K1*lengthNorm
		score += idf * (numerator / denominator)
	}

	return score
}

func okapiIDF(n, df int) float64 {
	return math.Log(
		(float64(n)-float64(df)+0.5)/(float64(df)+0.5) + 1,
	)
}

func safeDivAvg(avgdl float64) float64 {
	if avgdl == 0 {
		return 1
	}
	return avgdl
}
