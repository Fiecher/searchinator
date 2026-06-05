package highlight

import "strings"

type Options struct {
	Radius       int
	Open         string
	Close        string
	Ellipsis     string
	MaxFragments int
}

func DefaultOptions() Options {
	return Options{Radius: 6, Open: "**", Close: "**", Ellipsis: "...", MaxFragments: 12}
}

type Span struct {
	Text  string
	Match bool
}

type Analyze func(string) []string

func Snippet(text string, queryTerms []string, analyze Analyze, opts Options) string {
	if opts.Open == "" && opts.Close == "" {
		opts = DefaultOptions()
	}
	if opts.Ellipsis == "" {
		opts.Ellipsis = "..."
	}
	if opts.Radius <= 0 {
		opts.Radius = 6
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	want := make(map[string]struct{}, len(queryTerms))
	for _, t := range queryTerms {
		want[t] = struct{}{}
	}

	matched := make([]bool, len(words))
	first, last := -1, -1
	for i, w := range words {
		for _, form := range analyze(w) {
			if _, ok := want[form]; ok {
				matched[i] = true
				if first < 0 {
					first = i
				}
				last = i
				break
			}
		}
	}

	if first < 0 {
		end := min(len(words), opts.Radius*2)
		preview := strings.Join(words[:end], " ")
		if end < len(words) {
			preview += " " + opts.Ellipsis
		}
		return preview
	}

	start := max(0, first-opts.Radius)
	end := min(len(words), last+opts.Radius+1)

	var b strings.Builder
	if start > 0 {
		b.WriteString(opts.Ellipsis + " ")
	}
	for i := start; i < end; i++ {
		if i > start {
			b.WriteByte(' ')
		}
		if matched[i] {
			b.WriteString(opts.Open + words[i] + opts.Close)
		} else {
			b.WriteString(words[i])
		}
	}
	if end < len(words) {
		b.WriteString(" " + opts.Ellipsis)
	}
	return b.String()
}

func HighlightSpans(text string, queryTerms []string, analyze Analyze, opts Options) []Span {
	opts = withDefaults(opts)

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	want := make(map[string]struct{}, len(queryTerms))
	for _, t := range queryTerms {
		want[t] = struct{}{}
	}

	matched := make([]bool, len(words))
	var hits []int
	for i, w := range words {
		for _, form := range analyze(w) {
			if _, ok := want[form]; ok {
				matched[i] = true
				hits = append(hits, i)
				break
			}
		}
	}

	if len(hits) == 0 {
		end := min(len(words), opts.Radius*2)
		preview := strings.Join(words[:end], " ")
		if end < len(words) {
			preview += " " + opts.Ellipsis
		}
		return []Span{{Text: preview}}
	}

	type window struct{ start, end int }
	var wins []window
	for _, h := range hits {
		s := max(0, h-opts.Radius)
		e := min(len(words), h+opts.Radius+1)
		if n := len(wins); n > 0 && s <= wins[n-1].end {
			if e > wins[n-1].end {
				wins[n-1].end = e
			}
			continue
		}
		wins = append(wins, window{s, e})
	}
	trimmedTail := false
	if opts.MaxFragments > 0 && len(wins) > opts.MaxFragments {
		wins = wins[:opts.MaxFragments]
		trimmedTail = true
	}

	var spans []Span
	var plain strings.Builder
	space := false
	flush := func() {
		if plain.Len() > 0 {
			spans = append(spans, Span{Text: plain.String()})
			plain.Reset()
		}
	}
	addPlain := func(s string) {
		if space {
			plain.WriteByte(' ')
		}
		plain.WriteString(s)
		space = true
	}
	addMatch := func(s string) {
		if space {
			plain.WriteByte(' ')
		}
		flush()
		spans = append(spans, Span{Text: s, Match: true})
		space = true
	}

	for wi, wnd := range wins {
		if wi == 0 {
			if wnd.start > 0 {
				space = false
				addPlain(opts.Ellipsis)
			}
		} else {
			addPlain(opts.Ellipsis)
		}
		for i := wnd.start; i < wnd.end; i++ {
			if matched[i] {
				addMatch(words[i])
			} else {
				addPlain(words[i])
			}
		}
	}
	if last := wins[len(wins)-1]; trimmedTail || last.end < len(words) {
		addPlain(opts.Ellipsis)
	}
	flush()
	return spans
}

func Highlights(text string, queryTerms []string, analyze Analyze, opts Options) string {
	opts = withDefaults(opts)
	var b strings.Builder
	for _, sp := range HighlightSpans(text, queryTerms, analyze, opts) {
		if sp.Match {
			b.WriteString(opts.Open + sp.Text + opts.Close)
		} else {
			b.WriteString(sp.Text)
		}
	}
	return b.String()
}

func withDefaults(opts Options) Options {
	if opts.Open == "" && opts.Close == "" {
		opts.Open, opts.Close = "**", "**"
	}
	if opts.Ellipsis == "" {
		opts.Ellipsis = "..."
	}
	if opts.Radius <= 0 {
		opts.Radius = 6
	}
	if opts.MaxFragments <= 0 {
		opts.MaxFragments = 12
	}
	return opts
}
