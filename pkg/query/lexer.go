package query

import (
	"strings"
	"unicode"
)

type tokenKind int

const (
	tEOF tokenKind = iota
	tWord
	tPhrase
	tAnd
	tOr
	tNot
	tLParen
	tRParen
	tOp
)

type token struct {
	kind tokenKind
	val  string
}

func lex(input string) []token {
	var toks []token
	runes := []rune(input)
	i := 0
	n := len(runes)

	emitWord := func(s string) {
		switch s {
		case "AND":
			toks = append(toks, token{tAnd, s})
		case "OR":
			toks = append(toks, token{tOr, s})
		case "NOT":
			toks = append(toks, token{tNot, s})
		default:
			toks = append(toks, token{tWord, s})
		}
	}

	for i < n {
		c := runes[i]

		switch {
		case unicode.IsSpace(c):
			i++

		case c == '(':
			toks = append(toks, token{tLParen, "("})
			i++

		case c == ')':
			toks = append(toks, token{tRParen, ")"})
			i++

		case c == '"':
			i++
			start := i
			for i < n && runes[i] != '"' {
				i++
			}
			toks = append(toks, token{tPhrase, string(runes[start:i])})
			if i < n {
				i++
			}

		case c == '>' || c == '<' || c == '=' || c == '!':
			start := i
			i++
			if i < n && runes[i] == '=' {
				i++
			}
			op := string(runes[start:i])
			if op == "!" {
				toks = append(toks, token{tWord, "!"})
			} else {
				toks = append(toks, token{tOp, op})
			}

		default:
			start := i
			for i < n && !isWordBreak(runes[i]) {
				i++
			}
			emitWord(string(runes[start:i]))
		}
	}

	toks = append(toks, token{tEOF, ""})
	return toks
}

func isWordBreak(c rune) bool {
	if unicode.IsSpace(c) {
		return true
	}
	return strings.ContainsRune("()\"><=!", c)
}
