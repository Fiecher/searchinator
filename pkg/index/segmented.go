package index

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/Fiecher/searchinator"
)

type SegmentedIndex struct {
	dir string

	wmu sync.Mutex

	view atomic.Pointer[view]

	meta      sync.RWMutex
	idLoc     map[string]int
	liveToken int
	nextID    int
}

const bufferLoc = -1

const manifestName = "MANIFEST"

type view struct {
	buffer *InvertedIndex
	segs   []*segment
}

type segment struct {
	seqID      int
	file       string
	idx        *InvertedIndex
	tombstones map[string]struct{}
}

func (s *segment) live(id string) bool {
	_, dead := s.tombstones[id]
	return !dead
}

func (s *segment) get(term string) []string {
	ids := s.idx.Get(term)
	if len(s.tombstones) == 0 {
		return ids
	}
	out := ids[:0]
	for _, id := range ids {
		if s.live(id) {
			out = append(out, id)
		}
	}
	return out
}

func (s *segment) documentFrequency(term string) int {
	if len(s.tombstones) == 0 {
		return s.idx.DocumentFrequency(term)
	}
	return len(s.get(term))
}

func (s *segment) withTombstone(id string) *segment {
	ts := make(map[string]struct{}, len(s.tombstones)+1)
	for k := range s.tombstones {
		ts[k] = struct{}{}
	}
	ts[id] = struct{}{}
	return &segment{seqID: s.seqID, file: s.file, idx: s.idx, tombstones: ts}
}

func OpenSegmented(dir string) (*SegmentedIndex, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("index: open segmented: %w", err)
	}

	si := &SegmentedIndex{dir: dir, idLoc: make(map[string]int)}
	si.view.Store(&view{buffer: NewInvertedIndex(), segs: nil})

	m, err := readManifest(dir)
	if err != nil {
		return nil, err
	}
	if m == nil {

		cleanupOrphans(dir, nil)
		return si, nil
	}

	si.nextID = m.NextID
	segs := make([]*segment, 0, len(m.Segments))
	keep := make(map[string]struct{}, len(m.Segments))
	for _, ref := range m.Segments {
		idx := NewInvertedIndex()
		f, err := os.Open(filepath.Join(dir, ref.File))
		if err != nil {
			return nil, fmt.Errorf("index: open segment %q: %w", ref.File, err)
		}
		err = idx.Load(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("index: load segment %q: %w", ref.File, err)
		}
		ts := make(map[string]struct{}, len(ref.Tombstones))
		for _, id := range ref.Tombstones {
			ts[id] = struct{}{}
		}
		seg := &segment{seqID: ref.SeqID, file: ref.File, idx: idx, tombstones: ts}
		segs = append(segs, seg)
		keep[ref.File] = struct{}{}

		for _, id := range idx.DocumentIDs() {
			if _, dead := ts[id]; dead {
				continue
			}
			si.idLoc[id] = seg.seqID
			si.liveToken += idx.DocumentLength(id)
		}
	}

	si.view.Store(&view{buffer: NewInvertedIndex(), segs: segs})
	cleanupOrphans(dir, keep)
	return si, nil
}

func (si *SegmentedIndex) Add(doc searchinator.Document, tokens []string) error {
	if doc.ID == "" {
		return errors.New("index: document ID must not be empty")
	}

	si.wmu.Lock()
	defer si.wmu.Unlock()

	v := si.view.Load()

	si.meta.RLock()
	loc, exists := si.idLoc[doc.ID]
	si.meta.RUnlock()

	oldLen := 0
	tombstonedSegment := false
	if exists {
		if loc == bufferLoc {
			oldLen = v.buffer.DocumentLength(doc.ID)
		} else if seg := findSeg(v.segs, loc); seg != nil {
			oldLen = seg.idx.DocumentLength(doc.ID)

			si.view.Store(&view{buffer: v.buffer, segs: replaceSeg(v.segs, loc, seg.withTombstone(doc.ID))})
			tombstonedSegment = true
		}
	}

	if err := v.buffer.Add(doc, tokens); err != nil {
		return err
	}

	si.meta.Lock()
	if exists {
		si.liveToken -= oldLen
	}
	si.idLoc[doc.ID] = bufferLoc
	si.liveToken += len(tokens)
	si.meta.Unlock()

	if tombstonedSegment {

		return si.persistManifest()
	}
	return nil
}

func (si *SegmentedIndex) Remove(docID string) error {
	si.wmu.Lock()
	defer si.wmu.Unlock()

	v := si.view.Load()

	si.meta.Lock()
	loc, exists := si.idLoc[docID]
	if !exists {
		si.meta.Unlock()
		return fmt.Errorf("index: document %q not found", docID)
	}
	var oldLen int
	if loc == bufferLoc {
		oldLen = v.buffer.DocumentLength(docID)
	} else if seg := findSeg(v.segs, loc); seg != nil {
		oldLen = seg.idx.DocumentLength(docID)
	}
	delete(si.idLoc, docID)
	si.liveToken -= oldLen
	si.meta.Unlock()

	if loc == bufferLoc {
		return v.buffer.Remove(docID)
	}
	seg := findSeg(v.segs, loc)
	if seg == nil {
		return nil
	}
	si.view.Store(&view{buffer: v.buffer, segs: replaceSeg(v.segs, loc, seg.withTombstone(docID))})
	return si.persistManifest()
}

func (si *SegmentedIndex) Flush() error {
	si.wmu.Lock()
	defer si.wmu.Unlock()

	v := si.view.Load()
	if v.buffer.DocumentCount() == 0 {
		return nil
	}

	seqID := si.nextID
	file := fmt.Sprintf("seg-%06d.idx", seqID)
	if err := si.writeSegmentFile(file, v.buffer); err != nil {
		return err
	}

	newSeg := &segment{seqID: seqID, file: file, idx: v.buffer, tombstones: map[string]struct{}{}}
	newSegs := make([]*segment, len(v.segs)+1)
	copy(newSegs, v.segs)
	newSegs[len(v.segs)] = newSeg

	si.view.Store(&view{buffer: NewInvertedIndex(), segs: newSegs})

	si.meta.Lock()
	for _, id := range v.buffer.DocumentIDs() {
		si.idLoc[id] = seqID
	}
	si.nextID++
	si.meta.Unlock()

	return si.persistManifest()
}

func (si *SegmentedIndex) writeSegmentFile(file string, idx *InvertedIndex) error {
	tmp := filepath.Join(si.dir, file+".tmp")
	final := filepath.Join(si.dir, file)
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("index: write segment: %w", err)
	}
	if err := idx.Save(f); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("index: fsync segment: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, final); err != nil {
		return fmt.Errorf("index: commit segment: %w", err)
	}
	return nil
}

func (si *SegmentedIndex) persistManifest() error {
	v := si.view.Load()
	m := manifest{NextID: si.nextID}
	for _, s := range v.segs {
		ref := segmentRef{SeqID: s.seqID, File: s.file}
		for id := range s.tombstones {
			ref.Tombstones = append(ref.Tombstones, id)
		}
		sort.Strings(ref.Tombstones)
		m.Segments = append(m.Segments, ref)
	}

	tmp := filepath.Join(si.dir, manifestName+".tmp")
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("index: write manifest: %w", err)
	}
	if err := gob.NewEncoder(f).Encode(m); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("index: encode manifest: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("index: fsync manifest: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, filepath.Join(si.dir, manifestName)); err != nil {
		return fmt.Errorf("index: commit manifest: %w", err)
	}
	return nil
}

func (si *SegmentedIndex) Get(term string) []string {
	v := si.view.Load()
	out := v.buffer.Get(term)
	for _, s := range v.segs {
		out = append(out, s.get(term)...)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func (si *SegmentedIndex) GetDocument(id string) (searchinator.Document, bool) {
	v := si.view.Load()
	si.meta.RLock()
	loc, ok := si.idLoc[id]
	si.meta.RUnlock()
	if !ok {
		return searchinator.Document{}, false
	}
	if loc == bufferLoc {
		return v.buffer.GetDocument(id)
	}
	if seg := findSeg(v.segs, loc); seg != nil && seg.live(id) {
		return seg.idx.GetDocument(id)
	}
	return searchinator.Document{}, false
}

func (si *SegmentedIndex) DocumentCount() int {
	si.meta.RLock()
	defer si.meta.RUnlock()
	return len(si.idLoc)
}

func (si *SegmentedIndex) TermFrequency(term, docID string) int {
	v := si.view.Load()
	si.meta.RLock()
	loc, ok := si.idLoc[docID]
	si.meta.RUnlock()
	if !ok {
		return 0
	}
	if loc == bufferLoc {
		return v.buffer.TermFrequency(term, docID)
	}
	if seg := findSeg(v.segs, loc); seg != nil && seg.live(docID) {
		return seg.idx.TermFrequency(term, docID)
	}
	return 0
}

func (si *SegmentedIndex) DocumentFrequency(term string) int {
	v := si.view.Load()
	df := v.buffer.DocumentFrequency(term)
	for _, s := range v.segs {
		df += s.documentFrequency(term)
	}
	return df
}

func (si *SegmentedIndex) AverageDocumentLength() float64 {
	si.meta.RLock()
	defer si.meta.RUnlock()
	if len(si.idLoc) == 0 {
		return 0
	}
	return float64(si.liveToken) / float64(len(si.idLoc))
}

func (si *SegmentedIndex) DocumentLength(docID string) int {
	v := si.view.Load()
	si.meta.RLock()
	loc, ok := si.idLoc[docID]
	si.meta.RUnlock()
	if !ok {
		return 0
	}
	if loc == bufferLoc {
		return v.buffer.DocumentLength(docID)
	}
	if seg := findSeg(v.segs, loc); seg != nil && seg.live(docID) {
		return seg.idx.DocumentLength(docID)
	}
	return 0
}

func (si *SegmentedIndex) Positions(term, docID string) []int {
	v := si.view.Load()
	si.meta.RLock()
	loc, ok := si.idLoc[docID]
	si.meta.RUnlock()
	if !ok {
		return []int{}
	}
	if loc == bufferLoc {
		return v.buffer.Positions(term, docID)
	}
	if seg := findSeg(v.segs, loc); seg != nil && seg.live(docID) {
		return seg.idx.Positions(term, docID)
	}
	return []int{}
}

func (si *SegmentedIndex) Terms() []string {
	v := si.view.Load()
	set := make(map[string]struct{})
	for _, t := range v.buffer.Terms() {
		set[t] = struct{}{}
	}
	for _, s := range v.segs {
		for _, t := range s.idx.Terms() {
			if _, seen := set[t]; seen {
				continue
			}
			if s.documentFrequency(t) > 0 {
				set[t] = struct{}{}
			}
		}
	}
	terms := make([]string, 0, len(set))
	for t := range set {
		terms = append(terms, t)
	}
	return terms
}

func (si *SegmentedIndex) DocumentIDs() []string {
	si.meta.RLock()
	defer si.meta.RUnlock()
	ids := make([]string, 0, len(si.idLoc))
	for id := range si.idLoc {
		ids = append(ids, id)
	}
	return ids
}

func (si *SegmentedIndex) SegmentCount() int {
	return len(si.view.Load().segs)
}

func findSeg(segs []*segment, seqID int) *segment {
	for _, s := range segs {
		if s.seqID == seqID {
			return s
		}
	}
	return nil
}

func replaceSeg(segs []*segment, seqID int, repl *segment) []*segment {
	out := make([]*segment, len(segs))
	copy(out, segs)
	for i, s := range out {
		if s.seqID == seqID {
			out[i] = repl
			break
		}
	}
	return out
}

type manifest struct {
	NextID   int
	Segments []segmentRef
}

type segmentRef struct {
	SeqID      int
	File       string
	Tombstones []string
}

func readManifest(dir string) (*manifest, error) {
	p := filepath.Join(dir, manifestName)
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("index: read manifest: %w", err)
	}
	defer f.Close()
	var m manifest
	if err := gob.NewDecoder(f).Decode(&m); err != nil {
		return nil, fmt.Errorf("index: decode manifest: %w", err)
	}
	return &m, nil
}

func cleanupOrphans(dir string, keep map[string]struct{}) {
	tmps, _ := filepath.Glob(filepath.Join(dir, "*.tmp"))
	for _, p := range tmps {
		os.Remove(p)
	}
	segs, _ := filepath.Glob(filepath.Join(dir, "seg-*.idx"))
	for _, p := range segs {
		if _, ok := keep[filepath.Base(p)]; !ok {
			os.Remove(p)
		}
	}
}

var _ Index = (*SegmentedIndex)(nil)
