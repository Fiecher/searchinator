package query

import (
	"github.com/Fiecher/searchinator/pkg/index"
)

type node interface {
	eval(ctx *evalCtx) map[string]struct{}
	positiveTerms() []string
}

type evalCtx struct {
	idx      index.Index
	universe map[string]struct{}
}

func newEvalCtx(idx index.Index) *evalCtx {
	u := make(map[string]struct{})
	for _, id := range idx.DocumentIDs() {
		u[id] = struct{}{}
	}
	return &evalCtx{idx: idx, universe: u}
}

type emptyNode struct{}

func (emptyNode) eval(*evalCtx) map[string]struct{} { return map[string]struct{}{} }
func (emptyNode) positiveTerms() []string           { return nil }

type termNode struct{ term string }

func (t termNode) eval(ctx *evalCtx) map[string]struct{} {
	out := make(map[string]struct{})
	for _, id := range ctx.idx.Get(t.term) {
		out[id] = struct{}{}
	}
	return out
}
func (t termNode) positiveTerms() []string { return []string{t.term} }

type phraseNode struct{ terms []string }

func (p phraseNode) eval(ctx *evalCtx) map[string]struct{} {
	out := make(map[string]struct{})
	if len(p.terms) == 0 {
		return out
	}
	for _, id := range ctx.idx.Get(p.terms[0]) {
		if PhraseMatch(ctx.idx, id, p.terms) {
			out[id] = struct{}{}
		}
	}
	return out
}
func (p phraseNode) positiveTerms() []string { return p.terms }

type andNode struct{ left, right node }

func (a andNode) eval(ctx *evalCtx) map[string]struct{} {
	l := a.left.eval(ctx)
	r := a.right.eval(ctx)
	out := make(map[string]struct{})
	for id := range l {
		if _, ok := r[id]; ok {
			out[id] = struct{}{}
		}
	}
	return out
}
func (a andNode) positiveTerms() []string {
	return append(a.left.positiveTerms(), a.right.positiveTerms()...)
}

type orNode struct{ left, right node }

func (o orNode) eval(ctx *evalCtx) map[string]struct{} {
	out := o.left.eval(ctx)
	for id := range o.right.eval(ctx) {
		out[id] = struct{}{}
	}
	return out
}
func (o orNode) positiveTerms() []string {
	return append(o.left.positiveTerms(), o.right.positiveTerms()...)
}

type notNode struct{ child node }

func (nn notNode) eval(ctx *evalCtx) map[string]struct{} {
	excluded := nn.child.eval(ctx)
	out := make(map[string]struct{})
	for id := range ctx.universe {
		if _, ok := excluded[id]; !ok {
			out[id] = struct{}{}
		}
	}
	return out
}

func (nn notNode) positiveTerms() []string { return nil }

type filterNode struct {
	field string
	op    string
	value string
}

func (f filterNode) eval(ctx *evalCtx) map[string]struct{} {
	out := make(map[string]struct{})
	for id := range ctx.universe {
		doc, ok := ctx.idx.GetDocument(id)
		if !ok {
			continue
		}
		if matchMeta(doc.Meta, f.field, f.op, f.value) {
			out[id] = struct{}{}
		}
	}
	return out
}

func (f filterNode) positiveTerms() []string { return nil }
