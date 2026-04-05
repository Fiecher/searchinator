package analysis

import "strings"

type WhitespaceTokenizer struct{}

func NewWhitespaceTokenizer() *WhitespaceTokenizer {
	return &WhitespaceTokenizer{}
}

func (t *WhitespaceTokenizer) Tokenize(text string) []string {
	raw := strings.Fields(text)
	return raw
}
