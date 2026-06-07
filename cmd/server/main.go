package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/internal/sampledata"
	"github.com/Fiecher/searchinator/pkg/engine"
	"github.com/Fiecher/searchinator/pkg/index"
	"github.com/Fiecher/searchinator/pkg/semantic"
)

type server struct {
	engine   *engine.Engine
	semantic *semantic.Engine
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	data := flag.String("data", "", "directory for a durable segmented index; empty = in-memory")
	flag.Parse()

	cfg := engine.BilingualConfig()

	var e *engine.Engine
	var err error
	if *data != "" {
		idx, oerr := index.OpenSegmented(*data)
		if oerr != nil {
			log.Fatal(oerr)
		}
		if e, err = engine.NewEngineWithIndex(cfg, idx); err != nil {
			log.Fatal(err)
		}
		log.Printf("durable index at %s", *data)
	} else {
		if e, err = engine.NewEngine(cfg); err != nil {
			log.Fatal(err)
		}

		if err = e.Index(sampledata.Corpus()); err != nil {
			log.Fatal(err)
		}
	}

	sem, err := semantic.NewEngine(semantic.NewHashEmbedder(256), semantic.NewBruteForceIndex())
	if err != nil {
		log.Fatal(err)
	}
	if err = sem.Index(sampledata.Corpus()); err != nil {
		log.Fatal(err)
	}

	s := &server{engine: e, semantic: sem}
	mux := http.NewServeMux()
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/semantic", s.handleSemantic)
	mux.HandleFunc("/searchbool", s.handleSearchBool)
	mux.HandleFunc("/wildcard", s.handleWildcard)
	mux.HandleFunc("/suggest", s.handleSuggest)
	mux.HandleFunc("/facet", s.handleFacet)
	mux.HandleFunc("/snippet", s.handleSnippet)
	mux.HandleFunc("/stats", s.handleStats)

	log.Printf("searchinator API listening on %s (%d docs indexed)", *addr, e.Stats().DocumentCount)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

type hit struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
	Text  string  `json:"text"`
}

func toHits(results []searchinator.Result) []hit {
	hits := make([]hit, len(results))
	for i, r := range results {
		hits[i] = hit{ID: r.Document.ID, Score: r.Score, Text: r.Document.Text}
	}
	return hits
}

func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	results, err := s.engine.SearchN(q, limit)
	writeResult(w, map[string]any{"query": q, "hits": toHits(results)}, err)
}

func (s *server) handleSemantic(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	results, err := s.semantic.Search(q, limit)
	writeResult(w, map[string]any{"query": q, "hits": toHits(results)}, err)
}

func (s *server) handleSearchBool(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := s.engine.SearchBool(q)
	writeResult(w, map[string]any{"query": q, "hits": toHits(results)}, err)
}

func (s *server) handleWildcard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := s.engine.SearchWildcard(q)
	writeResult(w, map[string]any{
		"pattern": q,
		"terms":   s.engine.ExpandWildcard(q),
		"hits":    toHits(results),
	}, err)
}

func (s *server) handleSuggest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	corrected, changed := s.engine.DidYouMean(q)
	writeResult(w, map[string]any{"query": q, "suggestion": corrected, "changed": changed}, nil)
}

func (s *server) handleFacet(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	field := r.URL.Query().Get("field")
	counts, err := s.engine.Facet(q, field)
	writeResult(w, map[string]any{"query": q, "field": field, "counts": counts}, err)
}

func (s *server) handleSnippet(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	q := r.URL.Query().Get("q")
	snippet, ok := s.engine.Snippet(id, q)
	if !ok {
		http.Error(w, `{"error":"document not found"}`, http.StatusNotFound)
		return
	}
	writeResult(w, map[string]any{"id": id, "query": q, "snippet": snippet}, nil)
}

func (s *server) handleStats(w http.ResponseWriter, r *http.Request) {
	st := s.engine.Stats()
	writeResult(w, map[string]any{
		"documents":  st.DocumentCount,
		"terms":      st.TermCount,
		"avg_length": st.AverageDocumentLength,
	}, nil)
}

func writeResult(w http.ResponseWriter, body any, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}
