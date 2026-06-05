package eval

import (
	"math"
	"testing"
)

func approx(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

func TestPrecisionAtK(t *testing.T) {
	ranked := []string{"a", "b", "c", "d"}
	rel := Relevant{"a": true, "c": true}

	tests := []struct {
		k    int
		want float64
	}{
		{1, 1.0},
		{2, 0.5},
		{4, 0.5},
		{0, 0.5},
		{100, 0.5},
	}
	for _, tt := range tests {
		if got := PrecisionAtK(ranked, rel, tt.k); !approx(got, tt.want) {
			t.Errorf("PrecisionAtK(k=%d) = %v, want %v", tt.k, got, tt.want)
		}
	}
}

func TestRecallAtK(t *testing.T) {
	ranked := []string{"a", "b", "c", "d"}
	rel := Relevant{"a": true, "c": true}

	tests := []struct {
		k    int
		want float64
	}{
		{1, 0.5},
		{2, 0.5},
		{3, 1.0},
		{4, 1.0},
	}
	for _, tt := range tests {
		if got := RecallAtK(ranked, rel, tt.k); !approx(got, tt.want) {
			t.Errorf("RecallAtK(k=%d) = %v, want %v", tt.k, got, tt.want)
		}
	}
}

func TestAveragePrecision(t *testing.T) {
	ranked := []string{"a", "b", "c", "d"}
	rel := Relevant{"a": true, "c": true}

	if got := AveragePrecision(ranked, rel); !approx(got, 0.8333333) {
		t.Errorf("AveragePrecision = %v, want 0.8333333", got)
	}
}

func TestAveragePrecision_PerfectRanking(t *testing.T) {
	ranked := []string{"a", "b", "c"}
	rel := Relevant{"a": true, "b": true}

	if got := AveragePrecision(ranked, rel); !approx(got, 1.0) {
		t.Errorf("AveragePrecision = %v, want 1.0", got)
	}
}

func TestNDCGAtK(t *testing.T) {
	ranked := []string{"a", "b", "c", "d"}
	rel := Relevant{"a": true, "c": true}

	want := 1.5 / 1.6309298
	if got := NDCGAtK(ranked, rel, 4); !approx(got, want) {
		t.Errorf("NDCGAtK = %v, want %v", got, want)
	}
}

func TestNDCGAtK_PerfectRanking(t *testing.T) {
	ranked := []string{"a", "b", "c"}
	rel := Relevant{"a": true, "b": true}
	if got := NDCGAtK(ranked, rel, 3); !approx(got, 1.0) {
		t.Errorf("NDCGAtK perfect = %v, want 1.0", got)
	}
}

func TestMetrics_EmptyRelevant(t *testing.T) {
	ranked := []string{"a", "b"}
	empty := Relevant{}
	if got := PrecisionAtK(ranked, empty, 2); got != 0 {
		t.Errorf("PrecisionAtK empty rel = %v, want 0", got)
	}
	if got := RecallAtK(ranked, empty, 2); got != 0 {
		t.Errorf("RecallAtK empty rel = %v, want 0", got)
	}
	if got := AveragePrecision(ranked, empty); got != 0 {
		t.Errorf("AveragePrecision empty rel = %v, want 0", got)
	}
	if got := NDCGAtK(ranked, empty, 2); got != 0 {
		t.Errorf("NDCGAtK empty rel = %v, want 0", got)
	}
}

func TestMetrics_EmptyRanked(t *testing.T) {
	rel := Relevant{"a": true}
	if got := PrecisionAtK(nil, rel, 5); got != 0 {
		t.Errorf("PrecisionAtK empty ranked = %v, want 0", got)
	}
	if got := RecallAtK(nil, rel, 5); got != 0 {
		t.Errorf("RecallAtK empty ranked = %v, want 0", got)
	}
}

func TestMeanAveragePrecision(t *testing.T) {
	runs := map[string][]string{
		"q1": {"a", "b", "c"},
		"q2": {"x", "y"},
	}
	judgments := map[string]Relevant{
		"q1": {"a": true, "c": true},
		"q2": {"y": true},
	}

	if got := MeanAveragePrecision(runs, judgments); !approx(got, 0.6666666) {
		t.Errorf("MeanAveragePrecision = %v, want 0.6666666", got)
	}
}

func TestMeanAveragePrecision_NoJudgments(t *testing.T) {
	if got := MeanAveragePrecision(nil, nil); got != 0 {
		t.Errorf("MAP no judgments = %v, want 0", got)
	}
}
