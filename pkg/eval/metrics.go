package eval

import "math"

type Relevant map[string]bool

func PrecisionAtK(ranked []string, rel Relevant, k int) float64 {
	k = clampK(k, len(ranked))
	if k == 0 {
		return 0
	}
	hits := 0
	for i := 0; i < k; i++ {
		if rel[ranked[i]] {
			hits++
		}
	}
	return float64(hits) / float64(k)
}

func RecallAtK(ranked []string, rel Relevant, k int) float64 {
	total := countRelevant(rel)
	if total == 0 {
		return 0
	}
	k = clampK(k, len(ranked))
	hits := 0
	for i := 0; i < k; i++ {
		if rel[ranked[i]] {
			hits++
		}
	}
	return float64(hits) / float64(total)
}

func AveragePrecision(ranked []string, rel Relevant) float64 {
	total := countRelevant(rel)
	if total == 0 {
		return 0
	}
	hits := 0
	var sum float64
	for i, id := range ranked {
		if rel[id] {
			hits++
			sum += float64(hits) / float64(i+1)
		}
	}
	return sum / float64(total)
}

func NDCGAtK(ranked []string, rel Relevant, k int) float64 {
	k = clampK(k, len(ranked))
	if k == 0 {
		return 0
	}

	var dcg float64
	for i := 0; i < k; i++ {
		if rel[ranked[i]] {
			dcg += 1.0 / math.Log2(float64(i+2))
		}
	}

	ideal := countRelevant(rel)
	if ideal > k {
		ideal = k
	}
	var idcg float64
	for i := 0; i < ideal; i++ {
		idcg += 1.0 / math.Log2(float64(i+2))
	}
	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}

func MeanAveragePrecision(runs map[string][]string, judgments map[string]Relevant) float64 {
	if len(judgments) == 0 {
		return 0
	}
	var sum float64
	for q, rel := range judgments {
		sum += AveragePrecision(runs[q], rel)
	}
	return sum / float64(len(judgments))
}

func countRelevant(rel Relevant) int {
	n := 0
	for _, ok := range rel {
		if ok {
			n++
		}
	}
	return n
}

func clampK(k, n int) int {
	if k <= 0 || k > n {
		return n
	}
	return k
}
