package index

import (
	"encoding/gob"
	"fmt"
	"io"

	"github.com/Fiecher/searchinator"
)

type snapshot struct {
	Postings    postings
	Documents   map[string]searchinator.Document
	DocLengths  map[string]int
	TotalTokens int
}

func init() {

	gob.Register(map[string]any{})
	gob.Register("")
	gob.Register(int(0))
	gob.Register(int64(0))
	gob.Register(float64(0))
	gob.Register(true)
}

func (idx *InvertedIndex) Save(w io.Writer) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	snap := snapshot{
		Postings:    idx.postings,
		Documents:   idx.documents,
		DocLengths:  idx.docLengths,
		TotalTokens: idx.totalTokens,
	}
	if err := gob.NewEncoder(w).Encode(snap); err != nil {
		return fmt.Errorf("index: save: %w", err)
	}
	return nil
}

func (idx *InvertedIndex) Load(r io.Reader) error {
	var snap snapshot
	if err := gob.NewDecoder(r).Decode(&snap); err != nil {
		return fmt.Errorf("index: load: %w", err)
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.postings = snap.Postings
	idx.documents = snap.Documents
	idx.docLengths = snap.DocLengths
	idx.totalTokens = snap.TotalTokens

	if idx.postings == nil {
		idx.postings = make(postings)
	}
	if idx.documents == nil {
		idx.documents = make(map[string]searchinator.Document)
	}
	if idx.docLengths == nil {
		idx.docLengths = make(map[string]int)
	}
	return nil
}
