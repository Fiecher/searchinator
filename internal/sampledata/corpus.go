package sampledata

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/internal/docload"
)

//go:embed corpus/*.md
var corpusFS embed.FS

func Corpus() []searchinator.Document {
	entries, err := corpusFS.ReadDir("corpus")
	if err != nil {
		panic("sampledata: " + err.Error())
	}
	docs := make([]searchinator.Document, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		data, err := corpusFS.ReadFile("corpus/" + name)
		if err != nil {
			panic("sampledata: " + err.Error())
		}
		body, meta := splitFrontMatter(data)

		text, err := docload.Extract(filepath.Ext(name), body)
		if err != nil {
			panic("sampledata: " + name + ": " + err.Error())
		}
		docs = append(docs, searchinator.Document{
			ID:   strings.TrimSuffix(name, filepath.Ext(name)),
			Text: strings.TrimSpace(text),
			Meta: meta,
		})
	}
	return docs
}

func splitFrontMatter(data []byte) (body []byte, meta map[string]any) {
	meta = map[string]any{}
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return []byte(s), meta
	}
	rest := s[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return []byte(s), meta
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		if n, err := strconv.Atoi(val); err == nil {
			meta[key] = n
		} else {
			meta[key] = val
		}
	}
	return []byte(strings.TrimLeft(rest[end+len("\n---"):], "\n")), meta
}

var exampleStop = map[string]bool{

	"that": true, "with": true, "this": true, "from": true, "have": true,
	"used": true, "which": true, "their": true, "there": true, "into": true,

	"который": true, "которая": true, "которые": true, "благодаря": true,
	"включая": true, "поддерживает": true, "поддержку": true, "обеспечивающий": true,
	"общего": true, "назначения": true, "своей": true, "вместе": true,
}

func ExampleQuery() string { return ExampleQueryFrom(Corpus()) }

func ExampleQueryFrom(docs []searchinator.Document) string {
	if len(docs) == 0 {
		return "type a query..."
	}

	freq := map[string]int{}
	for _, d := range docs {
		for _, w := range words(d.Text) {
			if utf8.RuneCountInString(w) >= 5 && !exampleStop[w] {
				freq[w]++
			}
		}
	}

	top := rankByFreq(freq)
	phrase := firstPhrase(docs[0].Text, 2)

	switch {
	case len(top) >= 4 && phrase != "":
		return fmt.Sprintf("%s %s   ·   %q   ·   %s OR %s", top[0], top[1], phrase, top[2], top[3])
	case len(top) >= 2 && phrase != "":
		return fmt.Sprintf("%s %s   ·   %q", top[0], top[1], phrase)
	case len(top) >= 2:
		return fmt.Sprintf("%s %s", top[0], top[1])
	case phrase != "":
		return phrase
	default:
		return "type a query..."
	}
}

func words(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r)
	})
}

func rankByFreq(freq map[string]int) []string {
	out := make([]string, 0, len(freq))
	for w := range freq {
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool {
		if freq[out[i]] != freq[out[j]] {
			return freq[out[i]] > freq[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}

func firstPhrase(text string, n int) string {
	var picked []string
	for _, w := range words(text) {
		if utf8.RuneCountInString(w) >= 4 {
			picked = append(picked, w)
			if len(picked) == n {
				break
			}
		}
	}
	return strings.Join(picked, " ")
}
