package analysis

type FuzzyFilter struct {
	vocabulary  []string
	maxDistance int
}

func NewFuzzyFilter(vocabulary []string, maxDistance int) *FuzzyFilter {
	return &FuzzyFilter{
		vocabulary:  vocabulary,
		maxDistance: maxDistance,
	}
}

func (f *FuzzyFilter) Filter(tokens []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(tokens))

	for _, token := range tokens {
		matched := false

		for _, term := range f.vocabulary {
			d := levenshtein(token, term)
			if d <= f.maxDistance {
				if _, exists := seen[term]; !exists {
					seen[term] = struct{}{}
					out = append(out, term)
				}
				matched = true
			}
		}

		if !matched {
			if _, exists := seen[token]; !exists {
				seen[token] = struct{}{}
				out = append(out, token)
			}
		}
	}

	return out
}

func levenshtein(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)

	na := len(ra)
	nb := len(rb)

	if na == 0 {
		return nb
	}
	if nb == 0 {
		return na
	}

	d := make([][]int, na+1)
	for i := range d {
		d[i] = make([]int, nb+1)
	}

	for i := range na + 1 {
		d[i][0] = i
	}
	for j := range nb + 1 {
		d[0][j] = j
	}

	for i := 1; i <= na; i++ {
		for j := 1; j <= nb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}

			d[i][j] = min3(
				d[i-1][j]+1,
				d[i][j-1]+1,
				d[i-1][j-1]+cost,
			)

			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				d[i][j] = min3(d[i][j], d[i-2][j-2]+1, d[i][j])
			}
		}
	}

	return d[na][nb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
