package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/internal/sampledata"
	"github.com/Fiecher/searchinator/pkg/engine"
	"github.com/Fiecher/searchinator/pkg/eval"
	"github.com/Fiecher/searchinator/pkg/semantic"
)

type judged struct {
	query    string
	relevant []string
}

var judgments = []judged{
	{"systems programming low level memory", []string{"c", "rust"}},
	{"purely functional programming", []string{"haskell", "python"}},
	{"concurrency and memory safety", []string{"go", "rust"}},
	{"object oriented language", []string{"java", "python", "typescript"}},
	{"statically typed compiled language", []string{"go", "rust", "java", "swift", "typescript"}},
}

func main() {
	k := flag.Int("k", 3, "cutoff K for Precision@K / Recall@K / nDCG@K")
	flag.Parse()

	docs := sampledata.Corpus()

	lex, err := buildLexical(docs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lexical:", err)
		os.Exit(1)
	}
	sem, err := buildSemantic(docs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "semantic:", err)
		os.Exit(1)
	}

	fmt.Printf("Corpus: %d docs   Engines: BM25 (lexical) vs %s (semantic)   K=%d\n\n",
		len(docs), semanticName(), *k)

	lexRuns := map[string][]string{}
	semRuns := map[string][]string{}
	judg := map[string]eval.Relevant{}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Query\tEngine\tP@K\tR@K\tnDCG@K\tAP")
	fmt.Fprintln(w, "-----\t------\t---\t---\t------\t--")

	for _, j := range judgments {
		rel := eval.Relevant{}
		for _, id := range j.relevant {
			rel[id] = true
		}
		judg[j.query] = rel

		lexIDs := lexRanked(lex, j.query)
		semIDs, err := sem.RankedIDs(j.query, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, "semantic search:", err)
			os.Exit(1)
		}
		lexRuns[j.query] = lexIDs
		semRuns[j.query] = semIDs

		printRow(w, j.query, "BM25", lexIDs, rel, *k)
		printRow(w, "", "semantic", semIDs, rel, *k)
	}
	w.Flush()

	fmt.Printf("\nMAP  BM25=%.4f  semantic=%.4f\n",
		eval.MeanAveragePrecision(lexRuns, judg),
		eval.MeanAveragePrecision(semRuns, judg))
}

func printRow(w *tabwriter.Writer, query, name string, ranked []string, rel eval.Relevant, k int) {
	fmt.Fprintf(w, "%s\t%s\t%.3f\t%.3f\t%.3f\t%.3f\n",
		query, name,
		eval.PrecisionAtK(ranked, rel, k),
		eval.RecallAtK(ranked, rel, k),
		eval.NDCGAtK(ranked, rel, k),
		eval.AveragePrecision(ranked, rel))
}

func buildLexical(docs []searchinator.Document) (*engine.Engine, error) {
	e, err := engine.NewEngine(engine.EnglishConfig())
	if err != nil {
		return nil, err
	}
	if err := e.Index(docs); err != nil {
		return nil, err
	}
	return e, nil
}

func buildSemantic(docs []searchinator.Document) (*semantic.Engine, error) {
	emb, err := newEmbedder()
	if err != nil {
		return nil, err
	}
	e, err := semantic.NewEngine(emb, semantic.NewBruteForceIndex())
	if err != nil {
		return nil, err
	}
	if err := e.Index(docs); err != nil {
		return nil, err
	}
	return e, nil
}

func lexRanked(e *engine.Engine, query string) []string {
	results, err := e.SearchN(query, 0)
	if err != nil {
		return nil
	}
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.Document.ID
	}
	return ids
}
