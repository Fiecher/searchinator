package query

import (
	"strconv"
	"strings"

	"github.com/Fiecher/searchinator/pkg/analysis"
	"github.com/Fiecher/searchinator/pkg/index"
)

type Query struct {
	root node
}

func Parse(input string, a analysis.Analyzer) (*Query, error) {
	p := &parser{toks: lex(input), analyzer: a}
	root, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tEOF {
		return nil, &ParseError{msg: "unexpected token: " + p.peek().val}
	}
	if root == nil {
		root = emptyNode{}
	}
	return &Query{root: root}, nil
}

func (q *Query) Match(idx index.Index) map[string]struct{} {
	return q.root.eval(newEvalCtx(idx))
}

func (q *Query) Terms() []string {
	seen := make(map[string]struct{})
	var out []string
	for _, t := range q.root.positiveTerms() {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

type ParseError struct{ msg string }

func (e *ParseError) Error() string { return "query: " + e.msg }

type parser struct {
	toks     []token
	pos      int
	analyzer analysis.Analyzer
}

func (p *parser) peek() token { return p.toks[p.pos] }

func (p *parser) next() token {
	t := p.toks[p.pos]
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}

func (p *parser) parseOr() (node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tOr {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = orNode{left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		k := p.peek().kind
		if k == tAnd {
			p.next()
		} else if k == tWord || k == tPhrase || k == tNot || k == tLParen {

		} else {
			break
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = andNode{left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (node, error) {
	if p.peek().kind == tNot {
		p.next()
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return notNode{child: child}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (node, error) {
	t := p.peek()
	switch t.kind {
	case tLParen:
		p.next()
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tRParen {
			return nil, &ParseError{msg: "missing closing parenthesis"}
		}
		p.next()
		if inner == nil {
			return emptyNode{}, nil
		}
		return inner, nil

	case tPhrase:
		p.next()
		return p.nodeFromText(t.val), nil

	case tWord:

		if p.toks[p.pos+1].kind == tOp {
			field := t.val
			p.next()
			op := p.next().val
			val := p.peek()
			if val.kind != tWord && val.kind != tPhrase {
				return nil, &ParseError{msg: "filter " + field + op + " expects a value"}
			}
			p.next()
			return filterNode{field: field, op: op, value: val.val}, nil
		}
		p.next()
		return p.nodeFromText(t.val), nil

	case tEOF, tRParen:
		return nil, nil

	default:
		return nil, &ParseError{msg: "unexpected token: " + t.val}
	}
}

func (p *parser) nodeFromText(text string) node {
	terms := p.analyzer.Analyze(text)
	switch len(terms) {
	case 0:
		return emptyNode{}
	case 1:
		return termNode{term: terms[0]}
	default:
		return phraseNode{terms: terms}
	}
}

func PhraseMatch(idx index.Index, docID string, phrase []string) bool {
	if len(phrase) == 0 {
		return false
	}
	starts := idx.Positions(phrase[0], docID)
	if len(starts) == 0 {
		return false
	}
	for i := 1; i < len(phrase); i++ {
		posSet := make(map[int]struct{})
		for _, p := range idx.Positions(phrase[i], docID) {
			posSet[p] = struct{}{}
		}
		next := starts[:0:0]
		for _, s := range starts {
			if _, ok := posSet[s+i]; ok {
				next = append(next, s)
			}
		}
		if len(next) == 0 {
			return false
		}
		starts = next
	}
	return true
}

func matchMeta(meta map[string]any, field, op, value string) bool {
	raw, ok := meta[field]
	if !ok {
		return false
	}

	switch op {
	case "=", "!=":
		want := op == "="
		if lf, lok := toFloat(raw); lok {
			if rf, rok := parseFloat(value); rok {
				return (lf == rf) == want
			}
		}
		return (toString(raw) == value) == want

	case ">", "<", ">=", "<=":
		lf, lok := toFloat(raw)
		rf, rok := parseFloat(value)
		if !lok || !rok {
			return false
		}
		switch op {
		case ">":
			return lf > rf
		case "<":
			return lf < rf
		case ">=":
			return lf >= rf
		case "<=":
			return lf <= rf
		}
	}
	return false
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case string:
		return parseFloat(n)
	}
	return 0, false
}

func parseFloat(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func toString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case int:
		return strconv.Itoa(s)
	case int64:
		return strconv.FormatInt(s, 10)
	case float64:
		return strconv.FormatFloat(s, 'g', -1, 64)
	default:
		return ""
	}
}
