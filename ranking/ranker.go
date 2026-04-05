package ranking

import (
	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/index"
)

type Ranker interface {
	Score(queryTerms []string, doc searchinator.Document, idx index.Index) float64
}
