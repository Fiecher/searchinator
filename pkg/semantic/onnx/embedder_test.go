//go:build onnx

package onnx

import (
	"math"
	"os"
	"testing"
)

func TestONNXEmbedderSmoke(t *testing.T) {
	model := os.Getenv("SEARCHINATOR_ONNX_MODEL")
	tok := os.Getenv("SEARCHINATOR_ONNX_TOKENIZER")
	if model == "" || tok == "" {
		t.Skip("set SEARCHINATOR_ONNX_MODEL and SEARCHINATOR_ONNX_TOKENIZER to run")
	}

	emb, err := NewEmbedder(Config{
		ModelPath:         model,
		TokenizerPath:     tok,
		SharedLibraryPath: os.Getenv("SEARCHINATOR_ONNX_LIB"),
	})
	if err != nil {
		t.Fatalf("NewEmbedder: %v", err)
	}

	vec, err := emb.Embed("a compiled statically typed language for concurrency")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != emb.Dim() {
		t.Fatalf("len(vec) = %d, want Dim() = %d", len(vec), emb.Dim())
	}

	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}
	if math.Abs(sum-1) > 1e-4 {
		t.Fatalf("vector not unit length: ||v||^2 = %v", sum)
	}

	vec2, err := emb.Embed("a compiled statically typed language for concurrency")
	if err != nil {
		t.Fatalf("Embed (2nd): %v", err)
	}
	for i := range vec {
		if vec[i] != vec2[i] {
			t.Fatalf("non-deterministic at %d: %v vs %v", i, vec[i], vec2[i])
		}
	}

	related, _ := emb.Embed("go is concurrent and compiled")
	unrelated, _ := emb.Embed("a recipe for chocolate cake")
	if dot(vec, related) <= dot(vec, unrelated) {
		t.Fatalf("related sentence not closer: related=%v unrelated=%v",
			dot(vec, related), dot(vec, unrelated))
	}
}

func dot(a, b []float32) float64 {
	var s float64
	for i := range a {
		s += float64(a[i]) * float64(b[i])
	}
	return s
}
