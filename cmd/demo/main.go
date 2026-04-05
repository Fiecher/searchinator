package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/engine"
)

var corpus = []searchinator.Document{
	{
		ID:   "go",
		Text: "Go is an open source programming language that makes it easy to build simple reliable and efficient software. Go is statically typed compiled and designed for concurrency.",
		Meta: map[string]any{"url": "https://go.dev", "year": 2009},
	},
	{
		ID:   "python",
		Text: "Python is a high level general purpose programming language. Its design philosophy emphasises code readability. Python supports multiple programming paradigms including procedural object oriented and functional programming.",
		Meta: map[string]any{"url": "https://python.org", "year": 1991},
	},
	{
		ID:   "rust",
		Text: "Rust is a systems programming language focused on safety speed and concurrency. Rust achieves memory safety without a garbage collector using a system of ownership with rules the compiler checks.",
		Meta: map[string]any{"url": "https://rust-lang.org", "year": 2015},
	},
	{
		ID:   "typescript",
		Text: "TypeScript is a strongly typed programming language that builds on JavaScript giving you better tooling at any scale. TypeScript adds optional static typing and class based object oriented programming to JavaScript.",
		Meta: map[string]any{"url": "https://typescriptlang.org", "year": 2012},
	},
	{
		ID:   "haskell",
		Text: "Haskell is a purely functional programming language with strong static typing and lazy evaluation. Haskell features type inference and supports higher order functions and monads.",
		Meta: map[string]any{"url": "https://haskell.org", "year": 1990},
	},
	{
		ID:   "java",
		Text: "Java is a high level object oriented programming language designed to have as few implementation dependencies as possible. Java applications are compiled to bytecode that runs on the Java Virtual Machine.",
		Meta: map[string]any{"url": "https://java.com", "year": 1995},
	},
	{
		ID:   "c",
		Text: "C is a general purpose systems programming language that provides low level memory access. C has been widely used to implement operating systems kernels and embedded systems due to its efficiency.",
		Meta: map[string]any{"url": "https://en.wikipedia.org/wiki/C_(programming_language)", "year": 1972},
	},
	{
		ID:   "swift",
		Text: "Swift is a general purpose compiled programming language developed by Apple. Swift is designed to work alongside Objective C and is used to develop software for Apple platforms including iOS and macOS.",
		Meta: map[string]any{"url": "https://swift.org", "year": 2014},
	},
}

func main() {
	fmt.Println(banner())

	e, err := engine.NewEngine(engine.DefaultConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Indexing %d documents... ", len(corpus))
	if err := e.Index(corpus); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("done.")
	printCorpusSummary()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("\n%s", prompt())
		if !scanner.Scan() {
			break
		}
		query := strings.TrimSpace(scanner.Text())
		if query == "" {
			continue
		}
		if query == "quit" || query == "exit" || query == "q" {
			fmt.Println("\n  Goodbye!")
			break
		}
		if query == "help" {
			printHelp()
			continue
		}
		if query == "corpus" {
			printCorpus()
			continue
		}

		results, err := e.Search(query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  search error: %v\n", err)
			continue
		}
		printResults(query, results)
	}
}

func printResults(query string, results []searchinator.Result) {
	divider := strings.Repeat("─", 60)

	fmt.Println()
	fmt.Printf("  Query: %q\n", query)
	fmt.Println(" ", divider)

	if len(results) == 0 {
		fmt.Println("  No matching documents found.")
		return
	}

	maxScore := results[0].Score

	for rank, r := range results {
		bar := scoreBar(r.Score, maxScore, 20)
		url, _ := r.Document.Meta["url"].(string)
		year, _ := r.Document.Meta["year"].(int)

		fmt.Printf("  #%-2d  %-12s  %s  %.4f\n",
			rank+1, r.Document.ID, bar, r.Score)
		fmt.Printf("       %s (%d)\n", url, year)
	}

	fmt.Println(" ", divider)
	fmt.Printf("  %d result(s)\n", len(results))
}

func scoreBar(score, maxScore float64, width int) string {
	if maxScore == 0 {
		return strings.Repeat("░", width)
	}
	filled := int(score / maxScore * float64(width))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

func printCorpusSummary() {
	fmt.Printf("  %d documents indexed:\n", len(corpus))
	for _, doc := range corpus {
		fmt.Printf("    • %-12s — %s\n", doc.ID, truncate(doc.Text, 55))
	}
}

func printCorpus() {
	fmt.Println()
	fmt.Println("  Indexed documents:")
	fmt.Println(" ", strings.Repeat("─", 60))
	for _, doc := range corpus {
		fmt.Printf("  %-12s  %s\n", doc.ID, doc.Text)
	}
}

func printHelp() {
	fmt.Println()
	fmt.Println("  Commands:")
	fmt.Println("    <query>   — search the corpus")
	fmt.Println("    corpus    — show all indexed documents")
	fmt.Println("    help      — show this message")
	fmt.Println("    quit      — exit")
	fmt.Println()
	fmt.Println("  Example queries:")
	fmt.Println("    concurrency")
	fmt.Println("    functional programming")
	fmt.Println("    memory safety systems")
	fmt.Println("    compiled language")
	fmt.Println("    object oriented")
}

func banner() string {
	return `
 ┌─────────────────────────────────────────────────────────┐
 │           searchinator — full-text search demo          │
 │                  Phase 1 MVP · BM25 ranking             │
 └─────────────────────────────────────────────────────────┘
`
}

func prompt() string {
	return "  search> "
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
