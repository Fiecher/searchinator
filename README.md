# searchinator

A reusable, embeddable full-text search library written in Go.

## Installation

```bash  
go get github.com/Fiecher/searchinator@latest```  
  
Requires Go 1.22 or later.  
  
## Quick Start  
  
```go  
package main  
  
import (  
    "fmt"  
    "github.com/Fiecher/searchinator"    "github.com/Fiecher/searchinator/engine")  
  
func main() {  
    // Create engine with default config (BM25 + standard text pipeline)    e, err := engine.NewEngine(engine.DefaultConfig())    if err != nil {        panic(err)    }  
    // Index your documents    docs := []searchinator.Document{        {ID: "go",     Text: "Go is a compiled statically typed language designed for concurrency"},        {ID: "python", Text: "Python is a high level interpreted language with dynamic typing"},        {ID: "rust",   Text: "Rust is a systems language focused on memory safety and performance"},    }    if err := e.Index(docs); err != nil {        panic(err)    }  
    // Search    results, err := e.Search("compiled language")    if err != nil {        panic(err)    }  
    for _, r := range results {        fmt.Printf("%-8s  score=%.4f\n", r.Document.ID, r.Score)    }    // Output:    // go        score=1.0821    // python    score=0.4063    // rust      score=0.4063}  
```  

## Running the Demo

The interactive demo lets you search a built-in corpus of programming language

documents and see BM25 scores and rankings in real time.

```bash  
# Clone the repo  
git clone https://github.com/Fiecher/searchinator.git  
cd searchinator  
  
# Normal mode — exact matching  
go run ./cmd/demo  
  
# Fuzzy mode — typo tolerant  
go run ./cmd/demo -fuzzy  
  
# Fuzzy mode with distance 2  
go run ./cmd/demo -fuzzy -distance 2  
```  

## Running Tests

```bash  
# All tests  
go test ./...  
  
# Verbose output  
go test ./... -v  
  
# Specific package  
go test ./analysis/... -v  
go test ./engine/... -v  
  
# Benchmarks  
go test ./engine/... -bench=. -benchmem  
```  

## Configuration

### Default config

Standard pipeline: whitespace tokenization -> lowercase -> strip punctuation -> BM25.

```go  
e, _ := engine.NewEngine(engine.DefaultConfig())  
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
    Analyzer: analysis.NewPipelineAnalyzer(        analysis.NewWhitespaceTokenizer(),        analysis.NewLowercaseFilter(),        analysis.NewPunctuationFilter(),        myCustomStemmer,   // implement analysis.TokenFilter    ),    Ranker: ranking.NewBM25(ranking.BM25Params{        K1: 1.5,  // term frequency saturation (default 1.2)        B:  0.6,  // length normalization      (default 0.75)    }),}  
e, _ := engine.NewEngine(cfg)  
```  

## Info
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
results, _ := fe.Search("concurency")  // finds "concurrency"  
results, _ = fe.Search("compied")      // finds "compiled"  
results, _ = fe.Search("saftey")       // finds "safety"  
```  

**Recommended `maxDistance` values:**

| Value | Use case |
|---|---|
| `1` | Catches most single-character typos — transpositions, deletions, insertions, substitutions |
| `2` | Catches harder typos; may over-expand on short terms (2–3 characters) |

### Document Model

```go  
type Document struct {  
    ID   string         // unique identifier -- required    Text string         // full text to be indexed and searched    Meta map[string]any // arbitrary metadata -- stored and returned, never indexed}  
```  

`Meta` can hold anything: URLs, timestamps, author names, tags. It is returned

in search results exactly as provided.

### Search Results

```go  
type Result struct {  
    Document Document // the full document including Meta    Score    float64  // BM25 relevance score}  
```  

Results are always sorted by `Score` descending. Ties are broken

alphabetically by `Document.ID` for deterministic output.

### BM25 Parameters

BM25 is controlled by two parameters:

| Parameter | Default | Effect |
|---|---|---|
| `K1` | `1.2` | Term frequency saturation. Higher values give more weight to repeated terms. Typical range: 1.2–2.0 |
| `B` | `0.75` | Length normalization. `0` = no normalization, `1` = full normalization |

The defaults follow the original Okapi BM25 paper and work well for most corpora.