# searchinator

A reusable, embeddable full-text search library written in Go.

## Installation

```bash
go get github.com/Fiecher/searchinator@latest
```

Requires Go 1.22 or later.

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/pkg/engine"
)

func main() {
	e, err := engine.NewEngine(engine.DefaultConfig())
	if err != nil {
		panic(err)
	}

	// Index docs
	docs := []searchinator.Document{
		{ID: "go", Text: "Go is a compiled statically typed language designed for concurrency"},
		{ID: "python", Text: "Python is a high level interpreted language with dynamic typing"},
		{ID: "rust", Text: "Rust is a systems language focused on memory safety and performance"},
	}
	if err := e.Index(docs); err != nil {
		panic(err)
	}

	// Search
	results, err := e.Search("compiled language")
	if err != nil {
		panic(err)
	}
	for _, r := range results {
		fmt.Printf("%-8s  score=%.4f\n", r.Document.ID, r.Score)
	}
}
```

## Running the Demo

The windowed demo (built with [Fyne](https://fyne.io)) lets you search a
built-in corpus of Russian-language documents about programming languages and
see BM25/TF-IDF scores, rankings and highlighted snippets in real time. You can
switch the ranker, toggle typo-tolerant fuzzy matching, enable boolean query
mode (AND/OR/NOT and metadata filters), and run **semantic** search by
description (see below). Use **Settings…** to choose the index storage location
and tune the BM25/fuzzy parameters at runtime.

Supported formats: `.txt`, `.md`/`.markdown`,
`.docx` (Word), `.html`/`.htm`, `.rtf`, `.csv`/`.tsv`/`.log`; any other extension
is read as plain UTF-8 text.

The GUI uses cgo, so a C compiler must be on your PATH (`gcc`/MinGW on Windows,
build-essential on Linux, Xcode tools on macOS).

```bash
# Run the windowed demo
CGO_ENABLED=1 go run ./cmd/gui

# Or via the Makefile
make run

# Durable mode: back the index with an on-disk segmented store so imported
# documents survive a restart.
CGO_ENABLED=1 go run ./cmd/gui -data ./var/index
```

### Durable storage

Both demos can run over a **segmented, on-disk index** (`index.SegmentedIndex`,
an LSM-style store: a mutable in-memory buffer plus immutable on-disk segments,
committed atomically via a `MANIFEST`). Pass `-data <dir>` to either
command; omit it for an in-memory-only index.

```bash
# HTTP API over a durable index (starts empty; index your own documents)
go run ./cmd/server -addr :8080 -data ./var/index
```

Build an engine over a durable index programmatically:

```go
idx, err := index.OpenSegmented("./var/index") // recovers prior state
eng, err := engine.NewEngineWithIndex(cfg, idx)
// ... eng.Index(docs) ...
err = eng.Flush() // commit the in-memory buffer to a new on-disk segment
```

## Running Tests

```bash
# All tests
go test ./...

# Verbose output
go test ./... -v

# Specific package
go test ./pkg/analysis/... -v
go test ./pkg/engine/... -v

# Benchmarks
go test ./pkg/engine/... -bench=. -benchmem
```

## Configuration

### Default config

Standard pipeline: whitespace tokenization -> lowercase -> strip punctuation -> BM25.

```go
e, _ := engine.NewEngine(engine.DefaultConfig())
```

### Bilingual config (English + Russian)

The demo corpus is Russian, so the GUI and HTTP server use a bilingual pipeline:
English and Russian stop words are stripped, then the Porter (English) and
Snowball Russian stemmers run in turn. Each stemmer is a no-op on the other
script, so inflected Russian forms conflate (*владения*/*владение* match) while
English code terms like *javascript* are untouched.

```go
e, _ := engine.NewEngine(engine.BilingualConfig())
```

### Fuzzy config

Adds a `FuzzyFilter` at query time. Documents are still indexed with exact terms.

```go
e, _ := engine.NewEngine(engine.FuzzyConfig(vocab, 1))
```

### Custom config

Every component is an interface — replace any part of the pipeline:

```go
cfg := engine.Config{
Analyzer: analysis.NewPipelineAnalyzer(
analysis.NewWhitespaceTokenizer(),
analysis.NewLowercaseFilter(),
analysis.NewPunctuationFilter(),
myCustomStemmer, // implement analysis.TokenFilter
),
Ranker: ranking.NewBM25(ranking.BM25Params{
K1: 1.5, // term frequency saturation (default 1.2)
B:  0.6, // length normalization      (default 0.75)
}),
}
e, _ := engine.NewEngine(cfg)
```

## Info

### Semantic Search

Semantic mode searches by a *description* rather than by exact words. The
description is handed to a `semantic.WordPredictor`, which proposes the words
most likely to appear in matching documents; those predicted words are then
searched and highlighted by the lexical engine. The predicted words are shown in
the demo's status bar.

The default predictor is **offline and dependency-free**:
`semantic.NewCorpusPredictor(docs, analyze)` learns word associations from the
indexed corpus (term co-occurrence) and a small built-in thesaurus, so it works
without a network — ideal for demos and tests. The `analyze` argument is the
engine's own analyzer function (pass `nil` for a plain tokenizer): the predictor
aggregates statistics in the same **stem space** the engine searches in, so a
Russian description matches inflected forms in the corpus. It still emits real
**surface words** (not stems), because the engine re-analyzes whatever query it
is given.

```go
cfg := engine.BilingualConfig()
eng, _ := engine.NewEngine(cfg)
eng.Index(docs)

p := semantic.NewCorpusPredictor(docs, cfg.Analyzer.Analyze) // or nil
words, _ := p.Predict("язык со сборкой мусора", 6)
// e.g. ["владения" "заимствования" "сборщика" "мусора" ...]
results, _ := eng.Search(strings.Join(words, " "))
```

The predictor learns from text indexed after construction — call `Index` with
the new documents (the demo does this on every import) so later predictions take
them into account:

```go
p.Index(newDocs) // additive; no network, no rebuild
```

A real model plugs into the same interface — no other code changes:

```go
p, _ := semantic.NewLLMPredictor(semantic.LLMConfig{
    Endpoint: "http://localhost:11434/v1/chat/completions", // OpenAI-compatible
    Model:    "llama3.1",
    // APIKey: "…", // for hosted providers
})
words, _ := p.Predict("язык со сборкой мусора", 6)
```

Without a configured endpoint the demo uses the offline `CorpusPredictor`, so
semantic search always works.

A separate dense-vector engine (`semantic.Engine` with an `Embedder` and a
`VectorIndex`) is also available for cosine-similarity ranking and comparison.

### Fuzzy Search

Fuzzy search tolerates typos by expanding query tokens to the closest matching
terms in the index vocabulary using Damerau-Levenshtein edit distance.

```go
// Step 1: build a plain engine and index your documents
plain, _ := engine.NewEngine(engine.DefaultConfig())
plain.Index(docs)

// Step 2: extract vocabulary from the index
vocab := engine.VocabularyFromIndex(plain)

// Step 3: build a fuzzy engine with the same documents
fe, _ := engine.NewEngine(engine.FuzzyConfig(vocab, 1)) // max distance = 1
fe.Index(docs)

// Step 4: search with typos
results, _ := fe.Search("concurency") // finds "concurrency"
results, _ = fe.Search("compied") // finds "compiled"
results, _ = fe.Search("saftey") // finds "safety"
```

**Recommended `maxDistance` values:**

| Value | Use case                                                                                   |
|-------|--------------------------------------------------------------------------------------------|
| `1`   | Catches most single-character typos — transpositions, deletions, insertions, substitutions |
| `2`   | Catches harder typos; may over-expand on short terms (2–3 characters)                      |

### Document Model

```go
type Document struct {
ID   string // unique identifier -- required
Text string // full text to be indexed and searched
Meta map[string]any // arbitrary metadata -- stored and returned, never indexed
}
```

`Meta` can hold anything: URLs, timestamps, author names, tags. It is returned
in search results exactly as provided.

### Search Results

```go
type Result struct {
Document Document // the full document including Meta
Score    float64  // BM25 relevance score
}
```

Results are always sorted by `Score` descending. Ties are broken
alphabetically by `Document.ID` for deterministic output.

### BM25 Parameters

BM25 is controlled by two parameters:

| Parameter | Default | Effect                                                                                              |
|-----------|---------|-----------------------------------------------------------------------------------------------------|
| `K1`      | `1.2`   | Term frequency saturation. Higher values give more weight to repeated terms. Typical range: 1.2–2.0 |
| `B`       | `0.75`  | Length normalization. `0` = no normalization, `1` = full normalization                              |

The defaults follow the original Okapi BM25 paper and work well for most corpora.