package semantic

import (
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

type Embedder interface {
	Embed(text string) ([]float32, error)

	Dim() int
}

type HashEmbedder struct {
	dim int
}

func NewHashEmbedder(dim int) *HashEmbedder {
	if dim <= 0 {
		dim = 256
	}
	return &HashEmbedder{dim: dim}
}

func (h *HashEmbedder) Dim() int { return h.dim }

func (h *HashEmbedder) Embed(text string) ([]float32, error) {
	vec := make([]float32, h.dim)
	for _, tok := range tokenize(text) {
		bucket, sign := hashToken(tok, h.dim)
		vec[bucket] += sign
	}
	normalize(vec)
	return vec, nil
}

func tokenize(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func hashToken(tok string, dim int) (bucket int, sign float32) {
	hsh := fnv.New32a()
	_, _ = hsh.Write([]byte(tok))
	sum := hsh.Sum32()
	bucket = int(sum % uint32(dim))
	if sum&0x80000000 != 0 {
		return bucket, -1
	}
	return bucket, 1
}

func normalize(vec []float32) {
	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}
	if sum == 0 {
		return
	}
	inv := float32(1 / math.Sqrt(sum))
	for i := range vec {
		vec[i] *= inv
	}
}
