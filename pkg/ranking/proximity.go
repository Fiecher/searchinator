package ranking

import (
	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/index"
)

type Proximity struct {
	base   Ranker
	weight float64
}

func NewProximity(base Ranker, weight float64) *Proximity {
	return &Proximity{base: base, weight: weight}
}

func (p *Proximity) Score(queryTerms []string, doc searchinator.Document, idx index.Index) float64 {
	score := p.base.Score(queryTerms, doc, idx)
	if p.weight == 0 || len(queryTerms) < 2 {
		return score
	}

	seen := make(map[string]struct{})
	var lists [][]int
	for _, term := range queryTerms {
		if _, dup := seen[term]; dup {
			continue
		}
		seen[term] = struct{}{}
		if pos := idx.Positions(term, doc.ID); len(pos) > 0 {
			lists = append(lists, pos)
		}
	}
	if len(lists) < 2 {
		return score
	}

	window := minWindow(lists)
	if window < 0 {
		return score
	}
	return score + p.weight/(1+float64(window))
}

func minWindow(lists [][]int) int {
	idxs := make([]int, len(lists))
	best := -1

	for {
		lo, hi := 1<<62, -(1 << 62)
		minList := -1
		for i, l := range lists {
			v := l[idxs[i]]
			if v < lo {
				lo = v
				minList = i
			}
			if v > hi {
				hi = v
			}
		}

		if span := hi - lo; best < 0 || span < best {
			best = span
		}

		idxs[minList]++
		if idxs[minList] == len(lists[minList]) {
			return best
		}
	}
}
