package semantic

import (
	"errors"
	"sort"
)

type Match struct {
	ID    string
	Score float64
}

type VectorIndex interface {
	Add(id string, vec []float32)

	Remove(id string)

	Search(vec []float32, k int) []Match

	Len() int
}

var ErrDimMismatch = errors.New("semantic: vector dimensionality mismatch")

type entry struct {
	id  string
	vec []float32
}

type BruteForceIndex struct {
	entries []entry
	pos     map[string]int
}

func NewBruteForceIndex() *BruteForceIndex {
	return &BruteForceIndex{pos: make(map[string]int)}
}

func (b *BruteForceIndex) Add(id string, vec []float32) {
	if i, ok := b.pos[id]; ok {
		b.entries[i].vec = vec
		return
	}
	b.pos[id] = len(b.entries)
	b.entries = append(b.entries, entry{id: id, vec: vec})
}

func (b *BruteForceIndex) Remove(id string) {
	i, ok := b.pos[id]
	if !ok {
		return
	}
	last := len(b.entries) - 1
	if i != last {
		b.entries[i] = b.entries[last]
		b.pos[b.entries[i].id] = i
	}
	b.entries = b.entries[:last]
	delete(b.pos, id)
}

func (b *BruteForceIndex) Len() int { return len(b.entries) }

func (b *BruteForceIndex) Search(vec []float32, k int) []Match {
	matches := make([]Match, 0, len(b.entries))
	for _, e := range b.entries {
		if len(e.vec) != len(vec) {
			continue
		}
		matches = append(matches, Match{ID: e.id, Score: dot(vec, e.vec)})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].ID < matches[j].ID
	})
	if k > 0 && len(matches) > k {
		matches = matches[:k]
	}
	return matches
}

func dot(a, b []float32) float64 {
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}
